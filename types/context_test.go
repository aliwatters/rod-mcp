package types

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
)

// newTestContext creates a Context with a default Config suitable for unit tests.
// No browser is launched.
func newTestContext() *Context {
	cfg := Config{
		Mode:     Text,
		Headless: true,
	}
	return NewContext(context.Background(), cfg)
}

// ---------------------------------------------------------------------------
// NewContext
// ---------------------------------------------------------------------------

func TestNewContext_InitialState(t *testing.T) {
	ctx := newTestContext()

	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}
	if ctx.mode != Text {
		t.Errorf("mode = %q, want %q", ctx.mode, Text)
	}
	if ctx.browser != nil {
		t.Error("browser should be nil before first use")
	}
	if ctx.page != nil {
		t.Error("page should be nil before first use")
	}
	if ctx.snapshot != nil {
		t.Error("snapshot should be nil on creation")
	}
	if ctx.events == nil {
		t.Error("events collector should be initialized")
	}
	if ctx.events.consoleMessages == nil {
		t.Error("consoleMessages ring buffer should be initialized")
	}
	if ctx.events.networkRequests == nil {
		t.Error("networkRequests ring buffer should be initialized")
	}
	if ctx.events.wsFrames == nil {
		t.Error("wsFrames ring buffer should be initialized")
	}
}

func TestNewContext_ModeFromConfig(t *testing.T) {
	cfg := Config{Mode: Vision}
	ctx := NewContext(context.Background(), cfg)
	if ctx.mode != Vision {
		t.Errorf("mode = %q, want %q", ctx.mode, Vision)
	}
}

// ---------------------------------------------------------------------------
// CurrentMode / Config
// ---------------------------------------------------------------------------

func TestCurrentMode(t *testing.T) {
	ctx := newTestContext()
	if ctx.CurrentMode() != Text {
		t.Errorf("CurrentMode = %q, want %q", ctx.CurrentMode(), Text)
	}
}

func TestConfig_ReturnsConfigCopy(t *testing.T) {
	cfg := Config{Mode: Text, Headless: true}
	ctx := NewContext(context.Background(), cfg)
	got := ctx.Config()
	if got.Mode != Text {
		t.Errorf("Config().Mode = %q, want %q", got.Mode, Text)
	}
}

// ---------------------------------------------------------------------------
// InvalidateSnapshot / LatestSnapshot
// ---------------------------------------------------------------------------

func TestInvalidateSnapshot_ClearsSnapshot(t *testing.T) {
	ctx := newTestContext()

	// Manually set a snapshot to simulate a previous build.
	ctx.snapshotLock.Lock()
	ctx.snapshot = &Snapshot{}
	ctx.snapshotLock.Unlock()

	ctx.InvalidateSnapshot()

	ctx.snapshotLock.Lock()
	snap := ctx.snapshot
	ctx.snapshotLock.Unlock()

	if snap != nil {
		t.Error("InvalidateSnapshot: snapshot should be nil after invalidation")
	}
}

func TestLatestSnapshot_NilWhenEmpty(t *testing.T) {
	ctx := newTestContext()
	_, err := ctx.LatestSnapshot()
	if err == nil {
		t.Error("LatestSnapshot: expected error when no snapshot exists, got nil")
	}
}

func TestActivePageRequiresExistingPage(t *testing.T) {
	ctx := newTestContext()

	page, err := ctx.ActivePage()
	if err == nil {
		t.Fatal("ActivePage: expected error when no page exists")
	}
	if page != nil {
		t.Fatalf("ActivePage returned page = %v, want nil", page)
	}
	for _, want := range []string{"no active page", "rod_navigate", "domainHeaders"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ActivePage error = %q, want substring %q", err.Error(), want)
		}
	}
	if ctx.browser != nil || ctx.page != nil {
		t.Fatal("ActivePage should not initialize browser or page")
	}
}

func TestEnsurePageErrorsWhenInitialLeavesPageNil(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := NewContext(parent, Config{Mode: Text, Headless: true})
	defer func() {
		ctx.browserLock.Lock()
		if ctx.keepaliveCancel != nil {
			ctx.keepaliveCancel()
			ctx.keepaliveCancel = nil
		}
		ctx.releaseInstanceLockLocked()
		ctx.browser = nil
		ctx.browserLock.Unlock()
	}()

	withLaunchBrowserFunc(t, func(context.Context, Config) (*rod.Browser, string, error) {
		return rod.New(), "", nil
	})
	withCreatePageFunc(t, func(*Context, ...string) (*rod.Page, error) {
		return nil, nil
	})

	page, err := ctx.EnsurePage()
	if err == nil {
		t.Fatal("EnsurePage: expected error when initial succeeds without creating a page")
	}
	if page != nil {
		t.Fatalf("EnsurePage returned page = %v, want nil", page)
	}
	if !strings.Contains(err.Error(), "failed to create page after browser launch") {
		t.Fatalf("EnsurePage error = %q, want page creation failure message", err.Error())
	}
}

func TestLatestSnapshot_ReturnsExisting(t *testing.T) {
	ctx := newTestContext()

	expected := &Snapshot{textSnapshot: "test-snapshot"}
	ctx.snapshotLock.Lock()
	ctx.snapshot = expected
	ctx.snapshotLock.Unlock()

	got, err := ctx.LatestSnapshot()
	if err != nil {
		t.Fatalf("LatestSnapshot: unexpected error: %v", err)
	}
	if got != expected {
		t.Error("LatestSnapshot: returned wrong snapshot")
	}
}

func TestInvalidateSnapshot_ThenLatestSnapshot_Errors(t *testing.T) {
	ctx := newTestContext()

	// Set and then invalidate.
	ctx.snapshotLock.Lock()
	ctx.snapshot = &Snapshot{}
	ctx.snapshotLock.Unlock()

	ctx.InvalidateSnapshot()

	_, err := ctx.LatestSnapshot()
	if err == nil {
		t.Error("LatestSnapshot after Invalidate: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// ControlledPage / ControlledBrowser — no browser, expect errors
// ---------------------------------------------------------------------------

func TestControlledPage_NoBrowser_ReturnsError(t *testing.T) {
	// Manually create a Context with a browser set but nil page.
	ctx := newTestContext()

	// Simulate a browser being present but no page.
	// We can't launch a real browser in a unit test, so we only verify
	// the nil-page path error message when browser is non-nil.
	// When browser is nil, initLocked launches a browser — skip that path.

	// Test with browser=nil: initLocked will try to launch, which fails in CI.
	// Instead, test ControlledBrowser directly returns an error when browser is nil.
	_, err := ctx.ControlledBrowser()
	if err == nil {
		// If err is nil, a real browser was launched — skip in CI without Chrome.
		t.Skip("Chrome is available; skipping no-browser error path test")
	}
}

// ---------------------------------------------------------------------------
// EnsureSnapshot — without page, should error
// ---------------------------------------------------------------------------

func TestEnsureSnapshot_NoBrowser_Errors(t *testing.T) {
	ctx := newTestContext()
	// No page, so BuildSnapshot will fail.
	_, err := ctx.EnsureSnapshot()
	if err == nil {
		t.Skip("Chrome is available; skipping no-page error path test")
	}
}

func TestEnsureSnapshot_ReturnsExistingSnapshot(t *testing.T) {
	ctx := newTestContext()

	// Pre-populate the snapshot so EnsureSnapshot returns it without building.
	expected := &Snapshot{textSnapshot: "cached-snapshot"}
	ctx.snapshotLock.Lock()
	ctx.snapshot = expected
	ctx.snapshotLock.Unlock()

	got, err := ctx.EnsureSnapshot()
	if err != nil {
		t.Fatalf("EnsureSnapshot: unexpected error: %v", err)
	}
	if got != expected {
		t.Error("EnsureSnapshot: did not return the cached snapshot")
	}
}

// ---------------------------------------------------------------------------
// Reconfigure
// ---------------------------------------------------------------------------

func TestReconfigure_UpdatesConfig(t *testing.T) {
	ctx := newTestContext()

	headless := false
	endpoint := "http://localhost:9222"

	stealth := true
	err := ctx.Reconfigure(&headless, &endpoint, &stealth)
	if err != nil {
		t.Fatalf("Reconfigure: unexpected error: %v", err)
	}

	if ctx.config.Headless != false {
		t.Error("Reconfigure: Headless should be false")
	}
	if ctx.config.CDPEndpoint != endpoint {
		t.Errorf("Reconfigure: CDPEndpoint = %q, want %q", ctx.config.CDPEndpoint, endpoint)
	}
	if ctx.config.Stealth != true {
		t.Error("Reconfigure: Stealth should be true")
	}
}

func TestReconfigure_NilFields_NoChange(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Headless = true
	ctx.config.CDPEndpoint = "http://existing"

	err := ctx.Reconfigure(nil, nil, nil)
	if err != nil {
		t.Fatalf("Reconfigure(nil, nil, nil): unexpected error: %v", err)
	}

	if ctx.config.Headless != true {
		t.Error("Reconfigure(nil, nil, nil): Headless should remain true")
	}
	if ctx.config.CDPEndpoint != "http://existing" {
		t.Errorf("Reconfigure(nil, nil, nil): CDPEndpoint changed unexpectedly to %q", ctx.config.CDPEndpoint)
	}
}

// ---------------------------------------------------------------------------
// Close — with no browser, should not error
// ---------------------------------------------------------------------------

func TestClose_NoBrowser_NoError(t *testing.T) {
	ctx := newTestContext()
	err := ctx.Close()
	if err != nil {
		t.Errorf("Close with no browser: unexpected error: %v", err)
	}
}

func TestKeepaliveInterval(t *testing.T) {
	// Keepalive should be frequent enough to prevent the ~15min idle timeout.
	if keepaliveInterval >= 15*time.Minute {
		t.Errorf("keepaliveInterval = %v, should be less than 15 minutes to prevent idle timeout", keepaliveInterval)
	}
	// But not so frequent that it wastes resources.
	if keepaliveInterval < 1*time.Minute {
		t.Errorf("keepaliveInterval = %v, should be at least 1 minute", keepaliveInterval)
	}
}

func TestExecuteRecoversHandlerPanic(t *testing.T) {
	ctx := newTestContext()
	handler := ctx.Execute(func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		panic("key not defined")
	}, ToolHandlerCallOpts{})

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute returned nil result")
	}
	if !result.IsError {
		t.Fatal("Execute panic recovery should return an MCP error result")
	}
	if len(result.Content) == 0 {
		t.Fatal("Execute panic recovery returned no content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Execute panic recovery content type = %T, want mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, "key not defined") {
		t.Fatalf("Execute panic recovery text = %q, want panic value", text.Text)
	}
}

func TestExecuteHandlerErrorPreservesBrowserState(t *testing.T) {
	ctx := newTestContext()
	page := &rod.Page{}
	browser := &rod.Browser{}
	ctx.page = page
	ctx.browser = browser

	handler := ctx.Execute(func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, errors.New("transient cdp EOF")
	}, ToolHandlerCallOpts{})

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("Execute should convert handler errors to MCP error results")
	}
	if ctx.page != page {
		t.Fatal("Execute handler error cleared the active page")
	}
	if ctx.browser != browser {
		t.Fatal("Execute handler error cleared the active browser")
	}
}

func TestStartKeepalive_NilBrowser(t *testing.T) {
	ctx := newTestContext()
	// Should not panic with nil browser.
	ctx.startKeepalive()
	if ctx.keepaliveCancel != nil {
		t.Error("startKeepalive with nil browser should not set keepaliveCancel")
	}
}

func TestIsClosedBrowserSessionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "closed network connection",
			err:  errors.New("write tcp 127.0.0.1:62624->127.0.0.1:62622: use of closed network connection"),
			want: true,
		},
		{name: "net closed", err: net.ErrClosed, want: true},
		{name: "closed pipe", err: io.ErrClosedPipe, want: true},
		{name: "browser closed", err: errors.New("browser has been closed"), want: true},
		{name: "connection reset", err: errors.New("read tcp: connection reset by peer"), want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: false},
		{name: "connection refused", err: errors.New("dial tcp 127.0.0.1:9222: connect: connection refused"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClosedBrowserSessionError(tt.err); got != tt.want {
				t.Errorf("isClosedBrowserSessionError(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestInitial_LocalLaunchFailureBackoffSuppressesHotRetry(t *testing.T) {
	ctx := newTestContext()
	launchErr := errors.New("chrome exited during registration")
	var attempts int32
	withLaunchBrowserFunc(t, func(context.Context, Config) (*rod.Browser, string, error) {
		atomic.AddInt32(&attempts, 1)
		return nil, "", launchErr
	})

	err := ctx.initial()
	if err == nil {
		t.Fatal("initial: expected launch error")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("launch attempts after first failure = %d, want 1", got)
	}

	err = ctx.initial()
	if err == nil {
		t.Fatal("initial during backoff: expected error")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("launch attempts during backoff = %d, want 1", got)
	}
	if !strings.Contains(err.Error(), "browser launch suppressed") {
		t.Fatalf("backoff error = %q, want suppressed launch message", err.Error())
	}
	if !errors.Is(err, launchErr) {
		t.Fatalf("backoff error should wrap previous launch error, got %v", err)
	}
}

func TestInitial_LocalLaunchBackoffIsSerializedAcrossConcurrentCalls(t *testing.T) {
	ctx := newTestContext()
	launchErr := errors.New("chrome launch failed")
	var attempts int32
	withLaunchBrowserFunc(t, func(context.Context, Config) (*rod.Browser, string, error) {
		atomic.AddInt32(&attempts, 1)
		time.Sleep(20 * time.Millisecond)
		return nil, "", launchErr
	})

	const callers = 8
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- ctx.initial()
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err == nil {
			t.Fatal("initial: expected all concurrent callers to receive errors")
		}
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("concurrent launch attempts = %d, want 1", got)
	}
}

func TestInitial_CDPEndpointFailuresDoNotUseManagedLaunchBackoff(t *testing.T) {
	ctx := NewContext(context.Background(), Config{
		Mode:        Text,
		Headless:    true,
		CDPEndpoint: "http://127.0.0.1:9222",
	})
	launchErr := errors.New("connect refused")
	var attempts int32
	withLaunchBrowserFunc(t, func(context.Context, Config) (*rod.Browser, string, error) {
		atomic.AddInt32(&attempts, 1)
		return nil, "", launchErr
	})

	for i := 0; i < 2; i++ {
		err := ctx.initial()
		if err == nil {
			t.Fatal("initial: expected CDP connection error")
		}
		if strings.Contains(err.Error(), "browser launch suppressed") {
			t.Fatalf("CDP error unexpectedly used managed-launch backoff: %v", err)
		}
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("CDP launch attempts = %d, want 2", got)
	}
}

// TestInitial_LaunchUsesLongLivedContext is the regression test for the rod-mcp#308
// context bug: the browser must be created under the LONG-LIVED context, never a
// launch-timeout context that gets cancelled on return. rod ties a browser's CDP
// connection/event loop to its creation context, so a cancel-on-return launch
// timeout broke every later op with "context canceled". The launch timeout is
// now applied to the launcher (Chrome startup) inside launchBrowser, NOT to the
// launchBrowserFunc context.
func TestInitial_LaunchUsesLongLivedContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := NewContext(parent, Config{
		Mode:            Text,
		Headless:        true,
		LaunchTimeoutMs: 10,
	})

	var gotCtx context.Context
	withLaunchBrowserFunc(t, func(c context.Context, cfg Config) (*rod.Browser, string, error) {
		gotCtx = c
		// Return immediately (do NOT block on c.Done()) — the browser lifetime
		// context must not carry the launch timeout, so c.Done() must never fire.
		return nil, "", errors.New("stub: no real browser")
	})

	_ = ctx.initial() // errors from the stub; we only care about the context passed in
	if gotCtx == nil {
		t.Fatal("launchBrowserFunc was not called")
	}
	if dl, ok := gotCtx.Deadline(); ok {
		t.Fatalf("launchBrowserFunc context has deadline %v; want the long-lived lifetime context (launch timeout belongs to the launcher, not the browser)", dl)
	}
	if gotCtx.Err() != nil {
		t.Fatalf("launchBrowserFunc context already cancelled: %v", gotCtx.Err())
	}
}

func TestRecoverBrowserAfterErrorDropsState(t *testing.T) {
	ctx := newTestContext()
	ctx.browser = &rod.Browser{}
	ctx.page = &rod.Page{}
	ctx.pageCancel = func() {}
	ctx.keepaliveCancel = func() {}

	if !ctx.RecoverBrowserAfterError(context.DeadlineExceeded) {
		t.Fatal("RecoverBrowserAfterError returned false for deadline error")
	}
	if ctx.browser != nil {
		t.Fatal("RecoverBrowserAfterError did not clear browser")
	}
	if ctx.page != nil {
		t.Fatal("RecoverBrowserAfterError did not clear page")
	}
	if ctx.pageCancel != nil {
		t.Fatal("RecoverBrowserAfterError did not clear page cancel")
	}
	if ctx.keepaliveCancel != nil {
		t.Fatal("RecoverBrowserAfterError did not clear keepalive cancel")
	}
}

func TestRecoverBrowserAfterErrorIgnoresNonRecoverableError(t *testing.T) {
	ctx := newTestContext()
	page := &rod.Page{}
	browser := &rod.Browser{}
	ctx.page = page
	ctx.browser = browser

	if ctx.RecoverBrowserAfterError(errors.New("validation failed")) {
		t.Fatal("RecoverBrowserAfterError returned true for non-recoverable error")
	}
	if ctx.page != page {
		t.Fatal("RecoverBrowserAfterError cleared page for non-recoverable error")
	}
	if ctx.browser != browser {
		t.Fatal("RecoverBrowserAfterError cleared browser for non-recoverable error")
	}
}

func withLaunchBrowserFunc(t *testing.T, fn func(context.Context, Config) (*rod.Browser, string, error)) {
	t.Helper()
	prev := launchBrowserFunc
	launchBrowserFunc = fn
	t.Cleanup(func() {
		launchBrowserFunc = prev
	})
}

func withCreatePageFunc(t *testing.T, fn func(*Context, ...string) (*rod.Page, error)) {
	t.Helper()
	prev := createPageFunc
	createPageFunc = fn
	t.Cleanup(func() {
		createPageFunc = prev
	})
}

func TestClose_WithTempDir_Cleanup(t *testing.T) {
	ctx := newTestContext()
	// Create a temporary directory and assign it to the context.
	// Close should attempt to remove it.
	tmpDir := t.TempDir() // will be cleaned by testing framework too
	ctx.config.BrowserTempDir = tmpDir

	err := ctx.Close()
	if err != nil {
		t.Errorf("Close with temp dir: unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Stealth helpers
// ---------------------------------------------------------------------------

func TestChromeMajorVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"124.0.6367.91", "124"},
		{"130.0.6723.58", "130"},
		{"99", "99"},
		{"", ""},
	}
	for _, tc := range tests {
		got := chromeMajorVersion(tc.input)
		if got != tc.want {
			t.Errorf("chromeMajorVersion(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestApplyStealthHeaders_FillsMissingHeaders(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Stealth = true
	// No browser running, so chromeVersion returns "".
	// applyStealthHeaders should not crash; headers should remain empty for version-dependent fields.
	headers := ctx.applyStealthHeaders(nil)
	if headers == nil {
		t.Fatal("applyStealthHeaders returned nil")
	}
	// Without a running browser, User-Agent and Sec-CH-UA should not be set.
	if _, ok := headers["User-Agent"]; ok {
		t.Error("expected no User-Agent header without a running browser")
	}
	if _, ok := headers["Sec-CH-UA"]; ok {
		t.Error("expected no Sec-CH-UA header without a running browser")
	}
}

func TestApplyStealthHeaders_PreservesExistingHeaders(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Stealth = true

	existing := map[string]string{
		"User-Agent": "CustomAgent/1.0",
		"Sec-CH-UA":  `"Custom";v="1"`,
	}
	headers := ctx.applyStealthHeaders(existing)
	if headers["User-Agent"] != "CustomAgent/1.0" {
		t.Errorf("User-Agent was overwritten: %q", headers["User-Agent"])
	}
	if headers["Sec-CH-UA"] != `"Custom";v="1"` {
		t.Errorf("Sec-CH-UA was overwritten: %q", headers["Sec-CH-UA"])
	}
}

func TestApplyStealthHeaders_CaseInsensitiveLookup(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Stealth = true

	// Lower-case header keys should still prevent stealth from adding duplicates.
	existing := map[string]string{
		"user-agent": "CustomAgent/1.0",
		"sec-ch-ua":  `"Custom";v="1"`,
	}
	headers := ctx.applyStealthHeaders(existing)
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d: %v", len(headers), headers)
	}
	if headers["user-agent"] != "CustomAgent/1.0" {
		t.Errorf("user-agent was overwritten: %q", headers["user-agent"])
	}
	if headers["sec-ch-ua"] != `"Custom";v="1"` {
		t.Errorf("sec-ch-ua was overwritten: %q", headers["sec-ch-ua"])
	}
}

func TestReconfigure_Stealth(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Stealth = false

	stealth := true
	err := ctx.Reconfigure(nil, nil, &stealth)
	if err != nil {
		t.Fatalf("Reconfigure stealth: unexpected error: %v", err)
	}
	if !ctx.config.Stealth {
		t.Error("Reconfigure: Stealth should be true")
	}

	stealth = false
	err = ctx.Reconfigure(nil, nil, &stealth)
	if err != nil {
		t.Fatalf("Reconfigure stealth=false: unexpected error: %v", err)
	}
	if ctx.config.Stealth {
		t.Error("Reconfigure: Stealth should be false")
	}
}
