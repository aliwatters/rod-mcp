# Grooming Retrospective

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

Analyze session effectiveness, generate improvement recommendations, and decide whether to loop
(continuous) or exit (one-shot).

## Overview

This phase analyzes the grooming session by:

- Analyzing session effectiveness (issues processed, actions taken)
- Generating improvement recommendations
- Creating friction reports for blocking issues
- Updating grooming thresholds based on success rates
- Generating human escalation report
- Deciding whether to loop (continuous) or exit (one-shot)

## Sleep Threshold Configuration

**Issue #3988**: Grooming-cycle now checks remaining backlog before sleeping.

- **Threshold**: 100 issues
- **Behavior**:
  - If **<100 issues remain**: Sleep for configured duration (default 15 minutes)
  - If **≥100 issues remain**: Continue immediately to next cycle
- **Benefits**: Faster backlog reduction when there's work to do, avoiding unnecessary sleep delays

## Retrospective Analysis

### Step 1: Initialize Retrospective Session

```bash
# Update state to retrospective phase
ax groom update --phase retrospective

# Log retrospective start
echo "📊 Starting retrospective analysis at $(date)"
```

### Step 2: Analyze Session Effectiveness

```bash
# Generate session metrics
ax groom get | grep -E "ISSUES_PROCESSED|ACTIONS_TAKEN|SESSION_START"

# Calculate session duration using available data (manual calculation required)
SESSION_START=$(ax groom get | grep "^SESSION_START=" | cut -d= -f2)
SESSION_END=$(date +"%Y-%m-%d %H:%M:%S")
echo "Session started at: $SESSION_START"
echo "Session ended at: $SESSION_END"
echo "⚠️ Session duration calculation requires manual review. Future automation will be added."
```

### Step 3: Generate Success Metrics

```bash
# Calculate success rates and impact
echo "📈 Session Impact Analysis:"
echo "Issues processed: $(ax groom get-issues-processed)"
echo "Actions taken: $(ax groom get-actions-taken)"

# Calculate backlog reduction using available session data
echo "Backlog reduction: $(ax groom get | grep "^ISSUES_PROCESSED=" | cut -d= -f2)"
```

### Step 4: Analyze Action Breakdown

```bash
# Count different types of actions taken
echo "🎯 Action Breakdown:"
if [ -f "tmp/grooming-session.log" ]; then
    echo "Duplicates merged: $(grep -c "Merged duplicate" tmp/grooming-session.log || echo 0)"
    echo "Stale issues labeled: $(grep -c "Labeled stale issue" tmp/grooming-session.log || echo 0)"
    echo "Completed issues closed: $(grep -c "Closed completed issue" tmp/grooming-session.log || echo 0)"
    echo "Issues escalated: $(grep -c "Escalated" tmp/grooming-session.log || echo 0)"
else
    echo "No session log found - implementing session logging in future iteration"
fi
```

### Step 5: Generate Improvement Recommendations

```bash
# Analyze patterns and recommend threshold adjustments
echo "💡 Improvement Recommendations:"

# Check if false positive rate is too high
if [ -f "tmp/grooming-escalations-$(date +%Y%m%d).md" ]; then
    ESCALATIONS=$(wc -l < "tmp/grooming-escalations-$(date +%Y%m%d).md")
    ACTIONS=$(ax groom get-actions-taken)

    # Threshold analysis command will be added in a future iteration
fi
```

### Step 6: Generate Human Escalation Report

```bash
# Consolidate escalations for human review
if [ -f "tmp/grooming-escalations-$(date +%Y%m%d).md" ]; then
    echo "📋 Human Escalation Report Generated:"
    echo "Location: tmp/grooming-escalations-$(date +%Y%m%d).md"
    echo "Items requiring review: $(wc -l < tmp/grooming-escalations-$(date +%Y%m%d).md)"
else
    echo "✅ No escalations required - all actions completed autonomously"
fi
```

### Step 7: Create Friction Report

```bash
# Analyze any friction encountered during the session
if [ -f "tmp/friction-grooming-session.md" ]; then
    echo "⚠️ Friction Analysis:"
    echo "Friction events: $(grep -c "^##" tmp/friction-grooming-session.md || echo 0)"
    echo "Time lost: $(grep "Time Lost" tmp/friction-grooming-session.md | tail -1)"
else
    echo "✅ No friction detected - session ran smoothly"
fi
```

## Expected Outputs

### Successful Session

```text
📊 Session Impact Analysis:
Issues processed: 23
Actions taken: 18
Session duration: 0.5 hours
Estimated backlog reduction: 3.2%

🎯 Action Breakdown:
Duplicates merged: 5
Stale issues closed: 10
Completed issues closed: 3
Issues escalated: 0

💡 Improvement Recommendations:
Escalation rate: 12.5%
✅ Thresholds are well-calibrated

✅ No escalations required - all actions completed autonomously
✅ No friction detected - session ran smoothly
```

### Session with Escalations

```text
📊 Session Impact Analysis:
Issues processed: 15
Actions taken: 8
Session duration: 0.3 hours
Estimated backlog reduction: 2.1%

📋 Human Escalation Report Generated:
Location: tmp/grooming-escalations-20251103.md
Items requiring review: 7

💡 Improvement Recommendations:
Escalation rate: 46.7%
📈 Recommendation: Increase confidence thresholds (too many escalations)
```

## ⚠️ MANDATORY: Cycle Decision and Execution

**🤖 AUTOMATED CONTINUATION REQUIRED - THIS IS NOT OPTIONAL**

As an AI agent executing the grooming-ax workflow, you MUST decide whether to loop or exit based on
the configured mode.

**Check continuous mode setting**:

```bash
CONTINUOUS=$(ax groom get-continuous)
RETRO_INTERVAL=$(ax groom get-retro-interval)
ISSUES_PROCESSED=$(ax groom get-issues-processed)
```

**If continuous mode AND not at retro interval**, check backlog before looping:

```bash
# Use explicit shell logic to decide whether to continue looping
if [ "$CONTINUOUS" = "true" ] && [ "$ISSUES_PROCESSED" -ne "$RETRO_INTERVAL" ]; then
    # Check remaining work before sleeping (Issue #3988)
    # Only sleep when backlog is low (<100 issues), continue immediately otherwise
    REMAINING_ISSUES=$(ax groom candidates --limit 100 --format json | jq -r '.count')
    SLEEP_MINUTES=$(ax groom get-sleep)
    : "${SLEEP_MINUTES:=15}"  # Default to 15 if empty

    if [ "$REMAINING_ISSUES" -lt 100 ]; then
        echo "😴 Low backlog ($REMAINING_ISSUES issues) - sleeping for ${SLEEP_MINUTES} minutes"
        sleep $((SLEEP_MINUTES * 60))
    else
        echo "🔄 High backlog ($REMAINING_ISSUES issues) - continuing immediately"
    fi

    # Loop back to discovery
    echo "🔄 Continuous mode - looping back to discovery"
    ax groom update --phase discovery
    prompt groom-backlog/discovery
fi
```

**If one-shot mode OR at retro interval**, complete cycle:

```bash
# Log this grooming ax session (Issue #3395)
# Note: Use ax command instead of prompt to avoid nested slash command invocation
ax log-session create

# Reset state for next session
ax groom update --phase setup --issues-processed 0 --actions-taken 0

echo "✅ Grooming workflow complete"
echo "📊 Run 'ax groom get' to see final session metrics"
echo "📋 Check tmp/grooming-escalations-*.md for items requiring human review"
```

**CRITICAL ENFORCEMENT**:

- Decision logic executes immediately
- Loop transitions use `prompt groom-backlog/discovery` as **function call**
- Agent MUST execute transitions immediately using SlashCommand tool
- NO stopping, NO waiting, NO summaries between loops

**This is agent behavior enforcement.** The `prompt` command works correctly - agents must execute
it immediately, not wait for user confirmation.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
- **Grooming Specification**:
  ~/git/dotfiles/specifications/grooming-cycle-autonomous-specification.feature
