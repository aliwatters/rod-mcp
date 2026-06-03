package tools

import "testing"

func TestClickToolDefinition(t *testing.T) {
	props := Click.InputSchema.Properties
	if props == nil {
		t.Fatal("Click tool has no properties")
	}

	// Verify selector parameter exists
	if _, ok := props["selector"]; !ok {
		t.Error("Click tool missing 'selector' property")
	}

	// Verify ref parameter exists
	if _, ok := props["ref"]; !ok {
		t.Error("Click tool missing 'ref' property")
	}

	// Verify name parameter exists
	if _, ok := props["name"]; !ok {
		t.Error("Click tool missing 'name' property")
	}

	// Verify element is required
	found := false
	for _, r := range Click.InputSchema.Required {
		if r == "element" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Click tool 'element' parameter should be required")
	}

	// Verify description mentions CSS selector
	if Click.Description == "" {
		t.Error("Click tool should have a description")
	}
}

func TestHoverToolDefinition(t *testing.T) {
	props := Hover.InputSchema.Properties
	if props == nil {
		t.Fatal("Hover tool has no properties")
	}

	if _, ok := props["selector"]; !ok {
		t.Error("Hover tool missing 'selector' property")
	}
}

func TestFillToolDefinition(t *testing.T) {
	props := Fill.InputSchema.Properties
	if props == nil {
		t.Fatal("Fill tool has no properties")
	}

	// Verify selector parameter exists
	if _, ok := props["selector"]; !ok {
		t.Error("Fill tool missing 'selector' property")
	}

	// Verify ref parameter exists
	if _, ok := props["ref"]; !ok {
		t.Error("Fill tool missing 'ref' property")
	}

	// Verify value is required
	found := false
	for _, r := range Fill.InputSchema.Required {
		if r == "value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Fill tool 'value' parameter should be required")
	}
}

func TestSelectorToolDefinition(t *testing.T) {
	props := Selector.InputSchema.Properties
	if props == nil {
		t.Fatal("Selector tool has no properties")
	}

	if _, ok := props["selector"]; !ok {
		t.Error("Selector tool missing 'selector' property")
	}
}

func TestSnapshotToolMaxCharsParam(t *testing.T) {
	props := Snapshot.InputSchema.Properties
	if props == nil {
		t.Fatal("Snapshot tool has no properties")
	}

	if _, ok := props["max_chars"]; !ok {
		t.Error("Snapshot tool missing 'max_chars' property")
	}

	// max_chars must be optional — not in the required list.
	for _, r := range Snapshot.InputSchema.Required {
		if r == "max_chars" {
			t.Error("Snapshot tool 'max_chars' should not be required")
		}
	}
}

func TestSnapshotToolsRegistered(t *testing.T) {
	// Verify all snapshot tools are in the Snapshots slice
	expectedTools := map[string]bool{
		SnapshotToolKey: false,
		ClickToolKey:    false,
		HoverToolKey:    false,
		FillToolKey:     false,
		SelectorToolKey: false,
	}

	for _, tool := range Snapshots {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}

	for key, found := range expectedTools {
		if !found {
			t.Errorf("Tool %s not found in Snapshots slice", key)
		}
	}

	// Verify all snapshot tool handlers are registered
	for key := range expectedTools {
		if _, ok := SnapshotToolHandlers[key]; !ok {
			t.Errorf("Handler for %s not found in SnapshotToolHandlers", key)
		}
	}
}
