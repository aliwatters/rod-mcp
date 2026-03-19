package tools

import "testing"

func TestPermissionsToolDefinition(t *testing.T) {
	if Permissions.Name != PermissionsToolKey {
		t.Errorf("Permissions tool name = %q, want %q", Permissions.Name, PermissionsToolKey)
	}

	props := Permissions.InputSchema.Properties
	if props == nil {
		t.Fatal("Permissions tool has no properties")
	}

	for _, param := range []string{"action", "permissions", "origin"} {
		if _, ok := props[param]; !ok {
			t.Errorf("Permissions tool missing %q property", param)
		}
	}

	// "action" should be required
	found := false
	for _, r := range Permissions.InputSchema.Required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Permissions tool 'action' parameter should be required")
	}
}

func TestPermissionsToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == PermissionsToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Permissions tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[PermissionsToolKey]; !ok {
		t.Error("PermissionsHandler not found in CommonToolHandlers")
	}
}
