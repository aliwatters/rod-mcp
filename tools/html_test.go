package tools

import "testing"

func TestHTMLToolDefinition(t *testing.T) {
	if HTML.Name != HTMLToolKey {
		t.Errorf("HTML tool name = %q, want %q", HTML.Name, HTMLToolKey)
	}

	props := HTML.InputSchema.Properties
	if props == nil {
		t.Fatal("HTML tool has no properties")
	}

	for _, param := range []string{"selector", "outer", "max_chars"} {
		if _, ok := props[param]; !ok {
			t.Errorf("HTML tool missing %q property", param)
		}
	}

	// All parameters should be optional
	if len(HTML.InputSchema.Required) > 0 {
		t.Errorf("HTML tool should have no required parameters, got %v", HTML.InputSchema.Required)
	}
}

func TestHTMLToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == HTMLToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("HTML tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[HTMLToolKey]; !ok {
		t.Error("HTMLHandler not found in CommonToolHandlers")
	}
}
