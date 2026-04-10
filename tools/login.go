package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/types/js"
)

const (
	LoginToolKey = "rod_login"
)

var (
	Login = mcp.NewTool(LoginToolKey,
		mcp.WithDescription("Execute a login flow in one call: navigate to URL, fill credentials, submit, and verify success. Supports both dedicated login pages and modal dialogs triggered by a button click. Replaces 5-6 individual MCP calls."),
		mcp.WithString("url", mcp.Description("Login page URL (or base page URL when using trigger_selector)"), mcp.Required()),
		mcp.WithString("username", mcp.Description("Username or email to fill"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password to fill"), mcp.Required()),
		mcp.WithString("username_selector", mcp.Description("CSS selector for username/email field (default: input[type=email], input[name=email], input[name=username])")),
		mcp.WithString("password_selector", mcp.Description("CSS selector for password field (default: input[type=password])")),
		mcp.WithString("submit_selector", mcp.Description("CSS selector for submit button (default: button[type=submit])")),
		mcp.WithString("success_selector", mcp.Description("CSS selector that indicates login succeeded (waits for it to appear)")),
		mcp.WithString("success_url_contains", mcp.Description("Substring to match in URL after login to verify success")),
		mcp.WithNumber("timeout", mcp.Description("Max wait time for success verification in milliseconds (default: 15000)")),
		mcp.WithString("trigger_selector", mcp.Description("CSS selector for a button/link that opens the login form (for modal-based logins). Navigate to url first, then click this element before filling credentials.")),
		mcp.WithString("form_container", mcp.Description("CSS selector for the login form container to wait for after clicking trigger (default: [role=dialog]). Only used with trigger_selector.")),
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

// loginSmartFill fills an input using the embedded smart fill JS for React support.
func loginSmartFill(element *rod.Element, value string) error {
	obj, err := element.Eval(js.SmartFillJS, value)
	if err != nil {
		return fmt.Errorf("smart fill failed: %w", err)
	}
	var result struct {
		Success bool   `json:"success"`
		Value   string `json:"value"`
	}
	if parseErr := json.Unmarshal([]byte(obj.Value.Str()), &result); parseErr != nil {
		log.Warnf("loginSmartFill: failed to parse smart fill result: %s", parseErr)
		return nil // fill was attempted, can't verify
	}
	if !result.Success {
		return fmt.Errorf("fill produced %q instead of expected value", result.Value)
	}
	return nil
}

var (
	LoginHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			triggerSelector := getOptionalStringArg(args, "trigger_selector")
			formContainer := getOptionalStringArg(args, "form_container")

			if passSelector == "" {
				passSelector = "input[type=password]"
			}
			if submitSelector == "" {
				submitSelector = "button[type=submit]"
			}
			if formContainer == "" {
				formContainer = "[role=dialog]"
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

			// Fail fast if the login page returned an HTTP error (e.g. 404).
			// Use the final page URL to handle redirects (e.g. http→https).
			checkURL := loginURL
			if pageInfo, infoErr := page.Info(); infoErr == nil && pageInfo.URL != "" {
				checkURL = pageInfo.URL
			}
			if err := checkNavigationStatus(rodCtx, checkURL); err != nil {
				return toolErr("login navigate", err)
			}

			// Step 1b: If trigger_selector is set, click it to open the login form
			if triggerSelector != "" {
				triggerEl, err := page.Element(triggerSelector)
				if err != nil {
					return toolErr("login find trigger", fmt.Errorf("selector %q: %w", triggerSelector, err))
				}
				if err := triggerEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
					return toolErr("login click trigger", err)
				}
				// Wait for the form container to appear
				if _, err := page.Timeout(5 * time.Second).Element(formContainer); err != nil {
					return toolErr("login wait for form", fmt.Errorf("form container %q did not appear after clicking trigger: %w", formContainer, err))
				}
				waitDOMStable(page)

				// Scope selectors to form container for modal logins
				if userSelector == "" {
					// Try default selectors scoped to the form container
					filled := false
					for _, sel := range defaultUsernameSelectors {
						scopedSel := formContainer + " " + sel
						el, elErr := page.Element(scopedSel)
						if elErr == nil {
							if fillErr := loginSmartFill(el, username); fillErr == nil {
								filled = true
								break
							}
						}
					}
					if !filled {
						return toolErr("login fill username", fmt.Errorf("could not find username field in %s; provide username_selector", formContainer))
					}
				} else {
					el, err := page.Element(userSelector)
					if err != nil {
						return toolErr("login find username", fmt.Errorf("selector %q: %w", userSelector, err))
					}
					if err := loginSmartFill(el, username); err != nil {
						return toolErr("login fill username", err)
					}
				}

				// Fill password (scoped to form container if no explicit selector)
				passEl, err := page.Element(passSelector)
				if err != nil {
					// Try scoped selector
					passEl, err = page.Element(formContainer + " " + passSelector)
					if err != nil {
						return toolErr("login find password", fmt.Errorf("selector %q: %w", passSelector, err))
					}
				}
				if err := loginSmartFill(passEl, password); err != nil {
					return toolErr("login fill password", err)
				}

				// Submit (scoped to form container if no explicit selector)
				submitEl, err := page.Element(submitSelector)
				if err != nil {
					submitEl, err = page.Element(formContainer + " " + submitSelector)
					if err != nil {
						return toolErr("login find submit", fmt.Errorf("selector %q: %w", submitSelector, err))
					}
				}
				if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
					return toolErr("login click submit", err)
				}
			} else {
				// Standard login page flow (no trigger)

				// Step 2: Find and fill username field
				if userSelector != "" {
					el, err := page.Element(userSelector)
					if err != nil {
						return toolErr("login find username", fmt.Errorf("selector %q: %w", userSelector, err))
					}
					if err := loginSmartFill(el, username); err != nil {
						return toolErr("login fill username", err)
					}
				} else {
					filled := false
					for _, sel := range defaultUsernameSelectors {
						el, err := page.Element(sel)
						if err == nil {
							if fillErr := loginSmartFill(el, username); fillErr == nil {
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
				if err := loginSmartFill(passEl, password); err != nil {
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

			info, infoErr := page.Info()
			currentURL := ""
			title := ""
			if infoErr != nil {
				log.Warnf("login page info: %s", infoErr)
			} else if info != nil {
				currentURL = info.URL
				title = info.Title
			}

			// Count cookies set
			resp, cookieErr := proto.NetworkGetCookies{}.Call(page)
			cookieCount := 0
			if cookieErr != nil {
				log.Warnf("login get cookies: %s", cookieErr)
			} else if resp != nil {
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

			out, marshalErr := json.MarshalIndent(result, "", "  ")
			if marshalErr != nil {
				return toolErr("login marshal result", marshalErr)
			}
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)
