package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	StorageToolKey = "rod_storage"
)

var (
	Storage = mcp.NewTool(StorageToolKey,
		mcp.WithDescription("Read and write browser localStorage or sessionStorage"),
		mcp.WithString("type", mcp.Description("Storage type"), mcp.Required(), mcp.Enum("local", "session")),
		mcp.WithString("action", mcp.Description("Action to perform"), mcp.Required(), mcp.Enum("get", "set", "remove", "list", "clear")),
		mcp.WithString("key", mcp.Description("Storage key (required for get, set, remove)")),
		mcp.WithString("value", mcp.Description("Value to set (required for set)")),
	)
)

var (
	StorageHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("storage", err)
			}

			args := request.GetArguments()
			storageType, err := getStringArg(args, "type")
			if err != nil {
				return toolErr("storage", err)
			}
			action, err := getStringArg(args, "action")
			if err != nil {
				return toolErr("storage", err)
			}

			var storageObj string
			switch storageType {
			case "local":
				storageObj = "localStorage"
			case "session":
				storageObj = "sessionStorage"
			default:
				return nil, fmt.Errorf("invalid type %q: must be local or session", storageType)
			}

			switch action {
			case "get":
				key, err := getStringArg(args, "key")
				if err != nil {
					return toolErr("storage get", err)
				}
				r, err := page.Eval(fmt.Sprintf(`(key) => %s.getItem(key)`, storageObj), key)
				if err != nil {
					return toolErr("storage get", err)
				}
				if r.Value.Nil() {
					return mcp.NewToolResultText(fmt.Sprintf("Key %q not found in %s", key, storageObj)), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("%s[%q] = %s", storageObj, key, r.Value.Str())), nil

			case "set":
				key, err := getStringArg(args, "key")
				if err != nil {
					return toolErr("storage set", err)
				}
				value, err := getStringArg(args, "value")
				if err != nil {
					return toolErr("storage set", err)
				}
				_, err = page.Eval(fmt.Sprintf(`(key, value) => %s.setItem(key, value)`, storageObj), key, value)
				if err != nil {
					return toolErr("storage set", err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Set %s[%q] successfully", storageObj, key)), nil

			case "remove":
				key, err := getStringArg(args, "key")
				if err != nil {
					return toolErr("storage remove", err)
				}
				_, err = page.Eval(fmt.Sprintf(`(key) => %s.removeItem(key)`, storageObj), key)
				if err != nil {
					return toolErr("storage remove", err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Removed %s[%q] successfully", storageObj, key)), nil

			case "list":
				r, err := page.Eval(fmt.Sprintf(`() => {
					const s = %s;
					const items = {};
					for (let i = 0; i < s.length; i++) {
						const key = s.key(i);
						items[key] = s.getItem(key);
					}
					return JSON.stringify(items, null, 2);
				}`, storageObj))
				if err != nil {
					return toolErr("storage list", err)
				}
				val := r.Value.Str()
				if val == "{}" {
					return mcp.NewToolResultText(fmt.Sprintf("%s is empty", storageObj)), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("%s contents:\n%s", storageObj, val)), nil

			case "clear":
				_, err := page.Eval(fmt.Sprintf(`() => %s.clear()`, storageObj))
				if err != nil {
					return toolErr("storage clear", err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("%s cleared successfully", storageObj)), nil

			default:
				return nil, fmt.Errorf("invalid action %q: must be get, set, remove, list, or clear", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
