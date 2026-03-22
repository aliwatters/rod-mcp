package types

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
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
