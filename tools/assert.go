package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	AssertElementToolKey = "rod_assert_element"
)

var (
	AssertElement = mcp.NewTool(AssertElementToolKey,
		mcp.WithDescription("Assert element presence and optionally capture a screenshot. Returns element count and pass/fail status. On failure, includes page context for diagnosis."),
		mcp.WithString("selector", mcp.Description("CSS selector to find elements")),
		mcp.WithString("name", mcp.Description("Accessible name to search for (case-insensitive substring)")),
		mcp.WithString("role", mcp.Description("ARIA role to filter by when using name-based targeting")),
		mcp.WithBoolean("exists", mcp.Description("Whether element should exist (default: true)")),
		mcp.WithNumber("min_count", mcp.Description("Minimum expected element count")),
		mcp.WithNumber("timeout", mcp.Description("Max wait time in milliseconds (default: 5000)")),
		mcp.WithBoolean("screenshot", mcp.Description("Include page screenshot in response (default: false)")),
	)
)

var (
	AssertElementHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("assert element", err)
			}

			args := request.GetArguments()
			selector := getOptionalStringArg(args, "selector")
			name := getOptionalStringArg(args, "name")
			role := getOptionalStringArg(args, "role")
			expectExists := getOptionalBoolArg(args, "exists", true)
			minCount := int(getOptionalFloatArg(args, "min_count", 0))
			timeout := getOptionalFloatArg(args, "timeout", 5000)
			wantScreenshot := getOptionalBoolArg(args, "screenshot", false)

			if selector == "" && name == "" {
				return toolErr("assert element", fmt.Errorf("one of 'selector' or 'name' is required"))
			}

			var count int

			if selector != "" {
				// Poll for selector-based elements with timeout
				deadline := time.After(time.Duration(timeout) * time.Millisecond)
				ticker := time.NewTicker(200 * time.Millisecond)
				defer ticker.Stop()

				for {
					elements, _ := page.Elements(selector)
					count = len(elements)
					if expectExists && count > 0 && count >= minCount {
						break
					}
					if !expectExists && count == 0 {
						break
					}
					select {
					case <-deadline:
						goto done
					case <-ticker.C:
						continue
					}
				}
			} else {
				// Name-based via snapshot
				snapshot, snapErr := rodCtx.EnsureSnapshot()
				if snapErr != nil {
					return toolErr("assert element", snapErr)
				}
				matches := snapshot.FindByNameRole(name, role)
				count = len(matches)
			}
		done:

			// Build result
			passed := true
			var failReason string
			target := selector
			if target == "" {
				target = fmt.Sprintf("name=%q", name)
				if role != "" {
					target += fmt.Sprintf(" role=%q", role)
				}
			}

			if expectExists && count == 0 {
				passed = false
				failReason = fmt.Sprintf("expected %s to exist, but not found", target)
			} else if !expectExists && count > 0 {
				passed = false
				failReason = fmt.Sprintf("expected %s to not exist, but found %d", target, count)
			} else if minCount > 0 && count < minCount {
				passed = false
				failReason = fmt.Sprintf("expected at least %d elements matching %s, found %d", minCount, target, count)
			}

			result := map[string]interface{}{
				"passed": passed,
				"count":  count,
				"target": target,
			}
			if !passed {
				result["error"] = failReason
				info, infoErr := page.Info()
				if infoErr == nil {
					result["page_url"] = info.URL
					result["page_title"] = info.Title
				}
			}

			out, _ := json.MarshalIndent(result, "", "  ")

			if wantScreenshot {
				bin, shotErr := page.Screenshot(false, &proto.PageCaptureScreenshot{
					Format: proto.PageCaptureScreenshotFormatPng,
				})
				if shotErr == nil {
					encoded := base64.StdEncoding.EncodeToString(bin)
					return mcp.NewToolResultImage(string(out), encoded, "image/png"), nil
				}
			}

			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
