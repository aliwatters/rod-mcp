# Grooming Discovery

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

Find grooming candidates using intelligent prioritization and prepare them for autonomous
processing.

## Overview

This phase discovers grooming candidates by:

- Detecting obsolete issues from architecture migrations (Priority 0)
- Applying 6-level priority system for candidate detection
- Finding critical duplicates (>95% similarity)
- Identifying obvious stale issues (>365 days inactive)
- Detecting high-confidence duplicates (85-95% similarity)
- Finding medium stale issues (180-365 days inactive)
- Identifying low-confidence candidates requiring review
- Claiming candidates to prevent conflicts
- Sleeping intelligently when no candidates (continuous mode)

## Discovery Process

### Step 1: Initialize Discovery Session

```bash
# Update state to discovery phase
ax groom update --phase discovery

# Log discovery start
echo "🔍 Starting candidate discovery at $(date)"
```

### Step 2: Apply 6-Level Priority System

#### Priority 0: Obsolescence Candidates (Issues made obsolete by migrations)

**Purpose**: Systematically detect issues referencing deleted tools, removed commands, or superseded
by architecture changes.

**Detection strategy**: Reference `docs/migrations/MIGRATIONS.md` for known migrations and
obsolescence markers.

```bash
# Detect issues referencing deleted scripts/tools
# Note: Dedicated ax groom obsolete commands will be added in future iterations.
# Current approach uses gh issue list with search patterns to detect obsolescence candidates.
# Reference docs/migrations/MIGRATIONS.md for additional manual markers.

# Check for common obsolescence patterns
echo "🔍 Checking for obsolescence candidates..."

# Pattern 1: Issues referencing "cycle CLI" (removed 2025-11-08)
gh issue list --search 'created:<2025-11-08 "cycle CLI" OR "cycle pr" OR "cycle workflow"' \
  --state open --limit 20 --json number,title,createdAt

# Pattern 2: Issues referencing deleted scripts (see migrations/MIGRATIONS.md)
gh issue list --search 'created:<2025-11-04 "batch-reply-reviews.sh" OR "checkin.sh" OR "discover_prs_needing_prep.py"' \
  --state open --limit 20 --json number,title,createdAt

# Pattern 3: Issues with closed parent (orphaned children)
# Requires manual inspection - look for "Part of #XXXX" patterns in issue bodies

# Pattern 4: Issues referencing direct Python script calls (violates CLAUDE.md Principle 1)
gh issue list --search 'created:<2025-11-08 "python3 ~/git/dotfiles/tools"' \
  --state open --limit 20 --json number,title,createdAt
```

**Expected output**:

```text
🔍 Obsolescence candidates found:
   - 5 reference "cycle CLI" (removed 2025-11-08)
   - 3 reference deleted scripts (batch-reply-reviews.sh, etc.)
   - 2 reference direct Python calls (violates architecture)
   → Total: 10 high-priority obsolescence candidates
```

**Safety controls**:

- Skip issues updated within last 30 days (may be actively migrated)
- Skip issues with `priority:critical` label
- Escalate if uncertain about replacement availability

#### Priority 1: Critical Duplicates (>95% similarity, recent activity)

```bash
# Find critical duplicates with very high confidence
ax groom duplicates --threshold 0.95 | head -20
```

#### Priority 2: Completed Issues (work done but not closed)

```bash
# Find issues that were completed but not formally closed
ax groom detect-completed | head -10
```

#### Priority 3: Obvious Stale (>365 days, no activity, priority:low)

```bash
# Find obviously stale issues
ax groom stale --days-low 365 | head -20
```

#### Priority 4: High-Confidence Duplicates (85-95% similarity)

```bash
# Find high-confidence duplicates
ax groom duplicates --threshold 0.85 | head -20
```

#### Priority 5: Medium Stale (180-365 days, no recent activity)

```bash
# Find medium stale issues
# Find medium stale issues (180-365 days inactive)
ax groom stale --days 180 --json | jq '.[] | select(.stale_days <= 365)' | head -20
```

#### Priority 6: Low-Confidence Candidates (requires human review)

```bash
# Get general grooming candidates for review
ax groom candidates --limit 20
```

### Step 3: Claim High-Priority Candidates

```bash
# Note: Claiming logic to prevent conflicts will be implemented in future iteration
# For now, proceed to processing with discovered candidates
echo "📋 Discovery complete - proceeding to processing"
```

### Step 5: Handle No Candidates (Continuous Mode)

```bash
# Check if we're in continuous mode and should sleep
CONTINUOUS=$(ax groom get-continuous)
SLEEP_MINUTES=$(ax groom get-sleep)

if [ "$CONTINUOUS" = "true" ] && ! ax groom has-candidates; then
    echo "😴 No candidates found - sleeping for ${SLEEP_MINUTES} minutes"
    sleep $((SLEEP_MINUTES * 60))
    # Loop back to discovery
    prompt groom-backlog/discovery
    exit 0
fi
```

## Expected Outputs

### High-Priority Candidates Found

```text
🎯 Critical duplicates: 3 found (>95% similarity)
🗄️ Obvious stale: 8 found (>365 days inactive)
🔍 High-confidence duplicates: 5 found (85-95% similarity)
📅 Medium stale: 12 found (180-365 days inactive)
❓ Low-confidence: 15 found (requires review)
✅ Completed but not closed: 2 found
```

### No Candidates Found (Continuous Mode)

```text
😴 No candidates found - sleeping for 15 minutes
⏰ Next discovery cycle: 2025-11-03T15:30:00
```

## ⚠️ MANDATORY: Automatic Phase Transition

**🤖 AUTOMATED CONTINUATION REQUIRED - THIS IS NOT OPTIONAL**

As an AI agent executing the grooming-ax workflow, you MUST treat `prompt` commands as **function
calls that execute immediately**, NOT as user interaction requests.

**If candidates found**, execute:

```bash
# Update workflow state
ax groom update --phase processing

# MANDATORY: Execute processing phase immediately
# Agent MUST treat this as a function call, not a wait point
prompt groom-backlog/processing
```

**If no candidates found (one-shot mode)**, complete cycle:

```bash
# Update workflow state
ax groom update --phase retrospective

# MANDATORY: Execute retrospective phase immediately
prompt groom-backlog/retrospective
```

**If no candidates found (continuous mode)**, sleep and loop:

```bash
# Sleep for configured duration
SLEEP_MINUTES=$(ax groom get-sleep)
echo "😴 Sleeping for ${SLEEP_MINUTES} minutes..."
sleep $((SLEEP_MINUTES * 60))

# Loop back to discovery
prompt groom-backlog/discovery
```

**CRITICAL ENFORCEMENT**:

- `prompt groom-backlog/processing` is a **function call**, not a stop point
- Agent MUST execute it immediately using SlashCommand tool
- Agent MUST continue with loaded processing instructions
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
