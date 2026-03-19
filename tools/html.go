package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	HTMLToolKey = "rod_html"

	// maxHTMLLength is the maximum number of characters returned to prevent token explosion.
	maxHTMLLength = 100000
)

var (
	HTML = mcp.NewTool(HTMLToolKey,
		mcp.WithDescription("Get the HTML source of the page or a specific element"),
		mcp.WithString("selector", mcp.Description("CSS selector of element to get HTML for (default: entire page)")),
		mcp.WithBoolean("outer", mcp.Description("Return outerHTML (true, default) or innerHTML (false)")),
	)
)

var (
	HTMLHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("get HTML", err)
			}

			args := request.GetArguments()
			selector := getOptionalStringArg(args, "selector")
			outer := getOptionalBoolArg(args, "outer", true)

			var html string
			if selector == "" {
				// Full page HTML
				r, err := page.Eval(`() => document.documentElement.outerHTML`)
				if err != nil {
					return toolErr("get page HTML", err)
				}
				html = r.Value.Str()
			} else {
				// Element HTML
				el, err := page.Element(selector)
				if err != nil {
					return toolErr(fmt.Sprintf("find element %q", selector), err)
				}
				if outer {
					html, err = el.HTML()
				} else {
					r, err2 := el.Eval(`(el) => el.innerHTML`, el)
					if err2 != nil {
						return toolErr(fmt.Sprintf("get innerHTML of %q", selector), err2)
					}
					html = r.Value.Str()
				}
				if err != nil {
					return toolErr(fmt.Sprintf("get HTML of %q", selector), err)
				}
			}

			truncated := false
			if len(html) > maxHTMLLength {
				html = html[:maxHTMLLength]
				truncated = true
			}

			if truncated {
				html += fmt.Sprintf("\n\n... (truncated at %d characters)", maxHTMLLength)
			}

			return mcp.NewToolResultText(html), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
