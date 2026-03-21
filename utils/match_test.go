package utils

import "testing"

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		s, pattern string
		want       bool
	}{
		{"https://api.example.com/data", "*api.example.com*", true},
		{"https://api.example.com/data", "*other.com*", false},
		{"https://example.com/path", "*", true},
		{"https://example.com/path", "https://example.com/path", true},
		{"https://example.com/path", "https://example.com/other", false},
		{"https://example.com/a", "https://example.com/?", true},
		{"https://example.com/ab", "https://example.com/?", false},
		{"", "*", true},
		{"", "", true},
		{"a", "", false},
	}

	for _, tc := range tests {
		got := MatchWildcard(tc.s, tc.pattern)
		if got != tc.want {
			t.Errorf("MatchWildcard(%q, %q) = %v, want %v", tc.s, tc.pattern, got, tc.want)
		}
	}
}
