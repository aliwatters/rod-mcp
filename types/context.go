package types

import (
	"context"
	"errors"
	"fmt"
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
)

type Context struct {
	stdContext       context.Context
	config          Config
	browser         *rod.Browser
	page            *rod.Page
	// pageCancel cancels event-listener goroutines attached to the current page.
	// It is set when attachEventListeners creates a cancelable page copy, and
	// called in closePage before the page is closed.
	pageCancel      func()
	stateLock       sync.Mutex
	snapshot        *Snapshot
	mode            Mode
	consoleMessages *RingBuffer[ConsoleMessage]
	networkRequests *RingBuffer[NetworkRequest]
	// pendingRequests tracks in-flight requests by ID for response correlation.
	// Values are indices into the networkRequests ring buffer's internal items slice;
	// they remain valid because the ring buffer never shifts elements.
	pendingRequests map[string]int
	// Intercept state
	interceptRules   []InterceptRule
	interceptEnabled bool
	// interceptCancel cancels the EachEvent goroutine started by interceptEnable.
	interceptCancel func()
	// WebSocket tracking
	wsConnections *RingBuffer[WebSocketConnection]
	wsConnIndex   map[string]int // requestID → internal ring-buffer index in wsConnections
	wsFrames      *RingBuffer[WebSocketFrame]
	// clonedProfileDir is the temp directory from profile cloning, cleaned up on Close.
	clonedProfileDir string
	// keepaliveCancel stops the CDP keepalive goroutine when the browser is closed.
	keepaliveCancel func()
}

func NewContext(ctx context.Context, cfg Config) *Context {
	return &Context{
		stdContext:      ctx,
		config:          cfg,
		mode:            cfg.Mode,
		consoleMessages: NewRingBuffer[ConsoleMessage](maxConsoleMessages),
		networkRequests: NewRingBuffer[NetworkRequest](maxNetworkRequests),
		wsConnections:   NewRingBuffer[WebSocketConnection](maxWSConnections),
		wsFrames:        NewRingBuffer[WebSocketFrame](maxWSFrames),
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
	if err := ctx.initLocked(); err != nil {
		return nil, err
	}
	if ctx.page == nil {
		return nil, errors.New("no active tab, call rod_navigate first")
	}
	return ctx.page, nil
}

// ControlledBrowser returns the browser instance, or an error if no browser is running.
func (ctx *Context) ControlledBrowser() (*rod.Browser, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if err := ctx.initLocked(); err != nil {
		return nil, err
	}
	if ctx.browser == nil {
		return nil, errors.New("no browser running, call rod_navigate first")
	}
	return ctx.browser, nil
}

func (ctx *Context) initial() error {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.initLocked()
}

// initLocked performs lazy initialization of the browser and page.
// Must be called with stateLock held.
func (ctx *Context) initLocked() error {
	var err error
	if ctx.browser == nil {
		ctx.browser, ctx.clonedProfileDir, err = launchBrowser(ctx.stdContext, ctx.config)
		if err != nil {
			return err
		}
		ctx.startKeepalive()
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
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	if ctx.page == nil {
		return "", errors.New("no active tab, call rod_navigate first")
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
		return nil, errors.New("no snapshot available, call rod_snapshot first")
	}
	return ctx.snapshot, nil
}

// InvalidateSnapshot marks the current snapshot as stale by clearing it.
// Call this at the start of any handler that modifies the DOM so that the
// Execute wrapper (or the next EnsureSnapshot call) rebuilds a fresh snapshot.
func (ctx *Context) InvalidateSnapshot() {
	ctx.stateLock.Lock()
	ctx.snapshot = nil
	ctx.stateLock.Unlock()
}

// EnsureSnapshot returns the latest snapshot, building one if none exists.
func (ctx *Context) EnsureSnapshot() (*Snapshot, error) {
	ctx.stateLock.Lock()
	if ctx.snapshot != nil {
		s := ctx.snapshot
		ctx.stateLock.Unlock()
		return s, nil
	}
	ctx.stateLock.Unlock()

	// Build a new snapshot (BuildSnapshot acquires stateLock internally).
	if _, err := ctx.BuildSnapshot(); err != nil {
		return nil, err
	}
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
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
	// Cancel event listener goroutines before closing the page to avoid
	// goroutine leaks and use-after-close races on the page's CDP session.
	if ctx.pageCancel != nil {
		ctx.pageCancel()
		ctx.pageCancel = nil
	}
	// Cancel any intercept listener goroutine as well.
	if ctx.interceptCancel != nil {
		ctx.interceptCancel()
		ctx.interceptCancel = nil
	}
	err := ctx.page.Close()
	if err != nil {
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
		return err
	}

	if ctx.browser == nil {
		return nil
	}

	err = ctx.browser.Close()
	if err != nil {
		return fmt.Errorf("close browser: %w", err)
	}
	ctx.browser = nil
	return nil
}

// keepaliveInterval is how often the CDP keepalive ping is sent.
// Chrome's DevTools protocol WebSocket has no built-in keepalive; idle
// connections can be dropped by the OS or intermediate proxies after ~15 min.
const keepaliveInterval = 5 * time.Minute

// startKeepalive launches a background goroutine that periodically sends a
// lightweight CDP call (Browser.getVersion) to prevent the WebSocket from
// going idle and being killed. Must be called with stateLock held (or before
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
	page, err := ctx.browser.Page(proto.TargetCreateTarget{URL: targetURL})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	if _, err := page.EvalOnNewDocument(js.InjectedSnapShot); err != nil {
		return nil, fmt.Errorf("inject snapshot script: %w", err)
	}

	// Apply HTTP headers from config (global + domain-specific)
	allHeaders := ctx.config.GetHeadersForURL(targetURL)
	if len(allHeaders) > 0 {
		if _, err := page.SetExtraHeaders(utils.HeaderMapToSlice(allHeaders)); err != nil {
			return nil, fmt.Errorf("set extra HTTP headers: %w", err)
		}
	}

	cancel := ctx.attachEventListeners(page)
	ctx.pageCancel = cancel

	return page, nil
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
			return fmt.Errorf("set extra HTTP headers: %w", err)
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
