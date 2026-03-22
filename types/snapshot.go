package types

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-rod/rod"
	"gopkg.in/yaml.v3"

	"github.com/aliwatters/rod-mcp/types/js"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	// maxSequenceItems is the threshold above which sequences get truncated in compact mode.
	maxSequenceItems = 10
	// sequencePreviewItems is the number of items to keep when truncating long sequences.
	sequencePreviewItems = 5
	// maxRefScalarLen is the max length for scalars containing element refs.
	maxRefScalarLen = 120
	// maxStructuralScalarLen is the max length for structural role scalars.
	maxStructuralScalarLen = 80
	// truncationEllipsis is inserted when truncating scalars near a [ref=...] suffix.
	truncationEllipsis = "... "
	// truncationSuffixPad is the character budget reserved for the ellipsis when preserving ref suffixes.
	truncationSuffixPad = len(truncationEllipsis)
)

var (
	// refPattern matches [ref=...] suffixes in ARIA snapshot scalars.
	refPattern = regexp.MustCompile(`\[ref=(.*?)\]`)
	// iframeRefPattern matches iframe ref prefixes like "f1e42".
	iframeRefPattern = regexp.MustCompile(`^f(\d+)(.*)`)
)

const snapshotTpl = `
- Page URL: {{ .URL }}
- Page Title: {{ .Title }}
- Frame Count: {{ .Frames }}
- Page Snapshot
` + "```yaml\n" + "{{ .Snapshot }}" + "\n```\n"

// RefEntry stores metadata about an interactive element in the ARIA snapshot.
type RefEntry struct {
	Ref   string // e.g. "e42" or "f1e42"
	Role  string // e.g. "button", "link", "textbox"
	Name  string // accessible name, e.g. "Login"
	Raw   string // full scalar value
}

type Snapshot struct {
	frames       []*rod.Page
	textSnapshot string
	refIndex     []RefEntry
	// refAccum is used during walk() to accumulate ref entries in a single pass.
	// It is set to nil after walk() completes and the entries are moved to refIndex.
	refAccum *[]RefEntry
}

func BuildSnapshot(p *rod.Page, compact bool) (*Snapshot, error) {
	snapshot := &Snapshot{
		frames: []*rod.Page{},
	}
	yamlDoc, err := snapshot.captureSnapshotWithFrames(p)
	if err != nil {
		return nil, err
	}

	// refIndex is populated during walk() via walkScalarNode accumulator.
	// No separate buildRefIndex pass is needed.

	if compact {
		yamlDoc = compactSnapshot(yamlDoc)
	}

	yamlBytes, err := yaml.Marshal(yamlDoc)
	if err != nil {
		return nil, fmt.Errorf("snapshot yaml marshal: %w", err)
	}

	pageInfo, err := p.Info()
	if err != nil {
		return nil, fmt.Errorf("snapshot page info: %w", err)
	}

	tplInfo := map[string]any{
		"URL":      pageInfo.URL,
		"Title":    pageInfo.Title,
		"Snapshot": strings.TrimSpace(string(yamlBytes)),
		"Frames":   len(snapshot.frames),
	}
	res, err := utils.ExecuteTemplate(snapshotTpl, tplInfo)
	if err != nil {
		return nil, fmt.Errorf("snapshot template exec: %w", err)
	}
	snapshot.textSnapshot = res
	return snapshot, nil
}

func (s *Snapshot) String() string {
	return s.textSnapshot
}

func (s *Snapshot) captureSnapshotWithFrames(p *rod.Page) (*yaml.Node, error) {
	// Initialize the ref accumulator on the first (top-level) call.
	isRoot := s.refAccum == nil
	if isRoot {
		entries := make([]RefEntry, 0)
		s.refAccum = &entries
	}

	s.frames = append(s.frames, p)
	frameIndex := len(s.frames) - 1

	rawSnapshot, err := p.Eval(js.AriaSnapshot, "document.body", "({ref: true})")
	if err != nil {
		return nil, fmt.Errorf("snapshot frame %d eval: %w", frameIndex, err)
	}

	var snapNode yaml.Node

	err = yaml.Unmarshal([]byte(rawSnapshot.Value.String()), &snapNode)
	if err != nil {
		return nil, fmt.Errorf("snapshot frame %d yaml unmarshal: %w", frameIndex, err)
	}

	result, walkErr := s.walk(&snapNode, frameIndex, p)

	// On the root call, move accumulated refs to refIndex and clear the accumulator.
	if isRoot {
		s.refIndex = *s.refAccum
		s.refAccum = nil
	}

	return result, walkErr
}

func (s *Snapshot) walk(node *yaml.Node, frameIndex int, frame *rod.Page) (*yaml.Node, error) {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		processedRoot, err := s.walk(node.Content[0], frameIndex, frame)
		if err != nil {
			return nil, err
		}
		node.Content[0] = processedRoot
		return node, nil
	}

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			newKey, err := s.walk(node.Content[i], frameIndex, frame)
			if err != nil {
				return nil, err
			}
			newValue, err := s.walk(node.Content[i+1], frameIndex, frame)
			if err != nil {
				return nil, err
			}
			node.Content[i] = newKey
			node.Content[i+1] = newValue
		}

	case yaml.SequenceNode:
		for i, item := range node.Content {
			processed, err := s.walk(item, frameIndex, frame)
			if err != nil {
				return nil, err
			}
			node.Content[i] = processed
		}

	case yaml.ScalarNode:
		return s.walkScalarNode(node, frameIndex, frame)
	}

	return node, nil
}

// walkScalarNode processes scalar nodes: adjusts frame refs, expands iframe snapshots,
// and accumulates ref index entries in a single pass.
func (s *Snapshot) walkScalarNode(node *yaml.Node, frameIndex int, frame *rod.Page) (*yaml.Node, error) {
	if node.Tag != "!!str" {
		return node, nil
	}

	value := node.Value
	if frameIndex > 0 {
		node.Value = strings.Replace(value, "[ref=", fmt.Sprintf("[ref=f%d", frameIndex), 1)
	}

	if strings.HasPrefix(value, "iframe ") {
		if result := s.walkIframeNode(node, frame); result != nil {
			return result, nil
		}
	}

	// Accumulate ref index entries during the walk (avoids a separate tree pass).
	if s.refAccum != nil {
		effectiveValue := node.Value
		if entry, ok := parseRefScalar(effectiveValue); ok {
			*s.refAccum = append(*s.refAccum, entry)
		}
	}

	return node, nil
}

// walkIframeNode captures an iframe's snapshot and returns a mapping node with the result.
// Returns nil if the node doesn't have a ref or isn't an iframe.
func (s *Snapshot) walkIframeNode(node *yaml.Node, frame *rod.Page) *yaml.Node {
	matches := refPattern.FindStringSubmatch(node.Value)
	if len(matches) <= 1 {
		return nil
	}

	ref := matches[1]
	fallback := &yaml.Node{Kind: yaml.ScalarNode, Value: "<could not capture iframe snapshot>"}
	pairNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: node.Value}, fallback},
	}

	childFrameEle, err := utils.QueryEleByAria(frame, ref)
	if err != nil {
		return pairNode
	}
	childFrame, err := childFrameEle.Frame()
	if err != nil {
		return pairNode
	}
	childSnapshot, err := s.captureSnapshotWithFrames(childFrame)
	if err != nil || len(childSnapshot.Content) == 0 {
		return pairNode
	}

	pairNode.Content[1] = childSnapshot.Content[0]
	return pairNode
}

// compactSnapshot filters the YAML accessibility tree to reduce token usage.
// It preserves interactive elements (those with [ref=...] markers), structural
// containers, and page metadata while removing decorative text and truncating
// long content.
func compactSnapshot(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			node.Content[0] = compactSnapshot(node.Content[0])
		}
		return node
	case yaml.MappingNode:
		return compactMappingNode(node)
	case yaml.SequenceNode:
		return compactSequenceNode(node)
	case yaml.ScalarNode:
		return compactScalarNode(node)
	}

	return node
}

// compactMappingNode filters key-value pairs, keeping only those with refs or structural roles.
func compactMappingNode(node *yaml.Node) *yaml.Node {
	var newContent []*yaml.Node
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// Cache nodeHasRef result to avoid a second recursive subtree walk below.
		keyHasRef := nodeHasRef(keyNode)

		filteredKey := compactSnapshot(keyNode)
		if filteredKey == nil {
			continue
		}

		filteredValue := compactSnapshot(valueNode)
		if filteredValue == nil {
			// Key has ref or is structural but value is empty — keep key with empty value
			if keyHasRef || isStructuralRole(keyNode.Value) {
				filteredValue = &yaml.Node{Kind: yaml.ScalarNode, Value: ""}
			} else {
				continue
			}
		}

		newContent = append(newContent, filteredKey, filteredValue)
	}
	if len(newContent) == 0 {
		return nil
	}
	node.Content = newContent
	return node
}

// compactSequenceNode filters sequence items and truncates long repetitive sequences.
func compactSequenceNode(node *yaml.Node) *yaml.Node {
	var newContent []*yaml.Node
	for _, item := range node.Content {
		if filtered := compactSnapshot(item); filtered != nil {
			newContent = append(newContent, filtered)
		}
	}
	if len(newContent) == 0 {
		return nil
	}
	if len(newContent) > maxSequenceItems {
		summary := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: fmt.Sprintf("... (%d more items)", len(newContent)-sequencePreviewItems),
		}
		truncated := make([]*yaml.Node, sequencePreviewItems+1)
		copy(truncated, newContent[:sequencePreviewItems])
		truncated[sequencePreviewItems] = summary
		newContent = truncated
	}
	node.Content = newContent
	return node
}

// Compact snapshot filtering strategy:
// 1. Keep nodes with [ref=...] attributes (interactive elements)
// 2. Keep structural ARIA roles (headings, navigation, main, etc.)
// 3. Remove pure text nodes (text: ...) to reduce token count
// 4. Truncate long scalars to prevent bloated snapshots

// compactScalarNode filters scalars: keeps refs and structural roles, removes pure text.
func compactScalarNode(node *yaml.Node) *yaml.Node {
	if node.Tag != "!!str" {
		return node
	}
	value := node.Value

	if strings.Contains(value, "[ref=") {
		node.Value = truncateScalar(value, maxRefScalarLen)
		return node
	}
	if isStructuralRole(value) {
		node.Value = truncateScalar(value, maxStructuralScalarLen)
		return node
	}
	if strings.HasPrefix(value, "text: ") || strings.HasPrefix(value, "text:") {
		return nil
	}
	if len(value) > maxStructuralScalarLen {
		node.Value = truncateScalar(value, maxStructuralScalarLen)
	}
	return node
}

// nodeHasRef checks if any scalar in the subtree contains [ref=...]
func nodeHasRef(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Kind == yaml.ScalarNode {
		return strings.Contains(node.Value, "[ref=")
	}
	for _, child := range node.Content {
		if nodeHasRef(child) {
			return true
		}
	}
	return false
}

// structuralPrefixes lists ARIA role prefixes that identify structural/landmark elements.
// Declared at package level to avoid re-allocating the slice on every isStructuralRole call.
var structuralPrefixes = []string{
	"heading ", "navigation ", "main", "banner", "contentinfo",
	"complementary", "list ", "listitem", "table ", "row ", "cell ",
	"dialog ", "alert ", "form ", "search ", "img ", "iframe ",
	"region ", "article ", "section ", "group ", "toolbar ",
	"menu ", "menubar ", "menuitem ", "tab ", "tablist ", "tabpanel ",
}

// structuralRoleWords is a map of first words from structuralPrefixes for O(1) lookup.
// Built once at init time from structuralPrefixes.
var structuralRoleWords map[string]bool

func init() {
	structuralRoleWords = make(map[string]bool, len(structuralPrefixes))
	for _, prefix := range structuralPrefixes {
		// Extract first word (everything before the first space, or the whole string).
		word := prefix
		if idx := strings.Index(prefix, " "); idx >= 0 {
			word = prefix[:idx]
		}
		structuralRoleWords[word] = true
	}
}

// isStructuralRole checks if a scalar value starts with a structural ARIA role.
// Uses an O(1) map lookup on the first word of the value instead of a linear scan.
func isStructuralRole(value string) bool {
	lower := strings.ToLower(value)
	// Extract the first word of the value to look up in the map.
	firstWord := lower
	if idx := strings.Index(lower, " "); idx >= 0 {
		firstWord = lower[:idx]
	}
	return structuralRoleWords[firstWord]
}

// truncateScalar truncates a string to maxLen, preserving any [ref=...] suffix
func truncateScalar(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}

	// Preserve [ref=...] and other bracket attributes at the end
	refIdx := strings.LastIndex(value, "[ref=")
	if refIdx >= 0 {
		suffix := value[refIdx:]
		prefix := value[:refIdx]
		maxPrefix := maxLen - len(suffix) - truncationSuffixPad
		if maxPrefix > 0 && maxPrefix < len(prefix) {
			return prefix[:maxPrefix] + truncationEllipsis + suffix
		}
		return value // can't truncate safely, return as-is
	}

	return value[:maxLen-3] + "..."
}

// FindByNameRole searches the ref index for elements matching the given
// accessible name (case-insensitive substring) and optional ARIA role
// (case-insensitive exact match). Returns matching entries.
func (s *Snapshot) FindByNameRole(name, role string) []RefEntry {
	var matches []RefEntry
	nameLower := strings.ToLower(name)
	roleLower := strings.ToLower(role)
	for _, entry := range s.refIndex {
		if role != "" && strings.ToLower(entry.Role) != roleLower {
			continue
		}
		if !strings.Contains(strings.ToLower(entry.Name), nameLower) {
			continue
		}
		matches = append(matches, entry)
	}
	return matches
}

// parseRefScalar parses an ARIA snapshot scalar like:
//
//	`button "Login" [ref=e42]`
//	`textbox "Email address" [ref=e15]`
//	`link "Sign up" [ref=f1e7]`
//
// Returns the parsed RefEntry and true if the scalar contains a ref.
func parseRefScalar(value string) (RefEntry, bool) {
	refIdx := strings.Index(value, "[ref=")
	if refIdx < 0 {
		return RefEntry{}, false
	}
	refEnd := strings.Index(value[refIdx:], "]")
	if refEnd < 0 {
		return RefEntry{}, false
	}
	ref := value[refIdx+len("[ref=") : refIdx+refEnd]

	// Parse role: first word before a space
	var role string
	spaceIdx := strings.Index(value, " ")
	if spaceIdx > 0 {
		role = value[:spaceIdx]
	}

	// Parse name: text between quotes
	name := extractQuotedName(value)

	return RefEntry{
		Ref:  ref,
		Role: role,
		Name: name,
		Raw:  value,
	}, true
}

// extractQuotedName extracts the accessible name from between quotes in an
// ARIA snapshot scalar. Handles both `"name"` patterns.
func extractQuotedName(value string) string {
	first := strings.Index(value, "\"")
	if first < 0 {
		return ""
	}
	rest := value[first+1:]
	last := strings.Index(rest, "\"")
	if last < 0 {
		return ""
	}
	return rest[:last]
}

func (s *Snapshot) LocatorInFrame(ref string) (*rod.Element, error) {
	if len(s.frames) == 0 {
		return nil, errors.New("no frames available in snapshot")
	}
	frame := s.frames[0]
	matches := iframeRefPattern.FindStringSubmatch(ref)
	if len(matches) > 0 {
		frameIndex, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("locator frame index parse: %w", err)
		}

		if frameIndex < 0 || frameIndex >= len(s.frames) {
			return nil, fmt.Errorf("frame index %d out of range (0-%d)", frameIndex, len(s.frames)-1)
		}
		frame = s.frames[frameIndex]
		ref = matches[2]
	}
	ele, err := utils.QueryEleByAria(frame, ref)
	if err != nil {
		return nil, fmt.Errorf("locator frame query element by aria: %w", err)
	}
	return ele, nil

}
