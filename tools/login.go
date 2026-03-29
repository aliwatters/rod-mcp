package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	LoginToolKey = "rod_login"

	defaultLoginTimeoutMs = 15000.0
)

var (
	Login = mcp.NewTool(LoginToolKey,
		mcp.WithDescription("Execute a login flow in one call: navigate to URL, fill credentials, submit, and verify success. Replaces 5-6 individual MCP calls."),
		mcp.WithString("url", mcp.Description("Login page URL"), mcp.Required()),
		mcp.WithString("username", mcp.Description("Username or email to fill"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password to fill"), mcp.Required()),
		mcp.WithString("username_selector", mcp.Description("CSS selector for username/email field (default: input[type=email], input[name=email], input[name=username])")),
		mcp.WithString("password_selector", mcp.Description("CSS selector for password field (default: input[type=password])")),
		mcp.WithString("submit_selector", mcp.Description("CSS selector for submit button (default: button[type=submit])")),
		mcp.WithString("success_selector", mcp.Description("CSS selector that indicates login succeeded (waits for it to appear)")),
		mcp.WithString("success_url_contains", mcp.Description("Substring to match in URL after login to verify success")),
		mcp.WithNumber("timeout", mcp.Description("Max wait time for success verification in milliseconds (default: 15000)")),
	)
)

// defaultUsernameSelectors are tried in order when no username_selector is provided.
var defaultUsernameSelectors = []string{
	"input[type=email]",
	"input[name=email]",
	"input[name=username]",
	"input[name=login]",
	"input[id=email]",
	"input[id=username]",
}

var (
	LoginHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			rodCtx.InvalidateSnapshot()

			args := request.GetArguments()
			loginURL, err := getStringArg(args, "url")
			if err != nil {
				return toolErr("login", err)
			}
			username, err := getStringArg(args, "username")
			if err != nil {
				return toolErr("login", err)
			}
			password, err := getStringArg(args, "password")
			if err != nil {
				return toolErr("login", err)
			}

			userSelector := getOptionalStringArg(args, "username_selector")
			passSelector := getOptionalStringArg(args, "password_selector")
			submitSelector := getOptionalStringArg(args, "submit_selector")
			successSelector := getOptionalStringArg(args, "success_selector")
			successURL := getOptionalStringArg(args, "success_url_contains")
			timeout := getOptionalFloatArg(args, "timeout", defaultLoginTimeoutMs)

			if passSelector == "" {
				passSelector = "input[type=password]"
			}
			if submitSelector == "" {
				submitSelector = "button[type=submit]"
			}

			// Step 1: Navigate to login page
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("login navigate", err)
			}
			if err := page.Navigate(loginURL); err != nil {
				return toolErr("login navigate", err)
			}
			if err := page.WaitLoad(); err != nil {
				return toolErr("login wait load", err)
			}
			waitDOMStable(page)

			// Step 2: Find and fill username field
			if userSelector != "" {
				el, err := page.Element(userSelector)
				if err != nil {
					return toolErr("login find username", fmt.Errorf("selector %q: %w", userSelector, err))
				}
				if err := el.SelectAllText(); err != nil {
					return toolErr("login clear username", err)
				}
				if err := el.Input(username); err != nil {
					return toolErr("login fill username", err)
				}
			} else {
				// Try default selectors
				filled := false
				for _, sel := range defaultUsernameSelectors {
					el, err := page.Element(sel)
					if err == nil {
						_ = el.SelectAllText()
						if inputErr := el.Input(username); inputErr == nil {
							filled = true
							break
						}
					}
				}
				if !filled {
					return toolErr("login fill username", fmt.Errorf("could not find username field; provide username_selector"))
				}
			}

			// Step 3: Fill password
			passEl, err := page.Element(passSelector)
			if err != nil {
				return toolErr("login find password", fmt.Errorf("selector %q: %w", passSelector, err))
			}
			if err := passEl.SelectAllText(); err != nil {
				return toolErr("login clear password", err)
			}
			if err := passEl.Input(password); err != nil {
				return toolErr("login fill password", err)
			}

			// Step 4: Submit
			submitEl, err := page.Element(submitSelector)
			if err != nil {
				return toolErr("login find submit", fmt.Errorf("selector %q: %w", submitSelector, err))
			}
			if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return toolErr("login click submit", err)
			}

			// Step 5: Verify success
			timeoutDur := time.Duration(timeout) * time.Millisecond
			deadline := time.After(timeoutDur)
			ticker := time.NewTicker(300 * time.Millisecond)
			defer ticker.Stop()

			verified := false
			if successSelector == "" && successURL == "" {
				// No explicit success check — wait for navigation to settle
				if navErr := page.WaitLoad(); navErr == nil {
					waitDOMStable(page)
				}
				verified = true
			} else {
				for {
					if successSelector != "" {
						if _, err := page.Element(successSelector); err == nil {
							verified = true
							break
						}
					}
					if successURL != "" {
						info, err := page.Info()
						if err == nil {
							if strings.Contains(info.URL, successURL) {
								verified = true
								break
							}
						}
					}
					select {
					case <-deadline:
						goto done
					case <-ticker.C:
						continue
					}
				}
			}
		done:

			info, _ := page.Info()
			currentURL := ""
			title := ""
			if info != nil {
				currentURL = info.URL
				title = info.Title
			}

			// Count cookies set
			resp, _ := proto.NetworkGetCookies{}.Call(page)
			cookieCount := 0
			if resp != nil {
				cookieCount = len(resp.Cookies)
			}

			result := map[string]interface{}{
				"success":     verified,
				"url":         currentURL,
				"title":       title,
				"cookies_set": cookieCount,
			}
			if !verified {
				result["error"] = fmt.Sprintf("login verification timed out after %dms", int(timeout))
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)

