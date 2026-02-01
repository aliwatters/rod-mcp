# Implementation Phase: Code Changes and Testing

**Purpose**: Execute implementation phase using compound tools for efficient development workflow.

**Phase**: 2 (Implementation) - Executed after Discovery

**Pattern**: Analysis → AI Work → Execution (see COMPOUND_TOOLS_GUIDE.md)

## Overview

The Implementation phase uses **2 compound tool calls** (with AI work in between) instead of 15+
individual commands:

1. **Analysis Tool** (`ax_implement_prepare()`) - Validates readiness and gathers context
2. **AI Implementation Work** - Design, code, test, document (AI's responsibility)
3. **Execution Tool** (`ax_implement_commit_and_pr()`) - BLOCKING quality gates + PR creation

**Benefits**:

- 87% reduction in Implementation phase tool calls (15+ → 2)
- Intelligent context gathering (files to examine, related patterns)
- Mandatory quality gates (tests must pass before PR creation)
- Automatic session logging of implementation decisions

**⚠️ New MCP Tools**: If you CREATE a new MCP tool during implementation, it will NOT be available until Claude Code is restarted. MCP servers run as child processes - `go build` creates a new binary but the running process uses the old code. For development iteration, use [MCP Inspector](../../../../docs/AX_MCP_INTEGRATION.md#development-workflow-binary-reload-behavior).

## Prerequisites

**⚠️ MANDATORY**: Follow all instructions carefully - skipping steps will cause workflow failures.

**Required before Implementation**:

- Active workflow state (`PHASE=implementation`, `ISSUE` and `BRANCH` set)
- Feature branch created (currently on feature branch, not main)
- Issue claimed (issue has "claimed" label)
- Understanding of issue requirements and acceptance criteria

## Automation Level

**Automation Level**: SEMI-AUTOMATED with BLOCKING gates

**Automation Continues When**:

- ✅ All quality gates pass (`ax checkin`)
- ✅ Tests are added and passing
- ✅ No architecture violations

**Automation Stops When**:

- ❌ Quality gates fail (linting, tests, validation)
- ❌ Tests are missing or failing
- ⚠️ Architecture decisions need extended thinking

**See**: `~/git/dotfiles/docs/AUTOMATION_POLICY.md`

## Compound Tool Pattern

### Step 1: Analysis (Validate Readiness and Gather Context)

**MCP Tool (preferred)**:

```text
mcp__ax-mcp__ax_implement_prepare()
```

**CLI Fallback**: Not available - MCP tool only

```bash
# NOTE: No CLI equivalent exists for this compound tool
# Use the MCP tool above or manually gather context:
# ax workflow get
# ax git check-branch
# gh issue view $ISSUE_NUMBER
# (additional commands needed - see implementation)
```

**Validates**:

- ✅ On feature branch (not main)
- ✅ Issue has "claimed" label
- ✅ Workflow state is correct (`PHASE=implementation`)
- ✅ No workspace conflicts

**Provides Context**:

```json
{
  "validation": {
    "on_feature_branch": true,
    "has_claimed_label": true,
    "has_in_progress_label": true,
    "workflow_state_valid": true,
    "ready": true
  },
  "context": {
    "issue": {
      "number": 3926,
      "title": "feat: complete Phase 3 compound tools integration",
      "requirements": "...",
      "acceptance_criteria": ["...", "..."]
    },
    "files_to_examine": ["prompts/dev-cycle/setup.md", "prompts/dev-cycle/discovery.md", "prompts/dev-cycle/implementation.md"],
    "related_patterns": ["prompts/dev-cycle/setup.md (compound tool usage example)"]
  },
  "intelligence_required": {
    "level": "high",
    "decisions": ["architecture_design", "edge_case_analysis", "solution_design"],
    "prompt_hint": "Use extended thinking for architecture decisions and edge case analysis"
  },
  "model_preferences": {
    "intelligencePriority": 1.0,
    "costPriority": 0.0,
    "speedPriority": 0.0
  }
}
```

**If Validation Fails**:

- Tool returns error with remediation steps
- Fix issues (return to main branch, claim issue, etc.)
- Retry analysis tool

### Step 2: AI Implementation Work

**⚠️ CRITICAL: AI Decision Delegation**

The AI (you) is responsible for all implementation work:

#### 2.1 Architecture Design (Use Extended Thinking)

**When to use extended thinking**:

- Complex design decisions (patterns, interfaces, data structures)
- Multiple valid approaches with trade-offs
- Edge cases requiring careful analysis
- Cross-cutting concerns (error handling, validation)

**Design Questions**:

- How should this be structured?
- Which design patterns should we follow?
- What are the edge cases and failure modes?
- How does this fit with existing architecture?
- What are the performance implications?

**Consult**:

- `context.files_to_examine` - Understand existing code
- `context.related_patterns` - Follow established patterns
- CLAUDE.md principles (Go-first, no-subprocess, etc.)

#### 2.2 Code Implementation

**Process**:

1. Read files from `context.files_to_examine`
2. Review `context.related_patterns` for consistency
3. Write code following project standards
4. Ensure no architecture violations (CLAUDE.md Principles 1 & 2)
5. Add appropriate error handling
6. Document complex logic with comments

**Quality Standards**:

- ✅ Follow conventional commit format (feat:, fix:, docs:, etc.)
- ✅ Use proper type hints (no `any` types in Python/TypeScript)
- ✅ Follow established code patterns
- ✅ No subprocess calls in Python (use SDKs)
- ✅ Workflows use `ax` commands (not Python scripts directly)

#### 2.3 Boy Scout Cleanup (Proactive Quality Improvement)

**⚠️ BOY SCOUT RULE**: "Leave your campsite cleaner than you found it"

**Philosophy**: Apply boy scout rule **proactively** (during implementation) instead of **reactively** (when Copilot flags issues in review).

**Time Budget**: 15-30 minutes total for proactive cleanup

- **Prevents**: 60-120 minutes of review cycles + comment replies
- **ROI**: 2-4x time savings by being proactive

**When to apply**: While implementing, actively look for improvement opportunities in files you're already touching.

**Using `ultrathink`**: Before committing, analyze files you touched for boy scout opportunities.

**Questions to ask yourself**:

- Is there repeated code I could DRY up? (<15 min)
- Are there documentation bugs in functions I called? (<5 min)
- Did I notice stale comments in code I read? (<2 min)
- Are there edge cases I should test? (<10 min)
- Are there magic strings/numbers I should extract? (<5 min)
- Is error handling inconsistent? (<5 min)

**⚠️ Scope Creep Detection**: If cleanup would exceed 30 minutes, **STOP** and create a separate issue. Reference issue #4305 for scope creep detection guidance.

##### What to Look For (6 Common Opportunities)

**1. DRY Violations (Repeated Code)**

**Example**: Found while implementing new validation

```python
# ❌ BEFORE: Duplicated validation logic
def validate_user(user):
    if not user.name:
        raise ValueError("Name required")
    if len(user.name) < 3:
        raise ValueError("Name too short")
    # ... 10 more validations

def validate_admin(admin):
    if not admin.name:
        raise ValueError("Name required")
    if len(admin.name) < 3:
        raise ValueError("Name too short")
    # ... same 10 validations copied

# ✅ AFTER: Extracted common validation
def validate_name(name):
    if not name:
        raise ValueError("Name required")
    if len(name) < 3:
        raise ValueError("Name too short")

def validate_user(user):
    validate_name(user.name)
    # ... user-specific validations

def validate_admin(admin):
    validate_name(admin.name)
    # ... admin-specific validations
```

**Time**: 10-15 minutes
**When**: You're already in this file adding a new validator
**Why**: Prevents future copy-paste bugs, easier to maintain

**2. Documentation Bugs**

**Example**: Found while reading function to understand how to call it

```python
# ❌ BEFORE: Wrong parameter name in docstring
def retry_with_backoff(max_retries=3):
    """
    Retry with exponential backoff.

    Args:
        max_attempts: Maximum number of retry attempts  # Wrong!

    Returns:
        Result of successful attempt
    """
    pass

# ✅ AFTER: Corrected while reading
def retry_with_backoff(max_retries=3):
    """
    Retry with exponential backoff.

    Args:
        max_retries: Maximum number of retry attempts  # Correct!

    Returns:
        Result of successful attempt
    """
    pass
```

**Time**: 30 seconds
**When**: You're reading this function to understand how to call it
**Why**: Prevents confusion for next developer (maybe you tomorrow)

**3. Stale Comments**

**Example**: Found while modifying nearby code

```go
// ❌ BEFORE: Outdated TODO comment
// TODO: Add retry logic - Issue #1234
func WaitForCopilot() error {
    // Retry logic was added in #2850 but comment not updated
    return retryWithBackoff(func() error {
        return checkCopilotReview()
    })
}

// ✅ AFTER: Removed stale TODO
func WaitForCopilot() error {
    return retryWithBackoff(func() error {
        return checkCopilotReview()
    })
}
```

**Time**: 10 seconds
**When**: You're in this file anyway
**Why**: Reduces noise, prevents confusion

**4. Missing Edge Case Tests**

**Example**: Found while implementing new feature

```python
# ❌ BEFORE: Only happy path tested
def test_validate_issue():
    """Test issue validation."""
    assert validate_issue(valid_issue) == True
    # Missing: empty issue, malformed issue, closed issue

# ✅ AFTER: Added edge cases discovered during implementation
def test_validate_issue():
    """Test issue validation."""
    assert validate_issue(valid_issue) == True

    # Edge cases discovered during implementation
    with pytest.raises(ValueError):
        validate_issue(None)  # Empty issue

    assert validate_issue(closed_issue) == False  # Closed issue

    with pytest.raises(ValueError):
        validate_issue(malformed_issue)  # Malformed issue
```

**Time**: 5-10 minutes
**When**: You're already writing tests for your feature
**Why**: Better test coverage, prevents regressions

**5. Magic Numbers/Strings**

**Example**: Found while adding new status check

```go
// ❌ BEFORE: Magic string
func CheckPRStatus(pr int) bool {
    status := getPRStatus(pr)
    if status == "ready_for_review" {  // Magic string
        return true
    }
    return false
}

// ✅ AFTER: Extracted constants
const (
    StatusReadyForReview = "ready_for_review"
    StatusDraft          = "draft"
    StatusMerged         = "merged"
)

func CheckPRStatus(pr int) bool {
    status := getPRStatus(pr)
    return status == StatusReadyForReview
}
```

**Time**: 3-5 minutes
**When**: You notice magic strings while adding similar code
**Why**: Easier to refactor, prevents typos

**6. Inconsistent Error Handling**

**Example**: Found while adding error handling to new code

```python
# ❌ BEFORE: Inconsistent error handling
def fetch_pr(pr_num):
    pr = api.get_pr(pr_num)
    if not pr:
        print("PR not found")  # Prints instead of raises
        return None
    return pr

def fetch_issue(issue_num):
    issue = api.get_issue(issue_num)
    if not issue:
        raise ValueError(f"Issue {issue_num} not found")  # Correct
    return issue

# ✅ AFTER: Made consistent
def fetch_pr(pr_num):
    pr = api.get_pr(pr_num)
    if not pr:
        raise ValueError(f"PR {pr_num} not found")  # Now consistent
    return pr
```

**Time**: 2 minutes
**When**: You're adding similar error handling
**Why**: Consistent error handling is easier to reason about

##### Boy Scout Checklist (Before Committing)

**⚠️ MANDATORY**: Review this checklist before calling execution tool.

**While implementing, did I notice and fix:**

- [ ] **DRY violations?** (Repeated code I could extract) - Budget: <15 min
- [ ] **Documentation bugs?** (Wrong param names, stale examples) - Budget: <5 min
- [ ] **Stale comments?** (Outdated TODOs, wrong references) - Budget: <2 min
- [ ] **Missing edge case tests?** (Cases discovered while implementing) - Budget: <10 min
- [ ] **Magic numbers/strings?** (Should be constants) - Budget: <5 min
- [ ] **Inconsistent patterns?** (Error handling, naming, structure) - Budget: <5 min

**Total time budget**: 15-30 minutes for all boy scout cleanup

**⚠️ If cleanup exceeds 30 minutes**: Create separate issue and defer (actual scope creep). See issue #4305 for guidance.

**Benefits**:

- ✅ Cleaner code before Copilot review
- ✅ Fewer review cycles (60-120 min saved)
- ✅ Proactive vs reactive improvements
- ✅ Compound benefits (each cleanup makes future work easier)

#### 2.4 Test Coverage (MANDATORY)

**⚠️ CRITICAL**: Tests are MANDATORY for all code changes.

**Test Requirements**:

| Change Type               | Test Requirement                    |
| ------------------------- | ----------------------------------- |
| New functionality         | New tests (happy path + edge cases) |
| Bug fixes                 | Regression tests                    |
| Refactoring               | Existing tests must pass            |
| Documentation/config only | Tests not required                  |

**Test Patterns**:

- Happy path scenarios
- Edge cases (empty input, null values, boundary conditions)
- Error conditions (invalid input, failures, timeouts)
- Integration points (API calls, file operations, etc.)

#### 2.5 Documentation Updates

**Update if applicable**:

- Code comments for complex logic
- README files for new features
- CLAUDE.md for new patterns
- Workflow documentation for process changes
- Specification files if behavior changes

#### 2.6 Pre-Commit Verification Checklist

**⚠️ MANDATORY**: Complete this checklist BEFORE calling execution tool.

**Requirement Verification**:

- [ ] All issue requirements addressed?
- [ ] All acceptance criteria met?
- [ ] Edge cases from issue handled?
- [ ] No scope creep (only solving THIS issue)?

**Code Quality**:

- [ ] Architecture principles followed (CLAUDE.md)?
- [ ] No subprocess calls (workflows → ax commands → SDKs)?
- [ ] Proper type hints (no `any` types)?
- [ ] Conventional commit format?

**Testing**:

- [ ] Tests added for new functionality?
- [ ] Regression tests for bug fixes?
- [ ] All tests passing locally?

**Boy Scout Cleanup**:

- [ ] Reviewed boy scout checklist (Section 2.3)?
- [ ] Applied proactive cleanup within 15-30 min budget?
- [ ] Created separate issue if cleanup exceeds 30 min?

**Documentation**:

- [ ] Code comments for complex logic?
- [ ] Documentation updated if needed?
- [ ] Specifications updated if behavior changed?

**⚠️ If ANY checkbox is unchecked, DO NOT proceed to Step 3!**

### Step 3: Execution (Quality Gates + Commit + PR)

**MCP Tool (preferred)**:

```text
mcp__ax-mcp__ax_implement_commit_and_pr({
  "commit_message": "fix: add exponential backoff retry

- Implement retry logic with exponential backoff
- Add tests for retry behavior
- Update documentation

Fixes #3850",
  "pr_title": "fix: add retry logic to wait-copilot",
  "pr_body": "## Summary

Add exponential backoff retry logic to wait-copilot command.

## Changes
- Modified: wait_copilot.go (retry logic)
- Added: wait_copilot_test.go (test coverage)

## Testing
- ✅ Unit tests pass
- ✅ Integration tests pass
- ✅ Manual testing completed

Fixes #3850",
  "implementation_summary": {
    "files_modified": ["wait_copilot.go"],
    "tests_added": ["wait_copilot_test.go"]
  }
})
```

**CLI Fallback**: Not available - MCP tool only

```bash
# NOTE: No CLI equivalent exists for this compound tool
# Use the MCP tool above or manually execute:
# ax checkin  # BLOCKING - must pass
# git add . && git commit -m "..."
# git push
# ax pr create --draft --title "..." --body "..."
# ax workflow update --phase refinement
```

**⚠️ CRITICAL: BLOCKING Quality Gates**

The execution tool runs `ax checkin` as a **BLOCKING** quality gate:

**Quality Gates Enforced**:

1. ✅ Linting passes (ruff lint)
2. ✅ Formatting correct (ruff format)
3. ✅ Architecture validation (no CLAUDE.md violations)
4. ✅ Tests pass (pytest)
5. ✅ No import errors

**If Quality Gates Fail**:

```text
ERROR: Quality gates failed
- 3 linting errors in prompts/discovery.md
- 2 test failures in test_workflow.py
- 1 architecture violation: workflow calls Python script directly

AI MUST:
1. Review error output
2. Fix all issues
3. Re-run execution tool
4. Repeat until tests pass

NO PR will be created until quality gates pass
```

**Internal Execution Steps** (automatic):

<details>
<summary><b>Reference Implementation</b></summary>

```bash
# STEP 1: MANDATORY quality gates (BLOCKING)
ax checkin || {
  ax session append --data "Implementation: Quality gates FAILED"
  return error "Tests must pass before PR creation"
}

# STEP 2: Session log
ax session append --data "Implementation: Quality gates PASSED
Files: $(echo $FILES_MODIFIED | tr ',' '\n')
Tests: $(echo $TESTS_ADDED | tr ',' '\n')"

# STEP 3: Commit + Push
git add . && git commit -m "$COMMIT_MESSAGE"
git push origin $(git branch --show-current)

# STEP 4: Create draft PR
ax pr create --draft --title "$PR_TITLE" --body "$PR_BODY"

# STEP 5: Update workflow state
ax workflow update --pr $PR_NUMBER --phase refinement
```

</details>

**Returns**:

```json
{
  "success": true,
  "quality_gates_passed": true,
  "pr": {
    "number": 3855,
    "state": "draft",
    "labels": ["ready for robots"]
  },
  "workflow_phase": "refinement",
  "next_phase": "refinement",
  "session_logged": true
}
```

### Iteration Pattern

**If quality gates fail on first attempt**:

1. Review error output from tool
2. Fix issues (linting, tests, validation)
3. **Log friction immediately** using heredoc syntax to preserve multi-line content:

   ```bash
   ax friction append "$(cat <<EOF
   ## $(date '+%Y-%m-%d %H:%M') - Quality Gate Failure

   **Issue**: Quality gates failed during ax checkin
   **Command**: ax checkin
   **Error**: 3 linting errors, 2 test failures
   **Category**: Build/test/lint failure
   **Resolution**: Fixed linting issues, corrected test assertions
   **Time Lost**: ~15 minutes
   EOF
   )"
   ```

4. Re-run `mcp__ax-mcp__ax_implement_commit_and_pr` with same arguments
5. Repeat until tests pass

**Example Failure Scenario**:

```bash
# First attempt fails
mcp__ax-mcp__ax_implement_commit_and_pr({...})
# Error: Quality gates failed - 3 linting errors, 2 test failures

# Fix issues
# (make corrections)

# Retry (same arguments)
mcp__ax-mcp__ax_implement_commit_and_pr({...})
# Success: PR #3855 created
```

## Error Handling

### Analysis Tool Failures

**Error**: Feature branch validation failed

```json
{
  "validation": {
    "ready": false,
    "on_feature_branch": false
  },
  "error": {
    "type": "not_on_feature_branch",
    "message": "Currently on main branch, expected feature branch",
    "remediation": "Create feature branch: ax git create-branch --issue <NUM> --prefix <TYPE>"
  }
}
```

**Action**: Create feature branch, retry analysis

### Execution Tool Failures

**Error**: Quality gates failed

```json
{
  "success": false,
  "quality_gates_passed": false,
  "error": {
    "type": "quality_gates_failed",
    "details": {
      "lint_errors": 3,
      "test_failures": 2,
      "output": "..."
    },
    "remediation": "Fix linting and test failures, then retry"
  }
}
```

**Action**: Fix issues, log friction, retry

## Extended Thinking Guidance

**Use extended thinking (`ultrathink`) for**:

### Architecture Decisions

- Design pattern selection (factory, strategy, observer, etc.)
- Interface design (method signatures, parameters, return types)
- Data structure choices (maps, lists, sets, trees)
- Cross-cutting concerns (logging, error handling, validation)

### Solution Design

- Root cause analysis (why does this bug exist?)
- Elegant approaches (simple solutions to complex problems)
- Trade-off evaluation (performance vs readability vs maintainability)

### Edge Case Analysis

- Failure modes (what can go wrong?)
- Boundary conditions (empty, null, max, min)
- Race conditions (concurrent access, timeouts)
- Error propagation (how do errors bubble up?)

**Example Extended Thinking**:

```markdown
Using `ultrathink`: Design retry logic architecture

**Problem**: Add exponential backoff retry to wait-copilot command

**Design Questions**:

1. Where should retry logic live? (utility, inline, middleware)
2. What parameters are configurable? (max retries, base delay, max delay)
3. How to handle timeout vs retry exhaustion?
4. Should retries be logged? How?

**Options**: A. Inline retry loop in wait-copilot command B. Generic retry utility in pkg/retry C.
Middleware pattern with retry decorator

**Analysis**:

- Option A: Simple, but not reusable
- Option B: Reusable, testable, follows Go best practices
- Option C: Over-engineered for this use case

**Decision**: Option B - Generic retry utility

- Reusable across commands (check-ci, wait-copilot, etc.)
- Easy to test in isolation
- Follows established pattern in pkg/
- Configurable parameters (max retries, delays)

**Implementation**:

- Create pkg/retry/retry.go with Retry() function
- Use functional options pattern for configuration
- Exponential backoff: delay = base \* 2^attempt
- Max delay cap to prevent infinite waits
- Context cancellation support
```

## Session Logging

**Automatic Logging**: Execution tool logs AI decisions to session for audit trail.

**Example Log Entry**:

```text
Implementation: Quality gates PASSED
Issue: #3926
Files Modified:
  - prompts/dev-cycle/setup.md
  - prompts/dev-cycle/implementation.md
Tests Added:
  - (documentation changes, no tests required)
Architecture Decisions:
  - Compound tool pattern throughout
  - MCP tool primary, CLI fallback secondary
PR Created: #3927 (draft, ready for robots)
```

## Friction Logging

**⚠️ MANDATORY**: Log ALL friction immediately when encountered.

**What to log**:

- Failed commands (full error output)
- Repeated attempts (how many, why)
- Unexpected delays (what took longer than expected)
- Confusing error messages (what was unclear)
- Missing documentation (what wasn't documented)
- Tool failures (MCP or CLI errors)

**Location**: `tmp/friction-<issue-no>.md`

**Format**:

```markdown
## 2025-11-10 12:34 - Quality Gate Failure

**Issue**: Linting errors in prompts/discovery.md **Command**: ax checkin **Error**:
prompts/discovery.md:123: Use absolute path ~/git/dotfiles/... not relative workflows/...
prompts/discovery.md:456: Use absolute path ~/git/dotfiles/... not relative workflows/...

**Resolution**: Updated all relative paths to absolute paths **Time Lost**: ~10 minutes
**Category**: Linting **Suggestion**: Add pre-commit hook to catch path issues earlier
```

## Specification Validation

**⚠️ CRITICAL**: If Gherkin specifications exist for this work, validate implementation BEFORE
creating PR.

**Check for specifications**:

```bash
SPEC_FILES=$(ax workflow get | grep "^SPEC_FILES=" | cut -d= -f2)

if [ "$SPEC_FILES" != "none" ] && [ -n "$SPEC_FILES" ]; then
    echo "🔍 Specification validation checkpoint triggered"
    echo "Specifications: $SPEC_FILES"
    echo ""
    echo "⚠️  You MUST validate implementation using ultrathink"
    echo "⚠️  Review ALL Given-When-Then scenarios"
fi
```

**If specifications exist**:

1. Read each specification file
2. For each scenario (Given-When-Then):
   - Identify preconditions (Given)
   - Identify action (When)
   - Identify expected outcome (Then)
3. Verify implementation satisfies scenario
4. Generate adherence report (✅ satisfied, ❌ violated, ⚠️ uncertain)

**If all scenarios satisfied**: Proceed to execution tool

**If violations found**:

- Block PR creation
- Fix implementation
- Re-validate
- Repeat until all satisfied

**See**: `docs/SPECIFICATIONS.md` for complete validation guide

## Testing Implementation Phase

### Manual Testing

**Test compound tool pattern**:

```bash
# Create test issue and claim it
gh issue create --title "Test implementation phase" --body "Test"
ax issue claim <NUM>

# Create feature branch
ax git create-branch --issue <NUM> --prefix test

# Update workflow state
ax workflow update --phase implementation --issue <NUM>

# Test analysis tool (MCP)
# (call via Claude Code MCP interface)

# Make some changes
echo "test" > test-file.txt

# Test execution tool (MCP)
# (call via Claude Code MCP interface with commit message, PR details)

# Verify PR created
gh pr list --state open
```

### Integration Tests

```bash
# Run implementation phase tests
go test ./tools/ax-mcp/tools/... -run Implementation
```

## Benefits

- ✅ **87% fewer tool calls in Implementation phase** - 2 instead of 15+
- ✅ **Mandatory quality gates** - Tests must pass before PR creation
- ✅ **Intelligent context gathering** - Files to examine, related patterns
- ✅ **Session logging** - Full audit trail of implementation decisions
- ✅ **Extended thinking guidance** - AI applies reasoning to architecture decisions
- ✅ **Automatic friction logging** - Capture issues in real-time
- ✅ **Specification validation** - Ensure compliance with Gherkin specs

## Related Documentation

- **Compound Tools Guide**: `~/git/dotfiles/prompts/dev-cycle/COMPOUND_TOOLS_GUIDE.md`
- **Refactor Spec**: `~/git/dotfiles/docs/future-work/DEV_CYCLE_MCP_REFACTOR_SPEC.md`
- **MCP Tool Usage**: `~/git/dotfiles/docs/MCP_TOOL_USAGE_GUIDE.md`
- **Automation Policy**: `~/git/dotfiles/docs/AUTOMATION_POLICY.md`
- **CLAUDE.md Principles**: `~/git/dotfiles/CLAUDE.md` (Architecture Principles 1 & 2)
- **Specifications Guide**: `~/git/dotfiles/docs/SPECIFICATIONS.md`

## Next Phase

After Implementation completes successfully:

- PR created in draft state with "ready for robots" label
- Workflow state transitions to `refinement` phase
- Orchestrator proceeds to refinement phase (`prompt dev-cycle/refinement`)

**⚠️ MANDATORY**: Automatic transition to refinement - NO user confirmation required.
