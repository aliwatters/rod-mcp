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
)

var (
	HTML = mcp.NewTool(HTMLToolKey,
		mcp.WithDescription("Get the HTML source of the page or a specific element"),
		mcp.WithString("selector", mcp.Description("CSS selector of element to get HTML for (default: entire page)")),
		mcp.WithBoolean("outer", mcp.Description("Return outerHTML (true, default) or innerHTML (false)")),
		mcp.WithNumber("max_chars", mcp.Description("Optional character limit for the returned HTML. 0 (default) = unlimited (preserves current behaviour). When set to a positive value and the output exceeds the limit, the text is truncated and a truncation notice is appended. Use rod_evaluate for targeted extraction instead of raising this limit.")),
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
				// Element HTML — use timeout so missing selectors return an error instead of hanging.
				el, err := page.Timeout(defaultSelectorTimeout).Element(selector)
				if err != nil {
					return toolErr(fmt.Sprintf("find element %q", selector), err)
				}
				if outer {
					html, err = el.HTML()
					if err != nil {
						return toolErr(fmt.Sprintf("get outerHTML of %q", selector), err)
					}
				} else {
					r, err := el.Eval(`() => this.innerHTML`)
					if err != nil {
						return toolErr(fmt.Sprintf("get innerHTML of %q", selector), err)
					}
					html = r.Value.Str()
				}
			}

			// Apply opt-in max_chars limit (0 = unlimited); fall back to the
			// built-in hard cap to guard against accidental context blow-up.
			maxChars := int(getOptionalFloatArg(args, "max_chars", 0))
			if maxChars <= 0 {
				maxChars = defaultMaxContentLength
			}
			html = applyMaxChars(html, maxChars)

			return mcp.NewToolResultText(html), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
