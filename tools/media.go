package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-rod/rod"
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
		mcp.WithString("ref", mcp.Description("Snapshot ref string (e.g. \"e42\") to screenshot a specific element")),
		mcp.WithString("save_to", mcp.Description("Save screenshot to this path within the output directory (relative paths resolved from output dir, absolute paths must be under it). Creates parent directories.")),
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

			// Validate mutually exclusive targeting modes
			if selector != "" && ref != "" {
				return toolErr("screenshot", fmt.Errorf("provide at most one of 'selector' or 'ref', not both"))
			}

			var bin []byte

			switch {
			case selector != "" || ref != "":
				var element *rod.Element
				if selector != "" {
					element, err = resolveBySelector(page, selector)
				} else {
					element, err = resolveByRef(rodCtx, ref)
				}
				if err != nil {
					return toolErr("screenshot element", err)
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

			cfg := rodCtx.Config()
			var path string

			if saveTo != "" {
				// Resolve save_to within the output directory for safety
				outDir, dirErr := types.ResolveOutputDir(cfg)
				if dirErr != nil {
					return toolErr("resolve output directory", dirErr)
				}
				if !filepath.IsAbs(saveTo) {
					saveTo = filepath.Join(outDir, saveTo)
				}
				cleaned := filepath.Clean(saveTo)
				if !strings.HasPrefix(cleaned, filepath.Clean(outDir)+string(os.PathSeparator)) {
					return toolErr("save_to", fmt.Errorf("path must be within output directory %s", outDir))
				}
				if mkdirErr := os.MkdirAll(filepath.Dir(cleaned), 0o755); mkdirErr != nil {
					return toolErr("create save_to directory", mkdirErr)
				}
				if writeErr := os.WriteFile(cleaned, bin, 0o644); writeErr != nil {
					return toolErr("save screenshot to custom path", writeErr)
				}
				path = cleaned
			} else {
				// Save to default output directory
				path, err = types.SaveOutput(cfg, bin, "screenshot", "png")
				if err != nil {
					return toolErr("save screenshot", err)
				}
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
