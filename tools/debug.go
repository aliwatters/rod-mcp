package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/types/js"
)

const (
	WaitForToolKey         = "rod_wait_for"
	ConsoleMessagesToolKey = "rod_console_messages"
	NetworkRequestsToolKey = "rod_network_requests"
)

var (
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
	NetworkRequests = mcp.NewTool(NetworkRequestsToolKey,
		mcp.WithDescription("Return captured network requests with method, URL, status code, and resource type"),
		mcp.WithString("url", mcp.Description("Filter requests by URL substring")),
		mcp.WithString("method", mcp.Description("Filter by HTTP method: GET, POST, etc.")),
		mcp.WithBoolean("clear", mcp.Description("Clear captured requests after returning (default: false)")),
	)
)

var (
	WaitForHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("wait for condition", err)
			}

			selector, _ := request.GetArguments()["selector"].(string)
			text, _ := request.GetArguments()["text"].(string)
			state, _ := request.GetArguments()["state"].(string)

			if selector == "" && text == "" {
				return nil, errors.New("either 'selector' or 'text' is required")
			}

			if state == "" {
				state = "visible"
			}

			timeout := defaultWaitTimeoutMs
			if t, ok := request.GetArguments()["timeout"].(float64); ok && t > 0 {
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
			case "visible", "hidden":
				el, err := timedPage.Element(selector)
				if err != nil {
					return toolErr(fmt.Sprintf("wait for %q", selector), err)
				}
				if state == "visible" {
					err = el.WaitVisible()
				} else {
					err = el.WaitInvisible()
				}
				if err != nil {
					return toolErr(fmt.Sprintf("wait for %q to be %s", selector, state), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Element %q is %s", selector, state)), nil

			case "attached":
				if _, err := timedPage.Element(selector); err != nil {
					return toolErr(fmt.Sprintf("wait for %q to be attached", selector), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Element %q is attached to DOM", selector)), nil

			case "detached":
				script := strings.ReplaceAll(js.WaitDetachedJS, "%SELECTOR%", fmt.Sprintf("%q", selector))
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
			filterLevel, _ := request.GetArguments()["level"].(string)
			clear, _ := request.GetArguments()["clear"].(bool)

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

	NetworkRequestsHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filterURL, _ := request.GetArguments()["url"].(string)
			filterMethod, _ := request.GetArguments()["method"].(string)
			clear, _ := request.GetArguments()["clear"].(bool)

			requests := rodCtx.NetworkRequests(filterURL, filterMethod, clear)
			if len(requests) == 0 {
				return mcp.NewToolResultText("No network requests captured"), nil
			}

			var result string
			for _, req := range requests {
				result += fmt.Sprintf("%s %s → %d (%s)\n", req.Method, req.URL, req.Status, req.Type)
			}
			result += fmt.Sprintf("\nTotal: %d requests", len(requests))
			return mcp.NewToolResultText(result), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
