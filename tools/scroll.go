package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	ScrollToolKey = "rod_scroll"
)

var (
	Scroll = mcp.NewTool(ScrollToolKey,
		mcp.WithDescription("Scroll the page or a specific element. Supports directional scrolling, absolute positions, and scrolling to top/bottom."),
		mcp.WithString("direction", mcp.Description("Scroll direction: up, down, left, right, top, bottom")),
		mcp.WithNumber("amount", mcp.Description("Number of pixels to scroll (default: 500). Ignored for top/bottom.")),
		mcp.WithNumber("x", mcp.Description("Absolute horizontal scroll position in pixels")),
		mcp.WithNumber("y", mcp.Description("Absolute vertical scroll position in pixels")),
		mcp.WithString("selector", mcp.Description("CSS selector of element to scroll into view")),
	)
)

var (
	ScrollHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("scroll", err)
			}

			args := request.GetArguments()
			direction := getOptionalStringArg(args, "direction")
			selector := getOptionalStringArg(args, "selector")

			// Scroll element into view
			if selector != "" {
				el, err := page.Element(selector)
				if err != nil {
					return toolErr(fmt.Sprintf("scroll to element %q", selector), err)
				}
				if err = el.ScrollIntoView(); err != nil {
					return toolErr(fmt.Sprintf("scroll element %q into view", selector), err)
				}
				waitDOMStable(page)

				pos, err := page.Eval(`() => ({ x: window.scrollX, y: window.scrollY })`)
				if err != nil {
					return mcp.NewToolResultText(fmt.Sprintf("Scrolled element %q into view", selector)), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("Scrolled element %q into view (position: %.0f, %.0f)",
					selector, pos.Value.Get("x").Num(), pos.Value.Get("y").Num())), nil
			}

			// Absolute position scroll
			if _, hasX := args["x"]; hasX {
				x, err := getFloatArg(args, "x")
				if err != nil {
					return toolErr("scroll", err)
				}
				y := getOptionalFloatArg(args, "y", 0)
				_, err = page.Eval(`(x, y) => window.scrollTo(x, y)`, x, y)
				if err != nil {
					return toolErr("scroll to position", err)
				}
				waitDOMStable(page)
				return mcp.NewToolResultText(fmt.Sprintf("Scrolled to position (%.0f, %.0f)", x, y)), nil
			}
			if _, hasY := args["y"]; hasY {
				y, err := getFloatArg(args, "y")
				if err != nil {
					return toolErr("scroll", err)
				}
				_, err = page.Eval(`(x, y) => window.scrollTo(x, y)`, 0, y)
				if err != nil {
					return toolErr("scroll to position", err)
				}
				waitDOMStable(page)
				return mcp.NewToolResultText(fmt.Sprintf("Scrolled to position (0, %.0f)", y)), nil
			}

			// Direction-based scrolling
			if direction == "" {
				direction = "down"
			}
			amount := getOptionalFloatArg(args, "amount", 500)

			var script string
			switch direction {
			case "down":
				script = fmt.Sprintf(`() => window.scrollBy(0, %f)`, amount)
			case "up":
				script = fmt.Sprintf(`() => window.scrollBy(0, -%f)`, amount)
			case "right":
				script = fmt.Sprintf(`() => window.scrollBy(%f, 0)`, amount)
			case "left":
				script = fmt.Sprintf(`() => window.scrollBy(-%f, 0)`, amount)
			case "top":
				script = `() => window.scrollTo(0, 0)`
			case "bottom":
				script = `() => window.scrollTo(0, document.body.scrollHeight)`
			default:
				return nil, fmt.Errorf("invalid direction %q: must be up, down, left, right, top, or bottom", direction)
			}

			if _, err = page.Eval(script); err != nil {
				return toolErr("scroll "+direction, err)
			}
			waitDOMStable(page)

			pos, err := page.Eval(`() => ({ x: window.scrollX, y: window.scrollY })`)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Scrolled %s", direction)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Scrolled %s (position: %.0f, %.0f)",
				direction, pos.Value.Get("x").Num(), pos.Value.Get("y").Num())), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)
