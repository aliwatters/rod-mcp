package types

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mappingNode builds a yaml.MappingNode with alternating key/value scalar pairs.
func mappingNode(pairs ...string) *yaml.Node {
	if len(pairs)%2 != 0 {
		panic("mappingNode: pairs must be even")
	}
	n := &yaml.Node{Kind: yaml.MappingNode}
	for i := 0; i < len(pairs); i += 2 {
		n.Content = append(n.Content, scalarNode(pairs[i]), scalarNode(pairs[i+1]))
	}
	return n
}

// sequenceNodeOf builds a yaml.SequenceNode from the provided scalar values.
func sequenceNodeOf(values ...string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode}
	for _, v := range values {
		n.Content = append(n.Content, scalarNode(v))
	}
	return n
}

// documentNode wraps a single node in a yaml.DocumentNode.
func documentNode(child *yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{child},
	}
}

// ---------------------------------------------------------------------------
// nodeHasRef
// ---------------------------------------------------------------------------

func TestNodeHasRef_NilNode(t *testing.T) {
	if nodeHasRef(nil) {
		t.Error("nodeHasRef(nil) should return false")
	}
}

func TestNodeHasRef_ScalarWithRef(t *testing.T) {
	n := scalarNode(`button "Submit" [ref=e42]`)
	if !nodeHasRef(n) {
		t.Error("nodeHasRef: scalar with [ref=...] should return true")
	}
}

func TestNodeHasRef_ScalarWithoutRef(t *testing.T) {
	n := scalarNode("heading 1")
	if nodeHasRef(n) {
		t.Error("nodeHasRef: scalar without [ref=...] should return false")
	}
}

func TestNodeHasRef_EmptyScalar(t *testing.T) {
	n := scalarNode("")
	if nodeHasRef(n) {
		t.Error("nodeHasRef: empty scalar should return false")
	}
}

func TestNodeHasRef_NestedRef(t *testing.T) {
	// A mapping node where one child scalar has a ref.
	parent := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			scalarNode("key"),
			scalarNode(`link "Home" [ref=e7]`),
		},
	}
	if !nodeHasRef(parent) {
		t.Error("nodeHasRef: mapping with a ref in child should return true")
	}
}

func TestNodeHasRef_SequenceWithRef(t *testing.T) {
	seq := sequenceNodeOf("plain text", `input [ref=e3]`, "more text")
	if !nodeHasRef(seq) {
		t.Error("nodeHasRef: sequence containing a ref scalar should return true")
	}
}

func TestNodeHasRef_SequenceWithoutRef(t *testing.T) {
	seq := sequenceNodeOf("heading 1", "some text", "footer")
	if nodeHasRef(seq) {
		t.Error("nodeHasRef: sequence without any ref should return false")
	}
}

// ---------------------------------------------------------------------------
// isStructuralRole
// ---------------------------------------------------------------------------

func TestIsStructuralRole_KnownRoles(t *testing.T) {
	knownRoles := []string{
		"heading 1",
		"heading 2",
		"navigation main",
		"main",
		"banner",
		"contentinfo",
		"complementary sidebar",
		"list 3 items",
		"listitem",
		"table",
		"row",
		"cell",
		"dialog modal",
		"alert",
		"form",
		"search",
		"img",
		"iframe",
		"region",
		"article",
		"section",
		"group",
		"toolbar",
		"menu",
		"menubar",
		"menuitem",
		"tab",
		"tablist",
		"tabpanel",
	}
	for _, role := range knownRoles {
		if !isStructuralRole(role) {
			t.Errorf("isStructuralRole(%q) = false, want true", role)
		}
	}
}

func TestIsStructuralRole_CaseInsensitive(t *testing.T) {
	cases := []string{
		"Heading 1",
		"MAIN",
		"Navigation",
		"Banner",
	}
	for _, val := range cases {
		if !isStructuralRole(val) {
			t.Errorf("isStructuralRole(%q): case-insensitive match should return true", val)
		}
	}
}

func TestIsStructuralRole_NonStructural(t *testing.T) {
	nonStructural := []string{
		"",
		"text: Hello",
		"button Submit",
		"link",
		"checkbox",
		"textbox",
		"span",
		"div",
	}
	for _, val := range nonStructural {
		if isStructuralRole(val) {
			t.Errorf("isStructuralRole(%q) = true, want false", val)
		}
	}
}

// ---------------------------------------------------------------------------
// compactMappingNode
// ---------------------------------------------------------------------------

func TestCompactMappingNode_KeepsRefPairs(t *testing.T) {
	// A mapping where the key has a ref — that pair must be kept.
	// A pair with a "text: ..." key gets filtered because compactScalarNode
	// drops text: scalars.
	node := mappingNode(
		`button "OK" [ref=e1]`, "value1",
		"text: decorative prose", "text: more prose",
	)
	got := compactMappingNode(node)
	if got == nil {
		t.Fatal("compactMappingNode: should not return nil when ref pair exists")
	}
	// The text: key is filtered by compactScalarNode, so only the ref pair remains.
	if len(got.Content) != 2 {
		t.Errorf("compactMappingNode: Content len = %d, want 2 (1 pair)", len(got.Content))
	}
}

func TestCompactMappingNode_KeepsStructuralKey(t *testing.T) {
	// A structural key (e.g. "heading 1") should be retained.
	node := mappingNode(
		"heading 1", `link "Home" [ref=e2]`,
		"text: some prose", "ignored",
	)
	got := compactMappingNode(node)
	if got == nil {
		t.Fatal("compactMappingNode: should not return nil with structural key")
	}
	// heading key with a ref value — 1 pair retained.
	if len(got.Content) < 2 {
		t.Errorf("compactMappingNode: expected at least 1 pair, got %d content nodes", len(got.Content))
	}
}

func TestCompactMappingNode_AllFilteredReturnsNil(t *testing.T) {
	// Only text: nodes — everything should be filtered out.
	node := mappingNode(
		"text: hello", "text: world",
	)
	got := compactMappingNode(node)
	if got != nil {
		t.Errorf("compactMappingNode: all-filtered mapping should return nil, got %v", got)
	}
}

func TestCompactMappingNode_EmptyMapping(t *testing.T) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	got := compactMappingNode(node)
	if got != nil {
		t.Error("compactMappingNode: empty mapping should return nil")
	}
}

func TestCompactMappingNode_RefKeyEmptyValue(t *testing.T) {
	// Key has a ref, value is a pure-text scalar that gets filtered.
	// Per the code, key-with-ref must be kept with an empty value placeholder.
	keyNode := scalarNode(`button "Click" [ref=e5]`)
	valNode := scalarNode("text: pure text prose")
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{keyNode, valNode},
	}
	got := compactMappingNode(node)
	if got == nil {
		t.Fatal("compactMappingNode: ref key with filtered value should produce non-nil with empty value placeholder")
	}
	if len(got.Content) != 2 {
		t.Errorf("compactMappingNode: expected 2 content nodes (key + empty placeholder), got %d", len(got.Content))
	}
	if got.Content[1].Value != "" {
		t.Errorf("compactMappingNode: placeholder value = %q, want empty string", got.Content[1].Value)
	}
}

// ---------------------------------------------------------------------------
// compactSequenceNode
// ---------------------------------------------------------------------------

func TestCompactSequenceNode_EmptySequence(t *testing.T) {
	node := &yaml.Node{Kind: yaml.SequenceNode}
	got := compactSequenceNode(node)
	if got != nil {
		t.Error("compactSequenceNode: empty sequence should return nil")
	}
}

func TestCompactSequenceNode_AllFilteredReturnsNil(t *testing.T) {
	// All text: items get filtered — result should be nil.
	node := sequenceNodeOf("text: hello", "text: world", "text: foo")
	got := compactSequenceNode(node)
	if got != nil {
		t.Error("compactSequenceNode: all-filtered sequence should return nil")
	}
}

func TestCompactSequenceNode_ShortSequenceUnchanged(t *testing.T) {
	// Sequences at or below maxSequenceItems (10) must not be truncated.
	vals := make([]string, maxSequenceItems)
	for i := range vals {
		vals[i] = fmt.Sprintf(`button "Item %d" [ref=e%d]`, i, i)
	}
	node := sequenceNodeOf(vals...)
	got := compactSequenceNode(node)
	if got == nil {
		t.Fatal("compactSequenceNode: short sequence should not return nil")
	}
	if len(got.Content) != maxSequenceItems {
		t.Errorf("compactSequenceNode: len(Content) = %d, want %d", len(got.Content), maxSequenceItems)
	}
}

func TestCompactSequenceNode_LongSequenceTruncated(t *testing.T) {
	// Sequences exceeding maxSequenceItems must be truncated to sequencePreviewItems+1.
	count := maxSequenceItems + 5
	vals := make([]string, count)
	for i := range vals {
		vals[i] = fmt.Sprintf(`button "Item %d" [ref=e%d]`, i, i)
	}
	node := sequenceNodeOf(vals...)
	got := compactSequenceNode(node)
	if got == nil {
		t.Fatal("compactSequenceNode: long sequence should not return nil")
	}
	wantLen := sequencePreviewItems + 1 // preview items + summary
	if len(got.Content) != wantLen {
		t.Errorf("compactSequenceNode: len(Content) = %d, want %d", len(got.Content), wantLen)
	}
}

func TestCompactSequenceNode_TruncationSummaryItem(t *testing.T) {
	// The last item in a truncated sequence must be a summary scalar.
	count := maxSequenceItems + 3
	vals := make([]string, count)
	for i := range vals {
		vals[i] = fmt.Sprintf(`link "Link %d" [ref=e%d]`, i, i)
	}
	node := sequenceNodeOf(vals...)
	got := compactSequenceNode(node)
	if got == nil {
		t.Fatal("compactSequenceNode: expected non-nil result")
	}
	last := got.Content[len(got.Content)-1]
	if last.Kind != yaml.ScalarNode {
		t.Fatal("compactSequenceNode: last item should be a scalar summary")
	}
	if !strings.Contains(last.Value, "more items") {
		t.Errorf("compactSequenceNode: summary item = %q, want to contain 'more items'", last.Value)
	}
}

func TestCompactSequenceNode_TruncationSummaryCount(t *testing.T) {
	// The summary must report the correct number of omitted items.
	extra := 7
	count := maxSequenceItems + extra
	vals := make([]string, count)
	for i := range vals {
		vals[i] = fmt.Sprintf(`link "Link %d" [ref=e%d]`, i, i)
	}
	node := sequenceNodeOf(vals...)
	got := compactSequenceNode(node)
	if got == nil {
		t.Fatal("compactSequenceNode: expected non-nil result")
	}
	last := got.Content[len(got.Content)-1]
	omitted := count - sequencePreviewItems
	wantFragment := fmt.Sprintf("(%d more items)", omitted)
	if !strings.Contains(last.Value, wantFragment) {
		t.Errorf("compactSequenceNode: summary = %q, want to contain %q", last.Value, wantFragment)
	}
}

func TestCompactSequenceNode_PreservesRefItems(t *testing.T) {
	// Items with refs must not be dropped during filtering.
	node := sequenceNodeOf(
		"text: ignore me",
		`input "Email" [ref=e10]`,
		"text: ignore me too",
	)
	got := compactSequenceNode(node)
	if got == nil {
		t.Fatal("compactSequenceNode: should keep ref items")
	}
	if len(got.Content) != 1 {
		t.Errorf("compactSequenceNode: len(Content) = %d, want 1 (only the ref item)", len(got.Content))
	}
}

// ---------------------------------------------------------------------------
// compactSnapshot (top-level dispatcher)
// ---------------------------------------------------------------------------

func TestCompactSnapshot_NilReturnsNil(t *testing.T) {
	got := compactSnapshot(nil)
	if got != nil {
		t.Error("compactSnapshot(nil) should return nil")
	}
}

func TestCompactSnapshot_DocumentNode(t *testing.T) {
	// Document wrapping a mapping with a ref pair should pass through.
	inner := mappingNode(`button "Go" [ref=e1]`, "active")
	doc := documentNode(inner)
	got := compactSnapshot(doc)
	if got == nil {
		t.Fatal("compactSnapshot: document with ref content should not return nil")
	}
	if got.Kind != yaml.DocumentNode {
		t.Errorf("compactSnapshot: result Kind = %v, want DocumentNode", got.Kind)
	}
}

func TestCompactSnapshot_DocumentNode_EmptyContent(t *testing.T) {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: nil}
	got := compactSnapshot(doc)
	// Should return the document node unchanged (no content to process).
	if got == nil {
		t.Fatal("compactSnapshot: empty document should not return nil (it returns the doc node itself)")
	}
}

func TestCompactSnapshot_MappingNode_Dispatches(t *testing.T) {
	// compactSnapshot should delegate mapping nodes to compactMappingNode.
	node := mappingNode("text: hello", "text: world")
	got := compactSnapshot(node)
	if got != nil {
		t.Error("compactSnapshot: all-text mapping should be filtered to nil")
	}
}

func TestCompactSnapshot_SequenceNode_Dispatches(t *testing.T) {
	// compactSnapshot should delegate sequence nodes to compactSequenceNode.
	node := sequenceNodeOf("text: a", "text: b")
	got := compactSnapshot(node)
	if got != nil {
		t.Error("compactSnapshot: all-text sequence should be filtered to nil")
	}
}

func TestCompactSnapshot_ScalarNode_Dispatches(t *testing.T) {
	// compactSnapshot should delegate scalar nodes to compactScalarNode.
	n := scalarNode(`button "Save" [ref=e9]`)
	got := compactSnapshot(n)
	if got == nil {
		t.Error("compactSnapshot: ref scalar should not be filtered")
	}
}

func TestCompactSnapshot_UnknownKind_PassThrough(t *testing.T) {
	// Alias or other nodes (Kind not Document/Mapping/Sequence/Scalar) pass through unchanged.
	n := &yaml.Node{Kind: yaml.AliasNode}
	got := compactSnapshot(n)
	if got != n {
		t.Error("compactSnapshot: alias node should be returned unchanged")
	}
}

func TestCompactSnapshot_NestedStructure(t *testing.T) {
	// A realistic nested structure: document > mapping with ref values.
	inner := &yaml.Node{Kind: yaml.MappingNode}
	inner.Content = append(inner.Content,
		scalarNode("navigation main"),
		sequenceNodeOf(
			`link "Home" [ref=e1]`,
			`link "About" [ref=e2]`,
			"text: decorative text",
		),
	)
	doc := documentNode(inner)
	got := compactSnapshot(doc)
	if got == nil {
		t.Fatal("compactSnapshot: nested structure with refs should not be nil")
	}
	if got.Kind != yaml.DocumentNode {
		t.Errorf("compactSnapshot: result Kind = %v, want DocumentNode", got.Kind)
	}
	// Inner mapping should still exist.
	if len(got.Content) == 0 {
		t.Error("compactSnapshot: document content should not be empty")
	}
}
