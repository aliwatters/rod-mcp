# Phase 1: Discovery (Dev-Cycle Workflow)

**📋 Specification**: This prompt implements
`specifications/phase-1-discovery-specification.feature`

**📖 Complete Workflow**: See `workflows/DISCOVERY_WORKFLOW.md` for detailed step-by-step
procedures.

**⚠️ MANDATORY COMPLIANCE**: You MUST follow ALL instructions in this prompt. Execute every step
completely before proceeding.

**🤖 FULL AUTOMATION MODE**: This workflow runs completely automated. NO user confirmation required
for normal operations. Only stop/alert for ERROR conditions.

**⚠️ WORKSPACE ISOLATION**: Stay within starting workspace. NEVER cd to sibling directories (e.g.,
`../dotfiles-agent-2`).

**Purpose**: Discover and claim work, validate issue appropriateness, and transition to
implementation with validated issue.

---

## Quick Start

Execute the complete discovery workflow by following all steps in `workflows/DISCOVERY_WORKFLOW.md`:

1. **Workspace Validation** (MANDATORY FIRST STEP)
2. **Issue Discovery** (with pre-instruction support)
3. **Issue Claiming** (atomic operation)
4. **Issue Validation** (comprehensive checks)
5. **Complexity Assessment**
6. **Architecture Assessment Gate** (MANDATORY - prevents tactical fixes)
7. **Goal/Architecture Alignment**
8. **Branch Creation**
9. **Workflow State Update**
10. **Transition to Implementation**

---

## Pre-Instruction Support

**Check for pre-instruction**: If user provided "work on issue #XYZ", use that issue number directly
and skip issue selection. Do NOT skip validation—pre-instructed issues must still be validated
through all checks.

---

## Key Commands

### Workspace Validation (Step 1 - MANDATORY)

```bash
pwd
# Workspace validation (2-minute timeout) - regex validation only
ax workflow validate-workspace
```

**Expected**: ✅ Workspace validation passed

**Exit Codes**:

- **Code 0**: Validation passed → Proceed to discovery
- **Code 1**: Validation failed (BLOCKING) → STOP

**Blocking Errors (Code 1)**:

- ❌ **Main workspace in multi-agent repo**: Use agent workspace like `dotfiles-agent-N` or
  `dotfiles-claude-N`
- ❌ **Agent workspace in single-agent repo**: Work in main workspace instead
- ❌ **Cross-workspace contamination detected**: Fix hardcoded paths or symlinks
- ❌ **Invalid workspace name**: Contains invalid characters (@ or special chars)
- ❌ **Project name mismatch**: Workspace doesn't match parent repository

**Allowed References**:

- ✅ References to main dotfiles (`/Users/ali/git/dotfiles/`) are allowed in all directories
- ✅ Documentation examples are excluded from contamination checks

**If validation fails**: STOP all work. Do not proceed to discovery.

---

## Step 0: Pre-Instruction Check (ABSOLUTE PRIORITY)

**⚠️ CRITICAL**: Pre-instructions from the user have ABSOLUTE PRIORITY over all automated discovery
logic. If a user explicitly says "work on issue #XYZ", that instruction must be honored immediately,
skipping PR prioritization and handoff checks.

**How to detect pre-instructions**: Check the user's original request in this conversation. Look for
patterns like:

- "work on issue #123"
- "work on issue 123"
- "start with issue #123"
- Any explicit reference to a specific issue number in the initial request

**If pre-instruction detected**:

1. Output: `✅ Pre-instruction detected: user explicitly requested issue #<NUMBER>`
2. Output: `Skipping PR prioritization and handoff checks to honor user preference`
3. **SKIP Steps 1-2** (PR prioritization and handoffs)
4. **Jump directly to Step 3** (Find Unclaimed Issue), using `ax issue next --issue <NUMBER>`
5. Proceed with normal claiming and validation

**If NO pre-instruction detected**: Continue to Step 1 for normal automated prioritization.

**Priority hierarchy** (when no pre-instruction):

1. **Incomplete PRs** (existing work from current user) - HIGH PRIORITY
2. **Handoffs** (PRs handed off from pr-ready-cycle) - HIGH PRIORITY
3. **New issue discovery** (weighted random selection) - NORMAL PRIORITY

**Why pre-instructions take absolute priority**:

- User autonomy must be respected above all automated logic
- Explicit instructions override all automated prioritization (PRs, handoffs, discovery)
- Enables urgent fixes and specific work requests
- Without this, users cannot override automation when needed

---

## Scenario Group 1a.5: PR Prioritization (HIGH PRIORITY - Step 1)

**⚠️ CRITICAL**: Discovery phase MUST check for incomplete PRs BEFORE discovering new issues (but
AFTER checking for pre-instructions). Completing existing work is higher priority than starting new
work.

### Step 1: Check for Incomplete PRs

**⚠️ NOTE**: This step is SKIPPED if a pre-instruction was detected in Step 0. User instructions
have absolute priority.

```bash
# Check for incomplete PRs needing work (by current user)
# (Skip this step if user provided explicit pre-instruction in Step 0)
INCOMPLETE_PR=$(ax pr discover-incomplete 2>/dev/null || echo "")

if [ -n "$INCOMPLETE_PR" ]; then
  echo "✅ Found incomplete PR #$INCOMPLETE_PR - prioritizing over new issues"

  # Get PR details
  PR_DETAILS=$(gh pr view $INCOMPLETE_PR --json number,title,headRefName,body --jq '{number, title, branch: .headRefName}')
  PR_BRANCH=$(echo "$PR_DETAILS" | jq -r '.branch')
  PR_TITLE=$(echo "$PR_DETAILS" | jq -r '.title')
  PR_ISSUE=$(gh pr view $INCOMPLETE_PR --json number,body --jq '.body' | grep -oE '#[0-9]+' | head -1 | tr -d '#')

  echo "  PR: #$INCOMPLETE_PR"
  echo "  Title: $PR_TITLE"
  echo "  Branch: $PR_BRANCH"
  echo "  Linked Issue: #$PR_ISSUE"

  # Phase 2: Atomic PR claim (prevents race conditions)
  # Check if already claimed by another workspace before claiming
  WORKSPACE_ID=$(ax workflow get | grep "WORKSPACE=" | cut -d= -f2 || echo "$(basename $(pwd))")
  # Search for claim comments - pattern matches both "🤖 **Claimed by" and "🤖 **Claimed by AI Agent"
  EXISTING_CLAIM=$(gh pr view $INCOMPLETE_PR --json comments --jq '.comments[] | select(.body | contains("🤖 **Claimed by") and (.body | contains("**Workspace**:"))) | .body' | head -1)

  if [ -n "$EXISTING_CLAIM" ]; then
    # Extract claiming workspace from comment
    # Extract workspace using grep -E (macOS compatible) instead of grep -P
    CLAIMING_WORKSPACE=$(echo "$EXISTING_CLAIM" | grep -oE '\*\*Workspace\*\*:\s*`[^`]+`' | sed 's/.*`\([^`]*\)`/\1/' || echo "")
    if [ -n "$CLAIMING_WORKSPACE" ] && [ "$CLAIMING_WORKSPACE" != "$WORKSPACE_ID" ]; then
      echo "❌ PR #$INCOMPLETE_PR already claimed by $CLAIMING_WORKSPACE"
      echo "⚠️  Skipping to avoid race condition"

      # Log friction: Race condition during PR claim
      ax friction append "$(cat <<EOF
## $(date '+%Y-%m-%d %H:%M') - Race Condition

**Issue**: PR #$INCOMPLETE_PR already claimed by $CLAIMING_WORKSPACE
**Category**: Race condition
**Resolution**: Skipped to avoid conflict, continuing to issue discovery
**Time Lost**: ~1 minute
EOF
)"

      INCOMPLETE_PR=""
      # Continue to issue discovery
    fi
  fi

  # Atomically claim PR if not already claimed
  if [ -n "$INCOMPLETE_PR" ]; then
    CLAIM_COMMENT="$(cat <<EOF
🤖 **Claimed by AI Agent**

This PR has been claimed for automated processing.

- **Workspace**: \`$WORKSPACE_ID\`
- **Workflow**: dev-cycle
- **Claimed at**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
- **Auto-release**: On workflow completion or error
EOF
)"

    # Add claim comment (atomic operation)
    if ax pr comment $INCOMPLETE_PR --body "$CLAIM_COMMENT"; then
      echo "✅ Claimed PR #$INCOMPLETE_PR for $WORKSPACE_ID"
    else
      echo "⚠️  Failed to claim PR #$INCOMPLETE_PR - may have been claimed by another agent"
      echo "⚠️  Falling back to new issue discovery"
      INCOMPLETE_PR=""
    fi
  fi

  # Fetch remote branch if it doesn't exist locally (handles multi-workspace scenarios)
  if ! git rev-parse --verify "$PR_BRANCH" >/dev/null 2>&1; then
    echo "  Branch '$PR_BRANCH' not found locally - fetching from remote"
    if ! git fetch origin "$PR_BRANCH:$PR_BRANCH"; then
      # Fetch failed - branch doesn't exist on remote or other error
      echo "❌ Failed to fetch branch '$PR_BRANCH' from remote"

      # Log friction: Branch fetch failure
      ax friction append "$(cat <<EOF
## $(date '+%Y-%m-%d %H:%M') - Branch Fetch Failure

**Issue**: Failed to fetch branch '$PR_BRANCH' for PR #$INCOMPLETE_PR
**Command**: \`git fetch origin \"$PR_BRANCH:$PR_BRANCH\"\`
**Category**: Branch operation failure
**Resolution**: Added needs-attention label, fallback to new issue discovery
**Time Lost**: ~2 minutes
EOF
)"

      gh pr edit "$INCOMPLETE_PR" --add-label "needs-attention"
      ax pr comment "$INCOMPLETE_PR" --body "⚠️ Failed to fetch branch during discovery. Branch may not exist on remote or has issues. Manual intervention needed."
      echo "⚠️  Falling back to new issue discovery"
      # Continue to issue discovery below
      INCOMPLETE_PR=""
    fi
  fi

  # Checkout PR branch (if fetch succeeded or branch already exists locally)
  if [ -n "$INCOMPLETE_PR" ]; then
    if git checkout "$PR_BRANCH"; then
    # Update workflow state with PR and refinement phase
    ax workflow update --pr "$INCOMPLETE_PR" --phase refinement

    if [ -n "$PR_ISSUE" ]; then
      ax workflow update --issue "$PR_ISSUE"
    fi

    echo "✅ Checked out PR #$INCOMPLETE_PR - transitioning to refinement phase"

    # Skip to refinement phase
    prompt refinement
    exit 0
  else
    # Checkout failed - add needs-attention label and fall back to issue discovery
    echo "❌ Failed to checkout PR #$INCOMPLETE_PR branch"

    # Log friction: Branch checkout failure
    ax friction append "$(cat <<EOF
## $(date '+%Y-%m-%d %H:%M') - Branch Checkout Failure

**Issue**: Failed to checkout branch '$PR_BRANCH' for PR #$INCOMPLETE_PR
**Command**: \`git checkout \"$PR_BRANCH\"\`
**Category**: Branch operation failure
**Resolution**: Added needs-attention label, fallback to new issue discovery
**Time Lost**: ~2 minutes
EOF
)"

    gh pr edit $INCOMPLETE_PR --add-label "needs-attention"
    ax pr comment $INCOMPLETE_PR --body "⚠️ Automated checkout failed during discovery phase. Branch may have conflicts or issues. Manual intervention needed."

    echo "⚠️  Falling back to new issue discovery"
    # Continue to Step 3 (issue discovery)
    fi
  fi
else
  echo "ℹ️  No incomplete PRs found - proceeding to new issue discovery"
  # Continue to Step 3 (issue discovery)
fi
```

**Priority logic in `ax pr discover-incomplete`**:

1. PRs with "needs-attention" label (highest priority)
2. Draft PRs with failing CI
3. Draft PRs with unacknowledged comments
4. Draft PRs older than 24 hours
5. Ready PRs with failing CI (edge case)

**Filters**:

- Author: current user only (prevents cross-user interference)
- State: OPEN only (excludes merged/closed)
- Excludes: `minion-task` labels (handled by minion orchestration)
- Excludes: `claude-web-task` labels (handled by claude.ai/code web interface)

**When incomplete PR found**:

- Checkout PR branch
- Set workflow phase to "refinement" (skip implementation)
- Transition directly to refinement phase
- Discovery phase exits early

**When no incomplete PRs**:

- Continue to Step 3 (issue discovery)
- Normal flow: discovery → implementation

---

## Scenario Group 1b: Issue Discovery (No PRs)

### Step 2: Check for Handed-Off PRs (HIGH PRIORITY)

**⚠️ CRITICAL**: Check for handoffs BEFORE normal issue discovery (but AFTER pre-instruction check
in Step 0). Handoffs are existing work that needs resumption and have high priority.

**⚠️ NOTE**: This step is SKIPPED if a pre-instruction was detected in Step 0. User instructions
have absolute priority.

```bash
# Check for PRs handed off from pr-ready-cycle
# (Skip this step if user provided explicit pre-instruction in Step 0)
HANDOFFS=$(ax pr handoff list-handoffs --limit 1 2>/dev/null)

# If handoffs exist, claim the first one
if [ -n "$HANDOFFS" ]; then
    # Extract PR number from first handoff (format: "PR #1234: title")
    # Use sed for more robust extraction than nested greps
    PR_NUMBER=$(echo "$HANDOFFS" | sed -n 's/.*PR #\([0-9]\+\).*/\1/p' | head -1)

    if [ -n "$PR_NUMBER" ] && [ "$PR_NUMBER" -gt 0 ] 2>/dev/null; then
        # Atomically claim the handoff
        if ax pr handoff claim-handoff --pr "$PR_NUMBER"; then
            # Continue to appropriate phase (implementation or refinement)
            # Note: claim-handoff sets PHASE automatically based on RESUME_PHASE
            ax workflow get

            # Skip remaining discovery steps - resume from handoff phase
            prompt implementation
            exit 0
        fi
    fi
fi
```

**What happens on handoff claim**:

- Issue is atomically claimed (prevents race conditions)
- Workflow state initialized from handoff metadata
- `PHASE` set to `RESUME_PHASE` (implementation or refinement)
- `needs-attention` label removed from PR
- Workflow continues at handoff phase

**If no handoffs found**: Continue to normal issue discovery below.

---

### Step 3: Find Unclaimed Issue with No PR

**Primary Approach: Compound Tools** (Default for all workflows)

**Use these compound MCP tools by default** to streamline discovery with AI decision delegation.
This approach reduces execution from 15+ individual commands to 2 compound tool calls:

#### Tool 1: `ax_discover_analyze_issue` (Analysis)

**Purpose**: Deep issue analysis with duplicate detection and intelligence hints

```bash
# For pre-instructed issues (from Step 0)
result = ax_discover_analyze_issue(
  issue_number: 3850  # From user's "work on issue #3850"
)

# For normal discovery (weighted random)
result = ax_discover_analyze_issue(
  use_next: true  # Calls ax issue next internally
)
```

**What this returns**:

- Issue details (number, title, body, labels)
- Duplicate analysis with similarity scores
- Actionability validation
- Intelligence hints for AI reasoning

**AI Decision Delegation** (Use `ultrathink`):

After receiving analysis results, AI must decide:

1. **Duplicate Detection**: Is this a duplicate of any candidate?
   - Analyze `result.duplicate_analysis.candidates`
   - Compare scope, subsumes relationships
   - Use `ultrathink` for deep reasoning

2. **Scope Validation**: Does this fit agent-first paradigm?
   - Check alignment with CLAUDE.md principles
   - Verify within project goals
   - Use `ultrathink` for architecture analysis

3. **Actionability**: Should we work on this issue?
   - Review `result.actionability` status
   - Verify all requirements clear

**Example AI Decisions**:

```bash
# AI analyzes duplicate candidates using ultrathink
duplicate_decision = {
  is_duplicate: false,
  reasoning: "Unique scope: retry logic vs timeout config (#3201)"
}

# AI validates scope alignment using ultrathink
scope_decision = {
  in_scope: true,
  reasoning: "Agent-first: improves automation reliability"
}
```

#### Tool 2: `ax_discover_claim_and_setup` (Execution)

**Purpose**: Atomic claim + branch + state setup with AI decisions

```bash
# Execute with AI's decisions
ax_discover_claim_and_setup(
  issue_number: 3850,
  branch_prefix: "fix",  # or "feature", "docs", "refactor", "test"
  duplicate_decision: duplicate_decision,
  scope_decision: scope_decision
)
```

**What this does** (atomic operation):

1. Session logging with AI reasoning
2. Atomic issue claim (prevents race conditions)
3. Branch creation (feature/issue-N-title)
4. Branch verification
5. Workflow state initialization
6. Label management (adds "claimed", "in-progress")

**On success**: Transitions directly to implementation phase

**On failure**: Releases claim, returns to issue discovery

---

**⚠️ FALLBACK: CLI Approach** (Only when MCP tools unavailable)

**Use compound tools above by default.** Only use this CLI fallback if MCP tools are not available
in your environment.

**For pre-instructed issues** (detected in Step 0): Use `ax issue next --issue <NUMBER>` to get the
specific issue requested by the user.

**For normal discovery** (no pre-instruction): Use `ax issue next` for weighted random selection.

**⚠️ CRITICAL**: Claiming (Step 4) is MANDATORY for ALL issues including pre-instructed ones—this
prevents race conditions and ensures workspace isolation.

**Priority order** (when no pre-instruction and no handoffs):

1. **Issues WITHOUT a GitHub Project** (untracked issues) - high priority
2. **Issues with priority labels** (weighted random: high=10x, medium=3x, low=1x, none=0.5x)

```bash
# Find next available issue
# - If pre-instruction in Step 0: ax issue next --issue <NUMBER>
# - If normal discovery: ax issue next
ax issue next

# View issue details
gh issue view $ISSUE_NUMBER --json number,title,body,labels,state
```

### Issue Claiming (Step 4)

**⚠️ CRITICAL**: The `ax issue claim-atomic` command performs:

1. Checks if already claimed (idempotent if claimed by THIS workspace)
2. Adds "claimed" label to GitHub
3. Posts workspace identification comment
4. Initializes workflow state file
5. Atomic rollback on any failure

This replaces the previous separate steps of claiming + workflow init, saving time and preventing
orphaned claims.

```bash
# Atomically claim the issue with workflow state initialization
# Note: This command makes multiple GitHub API calls and may take 1-3 minutes
# Recommended timeout: 5 minutes (timeout: 300000) for Claude Code Bash tool
ax issue claim-atomic $ISSUE_NUMBER
```

**⏱️ Timeout Note**: This command performs multiple GitHub API operations (check claim status, add
label, post comment, initialize state). Allow 5-minute timeout to prevent exit code 137 (SIGKILL).
See CLAUDE.md Principle 3 for details.

**If claim fails due to duplicate work**:

- Error message will show existing PR number or branch name
- Immediately find a different issue (no wasted time)
- Example:
  `Duplicate work detected: PR #1234 already exists for issue #3097 (title). Another workspace may be working on this issue.`

### Issue Validation (Step 5)

**⚠️ CRITICAL**: Add already-done detection BEFORE other validation checks to prevent duplicate
work.

#### Step 5.1: Already-Done Detection (MANDATORY)

**⚠️ IMPORTANT**: Use MCP tool when available for better performance, with CLI fallback.

**Validation approach**:

- **MCP Tool (preferred)**: `mcp__ax-mcp__ax_issue_validate_actionable` with
  `{"issue": ISSUE_NUMBER}`
- **CLI Fallback**: `ax issue validate-actionable ISSUE_NUMBER`

**What this checks**:

1. Issue is still open
2. No merged PRs reference this issue (Fixes #N, Closes #N, Resolves #N)
3. No commits in main branch reference this issue
4. Issue not marked as "duplicate" or "wontfix"

**Expected Response (MCP)**:

```json
{
  "valid": true,
  "issue_number": 3974,
  "errors": [],
  "warnings": []
}
```

**Exit Codes (CLI)**:

- **Code 0**: Issue is actionable → Continue to other validation checks
- **Code 1**: Issue NOT actionable (already resolved) → STOP, find different issue
- **Code 2**: Error during validation → STOP, investigate

**If NOT actionable**:

```bash
# Log friction: Issue not actionable
ax friction append "$(cat <<EOF
## $(date '+%Y-%m-%d %H:%M') - Issue Not Actionable

**Issue**: Issue #$ISSUE_NUMBER failed actionability validation
**Category**: Validation failure
**Resolution**: Abandoned issue, returning to discovery
**Time Lost**: ~5 minutes investigation
EOF
)"

# Issue already resolved - find a different issue
ax issue abandon --issue $ISSUE_NUMBER
# Return to Step 3 (issue discovery)
```

#### Step 5.1.5: Close Non-Actionable Issues (When Applicable)

**⚠️ IMPORTANT**: If pre-flight check reveals issue is genuinely obsolete or no longer relevant,
close it to prevent future discovery attempts.

**Using `ultrathink`**: Analyze why issue is not actionable:

- Does the file/script referenced still exist?
- Has the problem been resolved another way?
- Is the issue obsolete due to architecture changes?
- Is the issue a duplicate that was missed?

**Examples of non-actionable issues that should be closed**:

- "fix script X" but script was deleted in refactor (reference PR number)
- "add feature Y" but feature already implemented differently (reference PR/commit)
- "bug in Z" but Z was removed from codebase (reference PR that removed it)
- "improve X" but X no longer exists or was replaced

**If issue is clearly non-actionable and you have confidence**:

```bash
# Close with detailed explanation referencing the evidence
CLOSE_REASON="Script tools/old-script.py no longer exists after refactor in #3850. This issue is obsolete."
gh issue close $ISSUE_NUMBER --comment "$CLOSE_REASON"

# Log closure decision
echo "🗑️  Closed non-actionable issue #$ISSUE_NUMBER: $CLOSE_REASON"

# Return to Step 3 (issue discovery)
```

**If uncertain about closure**:

- Add comment explaining why it might be non-actionable
- Add "needs-attention" label for human review
- Move to next issue (return to Step 3)
- **Do NOT close** without high confidence - false closures waste time

**Confidence threshold for closure**:

- ✅ **High confidence (CLOSE)**: File definitively doesn't exist, feature verifiably implemented,
  clear duplicate
- ⚠️ **Medium confidence (LABEL)**: Add "needs-attention" label and comment, move to next issue
- ❌ **Low confidence (SKIP)**: Just move to next issue without changes

#### Step 5.2: Additional Validation Checks

Execute remaining validation checks using `ultrathink`:

- Check for duplicates (similar unresolved issues)
- Verify problem still exists
- Verify issue is within project scope
- Verify issue is well-documented

### Complexity Assessment (Step 6)

Use `ultrathink` to evaluate:

- Does this touch multiple subsystems/components?
- Will this require 15+ **unrelated** files?
- Multiple distinct features bundled?
- Will this take more than 8 hours?
- Could be broken into smaller pieces?

### Architecture Assessment Gate (Step 6.5) - MANDATORY

**⚠️ CRITICAL**: This gate prevents tactical fixes when strategic migrations are needed. Use
`ultrathink` to evaluate issue against architecture principles.

**Purpose**: Identify when issues require migration to proper technology stack instead of
perpetuating technical debt.

**🔧 Boy Scout Exemption**: Trivial fixes in files you're already touching (<4 hours) don't require
architecture assessment. Just apply the fix.

**Boy scout cleanup examples (SKIP architecture assessment)**:

- Syntax errors in files you're editing
- Linting violations in code you're modifying
- Typos in documentation you're reading
- Missing type hints in functions you're calling
- Pre-commit auto-fixes (commit them)

**For all other issues, proceed with architecture assessment below.**

#### Architecture Decision Tree (Required for EVERY issue)

Use `ultrathink` to answer each question:

**1. Is this Python script doing CLI-worthy operations?**

```text
Examples of CLI-worthy operations:
- GitHub API interactions (issues, PRs, labels)
- Workflow state management
- Issue/PR operations
- Git operations
- Process orchestration

If YES → Should migrate to ax CLI (Go-based)
If NO → Continue to next question
```

**Decision**:

- ✅ **Migrate to ax CLI**: Create migration issue, document in current issue
- ❌ **Keep in Python**: Continue with current issue (document reasoning)

**2. Is this GitHub automation?**

```text
Examples:
- PR creation/management
- Issue labeling/claiming
- Comment automation
- CI/CD integration
- Repository operations

If YES → Should use Go (per CLAUDE.md Go-first principle)
If NO → Continue to next question
```

**Decision**:

- ✅ **Use Go**: Create migration issue or implement in Go
- ❌ **Exception justified**: Document why (e.g., one-time script, data analysis)

**3. Is this tool used by AI workflows?**

```text
Check if:
- Called from prompts/*.md or workflows/*.md
- Part of dev-cycle/pr-ready-cycle
- Used by autonomous agents
- Needs to be discoverable by AI

If YES → Needs MCP exposure → Requires Go implementation
If NO → Continue to next question
```

**Decision**:

- ✅ **MCP exposure needed**: Must be in ax CLI (Go) for discoverability
- ❌ **Internal tool only**: Can use appropriate language

**4. Does this violate Go-first principle?**

```text
Go-first applies to:
- CLI tools
- Servers/services
- GitHub automation
- Workflow orchestration

Exceptions (not Go-first):
- Installation/setup → Shell
- ML/data science → Python
- Highly complex computation → Rust

If VIOLATION → Should create migration issue
If EXCEPTION → Document reasoning
```

**Decision**:

- ✅ **Violation detected**: Create migration issue, block implementation
- ❌ **Valid exception**: Continue with appropriate language (document why)

#### Architecture Assessment Workflow

**Step 6.5.1**: Run Architecture Decision Tree using `ultrathink`

```bash
# Example ultrathink reasoning format (for AI internal processing):
#
# Issue #3947: Fix Python tests for prepare_all_prs.py
#
# Question 1: CLI-worthy operations?
#   - prepare_all_prs.py does GitHub PR operations
#   - Used for automation
#   → YES, CLI-worthy
#
# Question 2: GitHub automation?
#   - Creates/manages PRs
#   - Repository operations
#   → YES, GitHub automation
#
# Question 3: Used by AI workflows?
#   - Called from workflows (grep confirms)
#   - Part of pr-ready automation
#   → YES, needs MCP exposure
#
# Question 4: Violates Go-first?
#   - GitHub automation → Should be Go
#   - Currently Python
#   → YES, violation
#
# ARCHITECTURE DECISION: MIGRATION REQUIRED
#   Reason: GitHub automation + AI-used → Must be Go + MCP
#   Action: Create migration issue, document in #3947
```

**Step 6.5.2**: Execute Architecture Decision

**If MIGRATION REQUIRED** (any decision tree question flagged for migration):

```bash
# Create migration issue
MIGRATION_TITLE="refactor: migrate [TOOL_NAME] to ax CLI with MCP exposure"
MIGRATION_BODY="$(cat <<EOF
## Migration Justification

**Original Issue**: #${ISSUE_NUMBER}
**Current Implementation**: [Language/Tool]
**Required Implementation**: Go (ax CLI)

## Architecture Assessment Results

1. CLI-worthy operations: [YES/NO - reasoning]
2. GitHub automation: [YES/NO - reasoning]
3. Used by AI workflows: [YES/NO - reasoning]
4. Go-first violation: [YES/NO - reasoning]

## Migration Requirements

- [ ] Implement in Go as ax command
- [ ] Add MCP server exposure (if AI-used)
- [ ] Migrate functionality from [current tool]
- [ ] Update workflows to use ax command
- [ ] Remove old Python implementation
- [ ] Update documentation

## Acceptance Criteria

- [ ] ax command functional with same behavior
- [ ] MCP tools available (if needed)
- [ ] All workflows updated
- [ ] Tests passing
- [ ] Old implementation removed
EOF
)"

# Create the migration issue
MIGRATION_ISSUE=$(ax issue create \
  --title "$MIGRATION_TITLE" \
  --body "$MIGRATION_BODY" \
  --label "enhancement,refactor,architecture" | grep -oE '#[0-9]+' | tr -d '#')

echo "✅ Created migration issue #$MIGRATION_ISSUE"

# Update current issue with migration recommendation
CURRENT_ISSUE_COMMENT="$(cat <<EOF
## ⚠️ Architecture Assessment: Migration Required

**Architecture gate detected**: This issue should be addressed through strategic migration, not tactical fix.

**Migration Issue**: #${MIGRATION_ISSUE}

**Reasoning**:
- CLI-worthy operations: [YES/NO]
- GitHub automation: [YES/NO]
- Used by AI workflows: [YES/NO]
- Go-first violation: [YES/NO]

**Recommendation**: Close this issue in favor of migration issue #${MIGRATION_ISSUE}, or update this issue to become the migration issue.

**See**: CLAUDE.md Section 'Language Selection for Development' for Go-first principle.
EOF
)"

gh issue comment $ISSUE_NUMBER --body "$CURRENT_ISSUE_COMMENT"

echo "❌ BLOCKING: Issue requires migration, not tactical fix"
echo "   Migration issue: #$MIGRATION_ISSUE"
echo "   Abandoning current issue and returning to discovery"

# Abandon current issue and return to discovery
ax issue abandon --issue $ISSUE_NUMBER

# Return to Step 3 (issue discovery)
exit 1
```

**If NO MIGRATION REQUIRED** (all decision tree questions passed):

```bash
echo "✅ Architecture assessment passed"
echo "   - Implementation approach aligns with Go-first principle"
echo "   - No migration flags detected"
echo "   - Proceeding to Goal/Architecture Alignment"

# Continue to Step 7
```

**Step 6.5.3**: Document Architecture Decision

```bash
# Log architecture assessment in friction log (even if passed)
ax friction append "$(cat <<EOF
## $(date '+%Y-%m-%d %H:%M') - Architecture Assessment

**Issue**: #$ISSUE_NUMBER
**Decision Tree Results**:
  - CLI-worthy operations: [YES/NO - reasoning]
  - GitHub automation: [YES/NO - reasoning]
  - Used by AI workflows: [YES/NO - reasoning]
  - Go-first violation: [YES/NO - reasoning]

**Final Decision**: [MIGRATION REQUIRED / APPROVED]
**Reasoning**: [Summary of ultrathink analysis]
**Time Spent**: ~3 minutes
EOF
)"
```

#### Why This Matters

**Example: Issue #3947**

- **What happened**: Fixed Python tests instead of recognizing migration need
- **What should have happened**: Architecture gate flags:
  - ✅ CLI-worthy: GitHub automation
  - ✅ GitHub automation: PR operations
  - ✅ AI-used: Called from workflows
  - ✅ Go-first violation: Should be Go, currently Python
- **Result**: Create migration issue, not tactical fix

**Prevents**:

- Perpetuating technical debt
- Tactical fixes for strategic problems
- Missing migration opportunities
- Architecture principle violations

**Ensures**:

- Strategic thinking about technology choices
- Alignment with Go-first principle
- Proper MCP exposure for AI-used tools
- Discoverable, maintainable tooling

### Goal/Architecture Alignment (Step 7)

Use `ultrathink` to analyze alignment with:

- Python-first development
- Cycle CLI migration
- Workflow automation
- Quality improvement
- Composable patterns

### Branch Creation (Step 8)

```bash
ax git create-branch --issue $ISSUE_NUMBER --prefix fix
```

### Verify Workflow State (Step 9)

**⚠️ NOTE**: Workflow state was already initialized by `claim-atomic` in Step 4.

```bash
# Verify workflow state is correct
ax workflow get
```

### Transition to Implementation (Step 10)

```bash
# Verify feature branch exists (MANDATORY)
ax git check-branch

# Initialize workflow state if not exists (handles edge cases)
# Check if workflow state exists using ax workflow get
# If state doesn't exist, initialize it before transition
ax workflow get > /dev/null 2>&1 || ax workflow init --phase implementation --issue $ISSUE_NUMBER

# Transition to implementation phase
ax workflow transition

# Continue to implementation prompt (MANDATORY)
prompt implementation
```

**⚠️ MANDATORY**: Workflow must continue through all phases without stopping.

**Why initialize before transition**: Although `claim-atomic` should initialize state in Step 4,
this fallback handles edge cases where state initialization may have failed or been interrupted.

---

## AI Decision Delegation Pattern

### Overview

The compound tool approach delegates intelligence-intensive decisions to the AI agent while ensuring
all decisions are logged for audit trails and pattern learning.

### Analysis → Decision → Execution Flow

```text
1. Analysis Tool (ax_discover_analyze_issue)
   ↓ Returns structured data + intelligence hints
2. AI Agent applies extended thinking (`ultrathink`)
   ↓ Makes decisions with reasoning
3. Execution Tool (ax_discover_claim_and_setup)
   ↓ Captures AI decisions + executes atomic operations
4. Session Logging
   ↓ Records all AI reasoning for retrospective analysis
```

### Intelligence Priority Levels

**intelligencePriority: 1.0** (Discovery phase default)

- **Claude**: Triggers extended thinking mode
- **GPT-4**: Uses deep reasoning (o1-preview)
- **Gemini**: Enables advanced reasoning mode
- **Other LLMs**: Use most capable model available

### Extended Thinking Guidance

**When to use `ultrathink`** in Discovery phase:

1. **Duplicate Detection** (CRITICAL)
   - Analyze similarity scores and scope overlap
   - Detect subsumes relationships (issue A contains issue B's scope)
   - Example reasoning:
     ```text
     Issue #3850 (retry logic) vs #3201 (timeout config):
     - Different scopes: retry mechanism vs timeout values
     - Complementary, not duplicate
     - #3850 adds new functionality, doesn't replace #3201
     ```

2. **Scope Validation** (CRITICAL)
   - Check alignment with agent-first paradigm
   - Verify fit with CLAUDE.md principles
   - Example reasoning:
     ```text
     Issue #3850 (retry logic for wait-copilot):
     - ✅ Agent-first: Improves automation reliability
     - ✅ CLAUDE.md: Uses ax commands, not subprocess calls
     - ✅ Composable: Retry pattern reusable across commands
     ```

3. **Architecture Alignment** (IMPORTANT)
   - Verify consistency with project goals
   - Check for technical debt introduction
   - Example reasoning:
     ```text
     Issue #3850 implementation approach:
     - ✅ Python-first: Implement in Go (ax CLI is Go-based)
     - ✅ Workflow automation: Reduces manual waiting
     - ✅ Quality improvement: Handles transient failures
     ```

### Example: Complete Discovery with Compound Tools

```bash
# Step 1: Analysis (gather data)
result = ax_discover_analyze_issue(
  issue_number: 3850  # From pre-instruction or use_next: true
)

# Step 2: AI Decision (using ultrathink)
# CRITICAL: Use extended thinking for these decisions
#
# NOTE: "ultrathink:" below is conceptual notation for AI agents,
# not an executable bash command. It indicates where to apply extended thinking.

# Duplicate Detection Analysis
ultrathink:  # Conceptual - triggers extended thinking mode in AI
  Analyzing duplicate_analysis.candidates:
  - Candidate #3201: "Improve timeout handling" (similarity: 0.78)
  - Scope comparison:
    * #3201: Configuration of timeout values (10min vs 5min)
    * #3850: Exponential backoff retry logic on transient failures
  - Relationship: Complementary, not duplicate
  - #3850 solves different problem: retry on failure vs wait longer

  Decision: NOT a duplicate
  Reasoning: "Unique scope: retry logic vs timeout config"

# Scope Validation Analysis
ultrathink:  # Conceptual - triggers extended thinking mode in AI
  Checking agent-first alignment:
  - Improves automation reliability (agents wait for Copilot reviews)
  - Reduces friction from transient failures
  - Composable pattern (can apply to other wait commands)

  Decision: IN SCOPE
  Reasoning: "Agent-first: improves automation reliability"

# Encode decisions
duplicate_decision = {
  is_duplicate: false,
  reasoning: "Unique scope: retry logic vs timeout config (#3201)"
}

scope_decision = {
  in_scope: true,
  reasoning: "Agent-first: improves automation reliability"
}

# Step 3: Execution (atomic operations with AI decisions)
ax_discover_claim_and_setup(
  issue_number: 3850,
  branch_prefix: "fix",
  duplicate_decision: duplicate_decision,
  scope_decision: scope_decision
)

# Result: Issue claimed, branch created, workflow initialized
# Session log captures all AI reasoning for retrospective analysis
```

### Intelligence Hints in Analysis Results

The `ax_discover_analyze_issue` tool returns intelligence hints:

```json
{
  "intelligence_required": {
    "level": "high",
    "decisions": ["duplicate_detection", "scope_validation", "architecture_alignment"],
    "prompt_hint": "Analyze scope overlap, subsumes relationships, architecture fit"
  },
  "model_preferences": {
    "intelligencePriority": 1.0,
    "costPriority": 0.0,
    "speedPriority": 0.0
  }
}
```

**How to use these hints**:

- `level: "high"` → Trigger extended thinking mode
- `decisions` → List of required AI decisions
- `prompt_hint` → Guidance on what to analyze
- `model_preferences` → Optimize for intelligence over cost/speed

### Session Logging

All execution tools log AI decisions for audit and learning:

```bash
# Logged by ax_discover_claim_and_setup (example format shown below)
# The actual implementation handles multi-line data internally
```

**Example session log entry:**

```text
Phase: Discovery
Issue: #3850
Decision: NOT duplicate
AI Reasoning: Unique scope - retry logic vs timeout config
Scope: IN SCOPE
AI Reasoning: Agent-first alignment
```

**Benefits**:

1. Full audit trail of AI decisions
2. Pattern learning for future cycles
3. Debugging when decisions are wrong
4. Retrospective analysis data

---

## Error Handling

**Blocking errors that STOP work and release claim**:

- Workspace validation failed
- No issues available (wait and retry)
- Race condition (issue already claimed)
- Issue closed during discovery
- Duplicate detected
- Problem no longer exists (or unclear)
- Out of scope (or scope unclear)
- Insufficient documentation
- Low alignment (<40%)
- Branch already exists with PR
- Not on main branch
- Detached HEAD (auto-fix failed)
- Not in git repository
- Still on main after branch creation

**For complete error handling procedures**: See `workflows/DISCOVERY_WORKFLOW.md`

---

## Related Documentation

- **Complete Workflow**: `~/git/dotfiles/workflows/DISCOVERY_WORKFLOW.md`
- **Specification**: `~/git/dotfiles/specifications/phase-1-discovery-specification.feature`
- **Architecture**: `~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md`
- **Quality Principles**: `~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md`
- **Agent Boundaries**: `~/git/dotfiles/docs/AGENT_BOUNDARIES.md`
- **Automation Policy**: `~/git/dotfiles/docs/AUTOMATION_POLICY.md`
