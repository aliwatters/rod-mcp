# PR Ready Setup

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

Initialize PR Ready Cycle workflow state and prepare for continuous PR preparation cycle.

## Overview

This phase initializes the PR Ready Cycle workflow by:

- Creating the `.pr-ready-state` file
- Setting up continuous mode and sleep duration
- Syncing main branch with remote
- Recording session start time
- Transitioning to Discovery phase

## Prerequisites

- **GitHub CLI (`gh`)**: For PR operations and status checks
- **Git**: For branch management and merge operations
- **Python 3**: For running workflow helper scripts
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
```

### Step 2: Initialize PR Ready State

```bash
# Initialize state with default settings (continuous mode, 15-minute sleep)
ax prready init --continuous --sleep 15

# Verify state was created
ax prready get
```

### Step 3: Sync Main Branch

```bash
# Ensure we're on main branch
git checkout main

# Pull latest changes
git pull origin main
```

### Step 4: Record Session Start

```bash
# Update state with session start time (already done by init)
ax prready get | grep SESSION_START
```

## Expected Output

```text
✅ PR ready state initialized PHASE=setup CONTINUOUS=true SLEEP_MINUTES=15
SESSION_START=2025-10-24T14:30:00 CURRENT_PR= PRS_PROCESSED= LAST_UPDATED=2025-10-24T14:30:00
```

## ⚠️ MANDATORY: Automatic Phase Transition

**🤖 AUTOMATED CONTINUATION REQUIRED - THIS IS NOT OPTIONAL**

As an AI agent executing the pr-ready-ax workflow, you MUST treat `prompt` commands as **function
calls that execute immediately**, NOT as user interaction requests.

**If setup completed successfully**, execute:

```bash
# Update workflow state
ax prready update --phase discovery

# MANDATORY: Execute discovery phase immediately
# Agent MUST treat this as a function call, not a wait point
prompt pr-ready-cycle/discovery
```

**If setup failed**, clear state and exit with error:

```bash
ax prready clear
echo "❌ Setup failed - state cleared"
exit 1
```

**CRITICAL ENFORCEMENT**:

- `prompt pr-ready-cycle/discovery` is a **function call**, not a stop point
- Agent MUST execute it immediately using Bash tool
- Agent MUST continue with loaded discovery instructions
- NO stopping, NO waiting, NO summaries

**This is agent behavior enforcement.** The `prompt` command works correctly - agents must execute
it immediately, not wait for user confirmation.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
