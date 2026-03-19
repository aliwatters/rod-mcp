package tools

import "testing"

func TestDragToolDefinition(t *testing.T) {
	if Drag.Name != DragToolKey {
		t.Errorf("Drag tool name = %q, want %q", Drag.Name, DragToolKey)
	}

	props := Drag.InputSchema.Properties
	if props == nil {
		t.Fatal("Drag tool has no properties")
	}

	for _, param := range []string{"startX", "startY", "endX", "endY", "sourceSelector", "targetSelector", "steps"} {
		if _, ok := props[param]; !ok {
			t.Errorf("Drag tool missing %q property", param)
		}
	}
}

func TestDragToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == DragToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Drag tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[DragToolKey]; !ok {
		t.Error("DragHandler not found in CommonToolHandlers")
	}
}
