package tools

import "testing"

func TestCoverageToolDefinition(t *testing.T) {
	if Coverage.Name != CoverageToolKey {
		t.Errorf("Coverage tool name = %q, want %q", Coverage.Name, CoverageToolKey)
	}

	props := Coverage.InputSchema.Properties
	if props == nil {
		t.Fatal("Coverage tool has no properties")
	}

	if _, ok := props["action"]; !ok {
		t.Error("Coverage tool missing 'action' property")
	}
	if _, ok := props["type"]; !ok {
		t.Error("Coverage tool missing 'type' property")
	}

	// "action" should be required
	found := false
	for _, r := range Coverage.InputSchema.Required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Coverage tool 'action' parameter should be required")
	}

	// "type" should be optional
	for _, r := range Coverage.InputSchema.Required {
		if r == "type" {
			t.Error("Coverage tool 'type' parameter should not be required")
		}
	}
}

func TestCoverageToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == CoverageToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Coverage tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[CoverageToolKey]; !ok {
		t.Error("CoverageHandler not found in CommonToolHandlers")
	}
}
