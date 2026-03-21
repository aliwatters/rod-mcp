package types

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/pkg/errors"

	"github.com/aliwatters/rod-mcp/utils"
)

// launchBrowser starts a new Chrome instance. When a user-data-dir is configured
// and cloning is enabled (the default), the profile is cloned to a temp directory
// whose path is returned as clonedDir so the caller can clean it up on exit.
func launchBrowser(ctx context.Context, cfg Config) (browser *rod.Browser, clonedDir string, err error) {

	if cfg.CDPEndpoint != "" {
		b, err := controlBrowser(ctx, cfg.CDPEndpoint)
		return b, "", err
	}

	// Validate flag combinations.
	if cfg.UserDataDir == "" && (cfg.NoClone || cfg.CloneAll || len(cfg.CloneDomains) > 0) {
		return nil, "", fmt.Errorf("--no-clone, --clone-all, and --clone-domains require --user-data-dir")
	}
	if cfg.NoClone && cfg.CloneAll {
		return nil, "", fmt.Errorf("--no-clone and --clone-all are mutually exclusive")
	}

	// Determine the user data directory for Chrome.
	var userDataDir string
	if cfg.UserDataDir != "" {
		if cfg.NoClone {
			// Use the profile directly — user accepted the risk.
			log.Warnf("--no-clone: using profile directly at %s (Chrome must not be running with this profile)", cfg.UserDataDir)
			userDataDir = cfg.UserDataDir
		} else if cfg.CloneAll {
			// Full recursive clone — slow but complete.
			log.Warnf("--clone-all: cloning ENTIRE Chrome profile — this includes passwords, history, extensions, and all browser data")
			clonedDir, err = cloneProfileFull(cfg.UserDataDir)
			if err != nil {
				return nil, "", fmt.Errorf("full profile clone: %w", err)
			}
			userDataDir = clonedDir
		} else {
			// Default: selective clone with domain-scoped cookies.
			clonedDir, err = cloneProfile(cfg.UserDataDir, cfg.CloneDomains)
			if err != nil {
				return nil, "", fmt.Errorf("profile clone: %w", err)
			}
			userDataDir = clonedDir
		}
	} else {
		if cfg.BrowserTempDir == "" {
			cfg.BrowserTempDir = DefaultBrowserTempDir
		}
		// browser must own a unique temp dir
		userDataDir = fmt.Sprintf("%s/%s", cfg.BrowserTempDir, utils.RandomString(tempDirSuffixLen))
	}

	browserLauncher := launcher.New().
		Context(ctx).
		Headless(cfg.Headless).
		NoSandbox(cfg.NoSandbox).
		Set("no-gpu").
		Set("--no-first-run").
		Set("ignore-certificate-errors").
		Set("disable-xss-auditor", "true").
		Set("disable-popup-blocking").
		Set("mute-audio", "true").
		Set("use-mock-keychain").
		Set("--remote-allow-origins", "*").
		Set("--disable-dev-shm-usage").
		Set("--disable-features", "HttpsUpgrades,PasswordLeakDetection").
		UserDataDir(userDataDir)

	if cfg.BrowserBinPath != "" {
		browserLauncher.Bin(cfg.BrowserBinPath)
	} else {
		if browserPath, has := launcher.LookPath(); has {
			browserLauncher.Bin(browserPath)
		} else {
			return nil, "", errors.New("Chrome not found; set browserBinPath in config or install Chrome/Chromium")
		}
	}

	if cfg.ChromeDebugPort != "" {
		port, err := strconv.Atoi(cfg.ChromeDebugPort)
		if err != nil || port < 1 || port > 65535 {
			return nil, "", fmt.Errorf("invalid chrome-debug-port %q: must be 1-65535", cfg.ChromeDebugPort)
		}
		browserLauncher.Set("remote-debugging-port", cfg.ChromeDebugPort)
	}

	if cfg.Proxy != "" {
		browserLauncher.Proxy(cfg.Proxy)
	}

	controlUrl, err := browserLauncher.Launch()
	if err != nil {
		return nil, "", errors.Wrap(err, "launch local browser failed")
	}
	b, err := controlBrowser(ctx, controlUrl)
	if err != nil {
		return nil, "", err
	}

	// Inject decrypted cookies via CDP when using profile cloning (not --no-clone).
	// The Cookies DB is encrypted by the OS, so we decrypt from the source profile
	// and inject into the fresh browser via CDP.
	if cfg.UserDataDir != "" && !cfg.NoClone {
		cookies, err := ReadChromeCookies(cfg.UserDataDir, cfg.CloneDomains)
		if err != nil {
			log.Warnf("cookie injection: %s (browser will start without cookies)", err)
		} else if len(cookies) > 0 {
			if err := b.SetCookies(cookies); err != nil {
				log.Warnf("cookie injection via CDP failed: %s", err)
			} else {
				log.Infof("injected %d cookies via CDP", len(cookies))
			}
		}
	}

	return b, clonedDir, nil
}

func controlBrowser(ctx context.Context, controlURL string) (*rod.Browser, error) {
	browser := rod.New().Context(ctx)
	err := browser.ControlURL(controlURL).Connect()
	if err != nil {
		closeErr := browser.Close()
		if closeErr != nil {
			return nil, errors.Wrap(closeErr, "close browser after connect failure")
		}
		return nil, errors.Wrap(err, "error connecting to browser")
	}
	if err := browser.IgnoreCertErrors(true); err != nil {
		return nil, errors.Wrap(err, "failed to set ignore certificate errors")
	}
	return browser, nil
}
