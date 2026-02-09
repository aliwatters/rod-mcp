package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const ConfigureToolKey = "rod_configure"

var Configure = mcp.NewTool(ConfigureToolKey,
	mcp.WithDescription("Configure browser settings. If the browser is already running it will be closed and restarted with the new settings on the next tool call."),
	mcp.WithBoolean("headless", mcp.Description("Run the browser in headless mode (true) or with a visible GUI (false)")),
	mcp.WithString("cdp_endpoint", mcp.Description("Connect to an existing Chrome instance via CDP endpoint URL (e.g. http://127.0.0.1:9222)")),
)

var ConfigureHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments

		var headless *bool
		if v, ok := args["headless"].(bool); ok {
			headless = &v
		}

		var cdpEndpoint *string
		if v, ok := args["cdp_endpoint"].(string); ok {
			cdpEndpoint = &v
		}

		if err := rodCtx.Reconfigure(headless, cdpEndpoint); err != nil {
			return toolErr("configure browser", err)
		}

		var parts []string
		if headless != nil {
			parts = append(parts, fmt.Sprintf("headless=%t", *headless))
		}
		if cdpEndpoint != nil {
			if *cdpEndpoint == "" {
				parts = append(parts, "cdp_endpoint cleared")
			} else {
				parts = append(parts, fmt.Sprintf("cdp_endpoint=%s", *cdpEndpoint))
			}
		}
		if len(parts) == 0 {
			return mcp.NewToolResultText("No configuration changes requested"), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Browser configured: %s. Changes take effect on next browser action.", strings.Join(parts, ", "))), nil
	}
	return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
}
