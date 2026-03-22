package tools

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	NavigationToolKey = "rod_navigate"
	GoBackToolKey     = "rod_go_back"
	GoForwardToolKey  = "rod_go_forward"
	ReloadToolKey     = "rod_reload"
)

var (
	Navigation = mcp.NewTool(NavigationToolKey,
		mcp.WithDescription("Navigate to a URL"),
		mcp.WithString("url", mcp.Description("URL to navigate to"), mcp.Required()),
	)
	GoBack = mcp.NewTool(GoBackToolKey,
		mcp.WithDescription("Go back in the browser history, go back to the previous page"),
	)
	GoForward = mcp.NewTool(GoForwardToolKey,
		mcp.WithDescription("Go forward in the browser history, go to the next page"),
	)
	Reload = mcp.NewTool(ReloadToolKey,
		mcp.WithDescription("Reload the current page"),
	)
)

// simplePageAction creates a handler for simple page navigation actions
// (go back, go forward, reload) that share the same pattern.
func simplePageAction(rodCtx *types.Context, name string, action func(*rod.Page) error) server.ToolHandlerFunc {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Navigation changes the page; invalidate so Execute rebuilds the snapshot.
		rodCtx.InvalidateSnapshot()
		page, err := rodCtx.ControlledPage()
		if err != nil {
			return toolErr(name, err)
		}
		if err = action(page); err != nil {
			return toolErr(name, err)
		}
		waitDOMStable(page)
		return mcp.NewToolResultText(fmt.Sprintf("%s successfully", name)), nil
	}
	return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
}

var (
	NavigationHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Navigation changes the page; invalidate so Execute rebuilds the snapshot.
			rodCtx.InvalidateSnapshot()
			url, err := getStringArg(request.GetArguments(), "url")
			if err != nil {
				return toolErr("navigate", err)
			}
			if !utils.IsHttp(url) {
				return nil, toolError("navigate", fmt.Errorf("invalid URL: %s", url))
			}

			page, err := rodCtx.EnsurePage()
			if err != nil {
				return toolErr("navigate to "+url, err)
			}

			// Update headers for the target URL (applies domain-specific headers from config)
			if err := rodCtx.UpdateHeadersForURL(url); err != nil {
				log.Warnf("Failed to update headers for %s: %s", url, err)
			}

			if err = page.Navigate(url); err != nil {
				return toolErr("navigate to "+url, err)
			}
			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", url)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	GoBackHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		return simplePageAction(rodCtx, "Go back", (*rod.Page).NavigateBack)
	}

	GoForwardHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		return simplePageAction(rodCtx, "Go forward", (*rod.Page).NavigateForward)
	}

	ReloadHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		return simplePageAction(rodCtx, "Reload current page", (*rod.Page).Reload)
	}
)
