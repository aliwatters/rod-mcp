package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	WebSocketToolKey = "rod_websocket"
)

var (
	WebSocket = mcp.NewTool(WebSocketToolKey,
		mcp.WithDescription("Monitor WebSocket connections and their messages. Frames are captured automatically after rod_navigate."),
		mcp.WithString("action", mcp.Description("Action to perform"), mcp.Required(), mcp.Enum("list", "frames", "clear")),
		mcp.WithString("urlFilter", mcp.Description("Filter connections/frames by URL substring")),
		mcp.WithString("direction", mcp.Description("Filter frames by direction"), mcp.Enum("sent", "received")),
		mcp.WithNumber("maxFrames", mcp.Description("Maximum number of frames to return (default: 100)")),
	)
)

var (
	WebSocketHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			action, err := getStringArg(args, "action")
			if err != nil {
				return toolErr("websocket", err)
			}

			switch action {
			case "list":
				connections := rodCtx.WebSocketConnections(getOptionalStringArg(args, "urlFilter"))
				if len(connections) == 0 {
					return mcp.NewToolResultText("No WebSocket connections captured"), nil
				}

				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("WebSocket connections (%d):\n", len(connections)))
				for i, conn := range connections {
					status := "open"
					if conn.Closed {
						status = "closed"
					}
					sb.WriteString(fmt.Sprintf("  %d. [%s] %s (frames: %d sent, %d received)\n",
						i, status, conn.URL, conn.SentCount, conn.ReceivedCount))
				}
				return mcp.NewToolResultText(sb.String()), nil

			case "frames":
				urlFilter := getOptionalStringArg(args, "urlFilter")
				direction := getOptionalStringArg(args, "direction")
				maxFrames := int(getOptionalFloatArg(args, "maxFrames", 100))

				frames := rodCtx.WebSocketFrames(urlFilter, direction)
				if len(frames) == 0 {
					return mcp.NewToolResultText("No WebSocket frames captured"), nil
				}

				total := len(frames)
				if len(frames) > maxFrames {
					frames = frames[len(frames)-maxFrames:]
				}

				var sb strings.Builder
				if total > maxFrames {
					sb.WriteString(fmt.Sprintf("WebSocket frames (showing last %d of %d):\n", maxFrames, total))
				} else {
					sb.WriteString(fmt.Sprintf("WebSocket frames (%d):\n", len(frames)))
				}
				for _, f := range frames {
					dir := "→"
					if f.Direction == "received" {
						dir = "←"
					}
					payload := f.PayloadData
					if len(payload) > 200 {
						payload = payload[:200] + "..."
					}
					sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", dir, f.URL, payload))
				}
				return mcp.NewToolResultText(sb.String()), nil

			case "clear":
				rodCtx.ClearWebSocketData()
				return mcp.NewToolResultText("WebSocket data cleared"), nil

			default:
				return nil, fmt.Errorf("invalid action %q: must be list, frames, or clear", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
