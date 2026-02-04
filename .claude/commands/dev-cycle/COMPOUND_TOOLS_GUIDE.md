# Compound Tools Usage Guide for Dev-Cycle

**Purpose**: Reference guide for using compound MCP tools in dev-cycle workflow phases.

**Target**: Phase 3 implementation (issues #3915-#3919) - provides patterns for all phases to follow.

## Overview

Dev-cycle phases use **compound tools** - pairs of analysis and execution tools that delegate intelligent decisions to AI agents while maintaining structured automation.

**Pattern**: Analysis → AI Reasoning → Execution

```text
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Analysis   │ →   │ AI Reasoning │  →  │  Execution  │
│    Tool     │     │  (extended   │     │    Tool     │
│             │     │   thinking)  │     │             │
└─────────────┘     └──────────────┘     └─────────────┘
      ↓                    ↓                     ↓
  Gathers data       Makes decisions       Executes & logs
  + hints           (duplicate check,      (claim, commit,
                     scope validation)      label manage)
```

## Core Principles

### Principle 1: Client-Side AI Decisions (Default)

**Use for**: Discovery, Implementation, Refinement, Retrospective, Verify

**Pattern**:

```text
AI Agent → Analysis Tool (gathers data) → Returns structured data + intelligence hints
          ↓
AI Agent applies reasoning (extended thinking for Claude, deep reasoning for others)
          ↓
AI Agent → Execution Tool (with AI's decision) → Executes and logs
```

**Benefits**:

- AI agent maintains control over decisions
- Extended thinking can be applied between steps
- Clear audit trail of reasoning
- Testable decision points

### Principle 2: Compound Over Individual

**Before (60-80+ commands)**:

```bash
ax workflow get
ax workflow validate-workspace
ax git check-branch
ax workflow clear  # if stale
# ... many more
```

**After (2 compound tool calls)**:

```bash
# 1. Analysis
result = ax_setup_analyze_state(pre_instruction)

# 2. AI reasoning applied here (with extended thinking)

# 3. Execution with AI decision
ax_setup_execute_transition(
  action=result.recommendation.action,
  preserve_state=true
)
```

**Reduction**: ~85% fewer tool invocations per phase

### Principle 3: Intelligence Hints Guide AI

Analysis tools return `intelligence_required` fields to guide AI decision-making:

```json
{
  "intelligence_required": {
    "level": "high", // high, medium, low
    "decisions": ["duplicate_detection", "scope_validation"],
    "prompt_hint": "Analyze scope overlap, subsumes relationships"
  },
  "model_preferences": {
    "intelligencePriority": 1.0, // Triggers extended thinking
    "costPriority": 0.0,
    "speedPriority": 0.0
  }
}
```

**How to use**:

- `level: "high"` → Use extended thinking
- `decisions` → Focus areas for AI analysis
- `prompt_hint` → Specific guidance for reasoning

## Tool Calling Pattern Template

### Step 1: Call Analysis Tool

**MCP Tool (preferred)**:

```text
mcp__ax-mcp__ax_<phase>_analyze_<aspect>
```

**CLI Fallback**:

```bash
ax <phase> analyze-<aspect>
```

**Example (Discovery)**:

```text
# MCP Tool (preferred)
mcp__ax-mcp__ax_discover_analyze_issue({"issue_number": 3850})

# CLI Fallback
# Note: 'ax discover analyze-issue' doesn't exist
# This is an MCP-only compound tool - no CLI equivalent
# For simple discovery, use 'ax discover issue' or 'ax issue next'
```

### Step 2: Apply AI Reasoning

When `intelligence_required.level` is "high", use extended thinking:

```markdown
Using extended thinking: Analyze duplicate detection

**Duplicate Analysis**:

- Issue #3850: Add retry logic to wait-copilot
- Candidate #3201: Improve timeout handling

**Scope Comparison**:

- #3201: Focuses on timeout configuration (global setting)
- #3850: Focuses on retry logic (per-operation behavior)

**Subsumes Relationship**: None - distinct scopes

- Timeout = how long to wait before giving up
- Retry = how many attempts with exponential backoff

**Decision**: NOT a duplicate - complementary features

**Reasoning**: #3201 addresses "how long to wait total" while #3850 addresses "how to handle transient failures". Both can coexist.
```

### Step 3: Call Execution Tool

**MCP Tool (preferred)**:

```text
mcp__ax-mcp__ax_<phase>_<action>
```

**CLI Fallback**:

```bash
ax <phase> <action>
```

**Example (Discovery)**:

```text
# MCP Tool
mcp__ax-mcp__ax_discover_claim_and_setup({
  "issue_number": 3850,
  "branch_prefix": "fix",
  "duplicate_decision": {
    "is_duplicate": false,
    "reasoning": "Complementary features - retry vs timeout"
  },
  "scope_decision": {
    "in_scope": true,
    "reasoning": "Agent-first alignment - improves automation"
  }
})

# CLI Fallback
# Note: 'ax discover claim-and-setup' doesn't exist
# This is an MCP-only compound tool - no CLI equivalent
# Use multi-step CLI commands instead:
# ax issue claim 3850
# ax git create-branch --issue 3850 --prefix fix
# ax workflow update --phase discovery --issue 3850
```

## Phase-Specific Examples

### Setup Phase (Phase 0)

**Tools**:

1. `ax_setup_analyze_state` - Detect scenario (fresh, resume, stale, etc.)
2. `ax_setup_execute_transition` - Execute cleanup/initialization

**Intelligence Required**: Low (mostly deterministic)

**Example**:

```bash
# 1. Analysis (MCP or CLI)
# MCP: mcp__ax-mcp__ax_setup_analyze_state({})
# CLI: ax workflow intelligent-setup

# Output provides recommendation
# {
#   "scenario": "resume_on_pr",
#   "recommendation": {
#     "action": "resume_at_refinement",
#     "skip_to_phase": "refinement"
#   }
# }

# 2. No AI reasoning needed (low intelligence)

# 3. Execution (MCP only)
# MCP: mcp__ax-mcp__ax_setup_execute_transition({
#   "action": "resume_at_refinement",
#   "preserve_state": true
# })
# CLI: No direct equivalent - use manual workflow commands
```

**Pattern Note**: Setup is mostly deterministic - AI reasoning is minimal.

### Discovery Phase (Phase 1)

**Tools**:

1. `ax_discover_analyze_issue` - Deep analysis with duplicate detection
2. `ax_discover_claim_and_setup` - Atomic claim + branch + state

**Intelligence Required**: High (duplicate detection, scope validation)

**Example**:

```bash
# 1. Analysis (MCP only)
# MCP: mcp__ax-mcp__ax_discover_analyze_issue({"issue_number": 3850})
# CLI: No direct equivalent - use 'ax issue next' or 'ax discover issue'

# Output includes duplicate candidates and intelligence hints
# {
#   "duplicate_analysis": {...},
#   "intelligence_required": {
#     "level": "high",
#     "decisions": ["duplicate_detection", "scope_validation"]
#   }
# }

# 2. AI Reasoning (extended thinking)
# - Analyze duplicate candidates
# - Validate scope alignment
# - Assess architecture fit

# 3. Execution (MCP only)
# MCP: mcp__ax-mcp__ax_discover_claim_and_setup({
#   "issue_number": 3850,
#   "branch_prefix": "fix",
#   "duplicate_decision": {"is_duplicate": false, "reasoning": "Unique scope"},
#   "scope_decision": {"in_scope": true, "reasoning": "Agent-first alignment"}
# })
# CLI: Use multi-step commands (ax issue claim, ax git create-branch, etc.)
```

### Implementation Phase (Phase 2)

**Tools**:

1. `ax_implement_prepare` - Validate readiness, gather context
2. `ax_implement_commit_and_pr` - Quality gates + commit + PR

**Intelligence Required**: High (architecture design, edge cases)

**Example**:

```bash
# 1. Analysis (CLI)
ax implement prepare

# Output provides context and validation
# {
#   "context": {
#     "files_to_examine": ["wait_copilot.go"],
#     "related_patterns": ["check_ci.go (retry example)"]
#   },
#   "intelligence_required": {
#     "level": "high",
#     "decisions": ["architecture_design", "edge_case_analysis"]
#   }
# }

# 2. AI Reasoning (extended thinking)
# - Design retry strategy
# - Identify edge cases
# - Plan test coverage

# 3. After implementing code changes...
# Execution (CLI)
ax implement commit-and-pr \
  --message "fix: add exponential backoff retry" \
  --pr-title "Add retry logic to wait-copilot" \
  --pr-body "$(cat pr_body.md)"
```

**CRITICAL**: `ax implement commit-and-pr` runs quality gates (BLOCKING) before commit.

### Refinement Phase (Phase 3)

**Tools**:

1. `ax_refine_analyze_feedback` - Gather Copilot feedback, classify comments
2. `ax_refine_apply_responses` - Apply AI decisions to each comment

**Intelligence Required**: High (feedback analysis, scope creep detection)

**Example**:

```bash
# 1. Analysis (CLI)
ax refine analyze-feedback 3855

# Output classifies comments and suggests responses
# {
#   "review_comments": {
#     "comments": [
#       {
#         "id": 12345,
#         "classification": "potential_bug",
#         "suggested_response": "fix"
#       },
#       {
#         "id": 12346,
#         "classification": "refactoring",
#         "suggested_response": "defer"  // Scope creep
#       }
#     ]
#   },
#   "intelligence_required": {
#     "level": "high",
#     "decisions": ["review_feedback_analysis", "scope_creep_detection"]
#   }
# }

# 2. AI Reasoning (extended thinking)
# - Understand reviewer's underlying concerns
# - Detect scope creep (refactoring = separate issue)
# - Decide: fix vs defer vs acknowledge

# 3. Execution (CLI)
ax refine apply-responses 3855 \
  --comment 12345 --action fix --changes "Added validation" --commit abc123 \
  --comment 12346 --action defer --issue 3900
```

**CRITICAL**: `ax refine apply-responses` runs quality gates (BLOCKING) if changes made.

### Retrospective Phase (Phase 4)

**Tools**:

1. `ax_retrospective_analyze_session` - Deep session analysis
2. `ax_retrospective_document_learnings` - Create comment + metrics

**Intelligence Required**: High (pattern detection, root cause analysis)

**Example**:

```bash
# 1. Analysis (MCP or CLI)
# MCP: mcp__ax-mcp__ax_retrospective_analyze_session()
# CLI: ax workflow retrospective (monolithic, not just analysis)

# Output detects patterns and friction
# {
#   "friction_detected": [
#     {"issue": "Copilot timeout", "impact": "6 min wait"}
#   ],
#   "patterns": {
#     "success_factors": ["Clear requirements saved ~30 min"]
#   },
#   "intelligence_required": {
#     "level": "high",
#     "decisions": ["pattern_detection", "friction_root_cause"]
#   }
# }

# 2. AI Reasoning (extended thinking)
# - Identify success patterns (what enabled smooth execution?)
# - Analyze friction root causes (systemic issues)
# - Extract meta-learnings (generalizable lessons)

# 3. Execution (MCP only)
# MCP: mcp__ax-mcp__ax_retrospective_document_learnings({
#   "pr": 3855,
#   "learnings": [{"pattern": "Clear acceptance criteria saved ~30 min"}],
#   "friction": [{"issue": "Copilot timeout", "recommendation": "Create #3900"}]
# })
# CLI: No direct equivalent - manual issue creation and commenting
```

### Verify Phase (Phase 5)

**Tools**:

1. `ax_verify_final_checks` - Comprehensive readiness verification
2. `ax_verify_complete_cycle` - Clean up, return to main

**Intelligence Required**: Low (mostly deterministic)

**Example**:

```bash
# 1. Analysis (CLI)
ax verify final-checks 3855

# Output verifies all gates passed
# {
#   "verification": {
#     "ci_passing": true,
#     "all_comments_acknowledged": true,
#     "has_ready_for_humans_label": true
#   },
#   "all_checks_passed": true
# }

# 2. No AI reasoning needed (low intelligence)

# 3. Execution (CLI)
ax verify complete-cycle
```

## Session Logging Pattern

**All execution tools log AI decisions** for audit trail and retrospective analysis:

```bash
# Automatic session logging (done by execution tools)
ax session append --data "Phase: Discovery
Issue: #3850
Decision: NOT duplicate
AI Reasoning: Unique scope - retry logic vs timeout config
Scope: IN SCOPE
AI Reasoning: Agent-first alignment"
```

**Benefits**:

1. Full audit trail of AI decisions
2. Pattern learning for future cycles
3. Debugging when decisions are wrong
4. Retrospective analysis data

## Intelligence Hints Usage

### When Analysis Tool Returns High Intelligence

```json
{
  "intelligence_required": {
    "level": "high",
    "decisions": ["duplicate_detection"],
    "prompt_hint": "Analyze scope overlap"
  }
}
```

**Action**: Use extended thinking

### When Analysis Tool Returns Low Intelligence

```json
{
  "intelligence_required": {
    "level": "low",
    "decisions": []
  }
}
```

**Action**: Skip extended thinking, proceed directly to execution

## Error Handling Pattern

**Analysis tool failures**: Return structured errors, AI decides next action

```json
{
  "error": {
    "type": "validation_failed",
    "message": "Issue #3850 is already closed",
    "recommendation": "Return to discovery, select new issue"
  }
}
```

**Execution tool failures**: Return structured errors with remediation

```json
{
  "success": false,
  "error": {
    "type": "quality_gates_failed",
    "message": "Tests failed: 3 failures in wait_copilot_test.go",
    "remediation": "Fix failing tests before proceeding"
  }
}
```

## Benefits Summary

### Performance

- **85% reduction** in tool invocations (60-80+ → 12)
- **50-80% faster** execution (parallel internal operations)
- **40%+ reduction** in token usage (structured JSON vs text parsing)

### Quality

- **Mandatory quality gates** - Cannot bypass (built into execution tools)
- **Session logging** - Full audit trail of AI decisions
- **Intelligence hints** - Guides AI to use extended thinking appropriately
- **Clear decision points** - Explicit AI reasoning between analysis and execution

### Maintainability

- **Clear separation** - Analysis gathers data, AI reasons, execution acts
- **Testable** - Mock analysis tools to test AI decision logic
- **Composable** - Each phase uses same pattern (easy to add new phases)
- **Cross-LLM compatible** - Intelligence hints work with Claude, GPT-4, Gemini

## Migration from Individual Commands

**Before** (individual commands in workflow):

```bash
ax workflow get
ax workflow validate-workspace
ax git check-branch
ax workflow clear
ax workflow init --phase discovery
# ... many more
```

**After** (compound tools):

```bash
# 1. Analysis (MCP or CLI)
# MCP: mcp__ax-mcp__ax_setup_analyze_state({})
# CLI: ax workflow intelligent-setup

# 2. AI reasoning (if intelligence_required.level == "high")

# 3. Execution (MCP only)
# MCP: mcp__ax-mcp__ax_setup_execute_transition({"action": $ACTION})
# CLI: No direct equivalent - use manual workflow commands
```

**Steps for each phase**:

1. Identify all individual `ax` commands in phase
2. Group into analysis vs execution operations
3. Replace with compound tool calls
4. Add AI reasoning between analysis and execution (if high intelligence)
5. Test end-to-end with all scenarios

## Testing Compound Tools

### Unit Tests

```bash
# Test analysis tool
pytest tools/ax-mcp/tests/test_setup_analyze.py

# Test execution tool
pytest tools/ax-mcp/tests/test_setup_execute.py
```

### Integration Tests

```bash
# Test full Setup phase with compound tools
pytest tools/ax-mcp/tests/test_setup_integration.py
```

### End-to-End Tests

```bash
# Test complete dev-cycle with compound tools
pytest specifications/dev-cycle-gherkin-specification.feature
```

## Related Documentation

- **Specification**: `~/git/dotfiles/docs/future-work/DEV_CYCLE_MCP_REFACTOR_SPEC.md`
- **MCP Tool Usage**: `~/git/dotfiles/docs/MCP_TOOL_USAGE_GUIDE.md`
- **Agent Boundaries**: `~/git/dotfiles/docs/AGENT_BOUNDARIES.md`
- **Architecture**: `~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md`

## For Other Agents (Issues #3915-#3919)

**After this guide is merged**, follow this pattern for your phase:

1. Read this guide completely
2. Identify your phase's analysis vs execution operations
3. Use the Tool Calling Pattern Template (Step 1-3)
4. Add AI reasoning between steps (if high intelligence)
5. Update phase-specific prompt to use compound tools
6. Test with all scenarios for your phase
7. Document any phase-specific nuances in this guide

**Questions?** Reference the phase-specific examples above or ask in issue comments.
