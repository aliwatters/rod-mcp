package types

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// cloneProfile copies auth-relevant files from a Chrome user data directory
// into a temporary directory. If domains is non-empty and sqlite3 is available,
// cookies are filtered to only include matching domains.
//
// Returns the path to the cloned directory, which the caller must clean up.
func cloneProfile(srcDir string, domains []string) (string, error) {
	// Chrome profiles live in a "Default" subdirectory (or "Profile N").
	// The top-level dir also contains files Chrome needs to start.
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return "", fmt.Errorf("user-data-dir %q does not exist", srcDir)
	}

	profileDir := filepath.Join(srcDir, "Default")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no Default profile found in %q", srcDir)
	}

	tmpDir, err := os.MkdirTemp("", "rod-mcp-profile-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	tmpProfile := filepath.Join(tmpDir, "Default")
	if err := os.MkdirAll(tmpProfile, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("create profile dir: %w", err)
	}

	// Copy top-level files Chrome needs to start (Local State, etc.)
	for _, name := range []string{"Local State"} {
		src := filepath.Join(srcDir, name)
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, filepath.Join(tmpDir, name)); err != nil {
				log.Warnf("clone profile: skip %s: %s", name, err)
			}
		}
	}

	// Copy profile-level files needed for auth state.
	profileFiles := []string{
		"Preferences",
		"Secure Preferences",
	}
	for _, name := range profileFiles {
		src := filepath.Join(profileDir, name)
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, filepath.Join(tmpProfile, name)); err != nil {
				log.Warnf("clone profile: skip %s: %s", name, err)
			}
		}
	}

	// Copy cookies — filtered by domain if possible.
	cookieSrc := filepath.Join(profileDir, "Cookies")
	cookieDst := filepath.Join(tmpProfile, "Cookies")
	if _, err := os.Stat(cookieSrc); err == nil {
		if err := copyFile(cookieSrc, cookieDst); err != nil {
			log.Warnf("clone profile: skip Cookies: %s", err)
		} else if len(domains) > 0 {
			if err := filterCookiesByDomain(cookieDst, domains); err != nil {
				log.Warnf("clone profile: cookie filtering failed (keeping all): %s", err)
			}
		}
	}

	// Copy Local Storage directory (usually small, contains auth tokens).
	localStorageSrc := filepath.Join(profileDir, "Local Storage")
	if _, err := os.Stat(localStorageSrc); err == nil {
		if err := copyDir(localStorageSrc, filepath.Join(tmpProfile, "Local Storage")); err != nil {
			log.Warnf("clone profile: skip Local Storage: %s", err)
		}
	}

	// Copy Session Storage directory.
	sessionStorageSrc := filepath.Join(profileDir, "Session Storage")
	if _, err := os.Stat(sessionStorageSrc); err == nil {
		if err := copyDir(sessionStorageSrc, filepath.Join(tmpProfile, "Session Storage")); err != nil {
			log.Warnf("clone profile: skip Session Storage: %s", err)
		}
	}

	log.Infof("cloned Chrome profile to %s", tmpDir)
	return tmpDir, nil
}

// cloneProfileFull recursively copies the entire Chrome user data directory.
// This is slow for large profiles but preserves everything (extensions, history, etc.).
func cloneProfileFull(srcDir string) (string, error) {
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return "", fmt.Errorf("user-data-dir %q does not exist", srcDir)
	}

	tmpDir, err := os.MkdirTemp("", "rod-mcp-profile-full-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	// Remove the temp dir so copyDir creates it fresh from the source.
	if err := os.Remove(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("prepare temp dir: %w", err)
	}

	if err := copyDir(srcDir, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("full clone failed: %w", err)
	}

	log.Warnf("full profile clone to %s (this includes ALL browser data — passwords, history, extensions)", tmpDir)
	return tmpDir, nil
}

// isValidDomainPattern checks that a domain string contains only safe characters
// to prevent SQL injection when used in sqlite3 queries.
func isValidDomainPattern(d string) bool {
	for _, c := range d {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '*') {
			return false
		}
	}
	return len(d) > 0
}

// filterCookiesByDomain uses sqlite3 CLI to delete cookies not matching the given domains.
// Falls back silently if sqlite3 is not available.
func filterCookiesByDomain(cookiesDB string, domains []string) error {
	sqlite3Path, err := exec.LookPath("sqlite3")
	if err != nil {
		return fmt.Errorf("sqlite3 not found in PATH: cookie filtering unavailable")
	}

	// Build WHERE clause: keep cookies matching any domain pattern.
	// Domain patterns: "example.com" matches ".example.com" and "example.com"
	// Wildcard "*.example.com" matches ".example.com" subdomains.
	var conditions []string
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if !isValidDomainPattern(d) {
			return fmt.Errorf("invalid domain pattern %q: only alphanumeric, dots, hyphens, and wildcards allowed", d)
		}
		if strings.HasPrefix(d, "*.") {
			// *.example.com → match .example.com suffix
			suffix := d[1:] // ".example.com"
			conditions = append(conditions, fmt.Sprintf("host_key LIKE '%%%s'", suffix))
		} else {
			// exact domain: match both ".domain" and "domain"
			conditions = append(conditions, fmt.Sprintf("host_key LIKE '%%%s'", d))
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	keepClause := strings.Join(conditions, " OR ")
	query := fmt.Sprintf("DELETE FROM cookies WHERE NOT (%s);", keepClause)

	cmd := exec.Command(sqlite3Path, cookiesDB, query)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sqlite3 filter: %w: %s", err, string(output))
	}

	log.Infof("filtered cookies to domains: %s", strings.Join(domains, ", "))
	return nil
}

// copyFile copies a single file from src to dst, ensuring data is flushed to disk.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	// Always close; prefer the copy error if both fail.
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// copyDir recursively copies a directory tree, skipping symlinks.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Skip symlinks to avoid following links outside the profile directory.
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
