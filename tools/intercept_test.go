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

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		url, pattern string
		want         bool
	}{
		{"https://api.example.com/data", "*api.example.com*", true},
		{"https://api.example.com/data", "*other.com*", false},
		{"https://example.com/path", "*", true},
		{"https://example.com/path", "https://example.com/path", true},
		{"https://example.com/path", "https://example.com/other", false},
		{"https://example.com/a", "https://example.com/?", true},
		{"https://example.com/ab", "https://example.com/?", false},
		{"", "*", true},
		{"", "", true},
		{"a", "", false},
	}

	for _, tc := range tests {
		got := matchWildcard(tc.url, tc.pattern)
		if got != tc.want {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", tc.url, tc.pattern, got, tc.want)
		}
	}
}
