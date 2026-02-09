package tools

import (
	"context"
	"testing"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestConfigureToolDefinition(t *testing.T) {
	if Configure.Name != ConfigureToolKey {
		t.Errorf("Configure tool name = %q, want %q", Configure.Name, ConfigureToolKey)
	}

	props := Configure.InputSchema.Properties
	if props == nil {
		t.Fatal("Configure tool has no properties")
	}
	if _, ok := props["headless"]; !ok {
		t.Error("Configure tool missing 'headless' property")
	}
	if _, ok := props["cdp_endpoint"]; !ok {
		t.Error("Configure tool missing 'cdp_endpoint' property")
	}

	// headless and cdp_endpoint should be optional (not required)
	for _, r := range Configure.InputSchema.Required {
		if r == "headless" || r == "cdp_endpoint" {
			t.Errorf("Configure tool parameter %q should not be required", r)
		}
	}
}

func TestConfigureToolRegistered(t *testing.T) {
	found := false
	for _, tool := range CommonTools {
		if tool.Name == ConfigureToolKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("Configure tool not found in CommonTools")
	}

	if _, ok := CommonToolHandlers[ConfigureToolKey]; !ok {
		t.Error("ConfigureHandler not found in CommonToolHandlers")
	}
}

func TestConfigureHandlerNoArgs(t *testing.T) {
	cfg := types.DefaultConfig
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "No configuration changes requested" {
		t.Errorf("unexpected result text: %s", text)
	}
}

func TestConfigureHandlerHeadless(t *testing.T) {
	cfg := types.DefaultConfig
	cfg.Headless = false
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"headless": true,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "Browser configured: headless=true. Changes take effect on next browser action." {
		t.Errorf("unexpected result text: %s", text)
	}

	// Verify config was updated
	if !rodCtx.Config().Headless {
		t.Error("expected Headless to be true after reconfigure")
	}
}

func TestConfigureHandlerCDPEndpoint(t *testing.T) {
	cfg := types.DefaultConfig
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"cdp_endpoint": "http://127.0.0.1:9222",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "Browser configured: cdp_endpoint=http://127.0.0.1:9222. Changes take effect on next browser action." {
		t.Errorf("unexpected result text: %s", text)
	}

	if rodCtx.Config().CDPEndpoint != "http://127.0.0.1:9222" {
		t.Errorf("expected CDPEndpoint to be set, got %q", rodCtx.Config().CDPEndpoint)
	}
}

func TestConfigureHandlerClearCDPEndpoint(t *testing.T) {
	cfg := types.DefaultConfig
	cfg.CDPEndpoint = "http://127.0.0.1:9222"
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"cdp_endpoint": "",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "Browser configured: cdp_endpoint cleared. Changes take effect on next browser action." {
		t.Errorf("unexpected result text: %s", text)
	}

	if rodCtx.Config().CDPEndpoint != "" {
		t.Errorf("expected CDPEndpoint to be cleared, got %q", rodCtx.Config().CDPEndpoint)
	}
}

func TestConfigureHandlerBothArgs(t *testing.T) {
	cfg := types.DefaultConfig
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"headless":     true,
		"cdp_endpoint": "http://127.0.0.1:9222",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text == "No configuration changes requested" {
		t.Error("expected configuration changes to be reported")
	}

	if !rodCtx.Config().Headless {
		t.Error("expected Headless to be true")
	}
	if rodCtx.Config().CDPEndpoint != "http://127.0.0.1:9222" {
		t.Error("expected CDPEndpoint to be set")
	}
}
