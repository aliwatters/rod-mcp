package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/types/js"
)

const (
	PerformanceToolKey = "rod_performance"
)

var (
	Performance = mcp.NewTool(PerformanceToolKey,
		mcp.WithDescription("Collect browser performance metrics and Core Web Vitals. Returns CDP Performance domain metrics and optionally web vitals (LCP, CLS, FID, TTFB) via JS evaluation."),
		mcp.WithString("action", mcp.Description("Action to perform"), mcp.Required(), mcp.Enum("metrics", "vitals")),
	)
)

var (
	PerformanceHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("performance", err)
			}

			action, err := getStringArg(request.GetArguments(), "action")
			if err != nil {
				return toolErr("performance", err)
			}

			switch action {
			case "metrics":
				err = proto.PerformanceEnable{}.Call(page)
				if err != nil {
					return toolErr("enable performance", err)
				}
				result, err := proto.PerformanceGetMetrics{}.Call(page)
				if err != nil {
					return toolErr("get performance metrics", err)
				}

				sort.Slice(result.Metrics, func(i, j int) bool {
					return result.Metrics[i].Name < result.Metrics[j].Name
				})

				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Performance metrics (%d):\n", len(result.Metrics)))
				for _, m := range result.Metrics {
					if m.Value == float64(int64(m.Value)) {
						sb.WriteString(fmt.Sprintf("  %s: %d\n", m.Name, int64(m.Value)))
					} else {
						sb.WriteString(fmt.Sprintf("  %s: %.3f\n", m.Name, m.Value))
					}
				}
				return mcp.NewToolResultText(sb.String()), nil

			case "vitals":
				r, err := page.Eval(js.PerformanceVitalsJS)
				if err != nil {
					return toolErr("get web vitals", err)
				}

				return mcp.NewToolResultText(fmt.Sprintf("Web Vitals:\n%s", r.Value.Str())), nil

			default:
				return nil, fmt.Errorf("invalid action %q: must be metrics or vitals", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
