package tools

import "testing"

func TestResponseBodyToolDefinition(t *testing.T) {
	if ResponseBody.Name != ResponseBodyToolKey {
		t.Errorf("ResponseBody tool name = %q, want %q", ResponseBody.Name, ResponseBodyToolKey)
	}

	props := ResponseBody.InputSchema.Properties
	if props == nil {
		t.Fatal("ResponseBody tool has no properties")
	}

	for _, param := range []string{"index", "maxLength"} {
		if _, ok := props[param]; !ok {
			t.Errorf("ResponseBody tool missing %q property", param)
		}
	}

	// "index" should be required
	found := false
	for _, r := range ResponseBody.InputSchema.Required {
		if r == "index" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ResponseBody tool 'index' parameter should be required")
	}
}

func TestResponseBodyToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == ResponseBodyToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("ResponseBody tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[ResponseBodyToolKey]; !ok {
		t.Error("ResponseBodyHandler not found in CommonToolHandlers")
	}
}
