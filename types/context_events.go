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
		if len(ctx.consoleMessages) >= maxConsoleMessages {
			drop := maxConsoleMessages / 10
			copy(ctx.consoleMessages, ctx.consoleMessages[drop:])
			ctx.consoleMessages = ctx.consoleMessages[:len(ctx.consoleMessages)-drop]
		}
		ctx.consoleMessages = append(ctx.consoleMessages, ConsoleMessage{
			Level: string(e.Type),
			Text:  text,
		})
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkRequestWillBeSent) {
		ctx.stateLock.Lock()
		if ctx.pendingRequests == nil {
			ctx.pendingRequests = make(map[string]int)
		}
		if len(ctx.networkRequests) >= maxNetworkRequests {
			drop := maxNetworkRequests / 10
			// Shift pendingRequests indices down.
			for k, v := range ctx.pendingRequests {
				if v < drop {
					delete(ctx.pendingRequests, k)
				} else {
					ctx.pendingRequests[k] = v - drop
				}
			}
			copy(ctx.networkRequests, ctx.networkRequests[drop:])
			ctx.networkRequests = ctx.networkRequests[:len(ctx.networkRequests)-drop]
		}
		idx := len(ctx.networkRequests)
		ctx.networkRequests = append(ctx.networkRequests, NetworkRequest{
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
			ctx.networkRequests[idx].Status = e.Response.Status
			ctx.networkRequests[idx].Type = string(e.Type)
			delete(ctx.pendingRequests, string(e.RequestID))
		}
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketCreated) {
		ctx.stateLock.Lock()
		if ctx.wsConnIndex == nil {
			ctx.wsConnIndex = make(map[string]int)
		}
		ctx.wsConnIndex[string(e.RequestID)] = len(ctx.wsConnections)
		ctx.wsConnections = append(ctx.wsConnections, WebSocketConnection{
			RequestID: string(e.RequestID),
			URL:       e.URL,
		})
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketFrameSent) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.wsConnIndex[string(e.RequestID)]; ok {
			ctx.wsConnections[idx].SentCount++
			ctx.appendWSFrame(ctx.wsConnections[idx].URL, "sent", e.Response)
		}
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketFrameReceived) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.wsConnIndex[string(e.RequestID)]; ok {
			ctx.wsConnections[idx].ReceivedCount++
			ctx.appendWSFrame(ctx.wsConnections[idx].URL, "received", e.Response)
		}
		ctx.stateLock.Unlock()
	}, func(e *proto.NetworkWebSocketClosed) {
		ctx.stateLock.Lock()
		if idx, ok := ctx.wsConnIndex[string(e.RequestID)]; ok {
			ctx.wsConnections[idx].Closed = true
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

	var result []ConsoleMessage
	for _, msg := range ctx.consoleMessages {
		if filterLevel == "" || msg.Level == filterLevel {
			result = append(result, msg)
		}
	}

	if clear {
		if filterLevel == "" {
			ctx.consoleMessages = nil
		} else {
			// Only clear matching messages
			var remaining []ConsoleMessage
			for _, msg := range ctx.consoleMessages {
				if msg.Level != filterLevel {
					remaining = append(remaining, msg)
				}
			}
			ctx.consoleMessages = remaining
		}
	}

	return result
}

// NetworkRequests returns captured network requests, optionally filtered by URL pattern and method.
// If clear is true, the buffer is emptied after returning.
func (ctx *Context) NetworkRequests(filterURL, filterMethod string, clear bool) []NetworkRequest {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var result []NetworkRequest
	for _, req := range ctx.networkRequests {
		if filterURL != "" && !strings.Contains(req.URL, filterURL) {
			continue
		}
		if filterMethod != "" && !strings.EqualFold(req.Method, filterMethod) {
			continue
		}
		result = append(result, req)
	}

	if clear {
		ctx.networkRequests = nil
		ctx.pendingRequests = nil
	}

	return result
}

// appendWSFrame adds a WebSocket frame, dropping the oldest if the buffer is full.
// Must be called with stateLock held.
func (ctx *Context) appendWSFrame(url, direction string, frame *proto.NetworkWebSocketFrame) {
	if len(ctx.wsFrames) >= maxWSFrames {
		// Drop oldest 10% to avoid frequent shifting
		drop := maxWSFrames / 10
		copy(ctx.wsFrames, ctx.wsFrames[drop:])
		ctx.wsFrames = ctx.wsFrames[:len(ctx.wsFrames)-drop]
	}
	ctx.wsFrames = append(ctx.wsFrames, WebSocketFrame{
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

	if len(ctx.networkRequests) == 0 {
		return "", fmt.Errorf("no network requests captured")
	}
	if index < 0 || index >= len(ctx.networkRequests) {
		return "", fmt.Errorf("request index %d out of range (0-%d)", index, len(ctx.networkRequests)-1)
	}
	return ctx.networkRequests[index].RequestID, nil
}

// WebSocketConnections returns tracked WebSocket connections, optionally filtered by URL.
func (ctx *Context) WebSocketConnections(urlFilter string) []WebSocketConnection {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var result []WebSocketConnection
	for _, conn := range ctx.wsConnections {
		if urlFilter != "" && !strings.Contains(conn.URL, urlFilter) {
			continue
		}
		result = append(result, conn)
	}
	return result
}

// WebSocketFrames returns captured WebSocket frames, optionally filtered by URL and direction.
func (ctx *Context) WebSocketFrames(urlFilter, direction string) []WebSocketFrame {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()

	var result []WebSocketFrame
	for _, f := range ctx.wsFrames {
		if urlFilter != "" && !strings.Contains(f.URL, urlFilter) {
			continue
		}
		if direction != "" && f.Direction != direction {
			continue
		}
		result = append(result, f)
	}
	return result
}

// ClearWebSocketData clears all WebSocket connections and frames.
func (ctx *Context) ClearWebSocketData() {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.wsConnections = nil
	ctx.wsConnIndex = nil
	ctx.wsFrames = nil
}
