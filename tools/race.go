package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	RaceToolKey = "rod_race"
)

var (
	Race = mcp.NewTool(RaceToolKey,
		mcp.WithDescription("Wait for the first of multiple selectors to appear. Useful for branching outcomes like success/error messages, login/redirect, loading/content states."),
		mcp.WithArray("selectors",
			mcp.Description("Array of CSS selectors to race against each other"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
		mcp.WithNumber("timeout", mcp.Description("Max wait time in milliseconds (default: 30000)")),
	)
)

var (
	RaceHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("race", err)
			}

			args := request.GetArguments()
			timeout := getOptionalFloatArg(args, "timeout", defaultWaitTimeoutMs)

			// Extract selectors array
			selectorsRaw, ok := args["selectors"]
			if !ok {
				return toolErr("race", fmt.Errorf("missing required argument: selectors"))
			}
			selectorArr, ok := selectorsRaw.([]interface{})
			if !ok || len(selectorArr) == 0 {
				return toolErr("race", fmt.Errorf("selectors must be a non-empty array of strings"))
			}
			var selectors []string
			for i, v := range selectorArr {
				s, ok := v.(string)
				if !ok || s == "" {
					return toolErr("race", fmt.Errorf("selectors[%d] must be a non-empty string", i))
				}
				selectors = append(selectors, s)
			}

			if len(selectors) < 2 {
				return toolErr("race", fmt.Errorf("at least 2 selectors are required for a race"))
			}

			// Build the race context
			timeoutDur := time.Duration(timeout) * time.Millisecond
			racePage := page.Timeout(timeoutDur)
			rc := racePage.Race()

			// Track which selector won
			var matchedIndex int
			for i, sel := range selectors {
				idx := i
				s := sel
				rc = rc.Element(s).Handle(func(_ *rod.Element) error {
					matchedIndex = idx
					return nil
				})
			}

			el, raceErr := rc.Do()
			if raceErr != nil {
				return toolErr("race", fmt.Errorf("no selector matched within %dms: %w", int(timeout), raceErr))
			}

			// Get text content of the matched element
			text, _ := el.Text()
			tag, _ := el.Eval(`() => this.tagName.toLowerCase()`)
			tagName := ""
			if tag != nil {
				tagName = tag.Value.Str()
			}

			result := map[string]interface{}{
				"matched_index":    matchedIndex,
				"matched_selector": selectors[matchedIndex],
				"tag":              tagName,
				"text":             truncateRaceText(text),
			}

			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return toolErr("marshal race result", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// truncateRaceText truncates matched element text to a reasonable preview length.
func truncateRaceText(s string) string {
	t, _ := truncateContent(s, 500)
	return t
}
