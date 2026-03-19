package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	CookiesToolKey = "rod_cookies"
)

var (
	Cookies = mcp.NewTool(CookiesToolKey,
		mcp.WithDescription("Manage browser cookies via CDP. Get, set, delete, or clear cookies including httpOnly cookies not visible to document.cookie."),
		mcp.WithString("action", mcp.Description("Action to perform"), mcp.Required(), mcp.Enum("get", "set", "delete", "clear")),
		mcp.WithString("url", mcp.Description("URL to get cookies for (get) or associate with (set/delete). Defaults to current page URL.")),
		mcp.WithString("name", mcp.Description("Cookie name (required for set and delete)")),
		mcp.WithString("value", mcp.Description("Cookie value (required for set)")),
		mcp.WithString("domain", mcp.Description("Cookie domain (optional for set/delete)")),
		mcp.WithString("path", mcp.Description("Cookie path (optional for set/delete)")),
		mcp.WithBoolean("secure", mcp.Description("Secure flag (optional for set)")),
		mcp.WithBoolean("httpOnly", mcp.Description("HttpOnly flag (optional for set)")),
		mcp.WithString("sameSite", mcp.Description("SameSite attribute (optional for set)"), mcp.Enum("Strict", "Lax", "None")),
	)
)

var (
	CookiesHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("cookies", err)
			}

			action, err := getStringArg(request.GetArguments(), "action")
			if err != nil {
				return toolErr("cookies", err)
			}
			args := request.GetArguments()

			switch action {
			case "get":
				var urls []string
				if u := getOptionalStringArg(args, "url"); u != "" {
					urls = []string{u}
				}
				resp, err := proto.NetworkGetCookies{Urls: urls}.Call(page)
				if err != nil {
					return toolErr("get cookies", err)
				}
				if len(resp.Cookies) == 0 {
					return mcp.NewToolResultText("No cookies found"), nil
				}

				type cookieInfo struct {
					Name     string `json:"name"`
					Value    string `json:"value"`
					Domain   string `json:"domain"`
					Path     string `json:"path"`
					Secure   bool   `json:"secure,omitempty"`
					HTTPOnly bool   `json:"httpOnly,omitempty"`
					SameSite string `json:"sameSite,omitempty"`
					Session  bool   `json:"session,omitempty"`
				}
				cookies := make([]cookieInfo, 0, len(resp.Cookies))
				for _, c := range resp.Cookies {
					cookies = append(cookies, cookieInfo{
						Name:     c.Name,
						Value:    c.Value,
						Domain:   c.Domain,
						Path:     c.Path,
						Secure:   c.Secure,
						HTTPOnly: c.HTTPOnly,
						SameSite: string(c.SameSite),
						Session:  c.Session,
					})
				}
				out, err := json.MarshalIndent(cookies, "", "  ")
				if err != nil {
					return toolErr("format cookies", err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("%d cookies:\n%s", len(cookies), string(out))), nil

			case "set":
				name, err := getStringArg(args, "name")
				if err != nil {
					return toolErr("set cookie", err)
				}
				value, err := getStringArg(args, "value")
				if err != nil {
					return toolErr("set cookie", err)
				}
				cookie := proto.NetworkSetCookie{
					Name:     name,
					Value:    value,
					URL:      getOptionalStringArg(args, "url"),
					Domain:   getOptionalStringArg(args, "domain"),
					Path:     getOptionalStringArg(args, "path"),
					Secure:   getOptionalBoolArg(args, "secure", false),
					HTTPOnly: getOptionalBoolArg(args, "httpOnly", false),
				}
				// Default to current page URL if neither url nor domain is specified
				if cookie.URL == "" && cookie.Domain == "" {
					info, err := page.Info()
					if err == nil {
						cookie.URL = info.URL
					}
				}
				if ss := getOptionalStringArg(args, "sameSite"); ss != "" {
					cookie.SameSite = proto.NetworkCookieSameSite(ss)
				}
				_, err = cookie.Call(page)
				if err != nil {
					return toolErr(fmt.Sprintf("set cookie %q", name), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Cookie %q set successfully", name)), nil

			case "delete":
				name, err := getStringArg(args, "name")
				if err != nil {
					return toolErr("delete cookie", err)
				}
				delReq := proto.NetworkDeleteCookies{
					Name:   name,
					URL:    getOptionalStringArg(args, "url"),
					Domain: getOptionalStringArg(args, "domain"),
					Path:   getOptionalStringArg(args, "path"),
				}
				// Default to current page URL if no scope is specified
				if delReq.URL == "" && delReq.Domain == "" {
					info, err := page.Info()
					if err == nil {
						delReq.URL = info.URL
					}
				}
				err = delReq.Call(page)
				if err != nil {
					return toolErr(fmt.Sprintf("delete cookie %q", name), err)
				}
				return mcp.NewToolResultText(fmt.Sprintf("Cookie %q deleted successfully", name)), nil

			case "clear":
				err := proto.NetworkClearBrowserCookies{}.Call(page)
				if err != nil {
					return toolErr("clear cookies", err)
				}
				return mcp.NewToolResultText("All cookies cleared"), nil

			default:
				return nil, fmt.Errorf("invalid action %q: must be get, set, delete, or clear", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
