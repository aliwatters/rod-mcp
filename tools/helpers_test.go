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
