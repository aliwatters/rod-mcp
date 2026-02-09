package types

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMergesDefaults(t *testing.T) {
	// Write a minimal config with only mode set
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "minimal.yaml")
	if err := os.WriteFile(cfgPath, []byte("mode: vision\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.ServerName != DefaultServerName {
		t.Errorf("ServerName = %q, want %q (from DefaultConfig)", cfg.ServerName, DefaultServerName)
	}
	if cfg.Mode != Vision {
		t.Errorf("Mode = %q, want %q (from YAML)", cfg.Mode, Vision)
	}
	if cfg.ImageResponses != ImageResponsesAllow {
		t.Errorf("ImageResponses = %q, want %q (from DefaultConfig)", cfg.ImageResponses, ImageResponsesAllow)
	}
}

func TestLoadConfigAcceptsCustomFilename(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "my-custom-config.yaml")
	if err := os.WriteFile(cfgPath, []byte("mode: text\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig should accept custom filename, got: %v", err)
	}
	if cfg.Mode != Text {
		t.Errorf("Mode = %q, want %q", cfg.Mode, Text)
	}
}

func TestMatchDomainPattern(t *testing.T) {
	tests := []struct {
		pattern string
		host    string
		want    bool
	}{
		// Exact matches
		{"example.com", "example.com", true},
		{"example.com", "www.example.com", false},
		{"example.com", "other.com", false},

		// Wildcard matches
		{"*.example.com", "www.example.com", true},
		{"*.example.com", "api.example.com", true},
		{"*.example.com", "sub.api.example.com", true},
		{"*.example.com", "example.com", true}, // base domain should also match
		{"*.example.com", "other.com", false},
		{"*.example.com", "exampleXcom", false},

		// Case insensitivity
		{"*.Example.COM", "www.example.com", true},
		{"example.com", "EXAMPLE.COM", true},

		// Additional wildcard patterns
		{"*.myapp.org", "www.myapp.org", true},
		{"*.myapp.org", "cdn.myapp.org", true},
		{"*.myapp.org", "myapp.org", true},
		{"*.myapp.test", "www.myapp.test", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.host, func(t *testing.T) {
			got := matchDomainPattern(tt.pattern, tt.host)
			if got != tt.want {
				t.Errorf("matchDomainPattern(%q, %q) = %v, want %v", tt.pattern, tt.host, got, tt.want)
			}
		})
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://www.example.com/path", "www.example.com"},
		{"https://www.example.com:443/path", "www.example.com"},
		{"http://example.com", "example.com"},
		{"https://api.myapp.org/v1/endpoint", "api.myapp.org"},
		{"www.example.com", "www.example.com"}, // no scheme
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractHost(tt.url)
			if got != tt.want {
				t.Errorf("extractHost(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestGetHeadersForURL(t *testing.T) {
	config := &Config{
		ExtraHTTPHeaders: map[string]string{
			"X-Global": "global-value",
		},
		DomainHeaders: map[string]map[string]string{
			"*.myapp.org": {
				"X-Bypass-Token": "secret123",
			},
			"*.myapp.test": {
				"X-Bypass-Token": "test-secret",
			},
			"api.example.com": {
				"Authorization": "Bearer token",
			},
		},
	}

	tests := []struct {
		url         string
		wantHeaders map[string]string
	}{
		{
			"https://www.myapp.org/page",
			map[string]string{
				"X-Global":       "global-value",
				"X-Bypass-Token": "secret123",
			},
		},
		{
			"https://www.myapp.test/page",
			map[string]string{
				"X-Global":       "global-value",
				"X-Bypass-Token": "test-secret",
			},
		},
		{
			"https://api.example.com/v1",
			map[string]string{
				"X-Global":      "global-value",
				"Authorization": "Bearer token",
			},
		},
		{
			"https://other.com/page",
			map[string]string{
				"X-Global": "global-value",
			},
		},
		{
			"", // empty URL
			map[string]string{
				"X-Global": "global-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := config.GetHeadersForURL(tt.url)
			if len(got) != len(tt.wantHeaders) {
				t.Errorf("GetHeadersForURL(%q) returned %d headers, want %d", tt.url, len(got), len(tt.wantHeaders))
			}
			for k, v := range tt.wantHeaders {
				if got[k] != v {
					t.Errorf("GetHeadersForURL(%q)[%q] = %q, want %q", tt.url, k, got[k], v)
				}
			}
		})
	}
}
