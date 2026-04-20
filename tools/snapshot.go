package tools

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	SnapshotToolKey = "rod_snapshot"
	ClickToolKey    = "rod_click"
	HoverToolKey    = "rod_hover"
	FillToolKey     = "rod_fill"
	SelectorToolKey = "rod_selector"
)

var (
	Snapshot = mcp.NewTool("rod_snapshot",
		mcp.WithDescription("Capture accessibility snapshot of the current page, this is better than screenshot"),
	)

	Click = mcp.NewTool(ClickToolKey,
		mcp.WithDescription("Perform click or tap on a web page. Target element by ref (from snapshot), CSS selector, or accessible name/role. Priority: ref > selector > name. Supports right-click, double-click, and touch tap."),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot.")),
		mcp.WithString("selector", mcp.Description("CSS selector to find the element (e.g. '#submit-btn', '.my-class', 'input[name=email]').")),
		mcp.WithString("name", mcp.Description("Accessible name to find the element by (case-insensitive substring match).")),
		mcp.WithString("role", mcp.Description("ARIA role to filter by when using name-based targeting (e.g. button, link, textbox). Optional, used to disambiguate.")),
		mcp.WithString("button", mcp.Description("Mouse button: left (default), right, or middle")),
		mcp.WithNumber("click_count", mcp.Description("Number of clicks: 1 (default) or 2 for double-click")),
		mcp.WithBoolean("tap", mcp.Description("Use touch tap instead of mouse click (for mobile emulation)")),
	)

	Hover = mcp.NewTool(HoverToolKey,
		mcp.WithDescription("Hover over an element to trigger CSS :hover states, tooltips, or dropdown menus. Target by ref, CSS selector, or name/role. Priority: ref > selector > name."),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot.")),
		mcp.WithString("selector", mcp.Description("CSS selector to find the element (e.g. '#menu-trigger', '.dropdown').")),
		mcp.WithString("name", mcp.Description("Accessible name to find the element by (case-insensitive substring match).")),
		mcp.WithString("role", mcp.Description("ARIA role to filter by when using name-based targeting.")),
	)

	Fill = mcp.NewTool(FillToolKey,
		mcp.WithDescription("Type text into editable element. Target by ref, CSS selector, or name/role. Priority: ref > selector > name. Automatically detects React controlled inputs and uses the appropriate fill strategy."),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("value", mcp.Description("Text to type into the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot.")),
		mcp.WithString("selector", mcp.Description("CSS selector to find the element (e.g. '#email', 'input[name=username]').")),
		mcp.WithString("name", mcp.Description("Accessible name to find the element by (case-insensitive substring match).")),
		mcp.WithString("role", mcp.Description("ARIA role to filter by when using name-based targeting.")),
		mcp.WithBoolean("submit", mcp.Description("Whether to submit entered text (press Enter after)"), mcp.Required()),
	)
	Selector = mcp.NewTool(SelectorToolKey,
		mcp.WithDescription("Select an option in a dropdown. Target by ref, CSS selector, or name/role. Priority: ref > selector > name."),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot.")),
		mcp.WithString("selector", mcp.Description("CSS selector to find the element (e.g. '#country-select', 'select[name=state]').")),
		mcp.WithString("name", mcp.Description("Accessible name to find the element by (case-insensitive substring match).")),
		mcp.WithString("role", mcp.Description("ARIA role to filter by when using name-based targeting.")),
		mcp.WithArray("values", mcp.Description("Array of values to select in the dropdown. This can be a single value or multiple values."), mcp.Items(map[string]interface{}{"type": "string"}), mcp.Required()),
	)
)

var (
	SnapshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			snapshot, err := rodCtx.BuildSnapshot()
			if err != nil {
				return toolErr("capture snapshot", err)
			}
			return mcp.NewToolResultText(snapshot), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	ClickHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			page, element, ele, err := resolveSnapshotElement(rodCtx, args, "click element")
			if err != nil {
				return toolErr("click element", err)
			}

			useTap := getOptionalBoolArg(args, "tap", false)

			if useTap {
				if err = element.Tap(); err != nil {
					return toolErr("tap element "+ele, err)
				}
				waitDOMStable(page)
				return mcp.NewToolResultText(fmt.Sprintf("Tap element %s successfully", ele)), nil
			}

			// Determine mouse button
			btn := proto.InputMouseButtonLeft
			btnName := getOptionalStringArg(args, "button")
			switch btnName {
			case "right":
				btn = proto.InputMouseButtonRight
			case "middle":
				btn = proto.InputMouseButtonMiddle
			case "", "left":
				// default
			default:
				return toolErr("click element", fmt.Errorf("invalid button %q: must be left, right, or middle", btnName))
			}

			clickCount := int(getOptionalFloatArg(args, "click_count", 1))
			if clickCount < 1 {
				clickCount = 1
			}

			if err = element.Click(btn, clickCount); err != nil {
				return toolErr("click element "+ele, err)
			}

			waitDOMStable(page)
			action := "Click"
			if clickCount == 2 {
				action = "Double-click"
			}
			if btnName == "right" {
				action = "Right-click"
			}
			return mcp.NewToolResultText(fmt.Sprintf("%s element %s successfully", action, ele)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}

	HoverHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, element, ele, err := resolveSnapshotElement(rodCtx, request.GetArguments(), "hover element")
			if err != nil {
				return toolErr("hover element", err)
			}
			if err = element.Hover(); err != nil {
				return toolErr("hover element "+ele, err)
			}

			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Hover element %s successfully", ele)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}

	FillHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, element, ele, err := resolveSnapshotElement(rodCtx, request.GetArguments(), "fill element")
			if err != nil {
				return toolErr("fill element", err)
			}

			value, err := getStringArg(request.GetArguments(), "value")
			if err != nil {
				return toolErr("fill element "+ele, err)
			}

			result, err := smartFillElement(element, value)
			if err != nil {
				return toolErr("fill element "+ele, err)
			}

			if submit, ok := request.GetArguments()["submit"].(bool); ok && submit {
				if err = element.Page().Keyboard.Press(input.Enter); err != nil {
					return toolErr("submit element "+ele, err)
				}
			}

			waitDOMStable(page)
			msg := fmt.Sprintf("Fill element %s: method=%s, success=%v", ele, result.Method, result.Success)
			if !result.Success {
				msg += fmt.Sprintf(" (final value: %q)", result.Value)
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}

	SelectorHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, element, ele, err := resolveSnapshotElement(rodCtx, request.GetArguments(), "select option in element")
			if err != nil {
				return toolErr("select option", err)
			}
			values, err := utils.OptionalStringArrayParam(request, "values")
			if err != nil {
				return toolErr("select option in element "+ele, err)
			}
			if err = element.Select(values, true, rod.SelectorTypeText); err != nil {
				return toolErr("select option in element "+ele, err)
			}
			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Select option(s) in element %s successfully", ele)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)

// SnapshotRegistrations is the single source of truth for text-mode (snapshot)
// tool definitions and their handlers.
var SnapshotRegistrations = Registrations{
	{Snapshot, SnapshotHandler},
	{Click, ClickHandler},
	{Hover, HoverHandler},
	{Fill, FillHandler},
	{Selector, SelectorHandler},
}

// Derived slices and maps kept for backward compatibility with existing tests.
var (
	SnapshotToolHandlers = SnapshotRegistrations.Handlers()
	Snapshots            = SnapshotRegistrations.Tools()
)
