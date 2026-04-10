package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
)

const (
	CompareScreenshotsToolKey = "rod_compare_screenshots"

	defaultSimilarityThreshold = 0.95
)

var (
	CompareScreenshots = mcp.NewTool(CompareScreenshotsToolKey,
		mcp.WithDescription("Compare two screenshot images pixel-by-pixel. Returns similarity score, pass/fail status, and optional diff image. Useful for visual regression testing."),
		mcp.WithString("baseline", mcp.Description("Path to the baseline image (PNG)"), mcp.Required()),
		mcp.WithString("current", mcp.Description("Path to the current image (PNG)"), mcp.Required()),
		mcp.WithNumber("threshold", mcp.Description("Similarity threshold (0-1) for pass/fail (default: 0.95)")),
		mcp.WithBoolean("save_diff", mcp.Description("Save diff image to output directory (default: false)")),
	)
)

var (
	CompareScreenshotsHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			baselinePath, err := getStringArg(args, "baseline")
			if err != nil {
				return toolErr("compare screenshots", err)
			}
			currentPath, err := getStringArg(args, "current")
			if err != nil {
				return toolErr("compare screenshots", err)
			}
			threshold := getOptionalFloatArg(args, "threshold", defaultSimilarityThreshold)
			saveDiff := getOptionalBoolArg(args, "save_diff", false)

			// Load images
			baselineImg, err := loadPNG(baselinePath)
			if err != nil {
				return toolErr("load baseline", err)
			}
			currentImg, err := loadPNG(currentPath)
			if err != nil {
				return toolErr("load current", err)
			}

			// Compare
			baseBounds := baselineImg.Bounds()
			currBounds := currentImg.Bounds()

			if baseBounds.Dx() != currBounds.Dx() || baseBounds.Dy() != currBounds.Dy() {
				return toolErr("compare screenshots", fmt.Errorf(
					"image dimensions differ: baseline %dx%d, current %dx%d",
					baseBounds.Dx(), baseBounds.Dy(), currBounds.Dx(), currBounds.Dy(),
				))
			}

			width := baseBounds.Dx()
			height := baseBounds.Dy()
			totalPixels := width * height
			var diffPixels int

			// Create diff image
			diffImg := image.NewRGBA(baseBounds)

			for y := baseBounds.Min.Y; y < baseBounds.Max.Y; y++ {
				for x := baseBounds.Min.X; x < baseBounds.Max.X; x++ {
					br, bg, bb, ba := baselineImg.At(x, y).RGBA()
					cr, cg, cb, ca := currentImg.At(x, y).RGBA()

					if pixelsDiffer(br, bg, bb, ba, cr, cg, cb, ca) {
						diffPixels++
						diffImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 180})
					} else {
						// Dim the matching pixels
						r, g, b, _ := baselineImg.At(x, y).RGBA()
						diffImg.Set(x, y, color.RGBA{
							R: uint8(r >> 9), // half brightness
							G: uint8(g >> 9),
							B: uint8(b >> 9),
							A: 255,
						})
					}
				}
			}

			similarity := 1.0
			if totalPixels > 0 {
				similarity = 1.0 - float64(diffPixels)/float64(totalPixels)
			}
			passed := similarity >= threshold

			result := map[string]interface{}{
				"similarity":   math.Round(similarity*10000) / 10000,
				"passed":       passed,
				"diff_pixels":  diffPixels,
				"total_pixels": totalPixels,
				"threshold":    threshold,
			}

			if saveDiff {
				diffPath, saveErr := saveDiffImage(rodCtx.Config(), diffImg)
				if saveErr != nil {
					return toolErr("save diff image", saveErr)
				}
				result["diff_image_path"] = diffPath
			}

			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return toolErr("marshal compare result", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

// loadPNG reads a PNG image from disk.
func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		// Try generic decode as fallback (handles other formats registered)
		_, seekErr := f.Seek(0, 0)
		if seekErr != nil {
			return nil, fmt.Errorf("cannot decode %s as PNG: %w", path, err)
		}
		img, _, decodeErr := image.Decode(f)
		if decodeErr != nil {
			return nil, fmt.Errorf("cannot decode %s: %w", path, err)
		}
		return img, nil
	}
	return img, nil
}

// pixelsDiffer returns true if two pixels differ beyond a small tolerance.
// RGBA values are in the 0-65535 range from Color.RGBA().
func pixelsDiffer(r1, g1, b1, a1, r2, g2, b2, a2 uint32) bool {
	const tolerance = 1280 // ~2% of 65535
	return absDiff(r1, r2) > tolerance ||
		absDiff(g1, g2) > tolerance ||
		absDiff(b1, b2) > tolerance ||
		absDiff(a1, a2) > tolerance
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// saveDiffImage encodes and saves the diff image to the output directory.
func saveDiffImage(cfg types.Config, img *image.RGBA) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encode diff image: %w", err)
	}
	return types.SaveOutput(cfg, buf.Bytes(), "diff", "png")
}
