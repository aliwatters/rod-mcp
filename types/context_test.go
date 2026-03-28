package types

import (
	"context"
	"testing"
	"time"
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
	if ctx.consoleMessages == nil {
		t.Error("consoleMessages ring buffer should be initialized")
	}
	if ctx.networkRequests == nil {
		t.Error("networkRequests ring buffer should be initialized")
	}
	if ctx.wsFrames == nil {
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
	ctx.stateLock.Lock()
	ctx.snapshot = &Snapshot{}
	ctx.stateLock.Unlock()

	ctx.InvalidateSnapshot()

	ctx.stateLock.Lock()
	snap := ctx.snapshot
	ctx.stateLock.Unlock()

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

func TestLatestSnapshot_ReturnsExisting(t *testing.T) {
	ctx := newTestContext()

	expected := &Snapshot{textSnapshot: "test-snapshot"}
	ctx.stateLock.Lock()
	ctx.snapshot = expected
	ctx.stateLock.Unlock()

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
	ctx.stateLock.Lock()
	ctx.snapshot = &Snapshot{}
	ctx.stateLock.Unlock()

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
	ctx.stateLock.Lock()
	ctx.snapshot = expected
	ctx.stateLock.Unlock()

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

	err := ctx.Reconfigure(&headless, &endpoint)
	if err != nil {
		t.Fatalf("Reconfigure: unexpected error: %v", err)
	}

	if ctx.config.Headless != false {
		t.Error("Reconfigure: Headless should be false")
	}
	if ctx.config.CDPEndpoint != endpoint {
		t.Errorf("Reconfigure: CDPEndpoint = %q, want %q", ctx.config.CDPEndpoint, endpoint)
	}
}

func TestReconfigure_NilFields_NoChange(t *testing.T) {
	ctx := newTestContext()
	ctx.config.Headless = true
	ctx.config.CDPEndpoint = "http://existing"

	err := ctx.Reconfigure(nil, nil)
	if err != nil {
		t.Fatalf("Reconfigure(nil, nil): unexpected error: %v", err)
	}

	if ctx.config.Headless != true {
		t.Error("Reconfigure(nil, nil): Headless should remain true")
	}
	if ctx.config.CDPEndpoint != "http://existing" {
		t.Errorf("Reconfigure(nil, nil): CDPEndpoint changed unexpectedly to %q", ctx.config.CDPEndpoint)
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

func TestStartKeepalive_NilBrowser(t *testing.T) {
	ctx := newTestContext()
	// Should not panic with nil browser.
	ctx.startKeepalive()
	if ctx.keepaliveCancel != nil {
		t.Error("startKeepalive with nil browser should not set keepaliveCancel")
	}
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
