package types

import (
	"strings"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"

	"github.com/aliwatters/rod-mcp/types/js"
)

// TestBuildSnapshot_RedactsPasswordFields verifies that password input values
// are replaced with [REDACTED] in ARIA snapshots to prevent secret leakage.
func TestBuildSnapshot_RedactsPasswordFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	bin, found := launcher.LookPath()
	if !found {
		t.Skip("skipping: no browser binary found for rod launcher")
	}

	l := launcher.New().Bin(bin).Headless(true)
	u, err := l.Launch()
	if err != nil {
		t.Skipf("skipping: cannot launch browser: %v", err)
	}

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")

	// Inject the snapshot engine JS so it runs on subsequent navigations.
	// EvalOnNewDocument ensures the script is available on future documents.
	if _, err := page.EvalOnNewDocument(js.InjectedSnapShot); err != nil {
		t.Fatalf("EvalOnNewDocument: %v", err)
	}

	// Navigate to about:blank to trigger EvalOnNewDocument, then set content.
	page.MustNavigate("about:blank").MustWaitStable()

	page.MustSetDocumentContent(`
		<form>
			<label for="user">Username</label>
			<input id="user" type="text" value="alice" />
			<label for="pass">Password</label>
			<input id="pass" type="password" value="SuperSecret123!" />
		</form>
	`)
	page.MustWaitStable()

	snap, err := BuildSnapshot(page, false)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	text := snap.String()

	// The password value must be redacted.
	if strings.Contains(text, "SuperSecret123!") {
		t.Errorf("snapshot contains raw password value; expected [REDACTED]\n\nsnapshot:\n%s", text)
	}

	if !strings.Contains(text, "[REDACTED]") {
		t.Errorf("snapshot missing [REDACTED] marker for password field\n\nsnapshot:\n%s", text)
	}

	// The username (non-password) value should still appear.
	if !strings.Contains(text, "alice") {
		t.Errorf("snapshot missing non-password input value 'alice'\n\nsnapshot:\n%s", text)
	}
}
