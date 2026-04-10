package tools

import (
	"testing"

	"github.com/aliwatters/rod-mcp/types"
)

func TestLoginToolDefinition(t *testing.T) {
	if Login.Name != LoginToolKey {
		t.Errorf("Login tool name = %q, want %q", Login.Name, LoginToolKey)
	}

	props := Login.InputSchema.Properties
	if props == nil {
		t.Fatal("Login tool has no properties")
	}

	required := []string{"url", "username", "password"}
	for _, key := range required {
		if _, ok := props[key]; !ok {
			t.Errorf("Login tool missing required property %q", key)
		}
	}

	optional := []string{
		"username_selector", "password_selector", "submit_selector",
		"success_selector", "success_url_contains", "timeout",
		"trigger_selector", "form_container",
	}
	for _, key := range optional {
		if _, ok := props[key]; !ok {
			t.Errorf("Login tool missing optional property %q", key)
		}
	}
}

func TestParseLoginParams(t *testing.T) {
	cfg := types.Config{}

	t.Run("required fields", func(t *testing.T) {
		args := map[string]interface{}{
			"url":      "https://example.com/login",
			"username": "user@test.com",
			"password": "secret",
		}
		p, err := parseLoginParams(args, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.loginURL != "https://example.com/login" {
			t.Errorf("loginURL = %q, want %q", p.loginURL, "https://example.com/login")
		}
		if p.username != "user@test.com" {
			t.Errorf("username = %q, want %q", p.username, "user@test.com")
		}
		if p.password != "secret" {
			t.Errorf("password = %q, want %q", p.password, "secret")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		args := map[string]interface{}{
			"url":      "https://example.com/login",
			"username": "user",
			"password": "pass",
		}
		p, err := parseLoginParams(args, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.passSelector != "input[type=password]" {
			t.Errorf("passSelector = %q, want default", p.passSelector)
		}
		if p.submitSelector != "button[type=submit]" {
			t.Errorf("submitSelector = %q, want default", p.submitSelector)
		}
		if p.formContainer != "[role=dialog]" {
			t.Errorf("formContainer = %q, want default", p.formContainer)
		}
		if p.timeout != defaultLoginTimeoutMs {
			t.Errorf("timeout = %v, want %v", p.timeout, defaultLoginTimeoutMs)
		}
	})

	t.Run("config overrides defaults", func(t *testing.T) {
		cfgOverride := types.Config{
			LoginPasswordSelector: "#my-password",
			LoginSubmitSelector:   "#my-submit",
		}
		args := map[string]interface{}{
			"url":      "https://example.com/login",
			"username": "user",
			"password": "pass",
		}
		p, err := parseLoginParams(args, cfgOverride)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.passSelector != "#my-password" {
			t.Errorf("passSelector = %q, want config override %q", p.passSelector, "#my-password")
		}
		if p.submitSelector != "#my-submit" {
			t.Errorf("submitSelector = %q, want config override %q", p.submitSelector, "#my-submit")
		}
	})

	t.Run("request overrides config", func(t *testing.T) {
		cfgOverride := types.Config{
			LoginPasswordSelector: "#config-password",
			LoginSubmitSelector:   "#config-submit",
		}
		args := map[string]interface{}{
			"url":               "https://example.com/login",
			"username":          "user",
			"password":          "pass",
			"password_selector": "#request-password",
			"submit_selector":   "#request-submit",
		}
		p, err := parseLoginParams(args, cfgOverride)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.passSelector != "#request-password" {
			t.Errorf("passSelector = %q, want request override %q", p.passSelector, "#request-password")
		}
		if p.submitSelector != "#request-submit" {
			t.Errorf("submitSelector = %q, want request override %q", p.submitSelector, "#request-submit")
		}
	})

	t.Run("missing url", func(t *testing.T) {
		args := map[string]interface{}{
			"username": "user",
			"password": "pass",
		}
		_, err := parseLoginParams(args, cfg)
		if err == nil {
			t.Error("expected error for missing url")
		}
	})

	t.Run("missing username", func(t *testing.T) {
		args := map[string]interface{}{
			"url":      "https://example.com/login",
			"password": "pass",
		}
		_, err := parseLoginParams(args, cfg)
		if err == nil {
			t.Error("expected error for missing username")
		}
	})

	t.Run("missing password", func(t *testing.T) {
		args := map[string]interface{}{
			"url":      "https://example.com/login",
			"username": "user",
		}
		_, err := parseLoginParams(args, cfg)
		if err == nil {
			t.Error("expected error for missing password")
		}
	})

	t.Run("optional selectors", func(t *testing.T) {
		args := map[string]interface{}{
			"url":                  "https://example.com/login",
			"username":             "user",
			"password":             "pass",
			"username_selector":    "#user",
			"success_selector":     ".dashboard",
			"success_url_contains": "/home",
			"timeout":              5000.0,
			"trigger_selector":     "#open-login",
			"form_container":       "#login-modal",
		}
		p, err := parseLoginParams(args, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.userSelector != "#user" {
			t.Errorf("userSelector = %q, want %q", p.userSelector, "#user")
		}
		if p.successSelector != ".dashboard" {
			t.Errorf("successSelector = %q, want %q", p.successSelector, ".dashboard")
		}
		if p.successURL != "/home" {
			t.Errorf("successURL = %q, want %q", p.successURL, "/home")
		}
		if p.timeout != 5000.0 {
			t.Errorf("timeout = %v, want %v", p.timeout, 5000.0)
		}
		if p.triggerSelector != "#open-login" {
			t.Errorf("triggerSelector = %q, want %q", p.triggerSelector, "#open-login")
		}
		if p.formContainer != "#login-modal" {
			t.Errorf("formContainer = %q, want %q", p.formContainer, "#login-modal")
		}
	})
}

func TestDefaultUsernameSelectors(t *testing.T) {
	if len(defaultUsernameSelectors) == 0 {
		t.Fatal("defaultUsernameSelectors should not be empty")
	}
	// Verify all selectors are valid CSS-like patterns
	for _, sel := range defaultUsernameSelectors {
		if sel == "" {
			t.Error("empty selector in defaultUsernameSelectors")
		}
	}
}
