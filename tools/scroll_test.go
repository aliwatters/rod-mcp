package tools

import "testing"

func TestScrollToolDefinition(t *testing.T) {
	if Scroll.Name != ScrollToolKey {
		t.Errorf("Scroll tool name = %q, want %q", Scroll.Name, ScrollToolKey)
	}

	props := Scroll.InputSchema.Properties
	if props == nil {
		t.Fatal("Scroll tool has no properties")
	}

	for _, param := range []string{"direction", "amount", "x", "y", "selector"} {
		if _, ok := props[param]; !ok {
			t.Errorf("Scroll tool missing %q property", param)
		}
	}

	// All parameters should be optional
	if len(Scroll.InputSchema.Required) > 0 {
		t.Errorf("Scroll tool should have no required parameters, got %v", Scroll.InputSchema.Required)
	}
}

func TestScrollToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == ScrollToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Scroll tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[ScrollToolKey]; !ok {
		t.Error("ScrollHandler not found in CommonToolHandlers")
	}
}
