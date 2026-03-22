package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	PressKeyToolKey    = "rod_press"
	FileUploadToolKey  = "rod_file_upload"
)

var (
	PressKey = mcp.NewTool(PressKeyToolKey,
		mcp.WithDescription("Press a key on the keyboard"),
		mcp.WithString("key", mcp.Description("Name of the key to press or a character to generate, such as `ArrowLeft` or `a`"), mcp.Required()),
	)
	FileUpload = mcp.NewTool(FileUploadToolKey,
		mcp.WithDescription("Upload file(s) to a file input element"),
		mcp.WithString("selector", mcp.Description("CSS selector of the file input element")),
		mcp.WithString("ref", mcp.Description("Element reference from the page snapshot")),
		mcp.WithArray("paths", mcp.Description("File paths to upload"), mcp.Items(map[string]interface{}{"type": "string"}), mcp.Required()),
	)
)

var (
	PressKeyHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Key presses may modify DOM; invalidate so Execute rebuilds the snapshot.
			rodCtx.InvalidateSnapshot()
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("press key", err)
			}
			keyStr, err := getStringArg(request.GetArguments(), "key")
			if err != nil {
				return toolErr("press key", err)
			}
			key, err := parseKey(keyStr)
			if err != nil {
				return toolErr("parse key "+keyStr, err)
			}
			if err = page.Keyboard.Press(key); err != nil {
				return toolErr("press key "+keyStr, err)
			}
			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Press key %s successfully", keyStr)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	FileUploadHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			selector, _ := request.GetArguments()["selector"].(string)
			ref, _ := request.GetArguments()["ref"].(string)

			if selector == "" && ref == "" {
				return nil, errors.New("either 'selector' or 'ref' is required")
			}

			paths, err := utils.OptionalStringArrayParam(request, "paths")
			if err != nil {
				return toolErr("parse file paths", err)
			}
			if len(paths) == 0 {
				return nil, errors.New("at least one file path is required")
			}

			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("upload files", err)
			}

			var element *rod.Element
			if ref != "" {
				snapshot, err := rodCtx.LatestSnapshot()
				if err != nil {
					return toolErr("upload files", err)
				}
				element, err = snapshot.LocatorInFrame(ref)
				if err != nil {
					return toolErr("upload files", err)
				}
			} else {
				element, err = page.Element(selector)
				if err != nil {
					return toolErr("find file input "+selector, err)
				}
			}

			if err = element.SetFiles(paths); err != nil {
				return toolErr("upload files", err)
			}

			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Uploaded %d file(s) successfully", len(paths))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
