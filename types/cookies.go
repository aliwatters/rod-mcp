package types

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod/lib/proto"
)

// chromeEpochToUnix converts Chrome's expires_utc (microseconds since 1601-01-01)
// to Unix epoch seconds. Returns 0 for session cookies.
func chromeEpochToUnix(chromeTimestamp int64) float64 {
	if chromeTimestamp == 0 {
		return 0
	}
	// Chrome epoch starts at 1601-01-01. Offset to Unix epoch (1970-01-01) in microseconds.
	const chromeToUnixOffsetMicroseconds = 11644473600000000
	unixMicro := chromeTimestamp - chromeToUnixOffsetMicroseconds
	if unixMicro <= 0 {
		return 0
	}
	return float64(unixMicro) / 1000000.0
}

// chromeSameSiteToCDP converts Chrome's samesite integer to CDP enum.
func chromeSameSiteToCDP(ss int) proto.NetworkCookieSameSite {
	switch ss {
	case 0:
		return proto.NetworkCookieSameSiteNone
	case 1:
		return proto.NetworkCookieSameSiteLax
	case 2:
		return proto.NetworkCookieSameSiteStrict
	default:
		return ""
	}
}

// readChromeEncryptionKey reads the Chrome Safe Storage password from the macOS Keychain.
func readChromeEncryptionKey() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("cookie decryption is currently only supported on macOS")
	}

	cmd := exec.Command("security", "find-generic-password", "-w", "-s", "Chrome Safe Storage", "-a", "Chrome")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read Chrome Safe Storage from Keychain: %w (you may need to allow access)", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// deriveChromeKey derives the AES-128-CBC key from the Keychain password using PBKDF2.
func deriveChromeKey(password string) ([]byte, error) {
	salt := []byte("saltysalt")
	return pbkdf2.Key(sha1.New, password, salt, 1003, 16)
}

// decryptCookieValue decrypts a Chrome encrypted cookie value.
// Chrome prefixes encrypted values with "v10" (3 bytes), followed by AES-128-CBC ciphertext
// with a 16-byte zero IV.
func decryptCookieValue(encrypted []byte, key []byte) (string, error) {
	if len(encrypted) <= 3 {
		return "", fmt.Errorf("encrypted value too short (%d bytes)", len(encrypted))
	}

	// Strip the "v10" prefix (or "v11" on newer Chrome).
	prefix := string(encrypted[:3])
	if prefix != "v10" && prefix != "v11" {
		return "", fmt.Errorf("unexpected encryption prefix: %q", prefix)
	}
	ciphertext := encrypted[3:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext length %d is not a multiple of block size", len(ciphertext))
	}

	iv := make([]byte, aes.BlockSize) // 16 zero bytes
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS#7 padding.
	if len(plaintext) == 0 {
		return "", nil
	}
	padLen := int(plaintext[len(plaintext)-1])
	if padLen > aes.BlockSize || padLen > len(plaintext) || padLen == 0 {
		return "", fmt.Errorf("invalid PKCS7 padding: %d", padLen)
	}
	plaintext = plaintext[:len(plaintext)-padLen]

	return string(plaintext), nil
}

// ReadChromeCookies reads and decrypts cookies from a Chrome profile's Cookies SQLite DB.
// If domains is non-empty, only cookies matching those domains are returned.
// Returns CDP-compatible cookie params ready for injection.
func ReadChromeCookies(profileDir string, domains []string) ([]*proto.NetworkCookieParam, error) {
	cookiesDB := filepath.Join(profileDir, "Default", "Cookies")
	if _, err := os.Stat(cookiesDB); os.IsNotExist(err) {
		return nil, fmt.Errorf("cookies database not found at %s", cookiesDB)
	}

	// Read encryption key from Keychain.
	password, err := readChromeEncryptionKey()
	if err != nil {
		return nil, err
	}
	key, err := deriveChromeKey(password)
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	sqlite3Path, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil, fmt.Errorf("sqlite3 not found in PATH")
	}

	// Build domain filter.
	whereClause := ""
	if len(domains) > 0 {
		var conditions []string
		for _, d := range domains {
			d = strings.TrimSpace(d)
			if d == "" {
				continue
			}
			if !isValidDomainPattern(d) {
				return nil, fmt.Errorf("invalid domain pattern %q", d)
			}
			if strings.HasPrefix(d, "*.") {
				suffix := d[1:]
				conditions = append(conditions, fmt.Sprintf("host_key LIKE '%%%s'", suffix))
			} else {
				conditions = append(conditions, fmt.Sprintf("host_key LIKE '%%%s'", d))
			}
		}
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " OR ")
		}
	}

	// Query cookies with hex-encoded encrypted_value for safe transport.
	query := fmt.Sprintf(
		"SELECT host_key, name, hex(encrypted_value), value, path, is_secure, is_httponly, expires_utc, samesite FROM cookies%s;",
		whereClause,
	)

	cmd := exec.Command(sqlite3Path, "-separator", "\t", cookiesDB, query)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 query failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		log.Warnf("no cookies found matching domains: %v", domains)
		return nil, nil
	}

	cookies := make([]*proto.NetworkCookieParam, 0, len(lines))
	var decryptErrors int

	for _, line := range lines {
		fields := strings.SplitN(line, "\t", 9)
		if len(fields) < 9 {
			continue
		}

		hostKey := fields[0]
		name := fields[1]
		encryptedHex := fields[2]
		plainValue := fields[3]
		path := fields[4]
		isSecure := fields[5] == "1"
		isHTTPOnly := fields[6] == "1"
		expiresUTC, _ := strconv.ParseInt(fields[7], 10, 64)
		sameSite, _ := strconv.Atoi(fields[8])

		// Try to get the decrypted value.
		var value string
		if plainValue != "" {
			// Some cookies have unencrypted values.
			value = plainValue
		} else if encryptedHex != "" {
			encBytes, err := hex.DecodeString(encryptedHex)
			if err != nil {
				decryptErrors++
				continue
			}
			decrypted, err := decryptCookieValue(encBytes, key)
			if err != nil {
				decryptErrors++
				continue
			}
			value = decrypted
		}

		if value == "" {
			continue
		}

		cookie := &proto.NetworkCookieParam{
			Name:     name,
			Value:    value,
			Domain:   hostKey,
			Path:     path,
			Secure:   isSecure,
			HTTPOnly: isHTTPOnly,
			SameSite: chromeSameSiteToCDP(sameSite),
		}

		if expiresUTC > 0 {
			cookie.Expires = proto.TimeSinceEpoch(chromeEpochToUnix(expiresUTC))
		}

		cookies = append(cookies, cookie)
	}

	if decryptErrors > 0 {
		log.Warnf("failed to decrypt %d cookies (skipped)", decryptErrors)
	}

	log.Infof("read %d cookies from Chrome profile", len(cookies))
	return cookies, nil
}
