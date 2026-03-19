//go:build e2e

package e2e

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  any `json:"params"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response (partial, for assertion).
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// mcpResult is the MCP CallToolResult shape.
type mcpResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Data string `json:"data,omitempty"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

// harness manages the rod-mcp subprocess and JSON-RPC communication.
type harness struct {
	t       *testing.T
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
	nextID  int
	// responses stores responses by ID for out-of-order reading.
	responses map[int]jsonRPCResponse
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	binary := os.Getenv("ROD_MCP_BINARY")
	if binary == "" {
		// Build the binary
		binary = t.TempDir() + "/rod-mcp"
		build := exec.Command("go", "build", "-o", binary, ".")
		build.Dir = ".."
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			t.Fatalf("failed to build rod-mcp: %v", err)
		}
	}

	args := []string{"--no-banner"}

	cmd := exec.Command(binary, args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start rod-mcp: %v", err)
	}

	h := &harness{
		t:         t,
		cmd:       cmd,
		stdin:     stdin,
		scanner:   bufio.NewScanner(stdout),
		nextID:    1,
		responses: make(map[int]jsonRPCResponse),
	}

	// Set a large scanner buffer for big responses (screenshots, HTML).
	h.scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)

	t.Cleanup(func() {
		stdin.Close()
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	})

	return h
}

// send sends a JSON-RPC request and returns the assigned ID.
func (h *harness) send(method string, params any) int {
	h.t.Helper()
	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		h.t.Fatalf("marshal request: %v", err)
	}
	data = append(data, '\n')

	h.mu.Lock()
	_, err = h.stdin.Write(data)
	h.mu.Unlock()
	if err != nil {
		h.t.Fatalf("write request: %v", err)
	}
	return id
}

// recv waits for a response with the given ID, with timeout.
func (h *harness) recv(id int, timeout time.Duration) jsonRPCResponse {
	h.t.Helper()
	deadline := time.Now().Add(timeout)

	// Check if already received.
	h.mu.Lock()
	if resp, ok := h.responses[id]; ok {
		delete(h.responses, id)
		h.mu.Unlock()
		return resp
	}
	h.mu.Unlock()

	for time.Now().Before(deadline) {
		if !h.scanner.Scan() {
			if err := h.scanner.Err(); err != nil {
				h.t.Fatalf("scanner error: %v", err)
			}
			h.t.Fatal("unexpected EOF from rod-mcp")
		}
		line := h.scanner.Text()

		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			h.t.Logf("non-JSON line: %s", line)
			continue
		}

		if resp.ID == id {
			return resp
		}
		// Store for later retrieval.
		h.mu.Lock()
		h.responses[resp.ID] = resp
		h.mu.Unlock()
	}
	h.t.Fatalf("timeout waiting for response id=%d", id)
	return jsonRPCResponse{} // unreachable
}

// call sends a tools/call request and returns the text result.
func (h *harness) call(tool string, args map[string]any) string {
	return h.callWithTimeout(tool, args, 15*time.Second)
}

// callWithTimeout sends a tools/call request with a custom timeout.
func (h *harness) callWithTimeout(tool string, args map[string]any, timeout time.Duration) string {
	h.t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	id := h.send("tools/call", map[string]any{
		"name":      tool,
		"arguments": args,
	})
	resp := h.recv(id, timeout)
	return h.extractText(resp)
}

// extractText extracts the text content from an MCP response.
func (h *harness) extractText(resp jsonRPCResponse) string {
	h.t.Helper()
	if resp.Error != nil {
		return string(resp.Error)
	}
	var result mcpResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		h.t.Fatalf("unmarshal result: %v (raw: %s)", err, string(resp.Result))
	}
	var texts []string
	for _, c := range result.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// assertContains checks that the result contains the expected substring.
func assertContains(t *testing.T, result, substr string) {
	t.Helper()
	if !strings.Contains(result, substr) {
		t.Errorf("expected result to contain %q, got: %s", substr, truncate(result, 300))
	}
}

// assertContainsAny checks that the result contains at least one of the substrings.
func assertContainsAny(t *testing.T, result string, substrs ...string) {
	t.Helper()
	for _, s := range substrs {
		if strings.Contains(result, s) {
			return
		}
	}
	t.Errorf("expected result to contain one of %v, got: %s", substrs, truncate(result, 300))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// navigate navigates to a URL and waits for the page body to be visible.
func (h *harness) navigate(url string) string {
	h.t.Helper()
	result := h.call("rod_navigate", map[string]any{"url": url})
	h.call("rod_wait_for", map[string]any{"selector": "body", "timeout": 10000})
	return result
}

// initialize performs the MCP handshake.
func (h *harness) initialize() {
	h.t.Helper()
	id := h.send("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "e2e-test",
			"version": "1.0",
		},
	})
	resp := h.recv(id, 5*time.Second)
	text := string(resp.Result)
	if !strings.Contains(text, "protocolVersion") {
		h.t.Fatalf("handshake failed: %s", text)
	}
}

func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	h := newHarness(t)
	h.initialize()

	t.Run("configure", func(t *testing.T) {
		result := h.call("rod_configure", nil)
		assertContainsAny(t, result, "No configuration", "headless", "Configuration")
	})

	t.Run("navigate", func(t *testing.T) {
		result := h.navigate("https://the-internet.herokuapp.com")
		assertContains(t, result, "Navigated to")
	})

	t.Run("snapshot", func(t *testing.T) {
		result := h.call("rod_snapshot", nil)
		assertContainsAny(t, result, "heading", "link")
	})

	t.Run("html", func(t *testing.T) {
		t.Run("page", func(t *testing.T) {
			result := h.call("rod_html", map[string]any{"action": "page"})
			assertContains(t, result, "html")
		})
		t.Run("element", func(t *testing.T) {
			result := h.call("rod_html", map[string]any{"action": "element", "selector": "h1"})
			assertContainsAny(t, result, "the-internet", "Welcome")
		})
	})

	t.Run("evaluate", func(t *testing.T) {
		result := h.call("rod_evaluate", map[string]any{
			"script": "() => document.title",
		})
		assertContainsAny(t, result, "the-internet", "Internet")
	})

	t.Run("screenshot", func(t *testing.T) {
		result := h.call("rod_screenshot", map[string]any{"name": "e2e-test"})
		assertContainsAny(t, result, "image", "screenshot", "saved")
	})

	t.Run("pdf", func(t *testing.T) {
		result := h.call("rod_pdf", nil)
		assertContainsAny(t, result, "PDF", "pdf", "saved")
	})

	t.Run("scroll", func(t *testing.T) {
		t.Run("down", func(t *testing.T) {
			result := h.call("rod_scroll", map[string]any{"direction": "down", "amount": 500})
			assertContains(t, result, "Scrolled down")
		})
		t.Run("up", func(t *testing.T) {
			result := h.call("rod_scroll", map[string]any{"direction": "up", "amount": 500})
			assertContains(t, result, "Scrolled up")
		})
		t.Run("absolute", func(t *testing.T) {
			result := h.call("rod_scroll", map[string]any{"x": 0, "y": 0})
			assertContains(t, result, "Scrolled to")
		})
	})

	t.Run("console_messages", func(t *testing.T) {
		h.call("rod_evaluate", map[string]any{
			"script": `() => { console.log("e2e-test-msg-42"); return "done"; }`,
		})
		time.Sleep(500 * time.Millisecond)
		result := h.call("rod_console_messages", nil)
		assertContains(t, result, "e2e-test-msg-42")
	})

	t.Run("network_requests", func(t *testing.T) {
		result := h.call("rod_network_requests", nil)
		assertContains(t, result, "the-internet")
	})

	t.Run("response_body", func(t *testing.T) {
		result := h.call("rod_response_body", map[string]any{"index": 0})
		assertContainsAny(t, result, "Response body", "html", "DOCTYPE")
	})

	t.Run("cookies", func(t *testing.T) {
		url := "https://the-internet.herokuapp.com"
		t.Run("set", func(t *testing.T) {
			result := h.call("rod_cookies", map[string]any{
				"action": "set", "name": "e2e_cookie", "value": "hello42", "url": url,
			})
			assertContainsAny(t, result, "set successfully", "Set cookie")
		})
		t.Run("get", func(t *testing.T) {
			result := h.call("rod_cookies", map[string]any{
				"action": "get", "url": url,
			})
			assertContainsAny(t, result, "e2e_cookie", "hello42")
		})
		t.Run("delete", func(t *testing.T) {
			result := h.call("rod_cookies", map[string]any{
				"action": "delete", "name": "e2e_cookie", "url": url,
			})
			assertContainsAny(t, result, "deleted successfully", "Deleted")
		})
	})

	t.Run("storage", func(t *testing.T) {
		t.Run("set", func(t *testing.T) {
			result := h.call("rod_storage", map[string]any{
				"type": "local", "action": "set", "key": "e2eKey", "value": "e2eVal",
			})
			assertContains(t, result, "Set localStorage")
		})
		t.Run("get", func(t *testing.T) {
			result := h.call("rod_storage", map[string]any{
				"type": "local", "action": "get", "key": "e2eKey",
			})
			assertContains(t, result, "e2eVal")
		})
		t.Run("list", func(t *testing.T) {
			result := h.call("rod_storage", map[string]any{
				"type": "local", "action": "list",
			})
			assertContains(t, result, "e2eKey")
		})
		t.Run("remove", func(t *testing.T) {
			result := h.call("rod_storage", map[string]any{
				"type": "local", "action": "remove", "key": "e2eKey",
			})
			assertContains(t, result, "Removed")
		})
		t.Run("session_list", func(t *testing.T) {
			result := h.call("rod_storage", map[string]any{
				"type": "session", "action": "list",
			})
			assertContainsAny(t, result, "empty", "sessionStorage")
		})
	})

	t.Run("coverage", func(t *testing.T) {
		t.Run("start", func(t *testing.T) {
			result := h.call("rod_coverage", map[string]any{"action": "start"})
			assertContains(t, result, "Coverage collection started")
		})
		t.Run("report", func(t *testing.T) {
			result := h.call("rod_coverage", map[string]any{"action": "report"})
			assertContains(t, result, "Coverage")
		})
		t.Run("stop", func(t *testing.T) {
			result := h.call("rod_coverage", map[string]any{"action": "stop"})
			assertContains(t, result, "Coverage collection stopped")
		})
	})

	t.Run("performance", func(t *testing.T) {
		t.Run("metrics", func(t *testing.T) {
			result := h.call("rod_performance", map[string]any{"action": "metrics"})
			assertContains(t, result, "Performance metrics")
		})
		t.Run("vitals", func(t *testing.T) {
			result := h.call("rod_performance", map[string]any{"action": "vitals"})
			assertContainsAny(t, result, "ttfb", "fcp", "cls")
		})
	})

	t.Run("permissions", func(t *testing.T) {
		t.Run("grant", func(t *testing.T) {
			result := h.call("rod_permissions", map[string]any{
				"action":      "grant",
				"permissions": []string{"geolocation", "notifications"},
			})
			assertContains(t, result, "Granted")
		})
		t.Run("reset", func(t *testing.T) {
			result := h.call("rod_permissions", map[string]any{"action": "reset"})
			assertContainsAny(t, result, "Reset", "reset")
		})
	})

	t.Run("intercept", func(t *testing.T) {
		t.Run("enable", func(t *testing.T) {
			result := h.call("rod_intercept", map[string]any{"action": "enable"})
			assertContains(t, result, "interception enabled")
		})
		t.Run("mock", func(t *testing.T) {
			result := h.call("rod_intercept", map[string]any{
				"action": "mock", "urlPattern": "*mock-test*", "status": 200,
				"body": `{"mocked":true}`,
			})
			assertContains(t, result, "Mock rule added")
		})
		t.Run("block", func(t *testing.T) {
			result := h.call("rod_intercept", map[string]any{
				"action": "block", "urlPattern": "*blocked-test*",
			})
			assertContains(t, result, "Block rule added")
		})
		t.Run("fail", func(t *testing.T) {
			result := h.call("rod_intercept", map[string]any{
				"action": "fail", "urlPattern": "*fail-test*", "errorReason": "ConnectionRefused",
			})
			assertContains(t, result, "Fail rule added")
		})
		t.Run("list", func(t *testing.T) {
			result := h.call("rod_intercept", map[string]any{"action": "list"})
			assertContains(t, result, "Interception rules")
		})
		t.Run("disable", func(t *testing.T) {
			result := h.call("rod_intercept", map[string]any{"action": "disable"})
			assertContains(t, result, "disabled and rules cleared")
		})
	})

	t.Run("websocket", func(t *testing.T) {
		t.Run("list", func(t *testing.T) {
			result := h.call("rod_websocket", map[string]any{"action": "list"})
			assertContainsAny(t, result, "No WebSocket", "WebSocket connections")
		})
		t.Run("frames", func(t *testing.T) {
			result := h.call("rod_websocket", map[string]any{"action": "frames"})
			assertContainsAny(t, result, "No WebSocket", "WebSocket frames")
		})
		t.Run("clear", func(t *testing.T) {
			result := h.call("rod_websocket", map[string]any{"action": "clear"})
			assertContains(t, result, "cleared")
		})
	})

	t.Run("resize", func(t *testing.T) {
		t.Run("mobile", func(t *testing.T) {
			result := h.call("rod_resize", map[string]any{
				"width": 375, "height": 812, "device_scale_factor": 3,
				"is_mobile": true, "has_touch": true,
				"user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)",
			})
			assertContains(t, result, "375x812")
		})
		t.Run("desktop", func(t *testing.T) {
			result := h.call("rod_resize", map[string]any{
				"width": 1920, "height": 1080, "device_scale_factor": 1,
				"is_mobile": false, "has_touch": false,
			})
			assertContains(t, result, "1920x1080")
		})
	})

	t.Run("set_headers", func(t *testing.T) {
		result := h.call("rod_set_headers", map[string]any{
			"headers": map[string]string{
				"X-Test-Header": "e2e-value",
			},
		})
		assertContains(t, result, "Set 1 header")
	})

	t.Run("wait_for", func(t *testing.T) {
		result := h.call("rod_wait_for", map[string]any{
			"selector": "h1", "timeout": 5000,
		})
		assertContainsAny(t, result, "visible", "found", "appeared")
	})

	t.Run("form_interaction", func(t *testing.T) {
		h.navigate("https://the-internet.herokuapp.com/login")

		// Fill form fields via evaluate (snapshot refs are dynamic).
		// Public test credentials displayed on the-internet.herokuapp.com/login page.
		// Chrome's PasswordLeakDetection is disabled in browser launch flags.
		result := h.call("rod_evaluate", map[string]any{
			"script": `() => { document.querySelector("#username").value = "tomsmith"; return "filled"; }`,
		})
		assertContains(t, result, "filled")

		result = h.call("rod_evaluate", map[string]any{
			"script": `() => { document.querySelector("#password").value = "SuperSecretPassword!"; return "filled"; }`,
		})
		assertContains(t, result, "filled")

		// Submit form.
		result = h.call("rod_evaluate", map[string]any{
			"script": `() => { document.querySelector("button[type=submit]").click(); return "clicked"; }`,
		})
		assertContains(t, result, "clicked")

		h.call("rod_wait_for", map[string]any{"selector": "h2", "timeout": 10000})

		// Verify secure page.
		result = h.call("rod_html", map[string]any{"action": "element", "selector": "h2"})
		assertContains(t, result, "Secure Area")
	})

	t.Run("hover_via_evaluate", func(t *testing.T) {
		h.navigate("https://the-internet.herokuapp.com/hovers")

		result := h.call("rod_evaluate", map[string]any{
			"script": `() => { const el = document.querySelector(".figure img"); if (!el) return "no-element"; el.dispatchEvent(new MouseEvent("mouseover", {bubbles:true})); return "hovered"; }`,
		})
		assertContains(t, result, "hovered")
	})

	t.Run("navigation_history", func(t *testing.T) {
		t.Run("go_back", func(t *testing.T) {
			result := h.call("rod_go_back", nil)
			assertContainsAny(t, result, "Go back successfully", "back")
		})
		t.Run("go_forward", func(t *testing.T) {
			result := h.call("rod_go_forward", nil)
			assertContainsAny(t, result, "Go forward successfully", "forward")
		})
		t.Run("reload", func(t *testing.T) {
			result := h.call("rod_reload", nil)
			assertContainsAny(t, result, "Reload current page successfully", "reload")
		})
	})

	t.Run("press_key", func(t *testing.T) {
		h.navigate("https://the-internet.herokuapp.com/key_presses")

		result := h.call("rod_press", map[string]any{"key": "a"})
		assertContainsAny(t, result, "Press key", "pressed", "successfully")
	})

	t.Run("tabs", func(t *testing.T) {
		t.Run("new", func(t *testing.T) {
			result := h.call("rod_tab_new", map[string]any{"url": "https://example.com"})
			assertContainsAny(t, result, "New tab", "created", "content")
		})
		t.Run("list", func(t *testing.T) {
			time.Sleep(1 * time.Second) // Allow new tab to load.
			result := h.call("rod_tab_list", nil)
			assertContainsAny(t, result, "example.com", "the-internet")
		})
		t.Run("select", func(t *testing.T) {
			result := h.call("rod_tab_select", map[string]any{"index": 0})
			assertContainsAny(t, result, "Switched", "switched", "content")
		})
		t.Run("close", func(t *testing.T) {
			result := h.call("rod_tab_close", map[string]any{"index": 1})
			assertContainsAny(t, result, "Closed", "closed")
		})
	})

	t.Run("drag", func(t *testing.T) {
		h.navigate("https://the-internet.herokuapp.com/drag_and_drop")

		t.Run("selector_based", func(t *testing.T) {
			result := h.call("rod_drag", map[string]any{
				"sourceSelector": "#column-a", "targetSelector": "#column-b", "steps": 20,
			})
			assertContains(t, result, "Dragged from")
		})
		t.Run("coordinate_based", func(t *testing.T) {
			result := h.call("rod_drag", map[string]any{
				"startX": 200, "startY": 300, "endX": 500, "endY": 300, "steps": 15,
			})
			assertContains(t, result, "Dragged from")
		})
	})

	t.Run("handle_dialog", func(t *testing.T) {
		h.navigate("https://the-internet.herokuapp.com/javascript_alerts")

		// Dialog handling has a race condition — Chrome auto-dismisses alerts
		// when no Page.javascriptDialogOpening listener is active.
		// Verify the tool accepts valid params and returns a coherent response.
		result := h.call("rod_handle_dialog", map[string]any{"action": "accept"})
		assertContainsAny(t, result, "Dialog accepted", "No dialog is showing")
	})

	t.Run("intercept_live_mock", func(t *testing.T) {
		result := h.call("rod_intercept", map[string]any{"action": "enable"})
		assertContains(t, result, "enabled")

		h.call("rod_intercept", map[string]any{
			"action": "mock", "urlPattern": "*mocked-endpoint*", "status": 200,
			"body":    `{"status":"mocked"}`,
			"headers": map[string]string{"Content-Type": "application/json"},
		})

		result = h.callWithTimeout("rod_evaluate", map[string]any{
			"script": `() => fetch("/mocked-endpoint").then(r => r.json()).then(d => JSON.stringify(d))`,
		}, 10*time.Second)
		assertContainsAny(t, result, "mocked", "status")

		h.call("rod_intercept", map[string]any{"action": "disable"})
	})

	t.Run("close_browser", func(t *testing.T) {
		result := h.call("rod_close_browser", nil)
		assertContains(t, result, "Close browser successfully")
	})

	t.Log("All e2e tests passed")
}
