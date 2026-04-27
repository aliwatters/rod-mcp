package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
			// Also handles async variants — without wrapping, "async () => ..." evaluates
			// to a function reference and serializes as "{}" via ReturnByValue.
			expression := wrapFunctionExpression(script)
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
			// Check for JavaScript exceptions first.
			if r.ExceptionDetails != nil {
				exMsg := r.ExceptionDetails.Text
				if r.ExceptionDetails.Exception != nil && r.ExceptionDetails.Exception.Description != "" {
					exMsg = r.ExceptionDetails.Exception.Description
				}
				return toolErr("evaluate code", fmt.Errorf("JavaScript error: %s", exMsg))
			}
			// Format the result based on its type to avoid returning "<nil>" for
			// string values. gson.JSON.String() returns "<nil>" when the underlying
			// value is nil, and .Str() returns the raw string (unquoted) but returns
			// "" for non-strings. We treat "undefined" explicitly, return unquoted
			// strings via .Str(), and use JSON("","") for other types so numbers/bools
			// are JSON-serialized and null becomes "null".
			result := r.Result
			var resultStr string
			// Handle special unserializable values (NaN, Infinity, -0) first.
			if result.UnserializableValue != "" {
				resultStr = string(result.UnserializableValue)
			} else {
				switch result.Type {
				case "undefined":
					resultStr = "undefined"
				case "string":
					resultStr = result.Value.Str()
				default:
					if result.Value.Nil() {
						if result.Description != "" {
							resultStr = result.Description
						} else {
							resultStr = "null"
						}
					} else {
						resultStr = result.Value.JSON("", "")
					}
				}
			}
			return mcp.NewToolResultText(fmt.Sprintf("Evaluate code successfully with result: %s", resultStr)), nil
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
			case <-time.After(defaultDialogTimeout):
				return mcp.NewToolResultText("No dialog appeared within timeout"), nil
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// wrapFunctionExpression wraps a bare function expression so the CDP
// Runtime.evaluate call invokes it instead of returning the function reference.
// Without this, scripts like "() => ..." or "async () => ..." evaluate to a
// function value which serializes as "{}" under ReturnByValue.
//
// Recognised forms (after trimming leading whitespace):
//   - "(...)" or "(...) => ..."
//   - "function ..." / "function (...) ..."
//   - "async () => ..." / "async function ..."
//
// Anything else (a statement, a bare identifier, an already-invoked expression)
// is returned unchanged.
func wrapFunctionExpression(script string) string {
	trimmed := strings.TrimLeft(script, " \t\n\r")
	if trimmed == "" {
		return script
	}
	isFunc := false
	switch {
	case trimmed[0] == '(':
		isFunc = true
	case strings.HasPrefix(trimmed, "function") &&
		(len(trimmed) == len("function") || !isIdentChar(trimmed[len("function")])):
		isFunc = true
	case strings.HasPrefix(trimmed, "async") &&
		len(trimmed) > len("async") && !isIdentChar(trimmed[len("async")]):
		// "async (" or "async function" or "async\t..." — anything where the
		// next non-identifier char starts a function form.
		isFunc = true
	}
	if !isFunc {
		return script
	}
	return "(" + script + ")()"
}

// isIdentChar reports whether c can appear inside a JS identifier.
// Used to ensure we match the "function" / "async" keyword and not a longer
// identifier like "functionFoo" or "asyncify".
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '$'
}
