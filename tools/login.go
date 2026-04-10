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

// loginParams holds parsed and defaulted parameters for the login flow.
type loginParams struct {
	loginURL        string
	username        string
	password        string
	userSelector    string
	passSelector    string
	submitSelector  string
	successSelector string
	successURL      string
	timeout         float64
	triggerSelector string
	formContainer   string
}

// parseLoginParams extracts and defaults login parameters from the MCP request.
// Config-level defaults are used when the request doesn't specify selectors.
func parseLoginParams(args map[string]interface{}, cfg types.Config) (loginParams, error) {
	loginURL, err := getStringArg(args, "url")
	if err != nil {
		return loginParams{}, err
	}
	username, err := getStringArg(args, "username")
	if err != nil {
		return loginParams{}, err
	}
	password, err := getStringArg(args, "password")
	if err != nil {
		return loginParams{}, err
	}

	p := loginParams{
		loginURL:        loginURL,
		username:        username,
		password:        password,
		userSelector:    getOptionalStringArg(args, "username_selector"),
		passSelector:    getOptionalStringArg(args, "password_selector"),
		submitSelector:  getOptionalStringArg(args, "submit_selector"),
		successSelector: getOptionalStringArg(args, "success_selector"),
		successURL:      getOptionalStringArg(args, "success_url_contains"),
		timeout:         getOptionalFloatArg(args, "timeout", defaultLoginTimeoutMs),
		triggerSelector: getOptionalStringArg(args, "trigger_selector"),
		formContainer:   getOptionalStringArg(args, "form_container"),
	}

	if p.passSelector == "" {
		p.passSelector = cfg.LoginPasswordSelector
	}
	if p.passSelector == "" {
		p.passSelector = "input[type=password]"
	}
	if p.submitSelector == "" {
		p.submitSelector = cfg.LoginSubmitSelector
	}
	if p.submitSelector == "" {
		p.submitSelector = "button[type=submit]"
	}
	if p.formContainer == "" {
		p.formContainer = "[role=dialog]"
	}
	return p, nil
}

// loginFillUsername finds and fills the username field on the page.
// If userSelector is empty, tries the provided selectors (or defaults) with optional scope prefix.
func loginFillUsername(page *rod.Page, username, userSelector, scope string, selectors []string) error {
	if userSelector != "" {
		el, err := page.Element(userSelector)
		if err != nil {
			return fmt.Errorf("selector %q: %w", userSelector, err)
		}
		return loginSmartFill(el, username)
	}

	if len(selectors) == 0 {
		selectors = defaultUsernameSelectors
	}
	for _, sel := range selectors {
		if scope != "" {
			sel = scope + " " + sel
		}
		el, err := page.Element(sel)
		if err == nil {
			if fillErr := loginSmartFill(el, username); fillErr == nil {
				return nil
			}
		}
	}
	if scope != "" {
		return fmt.Errorf("could not find username field in %s; provide username_selector", scope)
	}
	return fmt.Errorf("could not find username field; provide username_selector")
}

// loginFillPassword finds and fills the password field, trying a scoped selector as fallback.
func loginFillPassword(page *rod.Page, password, passSelector, scope string) error {
	el, err := page.Element(passSelector)
	if err != nil && scope != "" {
		el, err = page.Element(scope + " " + passSelector)
	}
	if err != nil {
		return fmt.Errorf("selector %q: %w", passSelector, err)
	}
	return loginSmartFill(el, password)
}

// loginSubmit finds and clicks the submit button, trying a scoped selector as fallback.
func loginSubmit(page *rod.Page, submitSelector, scope string) error {
	el, err := page.Element(submitSelector)
	if err != nil && scope != "" {
		el, err = page.Element(scope + " " + submitSelector)
	}
	if err != nil {
		return fmt.Errorf("selector %q: %w", submitSelector, err)
	}
	return el.Click(proto.InputMouseButtonLeft, 1)
}

// loginOpenTrigger clicks a trigger element and waits for the form container to appear.
func loginOpenTrigger(page *rod.Page, triggerSelector, formContainer string) error {
	triggerEl, err := page.Element(triggerSelector)
	if err != nil {
		return fmt.Errorf("find trigger %q: %w", triggerSelector, err)
	}
	if err := triggerEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click trigger: %w", err)
	}
	if _, err := page.Timeout(defaultFormContainerTimeout).Element(formContainer); err != nil {
		return fmt.Errorf("form container %q did not appear after clicking trigger: %w", formContainer, err)
	}
	waitDOMStable(page)
	return nil
}

// loginVerifySuccess polls for success indicators until timeout.
func loginVerifySuccess(page *rod.Page, successSelector, successURL string, timeout float64) bool {
	if successSelector == "" && successURL == "" {
		if navErr := page.WaitLoad(); navErr == nil {
			waitDOMStable(page)
		}
		return true
	}

	timeoutDur := time.Duration(timeout) * time.Millisecond
	deadline := time.After(timeoutDur)
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		if successSelector != "" {
			if _, err := page.Element(successSelector); err == nil {
				return true
			}
		}
		if successURL != "" {
			if info, err := page.Info(); err == nil && strings.Contains(info.URL, successURL) {
				return true
			}
		}
		select {
		case <-deadline:
			return false
		case <-ticker.C:
			continue
		}
	}
}

// loginBuildResult collects the login result (URL, title, cookies) into JSON.
func loginBuildResult(page *rod.Page, verified bool, timeout float64) (string, error) {
	info, infoErr := page.Info()
	currentURL := ""
	title := ""
	if infoErr != nil {
		log.Debugf("login page info: %s", infoErr)
	} else if info != nil {
		currentURL = info.URL
		title = info.Title
	}

	resp, cookieErr := proto.NetworkGetCookies{}.Call(page)
	cookieCount := 0
	if cookieErr != nil {
		log.Debugf("login get cookies: %s", cookieErr)
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

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal login result: %w", err)
	}
	return string(out), nil
}

var (
	LoginHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			cfg := rodCtx.Config()
			p, err := parseLoginParams(request.GetArguments(), cfg)
			if err != nil {
				return toolErr("login", err)
			}

			// Step 1: Navigate to login page
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("login navigate", err)
			}
			if err := page.Navigate(p.loginURL); err != nil {
				return toolErr("login navigate", err)
			}
			if err := page.WaitLoad(); err != nil {
				return toolErr("login wait load", err)
			}
			waitDOMStable(page)

			// Fail fast if the login page returned an HTTP error (e.g. 404).
			checkURL := p.loginURL
			if pageInfo, infoErr := page.Info(); infoErr == nil && pageInfo.URL != "" {
				checkURL = pageInfo.URL
			}
			if err := checkNavigationStatus(rodCtx, checkURL); err != nil {
				return toolErr("login navigate", err)
			}

			// Step 2: Fill credentials and submit
			scope := "" // no scoping for standard login pages
			if p.triggerSelector != "" {
				if err := loginOpenTrigger(page, p.triggerSelector, p.formContainer); err != nil {
					return toolErr("login trigger", err)
				}
				scope = p.formContainer
			}

			if err := loginFillUsername(page, p.username, p.userSelector, scope, cfg.LoginUsernameSelectors); err != nil {
				return toolErr("login fill username", err)
			}
			if err := loginFillPassword(page, p.password, p.passSelector, scope); err != nil {
				return toolErr("login fill password", err)
			}
			if err := loginSubmit(page, p.submitSelector, scope); err != nil {
				return toolErr("login submit", err)
			}

			// Step 3: Verify success
			verified := loginVerifySuccess(page, p.successSelector, p.successURL, p.timeout)

			// Step 4: Build result
			out, err := loginBuildResult(page, verified, p.timeout)
			if err != nil {
				return toolErr("login", err)
			}
			return mcp.NewToolResultText(out), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: true})
	}
)
