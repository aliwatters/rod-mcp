# PR Ready Prepare-for-Verified Phase

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

Re-validate PRs with "ready for humans" label to add "verified" label.

**⚠️ WORKSPACE ISOLATION**: Stay within starting workspace. NEVER cd to sibling directories.

**📖 Detailed Implementation**: For complete step-by-step instructions with all commands, see:
`~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` (Phase 3 - Prepare-for-Verified)

## Prerequisites

Before using this workflow, ensure you have:

- **GitHub CLI (`gh`)**: For PR operations and label management
- **Git**: For branch operations and merging
- **Python 3**: For running cycle commands and helper scripts
- **Active PR Ready Cycle**: This phase is part of the pr-ready-ax workflow
- **Workflow state**: Understanding of the `.pr-ready-state` file format
- **Workspace awareness**: Knowledge of multi-instance workspace patterns

### Authentication Setup

```bash
# Ensure GitHub CLI is authenticated
gh auth status

# If not authenticated:
gh auth login
```

---

## Core Steps

### Step 1: Get Current PR and Checkout Branch

**What to Do**:

- Get PR number from state: `ax prready get | grep "CURRENT_PR="`
- Get PR branch name: `ax pr view <PR_NUMBER> --json headRefName --jq '.headRefName'`
- Checkout PR branch: `git checkout <PR_BRANCH>`

### Step 2: Prepare PR (Merge, Fix, Test)

**⚠️ CRITICAL**: Execute all standard preparation steps before validation:

**Step 2.1: Check Mergeable Status and Resolve Conflicts Only if Needed**

```bash
# Check if PR has conflicts with main
MERGEABLE=$(gh pr view "$CURRENT_PR" --json mergeable --jq '.mergeable')

if [ "$MERGEABLE" = "CONFLICTING" ]; then
    echo "⚠️ Conflicts detected - merging main to resolve"
    git fetch origin main
    git merge origin/main --no-edit || {
        # Merge conflicts detected - cannot auto-resolve
        # Mark as needs work
        git merge --abort
        git checkout main
        ax pr labels manage --pr <PR_NUMBER> --add-pr-label "needs work"
        # Downgrade and exit (will be handled in Step 3)
    }
else
    echo "✅ No conflicts detected - skipping merge with main"
fi
```

**Step 2.2: Run Quality Gates**

```bash
# Run quality gates (linting, formatting, validation)
ax checkin || true
```

**Step 2.3: Run Tests**

```bash
# Run tests
ax checkin || {
    # Tests failing - mark as needs work
    git checkout main
    ax pr labels manage --pr <PR_NUMBER> --add-pr-label "needs work"
    # Downgrade and exit (will be handled in Step 3)
}
```

**Step 2.4: Verify Review Comments Addressed**

```bash
# Verify all review comments are acknowledged
# If verification fails, address comments using standard acknowledgment patterns
ax pr review-comments verify <PR_NUMBER>
# Use AI analysis to respond to unacknowledged comments with proper format:
# ✅ Fixed in <hash> - <description>
# ✅ Acknowledged - won't do <reason>
# ✅ Created issue #<number> for follow-up
```

**Step 2.5: Remove Stale Labels**

```bash
# Remove in-progress label if present
LABELS=$(ax pr get-labels <PR_NUMBER> --space-separated)
if echo "$LABELS" | grep -q "in-progress"; then
    ax pr labels manage --pr <PR_NUMBER> --remove-pr-label "in-progress"
fi
```

**Step 2.6: Commit and Push Fixes**

```bash
# Commit and push any fixes
if [ -n "$(git status --porcelain)" ]; then
    git add .
    git commit -m "chore: auto-fix PR #<PR_NUMBER> (merge, lint, tests, comments)"
    git push origin <PR_BRANCH>
fi

# Return to main
git checkout main
```

### Step 3: Run Validation and Add Verified Label

**What to Do**:

- Execute validation command: `ax pr validate-and-update <PR_NUMBER> --add-label --update-state`

**What this command does**:

- Validates all merge-readiness criteria:
  - No merge conflicts
  - All CI checks passing
  - All comments addressed
  - PR branch up-to-date with main
  - All tests passing
  - Linked to issue
- **Automatically adds 'verified' label if all checks pass**
- **Marks PR as processed if validation passes**
- **Updates workflow state and metrics**
- **Transitions to appropriate next phase**
- Returns exit code 0 if all checks pass and label added
- Returns exit code 1 if any check fails

**Note**: This is the core command for PR Ready Cycle phase management (Issue #2958)

### Step 4: Process Validation Result

**Success Path (Exit Code 0)**:

If validation passes:

1. ✅ Verified label already added by validate-and-update command
2. Add success comment with checklist
3. See "Automatic Transition" section (transitions to Verification phase)

**Note**: Do NOT mark as processed here - verification phase handles that.

**Failure Path (Exit Code 1)**:

If validation fails after preparation attempts:

1. Downgrade PR to "ready for robots":
   - Remove "ready for humans" label:
     `ax pr labels manage --pr <PR> --remove-pr-label "ready for humans"`
   - Add "ready for robots" label: `ax pr labels manage --pr <PR> --add-pr-label "ready for robots"`
2. Convert to draft: `ax pr ready <PR> --undo`
3. Add failure comment with details from validation output:

   ```bash
   ax pr comment <PR> --body "⚠️ **Verification Failed**

   This PR was prepared (merged with main, linting fixed, tests run, comments acknowledged) but still failed verification.

   **Validation Output**:
   [Include validation error details]

   **Next Steps**:
   - Review the validation errors above
   - Fix any remaining issues
   - PR will be retried in next pr-ready-cycle

   🤖 Automated by pr-ready-cycle"
   ```

4. **Classify failure and handle** (Issue #2126):

   ```bash
   # Classify the failure to determine retry behavior
   ax prready classify-failure \
     "$VALIDATION_ERROR" \
     --pr-labels "$(gh pr view <PR> --json labels --jq '.labels[].name | join(",")')" \
     --format json > /tmp/failure-classification.json

   # Handle failure based on classification (fixable vs non-fixable)
   # This command releases claim, conditionally marks as processed, and adds appropriate label
   ax prready handle-failure <PR> /tmp/failure-classification.json
   ```

5. **Transition to complete**:
6. Update phase: `ax prready update --phase complete`
7. Transition: `prompt pr-ready-cycle/complete`

**Note**: Fixable failures don't mark PR as processed, allowing retry in next cycle.

**How**: See workflow Phase 3 → "For prepare-for-verified Workflow" for complete commands and
comment templates

---

## Error Handling and Claim Cleanup (Issue #2126)

**⚠️ CRITICAL**: ALWAYS release PR claims on ALL paths (success, failure, error).

### Mandatory Claim Release

**Claims MUST be released in ALL scenarios**:

- ✅ **SUCCESS**: Validation passes and "verified" label added
- ❌ **FAILURE**: Validation fails (fixable or non-fixable)
- 🔥 **ERROR**: Workflow crashes, interrupts, or encounters blocking error

**Why**: Issue #2126 - Claims not released on failure block other agents for 15+ minutes.

### Universal Claim Release Pattern

**Use the release helper** (handles state, labels, comments):

```bash
# Get PR and workspace
PR_NUMBER=$(ax prready get | grep "CURRENT_PR=" | cut -d= -f2)
WORKSPACE=$(basename "$PWD")

# Release claim with reason
ax prready release-claim "$PR_NUMBER" \
  --workspace "$WORKSPACE" \
  --reason "validation completed/failed/error"
```

**Alternative** (if helper not available):

```bash
# Manual release (3 steps)
ax prready release "$PR_NUMBER" "$WORKSPACE"
ax pr labels manage --pr "$PR_NUMBER" --remove-pr-label "claimed"
ax pr comment "$PR_NUMBER" --body "🔓 Claim released: $REASON"
```

### Error Handling by Type

**Temporary/Fixable Errors** (API rate limit, network, timeout):

1. Classify failure: `ax prready classify-failure "$ERROR"`
2. Release claim (mandatory)
3. DON'T mark as processed (allow retry)
4. Add label: "needs-retry"

**Permanent/Non-fixable Errors** (merge conflict, test failure, missing file):

1. Classify failure
2. Release claim (mandatory)
3. Mark as processed (prevent retry loop)
4. Add label: "needs-attention"

---

## MANDATORY: Automatic Transition

**⚠️ AUTOMATIC TRANSITION**: After completing validation, automatically transition to the next
phase.

**Success Path Transition** (validation passed):

```bash
# Update phase to verification
ax prready update --phase verification

# MANDATORY: Transition to Verification phase
prompt pr-ready-cycle/verification
```

**Failure Path Transition** (validation failed):

```bash
# Update phase to complete
ax prready update --phase complete

# MANDATORY: Skip verification, go to Complete
prompt pr-ready-cycle/complete
```

**Note**: Success path goes to Verification (to add "verified" label), failure path goes to Complete
(skips verification).

**How**: See workflow Phase 3 → Transition sections for complete logic

---

## Troubleshooting

### Validation command hangs

**Cause**: Network issues or GitHub API rate limiting

**Solution**: Check API rate limits with `gh api rate_limit`, wait for reset if needed

### Cannot add/remove labels

**Cause**: Insufficient permissions or invalid label names

**Solution**: Check if labels exist with `gh label list`, create if needed

### Cannot convert to draft

**Cause**: PR already in draft state

**Solution**: Check current PR state with `ax pr view <PR>` before attempting conversion

---

**Complete Workflow**: See `~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` for all commands
and detailed logic

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
- **Validation Guide**: ~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW_VALIDATION.md
