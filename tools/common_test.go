package tools

import (
	"testing"

	"github.com/go-rod/rod/lib/input"
)

func TestPdfToolDefinition(t *testing.T) {
	if Pdf.Name != PdfToolKey {
		t.Errorf("Pdf tool name = %q, want %q", Pdf.Name, PdfToolKey)
	}

	// Verify description mentions saving to output directory
	if Pdf.Description != "Generate a PDF of the current page and save to the output directory" {
		t.Errorf("Pdf tool description = %q, want file-based description", Pdf.Description)
	}

	// Verify "name" parameter exists and is required
	props := Pdf.InputSchema.Properties
	if props == nil {
		t.Fatal("Pdf tool has no properties")
	}
	if _, ok := props["name"]; !ok {
		t.Error("Pdf tool missing 'name' property")
	}

	// Verify old file_path/file_name params are removed
	if _, ok := props["file_path"]; ok {
		t.Error("Pdf tool should not have 'file_path' property")
	}
	if _, ok := props["file_name"]; ok {
		t.Error("Pdf tool should not have 'file_name' property")
	}

	// Verify "name" is required
	required := Pdf.InputSchema.Required
	found := false
	for _, r := range required {
		if r == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Pdf tool 'name' parameter should be required")
	}
}

func TestPdfToolRegistered(t *testing.T) {
	// Verify Pdf is in CommonTools
	found := false
	for _, tool := range CommonTools {
		if tool.Name == PdfToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Pdf tool not found in CommonTools")
	}

	// Verify PdfHandler is in CommonToolHandlers
	if _, ok := CommonToolHandlers[PdfToolKey]; !ok {
		t.Error("PdfHandler not found in CommonToolHandlers")
	}
}

func TestAllCommonToolsHaveHandlers(t *testing.T) {
	for _, tool := range CommonTools {
		if _, ok := CommonToolHandlers[tool.Name]; !ok {
			t.Errorf("Tool %q registered in CommonTools but missing from CommonToolHandlers", tool.Name)
		}
	}
}

func TestAllCommonHandlersHaveTools(t *testing.T) {
	toolNames := make(map[string]bool)
	for _, tool := range CommonTools {
		toolNames[tool.Name] = true
	}
	for name := range CommonToolHandlers {
		if !toolNames[name] {
			t.Errorf("Handler %q registered in CommonToolHandlers but missing from CommonTools", name)
		}
	}
}

func TestTextToolsIncludesPdf(t *testing.T) {
	found := false
	for _, tool := range TextTools {
		if tool.Name == PdfToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Pdf tool not found in TextTools (should be included via CommonTools)")
	}

	if _, ok := TextToolHandlers[PdfToolKey]; !ok {
		t.Error("PdfHandler not found in TextToolHandlers (should be included via CommonToolHandlers)")
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		keyStr  string
		want    input.Key
		wantErr bool
	}{
		// Named keys
		{"Tab", "Tab", input.Tab, false},
		{"Enter", "Enter", input.Enter, false},
		{"Return alias", "Return", input.Enter, false},
		{"Escape", "Escape", input.Escape, false},
		{"Backspace", "Backspace", input.Backspace, false},
		{"Delete", "Delete", input.Delete, false},
		{"Space", "Space", input.Space, false},

		// Arrow keys
		{"ArrowUp", "ArrowUp", input.ArrowUp, false},
		{"ArrowDown", "ArrowDown", input.ArrowDown, false},
		{"ArrowLeft", "ArrowLeft", input.ArrowLeft, false},
		{"ArrowRight", "ArrowRight", input.ArrowRight, false},

		// Function keys
		{"F1", "F1", input.F1, false},
		{"F12", "F12", input.F12, false},

		// Modifiers
		{"Shift", "Shift", input.ShiftLeft, false},
		{"Control", "Control", input.ControlLeft, false},
		{"Alt", "Alt", input.AltLeft, false},
		{"Meta", "Meta", input.MetaLeft, false},

		// Single characters
		{"lowercase a", "a", input.Key('a'), false},
		{"uppercase A", "A", input.Key('A'), false},
		{"digit 1", "1", input.Key('1'), false},
		{"special char @", "@", input.Key('@'), false},

		// Errors
		{"unknown key", "UnknownKey", 0, true},
		{"multi-char non-key", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKey(tt.keyStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKey(%q) error = %v, wantErr %v", tt.keyStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseKey(%q) = %v, want %v", tt.keyStr, got, tt.want)
			}
		})
	}
}

func TestScreenshotToolDefinition(t *testing.T) {
	props := Screenshot.InputSchema.Properties
	if props == nil {
		t.Fatal("Screenshot tool has no properties")
	}

	// Required params
	if _, ok := props["name"]; !ok {
		t.Error("Screenshot tool should have 'name' parameter")
	}

	// Optional params for element and full-page screenshots
	if _, ok := props["full_page"]; !ok {
		t.Error("Screenshot tool should have 'full_page' parameter")
	}
	if _, ok := props["selector"]; !ok {
		t.Error("Screenshot tool should have 'selector' parameter")
	}
	if _, ok := props["ref"]; !ok {
		t.Error("Screenshot tool should have 'ref' parameter")
	}
	if _, ok := props["save_to"]; !ok {
		t.Error("Screenshot tool should have 'save_to' parameter")
	}

	// Removed params should stay removed
	if _, ok := props["width"]; ok {
		t.Error("Screenshot tool should not have 'width' parameter")
	}
	if _, ok := props["height"]; ok {
		t.Error("Screenshot tool should not have 'height' parameter")
	}

	// Only "name" should be required
	for _, r := range Screenshot.InputSchema.Required {
		if r != "name" {
			t.Errorf("Screenshot tool has unexpected required param %q — selector/ref/full_page/save_to should be optional", r)
		}
	}
}

func TestTypeToolDefinition(t *testing.T) {
	if Type.Name != TypeToolKey {
		t.Errorf("Type tool name = %q, want %q", Type.Name, TypeToolKey)
	}

	props := Type.InputSchema.Properties
	if props == nil {
		t.Fatal("Type tool has no properties")
	}
	if _, ok := props["text"]; !ok {
		t.Error("Type tool missing 'text' property")
	}
	if _, ok := props["delay"]; !ok {
		t.Error("Type tool missing 'delay' property")
	}

	// Verify "text" is required
	found := false
	for _, r := range Type.InputSchema.Required {
		if r == "text" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Type tool 'text' parameter should be required")
	}
}

func TestTypeToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == TypeToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Type tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[TypeToolKey]; !ok {
		t.Error("TypeHandler not found in CommonToolHandlers")
	}
}

func TestTypeToolInTextTools(t *testing.T) {
	found := false
	for _, tool := range TextTools {
		if tool.Name == TypeToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Type tool not found in TextTools")
	}

	if _, ok := TextToolHandlers[TypeToolKey]; !ok {
		t.Error("TypeHandler not found in TextToolHandlers")
	}
}

func TestEvaluateToolDefinition(t *testing.T) {
	if Evaluate.Name != EvaluateToolKey {
		t.Errorf("Evaluate tool name = %q, want %q", Evaluate.Name, EvaluateToolKey)
	}

	props := Evaluate.InputSchema.Properties
	if props == nil {
		t.Fatal("Evaluate tool has no properties")
	}
	if _, ok := props["script"]; !ok {
		t.Error("Evaluate tool missing 'script' property")
	}

	found := false
	for _, r := range Evaluate.InputSchema.Required {
		if r == "script" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Evaluate tool 'script' parameter should be required")
	}
}

// TestRegistrationsDRY verifies that the Registrations type correctly derives
// both Tools() slices and Handlers() maps, so the two structures cannot drift.
func TestRegistrationsDRY(t *testing.T) {
	// Every tool in CommonRegistrations must appear in CommonTools with a handler.
	for _, reg := range CommonRegistrations {
		name := reg.Tool.Name

		foundTool := false
		for _, t2 := range CommonTools {
			if t2.Name == name {
				foundTool = true
				break
			}
		}
		if !foundTool {
			t.Errorf("CommonRegistrations entry %q missing from CommonTools", name)
		}

		if _, ok := CommonToolHandlers[name]; !ok {
			t.Errorf("CommonRegistrations entry %q missing from CommonToolHandlers", name)
		}
	}

	// Tools() and Handlers() must have the same count as CommonRegistrations.
	if len(CommonTools) != len(CommonRegistrations) {
		t.Errorf("CommonTools len = %d, want %d (same as CommonRegistrations)", len(CommonTools), len(CommonRegistrations))
	}
	if len(CommonToolHandlers) != len(CommonRegistrations) {
		t.Errorf("CommonToolHandlers len = %d, want %d (same as CommonRegistrations)", len(CommonToolHandlers), len(CommonRegistrations))
	}
}

func TestTextRegistrationsComplete(t *testing.T) {
	// TextRegistrations must contain all common + all snapshot tools.
	wantLen := len(CommonRegistrations) + len(SnapshotRegistrations)
	if len(TextRegistrations) != wantLen {
		t.Errorf("TextRegistrations len = %d, want %d", len(TextRegistrations), wantLen)
	}
	handlers := TextRegistrations.Handlers()
	for _, reg := range TextRegistrations {
		if _, ok := handlers[reg.Tool.Name]; !ok {
			t.Errorf("TextRegistrations entry %q has no handler in derived map", reg.Tool.Name)
		}
	}
}

func TestNavigationUsesToolKey(t *testing.T) {
	if Navigation.Name != NavigationToolKey {
		t.Errorf("Navigation tool name = %q, want %q (should use NavigationToolKey constant)", Navigation.Name, NavigationToolKey)
	}
}

func TestReloadNaming(t *testing.T) {
	if Reload.Name != ReloadToolKey {
		t.Errorf("Reload tool name = %q, want %q", Reload.Name, ReloadToolKey)
	}

	// Verify handler is registered with correct key
	if _, ok := CommonToolHandlers[ReloadToolKey]; !ok {
		t.Error("ReloadHandler not found in CommonToolHandlers")
	}
}
