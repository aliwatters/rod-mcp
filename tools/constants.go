package tools

import "time"

// Response and content size limits.
const (
	// defaultMaxContentLength is the max characters returned for HTML or response body content.
	defaultMaxContentLength = 100000

	// wsPayloadPreviewLen is the max length for WebSocket frame payload previews.
	wsPayloadPreviewLen = 200
)

// Timeout and timing defaults.
const (
	// defaultWaitStableDur is how long the DOM must remain unchanged to be considered stable.
	defaultWaitStableDur = 1 * time.Second

	// defaultDomDiff is the fraction of DOM change allowed during stability wait.
	defaultDomDiff = 0.2

	// defaultDialogTimeout is the max time to wait for a browser dialog to appear.
	defaultDialogTimeout = 30 * time.Second

	// defaultNavigationTimeout is the max time to wait for a page navigation to complete.
	defaultNavigationTimeout = 30 * time.Second

	// defaultWaitTimeoutMs is the default timeout for wait operations in milliseconds.
	defaultWaitTimeoutMs = 30000.0

	// defaultLoginTimeoutMs is the default timeout for login success verification in milliseconds.
	defaultLoginTimeoutMs = 15000.0

	// defaultFormContainerTimeout is the max time to wait for a modal form container to appear.
	defaultFormContainerTimeout = 5 * time.Second
)

// Default parameter values for tool options.
const (
	// defaultScrollPixels is the default pixel count for scroll operations.
	defaultScrollPixels = 500

	// defaultMaxWSFrames is the default number of WebSocket frames to return.
	defaultMaxWSFrames = 100

	// defaultHTTPStatus is the default status code for intercepted mock responses.
	defaultHTTPStatus = 200
)
