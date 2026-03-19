package tools

import (
	"context"
	"fmt"

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

			if err = page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
				return toolErr(fmt.Sprintf("vision click move to (%.0f, %.0f)", x, y), err)
			}
			if err = page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return toolErr(fmt.Sprintf("vision click at (%.0f, %.0f)", x, y), err)
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

			// Click to focus the input
			if err = page.Mouse.MoveTo(proto.Point{X: x, Y: y}); err != nil {
				return toolErr(fmt.Sprintf("vision fill move to (%.0f, %.0f)", x, y), err)
			}
			if err = page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return toolErr(fmt.Sprintf("vision fill click at (%.0f, %.0f)", x, y), err)
			}

			// Type the text
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

var (
	VisionToolHandlers = map[string]ToolHandler{
		VisionClickToolKey: VisionClickHandler,
		VisionFillToolKey:  VisionFillHandler,
	}
	VisionToolDefs = []mcp.Tool{
		VisionClick,
		VisionFill,
	}
)
