package tools

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aliwatters/rod-mcp/types"
)

// toolErr creates a nil result + toolError pair for handler return statements.
func toolErr(action string, err error) (*mcp.CallToolResult, error) {
	return nil, toolError(action, err)
}

// truncateContent truncates s to maxLen characters and appends a truncation notice.
// If s is already within maxLen, it is returned unchanged with truncated=false.
func truncateContent(s string, maxLen int) (string, bool) {
	if len(s) <= maxLen {
		return s, false
	}
	return s[:maxLen], true
}

// waitDOMStable waits for the DOM to stabilize, logging errors at debug level.
func waitDOMStable(page *rod.Page) {
	if err := page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff); err != nil {
		log.Debugf("WaitDOMStable: %s", err)
	}
}

// resolveSnapshotElement resolves a snapshot element reference from an MCP request.
// Supports two targeting modes:
//   - ref-based: uses the exact ref from a prior snapshot (fast, requires rod_snapshot first)
//   - name-based: searches by accessible name/role, building a snapshot if needed (semantic targeting)
//
// Returns the page, resolved element, and the human-readable element description.
func resolveSnapshotElement(rodCtx *types.Context, args map[string]interface{}, toolName string) (*rod.Page, *rod.Element, string, error) {
	ele, err := getStringArg(args, "element")
	if err != nil {
		return nil, nil, "", fmt.Errorf("%s: %w", toolName, err)
	}
	page, err := rodCtx.ControlledPage()
	if err != nil {
		return nil, nil, ele, fmt.Errorf("%s %s: %w", toolName, ele, err)
	}

	ref := getOptionalStringArg(args, "ref")
	name := getOptionalStringArg(args, "name")

	if ref == "" && name == "" {
		return nil, nil, ele, fmt.Errorf("%s %s: either 'ref' or 'name' must be provided", toolName, ele)
	}

	// If ref is provided, use the existing fast path.
	if ref != "" {
		snapshot, err := rodCtx.LatestSnapshot()
		if err != nil {
			return nil, nil, ele, fmt.Errorf("%s %s: %w", toolName, ele, err)
		}
		element, err := snapshot.LocatorInFrame(ref)
		if err != nil {
			return nil, nil, ele, fmt.Errorf("%s %s: %w", toolName, ele, err)
		}
		return page, element, ele, nil
	}

	// Name-based semantic targeting: build a snapshot and search by name/role.
	snapshot, err := rodCtx.EnsureSnapshot()
	if err != nil {
		return nil, nil, ele, fmt.Errorf("%s %s: failed to build snapshot for semantic search: %w", toolName, ele, err)
	}

	role := getOptionalStringArg(args, "role")
	matches := snapshot.FindByNameRole(name, role)

	if len(matches) == 0 {
		if role != "" {
			return nil, nil, ele, fmt.Errorf("%s %s: no element found with name %q and role %q", toolName, ele, name, role)
		}
		return nil, nil, ele, fmt.Errorf("%s %s: no element found with name %q", toolName, ele, name)
	}

	if len(matches) > 1 {
		var descriptions []string
		for _, m := range matches {
			descriptions = append(descriptions, fmt.Sprintf("  ref=%s  %s", m.Ref, m.Raw))
		}
		return nil, nil, ele, fmt.Errorf(
			"%s %s: %d elements match name %q — use 'ref' to disambiguate:\n%s",
			toolName, ele, len(matches), name, strings.Join(descriptions, "\n"),
		)
	}

	element, err := snapshot.LocatorInFrame(matches[0].Ref)
	if err != nil {
		return nil, nil, ele, fmt.Errorf("%s %s: %w", toolName, ele, err)
	}
	return page, element, ele, nil
}

// keyMap maps string key names to input.Key constants.
// Supports both named keys (Tab, Enter, ArrowUp) and single characters.
var keyMap = map[string]input.Key{
	// Function keys
	"Escape": input.Escape,
	"F1":     input.F1,
	"F2":     input.F2,
	"F3":     input.F3,
	"F4":     input.F4,
	"F5":     input.F5,
	"F6":     input.F6,
	"F7":     input.F7,
	"F8":     input.F8,
	"F9":     input.F9,
	"F10":    input.F10,
	"F11":    input.F11,
	"F12":    input.F12,

	// Navigation
	"Backspace":  input.Backspace,
	"Tab":        input.Tab,
	"Enter":      input.Enter,
	"Return":     input.Enter,
	"CapsLock":   input.CapsLock,
	"Delete":     input.Delete,
	"End":        input.End,
	"Home":       input.Home,
	"Insert":     input.Insert,
	"PageDown":   input.PageDown,
	"PageUp":     input.PageUp,
	"ArrowDown":  input.ArrowDown,
	"ArrowLeft":  input.ArrowLeft,
	"ArrowRight": input.ArrowRight,
	"ArrowUp":    input.ArrowUp,

	// Modifiers
	"Alt":          input.AltLeft,
	"AltLeft":      input.AltLeft,
	"AltRight":     input.AltRight,
	"Control":      input.ControlLeft,
	"ControlLeft":  input.ControlLeft,
	"ControlRight": input.ControlRight,
	"Meta":         input.MetaLeft,
	"MetaLeft":     input.MetaLeft,
	"MetaRight":    input.MetaRight,
	"Shift":        input.ShiftLeft,
	"ShiftLeft":    input.ShiftLeft,
	"ShiftRight":   input.ShiftRight,

	// Special keys
	"Space":       input.Space,
	"PrintScreen": input.PrintScreen,
	"ScrollLock":  input.ScrollLock,
	"Pause":       input.Pause,
	"ContextMenu": input.ContextMenu,
	"NumLock":     input.NumLock,
}

// parseKey converts a key string to an input.Key.
// For single characters, returns the rune as input.Key.
// For named keys (Tab, Enter, ArrowUp, etc.), looks up in keyMap.
func parseKey(keyStr string) (input.Key, error) {
	// Check if it's a named key first
	if key, ok := keyMap[keyStr]; ok {
		return key, nil
	}

	// For single characters, use the rune value directly
	if len(keyStr) == 1 {
		return input.Key(keyStr[0]), nil
	}

	return 0, fmt.Errorf("unknown key: %s", keyStr)
}
