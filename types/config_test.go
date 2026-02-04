package types

import "testing"

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

		// TravelBlog specific
		{"*.travelblog.org", "www.travelblog.org", true},
		{"*.travelblog.org", "cdn.travelblog.org", true},
		{"*.travelblog.org", "travelblog.org", true},
		{"*.travelblog.test", "www.travelblog.test", true},
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
		{"https://api.travelblog.org/v1/endpoint", "api.travelblog.org"},
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
			"*.travelblog.org": {
				"X-TravelBlog-Secret": "secret123",
			},
			"*.travelblog.test": {
				"X-TravelBlog-Secret": "test-secret",
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
			"https://www.travelblog.org/page",
			map[string]string{
				"X-Global":            "global-value",
				"X-TravelBlog-Secret": "secret123",
			},
		},
		{
			"https://www.travelblog.test/page",
			map[string]string{
				"X-Global":            "global-value",
				"X-TravelBlog-Secret": "test-secret",
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
