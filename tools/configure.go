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
	mcp.WithString("user_data_dir", mcp.Description("Chrome profile directory to clone from, or to use directly with no_clone=true. Set this with headless=false for interactive login flows.")),
	mcp.WithString("clone_domains", mcp.Description("Comma-separated domains whose cookies should be cloned from user_data_dir. Empty string clears the domain filter.")),
	mcp.WithBoolean("no_clone", mcp.Description("Use user_data_dir directly instead of cloning it. This preserves login state across sessions but requires Chrome not to be using the same profile.")),
	mcp.WithBoolean("clone_all", mcp.Description("Clone the full profile instead of cookie-only domain cloning. Slower and copies more sensitive browser data.")),
	mcp.WithBoolean("stealth", mcp.Description("[EXPERIMENTAL] Enable stealth mode to bypass bot detection. Removes automation indicators, patches navigator.webdriver, injects realistic browser fingerprints, and sets a realistic User-Agent header.")),
)

var ConfigureHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		headless := getOptionalBoolPtr(args, "headless")
		cdpEndpoint := getOptionalStringPtr(args, "cdp_endpoint")
		userDataDir := getOptionalStringPtr(args, "user_data_dir")
		cloneDomainsArg := getOptionalStringPtr(args, "clone_domains")
		noClone := getOptionalBoolPtr(args, "no_clone")
		cloneAll := getOptionalBoolPtr(args, "clone_all")
		stealth := getOptionalBoolPtr(args, "stealth")

		var cloneDomains *[]string
		if cloneDomainsArg != nil {
			parsed := parseCSV(*cloneDomainsArg)
			cloneDomains = &parsed
		}

		if err := rodCtx.Reconfigure(types.ReconfigureOptions{
			Headless:     headless,
			CDPEndpoint:  cdpEndpoint,
			Stealth:      stealth,
			UserDataDir:  userDataDir,
			CloneDomains: cloneDomains,
			NoClone:      noClone,
			CloneAll:     cloneAll,
		}); err != nil {
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
		if userDataDir != nil {
			if *userDataDir == "" {
				parts = append(parts, "user_data_dir cleared")
			} else {
				parts = append(parts, fmt.Sprintf("user_data_dir=%s", *userDataDir))
			}
		}
		if cloneDomainsArg != nil {
			if *cloneDomainsArg == "" {
				parts = append(parts, "clone_domains cleared")
			} else {
				parts = append(parts, fmt.Sprintf("clone_domains=%s", *cloneDomainsArg))
			}
		}
		if noClone != nil {
			parts = append(parts, fmt.Sprintf("no_clone=%t", *noClone))
		}
		if cloneAll != nil {
			parts = append(parts, fmt.Sprintf("clone_all=%t", *cloneAll))
		}
		if stealth != nil {
			parts = append(parts, fmt.Sprintf("stealth=%t", *stealth))
		}
		if len(parts) == 0 {
			return mcp.NewToolResultText("No configuration changes requested"), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Browser configured: %s. Changes take effect on next browser action.", strings.Join(parts, ", "))), nil
	}
	return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
