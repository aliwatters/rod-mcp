package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	TabNewToolKey    = "rod_tab_new"
	TabListToolKey   = "rod_tab_list"
	TabSelectToolKey = "rod_tab_select"
	TabCloseToolKey  = "rod_tab_close"
)

var (
	TabNew = mcp.NewTool(TabNewToolKey,
		mcp.WithDescription("Open a new browser tab, optionally navigating to a URL"),
		mcp.WithString("url", mcp.Description("URL to navigate to in the new tab")),
	)
	TabList = mcp.NewTool(TabListToolKey,
		mcp.WithDescription("List all open browser tabs with their titles, URLs, and which is active"),
	)
	TabSelect = mcp.NewTool(TabSelectToolKey,
		mcp.WithDescription("Switch to a specific browser tab by index"),
		mcp.WithNumber("index", mcp.Description("Tab index (from rod_tab_list)"), mcp.Required()),
	)
	TabClose = mcp.NewTool(TabCloseToolKey,
		mcp.WithDescription("Close a browser tab by index"),
		mcp.WithNumber("index", mcp.Description("Tab index to close (from rod_tab_list)"), mcp.Required()),
	)
)

var (
	TabNewHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url := ""
			if u, ok := request.GetArguments()["url"].(string); ok {
				url = u
			}
			if url != "" && !utils.IsHttp(url) {
				return nil, toolError("create new tab", fmt.Errorf("invalid URL: %s", url))
			}
			if _, err := rodCtx.NewTab(url); err != nil {
				return toolErr("create new tab", err)
			}
			if url != "" {
				return mcp.NewToolResultText(fmt.Sprintf("New tab opened and navigated to %s", url)), nil
			}
			return mcp.NewToolResultText("New tab opened"), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	TabListHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tabs, err := rodCtx.ListTabs()
			if err != nil {
				return toolErr("list tabs", err)
			}
			var result string
			for _, tab := range tabs {
				active := ""
				if tab.IsActive {
					active = " (active)"
				}
				result += fmt.Sprintf("[%d] %s - %s%s\n", tab.Index, tab.Title, tab.URL, active)
			}
			return mcp.NewToolResultText(result), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	TabSelectHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			indexF, err := getFloatArg(request.GetArguments(), "index")
			if err != nil {
				return toolErr("select tab", err)
			}
			index := int(indexF)
			page, err := rodCtx.SelectTab(index)
			if err != nil {
				return toolErr("select tab", err)
			}
			info, err := page.Info()
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Switched to tab %d", index)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Switched to tab %d: %s - %s", index, info.Title, info.URL)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	TabCloseHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			indexF, err := getFloatArg(request.GetArguments(), "index")
			if err != nil {
				return toolErr("close tab", err)
			}
			index := int(indexF)
			if err := rodCtx.CloseTab(index); err != nil {
				return toolErr("close tab", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Tab %d closed", index)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
