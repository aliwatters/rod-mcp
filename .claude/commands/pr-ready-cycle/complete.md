# PR Ready Complete Phase

# ⚠️ DEPRECATED ⚠️

**This phase is DEPRECATED as of Issue #2914.**

**Use `ax prready run` instead**, which integrates the Complete phase logic directly into the
orchestrator.

**Migration Path:**

- Old: `prompt pr-ready-cycle/complete` (loads this 320-line prompt)
- New: `ax prready run --continuous` (integrated orchestrator)

**Rationale:** The Complete phase was a redundant orchestration layer that:

1. Checked continuous mode flag
2. Incremented counters
3. Decided next phase (Discovery or Retrospective)
4. Loaded another prompt

This logic now lives in the `ax prready run` orchestrator, eliminating:

- Unnecessary phase boundaries
- Fragile bash conditionals
- Python script calls for state updates
- Prompt loading overhead

**For legacy support only. Will be removed in future version.**

---

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

**Purpose**: Decide: loop back to Discovery or proceed to Retrospective

**⚠️ WORKSPACE ISOLATION**: Stay within starting workspace. NEVER cd to sibling directories.

Follow the PR Ready Cycle Workflow for Complete phase:
`~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` (Phase 5)

## Core Steps

### 1. Check Continuous Mode

Get continuous mode setting:

```bash
ax prready get | grep "^CONTINUOUS=" | cut -d= -f2
```

**Note**: If output is empty, default to "false".

### 2. Check Retrospective Interval (Continuous Mode Only)

```bash
if [ "$CONTINUOUS" = "true" ]; then
    # Get retrospective interval settings
    RETRO_INTERVAL=$(ax prready get | grep "^RETROSPECTIVE_INTERVAL=" | cut -d= -f2)
    PRS_SINCE_LAST_RETRO=$(ax prready get | grep "^PRS_SINCE_LAST_RETRO=" | cut -d= -f2)

    # Default values if not set
    RETRO_INTERVAL=${RETRO_INTERVAL:-10}
    PRS_SINCE_LAST_RETRO=${PRS_SINCE_LAST_RETRO:-0}

    # Increment counter (PR was just processed)
    PRS_SINCE_LAST_RETRO=$((PRS_SINCE_LAST_RETRO + 1))

    echo "📊 PRs since last retrospective: $PRS_SINCE_LAST_RETRO / $RETRO_INTERVAL"

    # Check if it's time for periodic retrospective
    if [ $PRS_SINCE_LAST_RETRO -ge $RETRO_INTERVAL ]; then
        echo "🔄 Retrospective interval reached - running periodic retrospective"

        # Reset counter using Python state management API
        python3 << EOF
import sys
sys.path.insert(0, '$HOME/git/dotfiles/tools/workflow')
from pr_ready_state import PRReadyState
state = PRReadyState()
state.update_state(phase='retrospective', PRS_SINCE_LAST_RETRO='0')
EOF

        # MANDATORY: Run periodic retrospective (will not clear state)
        prompt pr-ready-cycle/retrospective
    else
        echo "🔄 Continuous mode - returning to Discovery for next PR"

        # Update counter using Python state management API
        python3 << EOF
import sys
sys.path.insert(0, '$HOME/git/dotfiles/tools/workflow')
from pr_ready_state import PRReadyState
state = PRReadyState()
state.update_state(PRS_SINCE_LAST_RETRO='$PRS_SINCE_LAST_RETRO')
EOF

        # Update phase to Discovery
        ax prready update --phase discovery

        # MANDATORY: Loop back to Discovery phase
        prompt pr-ready-cycle/discovery
    fi
fi
```

**Behavior**:

- Immediately returns to Discovery phase
- Scans for next PR needing preparation
- Will sleep if no PRs found
- Runs indefinitely until interrupted

#### One-Shot Mode (Continuous Disabled)

```bash
if [ "$CONTINUOUS" = "false" ]; then
    echo "📊 One-shot mode - proceeding to Retrospective"

    # Update state to Retrospective phase
    ax prready update --phase retrospective

    # MANDATORY: Transition to Retrospective phase
    prompt pr-ready-cycle/retrospective
fi
```

**Behavior**:

- Proceeds to Retrospective for session analysis
- Generates improvement issues
- Clears state and exits
- Returns control to user

## Complete Logic Flow

```text
┌─────────────────────────────────────────┐
│ Complete Phase                          │
└──────────────┬──────────────────────────┘
               │
               ├─ Check: CONTINUOUS flag
               │
               ├─ If true:
               │   ├─ Check PRS_SINCE_LAST_RETRO >= INTERVAL?
               │   │   ├─ Yes → Retrospective (periodic, don't clear state) → Discovery
               │   │   └─ No  → Discovery (loop)
               │
               └─ If false:
                  └─ Retrospective (final, clear state, exit)
```

## Expected Output

### Continuous Mode

```bash

🔍 Continuous mode: true 🔄 Continuous mode enabled - returning to Discovery for next PR

➡️ Transitioning to Discovery phase...
```

### One-Shot Mode

```bash

🔍 Continuous mode: false 📊 One-shot mode - proceeding to Retrospective

➡️ Transitioning to Retrospective phase...
```

## Session Summary (Optional)

Before transitioning, optionally display session summary:

```bash
# Get metrics from state and display session summary
echo "📊 Session Summary:"
echo "   PRs processed: $(ax prready get | grep "^PRS_PROCESSED=" | cut -d= -f2 | tr ',' '\n' | grep -c '^[0-9]')"
echo "   Conflicts resolved: $(ax prready get | grep "^CONFLICTS_RESOLVED=" | cut -d= -f2)"
echo "   Lint fixes applied: $(ax prready get | grep "^LINT_FIXES=" | cut -d= -f2)"
```

**Example Output**:

```text
📊 Session Summary: PRs processed: 3 Conflicts resolved: 5 Lint fixes applied: 12
```

## State Management

### Clear Current PR

Ensure CURRENT_PR is cleared (should already be done in Verification phase):

```bash
# Verify current PR is cleared
CURRENT_PR=$(ax pr get-current)

if [ -n "$CURRENT_PR" ]; then
    echo "⚠️  Current PR not cleared, clearing now"
    ax prready update --clear-current-pr
fi
```

## Update Last Updated

State update automatically updates LAST_UPDATED timestamp:

```bash
# Update phase (automatically updates LAST_UPDATED)
ax prready update --phase discovery  # or retrospective
```

## MANDATORY Transitions

**✅ DO**:

- Check CONTINUOUS flag
- Execute `prompt pr-ready-cycle/discovery` (if continuous) OR `prompt pr-ready-cycle/retrospective` (if
  one-shot)
- Update state before transitioning

**❌ DON'T**:

- Execute multiple phases in single response
- Skip transition based on CONTINUOUS flag
- Exit without retrospective in one-shot mode

## Troubleshooting

### CONTINUOUS flag not found

**Cause**: State missing CONTINUOUS flag (corrupted state)

**Solution**:

```bash
# Set default
ax prready update --continuous false

# Or reinitialize state
ax prready clear
ax prready init
```

## Stuck in loop

**Cause**: Continuous mode enabled, user wants to stop

**Solution**:

- Interrupt with Ctrl+C
- Or set continuous to false:

  ```bash
  ax prready update --continuous false
  ```

---

**Complete Workflow**: See `~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md`

## MANDATORY: Automatic Transition

**⚠️ AUTOMATIC TRANSITION**: After completing the complete phase, automatically transition to the
next phase based on continuous mode.

**⚠️ CRITICAL**: This transition happens **IMMEDIATELY** without waiting for user input. Do NOT
pause or stop after executing the complete phase logic.

**Transition Logic**:

1. **If continuous mode is enabled (`CONTINUOUS=true`):**
   - Check if retrospective interval reached (`PRS_SINCE_LAST_RETRO >= RETRO_INTERVAL`)
   - If interval reached:
     - Update phase to `retrospective`
     - Automatically invoke `prompt pr-ready-cycle/retrospective`
   - Otherwise:
     - Update phase to `discovery`
     - Automatically invoke `prompt pr-ready-cycle/discovery`
   - **Do NOT wait for user confirmation**
   - **Do NOT pause between PRs**

2. **If one-shot mode (`CONTINUOUS=false`):**
   - Update phase to `retrospective`
   - Automatically invoke `prompt pr-ready-cycle/retrospective`
   - **Do NOT wait for user confirmation**

**Implementation Examples**:

```bash
# Continuous mode - return to discovery
if [ "$CONTINUOUS" = "true" ] && [ $PRS_SINCE_LAST_RETRO -lt $RETRO_INTERVAL ]; then
    ax prready update --phase discovery
    prompt pr-ready-cycle/discovery  # Execute immediately, no pause
fi

# Continuous mode - periodic retrospective
if [ "$CONTINUOUS" = "true" ] && [ $PRS_SINCE_LAST_RETRO -ge $RETRO_INTERVAL ]; then
    ax prready update --phase retrospective
    prompt pr-ready-cycle/retrospective  # Execute immediately, no pause
fi

# One-shot mode
if [ "$CONTINUOUS" = "false" ]; then
    ax prready update --phase retrospective
    prompt pr-ready-cycle/retrospective  # Execute immediately, no pause
fi
```

**⚠️ CRITICAL REMINDERS**:

- The `prompt` command **MUST** be executed immediately after the phase update
- **NEVER** pause or wait for user input between complete and next phase
- **NEVER** ask the user if they want to continue
- **ALWAYS** automatically invoke the next `prompt` command
- The transition is **MANDATORY** and **AUTOMATIC**
- In continuous mode, the cycle should run indefinitely without interruption

**Note**: This automatic transition is essential for continuous automation. The agent should execute
the transition commands immediately after completing the complete phase steps, without any user
intervention or confirmation prompts.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
