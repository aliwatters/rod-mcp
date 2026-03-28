package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/types/js"
)

const FillFormToolKey = "rod_fill_form"

var FillForm = mcp.NewTool(FillFormToolKey,
	mcp.WithDescription("Batch fill multiple form fields in one call. Each field is targeted by CSS selector. Automatically handles React controlled inputs. Returns a summary of results per field."),
	mcp.WithArray("fields", mcp.Description("Array of fields to fill. Each field has a 'selector' (CSS selector) and 'value' (text to fill)."),
		mcp.Items(map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "CSS selector for the form field (e.g. '#email', 'input[name=username]')",
				},
				"value": map[string]interface{}{
					"type":        "string",
					"description": "Text value to fill into the field",
				},
			},
			"required": []string{"selector", "value"},
		}),
		mcp.Required(),
	),
)

// smartFillResult holds the result of a smart fill operation.
type smartFillResult struct {
	Method  string `json:"method"`
	Value   string `json:"value"`
	React   bool   `json:"react"`
	Success bool   `json:"success"`
}

// smartFillElement fills a rod element using React-aware strategies.
// It executes the embedded smart_fill.js script which tries multiple
// fill strategies and returns the method used and final value.
func smartFillElement(element *rod.Element, value string) (*smartFillResult, error) {
	// Use the embedded smart fill JavaScript.
	obj, err := element.Eval(js.SmartFillJS, value)
	if err != nil {
		// Fallback to basic rod Input if JS eval fails.
		log.Warnf("Smart fill JS failed, falling back to basic Input: %s", err)
		if selErr := element.SelectAllText(); selErr != nil {
			log.Debugf("SelectAllText failed (may be empty): %s", selErr)
		}
		if inputErr := element.Input(value); inputErr != nil {
			return nil, fmt.Errorf("smart fill failed and basic input also failed: %w", inputErr)
		}
		return &smartFillResult{Method: "basic_fallback", Value: value, Success: true}, nil
	}

	var result smartFillResult
	if err := json.Unmarshal([]byte(obj.Value.Str()), &result); err != nil {
		// If we can't parse the result, assume the fill was attempted.
		log.Warnf("Failed to parse smart fill result: %s", err)
		return &smartFillResult{Method: "unknown", Value: value, Success: false}, nil
	}
	return &result, nil
}

var FillFormHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rodCtx.InvalidateSnapshot()

		page, err := rodCtx.ControlledPage()
		if err != nil {
			return toolErr("fill form", err)
		}

		fieldsRaw, ok := request.GetArguments()["fields"]
		if !ok {
			return toolErr("fill form", fmt.Errorf("missing required argument: fields"))
		}

		fieldsSlice, ok := fieldsRaw.([]interface{})
		if !ok {
			return toolErr("fill form", fmt.Errorf("'fields' must be an array"))
		}

		if len(fieldsSlice) == 0 {
			return toolErr("fill form", fmt.Errorf("'fields' array must not be empty"))
		}

		type fieldResult struct {
			Selector string `json:"selector"`
			Value    string `json:"value"`
			Method   string `json:"method"`
			Success  bool   `json:"success"`
			Error    string `json:"error,omitempty"`
		}

		results := make([]fieldResult, 0, len(fieldsSlice))
		successCount := 0

		for _, f := range fieldsSlice {
			fieldMap, ok := f.(map[string]interface{})
			if !ok {
				results = append(results, fieldResult{
					Error: "invalid field entry: expected object with 'selector' and 'value'",
				})
				continue
			}

			selector, _ := fieldMap["selector"].(string)
			value, _ := fieldMap["value"].(string)

			if selector == "" {
				results = append(results, fieldResult{
					Selector: selector,
					Value:    value,
					Error:    "selector is required",
				})
				continue
			}

			element, err := resolveBySelector(page, selector)
			if err != nil {
				results = append(results, fieldResult{
					Selector: selector,
					Value:    value,
					Error:    fmt.Sprintf("element not found: %s", err),
				})
				continue
			}

			fillResult, err := smartFillElement(element, value)
			if err != nil {
				results = append(results, fieldResult{
					Selector: selector,
					Value:    value,
					Error:    fmt.Sprintf("fill failed: %s", err),
				})
				continue
			}

			fr := fieldResult{
				Selector: selector,
				Value:    fillResult.Value,
				Method:   fillResult.Method,
				Success:  fillResult.Success,
			}
			if fillResult.Success {
				successCount++
			}
			results = append(results, fr)
		}

		waitDOMStable(page)

		// Build summary text.
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Filled %d/%d fields successfully.\n", successCount, len(fieldsSlice)))
		for _, r := range results {
			if r.Error != "" {
				sb.WriteString(fmt.Sprintf("  %s: ERROR — %s\n", r.Selector, r.Error))
			} else if r.Success {
				sb.WriteString(fmt.Sprintf("  %s: OK (method=%s)\n", r.Selector, r.Method))
			} else {
				sb.WriteString(fmt.Sprintf("  %s: PARTIAL (method=%s, final=%q)\n", r.Selector, r.Method, r.Value))
			}
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
	return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
}
