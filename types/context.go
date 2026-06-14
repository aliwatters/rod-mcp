package types

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types/js"
	"github.com/aliwatters/rod-mcp/utils"
)

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
	// launchBackoffInitial is the first delay after a managed local browser launch failure.
	launchBackoffInitial = time.Second
	// launchBackoffMax caps managed local browser launch retry delays.
	launchBackoffMax = 30 * time.Second
)

type Context struct {
	stdContext context.Context

	// browserLock serialises access to all browser-lifecycle fields:
	// browser, page, pageCancel, keepaliveCancel, clonedProfileDir, and config.
	browserLock sync.Mutex
	config      Config
	browser     *rod.Browser
	page        *rod.Page
	// pageCancel cancels event-listener goroutines attached to the current page.
	// It is set when attachEventListeners creates a cancelable page copy, and
	// called in closePage before the page is closed.
	pageCancel func()
	// keepaliveCancel stops the CDP keepalive goroutine when the browser is closed.
	keepaliveCancel func()
	// clonedProfileDir is the temp directory from profile cloning, cleaned up on Close.
	clonedProfileDir string
	// launchFailures tracks consecutive managed local Chrome launch failures.
	launchFailures int
	// nextLaunchAt is the earliest time a managed local Chrome launch may be retried.
	nextLaunchAt time.Time
	// lastLaunchErr is returned while managed local Chrome launch backoff is active.
	lastLaunchErr error
	// instanceLock prevents multiple rod-mcp processes from silently sharing
	// explicit browser resources such as a CDP endpoint or debug port.
	instanceLock *instanceLock

	// snapshotLock serialises access to the cached ARIA snapshot.
	// Lock ordering: always acquire snapshotLock before browserLock if both are
	// needed (EnsureSnapshot reads page under a brief browserLock while holding
	// snapshotLock).
	snapshotLock sync.Mutex
	snapshot     *Snapshot

	// eventsLock serialises access to the event collector (console messages,
	// network requests, WebSocket frames). Kept separate from browserLock so
	// event listener goroutines do not contend with browser lifecycle operations.
	eventsLock sync.Mutex
	events     *eventCollector

	mode Mode

	// interceptLock serialises access to the network interceptor state.
	// Kept separate from browserLock to allow intercept configuration without
	// contending with browser lifecycle operations.
	interceptLock sync.Mutex
	intercept     *networkInterceptor
}

var browserLaunchNow = time.Now

func NewContext(ctx context.Context, cfg Config) *Context {
	return &Context{
		stdContext: ctx,
		config:     cfg,
		mode:       cfg.Mode,
		events:     newEventCollector(),
		intercept:  newNetworkInterceptor(),
	}
}

func (ctx *Context) EnsurePage() (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	return ctx.page, nil
}

func (ctx *Context) ControlledPage() (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	if ctx.page == nil {
		return nil, errors.New("no active tab, call rod_navigate first")
	}
	return ctx.page, nil
}

func (ctx *Context) ActivePage() (*rod.Page, error) {
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	if ctx.page == nil {
		return nil, errors.New("no active page, call rod_navigate first, or configure domainHeaders in rod-mcp.yaml")
	}
	return ctx.page, nil
}

// ControlledBrowser returns the browser instance, or an error if no browser is running.
func (ctx *Context) ControlledBrowser() (*rod.Browser, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	if ctx.browser == nil {
		return nil, errors.New("no browser running, call rod_navigate first")
	}
	return ctx.browser, nil
}

func (ctx *Context) initial() error {
	ctx.browserLock.Lock()
	recovered, err := ctx.initLocked()
	ctx.browserLock.Unlock()
	if recovered {
		ctx.InvalidateSnapshot()
	}
	return err
}

// initLocked performs lazy initialization of the browser and page.
// Must be called with browserLock held.
func (ctx *Context) initLocked() (bool, error) {
	var err error
	recovered := false
	launchReason := "initial browser launch"
	if ctx.browser != nil {
		if err := ctx.checkBrowserAliveLocked(); err != nil {
			if !isClosedBrowserSessionError(err) {
				return false, fmt.Errorf("check browser session: %w", err)
			}
			log.Warnf("browser session ended or crashed; relaunching on next action: %s", err)
			if closeErr := ctx.closeBrowser(); closeErr != nil {
				log.Warnf("drop stale browser session after close error: %s", closeErr)
				ctx.dropBrowserStateLocked()
			}
			recovered = true
			launchReason = "relaunch after ended browser session"
		}
	}
	if ctx.browser == nil {
		if err := ctx.launchBrowserLocked(launchReason); err != nil {
			return recovered, err
		}
		ctx.startKeepalive()
		ctx.page, err = ctx.createPage()
		if err != nil {
			pageErr := fmt.Errorf("create initial page: %w", err)
			log.Warnf("browser launch failed after connect: reason=%s error=%s", launchReason, pageErr)
			if ctx.isManagedLocalLaunchLocked() {
				ctx.recordLaunchFailureLocked(launchReason, pageErr)
			}
			if closeErr := ctx.closeBrowser(); closeErr != nil {
				log.Warnf("close browser after initial page failure: %s", closeErr)
				ctx.dropBrowserStateLocked()
			}
			return recovered, fmt.Errorf("create initial page: %w", err)
		}
		ctx.recordLaunchSuccessLocked(launchReason)
		return recovered, nil
	}
	if ctx.page == nil {
		ctx.page, err = ctx.createPage()
		if err != nil {
			return recovered, fmt.Errorf("create page: %w", err)
		}
	}

	return recovered, nil
}

func (ctx *Context) isManagedLocalLaunchLocked() bool {
	return ctx.config.CDPEndpoint == ""
}

func (ctx *Context) launchBrowserLocked(reason string) error {
	if ctx.isManagedLocalLaunchLocked() {
		if err := ctx.checkLaunchBackoffLocked(reason); err != nil {
			return err
		}
	}

	log.Infof(
		"browser launch starting: reason=%s mode=%s headless=%t cdp=%t",
		reason,
		ctx.config.Mode,
		ctx.config.Headless,
		ctx.config.CDPEndpoint != "",
	)
	if err := ctx.acquireInstanceLockLocked(); err != nil {
		return err
	}
	launchCtx, cancel := context.WithTimeout(ctx.stdContext, ctx.config.LaunchTimeout())
	defer cancel()
	browser, clonedDir, err := launchBrowserFunc(launchCtx, ctx.config)
	if err != nil {
		log.Warnf("browser launch failed: reason=%s error=%s", reason, err)
		ctx.releaseInstanceLockLocked()
		if ctx.isManagedLocalLaunchLocked() {
			ctx.recordLaunchFailureLocked(reason, err)
		}
		return fmt.Errorf("launch browser: %w", err)
	}
	// launchCtx bounds only the launch/connect OPERATION (so a hung launch fails
	// fast). Re-bind the browser's LIFETIME context to the long-lived stdContext
	// so `defer cancel()` does not immediately close the just-launched browser
	// (which would make every later op fail with "context canceled").
	browser = browser.Context(ctx.stdContext)
	ctx.browser = browser
	ctx.clonedProfileDir = clonedDir
	return nil
}

func (ctx *Context) acquireInstanceLockLocked() error {
	if ctx.instanceLock != nil {
		return nil
	}
	lock, err := acquireInstanceLock(ctx.config)
	if err != nil {
		return err
	}
	ctx.instanceLock = lock
	return nil
}

func (ctx *Context) releaseInstanceLockLocked() {
	if ctx.instanceLock == nil {
		return
	}
	if err := ctx.instanceLock.Release(); err != nil {
		log.Warnf("release browser instance lock: %s", err)
	}
	ctx.instanceLock = nil
}

func (ctx *Context) checkLaunchBackoffLocked(reason string) error {
	if ctx.nextLaunchAt.IsZero() || !browserLaunchNow().Before(ctx.nextLaunchAt) {
		return nil
	}
	remaining := ctx.nextLaunchAt.Sub(browserLaunchNow()).Round(time.Millisecond)
	if remaining < 0 {
		remaining = 0
	}
	log.Warnf(
		"browser launch suppressed during backoff: reason=%s failures=%d retryAfter=%s remaining=%s",
		reason,
		ctx.launchFailures,
		ctx.nextLaunchAt.Format(time.RFC3339),
		remaining,
	)
	if ctx.lastLaunchErr == nil {
		return fmt.Errorf("browser launch suppressed after %d failed attempt(s); retry after %s", ctx.launchFailures, ctx.nextLaunchAt.Format(time.RFC3339))
	}
	return fmt.Errorf("browser launch suppressed after %d failed attempt(s); retry after %s: %w", ctx.launchFailures, ctx.nextLaunchAt.Format(time.RFC3339), ctx.lastLaunchErr)
}

func (ctx *Context) recordLaunchFailureLocked(reason string, err error) {
	ctx.launchFailures++
	delay := launchBackoffInitial
	for i := 1; i < ctx.launchFailures && delay < launchBackoffMax; i++ {
		delay *= 2
		if delay > launchBackoffMax {
			delay = launchBackoffMax
			break
		}
	}
	ctx.nextLaunchAt = browserLaunchNow().Add(delay)
	ctx.lastLaunchErr = err
	log.Warnf(
		"browser launch backoff set: reason=%s failures=%d delay=%s retryAfter=%s error=%s",
		reason,
		ctx.launchFailures,
		delay,
		ctx.nextLaunchAt.Format(time.RFC3339),
		err,
	)
}

func (ctx *Context) recordLaunchSuccessLocked(reason string) {
	if ctx.launchFailures > 0 {
		log.Infof("browser launch recovered: reason=%s previousFailures=%d", reason, ctx.launchFailures)
	}
	ctx.launchFailures = 0
	ctx.nextLaunchAt = time.Time{}
	ctx.lastLaunchErr = nil
	log.Infof(
		"browser launch succeeded: reason=%s mode=%s headless=%t cdp=%t",
		reason,
		ctx.config.Mode,
		ctx.config.Headless,
		ctx.config.CDPEndpoint != "",
	)
}

func (ctx *Context) checkBrowserAliveLocked() error {
	checkCtx, cancel := context.WithTimeout(ctx.stdContext, 5*time.Second)
	defer cancel()
	_, err := proto.BrowserGetVersion{}.Call(ctx.browser.Context(checkCtx))
	return err
}

func isClosedBrowserSessionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"use of closed network connection",
		"browser has been closed",
		"connection reset by peer",
		"connection is closed",
		"broken pipe",
		"session closed",
		"target closed",
		"websocket: close",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func isRecoverableBrowserSessionError(err error) bool {
	if err == nil {
		return false
	}
	if isClosedBrowserSessionError(err) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"context deadline exceeded",
		"i/o timeout",
		"timed out",
		"timeout",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func (ctx *Context) RecoverBrowserAfterError(err error) bool {
	if !isRecoverableBrowserSessionError(err) {
		return false
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	if ctx.browser == nil && ctx.page == nil {
		return false
	}
	log.Warnf("browser session appears wedged; dropping state so the next action relaunches: %s", err)
	ctx.dropBrowserStateLocked()
	return true
}

func (ctx *Context) dropBrowserStateLocked() {
	if ctx.keepaliveCancel != nil {
		ctx.keepaliveCancel()
		ctx.keepaliveCancel = nil
	}
	if ctx.pageCancel != nil {
		ctx.pageCancel()
		ctx.pageCancel = nil
	}
	ctx.intercept.Cancel()
	ctx.page = nil
	ctx.browser = nil
}

func (ctx *Context) CurrentMode() Mode {
	return ctx.mode
}

func (ctx *Context) Config() Config {
	return ctx.config
}

func (ctx *Context) ClosePage() error {
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	return ctx.closePage()
}

func (ctx *Context) Execute(handlerFunc server.ToolHandlerFunc, handlerCallOpts ToolHandlerCallOpts) server.ToolHandlerFunc {
	return func(stdCtx context.Context, request mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Errorf("Tool handler panic: %v", recovered)
				result = mcp.NewToolResultError(fmt.Sprintf("tool handler panic: %v", recovered))
				err = nil
			}
		}()

		result, err = handlerFunc(stdCtx, request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if handlerCallOpts.WithSnapshot {
			// Invalidate the snapshot after DOM-mutating handlers so stale
			// refs are never reused.  Handlers are free to read the
			// previous snapshot for ref-based element resolution during
			// their execution because invalidation happens here, after
			// they return.  Read-only handlers (WithSnapshot: false) do
			// NOT invalidate, preserving the cached snapshot for
			// subsequent ref-based tool calls.
			ctx.InvalidateSnapshot()
			snap, snapErr := ctx.EnsureSnapshot()
			var snapshotText string
			if snapErr != nil {
				log.Warnf("Failed to build snapshot: %s", snapErr)
				snapshotText = fmt.Sprintf("(snapshot unavailable: %s)", snapErr)
			} else {
				snapshotText = snap.String()
			}
			if snapshotText != "" {
				result.Content = append(result.Content, mcp.TextContent{
					Type: "text",
					Text: snapshotText,
				})
			}
		}
		return result, nil
	}
}

func (ctx *Context) BuildSnapshot() (string, error) {
	ctx.snapshotLock.Lock()
	defer ctx.snapshotLock.Unlock()
	return ctx.buildSnapshotLocked()
}

// buildSnapshotLocked builds a fresh snapshot and stores it.
// Must be called with snapshotLock held. Reads page and config under a brief
// browserLock acquisition to avoid holding two locks simultaneously.
func (ctx *Context) buildSnapshotLocked() (string, error) {
	// Read browser state under browserLock and immediately release it.
	ctx.browserLock.Lock()
	page := ctx.page
	compact := ctx.config.CompactSnapshot
	ctx.browserLock.Unlock()

	if page == nil {
		return "", errors.New("no active tab, call rod_navigate first")
	}
	snapshot, err := BuildSnapshot(page, compact)
	if err != nil {
		return "", err
	}
	ctx.snapshot = snapshot
	return snapshot.String(), nil
}

func (ctx *Context) LatestSnapshot() (*Snapshot, error) {
	ctx.snapshotLock.Lock()
	defer ctx.snapshotLock.Unlock()
	if ctx.snapshot == nil {
		return nil, errors.New("no snapshot available, call rod_snapshot first")
	}
	return ctx.snapshot, nil
}

// InvalidateSnapshot marks the current snapshot as stale by clearing it.
// Call this at the start of any handler that modifies the DOM so that the
// Execute wrapper (or the next EnsureSnapshot call) rebuilds a fresh snapshot.
func (ctx *Context) InvalidateSnapshot() {
	ctx.snapshotLock.Lock()
	ctx.snapshot = nil
	ctx.snapshotLock.Unlock()
}

// EnsureSnapshot returns the latest snapshot, building one if none exists.
// The check-then-build is performed atomically under snapshotLock to eliminate
// the TOCTOU window that existed when the lock was released between the nil check
// and the build call.
func (ctx *Context) EnsureSnapshot() (*Snapshot, error) {
	ctx.snapshotLock.Lock()
	defer ctx.snapshotLock.Unlock()
	if ctx.snapshot != nil {
		return ctx.snapshot, nil
	}
	if _, err := ctx.buildSnapshotLocked(); err != nil {
		return nil, err
	}
	return ctx.snapshot, nil
}

func (ctx *Context) CloseBrowser() error {
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
	return ctx.closeBrowser()
}

func (ctx *Context) closePage() error {
	if ctx.page == nil {
		return nil
	}
	// Cancel event listener goroutines before closing the page to avoid
	// goroutine leaks and use-after-close races on the page's CDP session.
	if ctx.pageCancel != nil {
		ctx.pageCancel()
		ctx.pageCancel = nil
	}
	// Cancel any intercept listener goroutine as well.
	ctx.intercept.Cancel()
	err := ctx.page.Close()
	if err != nil {
		if isClosedBrowserSessionError(err) {
			ctx.page = nil
			return nil
		}
		return fmt.Errorf("close page: %w", err)
	}
	ctx.page = nil
	return nil
}

func (ctx *Context) closeBrowser() error {
	// Stop the keepalive goroutine before closing the browser.
	if ctx.keepaliveCancel != nil {
		ctx.keepaliveCancel()
		ctx.keepaliveCancel = nil
	}

	err := ctx.closePage()
	if err != nil {
		return fmt.Errorf("close browser: %w", err)
	}

	if ctx.browser == nil {
		ctx.releaseInstanceLockLocked()
		return nil
	}

	err = ctx.browser.Close()
	if err != nil {
		if isClosedBrowserSessionError(err) {
			ctx.browser = nil
			ctx.releaseInstanceLockLocked()
			return nil
		}
		return fmt.Errorf("close browser: %w", err)
	}
	ctx.browser = nil
	ctx.releaseInstanceLockLocked()
	return nil
}

// keepaliveInterval is how often the CDP keepalive ping is sent.
// Chrome's DevTools protocol WebSocket has no built-in keepalive; idle
// connections can be dropped by the OS or intermediate proxies after ~15 min.
const keepaliveInterval = 5 * time.Minute

// startKeepalive launches a background goroutine that periodically sends a
// lightweight CDP call (Browser.getVersion) to prevent the WebSocket from
// going idle and being killed. Must be called with browserLock held (or before
// concurrent access begins). The goroutine stops when keepaliveCancel is called.
func (ctx *Context) startKeepalive() {
	if ctx.browser == nil {
		return
	}
	stopCtx, cancel := context.WithCancel(ctx.stdContext)
	ctx.keepaliveCancel = cancel

	// Use a browser handle bound to stopCtx so CDP calls are interrupted
	// when keepaliveCancel is called, preventing goroutine leaks on stalled
	// WebSocket connections.
	browser := ctx.browser.Context(stopCtx)
	go func() {
		ticker := time.NewTicker(keepaliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stopCtx.Done():
				return
			case <-ticker.C:
				// Send a cheap CDP call to keep the WebSocket alive.
				// Uses a per-ping timeout so a stalled connection doesn't
				// block the goroutine until the next tick.
				pingCtx, pingCancel := context.WithTimeout(stopCtx, 10*time.Second)
				_, err := proto.BrowserGetVersion{}.Call(browser.Context(pingCtx))
				pingCancel()
				if err != nil {
					log.Debugf("CDP keepalive ping failed: %s", err)
				}
			}
		}
	}()
}

func (ctx *Context) createPage(urls ...string) (*rod.Page, error) {
	targetURL := strings.Join(urls, "/")
	pageCtx, cancel := context.WithTimeout(ctx.stdContext, ctx.config.LaunchTimeout())
	defer cancel()
	browser := ctx.browser.Context(pageCtx)
	page, err := browser.Page(proto.TargetCreateTarget{URL: targetURL})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	setupPage := page.Context(pageCtx)
	if _, err := setupPage.EvalOnNewDocument(js.InjectedSnapShot); err != nil {
		return nil, fmt.Errorf("inject snapshot script: %w", err)
	}

	// Inject stealth patches before any page JS runs.
	if ctx.config.Stealth {
		if _, err := setupPage.EvalOnNewDocument(js.StealthJS); err != nil {
			return nil, fmt.Errorf("inject stealth script: %w", err)
		}
		log.Debugf("stealth mode: injected anti-detection script on new page")
	}

	// When stealth is enabled, auto-set a realistic User-Agent and Sec-CH-UA
	// header matching the running Chrome version, unless the user already
	// configured those headers.
	allHeaders := ctx.config.GetHeadersForURL(targetURL)
	if ctx.config.Stealth {
		allHeaders = ctx.applyStealthHeaders(allHeaders)
	}

	if len(allHeaders) > 0 {
		if _, err := setupPage.SetExtraHeaders(utils.HeaderMapToSlice(allHeaders)); err != nil {
			return nil, fmt.Errorf("set extra HTTP headers: %w", err)
		}
	}

	page = page.Context(ctx.stdContext)
	pageCancel := ctx.attachEventListeners(page)
	ctx.pageCancel = pageCancel

	return page, nil
}

// applyStealthHeaders sets realistic User-Agent and Sec-CH-UA headers derived
// from the running Chrome version. Existing user-configured headers take
// precedence — stealth only fills in missing values.
func (ctx *Context) applyStealthHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}

	// Fetch the Chrome version from the running browser.
	chromeVersion := ctx.chromeVersion()

	// Only set User-Agent if not already configured (case-insensitive check
	// since HTTP header names are case-insensitive).
	if !headerExists(headers, "User-Agent") {
		if chromeVersion != "" {
			headers["User-Agent"] = fmt.Sprintf(
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
				chromeVersion,
			)
		}
	}

	// Only set Sec-CH-UA if not already configured (case-insensitive).
	if !headerExists(headers, "Sec-CH-UA") {
		major := chromeMajorVersion(chromeVersion)
		if major != "" {
			headers["Sec-CH-UA"] = fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99"`, major, major)
		}
	}

	return headers
}

// chromeVersion returns the full Chrome version string (e.g. "124.0.6367.91")
// from the running browser, or an empty string if unavailable.
func (ctx *Context) chromeVersion() string {
	if ctx.browser == nil {
		return ""
	}
	res, err := proto.BrowserGetVersion{}.Call(ctx.browser)
	if err != nil {
		log.Debugf("stealth: failed to get browser version: %s", err)
		return ""
	}
	// Product is typically "Chrome/124.0.6367.91" or "HeadlessChrome/124..."
	product := res.Product
	if idx := strings.Index(product, "/"); idx >= 0 {
		return product[idx+1:]
	}
	return product
}

// headerExists checks whether a header name is present in the map using a
// case-insensitive comparison, since HTTP header names are case-insensitive.
func headerExists(headers map[string]string, name string) bool {
	for k := range headers {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}

// chromeMajorVersion extracts the major version number from a full Chrome
// version string (e.g. "124.0.6367.91" → "124").
func chromeMajorVersion(version string) string {
	if version == "" {
		return ""
	}
	if idx := strings.Index(version, "."); idx > 0 {
		return version[:idx]
	}
	return version
}

// UpdateHeadersForURL updates the extra HTTP headers on the current page based on the target URL.
// This should be called before navigating to a new domain to ensure domain-specific headers are applied.
func (ctx *Context) UpdateHeadersForURL(url string) error {
	if ctx.page == nil {
		return nil
	}

	allHeaders := ctx.config.GetHeadersForURL(url)
	if ctx.config.Stealth {
		allHeaders = ctx.applyStealthHeaders(allHeaders)
	}
	if len(allHeaders) > 0 {
		if _, err := ctx.page.SetExtraHeaders(utils.HeaderMapToSlice(allHeaders)); err != nil {
			return fmt.Errorf("set extra HTTP headers: %w", err)
		}
	}
	return nil
}

// Reconfigure updates browser settings and closes any running browser so the
// next tool call reinitializes with the new configuration.  Pass nil for any
// field to leave it unchanged.
func (ctx *Context) Reconfigure(headless *bool, cdpEndpoint *string, stealth *bool) error {
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()

	if headless != nil {
		ctx.config.Headless = *headless
	}
	if cdpEndpoint != nil {
		ctx.config.CDPEndpoint = *cdpEndpoint
	}
	if stealth != nil {
		ctx.config.Stealth = *stealth
	}

	// Close existing browser so the next initial() picks up new config.
	return ctx.closeBrowser()
}

// Close the browser
// PS: This method only used because of server exit
func (ctx *Context) Close() error {
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()
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
