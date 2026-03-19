package tools

import "testing"

func TestWebSocketToolDefinition(t *testing.T) {
	if WebSocket.Name != WebSocketToolKey {
		t.Errorf("WebSocket tool name = %q, want %q", WebSocket.Name, WebSocketToolKey)
	}

	props := WebSocket.InputSchema.Properties
	if props == nil {
		t.Fatal("WebSocket tool has no properties")
	}

	for _, param := range []string{"action", "urlFilter", "direction", "maxFrames"} {
		if _, ok := props[param]; !ok {
			t.Errorf("WebSocket tool missing %q property", param)
		}
	}

	// "action" should be required
	found := false
	for _, r := range WebSocket.InputSchema.Required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WebSocket tool 'action' parameter should be required")
	}
}

func TestWebSocketToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == WebSocketToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("WebSocket tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[WebSocketToolKey]; !ok {
		t.Error("WebSocketHandler not found in CommonToolHandlers")
	}
}
