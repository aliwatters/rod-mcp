# Grooming Processing

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

## ⚡ Agent-First Grooming Guidance

**⚠️ CRITICAL**: Apply agent-first principles when making grooming decisions to systematically
improve development tooling.

**Before processing any issue**, consider its impact on agent efficiency:

**🔥 Consider closing issues that**:

- Request progress bars for CLI tools used in automation
- Request interactive modes for development tooling
- Request verbose/pretty output as default without machine-readable alternatives

**⭐ Prioritize issues that**:

- Improve automation reliability and deterministic behavior
- Add minimal output modes to CLI tools
- Implement clear exit codes and error handling
- Reduce complexity in development workflows

**Decision Framework**: Document agent-first considerations in grooming comments when making
decisions about issue priority or closure.

---

**⚠️ MANDATORY COMPLIANCE**: You MUST follow ALL instructions in this prompt - failure to do so will
not be tolerated. Execute every step completely before proceeding.

## Purpose

Execute autonomous decisions based on confidence levels with comprehensive safety controls and audit
trails.

## Prerequisites

**Required before Processing**:

- Grooming workflow state initialized (`ax groom get` shows valid state)
- Phase set to processing (`PHASE=processing`)
- Discovery phase completed with candidates identified
- Understanding of safety controls and escalation criteria

## Overview

This phase executes autonomous grooming actions by:

- Verifying issue status before any action (file existence, command availability, etc.)
- Auto-closing obsolete issues with replacement documentation
- Auto-merging duplicates >=85% similarity with evidence comments
- Auto-closing stale issues >365 days with reopen instructions
- Auto-closing "already done" issues with PR evidence
- Escalating medium-confidence decisions to human queue
- Tracking all actions with full audit trail
- Maintaining safety controls for critical/active issues

## Safety Controls

### Never Auto-Close

- Issues with `priority:critical` label
- Issues with recent activity (<30 days)
- Issues with `claimed` or `in-progress` labels
- Issues assigned to users
- Issues with active PRs
- **Obsolescence candidates**: Issues referencing tools/commands that still exist
- **Obsolescence candidates**: Issues with no documented replacement
- **Obsolescence candidates**: Issues updated during migration window

### Always Escalate

- Medium confidence decisions (70-84% similarity)
- Complex duplicates with different scopes
- Edge cases requiring human judgment
- Issues with unclear resolution status
- **Obsolescence candidates**: Mixed signals (old but recently updated)
- **Obsolescence candidates**: Verification uncertain (file might exist elsewhere)
- **Obsolescence candidates**: No clear parent reference but suspected child

## Processing Workflow

### Step 1: Initialize Processing Session

```bash
# Update state to processing phase
ax groom update --phase processing

# Initialize session metrics
echo "⚙️ Starting autonomous processing at $(date)"
```

### Step 2: Verify and Process Obsolescence Candidates

**CRITICAL**: Always verify before closing as obsolete.

#### Verification Checklist

For each obsolescence candidate, verify:

```bash
# 1. Check if referenced file/script exists
SCRIPT="tools/gh-helpers/batch-reply-reviews.sh"
if [ -f "$SCRIPT" ]; then
    echo "⚠️ Script still exists - NOT obsolete"
else
    echo "✅ Script deleted - obsolescence confirmed"
fi

# 2. Check if referenced command exists
COMMAND="cycle"
if command -v "$COMMAND" >/dev/null 2>&1; then
    echo "⚠️ Command still exists - NOT obsolete"
else
    echo "✅ Command removed - obsolescence confirmed"
fi

# 3. Check if replacement exists
REPLACEMENT="ax"
if command -v "$REPLACEMENT" >/dev/null 2>&1; then
    echo "✅ Replacement exists - safe to close"
else
    echo "⚠️ No replacement - ESCALATE to human"
fi

# 4. Check parent issue status (if referenced)
PARENT=2509
if gh issue view "$PARENT" --json state --jq '.state' | grep -q "CLOSED"; then
    echo "✅ Parent closed - child may be obsolete"
fi

# 5. Check if issue recently updated (within 30 days)
# Use gh CLI search instead of fragile date arithmetic
# ISSUE_NUM must be set in the environment before running this workflow
if [ -z "$ISSUE_NUM" ]; then
    echo "❌ ERROR: ISSUE_NUM is not set. Please set the issue number as an environment variable before running."
    exit 1
fi
RECENT_UPDATES=$(gh issue list --search "updated:>$(date -d '30 days ago' +%Y-%m-%d 2>/dev/null || date -v-30d +%Y-%m-%d) is:open" --json number --jq ".[] | select(.number == $ISSUE_NUM)")
if [ -n "$RECENT_UPDATES" ]; then
    echo "⚠️ Recently updated - ESCALATE (may be actively migrated)"
else
    echo "✅ Not recently updated - safe to process"
fi
```

**⚠️ AI Agent Context Note**: The verification checklist above uses shell variables for
illustration. When executing in AI agent workflows, use the Read and Bash tools separately for each
check rather than relying on variable persistence across steps.

#### Process Obsolete Issues (High Confidence)

**Only after verification passes**, close obsolete issues:

```bash
# Set ISSUE_NUM to the relevant issue number (see verification checklist above)
# Step 1: Create closure comment file using Write tool
# (AI agents should use Write tool, not heredoc)
# Write to: tmp/closure-comment-${ISSUE_NUM}.md
# Content template below:

# Step 2: Close issue with comment file
gh issue close $ISSUE_NUM --body-file tmp/closure-comment-${ISSUE_NUM}.md
```

**Closure comment template** (use Write tool to create):

````markdown
Closing as **OBSOLETE** - [tool/script] removed, replaced with ax CLI.

## Context

**Migration**: cycle CLI → ax CLI (2025-11-08) **Reference**: docs/migrations/MIGRATIONS.md
**Related Issues**: #3746 (cycle CLI removal), #3785 (obsolescence detection)

## Evidence

- ❌ File check: `tools/gh-helpers/batch-reply-reviews.sh` not found
- ✅ Replacement: `ax pr review-comments reply-batch` exists
- ✅ Parent issue #2509 closed (2025-11-07)

## Replacement

**Old**: `batch-reply-reviews.sh` **New**: `ax pr review-comments reply-batch`

**Usage**:

```bash
# Old (REMOVED)
./tools/gh-helpers/batch-reply-reviews.sh --pr 123

# New (CURRENT)
ax pr review-comments reply-batch --pr 123
```
````

See `docs/migrations/MIGRATIONS.md` for complete replacement mappings.

**Closure comment templates by type**:

**OBSOLETE - Script Deleted**:

```markdown
Closing as **OBSOLETE** - script removed, replaced with ax CLI.

## Evidence

- File: `[path]` not found
- Replacement: `ax [command]` exists

## Replacement

**Old**: `[script].sh` **New**: `ax [command]`
```

**OBSOLETE - Command Removed**:

```markdown
Closing as **OBSOLETE** - cycle CLI removed on 2025-11-08.

## Evidence

- Command: `cycle [subcommand]` returns "command not found"
- Replacement: `ax [subcommand]` exists

## Replacement

**Old**: `cycle [subcommand]` **New**: `ax [subcommand]`

See #3746 for migration details.
```

**OBSOLETE - Parent Closed**:

```markdown
Closing as **OBSOLETE** - parent issue closed, work complete.

## Evidence

- Parent issue #[XXXX] closed on [date]
- Child issue created [date] (before parent completion)

## Context

This issue was part of #[XXXX] which has been completed. Work described here is no longer needed.
```

**COMPLETED - Work Done**:

````markdown
Closing as **COMPLETED** - work already done.

## Evidence

- PR #[XXXX] merged on [date]
- File/feature: [description] exists and works

## Verification

```bash
# Verification command
[command to verify functionality]
```
````

### Step 3: Process Critical Duplicates (>95% similarity)

```bash
# Auto-merge critical duplicates with evidence
ax groom duplicates --threshold 0.95 --auto-merge --dry-run

# If dry-run looks good, execute
ax groom duplicates --threshold 0.95 --auto-merge
```

### Step 4: Process Completed Issues

```bash
# Auto-close issues that were completed but not formally closed
ax groom detect-completed --auto-close --dry-run

# If dry-run looks good, execute
ax groom detect-completed --auto-close
```

### Step 5: Process High-Confidence Duplicates (85-95% similarity)

```bash
ax groom duplicates --threshold 0.85 --auto-merge --dry-run

# If dry-run looks good, execute
ax groom duplicates --threshold 0.85 --auto-merge
```

### Step 6: Label Stale Issues for Review

```bash
# Label stale issues for human review (NOT auto-close)
# IMPORTANT: Age/staleness alone is NOT a valid closure criterion
ax groom stale --days 365 --add-label stale --dry-run

# If dry-run looks good, execute
ax groom stale --days 365 --add-label stale
```

### Step 7: Escalate Medium-Confidence Decisions

```bash
# The --escalate-to-human flag is not supported; instead, generate the list and process output manually.
ax groom duplicates --threshold 0.70 --list > tmp/grooming-escalations-$(date +%Y%m%d).md

# Review tmp/grooming-escalations-$(date +%Y%m%d).md and escalate medium-confidence decisions to human queue as needed.
```

### Step 8: Update Session Metrics

```bash
# Track actions taken in this session
echo "Actions taken: $(ax groom get-actions-taken)"
echo "Issues processed: $(ax groom get-issues-processed)"
```

## Expected Actions and Evidence

### Auto-Merge Duplicates

```text
✅ Merged duplicate: #1234 → #5678 (similarity: 0.92)
   Evidence: Same problem, same solution approach
   Audit: Preserved all context and references
```

### Label Stale Issues for Review

```text
✅ Labeled stale issue: #1234 (inactive: 450 days)
   Action: Added 'stale' label for human review
   Rationale: Age alone is NOT a valid closure criterion - human decides
```

### Auto-Close Completed Issues

```text
✅ Closed completed issue: #1234
   Evidence: Fixed by PR #5678 (merged 30 days ago)
   Audit: Issue requirements met by merged changes
```

### Escalate to Human

```text
🚨 Escalated: #1234 vs #5678 (similarity: 0.78)
   Reason: Different scopes, requires strategic decision
   Action: Added to escalation report for human review
```

## Safety Verification

### Before Each Action

```bash
# Safety verification: check for critical or in-progress labels before auto-processing
# (Inline check; remove reference to non-existent ax groom verify-safe command)

# Check for safety violations
if grep -q "priority:critical\|claimed\|in-progress" <<< "$(gh issue view 1234 --json labels --jq '.labels[].name')"; then
    echo "⚠️ Safety violation detected - escalating to human"
    # Add to escalation report instead of auto-processing
fi
```

### Audit Trail Requirements

All actions must include:

- Issue numbers and titles
- Confidence scores or evidence
- Reasoning for the action
- Safety checks performed
- Rollback/reopen instructions

## ⚠️ MANDATORY: Automatic Phase Transition

**🤖 AUTOMATED CONTINUATION REQUIRED - THIS IS NOT OPTIONAL**

As an AI agent executing the grooming-ax workflow, you MUST treat `prompt` commands as **function
calls that execute immediately**, NOT as user interaction requests.

**After processing complete**, execute:

```bash
# Update workflow state
ax groom update --phase retrospective

# MANDATORY: Execute retrospective phase immediately
# Agent MUST treat this as a function call, not a wait point
prompt groom-backlog/retrospective
```

**If processing fails**, log error and escalate:

```bash
# Log processing failure
echo "❌ Processing failed at $(date)" >> tmp/grooming-failures.log

# Escalate to human
echo "Processing failure requires human intervention" > tmp/grooming-escalations-$(date +%Y%m%d).md

# Still transition to retrospective for analysis
ax groom update --phase retrospective
prompt groom-backlog/retrospective
```

**CRITICAL ENFORCEMENT**:

- `prompt groom-backlog/retrospective` is a **function call**, not a stop point
- Agent MUST execute it immediately using SlashCommand tool
- Agent MUST continue with loaded retrospective instructions
- NO stopping, NO waiting, NO summaries

**This is agent behavior enforcement.** The `prompt` command works correctly - agents must execute
it immediately, not wait for user confirmation.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
- **Grooming Specification**:
  ~/git/dotfiles/specifications/grooming-cycle-autonomous-specification.feature
