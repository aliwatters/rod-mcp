# PR Ready Retrospective: Verification Success Rate Analysis

# ⚠️ SILENT MODE - UNATTENDED EXECUTION ⚠️

**CRITICAL**: This workflow runs UNATTENDED without user intervention.

**Agent Requirements:**

1. Execute ALL commands immediately without pausing
2. Execute `prompt` transitions immediately
3. Do NOT provide status summaries
4. Do NOT wait for user confirmation
5. Do NOT stop between phases
6. Let command output speak for itself
7. Only stop on errors or completion

**Expected User Intervention**: ZERO (after initial command)

---

**⚠️ MANDATORY COMPLIANCE**: This sub-prompt is part of the PR Ready Retrospective phase and must be
executed completely.

## Purpose

Critically analyze why PRs don't reach verified state and create improvement issues for every
failure.

**Success Criteria**: >80% of PRs must reach verified state. Any PR that doesn't is a workflow
failure requiring root cause analysis.

## Prerequisites

This sub-prompt requires:

- PR list from workflow state (`ax prready get | grep "^PRS_PROCESSED="`)
- GitHub CLI (`gh`) for PR status checks
- Analysis results directory for metrics tracking

## Core Analysis Steps

### 5.5. CRITICAL: Verification Success Rate Analysis

**Purpose**: Zero tolerance for PRs that don't reach verified state. Every failure gets
investigated.

**What this measures**:

- How many PRs actually reached "verified" label vs total processed
- Success rate against >80% target
- Clear indication of workflow effectiveness

```bash
# Get processed PRs from state
PRS_PROCESSED=$(ax prready get | grep "^PRS_PROCESSED=" | cut -d= -f2)

# Calculate verification success rate
PR_COUNT=$(echo "$PRS_PROCESSED" | tr ',' '\n' | grep -v '^$' | wc -l | tr -d ' ')

# Fix: grep -c returns at least 1 even for empty input, count properly
if [ -z "$PRS_PROCESSED" ] || [ "$PRS_PROCESSED" = "0" ]; then
    PR_COUNT=0
fi

echo "📊 Verification Success Rate Analysis"
echo "═══════════════════════════════════════"
echo "Target: >80% of PRs reach verified state"
echo ""

# Declare associative array at top level to avoid scope issues
declare -A FAILURE_PATTERNS

if [ "$PR_COUNT" -gt 0 ]; then
    # Check each PR's current label state
    VERIFIED_COUNT=0
    FAILED_PRS=""

    for PR_NUM in $(echo "$PRS_PROCESSED" | tr ',' '\n'); do
        if [ -z "$PR_NUM" ]; then
            continue
        fi

        # Check if PR has verified label
        HAS_VERIFIED=$(ax pr view "$PR_NUM" --json labels --jq '.labels[].name' | grep -c "^verified$" || true)

        if [ "$HAS_VERIFIED" -gt 0 ]; then
            VERIFIED_COUNT=$((VERIFIED_COUNT + 1))
            echo "✅ PR #$PR_NUM: verified"
        else
            FAILED_PRS="${FAILED_PRS}${PR_NUM},"
            echo "❌ PR #$PR_NUM: NOT verified"
        fi
    done

    # Calculate success rate
    SUCCESS_RATE=$(( VERIFIED_COUNT * 100 / PR_COUNT ))

    echo ""
    echo "Results:"
    echo "  Verified: $VERIFIED_COUNT / $PR_COUNT ($SUCCESS_RATE%)"
    echo "  Target: >80%"

    if [ "$SUCCESS_RATE" -ge 80 ]; then
        echo "  Status: ✅ PASS"
    else
        echo "  Status: ❌ FAIL - Below target"
    fi
else
    echo "ℹ️  No PRs processed in this session"
    SUCCESS_RATE=0
    VERIFIED_COUNT=0
    FAILED_PRS=""
fi
```

### 5.6. Root Cause Analysis: Why Did Each PR Fail?

**Purpose**: Investigate EVERY PR that didn't reach verified state and create improvement issue.

```bash
# Skip if all PRs verified
if [ -z "$FAILED_PRS" ]; then
    echo ""
    echo "✅ All PRs reached verified state - no failures to analyze"
else
    echo ""
    echo "🔍 Root Cause Analysis: Failed PRs"
    echo "═══════════════════════════════════"

    for PR_NUM in $(echo "$FAILED_PRS" | tr ',' '\n'); do
        if [ -z "$PR_NUM" ]; then
            continue
        fi

        echo ""
        echo "Analyzing PR #$PR_NUM..."

        # Get PR state
        PR_STATE=$(ax pr view "$PR_NUM" --json state,isDraft,mergeable --jq '{state, isDraft, mergeable}')
        IS_DRAFT=$(echo "$PR_STATE" | grep -o '"isDraft":[^,}]*' | cut -d: -f2)
        IS_MERGEABLE=$(echo "$PR_STATE" | grep -o '"mergeable":"[^"]*"' | cut -d'"' -f4)
        PR_STATUS=$(echo "$PR_STATE" | grep -o '"state":"[^"]*"' | cut -d'"' -f4)

        # Check CI status (using 2>&1 to capture errors for debugging)
        CI_OUTPUT=$(gh pr checks "$PR_NUM" 2>&1)
        CI_EXIT=$?
        CI_FAILED=$(echo "$CI_OUTPUT" | grep -E '(fail[[:space:]]*$|✗[[:space:]]*fail)' || true)

        # Check for unresolved comments (using 2>&1 to capture errors)
        COMMENT_COUNT=$(gh api "repos/{owner}/{repo}/pulls/$PR_NUM/comments" --jq 'length' 2>&1 || echo "0")

        # Determine failure reason
        FAILURE_REASON=""
        if [ "$IS_DRAFT" = "true" ]; then
            FAILURE_REASON="draft_status"
            FAILURE_PATTERNS["draft_status"]=$((${FAILURE_PATTERNS["draft_status"]:-0} + 1))
            echo "  Reason: Still in draft state"
        elif [ "$IS_MERGEABLE" = "CONFLICTING" ]; then
            FAILURE_REASON="merge_conflicts"
            FAILURE_PATTERNS["merge_conflicts"]=$((${FAILURE_PATTERNS["merge_conflicts"]:-0} + 1))
            echo "  Reason: Merge conflicts"
        elif [ "$CI_EXIT" -ne 0 ] || [ -n "$CI_FAILED" ]; then
            FAILURE_REASON="ci_failure"
            FAILURE_PATTERNS["ci_failure"]=$((${FAILURE_PATTERNS["ci_failure"]:-0} + 1))
            echo "  Reason: CI checks failing"
        elif [ "$COMMENT_COUNT" -gt 0 ]; then
            FAILURE_REASON="unaddressed_comments"
            FAILURE_PATTERNS["unaddressed_comments"]=$((${FAILURE_PATTERNS["unaddressed_comments"]:-0} + 1))
            echo "  Reason: Unaddressed review comments"
        elif [ "$PR_STATUS" = "CLOSED" ]; then
            FAILURE_REASON="closed_prematurely"
            FAILURE_PATTERNS["closed_prematurely"]=$((${FAILURE_PATTERNS["closed_prematurely"]:-0} + 1))
            echo "  Reason: PR closed before verification"
        else
            FAILURE_REASON="unknown"
            FAILURE_PATTERNS["unknown"]=$((${FAILURE_PATTERNS["unknown"]:-0} + 1))
            echo "  Reason: Unknown (requires manual investigation)"
        fi

        # Create improvement issue for this failure
        ISSUE_TITLE="friction: PR #$PR_NUM failed to reach verified state ($FAILURE_REASON)"
        ISSUE_BODY="## Problem

PR #$PR_NUM was processed by pr-ready cycle but did not reach verified state.

**Failure Reason**: $FAILURE_REASON

**PR State**:
- Status: $PR_STATUS
- Draft: $IS_DRAFT
- Mergeable: $IS_MERGEABLE
- Comments: $COMMENT_COUNT

**CI Status**:
\`\`\`
$CI_OUTPUT
\`\`\`

## Failure Reasons

**$FAILURE_REASON**:
$(case "$FAILURE_REASON" in
    draft_status) echo "- PR was left in draft state instead of being marked ready for review" ;;
    merge_conflicts) echo "- PR has merge conflicts that were not resolved" ;;
    ci_failure) echo "- CI checks are failing and were not fixed" ;;
    unaddressed_comments) echo "- Review comments exist but were not addressed" ;;
    closed_prematurely) echo "- PR was closed before reaching verified state" ;;
    unknown) echo "- Reason unclear - needs manual investigation" ;;
esac)

## Root Cause Questions

1. Why did the pr-ready cycle not detect this issue?
2. Was there a validation step that should have caught this?
3. Did the workflow transition prematurely?
4. Was there an error that was silently ignored?

## Next Steps

1. Review pr-ready cycle logs for this PR
2. Check if validation gates were properly enforced
3. Identify missing checks or enforcement
4. Update workflow to prevent this failure type

## Impact

- Wasted preparation time
- PR not actually ready for merge
- Human reviewer time wasted
- Workflow effectiveness below target (<80%)

## Related

- PR: #$PR_NUM
- Session: $(date '+%Y-%m-%d %H:%M')
- Failure type: $FAILURE_REASON"

        echo "  Creating improvement issue..."
        gh issue create \
            --title "$ISSUE_TITLE" \
            --body "$ISSUE_BODY" \
            --label "friction,priority:high,pr-ready-cycle"

        echo "  ✅ Issue created"
    done
fi
```

### 5.7. Pattern Detection: Recurring Failure Types

**Purpose**: Identify systemic issues (3+ occurrences of same failure type) requiring CRITICAL
attention.

```bash
echo ""
echo "🔍 Pattern Detection: Recurring Issues"
echo "═════════════════════════════════════"

# Check for patterns (3+ occurrences of same failure type)
CRITICAL_PATTERNS=""

for pattern in "${!FAILURE_PATTERNS[@]}"; do
    count=${FAILURE_PATTERNS[$pattern]}

    if [ "$count" -ge 3 ]; then
        CRITICAL_PATTERNS="${CRITICAL_PATTERNS}${pattern},"

        # Calculate impact
        impact=$(( PR_COUNT > 0 ? count * 100 / PR_COUNT : 0 ))

        echo ""
        echo "⚠️  CRITICAL PATTERN: $pattern"
        echo "## Occurrences"
        echo "  Failed PRs: $count"
        echo "  Percentage: ${impact}%"
        echo ""
        echo "## Impact"
        echo "  This is a systemic issue affecting multiple PRs"
        echo "  Requires immediate workflow improvement"
        echo ""
        echo "## Root Cause Analysis Required"
        echo "  1. Why is this failure recurring?"
        echo "  2. What validation is missing?"
        echo "  3. What enforcement can prevent this?"

        # Create CRITICAL improvement issue for pattern
        PATTERN_TITLE="CRITICAL: Recurring verification failure pattern - $pattern (${count} PRs, ${impact}%)"
        PATTERN_BODY="## Systemic Issue Detected

The pr-ready cycle is systematically failing to verify PRs due to: **$pattern**

## Occurrence Data

- Failed PRs: $count out of $PR_COUNT processed (${impact}%)
- This is a recurring pattern, not isolated incidents
- Below 80% target threshold

## Action Items

1. **Root Cause Analysis**:
   - Review workflow for $pattern handling
   - Identify missing validation or enforcement
   - Check if error detection is working

2. **Workflow Improvements**:
   - Add validation gate for $pattern
   - Improve error handling and reporting
   - Add enforcement to prevent this failure

3. **Testing**:
   - Create test cases for $pattern scenario
   - Verify fixes prevent recurrence
   - Measure improvement in success rate

## Success Criteria

- [ ] Root cause identified and documented
- [ ] Workflow updated with prevention mechanism
- [ ] Test coverage added for this scenario
- [ ] Verification success rate improves to >80%

## Related

- Session: $(date '+%Y-%m-%d %H:%M')
- Pattern: $pattern
- Impact: ${impact}% of PRs"

        gh issue create \
            --title "$PATTERN_TITLE" \
            --body "$PATTERN_BODY" \
            --label "friction,priority:critical,pr-ready-cycle,pattern"
    fi
done

if [ -z "$CRITICAL_PATTERNS" ]; then
    echo "✅ No recurring patterns detected"
    echo "   All failures appear to be isolated incidents"
fi
```

### 5.8. Success Metrics Tracking

**Purpose**: Store metrics in JSONL format for historical trend analysis.

```bash
echo ""
echo "💾 Storing Success Metrics"
echo "═════════════════════════"

# Store metrics in analysis results for trend tracking
METRICS_FILE="tools/analysis-results/pr-ready-verification-rates.jsonl"

# Create metrics directory if it doesn't exist
mkdir -p "$(dirname "$METRICS_FILE")"

# Fix: Ensure SUCCESS_RATE is defined (default to 0 if not set from earlier)
SUCCESS_RATE=${SUCCESS_RATE:-0}

# Append metrics as JSON Lines format
echo "{\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"total_prs\":$PR_COUNT,\"verified_prs\":${VERIFIED_COUNT:-0},\"success_rate\":$SUCCESS_RATE,\"target\":80}" >> "$METRICS_FILE"

echo "✅ Metrics stored: $METRICS_FILE"

# Display trend if we have historical data
LINE_COUNT=$(wc -l < "$METRICS_FILE" | tr -d ' ')
if [ "$LINE_COUNT" -ge 2 ]; then
    echo ""
    echo "📈 Historical Trend"

    # Get previous session metrics (check for 2+ lines before using tail -2)
    PREV=$(tail -2 "$METRICS_FILE" | head -1)
    PREV_RATE=$(echo "$PREV" | grep -o '"success_rate":[0-9]*' | cut -d: -f2)

    if [ -n "$PREV_RATE" ]; then
        DIFF=$((SUCCESS_RATE - PREV_RATE))

        if [ "$DIFF" -gt 0 ]; then
            echo "  Previous: ${PREV_RATE}%"
            echo "  Current: ${SUCCESS_RATE}%"
            echo "  Change: +${DIFF}% ⬆️  IMPROVING"
        elif [ "$DIFF" -lt 0 ]; then
            echo "  Previous: ${PREV_RATE}%"
            echo "  Current: ${SUCCESS_RATE}%"
            echo "  Change: ${DIFF}% ⬇️  DECLINING"
        else
            echo "  Previous: ${PREV_RATE}%"
            echo "  Current: ${SUCCESS_RATE}%"
            echo "  Change: No change"
        fi
    else
        echo "  First session - no comparison available"
    fi
else
    echo "  First session - no historical data yet"
fi
```

## Output

This sub-prompt produces:

- Verification success rate analysis
- Root cause analysis for each failed PR
- Improvement issues created for failures
- Pattern detection for recurring issues
- Historical metrics stored for trend tracking

## Related Documentation

- Parent: prompts/pr-ready-retrospective.md
- Specification: workflows/PR_READY_CYCLE_SPECIFICATION.md
- Issue: #2158 (verification success rate tracking)
