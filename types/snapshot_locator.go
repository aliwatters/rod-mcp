package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-rod/rod"

	"github.com/aliwatters/rod-mcp/utils"
)

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

// LocatorInFrame resolves a ref string to a rod.Element, handling cross-frame refs
// (e.g. "f1e42" selects element e42 within frame 1).
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
