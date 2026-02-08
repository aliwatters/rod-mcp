package tools

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	defaultWaitStableDur = 1 * time.Second
	defaultDomDiff       = 0.2
)

// keyMap maps string key names to input.Key constants.
// Supports both named keys (Tab, Enter, ArrowUp) and single characters.
var keyMap = map[string]input.Key{
	// Function keys
	"Escape": input.Escape,
	"F1":     input.F1,
	"F2":     input.F2,
	"F3":     input.F3,
	"F4":     input.F4,
	"F5":     input.F5,
	"F6":     input.F6,
	"F7":     input.F7,
	"F8":     input.F8,
	"F9":     input.F9,
	"F10":    input.F10,
	"F11":    input.F11,
	"F12":    input.F12,

	// Navigation
	"Backspace":  input.Backspace,
	"Tab":        input.Tab,
	"Enter":      input.Enter,
	"Return":     input.Enter,
	"CapsLock":   input.CapsLock,
	"Delete":     input.Delete,
	"End":        input.End,
	"Home":       input.Home,
	"Insert":     input.Insert,
	"PageDown":   input.PageDown,
	"PageUp":     input.PageUp,
	"ArrowDown":  input.ArrowDown,
	"ArrowLeft":  input.ArrowLeft,
	"ArrowRight": input.ArrowRight,
	"ArrowUp":    input.ArrowUp,

	// Modifiers
	"Alt":        input.AltLeft,
	"AltLeft":    input.AltLeft,
	"AltRight":   input.AltRight,
	"Control":    input.ControlLeft,
	"ControlLeft": input.ControlLeft,
	"ControlRight": input.ControlRight,
	"Meta":       input.MetaLeft,
	"MetaLeft":   input.MetaLeft,
	"MetaRight":  input.MetaRight,
	"Shift":      input.ShiftLeft,
	"ShiftLeft":  input.ShiftLeft,
	"ShiftRight": input.ShiftRight,

	// Special keys
	"Space":       input.Space,
	"PrintScreen": input.PrintScreen,
	"ScrollLock":  input.ScrollLock,
	"Pause":       input.Pause,
	"ContextMenu": input.ContextMenu,
	"NumLock":     input.NumLock,
}

// parseKey converts a key string to an input.Key.
// For single characters, returns the rune as input.Key.
// For named keys (Tab, Enter, ArrowUp, etc.), looks up in keyMap.
func parseKey(keyStr string) (input.Key, error) {
	// Check if it's a named key first
	if key, ok := keyMap[keyStr]; ok {
		return key, nil
	}

	// For single characters, use the rune value directly
	if len(keyStr) == 1 {
		return input.Key(keyStr[0]), nil
	}

	return 0, fmt.Errorf("unknown key: %s", keyStr)
}

const (
	NavigationToolKey   = "rod_navigate"
	GoBackToolKey       = "rod_go_back"
	GoForwardToolKey    = "rod_go_forward"
	ReloadToolKey       = "rod_reload"
	PressKeyToolKey     = "rod_press"
	PdfToolKey          = "rod_pdf"
	ScreenshotToolKey   = "rod_screenshot"
	EvaluateToolKey     = "rod_evaluate"
	CloseBrowserToolKey = "rod_close_browser"
	SetHeadersToolKey   = "rod_set_headers"
	ResizeToolKey       = "rod_resize"
)

var (
	Navigation = mcp.NewTool("rod_navigate",
		mcp.WithDescription("Navigate to a URL"),
		mcp.WithString("url", mcp.Description("URL to navigate to"), mcp.Required()),
	)
	GoBack = mcp.NewTool(GoBackToolKey,
		mcp.WithDescription("Go back in the browser history, go back to the previous page"),
	)
	GoForward = mcp.NewTool(GoForwardToolKey,
		mcp.WithDescription("Go forward in the browser history, go to the next page"),
	)
	ReLoad = mcp.NewTool(ReloadToolKey,
		mcp.WithDescription("Reload the current page"),
	)
	PressKey = mcp.NewTool(PressKeyToolKey,
		mcp.WithDescription("Press a key on the keyboard"),
		mcp.WithString("key", mcp.Description("Name of the key to press or a character to generate, such as `ArrowLeft` or `a`"), mcp.Required()),
	)
	Pdf = mcp.NewTool(PdfToolKey,
		mcp.WithDescription("Generate a PDF of the current page and save to the output directory"),
		mcp.WithString("name", mcp.Description("Name or description of the PDF"), mcp.Required()),
	)
	CloseBrowser = mcp.NewTool(CloseBrowserToolKey,
		mcp.WithDescription("Close the browser"),
	)
	Screenshot = mcp.NewTool(ScreenshotToolKey,
		mcp.WithDescription("Take a screenshot of the current page or a specific element"),
		mcp.WithString("name", mcp.Description("Name of the screenshot"), mcp.Required()),
		mcp.WithString("selector", mcp.Description("CSS selector of the element to take a screenshot of")),
		mcp.WithNumber("width", mcp.Description("Width in pixels (default: 800)")),
		mcp.WithNumber("height", mcp.Description("Height in pixels (default: 600)")),
	)
	Evaluate = mcp.NewTool(EvaluateToolKey,
		mcp.WithDescription("Execute JavaScript in the browser console"),
		mcp.WithString("script", mcp.Description("A function name or an unnamed function definition"), mcp.Required()),
	)
	SetHeaders = mcp.NewTool(SetHeadersToolKey,
		mcp.WithDescription("Set extra HTTP headers for all requests. Useful for authentication, bypassing Cloudflare/Vercel protection, or custom headers."),
		mcp.WithObject("headers", mcp.Description("Headers as key-value pairs, e.g. {\"Authorization\": \"Bearer token\", \"X-Custom-Header\": \"value\"}"), mcp.Required()),
	)
	Resize = mcp.NewTool(ResizeToolKey,
		mcp.WithDescription("Set the browser viewport dimensions for responsive testing and layout control"),
		mcp.WithNumber("width", mcp.Description("Viewport width in pixels"), mcp.Required()),
		mcp.WithNumber("height", mcp.Description("Viewport height in pixels"), mcp.Required()),
		mcp.WithNumber("device_scale_factor", mcp.Description("Device pixel ratio (default: 1)")),
		mcp.WithBoolean("is_mobile", mcp.Description("Emulate mobile viewport (default: false)")),
	)
)

var (
	NavigationHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url := request.Params.Arguments["url"].(string)
			if !utils.IsHttp(url) {
				log.Errorf("Invalid URL: %s", url)
				return nil, errors.New("invalid URL")
			}

			page, err := rodCtx.EnsurePage()
			if err != nil {
				log.Errorf("Failed to navigate to %s: %s", url, err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to navigate to %s: %s", url, err.Error()))
			}

			// Update headers for the target URL (applies domain-specific headers from config)
			if err := rodCtx.UpdateHeadersForURL(url); err != nil {
				log.Warnf("Failed to update headers for %s: %s", url, err.Error())
			}

			err = page.Navigate(url)
			if err != nil {
				log.Errorf("Failed to navigate to %s: %s", url, err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to navigate to %s: %s", url, err.Error()))
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", url)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	GoBackHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to go back: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to go back: %s", err.Error()))
			}
			err = page.NavigateBack()
			if err != nil {
				log.Errorf("Failed to go back: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to go back: %s", err.Error()))
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText("Go back successfully"), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	GoForwardHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to go forward: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to go forward: %s", err.Error()))
			}
			err = page.NavigateForward()
			if err != nil {
				log.Errorf("Failed to go forward: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to go forward: %s", err.Error()))
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText("Go forward successfully"), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	ReLoadHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to reload current page: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to reload current page: %s", err.Error()))
			}
			err = page.Reload()
			if err != nil {
				log.Errorf("Failed to reload current page: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to reload current page: %s", err.Error()))
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText("Reload current page successfully"), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	PressKeyHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to press key: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to press key: %s", err.Error()))
			}
			keyStr := request.Params.Arguments["key"].(string)
			key, err := parseKey(keyStr)
			if err != nil {
				log.Errorf("Failed to parse key %s: %s", keyStr, err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to parse key %s: %s", keyStr, err.Error()))
			}
			err = page.Keyboard.Press(key)
			if err != nil {
				log.Errorf("Failed to press key %s: %s", keyStr, err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to press key %s: %s", keyStr, err.Error()))
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Press key %s successfully", keyStr)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}
	CloseBrowserHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			err := rodCtx.CloseBrowser()
			if err != nil {
				log.Errorf("Failed to close browser: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to close browser: %s", err.Error()))
			}
			return mcp.NewToolResultText("Close browser successfully"), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	EvaluateHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to evaluate: %s", err.Error())
			}
			script := request.Params.Arguments["script"].(string)
			r, err := proto.RuntimeEvaluate{
				Expression:            script,
				ObjectGroup:           "console",
				IncludeCommandLineAPI: true,
			}.Call(page)
			if err != nil {
				log.Errorf("Failed to evaluate code: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to evaluate code: %s", err.Error()))
			}
			return mcp.NewToolResultText(fmt.Sprintf("Evaluate code successfully with result: %s", r.Result.Value.String())), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	ScreenshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to screenshot: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to screenshot: %s", err.Error()))
			}
			req := &proto.PageCaptureScreenshot{
				Format: proto.PageCaptureScreenshotFormatPng,
			}
			bin, err := page.Screenshot(false, req)
			if err != nil {
				log.Errorf("Failed to screenshot: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to capture screenshot: %s", err.Error()))
			}
			name := request.Params.Arguments["name"].(string)

			// Always save to file
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "screenshot", "png")
			if err != nil {
				log.Errorf("Failed to save screenshot: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to save screenshot: %s", err.Error()))
			}

			// Return file path + optional inline image
			if cfg.ImageResponses != types.ImageResponsesOmit {
				encoded := base64.StdEncoding.EncodeToString(bin)
				return mcp.NewToolResultImage(
					fmt.Sprintf("Screenshot saved: %s (%s)", name, path),
					encoded, "image/png",
				), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Screenshot saved: %s (%s)", name, path)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	PdfHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to generate PDF: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to generate PDF: %s", err.Error()))
			}
			reader, err := page.PDF(&proto.PagePrintToPDF{})
			if err != nil {
				log.Errorf("Failed to generate PDF: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to generate PDF: %s", err.Error()))
			}
			bin, err := io.ReadAll(reader)
			if err != nil {
				log.Errorf("Failed to read PDF data: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to read PDF data: %s", err.Error()))
			}
			name := request.Params.Arguments["name"].(string)

			// Always save to file, never return inline (PDFs can't be rendered inline by MCP clients)
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "page", "pdf")
			if err != nil {
				log.Errorf("Failed to save PDF: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to save PDF: %s", err.Error()))
			}
			return mcp.NewToolResultText(fmt.Sprintf("PDF saved: %s (%s)", name, path)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	SetHeadersHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.EnsurePage()
			if err != nil {
				log.Errorf("Failed to set headers: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to set headers: %s", err.Error()))
			}
			headersArg := request.Params.Arguments["headers"]
			headersMap, ok := headersArg.(map[string]interface{})
			if !ok {
				return nil, errors.New("headers must be an object with key-value pairs")
			}
			headers := make([]string, 0, len(headersMap)*2)
			for k, v := range headersMap {
				headers = append(headers, k, fmt.Sprintf("%v", v))
			}
			_, err = page.SetExtraHeaders(headers)
			if err != nil {
				log.Errorf("Failed to set headers: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to set headers: %s", err.Error()))
			}
			return mcp.NewToolResultText(fmt.Sprintf("Set %d headers successfully", len(headersMap))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	ResizeHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.EnsurePage()
			if err != nil {
				log.Errorf("Failed to resize viewport: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to resize viewport: %s", err.Error()))
			}
			width := int(request.Params.Arguments["width"].(float64))
			height := int(request.Params.Arguments["height"].(float64))

			deviceScaleFactor := 1.0
			if dsf, ok := request.Params.Arguments["device_scale_factor"].(float64); ok {
				deviceScaleFactor = dsf
			}

			mobile := false
			if m, ok := request.Params.Arguments["is_mobile"].(bool); ok {
				mobile = m
			}

			err = proto.EmulationSetDeviceMetricsOverride{
				Width:             width,
				Height:            height,
				DeviceScaleFactor: deviceScaleFactor,
				Mobile:            mobile,
			}.Call(page)
			if err != nil {
				log.Errorf("Failed to resize viewport: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to resize viewport: %s", err.Error()))
			}
			return mcp.NewToolResultText(fmt.Sprintf("Viewport resized to %dx%d (scale: %.1f, mobile: %t)", width, height, deviceScaleFactor, mobile)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

var (
	CommonTools = []mcp.Tool{
		Navigation,
		GoBack,
		GoForward,
		ReLoad,
		PressKey,
		Screenshot,
		Pdf,
		Evaluate,
		CloseBrowser,
		SetHeaders,
		Resize,
	}
	CommonToolHandlers = map[string]ToolHandler{
		NavigationToolKey:   NavigationHandler,
		GoBackToolKey:       GoBackHandler,
		GoForwardToolKey:    GoForwardHandler,
		ReloadToolKey:       ReLoadHandler,
		PressKeyToolKey:     PressKeyHandler,
		ScreenshotToolKey:   ScreenshotHandler,
		PdfToolKey:          PdfHandler,
		EvaluateToolKey:     EvaluateHandler,
		CloseBrowserToolKey: CloseBrowserHandler,
		SetHeadersToolKey:   SetHeadersHandler,
		ResizeToolKey:       ResizeHandler,
	}
)
