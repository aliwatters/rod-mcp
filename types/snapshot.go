package types

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/aliwatters/rod-mcp/types/js"
	"github.com/aliwatters/rod-mcp/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"regexp"
	"strconv"
	"strings"
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

const snapshotTpl = `
- Page URL: {{ .URL }}
- Page Title: {{ .Title }}
- Frame Count: {{ .Frames }}
- Page Snapshot
` + "```yaml\n" + "{{ .Snapshot }}" + "\n```\n"

type Snapshot struct {
	frames       []*rod.Page
	textSnapshot string
}

func BuildSnapshot(p *rod.Page, compact bool) (*Snapshot, error) {
	snapshot := &Snapshot{
		frames: []*rod.Page{},
	}
	yamlDoc, err := snapshot.captureSnapshotWithFrames(p)
	if err != nil {
		return nil, err
	}

	if compact {
		yamlDoc = compactSnapshot(yamlDoc)
	}

	yamlBytes, err := yaml.Marshal(yamlDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed,because of yaml marshal")
	}

	pageInfo, err := p.Info()
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed")
	}

	tplInfo := map[string]any{
		"URL":      pageInfo.URL,
		"Title":    pageInfo.Title,
		"Snapshot": strings.TrimSpace(string(yamlBytes)),
		"Frames":   len(snapshot.frames),
	}
	res, err := utils.ExecuteTemplate(snapshotTpl, tplInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed, because of template exec failed")
	}
	snapshot.textSnapshot = res
	return snapshot, nil
}

func (s *Snapshot) String() string {
	return s.textSnapshot
}

func (s *Snapshot) captureSnapshotWithFrames(p *rod.Page) (*yaml.Node, error) {
	s.frames = append(s.frames, p)
	frameIndex := len(s.frames) - 1

	rawSnapshot, err := p.Eval(js.AriaSnapshot, "document.body", "({ref: true})")
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed, frame index: %d", frameIndex)
	}

	var snapNode yaml.Node

	err = yaml.Unmarshal([]byte(rawSnapshot.Value.String()), &snapNode)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed, frame index: %d", frameIndex)
	}
	return s.walk(&snapNode, frameIndex, p)

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

// walkScalarNode processes scalar nodes: adjusts frame refs and expands iframe snapshots.
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

	return node, nil
}

// walkIframeNode captures an iframe's snapshot and returns a mapping node with the result.
// Returns nil if the node doesn't have a ref or isn't an iframe.
func (s *Snapshot) walkIframeNode(node *yaml.Node, frame *rod.Page) *yaml.Node {
	re := regexp.MustCompile(`\[ref=(.*?)\]`)
	matches := re.FindStringSubmatch(node.Value)
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

		filteredKey := compactSnapshot(keyNode)
		if filteredKey == nil {
			continue
		}

		filteredValue := compactSnapshot(valueNode)
		if filteredValue == nil {
			// Key has ref or is structural but value is empty — keep key with empty value
			if nodeHasRef(keyNode) || isStructuralRole(keyNode.Value) {
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

// isStructuralRole checks if a scalar value starts with a structural ARIA role
func isStructuralRole(value string) bool {
	structuralPrefixes := []string{
		"heading ", "navigation ", "main", "banner", "contentinfo",
		"complementary", "list ", "listitem", "table ", "row ", "cell ",
		"dialog ", "alert ", "form ", "search ", "img ", "iframe ",
		"region ", "article ", "section ", "group ", "toolbar ",
		"menu ", "menubar ", "menuitem ", "tab ", "tablist ", "tabpanel ",
	}
	lower := strings.ToLower(value)
	for _, prefix := range structuralPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
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

func (s *Snapshot) LocatorInFrame(ref string) (*rod.Element, error) {
	if len(s.frames) == 0 {
		return nil, errors.New("no frames available in snapshot")
	}
	frame := s.frames[0]
	matches := regexp.MustCompile(`^f(\d+)(.*)`).FindStringSubmatch(ref)
	if len(matches) > 0 {
		frameIndex, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, errors.Wrapf(err, "locator frame failed, because of frame index is not number")
		}

		if frameIndex < 0 || frameIndex >= len(s.frames) {
			return nil, errors.Errorf("frame index %d out of range (0-%d)", frameIndex, len(s.frames)-1)
		}
		frame = s.frames[frameIndex]
		ref = matches[2]
	}
	ele, err := utils.QueryEleByAria(frame, ref)
	if err != nil {
		return nil, errors.Wrapf(err, "locator frame failed, because of query element by aria failed")
	}
	return ele, nil

}
