package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
	"github.com/aliwatters/rod-mcp/types/js"
)

const A11yAuditToolKey = "rod_a11y_audit"

var A11yAudit = mcp.NewTool(A11yAuditToolKey,
	mcp.WithDescription("Audit page accessibility: find elements missing accessible labels, check heading order, and report WCAG issues. Useful for screen reader testing and a11y compliance."),
	mcp.WithString("selector", mcp.Description("Optional CSS selector to scope the audit to a subtree (e.g. 'form#checkout'). Omit to audit the entire page.")),
)

// a11yIssue represents a single accessibility violation.
type a11yIssue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Rule     string `json:"rule"`
	Element  string `json:"element,omitempty"`
	Selector string `json:"selector,omitempty"`
	Message  string `json:"message"`
}

// a11ySummary is the summary section of the audit report.
type a11ySummary struct {
	Errors          int    `json:"errors"`
	Warnings        int    `json:"warnings"`
	ElementsScanned int    `json:"elements_scanned"`
	ElementsNamed   int    `json:"elements_with_names"`
	Coverage        string `json:"coverage"`
}

// a11yReport is the full audit report.
type a11yReport struct {
	Issues  []a11yIssue `json:"issues"`
	Summary a11ySummary `json:"summary"`
}

var A11yAuditHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page, err := rodCtx.ControlledPage()
		if err != nil {
			return toolErr("a11y audit", err)
		}

		selector := getOptionalStringArg(request.GetArguments(), "selector")

		// Use CDP Accessibility.getFullAXTree for the authoritative a11y tree.
		tree, err := proto.AccessibilityGetFullAXTree{}.Call(page)
		if err != nil {
			return toolErr("a11y audit: get accessibility tree", err)
		}
		axIssues, stats := analyzeAXTree(tree.Nodes)

		// Run the JS-based DOM audit for issues not visible in the AX tree.
		jsIssues, err := runJSAudit(page, selector)
		if err != nil {
			return toolErr("a11y audit: DOM scan", err)
		}

		reportJSON, err := mergeAndFormatReport(axIssues, jsIssues, stats)
		if err != nil {
			return toolErr("a11y audit: marshal report", err)
		}

		return mcp.NewToolResultText(reportJSON), nil
	}
	return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
}

// runJSAudit executes the embedded JS audit on the page for the given selector
// and returns the parsed list of a11y issues.
func runJSAudit(page *rod.Page, selector string) ([]a11yIssue, error) {
	domResult, err := page.Eval(buildJSAuditScript(selector))
	if err != nil {
		return nil, err
	}
	var issues []a11yIssue
	if err := json.Unmarshal([]byte(domResult.Value.String()), &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// mergeAndFormatReport merges AX-tree issues and JS-DOM issues, deduplicates
// them, tallies severities, and returns the formatted JSON report string.
func mergeAndFormatReport(axIssues, jsIssues []a11yIssue, stats axStats) (string, error) {
	allIssues := deduplicateIssues(append(jsIssues, axIssues...))

	var errors, warnings int
	for _, issue := range allIssues {
		switch issue.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		}
	}

	coverage := "100%"
	if stats.total > 0 {
		pct := float64(stats.named) / float64(stats.total) * 100
		coverage = fmt.Sprintf("%.1f%%", pct)
	}

	report := a11yReport{
		Issues: allIssues,
		Summary: a11ySummary{
			Errors:          errors,
			Warnings:        warnings,
			ElementsScanned: stats.total,
			ElementsNamed:   stats.named,
			Coverage:        coverage,
		},
	}

	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type axStats struct {
	total int
	named int
}

// axValueStr extracts a string from a gson.JSON AXValue field.
func axValueStr(v *proto.AccessibilityAXValue) string {
	if v == nil {
		return ""
	}
	return v.Value.String()
}

// analyzeAXTree walks the CDP accessibility tree looking for issues.
func analyzeAXTree(nodes []*proto.AccessibilityAXNode) ([]a11yIssue, axStats) {
	var issues []a11yIssue
	var stats axStats

	// Track heading levels for order validation.
	var lastHeadingLevel int

	for _, node := range nodes {
		if node.Ignored {
			continue
		}
		role := axValueStr(node.Role)

		// Skip non-interactive/non-semantic roles.
		switch role {
		case "none", "presentation", "generic", "InlineTextBox", "StaticText",
			"RootWebArea", "LineBreak", "paragraph", "group", "":
			continue
		}

		stats.total++
		name := axValueStr(node.Name)
		if name != "" {
			stats.named++
		}

		// Check heading order.
		if strings.HasPrefix(role, "heading") {
			level := headingLevel(node)
			if lastHeadingLevel > 0 && level > lastHeadingLevel+1 {
				issues = append(issues, a11yIssue{
					Severity: "warning",
					Rule:     "heading-order",
					Message:  fmt.Sprintf("Heading level skipped: h%d → h%d", lastHeadingLevel, level),
				})
			}
			if level > 0 {
				lastHeadingLevel = level
			}
		}
	}

	return issues, stats
}

// headingLevel extracts the heading level from an AX node's properties.
func headingLevel(node *proto.AccessibilityAXNode) int {
	for _, prop := range node.Properties {
		if string(prop.Name) == "level" {
			// gson.JSON wraps the value; try to get it as float64 via JSON round-trip.
			var f float64
			if err := json.Unmarshal([]byte(prop.Value.Value.String()), &f); err == nil {
				return int(f)
			}
		}
	}
	return 0
}

// buildJSAuditScript returns JavaScript that audits the DOM for a11y issues.
// It is a thin wrapper that interpolates the root element expression into the
// embedded a11y_audit.js template (placeholder: %ROOT_EXPR%).
func buildJSAuditScript(selector string) string {
	root := "document.body"
	if selector != "" {
		root = fmt.Sprintf("document.querySelector(%q) || document.body", selector)
	}
	return strings.ReplaceAll(js.A11yAuditJS, "%ROOT_EXPR%", root)
}

// deduplicateIssues removes duplicate issues. Uses selector as the primary
// location key; falls back to element + message for issues without selectors
// (e.g. heading-order, landmark-unique) to avoid collapsing distinct occurrences.
func deduplicateIssues(issues []a11yIssue) []a11yIssue {
	seen := make(map[string]bool)
	var result []a11yIssue
	for _, issue := range issues {
		loc := issue.Selector
		if loc == "" {
			loc = issue.Element
		}
		key := issue.Rule + "|" + loc + "|" + issue.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, issue)
	}
	return result
}
