package types

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const (
	// maxConsoleMessages is the maximum number of console messages retained.
	maxConsoleMessages = 10000
	// maxNetworkRequests is the maximum number of network requests retained.
	maxNetworkRequests = 10000
	// maxWSConnections is the maximum number of WebSocket connections retained.
	maxWSConnections = 1000
	// maxWSFrames is the maximum number of WebSocket frames retained.
	maxWSFrames = 10000
)

// ConsoleMessage represents a captured browser console message.
type ConsoleMessage struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// NetworkRequest represents a captured network request with its response.
type NetworkRequest struct {
	RequestID string `json:"requestId"`
	Method    string `json:"method"`
	URL       string `json:"url"`
	Status    int    `json:"status"`
	Type      string `json:"type"`
}

// WebSocketConnection represents a tracked WebSocket connection.
type WebSocketConnection struct {
	RequestID     string `json:"requestId"`
	URL           string `json:"url"`
	Closed        bool   `json:"closed"`
	SentCount     int    `json:"sentCount"`
	ReceivedCount int    `json:"receivedCount"`
}

// WebSocketFrame represents a single captured WebSocket message.
type WebSocketFrame struct {
	URL         string `json:"url"`
	Direction   string `json:"direction"` // "sent" or "received"
	PayloadData string `json:"payloadData"`
	Opcode      int    `json:"opcode"`
}

// attachEventListeners registers goroutine-based listeners for console messages
// and network requests on a cancelable page copy. It returns a cancel function
// that stops all listener goroutines; callers must invoke it when the page is
// no longer needed to avoid goroutine leaks.
func (ctx *Context) attachEventListeners(page *rod.Page) (cancel func()) {
	// Create a cancelable copy of the page so that cancelling stops EachEvent.
	cancelPage, cancelFn := page.WithCancel()

	go cancelPage.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		var parts []string
		for _, arg := range e.Args {
			parts = append(parts, arg.Value.String())
		}
		text := strings.Join(parts, " ")
		ctx.stateLock.Lock()
		ctx.consoleMessages.Add(ConsoleMessage{
			Level: string(e.Type),
			Text:  text,
		})
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkRequestWillBeSent) {
		ctx.stateLock.Lock()
		if ctx.pendingRequests == nil {
			ctx.pendingRequests = make(map[string]int)
		}
		// AddWithIndex returns the internal ring-buffer slot index, which is
		// stable until the slot is evicted by a future overflow.
		idx := ctx.networkRequests.AddWithIndex(NetworkRequest{
			RequestID: string(e.RequestID),
			Method:    e.Request.Method,
			URL:       e.Request.URL,
			Type:      string(e.Type),
		})
		ctx.pendingRequests[string(e.RequestID)] = idx
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkResponseReceived) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.pendingRequests[string(e.RequestID)]; ok {
			status := e.Response.Status
			typ := string(e.Type)
			ctx.networkRequests.UpdateAt(idx, func(req *NetworkRequest) {
				req.Status = status
				req.Type = typ
			})
			delete(ctx.pendingRequests, string(e.RequestID))
		}
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketCreated) {
		ctx.stateLock.Lock()
		if ctx.wsConnIndex == nil {
			ctx.wsConnIndex = make(map[string]int)
		}
		idx := ctx.wsConnections.AddWithIndex(WebSocketConnection{
			RequestID: string(e.RequestID),
			URL:       e.URL,
		})
		ctx.wsConnIndex[string(e.RequestID)] = idx
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketFrameSent) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.wsConnIndex[string(e.RequestID)]; ok {
			ctx.wsConnections.UpdateAt(idx, func(conn *WebSocketConnection) {
				conn.SentCount++
				ctx.appendWSFrame(conn.URL, "sent", e.Response)
			})
		}
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketFrameReceived) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.wsConnIndex[string(e.RequestID)]; ok {
			ctx.wsConnections.UpdateAt(idx, func(conn *WebSocketConnection) {
				conn.ReceivedCount++
				ctx.appendWSFrame(conn.URL, "received", e.Response)
			})
		}
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketClosed) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.wsConnIndex[string(e.RequestID)]; ok {
			ctx.wsConnections.UpdateAt(idx, func(conn *WebSocketConnection) {
				conn.Closed = true
			})
		}
		ctx.stateLock.Unlock()
	})()

	return cancelFn
}

// ConsoleMessages returns captured console messages, optionally filtered by level.
// If clear is true, the buffer is emptied after returning.
func (ctx *Context) ConsoleMessages(filterLevel string, clear bool) []ConsoleMessage {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	all := ctx.consoleMessages.Slice()
	result := filterSlice(all, func(msg ConsoleMessage) bool {
		return filterLevel == "" || msg.Level == filterLevel
	})

	if clear {
		if filterLevel == "" {
			ctx.consoleMessages.Clear()
		} else {
			// Rebuild the buffer keeping only non-matching messages.
			fresh := NewRingBuffer[ConsoleMessage](maxConsoleMessages)
			for _, msg := range filterSlice(all, func(msg ConsoleMessage) bool {
				return msg.Level != filterLevel
			}) {
				fresh.Add(msg)
			}
			ctx.consoleMessages = fresh
		}
	}

	return result
}

// NetworkRequests returns captured network requests, optionally filtered by URL pattern and method.
// If clear is true, the buffer is emptied after returning.
func (ctx *Context) NetworkRequests(filterURL, filterMethod string, clear bool) []NetworkRequest {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	result := filterSlice(ctx.networkRequests.Slice(), func(req NetworkRequest) bool {
		if filterURL != "" && !strings.Contains(req.URL, filterURL) {
			return false
		}
		if filterMethod != "" && !strings.EqualFold(req.Method, filterMethod) {
			return false
		}
		return true
	})

	if clear {
		ctx.networkRequests.Clear()
		ctx.pendingRequests = nil
	}

	return result
}

// appendWSFrame adds a WebSocket frame to the ring buffer (O(1), no copying).
// Must be called with stateLock held.
func (ctx *Context) appendWSFrame(url, direction string, frame *proto.NetworkWebSocketFrame) {
	ctx.wsFrames.Add(WebSocketFrame{
		URL:         url,
		Direction:   direction,
		PayloadData: frame.PayloadData,
		Opcode:      int(frame.Opcode),
	})
}

// GetRequestID returns the CDP request ID for a network request at the given index.
func (ctx *Context) GetRequestID(index int) (string, error) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	all := ctx.networkRequests.Slice()
	if len(all) == 0 {
		return "", fmt.Errorf("no network requests captured")
	}
	if index < 0 || index >= len(all) {
		return "", fmt.Errorf("request index %d out of range (0-%d)", index, len(all)-1)
	}
	return all[index].RequestID, nil
}

// WebSocketConnections returns tracked WebSocket connections, optionally filtered by URL.
func (ctx *Context) WebSocketConnections(urlFilter string) []WebSocketConnection {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	return filterSlice(ctx.wsConnections.Slice(), func(conn WebSocketConnection) bool {
		return urlFilter == "" || strings.Contains(conn.URL, urlFilter)
	})
}

// WebSocketFrames returns captured WebSocket frames, optionally filtered by URL and direction.
func (ctx *Context) WebSocketFrames(urlFilter, direction string) []WebSocketFrame {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	return filterSlice(ctx.wsFrames.Slice(), func(f WebSocketFrame) bool {
		if urlFilter != "" && !strings.Contains(f.URL, urlFilter) {
			return false
		}
		if direction != "" && f.Direction != direction {
			return false
		}
		return true
	})
}

// ClearWebSocketData clears all WebSocket connections and frames.
func (ctx *Context) ClearWebSocketData() {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.wsConnections.Clear()
	ctx.wsConnIndex = nil
	ctx.wsFrames.Clear()
}
