package types

import (
	"fmt"
	"strings"

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

// eventCollector owns the ring buffers for console messages, network requests,
// and WebSocket tracking. All methods assume the caller holds Context.eventsLock.
type eventCollector struct {
	consoleMessages *RingBuffer[ConsoleMessage]
	networkRequests *RingBuffer[NetworkRequest]
	// pendingRequests tracks in-flight requests by ID for response correlation.
	// Values are indices into the networkRequests ring buffer's internal items slice;
	// they remain valid because the ring buffer never shifts elements.
	pendingRequests map[string]int
	// WebSocket tracking
	wsConnections *RingBuffer[WebSocketConnection]
	wsConnIndex   map[string]int // requestID → internal ring-buffer index in wsConnections
	wsFrames      *RingBuffer[WebSocketFrame]
}

// newEventCollector creates an eventCollector with default-sized ring buffers.
func newEventCollector() *eventCollector {
	return &eventCollector{
		consoleMessages: NewRingBuffer[ConsoleMessage](maxConsoleMessages),
		networkRequests: NewRingBuffer[NetworkRequest](maxNetworkRequests),
		wsConnections:   NewRingBuffer[WebSocketConnection](maxWSConnections),
		wsFrames:        NewRingBuffer[WebSocketFrame](maxWSFrames),
	}
}

// AddConsoleMessage records a console message. Must be called with eventsLock held.
func (ec *eventCollector) AddConsoleMessage(msg ConsoleMessage) {
	ec.consoleMessages.Add(msg)
}

// AddNetworkRequest records a new outgoing request and tracks it as pending.
// Must be called with eventsLock held.
func (ec *eventCollector) AddNetworkRequest(req NetworkRequest) {
	if ec.pendingRequests == nil {
		ec.pendingRequests = make(map[string]int)
	}
	idx := ec.networkRequests.AddWithIndex(req)
	ec.pendingRequests[req.RequestID] = idx
}

// CompleteNetworkRequest updates a pending request with its response status and type.
// Must be called with eventsLock held.
func (ec *eventCollector) CompleteNetworkRequest(requestID string, status int, typ string) {
	if idx, ok := ec.pendingRequests[requestID]; ok {
		ec.networkRequests.UpdateAt(idx, func(req *NetworkRequest) {
			req.Status = status
			req.Type = typ
		})
		delete(ec.pendingRequests, requestID)
	}
}

// AddWSConnection records a new WebSocket connection. Must be called with eventsLock held.
func (ec *eventCollector) AddWSConnection(conn WebSocketConnection) {
	if ec.wsConnIndex == nil {
		ec.wsConnIndex = make(map[string]int)
	}
	idx := ec.wsConnections.AddWithIndex(conn)
	ec.wsConnIndex[conn.RequestID] = idx
}

// UpdateWSConnection calls fn on the WebSocket connection identified by requestID.
// Must be called with eventsLock held.
func (ec *eventCollector) UpdateWSConnection(requestID string, fn func(*WebSocketConnection)) {
	if idx, ok := ec.wsConnIndex[requestID]; ok {
		ec.wsConnections.UpdateAt(idx, fn)
	}
}

// AppendWSFrame adds a WebSocket frame to the ring buffer.
// Must be called with eventsLock held.
func (ec *eventCollector) AppendWSFrame(url, direction string, frame *proto.NetworkWebSocketFrame) {
	ec.wsFrames.Add(WebSocketFrame{
		URL:         url,
		Direction:   direction,
		PayloadData: frame.PayloadData,
		Opcode:      int(frame.Opcode),
	})
}

// ConsoleMessages returns captured console messages, optionally filtered by level.
// If clear is true, the buffer is emptied after returning.
// Must be called with eventsLock held.
func (ec *eventCollector) ConsoleMessages(filterLevel string, clear bool) []ConsoleMessage {
	all := ec.consoleMessages.Slice()
	result := filterSlice(all, func(msg ConsoleMessage) bool {
		return filterLevel == "" || msg.Level == filterLevel
	})

	if clear {
		if filterLevel == "" {
			ec.consoleMessages.Clear()
		} else {
			// Rebuild the buffer keeping only non-matching messages.
			fresh := NewRingBuffer[ConsoleMessage](maxConsoleMessages)
			for _, msg := range filterSlice(all, func(msg ConsoleMessage) bool {
				return msg.Level != filterLevel
			}) {
				fresh.Add(msg)
			}
			ec.consoleMessages = fresh
		}
	}

	return result
}

// NetworkRequests returns captured network requests, optionally filtered by URL pattern and method.
// If clear is true, the buffer is emptied after returning.
// Must be called with eventsLock held.
func (ec *eventCollector) NetworkRequests(filterURL, filterMethod string, clear bool) []NetworkRequest {
	result := filterSlice(ec.networkRequests.Slice(), func(req NetworkRequest) bool {
		if filterURL != "" && !strings.Contains(req.URL, filterURL) {
			return false
		}
		if filterMethod != "" && !strings.EqualFold(req.Method, filterMethod) {
			return false
		}
		return true
	})

	if clear {
		ec.networkRequests.Clear()
		ec.pendingRequests = nil
	}

	return result
}

// GetRequestID returns the CDP request ID for a network request at the given index.
// Must be called with eventsLock held.
func (ec *eventCollector) GetRequestID(index int) (string, error) {
	all := ec.networkRequests.Slice()
	if len(all) == 0 {
		return "", fmt.Errorf("no network requests captured")
	}
	if index < 0 || index >= len(all) {
		return "", fmt.Errorf("request index %d out of range (0-%d)", index, len(all)-1)
	}
	return all[index].RequestID, nil
}

// WebSocketConnections returns tracked WebSocket connections, optionally filtered by URL.
// Must be called with eventsLock held.
func (ec *eventCollector) WebSocketConnections(urlFilter string) []WebSocketConnection {
	return filterSlice(ec.wsConnections.Slice(), func(conn WebSocketConnection) bool {
		return urlFilter == "" || strings.Contains(conn.URL, urlFilter)
	})
}

// WebSocketFrames returns captured WebSocket frames, optionally filtered by URL and direction.
// Must be called with eventsLock held.
func (ec *eventCollector) WebSocketFrames(urlFilter, direction string) []WebSocketFrame {
	return filterSlice(ec.wsFrames.Slice(), func(f WebSocketFrame) bool {
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
// Must be called with eventsLock held.
func (ec *eventCollector) ClearWebSocketData() {
	ec.wsConnections.Clear()
	ec.wsConnIndex = nil
	ec.wsFrames.Clear()
}
