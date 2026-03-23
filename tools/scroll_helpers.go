package tools

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/mark3labs/mcp-go/mcp"
)

// scrollToElement scrolls a CSS-selected element into view and returns its resulting position.
func scrollToElement(page *rod.Page, selector string) (*mcp.CallToolResult, error) {
	el, err := page.Element(selector)
	if err != nil {
		return toolErr(fmt.Sprintf("scroll to element %q", selector), err)
	}
	if err = el.ScrollIntoView(); err != nil {
		return toolErr(fmt.Sprintf("scroll element %q into view", selector), err)
	}
	waitDOMStable(page)

	pos, err := page.Eval(`() => ({ x: window.scrollX, y: window.scrollY })`)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Scrolled element %q into view", selector)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Scrolled element %q into view (position: %.0f, %.0f)",
		selector, pos.Value.Get("x").Num(), pos.Value.Get("y").Num())), nil
}

// scrollToCoordinates scrolls the page to an absolute (x, y) position.
// Either x or y (or both) may be supplied; the other defaults to 0.
func scrollToCoordinates(page *rod.Page, args map[string]interface{}) (*mcp.CallToolResult, error) {
	var x, y float64

	if _, hasX := args["x"]; hasX {
		v, err := getFloatArg(args, "x")
		if err != nil {
			return toolErr("scroll", err)
		}
		x = v
	}
	if _, hasY := args["y"]; hasY {
		v, err := getFloatArg(args, "y")
		if err != nil {
			return toolErr("scroll", err)
		}
		y = v
	}

	_, err := page.Eval(`(x, y) => window.scrollTo(x, y)`, x, y)
	if err != nil {
		return toolErr("scroll to position", err)
	}
	waitDOMStable(page)
	return mcp.NewToolResultText(fmt.Sprintf("Scrolled to position (%.0f, %.0f)", x, y)), nil
}

// scrollByDirection scrolls the page in a named direction by the given pixel amount.
func scrollByDirection(page *rod.Page, args map[string]interface{}) (*mcp.CallToolResult, error) {
	direction := getOptionalStringArg(args, "direction")
	if direction == "" {
		direction = "down"
	}
	amount := getOptionalFloatArg(args, "amount", float64(defaultScrollPixels))

	script, err := scrollScript(direction, amount)
	if err != nil {
		return nil, err
	}

	if _, err = page.Eval(script); err != nil {
		return toolErr("scroll "+direction, err)
	}
	waitDOMStable(page)

	pos, evalErr := page.Eval(`() => ({ x: window.scrollX, y: window.scrollY })`)
	if evalErr != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Scrolled %s", direction)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Scrolled %s (position: %.0f, %.0f)",
		direction, pos.Value.Get("x").Num(), pos.Value.Get("y").Num())), nil
}

// scrollScript returns the JS expression that performs a directional scroll.
func scrollScript(direction string, amount float64) (string, error) {
	switch direction {
	case "down":
		return fmt.Sprintf(`() => window.scrollBy(0, %.0f)`, amount), nil
	case "up":
		return fmt.Sprintf(`() => window.scrollBy(0, -%.0f)`, amount), nil
	case "right":
		return fmt.Sprintf(`() => window.scrollBy(%.0f, 0)`, amount), nil
	case "left":
		return fmt.Sprintf(`() => window.scrollBy(-%.0f, 0)`, amount), nil
	case "top":
		return `() => window.scrollTo(0, 0)`, nil
	case "bottom":
		return `() => window.scrollTo(0, document.body.scrollHeight)`, nil
	default:
		return "", fmt.Errorf("invalid direction %q: must be up, down, left, right, top, or bottom", direction)
	}
}
