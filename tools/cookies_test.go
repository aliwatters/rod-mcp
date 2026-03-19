package tools

import "testing"

func TestCookiesToolDefinition(t *testing.T) {
	if Cookies.Name != CookiesToolKey {
		t.Errorf("Cookies tool name = %q, want %q", Cookies.Name, CookiesToolKey)
	}

	props := Cookies.InputSchema.Properties
	if props == nil {
		t.Fatal("Cookies tool has no properties")
	}

	for _, param := range []string{"action", "url", "name", "value", "domain", "path", "secure", "httpOnly", "sameSite"} {
		if _, ok := props[param]; !ok {
			t.Errorf("Cookies tool missing %q property", param)
		}
	}

	// "action" should be required
	found := false
	for _, r := range Cookies.InputSchema.Required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Cookies tool 'action' parameter should be required")
	}
}

func TestCookiesToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == CookiesToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Cookies tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[CookiesToolKey]; !ok {
		t.Error("CookiesHandler not found in CommonToolHandlers")
	}
}
