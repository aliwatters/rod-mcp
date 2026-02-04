# PR Refinement

---

🚨 **CRITICAL: EXECUTION REQUIRED - READ THIS FIRST** 🚨

**This is NOT documentation** - This prompt contains MANDATORY INSTRUCTIONS that you MUST EXECUTE NOW.

**BLOCKING PHASE**: You may NOT advance to the next phase until ALL steps in this prompt are completed.

**Your Responsibility as AI Agent**:

1. **EXECUTE** every step described below (do not just read them)
2. **VERIFY** each step completes successfully
3. **CHECK** all completion criteria at the end of this prompt
4. **ONLY THEN** allow phase advancement by running the explicit transition command

**If you skip steps or advance early**: The development cycle will fail and all work will be wasted.

**Continuous Mode**: Even though the orchestrator runs in continuous mode, YOU must ensure each phase completes its work before allowing advancement. The orchestrator displays prompts - YOU must execute them.

---

**IMPORTANT**: Please follow all instructions in this prompt carefully and execute each step
completely before proceeding.

## Architecture: Polling Pattern for Long-Running Operations

**Decision**: Refinement phase uses a **polling pattern** to handle long-running Copilot reviews without MCP timeouts.

**Pattern**: Agent-managed polling with short timeouts (replaces blocking `ax_refinement_enforce`)

**Why**: Copilot reviews can take 15+ minutes. Using a 20-minute MCP timeout (1200 seconds) allows sufficient time for Copilot to complete reviews without premature timeout.

**See**: Issue #4354 for implementation details.

## Purpose

Address review feedback and prepare PR for human review through systematic refinement, quality
checks, and automated verification using the compound enforcement tool.

**Non‑negotiable gating loop (do not skip)**:

- Wait: `ax pr wait-copilot <pr> --timeout 1200 --poll-interval 30` (or mark copilot-wait complete if no review appears after the timeout).
- Act: For every review comment pick exactly one path—fix (code change + commit), defer (create/link issue), or acknowledge (explicit reason). No silent ignores.
- Reply: Post threaded replies stating what you did (commit hash or issue link) and why it satisfies the comment; verify with `ax pr review-comments unacknowledged --pr <pr>` until zero remain.

## When to Use

- After PR creation in implementation phase
- When Copilot or human review feedback needs addressing
- To transition from "ready for robots" → "ready for humans"

## Workflow Overview

**⚠️ AUTONOMOUS EXECUTION REQUIRED**

Execute ALL operations below in ONE continuous response WITHOUT STOPPING.

**DO NOT**:

- Stop between operations
- Wait for user input
- Ask for confirmation
- Pause to summarize

This phase uses a **polling pattern** to handle Copilot reviews without MCP timeouts.

**Execution Flow** (execute sequentially without stopping):

1. Validate PR and branch → **THEN IMMEDIATELY**:
2. Wait for reviews (Copilot + Minion) → **THEN IMMEDIATELY**:
3. Address feedback with execution verification → **THEN IMMEDIATELY**:
4. Merge main, run checks, push → **THEN IMMEDIATELY**:
5. Wait for CI → **THEN IMMEDIATELY**:
6. Verify comments, run readiness gate → **THEN IMMEDIATELY**:
7. Mark PR ready → **Phase complete**

## MCP Tool Usage

**⚠️ IMPORTANT**: This workflow uses the ax-mcp MCP tools with polling pattern for long-running operations.

**Usage Pattern**:

- **MCP Tool**: `mcp__ax-mcp__ax_pr_wait_copilot` with `{"pr": PR_NUMBER, "timeout": 1200}`
- **Polling Pattern**: Agent calls tool with 20-minute timeout (1200s) to handle long Copilot reviews
- **CLI Fallback**: `ax pr wait-copilot $PR_NUMBER --timeout 1200 --poll-interval 30`

**Complete Guide**: See `~/git/dotfiles/docs/MCP_TOOL_USAGE_GUIDE.md`

---

## Execution Flow

**Validate PR and Branch** → Run validation command:

```bash
ax refinement validate-pr-and-branch --auto-fix
```

**Exit codes**:

- **0**: Validation passed → Continue to Step 1
- **1**: Manual fix required → Follow command guidance
- **2**: Skip refinement → Run `prompt retrospective`

### Validation Scenarios

**Scenario 1: Success (PR exists, on correct branch)**

```bash
✅ PR #1234 found for branch feature/issue-123
✅ On correct feature branch
   Proceeding to refinement workflow
```

→ Continue to Step 1

**Scenario 2: Auto-fixed (branch mismatch)**

```bash
❌ Branch mismatch detected!
   Current branch: main
   PR #1234 branch: feature/issue-123

📋 Auto-switching to correct branch...
✅ Switched to correct branch: feature/issue-123
```

→ Continue to Step 1

**Scenario 3: Issue closed (exit 2)**

```bash
❌ Issue #123 was closed as duplicate/not planned
📋 Transition Path: Skip refinement → retrospective
   Run: prompt retrospective
```

→ Skip to retrospective

**Scenario 4: Manual fix required (exit 1)**

```bash
❌ Issue #123 is still open but no PR exists
📋 Next steps:
   1. Return to implementation: prompt implementation
   2. Or abandon issue: ax issue abandon --issue 123
```

→ Follow guidance

→ **THEN IMMEDIATELY**:

**Wait for Reviews (Copilot + Minion)** → Get PR number and check reviews:

**Get PR number** (re-fetch to ensure availability):

```bash
# Re-fetch PR number (variables from Step 0 don't persist across tool invocations)
CURRENT_BRANCH=$(git branch --show-current)
PR_NUMBER=$(gh pr list --head "$CURRENT_BRANCH" number --jq '.[0].number // empty')
echo "Working on PR #$PR_NUMBER"
```

**Check Minion Review Status (Non-Blocking)**:

**⚠️ NEW**: Check for minion review feedback (triggered in Review phase).

Minion reviews run in **parallel** with Copilot reviews. Check if minion has posted feedback:

**MCP Tool (preferred)**:

```typescript
// Check PR for minion review comments
// Use PR number from workflow state
mcp__ax -
  mcp__ax_pr_review_comments_list({
    pr: 5362, // Replace with actual PR number from workflow state
  });
```

**Look for**: Comments from minion with "🤖 Minion Review" prefix or "minion" in author.

**If minion review found**:

- Note findings alongside Copilot feedback
- Prioritize: Copilot bugs > Minion architecture violations > Minion bugs > Copilot suggestions > Minion style

**If minion review NOT found**:

- Check if >10 minutes since PR creation → Assume minion skipped (credits, errors, timeout)
- Continue with Copilot-only review (non-blocking)

**⚠️ Non-Blocking**: Minion review absence does NOT block refinement.

→ **THEN IMMEDIATELY**:

**Wait for Copilot Review (Polling Pattern)**:

**Use polling pattern** to handle long Copilot review times without MCP timeout:

**Why polling?** Copilot reviews can take 15+ minutes. Instead of one long blocking call that times out MCP, we make many short calls (<2 min each) so MCP never times out.

**Poll for Copilot Review**:

Use the `ax pr wait-copilot` command with appropriate timeout and polling to wait for Copilot review.

**Instructions for AI Agent**:

1. Get the PR number for the current branch:

   ```bash
   PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')
   ```

2. Wait for Copilot review with 20-minute timeout (1200 seconds):

   ```bash
   ax pr wait-copilot $PR_NUMBER --timeout 1200 --poll-interval 30
   ```

3. Check exit code and **track timeout status for race condition mitigation (Issue #5613)**:
   - **Exit code 0**: Copilot review found → Set `copilot_wait_timed_out=false` → Continue to Step 1.1b
   - **Exit code 1**: Timeout (no review after 20 minutes) → Set `copilot_wait_timed_out=true` → Continue to Step 1.1b (grace period will catch late-arriving comments)
   - **Exit code 2**: Error during polling → Exit refinement phase with error

**⚠️ CRITICAL (Issue #5613 - Race Condition Fix)**: When Copilot wait times out, you MUST still check for comments by calling `ax_refine_analyze_feedback` with `copilot_wait_timed_out=true`. This flag:

- Triggers a 30-second grace period before checking comments (catches late-arriving Copilot comments)
- Enables the 10-second safety net re-check before marking PR ready
- Should be passed through to `ax_refine_apply_responses` and `ax_refine_complete` for full protection

→ **THEN IMMEDIATELY**:

**Address Feedback with Execution Verification**:

**⚠️ CRITICAL**: When addressing feedback, you MUST verify execution happens (not just analysis).

**Response Format Requirements for MCP Tool**:

When using `mcp__ax-mcp__ax_pr_review_comments_reply_with_verification`:

- The commit hash (7+ characters) MUST appear somewhere in either the `changes_made` or `verification` fields.
- Example: "tools/ax/internal/gitutil/gitutil.go:43 - Changed %s to %v" (no hash) + "Verified in 3392ad66" (hash in verification) is valid.
- The tool validates that the hash is present in one of these fields; it does NOT require a "Fixed in" prefix or for the hash to be at the start.
- The "Fixed in {commit}" prefix is automatically added by the tool when posting the response.

```bash
# Re-fetch PR number for this execution context
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')

# For EACH comment requiring a fix (boy scout rule <4 hours):
# 1. Use ultrathink to estimate effort (investigation + implementation + testing + docs)
# 2. Apply boy scout rule: <4 hours = FIX, ≥4 hours = DEFER
# 3. For each FIX decision:
#    - ACTUALLY edit the files (use Edit tool)
#    - RUN git diff to verify changes
#    - COMMIT the changes with descriptive message
#    - NOTE the commit hash for reply

# After addressing ALL comments classified as "fix":
# MANDATORY EXECUTION VERIFICATION (prevents analysis-only workflow failure)

# Count expected fixes from your analysis and verify execution
# Replace <YOUR_FIX_COUNT> with the actual number of fixes (e.g., 3)
# Example: If you classified 3 comments as "fix", use: --expected-fixes 3

# Use combined command with immediate exit code check (Issue #5112)
if ! ax pr verification verify-fixes-applied $PR_NUMBER --expected-fixes <YOUR_FIX_COUNT>; then
    echo "❌ EXECUTION GAP: Analysis-execution gap detected"
    echo "⚠️ Go back and ACTUALLY implement the missing fixes."
    exit 1
fi

echo "✅ Execution verified: Commits match expected fixes"

# Verify all review comments are addressed with responses
ax pr verification verify-feedback --pr $PR_NUMBER
```

**If polling loop times out** (no review after 20 minutes):

- Issue warning but continue with refinement
- Some PRs may not get Copilot reviews (too small, wrong labels, etc.)
- Human reviewers will catch any issues

**If verification fails**:

- Review comments exist but aren't addressed
- Fix the comments and re-run verification
- DO NOT proceed until all comments are addressed

→ **THEN IMMEDIATELY**:

**Merge Main, Run Checks, and Push** (REQUIRED):

Use the compound MCP tool to merge main, run quality gates, and push in a single atomic operation:

**MCP Tool (preferred)**:

```typescript
// Single tool replaces 5-7 manual bash commands
mcp__ax -
  mcp__ax_refinement_merge_and_validate({
    pr: 1234, // Required: PR number
    auto_push: true, // Default: true - push after successful validation
    skip_quality_gates: false, // Default: false - run ax checkin
  });
```

**Response Handling**:

- **Success**: Returns `merge_success: true, tests_passed: true, push_success: true`
- **Conflicts**: Returns `merge_conflicts: [files]` with structured info, aborts merge cleanly
- **Quality gate failure**: Returns `quality_gates_error` with details

**CLI Fallback** (only if MCP unavailable):

```bash
# Manual fallback - prefer MCP tool above
git fetch origin main
git merge origin/main --no-edit
ax checkin
git push origin $(git branch --show-current)
```

**🧠 ULTRATHINK: Conflict Resolution** (if `merge_conflicts` detected):

When the merge tool returns conflicts, you MUST use extended thinking to resolve them properly:

```bash
# Check merge status before proceeding
ax pr check-mergeable $PR_NUMBER

# If conflicts exist (exit code 1), resolve with ultrathink analysis
```

**Conflict Resolution Process**:

1. **Analyze each conflict** using extended thinking:

   ultrathink: For each conflicting file, analyze:
   - What changed in our branch (the feature we're adding)?
   - What changed in main (updates from other PRs)?
   - Should we keep ours, theirs, or merge both?
   - What is the intent of each change?

   Consider:
   - Keep ours: Our changes are the feature being added
   - Keep theirs: Main has a better/more complete implementation
   - Merge both: Both changes are needed and don't conflict semantically

2. **Document your reasoning** for each conflict resolution decision

3. **Resolve conflicts** manually or with strategy:

   ```bash
   # After ultrathink analysis, resolve using appropriate strategy
   ax pr resolve-conflicts $PR_NUMBER --strategy ours   # Keep our changes
   ax pr resolve-conflicts $PR_NUMBER --strategy theirs # Keep main's changes
   # Or resolve manually for complex merges
   ```

4. **Verify resolution** - Build and test after resolving:

   ```bash
   ax checkin  # Run quality gates after conflict resolution
   ```

**Example from real PR** (PR #5492):

- **labels.go**: Kept our StandardPRUsage changes (the feature being added)
- **verification.go**: Accepted main's NewResult pattern (more complete implementation from a parallel PR)

This required understanding that main's version superseded our fix with a better implementation.

→ **THEN IMMEDIATELY**:

**Wait for CI** (up to 15 minutes):

```bash
# Re-fetch PR number for this execution context
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')

# Wait for CI checks with 15-minute timeout (built-in polling)
ax pr check-ci $PR_NUMBER --timeout 900 --poll-interval 30

# Exit code 0 = passing, 1 = failing/timeout, 2 = error
if [ $? -eq 1 ]; then
    echo "⚠️ CI checks failed or timed out after 15 minutes"
    echo "Review CI failures and fix issues before proceeding"
    exit 1
elif [ $? -eq 2 ]; then
    echo "❌ Error checking CI status"
    exit 1
fi
```

→ **THEN IMMEDIATELY**:

**Verify Comments Acknowledged** (REQUIRED):

```bash
# Re-fetch PR number for this execution context
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')

# Check all review comments have been acknowledged
ax pr verification check-comments --pr $PR_NUMBER

if [ $? -ne 0 ]; then
    echo "❌ Not all comments acknowledged. Reply to all comments and re-run."
    exit 1
fi
```

→ **THEN IMMEDIATELY**:

**Validate Response Quality** (REQUIRED):

```bash
# Re-fetch PR number for this execution context
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" --json number --jq '.[0].number')

# Validate response quality (Issue #5086)
# - "Fixed" responses must include commit hash (7+ hex chars)
# - "Acknowledged" should not be used for fix suggestions
ax pr review-comments validate-responses --pr $PR_NUMBER

if [ $? -ne 0 ]; then
    echo "❌ Response quality validation failed."
    echo "   Fix responses to include proper commit references."
    echo "   Use 'Fixed in <commit_hash>' format for code changes."
    exit 1
fi
```

→ **THEN IMMEDIATELY**:

**Run Readiness Gate** (MUST PASS):

```bash
# Re-fetch PR number for this execution context
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')

# Final validation before marking ready
ax pr verification verify-ready --pr $PR_NUMBER

if [ $? -ne 0 ]; then
    echo "❌ PR not ready for human review. Fix issues and re-run."
    exit 1
fi
```

→ **THEN IMMEDIATELY**:

**Mark PR Ready with Complete Labels** (transition to "ready for humans"):

```bash
# Re-fetch PR number and issue number
CURRENT_BRANCH=$(git branch --show-current)
PR_NUMBER=$(gh pr list --head "$CURRENT_BRANCH" number --jq '.[0].number')
ISSUE_NUMBER=$(echo "$CURRENT_BRANCH" | grep -oE '[0-9]+')

# Use ax commands for label management (see PATTERN_LIBRARY.md Pattern 12)
# This avoids complex shell variable expansion issues in Claude Code

# Get issue labels and sync to PR (excluding "claimed")
# Use simple two-step approach to avoid piping complexity
ISSUE_LABELS=$(ax issue get-labels $ISSUE_NUMBER --exclude "claimed" --format comma-separated)
ax pr labels manage $PR_NUMBER --add "$ISSUE_LABELS"

# Add workflow label "ready for humans", remove "ready for robots"
ax pr labels manage $PR_NUMBER --add "ready for humans" --remove "ready for robots"

# Mark PR ready (remove draft status)
gh pr ready $PR_NUMBER

# Verify final state
echo "✅ PR #$PR_NUMBER is now ready for human review"
echo "Final state:"
gh pr view $PR_NUMBER isDraft,labels --jq '{isDraft, labels: [.labels[].name]}'
```

**Expected final state:**

- `isDraft`: false
- Labels: [all issue labels except "claimed"] + "ready for humans"
- NOT present: "ready for robots"

→ **Phase operations complete** → Continue to Phase Completion Checkpoint below.

---

## Manual Fallback (ONLY IF COMPOUND TOOLS BROKEN)

**Reference**: `~/git/dotfiles/docs/REFINEMENT_MANUAL_FALLBACK.md`

If compound tools (`ax_refinement_enforce`, `ax_refine_*`) are unavailable, follow the manual fallback guide which covers:

- Comment classification and response type determination
- Boy Scout rule application with effort estimation (< 4 hours = fix, >= 4 hours = defer)
- Fix verification and execution checkpoints
- Threaded response posting with verification evidence

**Quick Reference - Boy Scout Decision Matrix**:

| Estimate   | Decision    | Action                             |
| ---------- | ----------- | ---------------------------------- |
| < 4 hours  | FIX         | Implement immediately              |
| >= 4 hours | DEFER       | Create issue with effort breakdown |
| N/A        | ACKNOWLEDGE | No action needed                   |

## Verify Final State

**Check PR state**:

```bash
gh pr view $PR_NUMBER isDraft,labels --jq '.isDraft, .labels[].name'
```

**Expected**:

- `isDraft`: false
- Labels: "ready for humans" (NOT "ready for robots")

**Check workflow state**:

```bash
ax workflow get | grep PHASE
```

**Expected**: `PHASE=retrospective`

## Integration with Dev Cycle

After refinement completes, **you MUST continue to retrospective phase**:

```bash
# Automatic phase transition (done by ax_refinement_enforce)

# Just execute retrospective prompt

prompt retrospective
```

**⚠️ STOPPING BEFORE RETROSPECTIVE = WORKFLOW VIOLATION**

## Compound Tools for Fine-Grained Feedback Handling

**✅ IMPLEMENTED**: Compound tools are now available for manual refinement scenarios when the
orchestrator isn't suitable or fails.

**Analyze Feedback** → MCP Tool (preferred - handles race condition mitigation):

```typescript
// If Copilot wait timed out, set copilot_wait_timed_out=true
// This triggers a 30-second grace period to catch late-arriving comments (Issue #5613)
mcp__ax -
  mcp__ax_refine_analyze_feedback({
    pr: 1234,
    copilot_wait_timed_out: true, // Set to true if ax_pr_wait_copilot returned exit code 1
  });
```

**CLI Fallback**:

```bash
# Gather and analyze all PR feedback with AI classification
ax refine analyze-feedback <pr-number>
```

**This command**:

- Gathers all PR comments, review comments, and reviews
- Identifies Copilot feedback specifically
- Classifies comments by type (potential_bug, refactoring, scope_creep, etc.)
- Suggests actions (fix, defer, acknowledge) with AI reasoning
- Detects scope creep risks and provides intelligence hints
- Outputs structured JSON for consumption by apply-responses
- **When `copilot_wait_timed_out=true`**: Waits 30 seconds and re-checks for late-arriving comments (Issue #5613)

→ **THEN IMMEDIATELY**:

**🔢 MANDATORY: Comment Enumeration (Issue #5953)** → Explicitly enumerate ALL comments:

**⚠️ CRITICAL**: Before processing ANY comment, you MUST enumerate ALL comments to ensure none are skipped.

1. **Create todo list with ALL comments** → Use TodoWrite to track each comment:

```typescript
// After ax_refine_analyze_feedback returns, extract ALL comment IDs and create tracking todos
// Example: If analyze_feedback returned 5 comments with IDs 12345, 12346, 12347, 12348, 12349
TodoWrite({
  todos: [
    { content: "Comment #12345: [first 50 chars of body]", status: "pending", activeForm: "Addressing comment #12345" },
    { content: "Comment #12346: [first 50 chars of body]", status: "pending", activeForm: "Addressing comment #12346" },
    { content: "Comment #12347: [first 50 chars of body]", status: "pending", activeForm: "Addressing comment #12347" },
    { content: "Comment #12348: [first 50 chars of body]", status: "pending", activeForm: "Addressing comment #12348" },
    { content: "Comment #12349: [first 50 chars of body]", status: "pending", activeForm: "Addressing comment #12349" },
  ],
});
```

2. **Log the enumeration** → Output a clear list:

```text
📋 REVIEW COMMENTS TO ADDRESS (N total):
1. Comment #12345: "Consider adding validation..."
2. Comment #12346: "This could be simplified..."
3. Comment #12347: "Missing error handling..."
...
```

3. **Verify count matches** → Confirm the todo count matches `review_comments.total` from analyze_feedback output.

**WHY THIS MATTERS**: Without explicit enumeration, comments can be silently skipped. The todo list ensures:

- Every comment is tracked
- Progress is visible
- No comment is forgotten
- Final verification is possible

→ **THEN IMMEDIATELY**:

**AI Decision Making (Extended Thinking)** → Use the analysis output to make decisions:

- Review AI classifications and suggestions
- Apply extended thinking to complex feedback
- Decide on final actions for each comment
- **Mark each todo as `in_progress` when starting, `completed` when done**

→ **THEN IMMEDIATELY**:

**Apply Responses** → Execute decisions for each comment:

```bash
# Execute decisions for each comment atomically

ax refine apply-responses <pr-number> \
 --comment 12345 --action fix --changes "Added validation" --commit abc123 \
 --comment 12346 --action defer --issue 3900 \
 --comment 12347 --action acknowledge
```

**This command**:

- Applies code changes based on decisions
- Creates issues for deferred work
- Posts replies to comments with appropriate templates
- Runs BLOCKING quality gates (`ax checkin`) if changes made
- Handles all actions atomically

### When to Use Primary Compound Tool vs Fine-Grained Tools

**Use `ax_refinement_enforce` compound tool (preferred)**:

- Full automation desired
- Standard refinement workflow
- No special handling needed
- Single atomic operation for all 9 steps

**Use fine-grained compound tools (fallback)**:

- Primary compound tool failed or incomplete
- Need fine-grained control over responses
- Complex scope creep situations requiring manual judgment
- Want to see AI analysis before executing actions

### Intelligence Features

**Scope Creep Detection**: Automatically identifies comments suggesting work beyond PR scope

**Priority Classification**: High/medium/low priority based on comment content and type

**Action Suggestions**: AI reasoning for fix/defer/acknowledge decisions

**Quick Fix Identification**: Spots style and documentation changes for batch processing

## Comment Response Patterns

### Fix Response Pattern

```text
✅ Fixed in <commit_hash>

<Brief description of what was changed and why>

Example:
✅ Fixed in abc123

Added null check for config parameter to prevent NPE. Also added test case to verify behavior when config is null.
```

### Defer Response Pattern

```text
✅ Created #<issue_number> to track this enhancement

This is a valid suggestion but represents scope creep beyond the original issue (#<original_issue>). I've created a separate issue to ensure it gets proper attention.

Example:
✅ Created #3900 to track this enhancement

This is a valid suggestion but represents scope creep beyond the original issue (#3855). Extracting retry logic to a shared utility is a good idea, but it's better suited for a separate PR to ensure proper testing and integration across all consumers.
```

### Acknowledge Response Pattern

```text
✅ Acknowledged

<Brief response>

Examples:
✅ Acknowledged

Thanks for the positive feedback!

✅ Acknowledged

This is already handled by the error handler in lines 45-52.
```

## Scope Creep Detection Guidelines

**Questions to ask** (use `extended thinking`):

1. Does this suggestion require changes outside the files touched by original issue?
2. Does this suggestion add functionality not mentioned in original issue?
3. Does this suggestion require architectural changes?
4. Would this suggestion benefit from its own issue for proper scoping?
5. Is this suggestion extracting/generalizing code that only has one use case currently?

**If 2+ answers are YES → Scope creep → Action: DEFER**

## Extended Thinking Guidance

**Use `extended thinking` for**:

- Review feedback analysis (understanding intent)
- Scope creep detection (boundary analysis)
- Fix vs defer decisions (trade-off analysis)
- Issue description drafting (for deferrals)

**Questions for extended thinking**:

- What is the reviewer really asking for?
- Is this in scope of the original issue?
- Would fixing this introduce complexity or risk?
- Is this the right place for this change?
- Should this be its own PR/issue?

---

## 🤔 ULTRATHINK: Pre-Completion Label Verification

**MANDATORY: Use extended thinking to verify COMPLETE label state before marking phase complete.**

**Step 1: Identify all required labels**

Think through what labels should be present:

- **Issue labels**: All labels from original issue (except "claimed")
- **Workflow labels**: "ready for humans" (added during refinement)
- **Labels to remove**: "ready for robots" (if present)

**Step 2: Get current state**

```bash
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')
ISSUE_NUMBER=$(echo "$(git branch --show-current)" | grep -oE '[0-9]+')

# Current PR labels
echo "Current PR labels:"
gh pr view $PR_NUMBER labels --jq '.labels[].name'

# Required issue labels
echo "Required issue labels:"
gh issue view $ISSUE_NUMBER labels --jq '.labels[].name' | grep -v "claimed"
```

**Step 3: Analyze gaps**

Use extended thinking to identify:

- **Missing labels**: What's required but not present?
- **Extra labels**: What's present but shouldn't be?
- **Expected final state**: List exactly what labels should exist

**Step 4: Fix labels if needed**

```bash
# Add missing labels (if any identified in Step 3)
# Use ax commands to avoid shell variable expansion issues (see PATTERN_LIBRARY.md Pattern 12)
ax pr labels manage $PR_NUMBER --add "<missing-labels-comma-separated>"

# Remove incorrect labels (if any identified in Step 3)
ax pr labels manage $PR_NUMBER --remove "<extra-labels-comma-separated>"

# Verify final state
echo "✅ Final PR labels:"
gh pr view $PR_NUMBER labels --jq '.labels[].name'
```

**Only after label verification shows correct state, proceed to Phase Completion checkpoint below.**

---

## 🔒 PHASE COMPLETION - BLOCKING CHECKPOINT

**STOP**: Before proceeding, verify ALL of the following are complete:

### Refinement Phase Completion Criteria

**CRITICAL**: Use extended thinking to verify EACH item below:

- [ ] **Comment coverage verified** (Issue #5953): All comments from enumeration todo list are marked `completed`
- [ ] All automated review feedback addressed (Copilot, minion)
- [ ] All review comments acknowledged or resolved
- [ ] CI checks passing
- [ ] PR marked as "ready for review" (no longer draft)
- [ ] PR has ALL issue labels (from original issue, minus "claimed")
- [ ] PR has "ready for humans" workflow label
- [ ] PR does NOT have "ready for robots" label
- [ ] All quality gates passed
- [ ] No unacknowledged comments remaining
- [ ] No blocking errors

**🔢 Comment Coverage Gate (Issue #5953)**:

Before calling `ax_refine_complete`, verify:

1. Review your todo list - every comment todo should be `completed`
2. Count: Number of completed comment todos = `review_comments.total` from analyze_feedback
3. If any comment todos are still `pending` or `in_progress`, address them first

**Verification commands:**

```bash
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" number --jq '.[0].number')

# Verify labels
gh pr view $PR_NUMBER labels,isDraft --jq '{isDraft, labels: [.labels[].name]}'

# Verify comments acknowledged
ax pr review-comments unacknowledged --pr $PR_NUMBER

# Verify ready state
ax pr verification verify-ready --pr $PR_NUMBER
```

### Phase Advancement

⚠️ **CRITICAL AUTOMATIC TRANSITION** ⚠️

**DO NOT STOP TO SUMMARIZE** - The transition to retrospective is **IMMEDIATE and AUTOMATIC**.

**Key Points**:

- Human review and retrospective happen **IN PARALLEL**
- Human reviews the PR at their own pace (asynchronous)
- Retrospective analyzes the development cycle **immediately** (synchronous)
- Retrospective does **NOT** wait for PR approval/merge

**ONLY AFTER ALL ABOVE ARE ✅ COMPLETE**:

```bash
# Signal phase completion to orchestrator (Issue #4957)
ax workflow update --field "PHASE_STATUS=completed"

# Invoke orchestrator to validate and advance to next phase
# IMMEDIATELY continue (NO pause, NO summary)
ax cycle run dev-cycle
```

**CRITICAL ENFORCEMENT**:

- Agent sets `PHASE_STATUS=completed` to signal phase is done
- Agent calls `ax cycle run dev-cycle` (or MCP equivalent) to advance
- Orchestrator validates completion, advances `PHASE` to retrospective automatically
- NO manual `ax workflow update --phase` commands needed

**This is orchestrator-driven flow.** The cycle orchestrator handles all phase transitions
automatically when `PHASE_STATUS=completed` is set.

**VERIFICATION**: Run `ax workflow get` and confirm `PHASE=retrospective` after orchestrator advances.

---

## Related Documentation

- **Complete Workflow**: `~/git/dotfiles/workflows/PR_REFINEMENT_WORKFLOW.md` (detailed steps)
- **Review Process**: `~/git/dotfiles/workflows/PR_REVIEW_PROCESS.md` (handling feedback)
- **Cleanliness**: `~/git/dotfiles/workflows/PR_CLEANLINESS_WORKFLOW.md` (quality assessment)
- **Architecture**: `~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md`
- **Agent Boundaries**: `~/git/dotfiles/docs/AGENT_BOUNDARIES.md`
- **Specification**: `~/git/dotfiles/docs/future-work/DEV_CYCLE_MCP_REFACTOR_SPEC.md`
