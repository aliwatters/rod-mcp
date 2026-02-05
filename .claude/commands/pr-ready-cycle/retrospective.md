# PR Ready Retrospective Phase

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

**⚠️ MANDATORY COMPLIANCE**: You MUST follow ALL instructions in this prompt - failure to do so will
not be tolerated. Execute every step completely before proceeding.

## Purpose

Analyze the PR preparation session and extract learnings for continuous improvement.

**Modes**:

- **Periodic**: Run during continuous mode after N PRs (don't clear state, continue to Discovery)
- **Final**: Run in one-shot mode or when cycle ends (clear state and exit)

## Overview

This phase performs comprehensive retrospective analysis by:

- Collecting session metrics from state
- Analyzing friction and patterns
- Generating improvement issues
- **Periodic mode**: Continue to Discovery (preserve state)
- **Final mode**: Clear workflow state and complete cycle

## Composable Architecture

This workflow follows the composable AI workflows architecture:

- **Main orchestrator**: This file coordinates the retrospective phase
- **Sub-prompts**: Specialized analysis tasks are delegated to focused sub-prompts
- **Verification analysis**: Delegated to pr-ready-retrospective-verification-analysis.md

## Mode Detection

Check if in continuous mode:

```bash
ax prready get | grep "^CONTINUOUS=" | cut -d= -f2
```

**If output is "true"**: This is a periodic retrospective (continuous mode)

**Otherwise**: This is a final retrospective (one-shot mode)

## Core Steps

### 1. Collect Session Metrics

```bash
# Get all metrics from state
ax prready metrics

# Get processed PRs and display metrics
echo "📊 Session Metrics:"
echo "   PRs processed: $(ax prready get | grep "^PRS_PROCESSED=" | cut -d= -f2 | tr ',' '\n' | grep -c '^[0-9]')"
echo "   PRs: $(ax prready get | grep "^PRS_PROCESSED=" | cut -d= -f2)"
```

### 2. Run Label Success Verification

**Purpose**: Verify that PRs actually reached intended "ready for humans" + "verified" state.

```bash
# Use the dedicated metrics collection prompt
prompt pr-ready-cycle/retrospective-metrics
```

### 3. Analyze Session Performance

```bash
# Get detailed metrics and display performance analysis
echo "📈 Performance Analysis:"
echo "   Conflicts resolved: $(ax prready get | grep "^CONFLICTS_RESOLVED=" | cut -d= -f2)"
echo "   Lint fixes applied: $(ax prready get | grep "^LINT_FIXES=" | cut -d= -f2)"
echo "   Test failures: $(ax prready get | grep "^TEST_FAILURES=" | cut -d= -f2)"
echo "   Complex issues: $(ax prready get | grep "^COMPLEX_ISSUES=" | cut -d= -f2)"
echo "   Total time: $(ax prready get | grep "^TOTAL_PREP_TIME_MINUTES=" | cut -d= -f2) minutes"
```

### 4. Check for Friction Log

```bash
# Look for friction log
if [ -f "tmp/friction-pr-ready-session.md" ]; then
    echo "📋 Friction log found: tmp/friction-pr-ready-session.md"
    FRICTION_LOG="tmp/friction-pr-ready-session.md"
else
    echo "ℹ️  No friction log found (clean session)"
    FRICTION_LOG=""
fi
```

### 5. Run Retrospective Analysis

```bash
# Use the dedicated analysis prompt
prompt pr-ready-cycle/retrospective-analysis
```

### 5.5. CRITICAL: Verification Success Rate Analysis

**Purpose**: Analyze why PRs didn't reach verified state and create improvement issues.

**Implementation**: This analysis is delegated to a dedicated sub-prompt following the composable
architecture pattern.

```bash
# Execute verification success rate analysis (issue #2158)
prompt pr-ready-cycle/retrospective-verification-analysis
```

This sub-prompt performs:

- Calculate verification success rate (target: >80%)
- Root cause analysis for each failed PR
- Create improvement issues for failures
- Detect recurring failure patterns (3+ occurrences)
- Create CRITICAL issues for systemic problems
- Store metrics for historical trend tracking

**Why this matters**:

- Zero tolerance for non-verified PRs
- Every failure gets investigated and tracked
- Pattern recognition identifies systemic issues
- Data-driven continuous improvement
- Historical trend analysis

### 6. Display Retrospective Summary

```bash
# Show retrospective summary
if [ -f "tmp/retrospective-pr-ready-session.md" ]; then
    echo "📊 Retrospective Summary:"
    echo "================================"
    cat tmp/retrospective-pr-ready-session.md
    echo "================================"
else
    echo "📊 Basic Session Summary:"
    echo "   PRs processed: $PR_COUNT"
    if [ "$PR_COUNT" -gt 0 ]; then
        echo "   Success rate: $(( (PR_COUNT - COMPLEX_ISSUES) * 100 / PR_COUNT ))%"
        echo "   Average time per PR: $(( TOTAL_TIME / PR_COUNT )) minutes"
    else
        echo "   Success rate: N/A (no PRs processed)"
        echo "   Average time per PR: N/A (no PRs processed)"
    fi
fi
```

### 7. Generate Improvement Issues (if any)

```bash
# Check if improvement issues were generated
if [ -f "tmp/retrospective-pr-ready-session.md" ]; then
    # Extract improvement issues from retrospective
    grep -A 10 "## Improvement Issues" tmp/retrospective-pr-ready-session.md || echo "No improvement issues generated"
fi
```

### 8. Post Retrospective Analysis (Optional)

```bash
# Post retrospective as comment to tracking issue (if configured)
if [ -n "$TRACKING_ISSUE" ]; then
    echo "📤 Posting retrospective analysis to issue #$TRACKING_ISSUE"

    ax session create-analysis-comment \
        --issue "$TRACKING_ISSUE" \
        --type retrospective \
        --data "$(cat tmp/retrospective-pr-ready-session.md)"
else
    echo "ℹ️  No tracking issue configured, skipping comment posting"
fi
```

### 9. Clean Up Expired Processing State

```bash
# Remove pr-ready-processing labels from PRs processed > 1 hour ago
echo "🧹 Cleaning up expired processing labels..."

# Get all PRs with pr-ready-processing label and process line by line
FOUND_PRS=false
while IFS= read -r PR_NUM; do
    if [ -n "$PR_NUM" ]; then
        FOUND_PRS=true
        # Check if PR was processed >1 hour ago
        if ax prready is-processed "$PR_NUM" --within-hours 1; then
            echo "   PR #$PR_NUM still within 1-hour window - keeping label"
        else
            echo "   Removing 'pr-ready-processing' label from PR #$PR_NUM (expired)"
            ax pr labels manage --pr "$PR_NUM" --remove "pr-ready-processing" 2>/dev/null || true
        fi
    fi
done < <(ax pr list --label "pr-ready-processing" --json number --jq '.[].number')

if [ "$FOUND_PRS" = false ]; then
    echo "   No PRs with 'pr-ready-processing' label found"
fi

echo "✅ Expired processing state cleaned up"
```

### 10. Clean Up Session Files (Conditional)

```bash
# Clean up temporary files (only in final mode)
if [ "$RETRO_MODE" = "final" ]; then
    rm -f tmp/friction-pr-ready-session.md
    rm -f tmp/retrospective-pr-ready-session.md
    echo "🧹 Cleaned up session files"
else
    echo "📌 Preserving session files for next periodic retrospective"
fi
```

### 11. Complete or Continue (Conditional)

```bash
if [ "$RETRO_MODE" = "final" ]; then
    # Final retrospective - clear state and exit
    ax prready clear
    echo "✅ PR Ready Cycle complete"
else
    # Periodic retrospective - continue to Discovery
    echo "🔄 Periodic retrospective complete - returning to Discovery"
    ax prready update --phase discovery
    prompt pr-ready-cycle/discovery
fi
```

## MANDATORY: Cycle Completion (Mode-Dependent)

### Session Logging (Issue #3395)

**Before completing retrospective**, create a session log for analysis:

```bash
# Log this PR ready ax session
# Note: Use ax command instead of prompt to avoid nested slash command invocation
ax log-session create
```

This captures complete ax state, all phases executed, and session metadata for later analysis.

### Final Mode

**✅ DO**:

- Create session log with `ax log-session create`
- Clear workflow state with `ax prready clear`
- Display session summary
- Clean up temporary files
- Complete the cycle (no further transitions)

**❌ DON'T**:

- Execute additional phases after retrospective
- Leave state files in place
- Skip cleanup steps
- Continue to other workflows

### Periodic Mode

**✅ DO**:

- Keep workflow state intact
- Display session summary
- Preserve temporary files
- Continue to Discovery phase

**❌ DON'T**:

- Clear workflow state
- Clean up temporary files
- Exit the cycle
- Stop continuous mode

## MANDATORY: Automatic Transition

**⚠️ AUTOMATIC TRANSITION**: After completing retrospective, automatically transition based on mode.

### Final Mode Transition

```bash
# Clear workflow state
ax prready clear
echo "✅ PR Ready Cycle complete"
exit 0
```

### Periodic Mode Transition

```bash
# Continue to Discovery
ax prready update --phase discovery
prompt pr-ready-cycle/discovery
```

**Note**: The transition happens automatically based on the CONTINUOUS flag. No user intervention
required.

## Related Documentation

- **Sub-prompts**: prompts/pr-ready-retrospective-verification-analysis.md
- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Specification**: workflows/PR_READY_CYCLE_SPECIFICATION.md
- **Issue**: #2158 (verification success rate analysis)
