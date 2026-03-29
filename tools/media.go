package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	ScreenshotToolKey = "rod_screenshot"
	PdfToolKey        = "rod_pdf"
)

var (
	Screenshot = mcp.NewTool(ScreenshotToolKey,
		mcp.WithDescription("Take a screenshot of the current page, a specific element, or the full scrollable page"),
		mcp.WithString("name", mcp.Description("Name of the screenshot"), mcp.Required()),
		mcp.WithBoolean("full_page", mcp.Description("Capture the full scrollable page instead of just the viewport (default: false)")),
		mcp.WithString("selector", mcp.Description("CSS selector to screenshot a specific element instead of the page")),
		mcp.WithString("ref", mcp.Description("Snapshot ref (e.g. \"42\") to screenshot a specific element")),
		mcp.WithString("save_to", mcp.Description("Save screenshot to this file path (creates parent directories). Screenshot is also returned inline.")),
	)
	Pdf = mcp.NewTool(PdfToolKey,
		mcp.WithDescription("Generate a PDF of the current page and save to the output directory"),
		mcp.WithString("name", mcp.Description("Name or description of the PDF"), mcp.Required()),
	)
)

var (
	ScreenshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("screenshot", err)
			}

			args := request.GetArguments()
			name, err := getStringArg(args, "name")
			if err != nil {
				return toolErr("screenshot", err)
			}

			selector := getOptionalStringArg(args, "selector")
			ref := getOptionalStringArg(args, "ref")
			fullPage := getOptionalBoolArg(args, "full_page", false)
			saveTo := getOptionalStringArg(args, "save_to")

			var bin []byte

			switch {
			case selector != "":
				element, resolveErr := resolveBySelector(page, selector)
				if resolveErr != nil {
					return toolErr("screenshot element", resolveErr)
				}
				bin, err = element.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
				if err != nil {
					return toolErr("capture element screenshot", err)
				}

			case ref != "":
				element, resolveErr := resolveByRef(rodCtx, ref)
				if resolveErr != nil {
					return toolErr("screenshot element", resolveErr)
				}
				bin, err = element.Screenshot(proto.PageCaptureScreenshotFormatPng, 0)
				if err != nil {
					return toolErr("capture element screenshot", err)
				}

			default:
				req := &proto.PageCaptureScreenshot{
					Format: proto.PageCaptureScreenshotFormatPng,
				}
				bin, err = page.Screenshot(fullPage, req)
				if err != nil {
					return toolErr("capture screenshot", err)
				}
			}

			// Save to default output directory
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "screenshot", "png")
			if err != nil {
				return toolErr("save screenshot", err)
			}

			// Also save to custom path if specified
			if saveTo != "" {
				if mkdirErr := os.MkdirAll(filepath.Dir(saveTo), 0o755); mkdirErr != nil {
					return toolErr("create save_to directory", mkdirErr)
				}
				if writeErr := os.WriteFile(saveTo, bin, 0o644); writeErr != nil {
					return toolErr("save screenshot to custom path", writeErr)
				}
				path = saveTo
			}

			// Return file path + optional inline image
			if cfg.ImageResponses != types.ImageResponsesOmit {
				encoded := base64.StdEncoding.EncodeToString(bin)
				return mcp.NewToolResultImage(
					fmt.Sprintf("Screenshot saved: %s (%s)", name, path),
					encoded, "image/png",
				), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Screenshot saved: %s (%s)", name, path)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}

	PdfHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("generate PDF", err)
			}
			reader, err := page.PDF(&proto.PagePrintToPDF{})
			if err != nil {
				return toolErr("generate PDF", err)
			}
			bin, err := io.ReadAll(reader)
			if err != nil {
				return toolErr("read PDF data", err)
			}
			name, err := getStringArg(request.GetArguments(), "name")
			if err != nil {
				return toolErr("generate PDF", err)
			}

			// Always save to file, never return inline (PDFs can't be rendered inline by MCP clients)
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "page", "pdf")
			if err != nil {
				return toolErr("save PDF", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("PDF saved: %s (%s)", name, path)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)
