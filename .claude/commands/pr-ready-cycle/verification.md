# PR Ready Verification Phase

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

**Purpose**: Validate PR is ready for human review

**⚠️ WORKSPACE ISOLATION**: Stay within starting workspace. NEVER cd to sibling directories.

Follow the PR Ready Cycle Workflow for Verification phase:
`~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` (Phase 4)

## Core Steps

### 1. Get Current PR

```bash
# Get PR number from state
CURRENT_PR=$(ax pr get-current)

if [ -z "$CURRENT_PR" ]; then
    echo "❌ No current PR in state"
    exit 1
fi

echo "✅ Verifying PR #$CURRENT_PR"
```

### 2. Check PR Workflow Stage (NEW - Issue #1524)

**⚠️ CRITICAL**: Check if PR is ready-for-review before attempting verification. Skip verification
if PR is still in draft/in-progress state.

```bash
# Check PR workflow stage
PR_STAGE=$(ax pr get-stage "$CURRENT_PR")

echo "📋 PR #$CURRENT_PR workflow stage: $PR_STAGE"

if [ "$PR_STAGE" = "draft" ] || [ "$PR_STAGE" = "in-progress" ]; then
    echo "⏭️  Skipping verification - PR is in $PR_STAGE state"
    echo "ℹ️  PR must be ready-for-review before verification can succeed"

    # Add comment explaining why verification was skipped
    ax pr comment "$CURRENT_PR" --body "⏭️ **Verification Skipped**

PR is currently in \`$PR_STAGE\` state. Verification will be attempted when PR is ready-for-review.

**Next steps:**
- Complete work on PR
- Mark as ready-for-review when appropriate
- Verification will run automatically on next cycle"

    echo "✅ Added skip comment to PR"

    # Clear current PR and transition to complete
    ax prready update --clear-current-pr
    ax prready update --phase complete
    prompt pr-ready-cycle/complete
    exit 0
fi

echo "✅ PR is ready-for-review - proceeding with verification"
```

### 3. Verify Copilot Feedback (BLOCKING)

**⚠️ CRITICAL**: Explicitly check for Copilot feedback before running general verification gate.

```bash
# Step 3a: Verify Copilot feedback (BLOCKING - must pass before gate)
echo "🤖 Checking Copilot feedback..."
ax pr verification verify-feedback "$CURRENT_PR"
if [ $? -ne 0 ]; then
    echo "❌ Unaddressed Copilot feedback found"
    exit 1
fi
echo "✅ All Copilot feedback addressed"

# Step 3b: Run comprehensive verification gate
ax pr gate --pr "$CURRENT_PR"
if [ $? -ne 0 ]; then
    echo "❌ PR verification gate failed"
    exit 1
fi
echo "✅ PR verification gate passed"
```

**Verification Criteria**:

1. ✅ All Copilot feedback addressed with threaded replies (EXPLICIT CHECK - Issue #2500)
2. ✅ PR workflow stage is ready-for-review (not draft/in-progress)
3. ✅ PR branch up-to-date with main at time of check
4. ✅ All CI checks passing (Issue #1831: waits up to 5 minutes for pending checks to complete)
5. ✅ Mergeable (no conflicts)
6. ✅ All review comments acknowledged with threaded replies (if any exist)
7. ✅ Required labels present
8. ✅ Linked issue exists

### 4. Handle Verification Results

#### All Checks Passed

```bash
if [ $GATE_RESULT -eq 0 ]; then
    echo "✅ PR #${CURRENT_PR} verified and ready for humans"

    # Clear retry count since verification succeeded
    ax prready clear-retry-count "$CURRENT_PR"

    # Get current labels first
    CURRENT_LABELS=$(ax pr view "$CURRENT_PR" --json labels --jq '.labels[].name' | tr '\n' ' ')

    # Define labels to remove (all workflow labels except target state)
    LABELS_TO_REMOVE="needs-attention draft in-progress blocked ready for robots claimed"

    # Remove each label if present
    for label in $LABELS_TO_REMOVE; do
        if echo "$CURRENT_LABELS" | grep -qF "$label"; then
            ax pr labels manage --pr "$CURRENT_PR" --remove "$label"
            echo "   Removed: $label"
        fi
    done

    # Add 'verified' label (ready for humans is added by ax pr labels manage)
    ax pr labels manage --pr "$CURRENT_PR" --add "verified"

    # CRITICAL: Mark PR as ready using cycle command (handles labels + draft status)
    ax pr labels manage --pr "$CURRENT_PR" --add "ready for humans" --remove "ready for robots"
    echo "✅ Marked PR #$CURRENT_PR as ready for review (labels updated + draft removed)"

    # NEW (Issue #2413, #2423): Wait for CI to stabilize before releasing claim
    # Extracted to CLI command to reduce prompt size
    ax pr lifecycle wait-ci-stable "$CURRENT_PR"
    CI_WAIT_RESULT=$?

    if [ $CI_WAIT_RESULT -eq 1 ]; then
        echo "⚠️  Warning: CI did not stabilize within timeout"
        echo "   Proceeding with claim release anyway"
    elif [ $CI_EXIT -eq 3 ]; then
        echo "⚠️  Transient infrastructure error during CI stabilization (e.g., API/network issue)"
        echo "   Proceeding with claim release, but CI status may be unreliable."
    else
        echo "⚠️  Warning: CI stabilization check encountered unknown errors (exit code $CI_EXIT)"
        echo "   Proceeding with claim release anyway"
    fi

    # Release the claim on this PR
    WORKSPACE=$(basename "$PWD")
    ax prready release "$CURRENT_PR" "$WORKSPACE"
    echo "🔓 Released claim on PR #$CURRENT_PR"

    # Release claim on linked issue
    LINKED_ISSUE=$(ax pr view "$CURRENT_PR" --json body --jq '.body' | grep -oE 'Fixes #[0-9]+|Closes #[0-9]+|Resolves #[0-9]+' | grep -oE '[0-9]+' | head -1)

    if [ -n "$LINKED_ISSUE" ]; then
        echo "🔓 Releasing claim on issue #$LINKED_ISSUE"
        gh issue edit "$LINKED_ISSUE" --remove-label "claimed"

        if [ $? -eq 0 ]; then
            echo "✅ Removed 'claimed' label from issue #$LINKED_ISSUE"
        else
            echo "⚠️  Failed to remove 'claimed' label from issue #$LINKED_ISSUE (non-fatal)"
        fi
    fi

    # Add comment
    ax pr comment "$CURRENT_PR" --body "✅ **PR Verified**

This PR has passed all automated verification checks:
- ✅ No merge conflicts
- ✅ All CI checks passing
- ✅ All comments acknowledged
- ✅ Meets quality standards

Ready for human review."

    echo "🏷️  Labels cleaned: only 'ready for humans' + 'verified' remain"

    # Validate final label state (Fix #3 for issue #2132)
    FINAL_LABELS=$(ax pr view "$CURRENT_PR" --json labels --jq '.labels[].name' | tr '\n' ',')

    # Check for contradictions
    if echo "$FINAL_LABELS" | grep -q "ready for robots" && echo "$FINAL_LABELS" | grep -q "ready for humans"; then
        echo "⚠️  WARNING: PR has contradictory labels (both ready for robots and humans)"
        echo "   Removing 'ready for robots'..."
        ax pr labels manage --pr "$CURRENT_PR" --remove "ready for robots"
    fi

    if echo "$FINAL_LABELS" | grep -q "claimed"; then
        echo "⚠️  WARNING: PR still has 'claimed' label"
        echo "   Removing 'claimed'..."
        ax pr labels manage --pr "$CURRENT_PR" --remove "claimed"
    fi

    if echo "$FINAL_LABELS" | grep -q "in-progress" && echo "$FINAL_LABELS" | grep -q "ready for humans"; then
        echo "⚠️  WARNING: PR has contradictory labels (both in-progress and ready for humans)"
        echo "   Removing 'in-progress'..."
        ax pr labels manage --pr "$CURRENT_PR" --remove "in-progress"
    fi

    echo "✅ Final label state validated"
fi
```

#### Some Checks Failed

```bash
if [ $GATE_RESULT -ne 0 ]; then
    echo "⚠️  PR #${CURRENT_PR} verification failed"

    # Get detailed failure reasons from gate output
    FAILURES=$(ax pr gate --pr "$CURRENT_PR" --verbose 2>&1)

    # Check if failure is CI-related (Issue #1832)
    if echo "$FAILURES" | grep -qi "CI checks"; then
        echo "🔍 Detected CI failure - checking complexity before creating fix-ci issue"

        # Get PR metadata for issue creation
        PR_TITLE=$(gh pr view "$CURRENT_PR" --json title --jq '.title')
        PR_BRANCH=$(gh pr view "$CURRENT_PR" --json headRefName --jq '.headRefName')
        PR_AUTHOR=$(gh pr view "$CURRENT_PR" --json author --jq '.author.login')

        # Extract failing check names and logs for complexity analysis
        FAILING_CHECKS_JSON=$(gh pr checks "$CURRENT_PR" --json name,conclusion,state --jq '.[] | select(.conclusion == "FAILURE" or .state == "FAILURE")' 2>/dev/null || echo "")

        if [ -n "$FAILING_CHECKS_JSON" ]; then
            FAILING_CHECK_NAMES=$(echo "$FAILING_CHECKS_JSON" | jq -r '.name' | tr '\n' ',' | sed 's/,$//')
            FAILING_CHECK_COUNT=$(echo "$FAILING_CHECKS_JSON" | jq -s 'length')

            # Check if CI failures are test-related and complex
            # Get test output from CI logs (using gh run view if available)
            TEST_OUTPUT=$(gh pr checks "$CURRENT_PR" 2>&1 || echo "")

            echo "🔍 Checking if test failures are too complex for pr-ready automation..."

            python3 -c "
from cycle.core.complexity import detect_complex_test_failures
test_output = '''$TEST_OUTPUT'''
is_complex, reason = detect_complex_test_failures(test_output)
if is_complex:
    print(f'COMPLEX: {reason}')
    exit(1)
" 2>/dev/null

            if [ $? -eq 1 ]; then
                echo "⚠️  Complex test failures detected - returning PR to draft for dev-cycle"

                # Passive state management: fix labels and return to draft
                # Dev-cycle will discover this PR naturally
                gh pr edit "$CURRENT_PR" --remove-label "ready for robots" 2>/dev/null || true
                gh pr edit "$CURRENT_PR" --remove-label "ready for humans" 2>/dev/null || true
                gh pr edit "$CURRENT_PR" --remove-label "verified" 2>/dev/null || true
                gh pr edit "$CURRENT_PR" --remove-label "claimed" 2>/dev/null || true

                # Return to draft status
                gh pr ready --undo "$CURRENT_PR" 2>/dev/null || true

                # Add label to indicate why it went back
                gh pr edit "$CURRENT_PR" --add-label "needs-refinement"

                echo "✅ PR #$CURRENT_PR returned to draft - dev-cycle will pick it up"

                # Mark PR as processed to prevent retry loop
                ax prready mark-processed "$CURRENT_PR"

                # Clear current PR and transition to complete
                ax prready update --clear-current-pr
                ax prready update --phase complete
                prompt pr-ready-cycle/complete
                exit 0
            else
                echo "✅ Test failures are simple - can be handled by pr-ready automation"
            fi

            # Check if fix-ci issue already exists for this PR
            EXISTING_ISSUE=$(gh issue list --label "ci" --state open --json number,title,body --jq ".[] | select(.body | contains(\"PR: #$CURRENT_PR\")) | .number" | head -1)

            if [ -n "$EXISTING_ISSUE" ]; then
                echo "ℹ️  Fix-CI issue already exists: #$EXISTING_ISSUE"
            else
                echo "📝 Creating fix-ci issue for PR #$CURRENT_PR"

                # Create fix-ci issue with standardized format
                ISSUE_BODY="## Problem

PR #$CURRENT_PR failed verification due to CI check failures.

**Failing checks ($FAILING_CHECK_COUNT):**
$FAILING_CHECK_NAMES

**PR Details:**
- Title: $PR_TITLE
- Branch: \`$PR_BRANCH\`
- Author: @$PR_AUTHOR

## Steps to Fix

1. Checkout PR branch: \`git checkout $PR_BRANCH\`
2. Identify root cause of CI failures
3. Fix the failing checks
4. Verify locally: \`ax checkin\`
5. Push fixes
6. Wait for CI to re-run
7. Verification will automatically retry on next cycle

## Related

- PR: #$CURRENT_PR
- Auto-generated by PR verification workflow (Issue #1832)

## Acceptance Criteria

- [ ] All CI checks passing on PR #$CURRENT_PR
- [ ] PR can be verified successfully
- [ ] Root cause documented in this issue"

                NEW_ISSUE=$(gh issue create \
                    --title "fix: Resolve CI failures in PR #$CURRENT_PR" \
                    --label "bug,ci,priority:high,ready for robots" \
                    --body "$ISSUE_BODY" \
                    --json number --jq '.number')

                if [ $? -eq 0 ]; then
                    echo "✅ Created fix-ci issue: #$NEW_ISSUE"

                    # Add comment to PR linking to the fix-ci issue
                    ax pr comment "$CURRENT_PR" --body "🤖 **Auto-created Fix-CI Issue**

A dedicated issue has been created to track fixing the CI failures: #$NEW_ISSUE

The issue contains:
- List of failing checks
- Steps to reproduce and fix
- Acceptance criteria

Agents can claim and work on this issue to unblock the PR."
                else
                    echo "⚠️  Failed to create fix-ci issue (non-fatal)"
                fi
            fi
        fi
    fi

    # Check if failures are fixable (only match failure patterns, not success patterns)
    # Issue #2500: Include unaddressed Copilot feedback as a fixable issue
    if echo "$FAILURES" | grep -q "not up-to-date with main\|Comments not acknowledged\|unaddressed Copilot comments\|Unaddressed Copilot comments found"; then
        # Fixable - get retry count with robust error handling
        RETRY_COUNT=$({ ax prready get-retry-count "$CURRENT_PR" | grep -o '[0-9]\+$'; } 2>/dev/null || echo "0")

        if [ "$RETRY_COUNT" -lt 3 ]; then
            echo "⚠️  Verification failed with fixable issues (attempt $((RETRY_COUNT + 1))/3)"
            echo "🔄 Retrying preparation..."

            # Increment retry counter
            ax prready set-retry-count "$CURRENT_PR" $((RETRY_COUNT + 1))

            # Return to preparation phase using workflow composition
            ax prready update --phase preparation
            prompt workflow workflows/PR_READY_CYCLE_WORKFLOW.md
            exit 0
        else
            echo "❌ Max retries exceeded (3 attempts)"
            # NOW mark as processed and add needs-attention with improved label management

            # Get current labels
            CURRENT_LABELS=$(ax pr view "$CURRENT_PR" --json labels --jq '.labels[].name' | tr '\n' ' ')

            # Define labels to remove (contradictory labels)
            LABELS_TO_REMOVE="verified in-progress draft claimed ready for humans"

            # Remove each label if present
            for label in $LABELS_TO_REMOVE; do
                if echo "$CURRENT_LABELS" | grep -qF "$label"; then
                    ax pr labels manage --pr "$CURRENT_PR" --remove "$label"
                    echo "   Removed: $label"
                fi
            done

            # Add needs-attention label
            ax pr labels manage --pr "$CURRENT_PR" --add "needs-attention"

            ax prready mark-processed "$CURRENT_PR"
        fi
    else
        # Not fixable (e.g., linked issue closed, PR complexity too high)
        echo "⚠️  Verification failed with non-fixable issues"

        # Get current labels
        CURRENT_LABELS=$(ax pr view "$CURRENT_PR" --json labels --jq '.labels[].name' | tr '\n' ' ')

        # Define labels to remove (contradictory labels)
        LABELS_TO_REMOVE="verified in-progress draft claimed ready for humans"

        # Remove each label if present
        for label in $LABELS_TO_REMOVE; do
            if echo "$CURRENT_LABELS" | grep -qF "$label"; then
                ax pr labels manage --pr "$CURRENT_PR" --remove "$label"
                echo "   Removed: $label"
            fi
        done

        ax pr labels manage --pr "$CURRENT_PR" --add "blocked"
        ax prready mark-processed "$CURRENT_PR"
    fi

    # Release the claim on this PR so it can be retried
    WORKSPACE=$(basename "$PWD")
    ax prready release "$CURRENT_PR" "$WORKSPACE"
    echo "🔓 Released claim on PR #$CURRENT_PR (verification failed)"

    # Add comment with failure details
    ax pr comment "$CURRENT_PR" --body "⚠️ **PR Needs Attention**

Some verification checks failed:

$FAILURES

Please address these issues before requesting human review."

    echo "🏷️  Labels updated: 'needs-attention' added, contradictory labels removed"

    # Increment complex issues metric
    ax prready increment-metric complex_issues
fi
```

### 5. Clear Current PR

```bash
# Clear current PR from state (preparation complete)
ax prready update --clear-current-pr
```

### 6. Transition to Complete

```bash
# Update state
ax prready update --phase complete

# MANDATORY: Transition to Complete phase
prompt pr-ready-cycle/complete
```

## Success Scenarios

### Fully Verified PR

```text
✅ Verifying PR #1234

Running verification gate... ✅ Merged with main (2 hours ago) ✅ All CI checks passing (5/5) ✅ No
merge conflicts ✅ All comments acknowledged (3/3) ✅ Has required labels: ready for humans ✅
Linked to issue #100

✅ PR #1234 verified and ready for humans 🏷️ Added 'verified' label

➡️ Transitioning to Complete phase...
```

### Partially Ready PR

```text
✅ Verifying PR #1235

Running verification gate... ✅ Merged with main (1 hour ago) ⚠️ CI checks: 4/5 passing (1 failing:
lint) ✅ No merge conflicts ✅ All comments acknowledged (2/2) ✅ Has required labels: ready for
robots ✅ Linked to issue #101

⚠️ PR #1235 needs attention 🏷️ Added 'needs-attention' label

Failed checks:

- Lint check failing: please fix ruff violations

➡️ Transitioning to Complete phase...
```

## Label Management

**⚠️ CRITICAL**: Always maintain mutually exclusive label states to prevent contradictions.

### When Verification SUCCEEDS

```bash
# Add verified, remove needs-attention (mutually exclusive)
ax pr labels manage --pr "$CURRENT_PR" --add "verified" --remove-label "needs-attention"

# Also remove draft/in-progress labels if present
ax pr labels manage --pr "$CURRENT_PR" --remove "draft"
ax pr labels manage --pr "$CURRENT_PR" --remove "in-progress"
```

**Result**: PR will have "verified" but NOT "needs-attention"

### When Verification FAILS

```bash
# Add needs-attention, remove verified (mutually exclusive)
ax pr labels manage --pr "$CURRENT_PR" --add "needs-attention" --remove-label "verified"
```

**Result**: PR will have "needs-attention" but NOT "verified"

### Label State Rules

**Valid states:**

- ✅ "ready for humans" + "verified" (ready to merge)
- ✅ "ready for humans" + "needs-attention" (issues to fix)
- ✅ "ready for robots" + "verified" (automated merge candidate)

**Invalid states (prevented by this workflow):**

- ❌ "verified" + "needs-attention" (contradictory)
- ❌ "verified" + "draft" (contradictory)
- ❌ "verified" + "in-progress" (contradictory)

## Comment Templates

### Success Comment

```markdown
✅ **PR Verified**

This PR has passed all automated verification checks:

- ✅ No merge conflicts
- ✅ All CI checks passing
- ✅ All comments acknowledged
- ✅ Meets quality standards

Ready for human review.
```

### Failure Comment

```markdown
⚠️ **PR Needs Attention**

Some verification checks failed:

- ⚠️ CI checks: 1 failing
- ⚠️ Merge conflicts detected

Please address these issues before requesting human review.
```

## Expected Output

See "Success Scenarios" above for example outputs.

## MANDATORY Transitions

**✅ DO**:

- Execute `prompt pr-ready-cycle/complete` at the end
- Update state before transitioning
- Clear current PR from state
- Add appropriate labels

**❌ DON'T**:

- Execute multiple phases in single response
- Skip complete phase
- Leave PR in ambiguous state (always label: verified OR needs-attention)

---

**Complete Workflow**: See `~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md`

## MANDATORY: Automatic Transition

After verification completes, automatically transition to Complete phase without user confirmation.

**Full verification criteria and detailed examples**: See
`~/git/dotfiles/workflows/PR_READY_CYCLE_WORKFLOW.md` (Phase 4)

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
