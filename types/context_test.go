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
