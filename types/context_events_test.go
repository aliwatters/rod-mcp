package types

import (
	"context"
	"fmt"
	"testing"
)

// newEventsContext creates a Context pre-populated with test data (no browser needed).
func newEventsContext() *Context {
	cfg := Config{Mode: Text}
	ctx := NewContext(context.Background(), cfg)
	return ctx
}

// addConsoleMessages populates the context's console ring buffer with test data.
func addConsoleMessages(ctx *Context, msgs []ConsoleMessage) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	for _, m := range msgs {
		ctx.consoleMessages.Add(m)
	}
}

// addNetworkRequests populates the context's network ring buffer with test data.
func addNetworkRequests(ctx *Context, reqs []NetworkRequest) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	for _, r := range reqs {
		ctx.networkRequests.Add(r)
	}
}

// addWSFrames populates the context's WebSocket frame ring buffer with test data.
func addWSFrames(ctx *Context, frames []WebSocketFrame) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	for _, f := range frames {
		ctx.wsFrames.Add(f)
	}
}

// ---------------------------------------------------------------------------
// ConsoleMessages — basic retrieval
// ---------------------------------------------------------------------------

func TestConsoleMessages_Empty(t *testing.T) {
	ctx := newEventsContext()
	msgs := ctx.ConsoleMessages("", false)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestConsoleMessages_All_NoFilter(t *testing.T) {
	ctx := newEventsContext()
	addConsoleMessages(ctx, []ConsoleMessage{
		{Level: "log", Text: "hello"},
		{Level: "error", Text: "oops"},
		{Level: "warn", Text: "careful"},
	})

	msgs := ctx.ConsoleMessages("", false)
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}
}

func TestConsoleMessages_FilterByLevel(t *testing.T) {
	ctx := newEventsContext()
	addConsoleMessages(ctx, []ConsoleMessage{
		{Level: "log", Text: "a"},
		{Level: "error", Text: "b"},
		{Level: "log", Text: "c"},
	})

	msgs := ctx.ConsoleMessages("error", false)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(msgs))
	}
	if msgs[0].Level != "error" {
		t.Errorf("Level = %q, want %q", msgs[0].Level, "error")
	}
}

func TestConsoleMessages_FilterByLevel_NoMatch(t *testing.T) {
	ctx := newEventsContext()
	addConsoleMessages(ctx, []ConsoleMessage{
		{Level: "log", Text: "x"},
	})
	msgs := ctx.ConsoleMessages("debug", false)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for non-matching level, got %d", len(msgs))
	}
}

// ---------------------------------------------------------------------------
// ConsoleMessages — clear behaviour
// ---------------------------------------------------------------------------

func TestConsoleMessages_ClearAll(t *testing.T) {
	ctx := newEventsContext()
	addConsoleMessages(ctx, []ConsoleMessage{
		{Level: "log", Text: "a"},
		{Level: "error", Text: "b"},
	})

	msgs := ctx.ConsoleMessages("", true)
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages before clear, got %d", len(msgs))
	}

	after := ctx.ConsoleMessages("", false)
	if len(after) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(after))
	}
}

func TestConsoleMessages_ClearFilteredLevel(t *testing.T) {
	ctx := newEventsContext()
	addConsoleMessages(ctx, []ConsoleMessage{
		{Level: "log", Text: "keep"},
		{Level: "error", Text: "remove"},
		{Level: "log", Text: "keep2"},
	})

	// Clear only "error" messages.
	msgs := ctx.ConsoleMessages("error", true)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(msgs))
	}

	// The "log" messages should remain.
	remaining := ctx.ConsoleMessages("", false)
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining messages, got %d", len(remaining))
	}
	for _, m := range remaining {
		if m.Level == "error" {
			t.Errorf("error message should have been removed, but found: %+v", m)
		}
	}
}

// ---------------------------------------------------------------------------
// NetworkRequests — basic retrieval
// ---------------------------------------------------------------------------

func TestNetworkRequests_Empty(t *testing.T) {
	ctx := newEventsContext()
	reqs := ctx.NetworkRequests("", "", false)
	if len(reqs) != 0 {
		t.Errorf("expected 0 requests, got %d", len(reqs))
	}
}

func TestNetworkRequests_All(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "r1", Method: "GET", URL: "https://example.com/a"},
		{RequestID: "r2", Method: "POST", URL: "https://api.example.com/b"},
	})

	reqs := ctx.NetworkRequests("", "", false)
	if len(reqs) != 2 {
		t.Errorf("expected 2 requests, got %d", len(reqs))
	}
}

func TestNetworkRequests_FilterURL(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "r1", Method: "GET", URL: "https://example.com/a"},
		{RequestID: "r2", Method: "GET", URL: "https://other.com/b"},
	})

	reqs := ctx.NetworkRequests("example.com", "", false)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request matching URL filter, got %d", len(reqs))
	}
	if reqs[0].RequestID != "r1" {
		t.Errorf("unexpected request: %+v", reqs[0])
	}
}

func TestNetworkRequests_FilterMethod(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "r1", Method: "GET", URL: "https://a.com"},
		{RequestID: "r2", Method: "POST", URL: "https://b.com"},
		{RequestID: "r3", Method: "get", URL: "https://c.com"}, // case-insensitive
	})

	reqs := ctx.NetworkRequests("", "get", false)
	if len(reqs) != 2 {
		t.Errorf("expected 2 GET requests (case-insensitive), got %d", len(reqs))
	}
}

func TestNetworkRequests_FilterURLAndMethod(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "r1", Method: "GET", URL: "https://api.example.com/x"},
		{RequestID: "r2", Method: "POST", URL: "https://api.example.com/y"},
		{RequestID: "r3", Method: "GET", URL: "https://other.com/z"},
	})

	reqs := ctx.NetworkRequests("api.example.com", "POST", false)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 match, got %d", len(reqs))
	}
	if reqs[0].RequestID != "r2" {
		t.Errorf("unexpected request: %+v", reqs[0])
	}
}

// ---------------------------------------------------------------------------
// NetworkRequests — clear behaviour
// ---------------------------------------------------------------------------

func TestNetworkRequests_Clear(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "r1", Method: "GET", URL: "https://a.com"},
	})

	reqs := ctx.NetworkRequests("", "", true)
	if len(reqs) != 1 {
		t.Errorf("expected 1 request before clear, got %d", len(reqs))
	}

	after := ctx.NetworkRequests("", "", false)
	if len(after) != 0 {
		t.Errorf("expected 0 requests after clear, got %d", len(after))
	}
}

// ---------------------------------------------------------------------------
// GetRequestID — bounds checking
// ---------------------------------------------------------------------------

func TestGetRequestID_Empty(t *testing.T) {
	ctx := newEventsContext()
	_, err := ctx.GetRequestID(0)
	if err == nil {
		t.Error("GetRequestID(0) on empty buffer: expected error, got nil")
	}
}

func TestGetRequestID_Valid(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "req-abc", Method: "GET", URL: "https://a.com"},
		{RequestID: "req-xyz", Method: "POST", URL: "https://b.com"},
	})

	id, err := ctx.GetRequestID(0)
	if err != nil {
		t.Fatalf("GetRequestID(0): unexpected error: %v", err)
	}
	if id != "req-abc" {
		t.Errorf("GetRequestID(0) = %q, want %q", id, "req-abc")
	}

	id2, err := ctx.GetRequestID(1)
	if err != nil {
		t.Fatalf("GetRequestID(1): unexpected error: %v", err)
	}
	if id2 != "req-xyz" {
		t.Errorf("GetRequestID(1) = %q, want %q", id2, "req-xyz")
	}
}

func TestGetRequestID_OutOfRange(t *testing.T) {
	ctx := newEventsContext()
	addNetworkRequests(ctx, []NetworkRequest{
		{RequestID: "r1", Method: "GET", URL: "https://a.com"},
	})

	tests := []struct {
		index int
	}{
		{-1},
		{1},
		{100},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("index=%d", tt.index), func(t *testing.T) {
			_, err := ctx.GetRequestID(tt.index)
			if err == nil {
				t.Errorf("GetRequestID(%d): expected error for out-of-range index", tt.index)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WebSocketConnections
// ---------------------------------------------------------------------------

func TestWebSocketConnections_Empty(t *testing.T) {
	ctx := newEventsContext()
	conns := ctx.WebSocketConnections("")
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestWebSocketConnections_NoFilter(t *testing.T) {
	ctx := newEventsContext()
	ctx.stateLock.Lock()
	ctx.wsConnections.Add(WebSocketConnection{RequestID: "ws1", URL: "wss://a.com/ws", Closed: false})
	ctx.wsConnections.Add(WebSocketConnection{RequestID: "ws2", URL: "wss://b.com/ws", Closed: true})
	ctx.stateLock.Unlock()

	conns := ctx.WebSocketConnections("")
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

func TestWebSocketConnections_URLFilter(t *testing.T) {
	ctx := newEventsContext()
	ctx.stateLock.Lock()
	ctx.wsConnections.Add(WebSocketConnection{RequestID: "ws1", URL: "wss://a.com/ws"})
	ctx.wsConnections.Add(WebSocketConnection{RequestID: "ws2", URL: "wss://b.com/ws"})
	ctx.stateLock.Unlock()

	conns := ctx.WebSocketConnections("a.com")
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection matching filter, got %d", len(conns))
	}
	if conns[0].RequestID != "ws1" {
		t.Errorf("unexpected connection: %+v", conns[0])
	}
}

// ---------------------------------------------------------------------------
// WebSocketFrames
// ---------------------------------------------------------------------------

func TestWebSocketFrames_Empty(t *testing.T) {
	ctx := newEventsContext()
	frames := ctx.WebSocketFrames("", "")
	if len(frames) != 0 {
		t.Errorf("expected 0 frames, got %d", len(frames))
	}
}

func TestWebSocketFrames_NoFilter(t *testing.T) {
	ctx := newEventsContext()
	addWSFrames(ctx, []WebSocketFrame{
		{URL: "wss://a.com", Direction: "sent", PayloadData: "ping"},
		{URL: "wss://a.com", Direction: "received", PayloadData: "pong"},
	})

	frames := ctx.WebSocketFrames("", "")
	if len(frames) != 2 {
		t.Errorf("expected 2 frames, got %d", len(frames))
	}
}

func TestWebSocketFrames_URLFilter(t *testing.T) {
	ctx := newEventsContext()
	addWSFrames(ctx, []WebSocketFrame{
		{URL: "wss://a.com", Direction: "sent", PayloadData: "msg1"},
		{URL: "wss://b.com", Direction: "sent", PayloadData: "msg2"},
	})

	frames := ctx.WebSocketFrames("a.com", "")
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame matching URL, got %d", len(frames))
	}
	if frames[0].URL != "wss://a.com" {
		t.Errorf("unexpected frame URL: %q", frames[0].URL)
	}
}

func TestWebSocketFrames_DirectionFilter(t *testing.T) {
	ctx := newEventsContext()
	addWSFrames(ctx, []WebSocketFrame{
		{URL: "wss://a.com", Direction: "sent", PayloadData: "out"},
		{URL: "wss://a.com", Direction: "received", PayloadData: "in"},
		{URL: "wss://a.com", Direction: "sent", PayloadData: "out2"},
	})

	frames := ctx.WebSocketFrames("", "sent")
	if len(frames) != 2 {
		t.Errorf("expected 2 sent frames, got %d", len(frames))
	}

	for _, f := range frames {
		if f.Direction != "sent" {
			t.Errorf("expected direction 'sent', got %q", f.Direction)
		}
	}
}

func TestWebSocketFrames_URLAndDirectionFilter(t *testing.T) {
	ctx := newEventsContext()
	addWSFrames(ctx, []WebSocketFrame{
		{URL: "wss://a.com", Direction: "sent", PayloadData: "a-out"},
		{URL: "wss://a.com", Direction: "received", PayloadData: "a-in"},
		{URL: "wss://b.com", Direction: "sent", PayloadData: "b-out"},
	})

	frames := ctx.WebSocketFrames("a.com", "received")
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if frames[0].PayloadData != "a-in" {
		t.Errorf("unexpected payload: %q", frames[0].PayloadData)
	}
}

// ---------------------------------------------------------------------------
// ClearWebSocketData
// ---------------------------------------------------------------------------

func TestClearWebSocketData_ClearsAll(t *testing.T) {
	ctx := newEventsContext()

	ctx.stateLock.Lock()
	idx := ctx.wsConnections.AddWithIndex(WebSocketConnection{RequestID: "ws1", URL: "wss://a.com"})
	ctx.wsConnIndex = map[string]int{"ws1": idx}
	ctx.stateLock.Unlock()
	addWSFrames(ctx, []WebSocketFrame{
		{URL: "wss://a.com", Direction: "sent", PayloadData: "data"},
	})

	ctx.ClearWebSocketData()

	conns := ctx.WebSocketConnections("")
	if len(conns) != 0 {
		t.Errorf("expected 0 connections after clear, got %d", len(conns))
	}

	frames := ctx.WebSocketFrames("", "")
	if len(frames) != 0 {
		t.Errorf("expected 0 frames after clear, got %d", len(frames))
	}

	ctx.stateLock.Lock()
	connIndex := ctx.wsConnIndex
	ctx.stateLock.Unlock()
	if connIndex != nil {
		t.Error("wsConnIndex should be nil after clear")
	}
}

func TestClearWebSocketData_EmptyState_NoError(t *testing.T) {
	ctx := newEventsContext()
	// Should not panic on empty state.
	ctx.ClearWebSocketData()

	conns := ctx.WebSocketConnections("")
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}
