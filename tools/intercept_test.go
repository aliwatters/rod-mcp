package tools

import "testing"

func TestInterceptToolDefinition(t *testing.T) {
	if Intercept.Name != InterceptToolKey {
		t.Errorf("Intercept tool name = %q, want %q", Intercept.Name, InterceptToolKey)
	}

	props := Intercept.InputSchema.Properties
	if props == nil {
		t.Fatal("Intercept tool has no properties")
	}

	for _, param := range []string{"action", "urlPattern", "status", "headers", "body", "errorReason"} {
		if _, ok := props[param]; !ok {
			t.Errorf("Intercept tool missing %q property", param)
		}
	}

	// "action" should be required
	found := false
	for _, r := range Intercept.InputSchema.Required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Intercept tool 'action' parameter should be required")
	}
}

func TestInterceptToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == InterceptToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Intercept tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[InterceptToolKey]; !ok {
		t.Error("InterceptHandler not found in CommonToolHandlers")
	}
}
