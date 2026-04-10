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
	BatchToolKey = "rod_batch"
)

// batchAllowedHandlers maps tool names to their handler factories for batch execution.
// Only side-effect tools are allowed; navigation and read-only tools are excluded.
// Defined as a function to break the initialization cycle with CommonToolHandlers.
var batchAllowedHandlers = map[string]ToolHandler{
	FillToolKey:     FillHandler,
	ClickToolKey:    ClickHandler,
	HoverToolKey:    HoverHandler,
	SelectorToolKey: SelectorHandler,
	PressKeyToolKey: PressKeyHandler,
	ScrollToolKey:   ScrollHandler,
	TypeToolKey:     TypeHandler,
}

var (
	Batch = mcp.NewTool(BatchToolKey,
		mcp.WithDescription("Execute multiple actions in a single call. Reduces round trips for form fills and multi-step interactions. Stops on first error. Only side-effect tools allowed: rod_fill, rod_click, rod_hover, rod_selector, rod_press, rod_scroll, rod_type."),
		mcp.WithArray("actions",
			mcp.Description("Array of actions to execute sequentially. Each action has 'tool' (tool name) and 'args' (tool arguments)."),
			mcp.Items(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tool": map[string]interface{}{"type": "string"},
					"args": map[string]interface{}{"type": "object"},
				},
				"required": []string{"tool", "args"},
			}),
			mcp.Required(),
		),
	)
)

var (
	BatchHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()

			actionsRaw, ok := args["actions"]
			if !ok {
				return toolErr("batch", fmt.Errorf("missing required argument: actions"))
			}
			actionsArr, ok := actionsRaw.([]interface{})
			if !ok || len(actionsArr) == 0 {
				return toolErr("batch", fmt.Errorf("actions must be a non-empty array"))
			}

			type actionResult struct {
				Tool    string `json:"tool"`
				Success bool   `json:"success"`
				Message string `json:"message,omitempty"`
				Error   string `json:"error,omitempty"`
			}

			var results []actionResult

			for i, actionRaw := range actionsArr {
				actionMap, ok := actionRaw.(map[string]interface{})
				if !ok {
					return toolErr("batch", fmt.Errorf("actions[%d] must be an object", i))
				}
				toolName, ok := actionMap["tool"].(string)
				if !ok || toolName == "" {
					return toolErr("batch", fmt.Errorf("actions[%d].tool must be a non-empty string", i))
				}
				toolArgs, _ := actionMap["args"].(map[string]interface{})
				if toolArgs == nil {
					toolArgs = map[string]interface{}{}
				}

				handlerFactory, allowed := batchAllowedHandlers[toolName]
				if !allowed {
					return toolErr("batch", fmt.Errorf("actions[%d]: tool %q is not allowed in batch. Allowed: rod_fill, rod_click, rod_hover, rod_selector, rod_press, rod_scroll, rod_type", i, toolName))
				}

				// Build a synthetic MCP request
				innerReq := mcp.CallToolRequest{}
				innerReq.Params.Name = toolName
				innerReq.Params.Arguments = toolArgs

				// Execute the inner handler directly (bypass Execute wrapper)
				innerHandler := handlerFactory(rodCtx)
				result, err := innerHandler(ctx, innerReq)
				if err != nil {
					results = append(results, actionResult{
						Tool:    toolName,
						Success: false,
						Error:   err.Error(),
					})
					// Stop on first error
					break
				}

				// Extract text from result
				msg := ""
				if result != nil && len(result.Content) > 0 {
					for _, c := range result.Content {
						if tc, ok := c.(mcp.TextContent); ok {
							msg = tc.Text
							break
						}
					}
				}
				results = append(results, actionResult{
					Tool:    toolName,
					Success: true,
					Message: msg,
				})
			}

			out, marshalErr := json.MarshalIndent(map[string]interface{}{
				"total":     len(actionsArr),
				"completed": len(results),
				"results":   results,
			}, "", "  ")
			if marshalErr != nil {
				return toolErr("marshal batch result", marshalErr)
			}
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)
