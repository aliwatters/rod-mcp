package tools

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	VisionClickToolKey = "rod_vision_click"
	VisionFillToolKey  = "rod_vision_fill"
)

var (
	VisionClick = mcp.NewTool(VisionClickToolKey,
		mcp.WithDescription("Click at x,y coordinates on the page (use with screenshots to identify positions)"),
		mcp.WithNumber("x", mcp.Description("X coordinate in pixels"), mcp.Required()),
		mcp.WithNumber("y", mcp.Description("Y coordinate in pixels"), mcp.Required()),
	)

	VisionFill = mcp.NewTool(VisionFillToolKey,
		mcp.WithDescription("Click at x,y coordinates and type text (use with screenshots to identify input positions)"),
		mcp.WithNumber("x", mcp.Description("X coordinate in pixels"), mcp.Required()),
		mcp.WithNumber("y", mcp.Description("Y coordinate in pixels"), mcp.Required()),
		mcp.WithString("text", mcp.Description("Text to type after clicking"), mcp.Required()),
		mcp.WithBoolean("submit", mcp.Description("Whether to press Enter after typing (default: false)")),
	)
)

// visionClickAt moves the mouse to (x, y) and clicks. Shared by click and fill handlers.
func visionClickAt(page *rod.Page, x, y float64, op string) error {
	if err := page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
		return fmt.Errorf("%s move to (%.0f, %.0f): %w", op, x, y, err)
	}
	if err := page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("%s click at (%.0f, %.0f): %w", op, x, y, err)
	}
	return nil
}

var (
	VisionClickHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("vision click", err)
			}

			x, err := getFloatArg(request.GetArguments(), "x")
			if err != nil {
				return toolErr("vision click", err)
			}
			y, err := getFloatArg(request.GetArguments(), "y")
			if err != nil {
				return toolErr("vision click", err)
			}

			if err = visionClickAt(page, x, y, "vision click"); err != nil {
				return toolErr("vision click", err)
			}

			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Clicked at (%.0f, %.0f) successfully", x, y)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	VisionFillHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("vision fill", err)
			}

			x, err := getFloatArg(request.GetArguments(), "x")
			if err != nil {
				return toolErr("vision fill", err)
			}
			y, err := getFloatArg(request.GetArguments(), "y")
			if err != nil {
				return toolErr("vision fill", err)
			}
			text, err := getStringArg(request.GetArguments(), "text")
			if err != nil {
				return toolErr("vision fill", err)
			}

			if err = visionClickAt(page, x, y, "vision fill"); err != nil {
				return toolErr("vision fill", err)
			}

			if err = page.InsertText(text); err != nil {
				return toolErr("vision fill type text", err)
			}

			if submit, ok := request.GetArguments()["submit"].(bool); ok && submit {
				if err = page.Keyboard.Press(input.Enter); err != nil {
					return toolErr("vision fill submit", err)
				}
			}

			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Filled text at (%.0f, %.0f) successfully", x, y)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// VisionToolRegistrations is the single source of truth for vision-mode tool
// definitions and their handlers.
var VisionToolRegistrations = Registrations{
	{VisionClick, VisionClickHandler},
	{VisionFill, VisionFillHandler},
}

// Derived slices and maps kept for backward compatibility.
var (
	VisionToolHandlers = VisionToolRegistrations.Handlers()
	VisionToolDefs     = VisionToolRegistrations.Tools()
)
