package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	GeolocationToolKey = "rod_geolocation"
)

var (
	Geolocation = mcp.NewTool(GeolocationToolKey,
		mcp.WithDescription("Set or clear the browser's geolocation override. Automatically grants geolocation permission. Call with no arguments to clear the override."),
		mcp.WithNumber("latitude", mcp.Description("Mock latitude (-90 to 90)")),
		mcp.WithNumber("longitude", mcp.Description("Mock longitude (-180 to 180)")),
		mcp.WithNumber("accuracy", mcp.Description("Mock accuracy in meters (default: 1)")),
	)
)

var (
	GeolocationHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("geolocation", err)
			}

			args := request.GetArguments()
			_, hasLat := args["latitude"]
			_, hasLng := args["longitude"]

			// Clear override if no coordinates provided
			if !hasLat && !hasLng {
				clearErr := proto.EmulationSetGeolocationOverride{}.Call(page)
				if clearErr != nil {
					return toolErr("clear geolocation", clearErr)
				}
				return mcp.NewToolResultText("Geolocation override cleared"), nil
			}

			if !hasLat || !hasLng {
				return toolErr("geolocation", fmt.Errorf("both 'latitude' and 'longitude' are required"))
			}

			lat := args["latitude"].(float64)
			lng := args["longitude"].(float64)
			accuracy := getOptionalFloatArg(args, "accuracy", 1)

			if lat < -90 || lat > 90 {
				return toolErr("geolocation", fmt.Errorf("latitude must be between -90 and 90, got %f", lat))
			}
			if lng < -180 || lng > 180 {
				return toolErr("geolocation", fmt.Errorf("longitude must be between -180 and 180, got %f", lng))
			}

			// Auto-grant geolocation permission (requires browser-level access)
			browser, err := rodCtx.ControlledBrowser()
			if err != nil {
				return toolErr("geolocation", err)
			}
			grantErr := proto.BrowserGrantPermissions{
				Permissions: []proto.BrowserPermissionType{"geolocation"},
			}.Call(browser)
			if grantErr != nil {
				return toolErr("grant geolocation permission", grantErr)
			}

			// Set geolocation override on the page
			setErr := proto.EmulationSetGeolocationOverride{
				Latitude:  &lat,
				Longitude: &lng,
				Accuracy:  &accuracy,
			}.Call(page)
			if setErr != nil {
				return toolErr("set geolocation", setErr)
			}

			out, _ := json.MarshalIndent(map[string]interface{}{
				"latitude":  lat,
				"longitude": lng,
				"accuracy":  accuracy,
			}, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Geolocation set:\n%s", string(out))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
