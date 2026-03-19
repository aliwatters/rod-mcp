package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	PermissionsToolKey = "rod_permissions"
)

var (
	Permissions = mcp.NewTool(PermissionsToolKey,
		mcp.WithDescription("Grant or reset browser permissions (geolocation, notifications, camera, microphone, clipboard, etc.)"),
		mcp.WithString("action", mcp.Description("Action to perform"), mcp.Required(), mcp.Enum("grant", "reset")),
		mcp.WithArray("permissions", mcp.Description("Permissions to grant: geolocation, notifications, audioCapture, videoCapture, clipboardReadWrite, midi, sensors, idleDetection, backgroundSync, etc."),
			mcp.Items(map[string]interface{}{"type": "string"})),
		mcp.WithString("origin", mcp.Description("Origin to apply permissions to (e.g. https://example.com). Applies to all origins if not specified.")),
	)
)

var (
	PermissionsHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			browser, err := rodCtx.ControlledBrowser()
			if err != nil {
				return toolErr("permissions", err)
			}

			action, err := getStringArg(request.GetArguments(), "action")
			if err != nil {
				return toolErr("permissions", err)
			}
			origin := getOptionalStringArg(request.GetArguments(), "origin")

			switch action {
			case "grant":
				perms, err := utils.OptionalStringArrayParam(request, "permissions")
				if err != nil {
					return toolErr("permissions grant", err)
				}
				if len(perms) == 0 {
					return nil, fmt.Errorf("permissions array is required for grant action")
				}
				permTypes := make([]proto.BrowserPermissionType, len(perms))
				for i, p := range perms {
					permTypes[i] = proto.BrowserPermissionType(p)
				}
				err = proto.BrowserGrantPermissions{
					Permissions: permTypes,
					Origin:      origin,
				}.Call(browser)
				if err != nil {
					return toolErr("grant permissions", err)
				}
				scope := "all origins"
				if origin != "" {
					scope = origin
				}
				return mcp.NewToolResultText(fmt.Sprintf("Granted permissions [%s] for %s", strings.Join(perms, ", "), scope)), nil

			case "reset":
				err := proto.BrowserResetPermissions{}.Call(browser)
				if err != nil {
					return toolErr("reset permissions", err)
				}
				return mcp.NewToolResultText("All permissions reset to defaults"), nil

			default:
				return nil, fmt.Errorf("invalid action %q: must be grant or reset", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
