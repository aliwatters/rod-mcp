package tools

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/go-rod/rod-mcp/types"
	"github.com/go-rod/rod-mcp/utils"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultWaitStableDur = 1 * time.Second
	defaultDomDiff       = 0.2
)

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
		mcp.WithDescription("Generate a PDF from the current page"),
		mcp.WithString("file_path", mcp.Description("Path to save the PDF file"), mcp.Required()),
		mcp.WithString("file_name", mcp.Description("Name of the PDF file"), mcp.Required()),
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	PressKeyHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to press key: %s", err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to press key: %s", err.Error()))
			}
			key := request.Params.Arguments["key"].(rune)
			err = page.Keyboard.Type(input.Key(key))
			if err != nil {
				log.Errorf("Failed to press key %s: %s", string(key), err.Error())
				return nil, errors.New(fmt.Sprintf("Failed to press key %s: %s", string(key), err.Error()))
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Press key %s successfully", string(key))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
	}
	ScreenshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				log.Errorf("Failed to screenshot: %s", err.Error())
			}
			req := &proto.PageCaptureScreenshot{
				Format: proto.PageCaptureScreenshotFormatPng,
			}
			bin, err := page.Screenshot(false, req)
			if err != nil {
				log.Errorf("Failed to screenshot: %s", err.Error())
			}
			fileName := request.Params.Arguments["name"].(string)
			toFile := []string{"tmp", "screenshots", fileName + ".png"}
			filePath := filepath.Join(toFile...)
			err = os.WriteFile(filePath, bin, 0o664)
			if err != nil {
				log.Errorf("Failed to screenshot: %s", err.Error())
			}
			return mcp.NewToolResultText(fmt.Sprintf("Save to %s", filePath)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
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
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
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
		Evaluate,
		CloseBrowser,
		SetHeaders,
	}
	CommonToolHandlers = map[string]ToolHandler{
		NavigationToolKey:   NavigationHandler,
		GoBackToolKey:       GoBackHandler,
		GoForwardToolKey:    GoForwardHandler,
		ReloadToolKey:       ReLoadHandler,
		PressKeyToolKey:     PressKeyHandler,
		ScreenshotToolKey:   ScreenshotHandler,
		EvaluateToolKey:     EvaluateHandler,
		CloseBrowserToolKey: CloseBrowserHandler,
		SetHeadersToolKey:   SetHeadersHandler,
	}
)
