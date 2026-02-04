# Grooming Setup

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

Initialize Grooming Cycle workflow state and prepare for continuous issue grooming cycle.

## Overview

This phase initializes the Grooming Cycle workflow by:

- Creating the `.groom-issues-state` file
- Setting up continuous mode and session parameters
- Validating environment and GitHub access
- Recording session start time
- Transitioning to Discovery phase

## Prerequisites

- **GitHub CLI (`gh`)**: For issue operations and status checks
- **Git**: For repository validation
- **Python 3**: For running grooming helper scripts
- **Active repository**: Working in a valid Git repository with GitHub integration

## Setup Steps

### Step 1: Verify Environment

```bash
# Verify GitHub CLI is authenticated
gh auth status

# Verify Git is configured
git config --get user.name
git config --get user.email

# Verify Python 3 is available
python3 --version

# Verify we can list issues (basic GitHub access)
gh issue list --limit 1 >/dev/null
```

### Step 2: Initialize Grooming State

```bash
# Initialize state with default settings (continuous mode, 15-minute sleep)
ax groom init --continuous --sleep 15

# Verify state was created
ax groom get
```

### Step 3: Validate Repository State

```bash
# Ensure we're in the repository root
pwd

# Verify we have write access to the repository
git status

# Check that we can access issue labels
gh label list --limit 5 >/dev/null
```

### Step 4: Record Session Start

```bash
# Update state with session start time (already done by init)
ax groom get | grep SESSION_START
```

## Expected Output

```text
✅ Grooming state initialized PHASE=setup CONTINUOUS=true SLEEP_MINUTES=15
SESSION_START=2025-11-03T15:00:00 ISSUES_PROCESSED=0 ACTIONS_TAKEN=0 LAST_UPDATED=2025-11-03T15:00:00
```

## ⚠️ MANDATORY: Automatic Phase Transition

**🤖 AUTOMATED CONTINUATION REQUIRED - THIS IS NOT OPTIONAL**

As an AI agent executing the grooming-ax workflow, you MUST treat `prompt` commands as **function
calls that execute immediately**, NOT as user interaction requests.

**If setup completed successfully**, execute:

```bash
# Update workflow state
ax groom update --phase discovery

# MANDATORY: Execute discovery phase immediately
# Agent MUST treat this as a function call, not a wait point
prompt groom-backlog/discovery
```

**If setup failed**, clear state and exit with error:

```bash
# Note: ax groom clear command may not exist yet - use alternative
rm -f .groom-issues-state
echo "❌ Setup failed - state cleared"
exit 1
```

**CRITICAL ENFORCEMENT**:

- `prompt groom-backlog/discovery` is a **function call**, not a stop point
- Agent MUST execute it immediately using SlashCommand tool
- Agent MUST continue with loaded discovery instructions
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
