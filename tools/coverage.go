package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	CoverageToolKey = "rod_coverage"
)

var (
	Coverage = mcp.NewTool(CoverageToolKey,
		mcp.WithDescription("Collect CSS and JavaScript code coverage. Start collection, get a delta report (coverage since last report or since start), or stop collection."),
		mcp.WithString("action", mcp.Description("Action to perform: start, report, stop"), mcp.Required(), mcp.Enum("start", "report", "stop")),
		mcp.WithString("type", mcp.Description("Coverage type: js, css, or all (default: all)"), mcp.Enum("js", "css", "all")),
	)
)

var (
	CoverageHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("coverage", err)
			}

			action, err := getStringArg(request.GetArguments(), "action")
			if err != nil {
				return toolErr("coverage", err)
			}
			coverType := getOptionalStringArg(request.GetArguments(), "type")
			if coverType == "" {
				coverType = "all"
			}

			doJS := coverType == "js" || coverType == "all"
			doCSS := coverType == "css" || coverType == "all"

			switch action {
			case "start":
				var started []string
				if doJS {
					err := proto.ProfilerEnable{}.Call(page)
					if err != nil {
						return toolErr("enable JS profiler", err)
					}
					_, err = proto.ProfilerStartPreciseCoverage{
						CallCount: true,
						Detailed:  true,
					}.Call(page)
					if err != nil {
						return toolErr("start JS coverage", err)
					}
					started = append(started, "JS")
				}
				if doCSS {
					err := proto.DOMEnable{}.Call(page)
					if err != nil {
						return toolErr("enable DOM domain", err)
					}
					err = proto.CSSEnable{}.Call(page)
					if err != nil {
						return toolErr("enable CSS domain", err)
					}
					err = proto.CSSStartRuleUsageTracking{}.Call(page)
					if err != nil {
						return toolErr("start CSS coverage", err)
					}
					started = append(started, "CSS")
				}
				return mcp.NewToolResultText(fmt.Sprintf("Coverage collection started: %s", strings.Join(started, ", "))), nil

			case "report":
				var result strings.Builder

				if doJS {
					resp, err := proto.ProfilerTakePreciseCoverage{}.Call(page)
					if err != nil {
						return toolErr("get JS coverage", err)
					}
					result.WriteString("## JavaScript Coverage\n\n")
					if len(resp.Result) == 0 {
						result.WriteString("No JS coverage data (is collection started?)\n\n")
					} else {
						var totalBytes, usedBytes int
						for _, script := range resp.Result {
							url := script.URL
							if url == "" {
								continue
							}
							scriptTotal := 0
							scriptUsed := 0
							for _, fn := range script.Functions {
								for _, r := range fn.Ranges {
									size := r.EndOffset - r.StartOffset
									scriptTotal += size
									if r.Count > 0 {
										scriptUsed += size
									}
								}
							}
							if scriptTotal > 0 {
								pct := float64(scriptUsed) / float64(scriptTotal) * 100
								result.WriteString(fmt.Sprintf("- %s: %.1f%% (%d/%d bytes)\n", url, pct, scriptUsed, scriptTotal))
								totalBytes += scriptTotal
								usedBytes += scriptUsed
							}
						}
						if totalBytes > 0 {
							pct := float64(usedBytes) / float64(totalBytes) * 100
							result.WriteString(fmt.Sprintf("\nJS Total: %.1f%% (%d/%d bytes)\n\n", pct, usedBytes, totalBytes))
						}
					}
				}

				if doCSS {
					resp, err := proto.CSSTakeCoverageDelta{}.Call(page)
					if err != nil {
						return toolErr("get CSS coverage", err)
					}
					result.WriteString("## CSS Coverage\n\n")
					if len(resp.Coverage) == 0 {
						result.WriteString("No CSS coverage data (is collection started?)\n\n")
					} else {
						// Aggregate by stylesheet
						type sheetStats struct {
							total int
							used  int
						}
						sheets := make(map[string]*sheetStats)
						for _, rule := range resp.Coverage {
							id := string(rule.StyleSheetID)
							s, ok := sheets[id]
							if !ok {
								s = &sheetStats{}
								sheets[id] = s
							}
							size := int(rule.EndOffset - rule.StartOffset)
							s.total += size
							if rule.Used {
								s.used += size
							}
						}
						var totalBytes, usedBytes int
						for id, s := range sheets {
							if s.total > 0 {
								pct := float64(s.used) / float64(s.total) * 100
								result.WriteString(fmt.Sprintf("- stylesheet %s: %.1f%% (%d/%d bytes)\n", id, pct, s.used, s.total))
								totalBytes += s.total
								usedBytes += s.used
							}
						}
						if totalBytes > 0 {
							pct := float64(usedBytes) / float64(totalBytes) * 100
							result.WriteString(fmt.Sprintf("\nCSS Total: %.1f%% (%d/%d bytes)\n", pct, usedBytes, totalBytes))
						}
					}
				}

				return mcp.NewToolResultText(result.String()), nil

			case "stop":
				var stopped []string
				if doJS {
					_ = proto.ProfilerStopPreciseCoverage{}.Call(page)
					_ = proto.ProfilerDisable{}.Call(page)
					stopped = append(stopped, "JS")
				}
				if doCSS {
					_, _ = proto.CSSStopRuleUsageTracking{}.Call(page)
					_ = proto.CSSDisable{}.Call(page)
					stopped = append(stopped, "CSS")
				}
				return mcp.NewToolResultText(fmt.Sprintf("Coverage collection stopped: %s", strings.Join(stopped, ", "))), nil

			default:
				return nil, fmt.Errorf("invalid action %q: must be start, report, or stop", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
