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
	if _, ok := props["user_data_dir"]; !ok {
		t.Error("Configure tool missing 'user_data_dir' property")
	}
	if _, ok := props["clone_domains"]; !ok {
		t.Error("Configure tool missing 'clone_domains' property")
	}
	if _, ok := props["no_clone"]; !ok {
		t.Error("Configure tool missing 'no_clone' property")
	}
	if _, ok := props["clone_all"]; !ok {
		t.Error("Configure tool missing 'clone_all' property")
	}
	if _, ok := props["stealth"]; !ok {
		t.Error("Configure tool missing 'stealth' property")
	}

	// All configure fields should be optional.
	for _, r := range Configure.InputSchema.Required {
		t.Errorf("Configure tool parameter %q should not be required", r)
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

func TestConfigureHandlerStealth(t *testing.T) {
	cfg := types.DefaultConfig
	cfg.Stealth = false
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"stealth": true,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "Browser configured: stealth=true. Changes take effect on next browser action." {
		t.Errorf("unexpected result text: %s", text)
	}

	if !rodCtx.Config().Stealth {
		t.Error("expected Stealth to be true after reconfigure")
	}
}

func TestConfigureHandlerUserDataDir(t *testing.T) {
	cfg := types.DefaultConfig
	rodCtx := types.NewContext(context.Background(), cfg)
	defer rodCtx.Close()

	handler := ConfigureHandler(rodCtx)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"headless":      false,
		"user_data_dir": "/tmp/chrome-profile",
		"clone_domains": "example.com, *.example.org",
		"no_clone":      true,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "Browser configured: headless=false, user_data_dir=/tmp/chrome-profile, clone_domains=example.com, *.example.org, no_clone=true. Changes take effect on next browser action." {
		t.Errorf("unexpected result text: %s", text)
	}

	got := rodCtx.Config()
	if got.Headless {
		t.Error("expected Headless to be false")
	}
	if got.UserDataDir != "/tmp/chrome-profile" {
		t.Errorf("UserDataDir = %q, want /tmp/chrome-profile", got.UserDataDir)
	}
	if got.NoClone != true {
		t.Error("expected NoClone to be true")
	}
	if len(got.CloneDomains) != 2 || got.CloneDomains[0] != "example.com" || got.CloneDomains[1] != "*.example.org" {
		t.Errorf("CloneDomains = %#v, want parsed domains", got.CloneDomains)
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
