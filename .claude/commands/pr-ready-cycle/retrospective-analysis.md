# PR Ready Retrospective Analysis

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

**Purpose**: Perform comprehensive retrospective analysis and generate improvement recommendations.

## Run Retrospective Analysis

```bash
# Run comprehensive retrospective analysis
ax workflow retrospective \
    --session-type "pr-ready-cycle" \
    --friction-log "$FRICTION_LOG" \
    --max-improvements 5 \
    --output tmp/retrospective-pr-ready-session.md

# Check if analysis completed successfully
if [ $? -eq 0 ]; then
    echo "✅ Retrospective analysis completed"
else
    echo "⚠️  Retrospective analysis failed, continuing with basic summary"
fi
```

## Retrospective Analysis Framework

The retrospective analysis examines:

### 1. Context Review

- Session type: pr-ready-cycle
- PRs processed count
- Total session duration
- Workspace and environment

### 2. Label Success Verification

- PRs that reached "verified" state
- PRs stuck in intermediate states
- PRs with no ready labels (workflow failures)
- Root cause analysis for label failures

### 3. Friction Analysis

- Failed commands and error patterns
- Repeated attempts and delays
- Confusing error messages
- Missing documentation or unclear instructions

### 4. Success Patterns

- What worked well during preparation
- Effective conflict resolution strategies
- Successful auto-fix patterns
- Efficient verification processes

### 5. Complexity Debt Detection

- PRs with recurring issues
- Patterns of manual intervention needed
- Common failure modes
- Areas requiring human review

### 6. Command Sequence Consolidation

- Repeated command patterns
- Opportunities for script creation
- Manual processes that could be automated
- Inefficient workflows

### 7. Improvement Generation

- Create helper scripts for common patterns
- Add pre-merge checks for frequent issues
- Improve error messages and documentation
- Enhance automation capabilities

## Expected Output

### Successful Session

```text
✅ Retrospective analysis completed

# 📊 Retrospective Summary:

## Session Analysis

**PRs Processed**: 3 **Success Rate**: 100% **Average Time per PR**: 15 minutes

## Success Patterns

- Auto-conflict resolution worked well
- Lint fixes were comprehensive
- Verification gate caught all issues

## Improvement Opportunities

- Create helper script for common conflict patterns
- Add pre-merge check for lint issues
- Enhance error messages for test failures

## Generated Issues

- Issue #1622: Create conflict resolution helper script
- Issue #1623: Add pre-merge lint check
```

### Session with Issues

```text
⚠️  Retrospective analysis failed, continuing with basic summary

📊 Basic Session Summary:
   PRs processed: 2
   Success rate: 50%
   Average time per PR: 15 minutes
```

## Troubleshooting

### Retrospective Analysis Fails

**Cause**: `run_retrospective.py` script not found or error

**Solution**:

```bash
# Check if script exists
ls -la ~/git/dotfiles/tools/workflow/run_retrospective.py

# If missing, create basic summary
echo "📊 Basic Session Summary:"
echo "   PRs processed: $PR_COUNT"
echo "   Complex issues: $COMPLEX_ISSUES"
```

### State Clear Fails

**Cause**: State file locked or corrupted

**Solution**:

```bash
# Force clear state
rm -f .pr-ready-state
echo "✅ State cleared manually"
```

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
