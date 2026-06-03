package tools

import (
	"strings"
	"testing"
)

func TestResolveSnapshotElementMissingElement(t *testing.T) {
	// Missing "element" arg should fail before needing a Context.
	args := map[string]interface{}{
		"selector": "#foo",
	}
	_, _, _, err := resolveSnapshotElement(nil, args, "test")
	if err == nil {
		t.Error("resolveSnapshotElement should fail when 'element' is missing")
	}
}

func TestResolveSnapshotElementErrorMessageFormat(t *testing.T) {
	// Verify the error mentions the tool name when element is missing.
	args := map[string]interface{}{}
	_, _, _, err := resolveSnapshotElement(nil, args, "click element")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "click element") {
		t.Errorf("error should mention tool name; got: %s", err.Error())
	}
}

func TestApplyMaxChars(t *testing.T) {
	const full = "abcdefghij" // 10 chars

	t.Run("zero means unlimited", func(t *testing.T) {
		got := applyMaxChars(full, 0)
		if got != full {
			t.Errorf("applyMaxChars(full, 0): want full string, got %q", got)
		}
	})

	t.Run("negative means unlimited", func(t *testing.T) {
		got := applyMaxChars(full, -1)
		if got != full {
			t.Errorf("applyMaxChars(full, -1): want full string, got %q", got)
		}
	})

	t.Run("limit larger than content returns full string", func(t *testing.T) {
		got := applyMaxChars(full, 100)
		if got != full {
			t.Errorf("applyMaxChars(full, 100): want full string, got %q", got)
		}
	})

	t.Run("limit equal to content length returns full string", func(t *testing.T) {
		got := applyMaxChars(full, len(full))
		if got != full {
			t.Errorf("applyMaxChars(full, len): want full string, got %q", got)
		}
	})

	t.Run("truncation appends notice with counts", func(t *testing.T) {
		limit := 5
		got := applyMaxChars(full, limit)

		if !strings.HasPrefix(got, full[:limit]) {
			t.Errorf("applyMaxChars truncated prefix = %q, want %q", got[:limit], full[:limit])
		}
		if !strings.Contains(got, "truncated") {
			t.Error("applyMaxChars: truncation notice missing 'truncated'")
		}
		// Notice must include the limit and original length.
		if !strings.Contains(got, "5") {
			t.Error("applyMaxChars: truncation notice must include max_chars value (5)")
		}
		if !strings.Contains(got, "10") {
			t.Error("applyMaxChars: truncation notice must include original length (10)")
		}
		// Notice must mention rod_evaluate.
		if !strings.Contains(got, "rod_evaluate") {
			t.Error("applyMaxChars: truncation notice must mention rod_evaluate")
		}
	})
}
