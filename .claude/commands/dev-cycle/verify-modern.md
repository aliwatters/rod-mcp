# Verify PR Completion - Modern Compound Tools

**📦 Phase 3 Complete**: This phase uses MCP compound tools (issue #3926)

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

**🎯 PURPOSE**: Verify ALL work is complete before cleanup - failure to verify leads to incomplete PRs and wasted human review time.

**⚠️ COMPOUND TOOLS**: This prompt uses modern MCP compound tools to replace 15+ individual commands
with 2-3 efficient calls, providing 80%+ efficiency improvement.

## Purpose

This workflow provides MCP compound tools for the verification phase, ensuring comprehensive validation
and clean completion of the development cycle.

## Usage Pattern

**⚠️ PRIMARY APPROACH**: Use MCP compound tools for best performance:

**Step 1: Analysis** - Comprehensive verification

```typescript
mcp__ax - mcp__ax_verify_final_checks({ pr: PR_NUMBER });
```

**Step 2: AI Validation** - Review results and make decision (minimal intelligence needed - mostly deterministic)

**Step 3: Execution** - Complete and clean up

```typescript
mcp__ax - mcp__ax_verify_complete_cycle();
```

**CLI Fallback**: Only use `ax verify` commands if MCP tools unavailable.

## Step 0: Merge Conflict Failsafe Check (REQUIRED)

**⚠️ FAILSAFE**: Before running comprehensive verification, check for merge conflicts.
This catches conflicts that may have been introduced since the refinement phase.

```bash
# Check if PR is mergeable (exit 0 = yes, exit 1 = conflicts)
# Uses inline command substitution to avoid shell variable persistence issues
ax pr check-mergeable $(gh pr list --head "$(git branch --show-current)" --json number --jq '.[0].number')
```

**If conflicts exist** (exit code 1):

1. **Use ultrathink** to analyze the conflicts before resolving
2. Return to refinement-style conflict resolution (see refinement.md)
3. Re-run quality gates after resolution
4. Push changes and restart verify phase

```bash
# If conflicts detected, use ultrathink then resolve
# Uses inline command substitution for PR number
ax pr resolve-conflicts $(gh pr list --head "$(git branch --show-current)" --json number --jq '.[0].number') --strategy ours
ax checkin  # Verify build after resolution
git push origin $(git branch --show-current)
```

**If no conflicts** (exit code 0): Proceed to Step 1.

→ **THEN IMMEDIATELY**:

## Step 1: Analysis - Comprehensive Verification

**Run comprehensive verification** using MCP compound tool with PR number:

```typescript
// Replace 1234 with your actual PR number
mcp__ax - mcp__ax_verify_final_checks({ pr: 1234 });
```

**How to get PR number**: Check workflow state or use CLI:

```bash
# Get from workflow state
ax workflow get | grep PR

# Or get from current branch
gh pr list --head "$(git branch --show-current)" number --jq '.[0].number'
```

**CLI Fallback** (if MCP unavailable):

```bash
ax verify final-checks --pr $PR_NUMBER
```

**This tool validates**:

- ✅ PR state (not draft, has "ready for humans" label)
- ✅ Linked issue exists
- ✅ Merged with main (no conflicts)
- ✅ CI passing
- ✅ All review comments acknowledged
- ✅ Verification report posted

**Tool returns structured JSON**:

```json
{
  "pr": { "number": 1234, "draft": false },
  "verification": {
    "merged_with_main": true,
    "ci_passing": true,
    "all_comments_acknowledged": true,
    "has_ready_for_humans_label": true,
    "linked_issue_exists": true
  },
  "all_checks_passed": true,
  "verification_report_posted": false,
  "failure_reasons": [],
  "intelligence_required": { "level": "low" }
}
```

**Validation results**:

- `all_checks_passed: true` → Proceed to Step 3
- `all_checks_passed: false` → Review `failure_reasons` array, fix issues, re-run

## Step 2: AI Decision Making (Use Extended Thinking)

**⚠️ CRITICAL**: This is the LAST CHANCE to catch issues before marking PR ready for human review. Use extended thinking thoroughly.

ultrathink: Is this PR truly ready for human review, or am I missing critical issues?

This is the LAST CHANCE to catch problems before human review. Analyze thoroughly:

**Validation Results Review:**

- Which validation checks passed and which failed?
- For failed checks, what is the ROOT CAUSE (not just symptom)?
- Are there any warnings or edge cases flagged by the tools?
- Is the tool output consistent with manual inspection?

**PR State Verification:**

- Is PR marked ready (not draft) with correct labels ("ready for humans")?
- Are all review comments truly acknowledged with substantive responses?
- Is CI passing for the right reasons (not false positives)?
- Is the PR linked to the correct issue with "Fixes #N" in description?

**Code Quality Assessment:**

- Did I actually run and pass `ax checkin` (or am I assuming it would pass)?
- Are there any linting warnings I'm ignoring as "minor"?
- Did all tests pass locally before pushing?
- Are there code smells or technical debt I'm deferring without justification?

**Merge Readiness:**

- Is PR up to date with main branch (no stale divergence)?
- Are there any merge conflicts or rebase needed?
- Would merging this PR cause any breaking changes or regressions?

**Completeness Check:**

- Are ALL issue requirements and acceptance criteria met?
- Did I deliver what was promised or did scope creep expand deliverables?
- Are there any TODO comments or placeholder code that should be cleaned up?
- Is documentation updated if behavior changed?

**Risk Assessment:**

- What is the worst-case scenario if I mark this ready but missed something?
- How much human reviewer time would be wasted on incomplete/broken PR?
- Is there any uncertainty or doubt that should trigger a manual review before marking ready?
- Am I rushing to completion and cutting corners?

**Decision Gate:**

- Based on above analysis, is PR **objectively ready** or am I **prematurely completing**?
- If ANY doubt exists, what specific checks should I re-run?
- Should I mark ready (proceed) or fix issues first (block)?

**If validation passes** (`all_checks_passed: true`):

- All requirements met
- Ready to complete cycle
- Proceed to Step 3 (Execution)

**If validation fails** (`all_checks_passed: false`):

- Review the `failure_reasons` array from the tool output
- Identify what needs to be fixed
- Address blocking issues:
  - Draft PR: Mark as ready for review
  - No linked issue: Add "Fixes #ISSUE" to PR description
  - Merge conflicts: Merge main and resolve conflicts
  - CI failing: Fix failing tests/lint issues
  - Comments not acknowledged: Reply to all review comments
- Re-run Step 1 after fixes
- Do NOT proceed to Step 3 until validation passes

## Step 3: Execution - Complete and Clean Up

**ONLY run this after Step 1 validation passes**:

**Primary approach** (MCP compound tool):

```typescript
mcp__ax - mcp__ax_verify_complete_cycle();
```

**CLI Fallback** (if MCP unavailable):

```bash
ax verify complete-cycle
```

**This tool performs**:

- ✅ Calculate cycle metrics (duration, phases)
- ✅ Post completion metrics to PR comment
- ✅ Create session analysis for retrospective
- ✅ Release claim on issue/PR
- ✅ Transition workflow to cleanup phase
- ✅ Clean up temporary files

**Tool returns completion results**:

```json
{
  "success": true,
  "cycle_complete": true,
  "metrics": {
    "duration_seconds": 3420,
    "start_time": "2025-11-15T10:30:00Z",
    "end_time": "2025-11-15T11:27:00Z",
    "phase": "verify"
  },
  "issue": 3926,
  "pr": 1234,
  "verification_errors": [],
  "details": {
    "metrics_posted": true,
    "session_logged": true,
    "claim_released": true,
    "workflow_cleaned": true
  }
}
```

**Completion status**:

- `cycle_complete: true` → Development cycle finished successfully
- `cycle_complete: false` → Review `verification_errors` array, fix issues

## MCP Tool Invocation Notes

**Primary**: Use MCP compound tools for best performance and integration
**Fallback**: CLI commands available if MCP unavailable

**MCP tools always return JSON** - no need for `--json` flag

**Dry run mode**: Not available for MCP tools (Step 1 is read-only analysis anyway)

**CLI equivalents**:

- MCP: `mcp__ax-mcp__ax_verify_final_checks({"pr": 1234})`
- CLI: `ax verify final-checks --pr 1234`

- MCP: `mcp__ax-mcp__ax_verify_complete_cycle()`
- CLI: `ax verify complete-cycle`

## Error Handling

**If Step 1 (final-checks) returns `all_checks_passed: false`**:

1. Review the `failure_reasons` array in the JSON response
2. Fix each identified problem (see Step 2 guidance)
3. Re-run `mcp__ax-mcp__ax_verify_final_checks({"pr": PR_NUMBER})`
4. Do NOT proceed to Step 3 until `all_checks_passed: true`

**If Step 3 (complete-cycle) returns `cycle_complete: false`**:

1. Review the `verification_errors` array in the JSON response
2. The `details` object shows which operations succeeded/failed
3. Fix any issues and re-run the tool

**MCP Tool Errors**:

- Tool unavailable → Use CLI fallback commands
- Network/API errors → Check GitHub connectivity, retry
- Workspace errors → Verify you're in correct workspace directory

## Quality Gates Enforcement

**CRITICAL**: The verification tools enforce quality gates and will BLOCK completion if:

- Any dev-cycle phases are incomplete
- CI checks are failing
- Code quality issues exist (lint, tests, types)
- Documentation is incomplete
- PR is not ready for human review

**This prevents**:

- Incomplete PRs reaching human reviewers
- Quality regressions in the codebase
- Skipped workflow phases
- Inconsistent development cycles

## Efficiency Improvements

**Old approach**: 15+ individual commands

```bash
ax workflow validate-phase-sequence
ax workflow validate
ax pr verify $PR_NUMBER
ax pr ready $PR_NUMBER
ax log-session create
# + 10+ manual claim cleanup commands
# + multiple friction logging commands
# + multiple validation commands
```

**New approach**: 2-3 compound calls

```bash
ax verify final-checks
ax verify complete-cycle
```

**Benefits**:

- 80%+ reduction in command count
- Atomic operations (all-or-nothing)
- Comprehensive validation
- Structured error reporting
- JSON output for automation
- Dry run testing capability

## Agent Boundaries

**🟢 GREEN (Fully Automated)**:

- Reading validation results
- Running verification checks
- Generating completion reports

**🟡 YELLOW (Automated with Confirmation)**:

- Adding verification labels
- Posting completion metrics
- Cleaning up claim comments

**🔴 RED (Human Only)**:

- Merging PRs
- Making final approval decisions
- Overriding blocking quality gates

## Next Phase

After verification completes successfully, the workflow automatically transitions to the next phase as defined in `cycle.json`.

**Automatic Transition**:

```bash
# The ax verify complete-cycle command will:
# 1. Mark PR as verified
# 2. Post metrics and create session log
# 3. Clean up claim comments
# 4. Call: ax workflow transition (updates PHASE to next from cycle.json)

# Then the orchestrator proceeds to next phase:
ax cycle run dev-cycle
```

**MANDATORY**: Automatic transition to next phase - NO user confirmation required.

**Note**: The next phase is determined by the workflow configuration in `cycle.json`, not hardcoded in this prompt. This allows the workflow structure to evolve without updating individual phase prompts.

---

## 🔒 PHASE COMPLETION - BLOCKING CHECKPOINT

**STOP**: Before proceeding, verify ALL of the following are complete:

### Verify Phase Completion Criteria

- [ ] PR marked ready for review (not draft)
- [ ] All quality gates passed
- [ ] CI passing
- [ ] All review comments acknowledged
- [ ] PR labeled correctly (ready-for-humans, etc.)
- [ ] Verification checks all passed
- [ ] No blocking issues remaining

### Phase Advancement

**ONLY AFTER ALL ABOVE ARE ✅ COMPLETE**:

```bash
# Signal phase completion to orchestrator (Issue #4957)
ax workflow update --field "PHASE_STATUS=completed"

# Invoke orchestrator to validate and advance to next phase
ax cycle run dev-cycle
```

**CRITICAL ENFORCEMENT**:

- Agent sets `PHASE_STATUS=completed` to signal phase is done
- Agent calls `ax cycle run dev-cycle` (or MCP equivalent) to advance
- Orchestrator validates completion, advances `PHASE` to cleanup automatically
- NO manual `ax workflow update --phase` commands needed

**This is orchestrator-driven flow.** The cycle orchestrator handles all phase transitions
automatically when `PHASE_STATUS=completed` is set.

**VERIFICATION**: Run `ax workflow get` and confirm `PHASE=cleanup` after orchestrator advances.

---

## Related Documentation

- `prompts/dev-cycle/cycle.json` - Workflow configuration and phase transitions
- `workflows/PR_COMPLETION_VERIFICATION_WORKFLOW.md` - Legacy individual commands workflow
- `docs/COMPOUND_TOOLS_ARCHITECTURE.md` - Architecture principles for compound tools
- `tools/ax/cmd/verify/` - Implementation source code
