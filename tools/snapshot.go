package tools

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
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
	FillToolKey     = "rod_fill"
	SelectorToolKey = "rod_selector"
)

var (
	Snapshot = mcp.NewTool("rod_snapshot",
		mcp.WithDescription("Capture accessibility snapshot of the current page, this is better than screenshot"),
	)

	Click = mcp.NewTool(ClickToolKey,
		mcp.WithDescription("Perform click on a web page"),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot"), mcp.Required()),
	)

	Fill = mcp.NewTool(FillToolKey,
		mcp.WithDescription("Type text into editable element"),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("value", mcp.Description("Text to type into the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot"), mcp.Required()),
		mcp.WithBoolean("submit", mcp.Description("Whether to submit entered text (press Enter after)"), mcp.Required()),
	)
	Selector = mcp.NewTool(SelectorToolKey,
		mcp.WithDescription("Select an option in a dropdown"),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot"), mcp.Required()),
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
			ele := request.Params.Arguments["element"].(string)
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("click element "+ele, err)
			}

			snapshot, err := rodCtx.LatestSnapshot()
			if err != nil {
				return toolErr("click element "+ele, err)
			}

			ref := request.Params.Arguments["ref"].(string)
			element, err := snapshot.LocatorInFrame(ref)
			if err != nil {
				return toolErr("click element "+ele, err)
			}
			if err = element.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return toolErr("click element "+ele, err)
			}

			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Click element %s successfully", ele)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}

	FillHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ele := request.Params.Arguments["element"].(string)
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("fill element "+ele, err)
			}

			snapshot, err := rodCtx.LatestSnapshot()
			if err != nil {
				return toolErr("fill element "+ele, err)
			}

			ref := request.Params.Arguments["ref"].(string)
			element, err := snapshot.LocatorInFrame(ref)
			if err != nil {
				return toolErr("fill element "+ele, err)
			}

			value := request.Params.Arguments["value"].(string)
			// Clear existing value by selecting all text first, then input new value
			// This ensures password fields and React-controlled inputs work correctly
			if err = element.SelectAllText(); err != nil {
				log.Warnf("Failed to select all text in element %s (may be empty): %s", ele, err)
				// Continue anyway - field may be empty or select may not be supported
			}
			if err = element.Input(value); err != nil {
				return toolErr("fill element "+ele, err)
			}
			if submit, ok := request.Params.Arguments["submit"].(bool); ok && submit {
				if err = element.Page().Keyboard.Press(input.Enter); err != nil {
					return toolErr("submit element "+ele, err)
				}
			}

			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Fill out element %s successfully", ele)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}

	SelectorHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ele := request.Params.Arguments["element"].(string)
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("select option in element "+ele, err)
			}

			snapshot, err := rodCtx.LatestSnapshot()
			if err != nil {
				return toolErr("select option in element "+ele, err)
			}

			ref := request.Params.Arguments["ref"].(string)
			element, err := snapshot.LocatorInFrame(ref)
			if err != nil {
				return toolErr("select option in element "+ele, err)
			}
			values, err := utils.OptionalStringArrayParam(request, "values")
			if err != nil {
				return toolErr("select option in element "+ele, err)
			}
			if err = element.Select(values, true, rod.SelectorTypeText); err != nil {
				return toolErr("select option in element "+ele, err)
			}
			page.WaitDOMStable(defaultWaitStableDur, defaultDomDiff)
			return mcp.NewToolResultText(fmt.Sprintf("Select option(s) in element %s successfully", ele)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)

var (
	SnapshotToolHandlers = map[string]ToolHandler{
		SnapshotToolKey: SnapshotHandler,
		ClickToolKey:    ClickHandler,
		FillToolKey:     FillHandler,
		SelectorToolKey: SelectorHandler,
	}
	Snapshots = []mcp.Tool{
		Snapshot,
		Click,
		Fill,
		Selector,
	}
)
