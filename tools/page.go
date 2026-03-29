package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	PageInfoToolKey   = "rod_page_info"
	PageStatusToolKey = "rod_page_status"
)

var (
	PageInfo = mcp.NewTool(PageInfoToolKey,
		mcp.WithDescription("Return the current page URL, title, and loading state"),
	)
	PageStatus = mcp.NewTool(PageStatusToolKey,
		mcp.WithDescription("Quick page health check: URL, title, console errors, network failures, DOM readiness, and viewport size in one call"),
		mcp.WithBoolean("include_console", mcp.Description("Include console error and warning messages (default: true)")),
		mcp.WithBoolean("include_network", mcp.Description("Include failed network request details (default: false)")),
	)
)

var (
	PageInfoHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("page info", err)
			}

			info, err := page.Info()
			if err != nil {
				return toolErr("get page info", err)
			}

			result := map[string]interface{}{
				"url":   info.URL,
				"title": info.Title,
			}

			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return toolErr("marshal page info", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	PageStatusHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("page status", err)
			}

			args := request.GetArguments()
			includeConsole := getOptionalBoolArg(args, "include_console", true)
			includeNetwork := getOptionalBoolArg(args, "include_network", false)

			info, err := page.Info()
			if err != nil {
				return toolErr("get page info", err)
			}

			// Get DOM readiness and viewport via JS evaluation
			evalResult, err := page.Eval(`() => ({
				readyState: document.readyState,
				documentHeight: Math.max(document.body?.scrollHeight || 0, document.documentElement?.scrollHeight || 0),
				viewport: { width: window.innerWidth, height: window.innerHeight }
			})`)
			if err != nil {
				return toolErr("evaluate page status", err)
			}

			var domInfo struct {
				ReadyState     string                 `json:"readyState"`
				DocumentHeight int                    `json:"documentHeight"`
				Viewport       map[string]interface{} `json:"viewport"`
			}
			if err := evalResult.Value.Unmarshal(&domInfo); err != nil {
				return toolErr("parse page status", err)
			}

			result := map[string]interface{}{
				"url":             info.URL,
				"title":           info.Title,
				"dom_ready":       domInfo.ReadyState == "complete" || domInfo.ReadyState == "interactive",
				"ready_state":     domInfo.ReadyState,
				"document_height": domInfo.DocumentHeight,
				"viewport":        domInfo.Viewport,
			}

			if includeConsole {
				errors := rodCtx.ConsoleMessages("error", false)
				warnings := rodCtx.ConsoleMessages("warning", false)
				consoleErrors := make([]string, 0, len(errors))
				for _, e := range errors {
					consoleErrors = append(consoleErrors, e.Text)
				}
				consoleWarnings := make([]string, 0, len(warnings))
				for _, w := range warnings {
					consoleWarnings = append(consoleWarnings, w.Text)
				}
				// Limit to last 10 for readability
				if len(consoleErrors) > 10 {
					consoleErrors = consoleErrors[len(consoleErrors)-10:]
				}
				if len(consoleWarnings) > 10 {
					consoleWarnings = consoleWarnings[len(consoleWarnings)-10:]
				}
				result["console_errors"] = consoleErrors
				result["console_warnings"] = consoleWarnings
			}

			if includeNetwork {
				allRequests := rodCtx.NetworkRequests("", "", false)
				var failed []map[string]interface{}
				pending := 0
				for _, req := range allRequests {
					if req.Status == 0 {
						pending++
					} else if req.Status >= 400 {
						failed = append(failed, map[string]interface{}{
							"method": req.Method,
							"url":    req.URL,
							"status": req.Status,
						})
					}
				}
				result["network_errors"] = len(failed)
				result["network_pending"] = pending
				if len(failed) > 10 {
					failed = failed[len(failed)-10:]
				}
				result["failed_requests"] = failed
			}

			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return toolErr("marshal page status", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// pageContext returns a concise summary of the current page for error messages.
// Used to enrich element resolution failures with actionable context.
func pageContext(rodCtx *types.Context) string {
	page, err := rodCtx.ControlledPage()
	if err != nil {
		return ""
	}
	info, err := page.Info()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("page %q (%s)", info.Title, info.URL)
}
