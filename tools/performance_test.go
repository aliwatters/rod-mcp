package tools

import "testing"

func TestPerformanceToolDefinition(t *testing.T) {
	if Performance.Name != PerformanceToolKey {
		t.Errorf("Performance tool name = %q, want %q", Performance.Name, PerformanceToolKey)
	}

	props := Performance.InputSchema.Properties
	if props == nil {
		t.Fatal("Performance tool has no properties")
	}

	if _, ok := props["action"]; !ok {
		t.Error("Performance tool missing 'action' property")
	}

	// "action" should be required
	found := false
	for _, r := range Performance.InputSchema.Required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Performance tool 'action' parameter should be required")
	}
}

func TestPerformanceToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == PerformanceToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Performance tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[PerformanceToolKey]; !ok {
		t.Error("PerformanceHandler not found in CommonToolHandlers")
	}
}
