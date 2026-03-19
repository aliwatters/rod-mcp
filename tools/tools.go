package tools

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

type ToolHandler = func(rodCtx *types.Context) server.ToolHandlerFunc

// toolError logs the error and returns a formatted error for the MCP response.
func toolError(action string, err error) error {
	log.Errorf("Failed to %s: %s", action, err)
	return fmt.Errorf("Failed to %s: %s", action, err)
}

var (
	CommonTools = []mcp.Tool{
		Configure,
		Navigation,
		GoBack,
		GoForward,
		Reload,
		PressKey,
		Screenshot,
		Pdf,
		Evaluate,
		CloseBrowser,
		SetHeaders,
		Resize,
		HandleDialog,
		TabNew,
		TabList,
		TabSelect,
		TabClose,
		WaitFor,
		ConsoleMessages,
		FileUpload,
		NetworkRequests,
		Scroll,
		HTML,
		Coverage,
		Cookies,
		Storage,
		Permissions,
		Intercept,
		ResponseBody,
		WebSocket,
	}
	CommonToolHandlers = map[string]ToolHandler{
		ConfigureToolKey:       ConfigureHandler,
		NavigationToolKey:      NavigationHandler,
		GoBackToolKey:          GoBackHandler,
		GoForwardToolKey:       GoForwardHandler,
		ReloadToolKey:          ReloadHandler,
		PressKeyToolKey:        PressKeyHandler,
		ScreenshotToolKey:      ScreenshotHandler,
		PdfToolKey:             PdfHandler,
		EvaluateToolKey:        EvaluateHandler,
		CloseBrowserToolKey:    CloseBrowserHandler,
		SetHeadersToolKey:      SetHeadersHandler,
		ResizeToolKey:          ResizeHandler,
		HandleDialogToolKey:    HandleDialogHandler,
		TabNewToolKey:          TabNewHandler,
		TabListToolKey:         TabListHandler,
		TabSelectToolKey:       TabSelectHandler,
		TabCloseToolKey:        TabCloseHandler,
		WaitForToolKey:         WaitForHandler,
		ConsoleMessagesToolKey: ConsoleMessagesHandler,
		FileUploadToolKey:      FileUploadHandler,
		NetworkRequestsToolKey: NetworkRequestsHandler,
		ScrollToolKey:          ScrollHandler,
		HTMLToolKey:            HTMLHandler,
		CoverageToolKey:        CoverageHandler,
		CookiesToolKey:         CookiesHandler,
		StorageToolKey:         StorageHandler,
		PermissionsToolKey:     PermissionsHandler,
		InterceptToolKey:       InterceptHandler,
		ResponseBodyToolKey:    ResponseBodyHandler,
		WebSocketToolKey:       WebSocketHandler,
	}
)

var (
	TextTools        = append(CommonTools, Snapshots...)
	TextToolHandlers = utils.MergeMaps(CommonToolHandlers, SnapshotToolHandlers)

	VisionTools            = append(CommonTools, VisionToolDefs...)
	VisionCombinedHandlers = utils.MergeMaps(CommonToolHandlers, VisionToolHandlers)
)
