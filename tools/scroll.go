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
		mcp.WithString("direction", mcp.Description("Scroll direction: up, down, left, right, top, bottom"), mcp.Enum("up", "down", "left", "right", "top", "bottom")),
		mcp.WithNumber("amount", mcp.Description(fmt.Sprintf("Number of pixels to scroll (default: %d). Ignored for top/bottom.", defaultScrollPixels))),
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
			selector := getOptionalStringArg(args, "selector")

			if selector != "" {
				return scrollToElement(page, selector)
			}

			if _, hasX := args["x"]; hasX {
				return scrollToCoordinates(page, args)
			}
			if _, hasY := args["y"]; hasY {
				return scrollToCoordinates(page, args)
			}

			return scrollByDirection(page, args)
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)
