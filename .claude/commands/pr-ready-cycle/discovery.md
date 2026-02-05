# PR Ready Discovery Phase

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

## Purpose

Find PRs needing preparation (sleeps if none found in continuous mode).

## Prerequisites

**Required before Discovery**:

- PR-ready workflow state initialized (`ax prready get` shows valid state)
- Phase set to discovery (`PHASE=discovery`)
- Understanding of PR prioritization and claim mechanics
- Knowledge of label validation and cleanup procedures

**⚠️ WORKSPACE ISOLATION**: Stay within starting workspace. NEVER cd to sibling directories.

**📖 Detailed Implementation**: For complete step-by-step instructions with all commands, see:
`~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` (Phase 2: Discovery)

---

## Overview

The discovery phase executes in this order:

**⚠️ CRITICAL**: This check runs FIRST before all other discovery. Stale verified labels can lead to
merging broken PRs.

1. **Phase 0: Label Validation & Cleanup** (HIGHEST PRIORITY)
2. **Step 1: Discover Next PR** (Priority 2-4)
3. **Step 2: Evaluate PR Alignment** (AI-Powered)
4. **Step 2.5: Check if PR is Already Claimed**
5. **Step 3: Claim the PR and Transition**

---

## Phase 0: Label Validation & Cleanup

**⚠️ CRITICAL**: Runs FIRST every cycle before discovering PRs. Stale labels can lead to merging
broken PRs.

**What to Do**:

- Scan ALL open PRs (not just recently updated)
- Validate "verified" labels: Re-run verification checks, remove if criteria no longer met
- Validate "ready" labels: Remove if PR has conflicts or failing CI
- Validate "in-progress" labels: Remove if stale (>30 minutes old)
- Cleanup stale PR claims: Remove claims older than 30 minutes

**Why This Matters**:

- Prevents merging PRs that are no longer actually verified
- Keeps label state synchronized with actual PR state
- Ensures claims don't block PR processing indefinitely
- Highest priority because incorrect labels cause the most damage

**How**: Use the integrated `ax pr labels validate` command for comprehensive label validation

See `workflows/PR_LABELING_RULES.md` for complete invariant definitions.

```bash
# Phase 0: Validate ALL open PR labels using comprehensive validation
# This replaces ~200 lines of manual bash validation (Issue #2551)
echo "=== Phase 0: Validate Open PR Labels ==="

# Validate and auto-fix all open PRs against labeling invariants
# This enforces 6 invariants from workflows/PR_LABELING_RULES.md:
# - 'verified' requires 'ready for humans'
# - 'verified' requires no conflicts, passing CI
# - 'ready for humans' and 'ready for robots' are mutually exclusive
# (Issue #3373: Optimize pr-ready-cycle with selective label sync)
VALIDATION_OUTPUT=$(ax pr labels validate --all --fix 2>&1)
CHANGES=$(echo "$VALIDATION_OUTPUT" | grep -c "Fixed:" || echo "0")
echo "🔄 Label validation: $CHANGES label changes applied"

# If changes were detected, skip sleep and proceed immediately
if [ "$CHANGES" -gt 0 ]; then
    echo "✅ Changes detected - proceeding immediately to discovery"
    # Set flag to skip progressive sleep later
    LABEL_CHANGES_DETECTED=true
else
    echo "💤 No changes detected - will use progressive sleep if no PRs found"
    LABEL_CHANGES_DETECTED=false
fi

# Validate and sync all PR claims
WORKSPACE=$(basename "$PWD")
ax prready sync-claims "$WORKSPACE" --verbose
if [ $? -eq 0 ]; then
    ax prready increment-metric stale_claims_cleaned
fi

echo "✅ Phase 0 complete - all labels validated and claims cleaned"
```

---

## Step 1: Discover Next PR Needing Preparation

**What to Do**:

- Run discovery script: `ax discover pr-ready`
- Script outputs PR number if found, or nothing if no PRs need work
- Skip PRs already processed this session
- Skip PRs with "blocked" label
- **NEW (Issue #2413): Skip PRs with "verified" label** - these are already ready for merge

**Priority Order** (highest first):

1. Ready-for-Humans PRs **without "verified" label** (need final verification)
2. Ready-for-Robots PRs (need full preparation)
3. Draft PRs (need initial work)

**Note**: PRs with both "ready for humans" AND "verified" labels are excluded from discovery because
they're already ready for merge and don't need re-preparation.

**Why This Matters**:

- Focuses work on PRs closest to merge
- Systematic prioritization prevents PRs from stalling
- Single PR focus ensures quality over quantity

**How**: See workflow Phase 2 → "Discovery Steps" for complete priority logic and commands

---

## Step 1.5: Check Prerequisites (NEW - Issue #2166)

**What to Do**:

- Before claiming PR, verify prerequisites are met
- Run: `ax pr review-comments verify $PR_NUMBER`
- Skip PRs that don't meet prerequisites

**Prerequisites Check**:

```bash
# Before claiming PR, verify prerequisites
if ! ax pr review-comments verify $PR_NUMBER; then
    echo "⏭️  PR #$PR_NUMBER missing threaded replies - returning to draft for dev-cycle"

    # Passive state management: fix labels and return to draft
    # Dev-cycle will discover this PR naturally
    gh pr edit $PR_NUMBER --remove-label "ready for robots" 2>/dev/null || true
    gh pr edit $PR_NUMBER --remove-label "ready for humans" 2>/dev/null || true
    gh pr edit $PR_NUMBER --remove-label "verified" 2>/dev/null || true
    gh pr edit $PR_NUMBER --remove-label "claimed" 2>/dev/null || true

    # Return to draft status
    gh pr ready --undo $PR_NUMBER 2>/dev/null || true

    # Add label to indicate why it went back
    gh pr edit $PR_NUMBER --add-label "needs-refinement"

    # Mark as processed to prevent retry in this session
    ax prready mark-processed $PR_NUMBER

    echo "✅ PR #$PR_NUMBER returned to draft - dev-cycle will pick it up"
    continue
fi
```

**Why This Matters**:

- Prevents claiming PRs that aren't ready for preparation phase
- Ensures threaded replies are posted before preparation begins
- Maintains proper phase ordering (refinement → preparation)
- Prevents bypassing quality gates
- Uses passive state management (labels + draft) instead of active handoff
- Dev-cycle naturally discovers draft PRs with "needs-refinement" label

**How**: See workflow Phase 2 → "Prerequisites Check" for complete implementation

---

## Step 2: Evaluate PR Alignment

**What to Do**:

- Use AI to evaluate if PR aligns with project goals
- Run: `prompt evaluate-pr-alignment "PR #$PR_NUMBER"`
- AI produces alignment score (0-100%) with reasoning

## Step 2.1: Agent-First PR Evaluation

**⚠️ CRITICAL**: Consider agent-first principles when evaluating PR alignment to systematically
improve development tooling.

**Agent-First Evaluation Criteria**:

**✅ High Priority**:

- Reduces token usage in development tools
- Adds minimal output modes to CLI tools
- Implements deterministic behavior
- Removes interactive prompts from automation tools
- Simplifies command interfaces
- Adds structured data output formats

**⚠️ Review Carefully**:

- Adds progress bars to CLI tools
- Introduces interactive modes to development tooling
- Adds verbose output as default without machine-readable alternative

**Decision Criteria**:

- Score ≥ 70%: High alignment - proceed with preparation
- Score 40-69%: Medium alignment - proceed with caution
- Score < 40%: Low alignment - skip this PR and find another

**Why This Matters**:

- Prevents wasting time on PRs that don't align with project direction
- AI reasoning catches subtle misalignment that labels miss
- Human review can override AI decision if needed

**How**: See workflow Phase 2 → "🤖 Intelligent Alignment Assessment" for evaluation criteria

---

## Step 2.5: Check if PR is Already Claimed

**⚠️ MANDATORY**: Before processing discovered PR, check if another workspace already claimed it.

**What to Do**:

- Check claim status: `ax prready is-claimed <PR>`
- If claimed: Skip to next available PR
- If not claimed: Proceed to Step 3

**Why This Matters**:

- Prevents race conditions in multi-agent scenarios
- Ensures exclusive PR ownership per workspace
- Avoids duplicate work and conflicting changes
- Claims auto-expire after 30 minutes (stale claims removed in Phase 0)

**How**: See workflow Phase 2 → "Step 2.5: Check if PR is Already Claimed" for complete logic

---

## Step 3: Claim the PR and Transition

**What to Do**:

- Claim PR for exclusive processing: `ax prready claim <PR> <WORKSPACE>`
- Update state: `ax prready update --current-pr <PR>`
- Route to appropriate workflow based on PR labels:
  - "ready for humans" (without "verified") → verification phase (add "verified" label)
  - "ready for robots" → preparation phase (prepare-for-humans)
  - No routing label → default to preparation phase (prepare-for-humans)

**Why This Matters**:

- Exclusive claim prevents other agents from working on same PR
- State update enables tracking and retrospective analysis
- Proper routing ensures PR gets appropriate level of preparation:
  - "ready for humans" PRs go straight to verification (they've already been prepared)
  - "ready for robots" PRs need full preparation before verification

**How**: See workflow Phase 2 → "Step 3: Claim the PR for Exclusive Processing" for claim handling
and fallback logic

---

## If No PR Found

**Continuous Mode**:

- Log: "No PRs found needing preparation"
- Sleep for configured duration (default: 15 minutes)
- Loop back to Discovery phase automatically

**One-Shot Mode**:

- Log: "No PRs found - exiting"
- Transition to Retrospective phase automatically
- Exit after retrospective completes

**How**: See workflow Phase 2 → "If No PRs Found" sections for sleep and transition logic

### Implementation: Continuous Mode Sleep and Loop

**Execute this when no PR is found**:

Check if in continuous mode:

```bash
ax prready get | grep "^CONTINUOUS=" | cut -d= -f2
```

**Expected output**: "true", "false", or empty string

**If continuous mode is "true"**:

Get sleep duration:

```bash
ax prready get | grep "^SLEEP_MINUTES=" | cut -d= -f2
```

**Note**: If empty, default to 15 minutes.

Calculate sleep duration in seconds (multiply minutes by 60), then sleep:

**Progressive Sleep Strategy (Issue #3373)**:

Instead of monolithic 15-minute sleep, use progressive strategy for better responsiveness:

```bash
# Get configured sleep duration (default: 15 minutes)
SLEEP_MINUTES=$(ax prready get | grep "^SLEEP_MINUTES=" | cut -d= -f2)
if [ -z "$SLEEP_MINUTES" ]; then
    SLEEP_MINUTES=15
fi

# Calculate progressive sleep durations (split into 2 parts)
FIRST_SLEEP=$((SLEEP_MINUTES * 60 / 2))   # Half of total sleep time (in seconds)
SECOND_SLEEP=$((SLEEP_MINUTES * 60 / 2))  # Half of total sleep time (in seconds)

echo "💤 Progressive sleep strategy: ${FIRST_SLEEP}s + ${SECOND_SLEEP}s (total: ${SLEEP_MINUTES}m)"

# First sleep period
echo "💤 Sleeping for $(awk "BEGIN {printf \"%.1f\", $FIRST_SLEEP/60}") minutes..."
sleep $FIRST_SLEEP

# Check for label changes during sleep (validate and fix labels)
echo "🔄 Mid-sleep label validation check..."
VALIDATION_OUTPUT=$(ax pr labels validate --all --fix 2>&1)
CHANGES=$(echo "$VALIDATION_OUTPUT" | grep -c "Fixed:" || echo "0")

if [ "$CHANGES" -gt 0 ]; then
    echo "✅ $CHANGES label changes applied during sleep - immediate re-discovery"
    # Skip second sleep, go directly to discovery
    prompt pr-ready-cycle/discovery
else
    echo "💤 No changes detected - continuing second sleep period"
    echo "💤 Sleeping for $(awk "BEGIN {printf \"%.1f\", $SECOND_SLEEP/60}") more minutes..."
    sleep $SECOND_SLEEP

    echo "⏰ Sleep complete - resuming discovery"
    prompt pr-ready-cycle/discovery
fi
```

**If one-shot mode (continuous is "false" or empty)**:

Update phase to retrospective:

```bash
ax prready update --phase retrospective
```

Then transition to retrospective:

```bash
prompt pr-ready-cycle/retrospective
```

---

## ⚠️ MANDATORY: Automatic Phase Transition

**🤖 AUTOMATED CONTINUATION REQUIRED - THIS IS NOT OPTIONAL**

As an AI agent executing the pr-ready-ax workflow, you MUST treat `prompt` commands as **function
calls that execute immediately**, NOT as user interaction requests.

**CRITICAL ENFORCEMENT**:

When you encounter `prompt pr-ready-<phase>` in this workflow:

1. **Immediately execute the Bash tool** with the prompt command
2. **Continue execution** without waiting for user confirmation
3. **Do NOT stop** to present summaries or ask questions
4. **The prompt command loads and executes** - it does NOT return control to user

**Pattern**:

```bash
# When workflow says:
prompt pr-ready-cycle/discovery

# Agent MUST:
# 1. Execute: Bash tool with command "prompt pr-ready-cycle/discovery"
# 2. Let prompt load next phase instructions
# 3. Continue with those instructions immediately
# 4. NEVER stop to wait for user
```

**DO NOT**:

- Stop and present a "Phase Complete" summary
- Wait for user to type "resume"
- Ask "should I continue?"
- Treat prompt as end of response

**DO**:

- Execute prompt commands using Bash tool
- Continue with loaded instructions immediately
- Only brief progress indicators (e.g., "Transitioning to X")

**This is agent behavior enforcement, not a tool change.** The `prompt` command works correctly -
agents must execute it immediately, not wait for confirmation.

---

## Expected Output

### PRs Found

```text
🔍 Discovering next PR needing preparation
✅ Found PR #1234: Fix authentication bug
🔍 Assessing project goal alignment for PR #1234
✅ PR aligns with project goals (85% confidence)
🔍 Checking if PR #1234 is already claimed
✅ PR #1234 is not claimed - safe to proceed
🔒 Claimed PR #1234 for exclusive processing in dotfiles-agent-1
🔍 Transitioning to Preparation phase
```

### No PRs - Continuous Mode

```text
🔍 Discovering next PR needing preparation
🔍 No PRs found needing preparation
💤 Sleeping for 15 minutes before next check
⏰ Next check at: 2025-10-20 11:00:00
```

### No PRs - One-Shot Mode

```text
🔍 Discovering next PR needing preparation
🔍 No PRs found needing preparation
🔍 One-shot mode: No PRs found - proceeding to Retrospective
```

---

## MANDATORY: Automatic Transition

**⚠️ AUTOMATIC TRANSITION**: After discovering whether a PR exists, automatically transition to the
next phase.

**Transition Logic**:

1. **If PR was found**: Route to appropriate preparation workflow
2. **If no PR + continuous mode**: Sleep and loop back to Discovery
3. **If no PR + one-shot mode**: Transition to Retrospective

**Do NOT wait for user confirmation**. The agent executes transitions immediately.

**How**: See workflow Phase 2 → "Transition" section for complete routing logic

---

## Troubleshooting

### Discovery taking too long

**Cause**: Large number of PRs or GitHub API rate limiting

**Solution**: Check API rate limits with `gh api rate_limit`, wait for reset if needed

### Sleep not working in continuous mode

**Cause**: CONTINUOUS flag not set correctly

**Solution**: Check state with `ax prready get | grep CONTINUOUS`, set if needed

### PR claiming fails

**Cause**: PR was claimed between check and claim (race condition)

**Solution**: Discovery automatically loops to find next PR, no manual intervention needed

---

**Complete Workflow**: See `~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` for all commands
and detailed logic

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
- **Validation Guide**: ~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW_VALIDATION.md
