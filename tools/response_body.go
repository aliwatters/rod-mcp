package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	ResponseBodyToolKey = "rod_response_body"
)

var (
	ResponseBody = mcp.NewTool(ResponseBodyToolKey,
		mcp.WithDescription("Get the response body of a captured network request. Use rod_network_requests first to see available requests and their indices."),
		mcp.WithNumber("index", mcp.Description("Index of the request from rod_network_requests output (0-based)"), mcp.Required()),
		mcp.WithNumber("maxLength", mcp.Description(fmt.Sprintf("Maximum response body length to return in characters (default: %d)", defaultMaxContentLength))),
	)
)

var (
	ResponseBodyHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("response body", err)
			}

			args := request.GetArguments()
			index, err := getFloatArg(args, "index")
			if err != nil {
				return toolErr("response body", err)
			}
			maxLength := int(getOptionalFloatArg(args, "maxLength", float64(defaultMaxContentLength)))

			requestID, err := rodCtx.GetRequestID(int(index))
			if err != nil {
				return toolErr("response body", err)
			}

			result, err := proto.NetworkGetResponseBody{
				RequestID: proto.NetworkRequestID(requestID),
			}.Call(page)
			if err != nil {
				return toolErr("response body", err)
			}

			body := result.Body
			if result.Base64Encoded {
				decoded, err := base64.StdEncoding.DecodeString(result.Body)
				if err != nil {
					return toolErr("decode response body", err)
				}
				body = string(decoded)
			}

			body, truncated := truncateContent(body, maxLength)

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Response body (request #%d):\n", int(index)))
			sb.WriteString(body)
			if truncated {
				sb.WriteString(fmt.Sprintf("\n\n[truncated at %d characters]", maxLength))
			}

			return mcp.NewToolResultText(sb.String()), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
