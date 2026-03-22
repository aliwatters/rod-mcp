package types

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// truncateScalar
// ---------------------------------------------------------------------------

func TestTruncateScalar_ShortValue(t *testing.T) {
	// Values at or below maxLen should be returned unchanged.
	input := "button \"Login\" [ref=e42]"
	got := truncateScalar(input, maxRefScalarLen)
	if got != input {
		t.Errorf("truncateScalar short: got %q, want %q", got, input)
	}
}

func TestTruncateScalar_AtBoundary(t *testing.T) {
	// A string exactly at the limit should not be truncated.
	input := strings.Repeat("x", maxRefScalarLen)
	got := truncateScalar(input, maxRefScalarLen)
	if got != input {
		t.Errorf("truncateScalar at boundary: got len %d, want %d", len(got), maxRefScalarLen)
	}
}

func TestTruncateScalar_NoRef_LongValue(t *testing.T) {
	// Values without [ref=...] that exceed maxLen should be truncated with "...".
	input := strings.Repeat("a", maxStructuralScalarLen+10)
	got := truncateScalar(input, maxStructuralScalarLen)
	if len(got) >= len(input) {
		t.Errorf("truncateScalar long no-ref: expected truncation, got len %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncateScalar long no-ref: expected '...' suffix, got %q", got)
	}
}

func TestTruncateScalar_WithRefSuffix(t *testing.T) {
	// The [ref=...] suffix must be preserved verbatim after truncation.
	prefix := strings.Repeat("b", maxRefScalarLen)
	ref := "[ref=e99]"
	input := prefix + ref
	got := truncateScalar(input, maxRefScalarLen)
	if !strings.HasSuffix(got, ref) {
		t.Errorf("truncateScalar with ref: suffix %q not preserved in %q", ref, got)
	}
	if len(got) >= len(input) {
		t.Errorf("truncateScalar with ref: expected truncation, got same len %d", len(got))
	}
}

func TestTruncateScalar_VeryShortMaxWithRef(t *testing.T) {
	// When maxLen is too small to truncate safely (prefix budget ≤ 0),
	// the value should be returned unchanged.
	input := "button [ref=e1]"
	got := truncateScalar(input, 5)
	if got != input {
		t.Errorf("truncateScalar unsafe truncation: expected unchanged %q, got %q", input, got)
	}
}

// ---------------------------------------------------------------------------
// compactScalarNode
// ---------------------------------------------------------------------------

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func TestCompactScalarNode_RefNode(t *testing.T) {
	// Nodes with [ref=...] must be kept.
	n := scalarNode(`button "Submit" [ref=e12]`)
	got := compactScalarNode(n)
	if got == nil {
		t.Fatal("compactScalarNode: ref node should not be nil")
	}
	if !strings.Contains(got.Value, "[ref=e12]") {
		t.Errorf("compactScalarNode: ref missing from %q", got.Value)
	}
}

func TestCompactScalarNode_StructuralRole(t *testing.T) {
	// Structural role nodes (e.g. "heading 1") must be kept.
	cases := []string{
		"heading 1",
		"navigation ",
		"main",
		"banner",
		"list items",
		"dialog Login",
	}
	for _, val := range cases {
		n := scalarNode(val)
		got := compactScalarNode(n)
		if got == nil {
			t.Errorf("compactScalarNode: structural role %q should not be nil", val)
		}
	}
}

func TestCompactScalarNode_TextNode(t *testing.T) {
	// Pure text: nodes should be filtered out (return nil).
	cases := []string{
		"text: Hello world",
		"text:short",
	}
	for _, val := range cases {
		n := scalarNode(val)
		got := compactScalarNode(n)
		if got != nil {
			t.Errorf("compactScalarNode: text node %q should return nil, got %q", val, got.Value)
		}
	}
}

func TestCompactScalarNode_NonStringTag(t *testing.T) {
	// Non-string tags (e.g. !!int) should pass through unchanged.
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"}
	got := compactScalarNode(n)
	if got == nil || got.Value != "42" {
		t.Errorf("compactScalarNode: non-string tag should be kept unchanged")
	}
}

func TestCompactScalarNode_LongGenericScalar(t *testing.T) {
	// Long scalars that are not text: or structural should be truncated.
	input := strings.Repeat("z", maxStructuralScalarLen+20)
	n := scalarNode(input)
	got := compactScalarNode(n)
	if got == nil {
		t.Fatal("compactScalarNode: long generic scalar should not be nil")
	}
	if len(got.Value) >= len(input) {
		t.Errorf("compactScalarNode: long scalar not truncated (len %d)", len(got.Value))
	}
}

// ---------------------------------------------------------------------------
// parseRefScalar
// ---------------------------------------------------------------------------

func TestParseRefScalar_Button(t *testing.T) {
	entry, ok := parseRefScalar(`button "Login" [ref=e42]`)
	if !ok {
		t.Fatal("parseRefScalar: expected ok=true")
	}
	if entry.Role != "button" {
		t.Errorf("Role = %q, want %q", entry.Role, "button")
	}
	if entry.Name != "Login" {
		t.Errorf("Name = %q, want %q", entry.Name, "Login")
	}
	if entry.Ref != "e42" {
		t.Errorf("Ref = %q, want %q", entry.Ref, "e42")
	}
}

func TestParseRefScalar_TextboxWithSpaceInName(t *testing.T) {
	entry, ok := parseRefScalar(`textbox "Email address" [ref=e15]`)
	if !ok {
		t.Fatal("parseRefScalar: expected ok=true")
	}
	if entry.Role != "textbox" {
		t.Errorf("Role = %q, want %q", entry.Role, "textbox")
	}
	if entry.Name != "Email address" {
		t.Errorf("Name = %q, want %q", entry.Name, "Email address")
	}
	if entry.Ref != "e15" {
		t.Errorf("Ref = %q, want %q", entry.Ref, "e15")
	}
}

func TestParseRefScalar_IframeRef(t *testing.T) {
	// iframe refs use frame-prefixed ref IDs like "f1e7"
	entry, ok := parseRefScalar(`link "Sign up" [ref=f1e7]`)
	if !ok {
		t.Fatal("parseRefScalar: expected ok=true")
	}
	if entry.Ref != "f1e7" {
		t.Errorf("Ref = %q, want %q", entry.Ref, "f1e7")
	}
}

func TestParseRefScalar_NoRef(t *testing.T) {
	_, ok := parseRefScalar(`heading "Welcome"`)
	if ok {
		t.Error("parseRefScalar: expected ok=false for scalar without ref")
	}
}

func TestParseRefScalar_MalformedRef_NoCloseBracket(t *testing.T) {
	_, ok := parseRefScalar(`button "Click" [ref=e99`)
	if ok {
		t.Error("parseRefScalar: expected ok=false for unclosed ref bracket")
	}
}

func TestParseRefScalar_NoQuotedName(t *testing.T) {
	// Some scalars have a ref but no quoted name.
	entry, ok := parseRefScalar(`img [ref=e3]`)
	if !ok {
		t.Fatal("parseRefScalar: expected ok=true for ref without quoted name")
	}
	if entry.Name != "" {
		t.Errorf("Name = %q, want empty string", entry.Name)
	}
	if entry.Role != "img" {
		t.Errorf("Role = %q, want %q", entry.Role, "img")
	}
}

func TestParseRefScalar_RawPreserved(t *testing.T) {
	raw := `button "Save" [ref=e77]`
	entry, ok := parseRefScalar(raw)
	if !ok {
		t.Fatal("parseRefScalar: expected ok=true")
	}
	if entry.Raw != raw {
		t.Errorf("Raw = %q, want %q", entry.Raw, raw)
	}
}

// ---------------------------------------------------------------------------
// cookie domain matching (matchDomainPattern integration)
// ---------------------------------------------------------------------------

func TestMatchDomainPattern_CookieLikePatterns(t *testing.T) {
	// Browsers store cookies with a leading dot to indicate subdomain scope.
	// matchDomainPattern should handle these correctly.
	tests := []struct {
		pattern string
		host    string
		want    bool
	}{
		// exact domain — no subdomain scope
		{"example.com", "example.com", true},
		{"example.com", "sub.example.com", false},

		// wildcard — covers base domain and all subdomains
		{"*.example.com", "example.com", true},
		{"*.example.com", "www.example.com", true},
		{"*.example.com", "api.v2.example.com", true},
		{"*.example.com", "notexample.com", false},

		// port stripping — matchDomainPattern receives the host without port
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "other.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.host, func(t *testing.T) {
			got := matchDomainPattern(tt.pattern, tt.host)
			if got != tt.want {
				t.Errorf("matchDomainPattern(%q, %q) = %v, want %v", tt.pattern, tt.host, got, tt.want)
			}
		})
	}
}

func TestLocatorInFrame_BoundsCheck(t *testing.T) {
	// Create a snapshot with one frame (index 0)
	s := &Snapshot{
		frames: nil, // empty frames slice
	}

	// Any frame reference should fail on empty snapshot
	_, err := s.LocatorInFrame("f0e1")
	if err == nil {
		t.Error("LocatorInFrame with frameIndex=0 on empty frames should return error")
	}

	// Test with non-existent frame index that would pass with > but fail with >=
	_, err = s.LocatorInFrame("f1e1")
	if err == nil {
		t.Error("LocatorInFrame with frameIndex=1 on empty frames should return error")
	}
}
