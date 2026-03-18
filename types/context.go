package types

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/aliwatters/rod-mcp/types/js"
	"github.com/aliwatters/rod-mcp/utils"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
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
		Set("--disable-features", "HttpsUpgrades").
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

// Mode is the model type, indicates the model type of the tool
type Mode string

const (
	// Vision mode indicates the vision ll model,will load the vision tools
	Vision Mode = "vision"

	// Text mode indicates the no vision ll model,will load the text tools
	Text Mode = "text"
)

const (
	// tempDirSuffixLen is the number of random characters appended to the browser temp dir name.
	tempDirSuffixLen = 10
	// cleanupRetryCount is the number of times to retry removing the browser temp dir on Close.
	cleanupRetryCount = 3
	// cleanupRetryDelay is the wait between browser temp dir removal retries.
	cleanupRetryDelay = 200 * time.Millisecond
)

// ConsoleMessage represents a captured browser console message.
type ConsoleMessage struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// NetworkRequest represents a captured network request with its response.
type NetworkRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Status int    `json:"status"`
	Type   string `json:"type"`
}

type Context struct {
	stdContext       context.Context
	config          Config
	browser         *rod.Browser
	page            *rod.Page
	stateLock       sync.Mutex
	snapshot        *Snapshot
	mode            Mode
	consoleMessages []ConsoleMessage
	networkRequests []NetworkRequest
	// pendingRequests tracks in-flight requests by ID for response correlation.
	pendingRequests map[string]int
	// clonedProfileDir is the temp directory from profile cloning, cleaned up on Close.
	clonedProfileDir string
}

func NewContext(ctx context.Context, cfg Config) *Context {
	return &Context{
		stdContext: ctx,
		config:     cfg,
		mode:       cfg.Mode,
	}
}

func (ctx *Context) EnsurePage() (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	return ctx.page, nil
}

func (ctx *Context) ControlledPage() (*rod.Page, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.page == nil {
		return nil, errors.New("No tab to used, call rod_navigate first")
	}
	return ctx.page, nil
}

func (ctx *Context) initial() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var err error
	if ctx.browser == nil {
		ctx.browser, ctx.clonedProfileDir, err = launchBrowser(ctx.stdContext, ctx.config)
		if err != nil {
			return err
		}
		ctx.page, err = ctx.createPage()
		if err != nil {
			return err
		}
		return nil
	}
	if ctx.page == nil {
		ctx.page, err = ctx.createPage()
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctx *Context) CurrentMode() Mode {
	return ctx.mode
}

func (ctx *Context) Config() Config {
	return ctx.config
}

func (ctx *Context) ClosePage() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.closePage()
}

func (ctx *Context) Execute(handlerFunc server.ToolHandlerFunc, handlerCallOpts ToolHandlerCallOpts) server.ToolHandlerFunc {
	return func(stdCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := handlerFunc(stdCtx, request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if handlerCallOpts.WithSnapshot {
			snapshot, err := ctx.BuildSnapshot()
			if err != nil {
				log.Warnf("Failed to build snapshot: %s", err)
				snapshot = fmt.Sprintf("(snapshot unavailable: %s)", err)
			}
			if snapshot != "" {
				result.Content = append(result.Content, mcp.TextContent{
					Type: "text",
					Text: snapshot,
				})
			}
		}
		return result, nil
	}
}

func (ctx *Context) BuildSnapshot() (string, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.page == nil {
		return "", errors.New("No tab to capture snapshot, call rod_navigate first")
	}
	snapshot, err := BuildSnapshot(ctx.page, ctx.config.CompactSnapshot)
	if err != nil {
		return "", err
	}
	ctx.snapshot = snapshot
	return snapshot.String(), nil
}

func (ctx *Context) LatestSnapshot() (*Snapshot, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.snapshot == nil {
		return nil, errors.New("No snapshot to used, call rod_snapshot first")
	}
	return ctx.snapshot, nil
}

func (ctx *Context) CloseBrowser() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.closeBrowser()
}

func (ctx *Context) closePage() error {
	if ctx.page == nil {
		return nil
	}
	err := ctx.page.Close()
	if err != nil {
		return errors.Wrap(err, "close page failed")
	}
	ctx.page = nil
	return nil
}
func (ctx *Context) closeBrowser() error {
	err := ctx.closePage()
	if err != nil {
		return err
	}

	if ctx.browser == nil {
		return nil
	}

	err = ctx.browser.Close()
	if err != nil {
		return errors.Wrap(err, "close browser failed")
	}
	ctx.browser = nil
	return nil
}

func (ctx *Context) createPage(urls ...string) (*rod.Page, error) {
	targetURL := strings.Join(urls, "/")
	page, err := ctx.browser.Page(proto.TargetCreateTarget{URL: targetURL})
	if err != nil {
		return nil, errors.Wrap(err, "create page failed")
	}
	if _, err := page.EvalOnNewDocument(js.InjectedSnapShot); err != nil {
		return nil, errors.Wrap(err, "inject snapshot script failed")
	}

	// Apply HTTP headers from config (global + domain-specific)
	allHeaders := ctx.config.GetHeadersForURL(targetURL)
	if len(allHeaders) > 0 {
		if _, err := page.SetExtraHeaders(utils.HeaderMapToSlice(allHeaders)); err != nil {
			return nil, errors.Wrap(err, "set extra HTTP headers failed")
		}
	}

	ctx.attachEventListeners(page)

	return page, nil
}

// attachEventListeners registers goroutine-based listeners for console messages and network requests.
func (ctx *Context) attachEventListeners(page *rod.Page) {
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		var parts []string
		for _, arg := range e.Args {
			parts = append(parts, arg.Value.String())
		}
		text := strings.Join(parts, " ")
		ctx.stateLock.Lock()
		ctx.consoleMessages = append(ctx.consoleMessages, ConsoleMessage{
			Level: string(e.Type),
			Text:  text,
		})
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkRequestWillBeSent) {
		ctx.stateLock.Lock()
		if ctx.pendingRequests == nil {
			ctx.pendingRequests = make(map[string]int)
		}
		idx := len(ctx.networkRequests)
		ctx.networkRequests = append(ctx.networkRequests, NetworkRequest{
			Method: e.Request.Method,
			URL:    e.Request.URL,
			Type:   string(e.Type),
		})
		ctx.pendingRequests[string(e.RequestID)] = idx
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkResponseReceived) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.pendingRequests[string(e.RequestID)]; ok {
			ctx.networkRequests[idx].Status = e.Response.Status
			ctx.networkRequests[idx].Type = string(e.Type)
			delete(ctx.pendingRequests, string(e.RequestID))
		}
		ctx.stateLock.Unlock()
	})()
}

// ConsoleMessages returns captured console messages, optionally filtered by level.
// If clear is true, the buffer is emptied after returning.
func (ctx *Context) ConsoleMessages(filterLevel string, clear bool) []ConsoleMessage {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var result []ConsoleMessage
	for _, msg := range ctx.consoleMessages {
		if filterLevel == "" || msg.Level == filterLevel {
			result = append(result, msg)
		}
	}

	if clear {
		if filterLevel == "" {
			ctx.consoleMessages = nil
		} else {
			// Only clear matching messages
			var remaining []ConsoleMessage
			for _, msg := range ctx.consoleMessages {
				if msg.Level != filterLevel {
					remaining = append(remaining, msg)
				}
			}
			ctx.consoleMessages = remaining
		}
	}

	return result
}

// NetworkRequests returns captured network requests, optionally filtered by URL pattern and method.
// If clear is true, the buffer is emptied after returning.
func (ctx *Context) NetworkRequests(filterURL, filterMethod string, clear bool) []NetworkRequest {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var result []NetworkRequest
	for _, req := range ctx.networkRequests {
		if filterURL != "" && !strings.Contains(req.URL, filterURL) {
			continue
		}
		if filterMethod != "" && !strings.EqualFold(req.Method, filterMethod) {
			continue
		}
		result = append(result, req)
	}

	if clear {
		ctx.networkRequests = nil
		ctx.pendingRequests = nil
	}

	return result
}

// TabInfo represents a tab's metadata for listing.
type TabInfo struct {
	Index    int    `json:"index"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	IsActive bool   `json:"is_active"`
}

// NewTab creates a new tab, optionally navigating to a URL, and makes it the active page.
func (ctx *Context) NewTab(url string) (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var page *rod.Page
	var err error
	if url != "" {
		page, err = ctx.createPage(url)
	} else {
		page, err = ctx.createPage()
	}
	if err != nil {
		return nil, err
	}
	ctx.page = page
	return page, nil
}

// ListTabs returns info about all open tabs, marking which is active.
func (ctx *Context) ListTabs() ([]TabInfo, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	pages, err := ctx.browser.Pages()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tabs")
	}

	tabs := make([]TabInfo, 0, len(pages))
	for i, p := range pages {
		info, err := p.Info()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get tab info")
		}
		tabs = append(tabs, TabInfo{
			Index:    i,
			Title:    info.Title,
			URL:      info.URL,
			IsActive: ctx.page != nil && p.TargetID == ctx.page.TargetID,
		})
	}
	return tabs, nil
}

// SelectTab switches the active page to the tab at the given index.
func (ctx *Context) SelectTab(index int) (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	pages, err := ctx.browser.Pages()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tabs")
	}
	if index < 0 || index >= len(pages) {
		return nil, fmt.Errorf("tab index %d out of range (0-%d)", index, len(pages)-1)
	}
	ctx.page = pages[index]
	return ctx.page, nil
}

// CloseTab closes the tab at the given index. If it's the active tab,
// switches to the nearest remaining tab.
func (ctx *Context) CloseTab(index int) error {
	if err := ctx.initial(); err != nil {
		return err
	}
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	pages, err := ctx.browser.Pages()
	if err != nil {
		return errors.Wrap(err, "failed to list tabs")
	}
	if index < 0 || index >= len(pages) {
		return fmt.Errorf("tab index %d out of range (0-%d)", index, len(pages)-1)
	}
	if len(pages) == 1 {
		return errors.New("cannot close the last tab")
	}

	target := pages[index]
	isActive := ctx.page != nil && target.TargetID == ctx.page.TargetID

	if err := target.Close(); err != nil {
		return errors.Wrap(err, "failed to close tab")
	}

	if isActive {
		// Switch to nearest tab
		remaining, err := ctx.browser.Pages()
		if err != nil {
			return errors.Wrap(err, "failed to list tabs after close")
		}
		if len(remaining) > 0 {
			newIndex := index
			if newIndex >= len(remaining) {
				newIndex = len(remaining) - 1
			}
			ctx.page = remaining[newIndex]
		} else {
			ctx.page = nil
		}
	}

	return nil
}

// UpdateHeadersForURL updates the extra HTTP headers on the current page based on the target URL.
// This should be called before navigating to a new domain to ensure domain-specific headers are applied.
func (ctx *Context) UpdateHeadersForURL(url string) error {
	if ctx.page == nil {
		return nil
	}

	allHeaders := ctx.config.GetHeadersForURL(url)
	if len(allHeaders) > 0 {
		if _, err := ctx.page.SetExtraHeaders(utils.HeaderMapToSlice(allHeaders)); err != nil {
			return errors.Wrap(err, "set extra HTTP headers failed")
		}
	}
	return nil
}

// Reconfigure updates browser settings and closes any running browser so the
// next tool call reinitializes with the new configuration.  Pass nil for any
// field to leave it unchanged.
func (ctx *Context) Reconfigure(headless *bool, cdpEndpoint *string) error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	if headless != nil {
		ctx.config.Headless = *headless
	}
	if cdpEndpoint != nil {
		ctx.config.CDPEndpoint = *cdpEndpoint
	}

	// Close existing browser so the next initial() picks up new config.
	return ctx.closeBrowser()
}

// Close the browser
// PS: This method only used because of server exit
func (ctx *Context) Close() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if err := ctx.closeBrowser(); err != nil {
		log.Warnf("close browser: %s", err)
	}

	// remove cloned profile dir if we created one
	if ctx.clonedProfileDir != "" {
		if err := os.RemoveAll(ctx.clonedProfileDir); err != nil {
			log.Warnf("remove cloned profile dir: %s", err)
		} else {
			log.Infof("cleaned up cloned profile: %s", ctx.clonedProfileDir)
		}
	}

	// remove browser temp dir, retrying briefly to handle race with browser shutdown
	if ctx.config.BrowserTempDir != "" && ctx.config.CDPEndpoint == "" {
		var lastErr error
		for range cleanupRetryCount {
			if err := os.RemoveAll(ctx.config.BrowserTempDir); err == nil {
				lastErr = nil
				break
			} else {
				lastErr = err
				time.Sleep(cleanupRetryDelay)
			}
		}
		if lastErr != nil {
			log.Warnf("remove browser temp dir: %s", lastErr)
		}
	}
	return nil
}
