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

// toolErr creates a nil result + toolError pair for handler return statements.
func toolErr(action string, err error) (*mcp.CallToolResult, error) {
	return nil, toolError(action, err)
}

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
	SetHeadersToolKey    = "rod_set_headers"
	ResizeToolKey        = "rod_resize"
	HandleDialogToolKey  = "rod_handle_dialog"
	TabNewToolKey        = "rod_tab_new"
	TabListToolKey       = "rod_tab_list"
	TabSelectToolKey     = "rod_tab_select"
	TabCloseToolKey      = "rod_tab_close"
	WaitForToolKey          = "rod_wait_for"
	ConsoleMessagesToolKey  = "rod_console_messages"
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
	HandleDialog = mcp.NewTool(HandleDialogToolKey,
		mcp.WithDescription("Handle a JavaScript dialog (alert, confirm, prompt). Use this when a dialog is blocking page interaction."),
		mcp.WithString("action", mcp.Description("accept or dismiss"), mcp.Required()),
		mcp.WithString("text", mcp.Description("Text to enter for prompt() dialogs before accepting")),
	)
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
	WaitFor = mcp.NewTool(WaitForToolKey,
		mcp.WithDescription("Wait for a condition on the page: a CSS selector to appear/disappear, or text to become visible"),
		mcp.WithString("selector", mcp.Description("CSS selector to wait for")),
		mcp.WithString("text", mcp.Description("Text content to wait for on the page")),
		mcp.WithString("state", mcp.Description("Element state to wait for: visible (default), hidden, attached, detached")),
		mcp.WithNumber("timeout", mcp.Description("Max wait time in milliseconds (default: 30000)")),
	)
	ConsoleMessages = mcp.NewTool(ConsoleMessagesToolKey,
		mcp.WithDescription("Return captured browser console messages (log, warn, error, info)"),
		mcp.WithString("level", mcp.Description("Filter by level: log, warn, error, info (returns all if not specified)")),
		mcp.WithBoolean("clear", mcp.Description("Clear captured messages after returning (default: false)")),
	)
)

var (
	NavigationHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url := request.Params.Arguments["url"].(string)
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
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", url)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	GoBackHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("go back", err)
			}
			if err = page.NavigateBack(); err != nil {
				return toolErr("go back", err)
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
				return toolErr("go forward", err)
			}
			if err = page.NavigateForward(); err != nil {
				return toolErr("go forward", err)
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
				return toolErr("reload page", err)
			}
			if err = page.Reload(); err != nil {
				return toolErr("reload page", err)
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
				return toolErr("press key", err)
			}
			keyStr := request.Params.Arguments["key"].(string)
			key, err := parseKey(keyStr)
			if err != nil {
				return toolErr("parse key "+keyStr, err)
			}
			if err = page.Keyboard.Press(key); err != nil {
				return toolErr("press key "+keyStr, err)
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Press key %s successfully", keyStr)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
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
	EvaluateHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("evaluate", err)
			}
			script := request.Params.Arguments["script"].(string)
			r, err := proto.RuntimeEvaluate{
				Expression:            script,
				ObjectGroup:           "console",
				IncludeCommandLineAPI: true,
			}.Call(page)
			if err != nil {
				return toolErr("evaluate code", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Evaluate code successfully with result: %s", r.Result.Value.String())), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	ScreenshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("screenshot", err)
			}
			req := &proto.PageCaptureScreenshot{
				Format: proto.PageCaptureScreenshotFormatPng,
			}
			bin, err := page.Screenshot(false, req)
			if err != nil {
				return toolErr("capture screenshot", err)
			}
			name := request.Params.Arguments["name"].(string)

			// Always save to file
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "screenshot", "png")
			if err != nil {
				return toolErr("save screenshot", err)
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
				return toolErr("generate PDF", err)
			}
			reader, err := page.PDF(&proto.PagePrintToPDF{})
			if err != nil {
				return toolErr("generate PDF", err)
			}
			bin, err := io.ReadAll(reader)
			if err != nil {
				return toolErr("read PDF data", err)
			}
			name := request.Params.Arguments["name"].(string)

			// Always save to file, never return inline (PDFs can't be rendered inline by MCP clients)
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "page", "pdf")
			if err != nil {
				return toolErr("save PDF", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("PDF saved: %s (%s)", name, path)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	SetHeadersHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.EnsurePage()
			if err != nil {
				return toolErr("set headers", err)
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
				return toolErr("resize viewport", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Viewport resized to %dx%d (scale: %.1f, mobile: %t)", width, height, deviceScaleFactor, mobile)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	HandleDialogHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("handle dialog", err)
			}

			action := request.Params.Arguments["action"].(string)
			if action != "accept" && action != "dismiss" {
				return nil, errors.New("action must be 'accept' or 'dismiss'")
			}

			accept := action == "accept"
			promptText := ""
			if t, ok := request.Params.Arguments["text"].(string); ok {
				promptText = t
			}

			err = proto.PageHandleJavaScriptDialog{
				Accept:     accept,
				PromptText: promptText,
			}.Call(page)
			if err != nil {
				return toolErr("handle dialog", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Dialog %sed successfully", action)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	TabNewHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url := ""
			if u, ok := request.Params.Arguments["url"].(string); ok {
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
			index := int(request.Params.Arguments["index"].(float64))
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
			index := int(request.Params.Arguments["index"].(float64))
			if err := rodCtx.CloseTab(index); err != nil {
				return toolErr("close tab", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Tab %d closed", index)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	WaitForHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("wait for condition", err)
			}

			selector, _ := request.Params.Arguments["selector"].(string)
			text, _ := request.Params.Arguments["text"].(string)
			state, _ := request.Params.Arguments["state"].(string)

			if selector == "" && text == "" {
				return nil, errors.New("either 'selector' or 'text' is required")
			}

			if state == "" {
				state = "visible"
			}

			timeout := 30000.0
			if t, ok := request.Params.Arguments["timeout"].(float64); ok && t > 0 {
				timeout = t
			}
			timedPage := page.Timeout(time.Duration(timeout) * time.Millisecond)

			if text != "" {
				el, err := timedPage.ElementR("*", text)
				if err != nil {
					return toolErr(fmt.Sprintf("wait for text %q", text), err)
				}
				if err = el.WaitVisible(); err != nil {
					return toolErr(fmt.Sprintf("wait for text %q to be visible", text), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Text %q found and visible", text)), nil
			}

			switch state {
			case "visible":
				el, err := timedPage.Element(selector)
				if err != nil {
					return toolErr(fmt.Sprintf("wait for %q", selector), err)
				}
				if err = el.WaitVisible(); err != nil {
					return toolErr(fmt.Sprintf("wait for %q to be visible", selector), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Element %q is visible", selector)), nil

			case "hidden":
				el, err := timedPage.Element(selector)
				if err != nil {
					return toolErr(fmt.Sprintf("wait for %q", selector), err)
				}
				if err = el.WaitInvisible(); err != nil {
					return toolErr(fmt.Sprintf("wait for %q to be hidden", selector), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Element %q is hidden", selector)), nil

			case "attached":
				if _, err := timedPage.Element(selector); err != nil {
					return toolErr(fmt.Sprintf("wait for %q to be attached", selector), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Element %q is attached to DOM", selector)), nil

			case "detached":
				script := fmt.Sprintf(`() => new Promise((resolve) => {
					const check = () => {
						if (!document.querySelector(%q)) { resolve(true); return; }
						requestAnimationFrame(check);
					};
					check();
				})`, selector)
				if _, err := timedPage.Eval(script); err != nil {
					return toolErr(fmt.Sprintf("wait for %q to be detached", selector), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Element %q is detached from DOM", selector)), nil

			default:
				return nil, fmt.Errorf("invalid state %q: must be visible, hidden, attached, or detached", state)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
	ConsoleMessagesHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filterLevel, _ := request.Params.Arguments["level"].(string)
			clear, _ := request.Params.Arguments["clear"].(bool)

			messages := rodCtx.ConsoleMessages(filterLevel, clear)
			if len(messages) == 0 {
				return mcp.NewToolResultText("No console messages captured"), nil
			}

			var result string
			for _, msg := range messages {
				result += fmt.Sprintf("[%s] %s\n", msg.Level, msg.Text)
			}
			result += fmt.Sprintf("\nTotal: %d messages", len(messages))
			return mcp.NewToolResultText(result), nil
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
		HandleDialog,
		TabNew,
		TabList,
		TabSelect,
		TabClose,
		WaitFor,
		ConsoleMessages,
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
		ResizeToolKey:        ResizeHandler,
		HandleDialogToolKey: HandleDialogHandler,
		TabNewToolKey:       TabNewHandler,
		TabListToolKey:      TabListHandler,
		TabSelectToolKey:    TabSelectHandler,
		TabCloseToolKey:     TabCloseHandler,
		WaitForToolKey:         WaitForHandler,
		ConsoleMessagesToolKey: ConsoleMessagesHandler,
	}
)
