# PR Ready Preparation Phase

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

Fix one PR (conflicts, lint, tests) by resolving merge conflicts, fixing lint issues, running tests,
and acknowledging review comments to prepare it for verification.

**⚠️ WORKSPACE ISOLATION**: Stay within starting workspace. NEVER cd to sibling directories.

**📖 Detailed Implementation**: For complete step-by-step instructions with all commands, see:
`~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` (Phase 3: Preparation)

---

## Prerequisites

- Active PR Ready Cycle workflow in preparation phase
- Current PR identified in workflow state
- Access to cycle CLI commands
- Git repository with proper permissions

---

## Overview

This phase systematically prepares a PR for verification by addressing all common issues that
prevent successful merge. The process includes conflict resolution, lint fixes, test execution, and
review comment acknowledgment.

---

## Required Approach: Enforcement Orchestrator

**⚠️ CRITICAL - Enforcement Orchestrator Required**: This preparation phase REQUIRES using the
enforcement orchestrator to ensure all 9 steps execute in order with proper validation:

```bash
# Get current PR from state
CURRENT_PR=$(ax pr get-current)

# Execute preparation phase with enforcement
ax prready enforce "$CURRENT_PR"
```

**The orchestrator ensures**:

- All 9 steps execute in order without skipping
- Each step blocks until complete
- Progress checkpointing for resume capability
- Auto-fix for common issues (conflicts, draft status, labels)
- Comprehensive validation before marking ready
- Quality gates enforced before verification

**Do NOT** attempt manual step-by-step preparation unless the orchestrator is unavailable.

**If orchestrator fails**, address the blocking issue and resume with:

```bash
ax prready enforce "$CURRENT_PR" --resume
```

---

## Manual Fallback (Only if Orchestrator Unavailable)

**Note**: The steps below are provided as a fallback only. The enforcement orchestrator is the
required approach for preparation phase. Only use manual steps if the orchestrator is truly
unavailable.

For manual step-by-step instructions, see:

- **pr-ready-preparation-enhanced.md** - Detailed manual preparation steps
- **PR_READY_CYCLE_WORKFLOW.md** - Complete workflow documentation with all commands

---

## Error Handling and Claim Cleanup

**⚠️ CRITICAL**: If preparation fails or is interrupted, release the PR claim so other agents can
retry.

### When to Release Claim

Release the claim if:

- Preparation orchestrator fails with unrecoverable error
- Manual preparation encounters blocking issue
- Workflow is interrupted or cancelled
- Any critical error prevents completion

### How to Release Claim on Error

**See workflow for complete instructions**: `~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` →
Phase 3 → "Error Handling and Claim Cleanup"

**Quick Reference**:

1. Get current PR: `ax pr get-current`
2. Get workspace: `basename "$PWD"`
3. Release claim: `ax prready release <PR> <WORKSPACE>`
4. Add needs-attention label: `ax pr labels manage --pr <PR> --add "needs-attention"`
5. Post comment explaining failure

**Why This Matters**:

- Prevents PR from being locked by failed preparation
- Allows other agents or humans to retry
- Maintains workflow state integrity
- Claims automatically expire after 30 minutes anyway, but explicit release is cleaner

---

## Automatic Phase Transition

After preparation completes successfully (via orchestrator or manual fallback), automatically
transition to verification phase:

### Transition Steps

**1. If preparation succeeded:**

```bash
# Transition to verification phase
ax prready update --phase verification

# MANDATORY: Execute verification immediately (function call, not stop point)
prompt pr-ready-cycle/verification
```

**2. If preparation failed:**

```bash
# Log error and return to discovery
echo "❌ Preparation failed for PR #$CURRENT_PR"
ax prready mark-processed "$CURRENT_PR"
git checkout main
ax prready update --phase discovery

# MANDATORY: Return to discovery immediately (function call, not stop point)
prompt pr-ready-cycle/discovery
```

**⚠️ CRITICAL ENFORCEMENT**: These `prompt` commands are **function calls** that execute
immediately, NOT stop points. Agent MUST execute them using Bash tool and continue with loaded
instructions without waiting for user confirmation.

---

## Related Documentation

- **Enforcement Orchestrator**: `ax prready enforce` - Primary implementation
- **PR Ready Cycle Workflow**: `~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` - Complete
  workflow specification with all commands
- **PR Verification Workflow**: `~/git/dotfiles/workflows/PR_VERIFICATION_WORKFLOW.md` -
  Verification phase details
- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
- **Validation Guide**: ~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW_VALIDATION.md
