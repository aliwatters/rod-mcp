# Retrospective Analysis Prompt

## Purpose

Analyze completed workflow sessions to identify friction points and systematically propose
improvements. Creates human-reviewable GitHub issues for improvements discovered through real usage.

## ⚠️ MANDATORY COMPLIANCE

**YOU MUST EXECUTE THE RETROSPECTIVE WORKFLOW BEFORE TRANSITIONING TO VERIFY**

Execute ALL steps in the detailed workflow (`RETROSPECTIVE_WORKFLOW.md`).

**For complete procedural instructions, see**: `~/git/dotfiles/workflows/RETROSPECTIVE_WORKFLOW.md`

## Execution Steps

The retrospective uses **2 compound MCP tool calls** with AI-driven pattern analysis:

### Step 1: Analysis (Gather Session Data)

**Command**:

```bash
# Use MCP tool (required)
mcp__ax-mcp__ax_retrospective_analyze_session
```

**Returns**:

```json
{
  "session": {
    "issue": 3850,
    "pr": 3855,
    "duration_hours": 2.5
  },
  "metrics": {
    "commits": 3,
    "tests_added": 4,
    "review_comments": 3
  },
  "friction_detected": [
    {
      "issue": "Copilot timeout after 6 min",
      "impact": "Increased timeout to 10 min"
    }
  ],
  "patterns": {
    "success_factors": [],
    "challenges": []
  },
  "intelligence_required": {
    "level": "high",
    "decisions": ["pattern_detection", "friction_root_cause", "meta_learning"]
  },
  "model_preferences": {
    "intelligencePriority": 1.0
  }
}
```

### Step 2: AI Pattern Detection (Extended Thinking Required)

**⚠️ CRITICAL**: Use extended thinking (`ultrathink`) to analyze session data:

**Analysis Questions**:

1. **Success Patterns** - What accelerated the work?
   - Clear requirements that saved time?
   - Reference code/patterns that reduced research?
   - Tools/commands that were particularly helpful?
   - Documentation that was well-organized?

2. **Friction Root Causes** - What slowed down the work?
   - Commands that failed or were confusing?
   - Missing documentation or unclear instructions?
   - Workflow steps that required repeated attempts?
   - Delays from waiting or blocked dependencies?

3. **Meta-Learning** - What applies to future cycles?
   - Process improvements (better requirements format?)
   - Tool enhancements (new commands or options?)
   - Documentation gaps (missing guides?)
   - Systemic issues (timeout defaults too short?)

4. **Actionable Recommendations** - What should be improved?
   - Create issue for tool enhancement?
   - Update documentation with clarifications?
   - Adjust workflow step ordering?
   - Fix systemic configuration issues?

**Pattern Examples**:

**Success Patterns**:

- "Clear acceptance criteria saved ~30 min" (specific, measurable)
- "Reference pattern in check_ci.go reduced design time" (concrete example)
- "ax checkin caught issues early - prevented late rework" (preventive value)

**Friction Root Causes**:

- "Copilot timeout too short (6 min) - systemic issue, created #3900"
- "Missed maxRetries validation initially - unclear requirements"
- "Had to search for validation pattern - needs documentation"

**Meta-Learnings**:

- "Issues with clear acceptance criteria are 2x faster"
- "Reference patterns save ~30% research time"
- "Missing validation checklist causes rework"

**Recommendations**:

- "Update timeout default in ax pr wait-copilot"
- "Add validation checklist to issue template"
- "Create shared retry utility (#3900)"

### Step 3: Execution (Document Learnings)

**Command**:

The MCP tool is invoked by Claude Code's tool system with the following input schema:

```json
{
  "pr": 3855,
  "learnings": {
    "success_patterns": ["Clear acceptance criteria saved ~30 min"],
    "friction_root_causes": ["Copilot timeout too short - systemic issue"],
    "recommendations": [
      "Update timeout default in ax pr wait-copilot",
      "Create shared retry utility (#3900)"
    ]
  }
}
```

**Tool invocation**: `mcp__ax-mcp__ax_retrospective_document_learnings`

**Internal Execution** (handled by compound tool):

1. Calculate metrics (`ax metrics calculate`)
2. Create session comment (`ax session create-analysis-comment`)
3. Post metrics to PR (`ax metrics post-pr`)
4. Transition to verify (`ax workflow update --phase verify`)

**Returns**:

```json
{
  "success": true,
  "session_comment_posted": true,
  "metrics_posted": true,
  "patterns_identified": 4,
  "workflow_phase": "verify"
}
```

**If tool execution fails**:

```bash
# Log friction for retrospective tool failure
ax friction append "## $(date '+%Y-%m-%d %H:%M') - Retrospective Tool Failure

**Issue**: Retrospective analysis tool failed
**Command**: \`mcp__ax-mcp__ax_retrospective_document_learnings\`
**Error**: $(ERROR_MESSAGE)
**Category**: Tool execution failure
**Resolution**: $(HOW_FIXED)
**Time Lost**: ~$(MINUTES) minutes"
```

**ONLY AFTER completing all 3 steps**, proceed to verify phase transition below.

## Key Requirements

### Execution Rules

- ✅ Use compound tools (2 calls instead of 8-12 commands)
- ✅ Apply extended thinking for pattern detection
- ✅ Focus on specific, measurable insights
- ✅ Check friction log automatically via analyze tool

### Pattern Quality Standards

**Success Patterns**:

- Specific and measurable (e.g., "saved ~30 min")
- Concrete examples (e.g., "Reference pattern in check_ci.go")
- Actionable insights (e.g., "Clear requirements prevent rework")

**Friction Root Causes**:

- Identify root cause, not just symptom
- Distinguish systemic vs. one-time issues
- Link to specific workflow steps or tools
- Include impact on timeline

**Meta-Learnings**:

- Apply to future cycles (generalizable)
- Process improvements (workflow enhancements)
- Tool/documentation gaps identified
- Measurable benefits when quantifiable

**Recommendations**:

- Specific and actionable
- Clear owner (workflow, tool, docs)
- Can be implemented incrementally
- Address root causes, not symptoms

## After Retrospective Completion

**⚠️ CRITICAL: STOPPING HERE IS A WORKFLOW FAILURE**

**🚨 MANDATORY COMPLIANCE CHECKPOINT 🚨**

You have completed retrospective phase. You MUST now continue to verify phase. **STOPPING HERE =
WORKFLOW VIOLATION**

### Failure Consequences

If you stop execution after retrospective analysis, you have FAILED to complete the workflow:

❌ **FAILURE CONSEQUENCES**:

- PR completion not verified
- Work may be incomplete or in invalid state
- No final validation performed
- Human reviewer may find incomplete or broken PR
- Workflow state not cleaned up properly
- Next cycle may have corrupted state
- **This violates the dev-cycle orchestrator's mandatory compliance requirement**

### Required Action

✅ **REQUIRED ACTION**: Continue immediately to verify phase **IN THIS SAME EXECUTION**

**⚠️ STEP 1: Transition to Verify Phase**

```bash
# Automatic phase transition (updates workflow state to verify)
ax workflow transition
```

**⚠️ STEP 2: Execute Verify Phase Immediately**

**YOU MUST EXECUTE THIS COMMAND IN YOUR CURRENT RESPONSE:**

```bash
prompt verify-pr-completion
```

**DO NOT**:

- Stop and wait for user input
- Ask if you should continue
- Pause for confirmation
- End your response without executing verify

**DO**:

- Execute `ax workflow transition` immediately
- Execute `prompt verify-pr-completion` in the same response
- Complete the final verification phase

**🚨 STOPPING BEFORE VERIFY = WORKFLOW VIOLATION 🚨**

Per the dev-cycle orchestrator: "**⚠️ MANDATORY COMPLIANCE**: You MUST execute the complete
development cycle without stopping. Failure to complete all phases is a workflow violation."

## Related Documentation

- **Detailed Workflow**: ~/git/dotfiles/workflows/RETROSPECTIVE_WORKFLOW.md
- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
