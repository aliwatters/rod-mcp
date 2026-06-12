package tools

import (
	"strings"
	"testing"

	"github.com/aliwatters/rod-mcp/types/js"
)

func TestFillFormToolDefinition(t *testing.T) {
	if FillForm.Name != FillFormToolKey {
		t.Errorf("FillForm tool name = %q, want %q", FillForm.Name, FillFormToolKey)
	}

	props := FillForm.InputSchema.Properties
	if props == nil {
		t.Fatal("FillForm tool has no properties")
	}

	// Verify fields parameter exists
	if _, ok := props["fields"]; !ok {
		t.Error("FillForm tool missing 'fields' property")
	}

	// Verify submit parameter exists
	if _, ok := props["submit"]; !ok {
		t.Error("FillForm tool missing 'submit' property")
	}

	// Verify fields is required
	found := false
	for _, r := range FillForm.InputSchema.Required {
		if r == "fields" {
			found = true
			break
		}
	}
	if !found {
		t.Error("FillForm tool 'fields' parameter should be required")
	}

	// Verify submit is NOT required (optional)
	for _, r := range FillForm.InputSchema.Required {
		if r == "submit" {
			t.Error("FillForm tool 'submit' parameter should not be required")
		}
	}
}

func TestFillFormToolRegistered(t *testing.T) {
	// Verify FillForm is in CommonTools
	found := false
	for _, tool := range CommonTools {
		if tool.Name == FillFormToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("FillForm tool not found in CommonTools")
	}

	// Verify handler is registered
	if _, ok := CommonToolHandlers[FillFormToolKey]; !ok {
		t.Error("FillForm handler not found in CommonToolHandlers")
	}
}

func TestSmartFillResultType(t *testing.T) {
	// Verify the smartFillResult struct fields exist and are typed correctly
	r := smartFillResult{
		Method:  "standard",
		Value:   "test",
		React:   false,
		Success: true,
	}

	if r.Method != "standard" {
		t.Errorf("Method = %q, want %q", r.Method, "standard")
	}
	if r.Value != "test" {
		t.Errorf("Value = %q, want %q", r.Value, "test")
	}
	if r.React != false {
		t.Error("React should be false")
	}
	if r.Success != true {
		t.Error("Success should be true")
	}
}

func TestSmartFillSupportsContentEditableInputEvents(t *testing.T) {
	required := []string{
		"isContentEditable",
		"beforeinput",
		"InputEvent",
		"insertText",
		"contenteditable_input",
	}

	for _, s := range required {
		if !strings.Contains(js.SmartFillJS, s) {
			t.Fatalf("SmartFillJS missing %q contenteditable support", s)
		}
	}
}
