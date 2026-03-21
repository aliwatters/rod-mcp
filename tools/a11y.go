package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aliwatters/rod-mcp/types"
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

		// Also get DOM info for elements missing a11y names via JavaScript.
		jsAudit := buildJSAuditScript(selector)
		domResult, err := page.Eval(jsAudit)
		if err != nil {
			return toolErr("a11y audit: DOM scan", err)
		}

		var domIssues []a11yIssue
		if err := json.Unmarshal([]byte(domResult.Value.String()), &domIssues); err != nil {
			return toolErr("a11y audit: parse DOM results", err)
		}

		// Analyze the CDP accessibility tree for additional issues.
		axIssues, axStats := analyzeAXTree(tree.Nodes)

		// Merge issues (DOM-based + AX tree-based), dedup by rule+selector.
		allIssues := deduplicateIssues(append(domIssues, axIssues...))

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
		if axStats.total > 0 {
			pct := float64(axStats.named) / float64(axStats.total) * 100
			coverage = fmt.Sprintf("%.1f%%", pct)
		}

		report := a11yReport{
			Issues: allIssues,
			Summary: a11ySummary{
				Errors:          errors,
				Warnings:        warnings,
				ElementsScanned: axStats.total,
				ElementsNamed:   axStats.named,
				Coverage:        coverage,
			},
		}

		reportJSON, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return toolErr("a11y audit: marshal report", err)
		}

		return mcp.NewToolResultText(string(reportJSON)), nil
	}
	return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WithSnapshot: false})
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
func buildJSAuditScript(selector string) string {
	root := "document.body"
	if selector != "" {
		root = fmt.Sprintf("document.querySelector(%q) || document.body", selector)
	}

	return fmt.Sprintf(`function() {
		const root = %s;
		const issues = [];

		// Rule: img-alt — images without alt text
		root.querySelectorAll('img').forEach(img => {
			if (!img.hasAttribute('alt')) {
				issues.push({
					severity: 'error',
					rule: 'img-alt',
					element: img.outerHTML.substring(0, 120),
					selector: cssPath(img),
					message: 'Image missing alt text'
				});
			}
		});

		// Rule: button-name — buttons without accessible names
		root.querySelectorAll('button, [role="button"]').forEach(btn => {
			const name = btn.textContent.trim() ||
				btn.getAttribute('aria-label') ||
				btn.getAttribute('aria-labelledby') ||
				btn.getAttribute('title');
			if (!name) {
				issues.push({
					severity: 'error',
					rule: 'button-name',
					element: btn.outerHTML.substring(0, 120),
					selector: cssPath(btn),
					message: 'Button has no accessible name (no text, aria-label, or aria-labelledby)'
				});
			}
		});

		// Rule: input-label — form inputs without labels
		root.querySelectorAll('input:not([type="hidden"]):not([type="submit"]):not([type="button"]):not([type="reset"]), select, textarea').forEach(input => {
			const id = input.id;
			const hasLabel = (id && root.querySelector('label[for="' + id + '"]')) ||
				input.closest('label') ||
				input.getAttribute('aria-label') ||
				input.getAttribute('aria-labelledby') ||
				input.getAttribute('title');
			if (!hasLabel) {
				issues.push({
					severity: 'error',
					rule: 'input-label',
					element: input.outerHTML.substring(0, 120),
					selector: cssPath(input),
					message: 'Form input missing associated label, aria-label, aria-labelledby, or title'
				});
			}
		});

		// Rule: link-name — links without accessible names
		root.querySelectorAll('a[href]').forEach(link => {
			const name = link.textContent.trim() ||
				link.getAttribute('aria-label') ||
				link.getAttribute('aria-labelledby') ||
				link.getAttribute('title');
			if (!name) {
				// Check if link contains an image with alt
				const img = link.querySelector('img[alt]');
				if (!img || !img.alt.trim()) {
					issues.push({
						severity: 'error',
						rule: 'link-name',
						element: link.outerHTML.substring(0, 120),
						selector: cssPath(link),
						message: 'Link has no accessible name'
					});
				}
			}
		});

		// Rule: landmark-unique — duplicate landmark roles without unique labels
		const landmarks = {};
		root.querySelectorAll('[role="banner"], [role="navigation"], [role="main"], [role="contentinfo"], [role="complementary"], [role="search"], nav, main, header, footer, aside').forEach(el => {
			const role = el.getAttribute('role') || el.tagName.toLowerCase();
			const label = el.getAttribute('aria-label') || el.getAttribute('aria-labelledby') || '';
			const key = role + ':' + label;
			if (!landmarks[key]) {
				landmarks[key] = [];
			}
			landmarks[key].push(el);
		});
		for (const [key, elements] of Object.entries(landmarks)) {
			if (elements.length > 1) {
				const [role, label] = key.split(':');
				if (!label) {
					issues.push({
						severity: 'warning',
						rule: 'landmark-unique',
						message: elements.length + ' ' + role + ' landmarks without unique aria-label to distinguish them',
					});
				}
			}
		}

		function cssPath(el) {
			const parts = [];
			while (el && el.nodeType === 1) {
				let part = el.tagName.toLowerCase();
				if (el.id) {
					part += '#' + el.id;
					parts.unshift(part);
					break;
				}
				const cls = Array.from(el.classList).filter(c => !c.includes(':')).slice(0, 2).join('.');
				if (cls) part += '.' + cls;
				parts.unshift(part);
				el = el.parentElement;
			}
			return parts.join(' > ');
		}

		return JSON.stringify(issues);
	}`, root)
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
