package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	FrameListToolKey = "rod_frame_list"
)

var (
	FrameList = mcp.NewTool(FrameListToolKey,
		mcp.WithDescription("List all frames (iframes) on the current page with their URLs, names, and snapshot refs. Use ref-based targeting (e.g. f1e42) to interact with elements inside frames."),
	)
)

var (
	FrameListHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("list frames", err)
			}

			iframes, err := page.Elements("iframe")
			if err != nil {
				return toolErr("find iframes", err)
			}

			if len(iframes) == 0 {
				return mcp.NewToolResultText("No iframes found on the current page"), nil
			}

			var frames []map[string]interface{}
			for i, iframe := range iframes {
				frame := map[string]interface{}{
					"index": i,
				}
				// Get src attribute
				src, _ := iframe.Attribute("src")
				if src != nil {
					frame["src"] = *src
				}
				// Get name attribute
				name, _ := iframe.Attribute("name")
				if name != nil && *name != "" {
					frame["name"] = *name
				}
				// Get id attribute
				id, _ := iframe.Attribute("id")
				if id != nil && *id != "" {
					frame["id"] = *id
				}
				frame["ref_prefix"] = fmt.Sprintf("f%d", i)
				frames = append(frames, frame)
			}

			result := map[string]interface{}{
				"count":  len(frames),
				"frames": frames,
				"hint":   "Use ref-based targeting with frame prefix (e.g. f0e42 for element e42 in frame 0) to interact with elements inside iframes",
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// resolvePiercingSelector handles selectors with >>> (shadow DOM piercing combinator).
// For example: "my-component >>> .inner-button" traverses into shadow roots.
// Returns the element found after piercing through shadow roots.
func resolvePiercingSelector(page *rod.Page, selector string) (*rod.Element, error) {
	parts := strings.Split(selector, ">>>")
	if len(parts) == 1 {
		// No piercing — use normal selector
		return page.Element(strings.TrimSpace(selector))
	}

	// Traverse shadow roots
	el, err := page.Element(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("no element found for %q: %w", strings.TrimSpace(parts[0]), err)
	}

	for i := 1; i < len(parts); i++ {
		shadow, shadowErr := el.ShadowRoot()
		if shadowErr != nil {
			return nil, fmt.Errorf("element %q has no shadow root: %w", strings.TrimSpace(parts[i-1]), shadowErr)
		}
		el, err = shadow.Element(strings.TrimSpace(parts[i]))
		if err != nil {
			return nil, fmt.Errorf("no element found for %q inside shadow root: %w", strings.TrimSpace(parts[i]), err)
		}
	}
	return el, nil
}
