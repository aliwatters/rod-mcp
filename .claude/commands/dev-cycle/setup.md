# Setup Phase: Development Cycle Initialization

**Purpose**: Intelligent state analysis and workspace preparation using compound tools.

**Phase**: 0 (Setup) - Executed before Discovery

**Pattern**: Analysis → AI Reasoning → Execution (see COMPOUND_TOOLS_GUIDE.md)

## Overview

The Setup phase uses **2 compound tool calls** instead of 7+ individual commands:

1. **Analysis Tool** (`ax_setup_analyze_state`) - Detects scenario and recommends action
2. **Execution Tool** (`ax_setup_execute_transition`) - Executes cleanup/initialization

**Benefits**:

- 71% reduction in Setup phase tool calls (7+ → 2)
- Intelligent scenario detection (8+ scenarios)
- Automatic stale state cleanup
- Time-based validation (<30min fresh, >30min stale)

## Prerequisites

**⚠️ MANDATORY**: Follow all instructions carefully - skipping steps will cause workflow failures.

**Required before Setup**:

- Workspace validation passed (`ax workflow validate-workspace`)
- Claude Code settings initialized (`ax claude sync-all-settings`)
- On main branch or feature branch (any starting point)

## Automation Level

**Automation Level**: SEMI-AUTOMATED

**Automation Continues When**:

- ✅ All validation checks pass
- ✅ No conflicts detected
- ✅ No blocking errors

**Automation Stops When**:

- ❌ Validation failures detected
- ❌ Manual intervention required
- ⚠️ Human decision needed

**See**: `~/git/dotfiles/docs/AUTOMATION_POLICY.md`

## Compound Tool Pattern

### Step 1: Analysis (Scenario Detection)

**MCP Tool (preferred)**:

```text
mcp__ax-mcp__ax_setup_analyze_state({
  "pre_instruction": {
    "type": "issue",
    "number": 3916
  }
})
```

**CLI Fallback**:

```bash
# Note: 'ax setup' command group doesn't exist
# Use 'ax workflow intelligent-setup' instead

# Without pre-instruction
ax workflow intelligent-setup

# With pre-instruction (issue)
ax workflow intelligent-setup --issue 3916

# With pre-instruction (PR)
ax workflow intelligent-setup --pr 3855
```

**Scenarios Detected**:

1. **Fresh start** (no state, on main)
2. **Resume existing work** (valid state, work <30min old)
3. **Resume on PR** (state + open PR exists)
4. **Handoff from pr-ready-cycle** (HANDOFF_FROM marker in state)
5. **Stale state cleanup needed** (PR merged/closed, issue closed/blocked, work >30min old)
6. **Pre-instruction override** (user specified issue/PR, conflicts with state)
7. **Feature branch without state** (on feature branch, no workflow state)
8. **Feature branch with stale state** (on feature branch, state doesn't match)

**Returns**:

```json
{
  "workspace": {
    "path": "/Users/ali/git/dotfiles-agent-3",
    "is_valid": true
  },
  "scenario": "resume_on_pr",
  "current_state": {
    "exists": true,
    "phase": "refinement",
    "issue": 3850,
    "pr": 3855,
    "is_stale": false,
    "age_minutes": 15
  },
  "pr_status": {
    "exists": true,
    "number": 3855,
    "draft": true,
    "state": "open",
    "has_copilot_review": false,
    "ready_for_refinement": true
  },
  "handoff_detection": {
    "is_handoff": false,
    "handoff_from": null,
    "resume_phase": null
  },
  "recommendation": {
    "action": "resume_at_refinement",
    "skip_to_phase": "refinement",
    "preserve_state": true,
    "reasoning": "PR exists with draft status, work is fresh (<30min)"
  },
  "intelligence_required": {
    "level": "low", // Setup is mostly deterministic
    "decisions": []
  }
}
```

### Step 2: AI Reasoning (Optional)

**When to apply reasoning**:

- `intelligence_required.level` is "medium" or "high" → Use extended thinking
- `intelligence_required.level` is "low" → Skip to execution

**Setup phase intelligence level**: Usually "low" (deterministic scenarios)

**Exception scenarios requiring reasoning**:

- Pre-instruction conflicts with state (user says issue #123, state says issue #456)
- Ambiguous state (feature branch with no state, unclear if new work or abandoned)
- Multiple valid options (could resume or start fresh)

**Example reasoning (if needed)**:

```markdown
Using `ultrathink`: Resolve pre-instruction conflict

**Situation**:

- Pre-instruction: "work on issue #3916"
- Current state: ISSUE=3850, PHASE=implementation, age=45min

**Analysis**:

- State is stale (>30min old, work abandoned?)
- User explicitly requested different issue
- Current branch: feature/issue-3850 (doesn't match #3916)

**Options**:

1. Clear stale state, work on #3916 (fresh start)
2. Resume #3850, ignore pre-instruction (continue existing work)

**Decision**: Option 1 - Clear stale state, prioritize pre-instruction

- User intent is explicit (#3916)
- Stale work suggests abandonment
- Clean start preferred for clarity

**Action**: `cleanup_and_fresh_start` then initialize with issue #3916
```

### Step 3: Execution (Cleanup and Initialization)

**MCP Tool (preferred)**:

```text
mcp__ax-mcp__ax_setup_execute_transition({
  "action": "resume_at_refinement",
  "preserve_state": true,
  "context": {
    "issue": 3850,
    "pr": 3855
  }
})
```

**CLI Fallback**:

```bash
# Note: 'ax setup execute-transition' doesn't exist
# This is an MCP-only compound tool - no CLI equivalent
# Use MCP tool above or manual multi-step commands:

ax workflow update --phase refinement --issue 3850 --pr 3855
ax git check-branch
```

**Actions Available**:

- `init_new_workflow` - Fresh start (clear state, init discovery)
- `resume_at_discovery` - Resume work at discovery phase
- `resume_at_implementation` - Resume work at implementation phase
- `resume_at_refinement` - Resume work at refinement phase
- `resume_at_retrospective` - Resume work at retrospective phase
- `resume_at_verify` - Resume work at verify phase
- `cleanup_and_fresh_start` - Clear stale state, then fresh start
- `cleanup_and_restart` - Clear stale state, then restart at discovery

**Internal Execution** (example for `resume_at_refinement`):

- Logs session data: "Setup: resuming at refinement phase (PR #3855)"
- Validates state consistency
- Verifies feature branch
- Proceeds to refinement phase

<details>
<summary><b>Reference Implementation</b></summary>

```bash
# Reference: What the tool executes internally
ax session append --data "Setup: resuming at refinement phase (PR #3855)"
ax workflow validate
ax git check-branch
echo "Skipping to refinement phase"
```

</details>

**Internal Execution** (example for `cleanup_and_fresh_start`):

- Session logging via ax session append
- Clear workflow state via ax workflow clear
- If on feature branch: Return to main branch and pull latest changes
- Initialize fresh discovery phase via ax workflow init

<details>
<summary><b>Reference Implementation</b></summary>

```bash
# Reference: What the tool executes internally
ax session append --data "Setup: cleaning up stale state, starting fresh"
ax workflow clear
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
  git checkout main
  git pull origin main
fi
ax workflow init --phase discovery
```

</details>

**Returns**:

```json
{
  "success": true,
  "action_taken": "resume_at_refinement",
  "next_phase": "refinement",
  "context": {
    "issue": 3850,
    "pr": 3855
  },
  "session_logged": true
}
```

## Scenario Handling Guide

### Scenario 1: Fresh Start (No State, On Main)

**Detection**:

- No workflow state exists
- Currently on `main` branch
- No pre-instruction provided

**Recommendation**: `init_new_workflow`

**Actions**:

1. Initialize workflow state with `phase=discovery`
2. Proceed to discovery phase

**Example**:

```bash
# Analysis detects: scenario="fresh_start"
# AI: No reasoning needed (low intelligence)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "init_new_workflow", "preserve_state": false})
```

### Scenario 2: Resume Existing Work (Valid State, Fresh)

**Detection**:

- Workflow state exists
- State is valid (issue open, PR open/none)
- Work is fresh (<30min old)
- No pre-instruction conflicts

**Recommendation**: `resume_at_<current_phase>`

**Actions**:

1. Validate state consistency
2. Resume at current phase

**Example**:

```bash
# Analysis detects: scenario="resume_existing_work", phase="implementation", age=15min
# AI: No reasoning needed (low intelligence)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "resume_at_implementation", "preserve_state": true})
```

### Scenario 3: Resume on PR (State + Open PR)

**Detection**:

- Workflow state exists with PR number
- PR is open (draft or ready)
- No Copilot review yet → refinement phase
- Copilot review exists → verify phase

**Recommendation**: `resume_at_refinement` or `resume_at_verify`

**Actions**:

1. Detect PR status (draft vs ready, reviews exist)
2. Skip to appropriate phase

**Example**:

```bash
# Analysis detects: scenario="resume_on_pr", pr=3855, draft=true, copilot_review=false
# AI: No reasoning needed (low intelligence)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "resume_at_refinement", "preserve_state": true, "context": {"pr": 3855}})
```

### Scenario 4: Handoff from pr-ready-cycle

**Detection**:

- Workflow state contains `HANDOFF_FROM=pr-ready-cycle`
- Contains `RESUME_PHASE=implementation` or `refinement`

**Recommendation**: `resume_at_<handoff_phase>`

**Actions**:

1. Acknowledge handoff
2. Resume at specified phase
3. Clear handoff markers

**Example**:

```bash
# Analysis detects: scenario="handoff_resumption", handoff_from="pr-ready-cycle", resume_phase="implementation"
# AI: No reasoning needed (low intelligence)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "resume_at_implementation", "preserve_state": true, "context": {"clear_handoff": true}})
```

### Scenario 5: Stale State Cleanup Needed

**Detection**:

- PR merged or closed
- Issue closed or blocked
- Work is stale (>30min old with no activity)

**Recommendation**: `cleanup_and_fresh_start`

**Actions**:

1. Clear workflow state
2. Return to main branch
3. Pull latest changes
4. Initialize fresh discovery

**Example**:

```bash
# Analysis detects: scenario="stale_state_cleanup", pr=3855, pr_state="merged", age=120min
# AI: No reasoning needed (low intelligence)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "cleanup_and_fresh_start", "preserve_state": false})
```

### Scenario 6: Pre-Instruction Override

**Detection**:

- User provided pre-instruction (issue/PR number)
- Conflicts with current state OR state is stale
- User intent is explicit

**Recommendation**: `cleanup_and_fresh_start` then initialize with pre-instruction

**Actions**:

1. Clear conflicting state
2. Initialize with pre-instruction issue/PR
3. Proceed to appropriate phase

**Example**:

```bash
# Analysis detects: scenario="pre_instruction_override", pre_instruction="issue #3916", current_state="issue #3850, stale"
# AI: May need reasoning (if conflict is ambiguous)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "cleanup_and_fresh_start", "preserve_state": false, "context": {"issue": 3916}})
```

### Scenario 7: Feature Branch Without State

**Detection**:

- Currently on feature branch (not main)
- No workflow state exists
- Branch may or may not have PR

**Recommendation**: Depends on PR existence

- PR exists → `resume_at_refinement` (initialize state from PR)
- No PR → `cleanup_and_fresh_start` (abandoned work)

**Actions**:

- With PR: Initialize state from PR, resume refinement
- Without PR: Return to main, fresh start

**Example (with PR)**:

```bash
# Analysis detects: scenario="feature_branch_no_state", branch="feature/issue-3850", pr=3855
# AI: No reasoning needed (low intelligence)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "resume_at_refinement", "preserve_state": false, "context": {"initialize_from_pr": 3855}})
```

**Example (without PR)**:

```bash
# Analysis detects: scenario="feature_branch_no_state", branch="feature/issue-3850", pr=null
# AI: May need reasoning (abandoned work?)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "cleanup_and_fresh_start", "preserve_state": false})
```

### Scenario 8: Feature Branch with Stale State

**Detection**:

- On feature branch
- Workflow state exists but is stale (>30min)
- State may not match branch

**Recommendation**: Depends on staleness severity

- Moderately stale (30-60min) → Resume if issue/PR still valid
- Very stale (>60min) → Cleanup and fresh start

**Actions**:

- Validate issue/PR still relevant
- Clear if stale, resume if fresh

**Example**:

```bash
# Analysis detects: scenario="feature_branch_stale_state", age=90min, issue=3850, issue_state="closed"
# AI: No reasoning needed (issue closed = clear signal)
# Execution (MCP tool - preferred)
mcp__ax-mcp__ax_setup_execute_transition({"action": "cleanup_and_fresh_start", "preserve_state": false})
```

## Session Logging

All execution tool calls automatically log to session for audit trail:

```bash
# Automatic session logging (done by ax_setup_execute_transition)
ax session append --data "$(cat <<'EOF'
Setup: <action_taken>
Scenario: <scenario_name>
Decision: <recommendation>
Reasoning: <reasoning_if_applicable>
EOF
)"
```

**Example log entry**:

```text
Setup: resume_at_refinement
Scenario: resume_on_pr
Decision: Skip to refinement (PR #3855 draft, no Copilot review yet)
Reasoning: Deterministic - PR state indicates refinement phase readiness
```

## Error Handling

### Analysis Tool Failures

**Error**: Workspace validation failed

```json
{
  "error": {
    "type": "workspace_validation_failed",
    "message": "Cross-workspace symlinks detected",
    "remediation": "Run: ax workflow validate-workspace --fix"
  }
}
```

**Action**: Fix workspace issues, retry analysis

### Execution Tool Failures

**Error**: State clear failed

```json
{
  "success": false,
  "error": {
    "type": "state_clear_failed",
    "message": "Unable to clear workflow state file",
    "remediation": "Manual cleanup required: rm .workflow-state.*"
  }
}
```

**Action**: Manual intervention, then retry

## Testing Setup Phase

### Unit Tests

```bash
pytest tools/ax-mcp/tests/test_setup_analyze.py
pytest tools/ax-mcp/tests/test_setup_execute.py
```

### Integration Tests

```bash
# Test all 8 scenarios
pytest specifications/phase-0-setup-specification.feature
```

### Manual Testing

```bash
# Test scenario 1: Fresh start
git checkout main
rm -f .workflow-state.*
prompt dev-cycle

# Test scenario 2: Resume work
# (create state with phase=implementation, age=15min)
prompt dev-cycle

# Test scenario 3: Resume on PR
# (create state with PR, draft=true)
prompt dev-cycle

# ... test all 8 scenarios
```

## Benefits

- ✅ **71% fewer tool calls in Setup phase** - 2 instead of 7+
- ✅ **Intelligent scenario detection** - Handles 8+ scenarios automatically
- ✅ **Automatic cleanup** - Clears stale state (merged/closed PRs, closed/blocked issues)
- ✅ **Time-based validation** - Fresh work resumes (<30min), stale re-validates (>30min)
- ✅ **Handoff support** - Seamless pr-ready-cycle handoffs
- ✅ **Session logging** - Full audit trail of decisions
- ✅ **Pre-instruction support** - User can override with explicit issue/PR
- ✅ **Minimal overhead** - Single command replaces 5+ manual steps

## Related Documentation

- **Compound Tools Guide**: `~/git/dotfiles/prompts/dev-cycle/COMPOUND_TOOLS_GUIDE.md`
- **Setup Specification**: `~/git/dotfiles/specifications/phase-0-setup-specification.feature`
- **MCP Tool Usage**: `~/git/dotfiles/docs/MCP_TOOL_USAGE_GUIDE.md`
- **Automation Policy**: `~/git/dotfiles/docs/AUTOMATION_POLICY.md`
- **Refactor Spec**: `~/git/dotfiles/docs/future-work/DEV_CYCLE_MCP_REFACTOR_SPEC.md`

## Next Phase

After Setup completes, orchestrator proceeds to the recommended phase:

- `init_new_workflow` → Discovery
- `resume_at_*` → Specified phase (discovery, implementation, refinement, etc.)
- `cleanup_and_fresh_start` → Discovery
