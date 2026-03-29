package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
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
		mcp.WithString("action", mcp.Description("Action to perform"), mcp.Required(), mcp.Enum("get", "set", "delete", "clear", "save", "restore")),
		mcp.WithString("url", mcp.Description("URL to get cookies for (get) or associate with (set/delete). Defaults to current page URL.")),
		mcp.WithString("name", mcp.Description("Cookie name (required for set and delete)")),
		mcp.WithString("value", mcp.Description("Cookie value (required for set)")),
		mcp.WithString("domain", mcp.Description("Cookie domain (optional for set/delete)")),
		mcp.WithString("path", mcp.Description("Cookie path (optional for set/delete)")),
		mcp.WithBoolean("secure", mcp.Description("Secure flag (optional for set)")),
		mcp.WithBoolean("httpOnly", mcp.Description("HttpOnly flag (optional for set)")),
		mcp.WithString("sameSite", mcp.Description("SameSite attribute (optional for set)"), mcp.Enum("Strict", "Lax", "None")),
		mcp.WithString("file_path", mcp.Description("File path for save/restore actions. Must be within the output directory.")),
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

			case "save":
				savePath, err := getCookieFilePath(rodCtx, args)
				if err != nil {
					return toolErr("save cookies", err)
				}

				resp, err := proto.NetworkGetAllCookies{}.Call(page)
				if err != nil {
					return toolErr("save cookies", err)
				}
				data, err := json.MarshalIndent(resp.Cookies, "", "  ")
				if err != nil {
					return toolErr("save cookies", err)
				}
				if err := os.WriteFile(savePath, data, 0o644); err != nil {
					return toolErr("save cookies", fmt.Errorf("write %s: %w", savePath, err))
				}
				return mcp.NewToolResultText(fmt.Sprintf("Saved %d cookies to %s", len(resp.Cookies), savePath)), nil

			case "restore":
				restorePath, err := getCookieFilePath(rodCtx, args)
				if err != nil {
					return toolErr("restore cookies", err)
				}

				data, err := os.ReadFile(restorePath)
				if err != nil {
					return toolErr("restore cookies", fmt.Errorf("read %s: %w", restorePath, err))
				}
				var cookies []*proto.NetworkCookie
				if err := json.Unmarshal(data, &cookies); err != nil {
					return toolErr("restore cookies", fmt.Errorf("parse %s: %w", restorePath, err))
				}

				now := proto.TimeSinceEpoch(time.Now().Unix())
				var restored, skipped int
				for _, c := range cookies {
					// Skip expired cookies
					if c.Expires > 0 && c.Expires < now {
						log.Debugf("cookie %q expired, skipping", c.Name)
						skipped++
						continue
					}
					setCookie := proto.NetworkSetCookie{
						Name:     c.Name,
						Value:    c.Value,
						Domain:   c.Domain,
						Path:     c.Path,
						Secure:   c.Secure,
						HTTPOnly: c.HTTPOnly,
						SameSite: c.SameSite,
					}
					if c.Expires > 0 {
						setCookie.Expires = c.Expires
					}
					if _, err := setCookie.Call(page); err != nil {
						log.Warnf("failed to restore cookie %q: %s", c.Name, err)
						continue
					}
					restored++
				}
				msg := fmt.Sprintf("Restored %d cookies from %s", restored, restorePath)
				if skipped > 0 {
					msg += fmt.Sprintf(" (%d expired cookies skipped)", skipped)
				}
				return mcp.NewToolResultText(msg), nil

					default:
				return nil, fmt.Errorf("invalid action %q: must be get, set, delete, clear, save, or restore", action)
			}
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// getCookieFilePath resolves and validates the file path for cookie save/restore.
// Restricts paths to be within the output directory to prevent path traversal.
func getCookieFilePath(rodCtx *types.Context, args map[string]interface{}) (string, error) {
	fp := getOptionalStringArg(args, "file_path")
	if fp == "" {
		return "", fmt.Errorf("file_path is required for save/restore actions")
	}

	outDir, err := types.ResolveOutputDir(rodCtx.Config())
	if err != nil {
		return "", err
	}

	// Resolve relative to output directory
	if !filepath.IsAbs(fp) {
		fp = filepath.Join(outDir, fp)
	}
	cleaned := filepath.Clean(fp)
	if !strings.HasPrefix(cleaned, filepath.Clean(outDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("file_path must be within output directory %s", outDir)
	}
	return cleaned, nil
}
