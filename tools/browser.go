package tools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	EvaluateToolKey     = "rod_evaluate"
	CloseBrowserToolKey = "rod_close_browser"
	SetHeadersToolKey   = "rod_set_headers"
	ResizeToolKey       = "rod_resize"
	HandleDialogToolKey = "rod_handle_dialog"
)

var (
	Evaluate = mcp.NewTool(EvaluateToolKey,
		mcp.WithDescription("Execute JavaScript in the browser console"),
		mcp.WithString("script", mcp.Description("A function name or an unnamed function definition"), mcp.Required()),
	)
	CloseBrowser = mcp.NewTool(CloseBrowserToolKey,
		mcp.WithDescription("Close the browser"),
	)
	SetHeaders = mcp.NewTool(SetHeadersToolKey,
		mcp.WithDescription("Set extra HTTP headers for all requests. Useful for authentication, bypassing Cloudflare/Vercel protection, or custom headers."),
		mcp.WithObject("headers", mcp.Description("Headers as key-value pairs, e.g. {\"Authorization\": \"Bearer token\", \"X-Custom-Header\": \"value\"}"), mcp.Required()),
	)
	Resize = mcp.NewTool(ResizeToolKey,
		mcp.WithDescription("Set browser viewport dimensions and emulate device characteristics. Supports full device emulation with user agent, touch, and DPR override."),
		mcp.WithNumber("width", mcp.Description("Viewport width in pixels"), mcp.Required()),
		mcp.WithNumber("height", mcp.Description("Viewport height in pixels"), mcp.Required()),
		mcp.WithNumber("device_scale_factor", mcp.Description("Device pixel ratio (default: 1)")),
		mcp.WithBoolean("is_mobile", mcp.Description("Emulate mobile viewport (default: false)")),
		mcp.WithString("user_agent", mcp.Description("Override the browser User-Agent string")),
		mcp.WithBoolean("has_touch", mcp.Description("Enable touch event emulation (default: false)")),
	)
	HandleDialog = mcp.NewTool(HandleDialogToolKey,
		mcp.WithDescription("Handle a JavaScript dialog (alert, confirm, prompt). Call BEFORE triggering the dialog — sets up a listener that auto-handles the next dialog that appears. Returns dialog info once handled."),
		mcp.WithString("action", mcp.Description("accept or dismiss"), mcp.Required()),
		mcp.WithString("text", mcp.Description("Text to enter for prompt() dialogs before accepting")),
	)
)

var (
	EvaluateHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("evaluate", err)
			}
			script, err := getStringArg(request.GetArguments(), "script")
			if err != nil {
				return toolErr("evaluate", err)
			}
			// Wrap function expressions so they get invoked, not just defined.
			// e.g. "() => document.title" becomes "(() => document.title)()"
			expression := script
			if len(script) > 0 && (script[0] == '(' || len(script) > 8 && script[:8] == "function") {
				expression = "(" + script + ")()"
			}
			r, err := proto.RuntimeEvaluate{
				Expression:            expression,
				ObjectGroup:           "console",
				IncludeCommandLineAPI: true,
				ReturnByValue:         true,
				AwaitPromise:          true,
			}.Call(page)
			if err != nil {
				return toolErr("evaluate code", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Evaluate code successfully with result: %s", r.Result.Value.String())), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	CloseBrowserHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := rodCtx.CloseBrowser(); err != nil {
				return toolErr("close browser", err)
			}
			return mcp.NewToolResultText("Close browser successfully"), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	SetHeadersHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.EnsurePage()
			if err != nil {
				return toolErr("set headers", err)
			}
			headersArg := request.GetArguments()["headers"]
			headersMap, ok := headersArg.(map[string]interface{})
			if !ok {
				return nil, errors.New("headers must be an object with key-value pairs")
			}
			headers := make([]string, 0, len(headersMap)*2)
			for k, v := range headersMap {
				headers = append(headers, k, fmt.Sprintf("%v", v))
			}
			if _, err = page.SetExtraHeaders(headers); err != nil {
				return toolErr("set headers", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Set %d headers successfully", len(headersMap))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	ResizeHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.EnsurePage()
			if err != nil {
				return toolErr("resize viewport", err)
			}
			widthF, err := getFloatArg(request.GetArguments(), "width")
			if err != nil {
				return toolErr("resize viewport", err)
			}
			heightF, err := getFloatArg(request.GetArguments(), "height")
			if err != nil {
				return toolErr("resize viewport", err)
			}
			width := int(widthF)
			height := int(heightF)

			deviceScaleFactor := getOptionalFloatArg(request.GetArguments(), "device_scale_factor", 1.0)
			mobile := getOptionalBoolArg(request.GetArguments(), "is_mobile", false)

			err = proto.EmulationSetDeviceMetricsOverride{
				Width:             width,
				Height:            height,
				DeviceScaleFactor: deviceScaleFactor,
				Mobile:            mobile,
			}.Call(page)
			if err != nil {
				return toolErr("resize viewport", err)
			}

			userAgent := getOptionalStringArg(request.GetArguments(), "user_agent")
			if userAgent != "" {
				if err := (proto.EmulationSetUserAgentOverride{
					UserAgent: userAgent,
				}).Call(page); err != nil {
					return toolErr("set user agent", err)
				}
			}

			hasTouch := getOptionalBoolArg(request.GetArguments(), "has_touch", false)
			if err := (proto.EmulationSetTouchEmulationEnabled{
				Enabled: hasTouch,
			}).Call(page); err != nil {
				return toolErr("set touch emulation", err)
			}

			msg := fmt.Sprintf("Viewport resized to %dx%d (scale: %.1f, mobile: %t", width, height, deviceScaleFactor, mobile)
			if userAgent != "" {
				msg += ", ua: custom"
			}
			if hasTouch {
				msg += ", touch: on"
			}
			msg += ")"
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	HandleDialogHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("handle dialog", err)
			}

			action, err := getStringArg(request.GetArguments(), "action")
			if err != nil {
				return toolErr("handle dialog", err)
			}
			if action != "accept" && action != "dismiss" {
				return nil, errors.New("action must be 'accept' or 'dismiss'")
			}

			accept := action == "accept"
			promptText := ""
			if t, ok := request.GetArguments()["text"].(string); ok {
				promptText = t
			}

			// Use rod's HandleDialog which registers a Page.javascriptDialogOpening
			// listener. This blocks until a dialog appears, then handles it.
			wait, handle := page.HandleDialog()

			// Wait for the dialog in a goroutine with a timeout.
			type dialogResult struct {
				evt *proto.PageJavascriptDialogOpening
			}
			ch := make(chan dialogResult, 1)
			go func() {
				evt := wait()
				ch <- dialogResult{evt: evt}
			}()

			select {
			case dr := <-ch:
				err = handle(&proto.PageHandleJavaScriptDialog{
					Accept:     accept,
					PromptText: promptText,
				})
				if err != nil {
					return toolErr("handle dialog", err)
				}
				msg := fmt.Sprintf("Dialog %sed successfully (type: %s, message: %q)",
					action, dr.evt.Type, dr.evt.Message)
				return mcp.NewToolResultText(msg), nil
			case <-time.After(30 * time.Second):
				return mcp.NewToolResultText("No dialog appeared within timeout"), nil
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
