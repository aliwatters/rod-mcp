package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/utils"
)

const (
	PressKeyToolKey   = "rod_press"
	TypeToolKey       = "rod_type"
	FileUploadToolKey = "rod_file_upload"
)

const (
	typeTextChunkSize  = 32
	typeTextMaxRetries = 1
)

type textInserter func(string) error

var (
	PressKey = mcp.NewTool(PressKeyToolKey,
		mcp.WithDescription("Press a key on the keyboard, optionally with modifier keys (Ctrl, Shift, Alt, Meta)"),
		mcp.WithString("key", mcp.Description("Name of the key to press or a character to generate, such as `ArrowLeft` or `a`"), mcp.Required()),
		mcp.WithArray("modifiers", mcp.Description("Modifier keys to hold: control, shift, alt, meta"), mcp.Items(map[string]interface{}{"type": "string"})),
	)
	Type = mcp.NewTool(TypeToolKey,
		mcp.WithDescription("Type a string of text character by character with realistic keyboard events. Better than rod_fill for React controlled inputs. Use this instead of multiple rod_press calls."),
		mcp.WithString("text", mcp.Description("The text string to type"), mcp.Required()),
		mcp.WithNumber("delay", mcp.Description("Milliseconds between keystrokes (default: 50). Use higher values for React apps that need time to process state updates.")),
	)
	FileUpload = mcp.NewTool(FileUploadToolKey,
		mcp.WithDescription("Upload file(s) to a file input element"),
		mcp.WithString("selector", mcp.Description("CSS selector of the file input element")),
		mcp.WithString("ref", mcp.Description("Element reference from the page snapshot")),
		mcp.WithArray("paths", mcp.Description("File paths to upload"), mcp.Items(map[string]interface{}{"type": "string"}), mcp.Required()),
	)
)

var (
	PressKeyHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("press key", err)
			}
			args := request.GetArguments()
			keyStr, err := getStringArg(args, "key")
			if err != nil {
				return toolErr("press key", err)
			}
			key, err := parseKey(keyStr)
			if err != nil {
				return toolErr("parse key "+keyStr, err)
			}

			// Parse modifier keys
			modifiers, err := utils.OptionalStringArrayParam(request, "modifiers")
			if err != nil {
				return toolErr("parse modifiers", err)
			}

			if len(modifiers) > 0 {
				// Press modifiers down
				for _, mod := range modifiers {
					modKey, modErr := parseModifier(mod)
					if modErr != nil {
						return toolErr("press modifier", modErr)
					}
					if err = page.Keyboard.Press(modKey); err != nil {
						return toolErr("press modifier "+mod, err)
					}
				}
				// Type the key
				if err = page.Keyboard.Type(key); err != nil {
					return toolErr("type key "+keyStr, err)
				}
				// Release modifiers in reverse order
				for i := len(modifiers) - 1; i >= 0; i-- {
					modKey, _ := parseModifier(modifiers[i])
					if err = page.Keyboard.Release(modKey); err != nil {
						return toolErr("release modifier "+modifiers[i], err)
					}
				}
			} else {
				if err = page.Keyboard.Press(key); err != nil {
					return toolErr("press key "+keyStr, err)
				}
			}

			waitDOMStable(page)
			desc := keyStr
			if len(modifiers) > 0 {
				desc = fmt.Sprintf("%s+%s", joinModifiers(modifiers), keyStr)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Press key %s successfully", desc)), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	TypeHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("type text", err)
			}
			text, err := getStringArg(request.GetArguments(), "text")
			if err != nil {
				return toolErr("type text", err)
			}
			delayMs := getOptionalFloatArg(request.GetArguments(), "delay", 50)
			// Clamp delay to a reasonable range (0-5000ms).
			if delayMs < 0 {
				delayMs = 0
			} else if delayMs > 5000 {
				delayMs = 5000
			}
			delay := time.Duration(delayMs * float64(time.Millisecond))

			if err := typeTextInChunks(ctx, text, delay, page.InsertText); err != nil {
				return toolErr("type text", err)
			}

			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Typed %d characters successfully", len([]rune(text)))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	FileUploadHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			selector, _ := request.GetArguments()["selector"].(string)
			ref, _ := request.GetArguments()["ref"].(string)

			if selector == "" && ref == "" {
				return nil, errors.New("either 'selector' or 'ref' is required")
			}

			paths, err := utils.OptionalStringArrayParam(request, "paths")
			if err != nil {
				return toolErr("parse file paths", err)
			}
			if len(paths) == 0 {
				return nil, errors.New("at least one file path is required")
			}

			page, err := rodCtx.ControlledPage()
			if err != nil {
				return toolErr("upload files", err)
			}

			var element *rod.Element
			if ref != "" {
				snapshot, err := rodCtx.LatestSnapshot()
				if err != nil {
					return toolErr("upload files", err)
				}
				element, err = snapshot.LocatorInFrame(ref)
				if err != nil {
					return toolErr("upload files", err)
				}
			} else {
				element, err = page.Timeout(defaultSelectorTimeout).Element(selector)
				if err != nil {
					return toolErr("find file input "+selector, err)
				}
			}

			if err = element.SetFiles(paths); err != nil {
				return toolErr("upload files", err)
			}

			waitDOMStable(page)
			return mcp.NewToolResultText(fmt.Sprintf("Uploaded %d file(s) successfully", len(paths))), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
	}
)

func typeTextInChunks(ctx context.Context, text string, delay time.Duration, insert textInserter) error {
	for _, chunk := range splitTextChunks(text, typeTextChunkSize) {
		var err error
		for attempt := 0; attempt <= typeTextMaxRetries; attempt++ {
			err = insert(chunk)
			if err == nil {
				break
			}
			if attempt == typeTextMaxRetries || !isRetryableInputError(err) {
				return fmt.Errorf("insert text chunk %q: %w", chunk, err)
			}
		}

		if delay > 0 {
			wait := delay * time.Duration(len([]rune(chunk)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return nil
}

func splitTextChunks(text string, maxRunes int) []string {
	if text == "" {
		return nil
	}
	if maxRunes <= 0 {
		return []string{text}
	}

	chunks := make([]string, 0, (len([]rune(text))+maxRunes-1)/maxRunes)
	var builder strings.Builder
	count := 0
	for _, r := range text {
		builder.WriteRune(r)
		count++
		if count == maxRunes {
			chunks = append(chunks, builder.String())
			builder.Reset()
			count = 0
		}
	}
	if builder.Len() > 0 {
		chunks = append(chunks, builder.String())
	}
	return chunks
}

func isRetryableInputError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"eof",
		"use of closed network connection",
		"connection reset by peer",
		"connection is closed",
		"broken pipe",
		"session closed",
		"target closed",
		"websocket: close",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
