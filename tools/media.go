package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

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
		mcp.WithDescription("Take a screenshot of the current page"),
		mcp.WithString("name", mcp.Description("Name of the screenshot"), mcp.Required()),
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
			req := &proto.PageCaptureScreenshot{
				Format: proto.PageCaptureScreenshotFormatPng,
			}
			bin, err := page.Screenshot(false, req)
			if err != nil {
				return toolErr("capture screenshot", err)
			}
			name, err := getStringArg(request.Params.Arguments, "name")
			if err != nil {
				return toolErr("screenshot", err)
			}

			// Always save to file
			cfg := rodCtx.Config()
			path, err := types.SaveOutput(cfg, bin, "screenshot", "png")
			if err != nil {
				return toolErr("save screenshot", err)
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
			name, err := getStringArg(request.Params.Arguments, "name")
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
