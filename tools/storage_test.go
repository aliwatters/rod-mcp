package tools

import "testing"

func TestStorageToolDefinition(t *testing.T) {
	if Storage.Name != StorageToolKey {
		t.Errorf("Storage tool name = %q, want %q", Storage.Name, StorageToolKey)
	}

	props := Storage.InputSchema.Properties
	if props == nil {
		t.Fatal("Storage tool has no properties")
	}

	for _, param := range []string{"type", "action", "key", "value"} {
		if _, ok := props[param]; !ok {
			t.Errorf("Storage tool missing %q property", param)
		}
	}

	// "type" and "action" should be required
	requiredFound := map[string]bool{"type": false, "action": false}
	for _, r := range Storage.InputSchema.Required {
		if _, ok := requiredFound[r]; ok {
			requiredFound[r] = true
		}
	}
	for param, found := range requiredFound {
		if !found {
			t.Errorf("Storage tool %q parameter should be required", param)
		}
	}
}

func TestStorageToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == StorageToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Storage tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[StorageToolKey]; !ok {
		t.Error("StorageHandler not found in CommonToolHandlers")
	}
}
