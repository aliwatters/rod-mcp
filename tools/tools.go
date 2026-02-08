package tools

import (
	"fmt"

	"github.com/charmbracelet/log"
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
	TextTools        = append(CommonTools, Snapshots...)
	TextToolHandlers = utils.MergeMaps(CommonToolHandlers, SnapshotToolHandlers)
)
