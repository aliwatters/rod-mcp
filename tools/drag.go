package tools

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	DragToolKey = "rod_drag"
)

var (
	Drag = mcp.NewTool(DragToolKey,
		mcp.WithDescription("Perform drag-and-drop interactions by specifying source and target coordinates or CSS selectors. Simulates mouse press, move, and release."),
		mcp.WithNumber("startX", mcp.Description("Source X coordinate in CSS pixels")),
		mcp.WithNumber("startY", mcp.Description("Source Y coordinate in CSS pixels")),
		mcp.WithNumber("endX", mcp.Description("Target X coordinate in CSS pixels")),
		mcp.WithNumber("endY", mcp.Description("Target Y coordinate in CSS pixels")),
		mcp.WithString("sourceSelector", mcp.Description("CSS selector for the source element (alternative to startX/startY)")),
		mcp.WithString("targetSelector", mcp.Description("CSS selector for the target element (alternative to endX/endY)")),
		mcp.WithNumber("steps", mcp.Description("Number of intermediate mouse move steps (default: 10)")),
	)
)

// resolvePosition resolves X/Y coordinates from either a CSS selector or explicit coordinate args.
func resolvePosition(page *rod.Page, args map[string]interface{}, selectorKey, xKey, yKey string) (float64, float64, error) {
	selector := getOptionalStringArg(args, selectorKey)
	if selector != "" {
		el, err := page.Element(selector)
		if err != nil {
			return 0, 0, fmt.Errorf("find element %q: %w", selector, err)
		}
		shape, err := el.Shape()
		if err != nil {
			return 0, 0, fmt.Errorf("get element bounds for %q: %w", selector, err)
		}
		rect := shape.Box()
		if rect == nil {
			return 0, 0, fmt.Errorf("element %q has no visible bounds", selector)
		}
		return rect.X + rect.Width/2, rect.Y + rect.Height/2, nil
	}

	x, err := getFloatArg(args, xKey)
	if err != nil {
		return 0, 0, fmt.Errorf("%s or %s required", xKey, selectorKey)
	}
	y, err := getFloatArg(args, yKey)
	if err != nil {
		return 0, 0, fmt.Errorf("%s or %s required", yKey, selectorKey)
	}
	return x, y, nil
}

var (
	DragHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("drag", err)
			}

			args := request.GetArguments()
			steps := int(getOptionalFloatArg(args, "steps", 10))
			if steps < 1 {
				steps = 1
			}

			startX, startY, err := resolvePosition(page, args, "sourceSelector", "startX", "startY")
			if err != nil {
				return toolErr("drag source", err)
			}

			endX, endY, err := resolvePosition(page, args, "targetSelector", "endX", "endY")
			if err != nil {
				return toolErr("drag target", err)
			}

			mouse := page.Mouse

			// Move to start position
			if err := mouse.MoveTo(proto.NewPoint(startX, startY)); err != nil {
				return toolErr("move to start", err)
			}

			// Press at start
			if err := mouse.Down(proto.InputMouseButtonLeft, 1); err != nil {
				return toolErr("mouse press", err)
			}

			// Ensure mouse is released even if the linear move fails
			mouseReleased := false
			defer func() {
				if !mouseReleased {
					_ = mouse.Up(proto.InputMouseButtonLeft, 1)
				}
			}()

			// Move to end in steps using rod's built-in linear interpolation
			if err := mouse.MoveLinear(proto.NewPoint(endX, endY), steps); err != nil {
				return toolErr("mouse move", err)
			}

			// Release at end
			if err := mouse.Up(proto.InputMouseButtonLeft, 1); err != nil {
				return toolErr("mouse release", err)
			}
			mouseReleased = true

			return mcp.NewToolResultText(fmt.Sprintf("Dragged from (%.0f, %.0f) to (%.0f, %.0f) in %d steps", startX, startY, endX, endY, steps)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)
