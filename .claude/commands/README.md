# Prompts Directory

This directory contains AI prompts for various development workflows and tasks.

## ⚠️ Specifications

Some prompts and workflows have **Gherkin specification files** (`.feature`) that define
authoritative behavior.

**Before modifying prompts**: Check if specification exists and read it first.

````bash
ls specifications/*-gherkin-specification.md workflows/*_SPECIFICATION.feature
```bash

**See**: `docs/SPECIFICATIONS.md` for details

---

## Available Prompts

### Development Workflows

#### Core Dev-Cycle (Stateful Workflow)

- **[dev-cycle.md](dev-cycle.md)** ⭐ - **Development cycle orchestrator** - Routes between phases
  based on workflow state
  - **Workflow**: [DEV_CYCLE_WORKFLOW.md](~/git/dotfiles/workflows/DEV_CYCLE_WORKFLOW.md) - Complete
    procedural guide
  - **Specification**: [dev-cycle-gherkin-specification.md](~/git/dotfiles/specifications/dev-cycle-gherkin-specification.md) -
    Complete Gherkin specification defining all phases, edge cases, quality gates, and error
    handling
- **[discovery.md](discovery.md)** - Phase 1: Analyze issue, understand requirements, plan approach
- **pre-flight.md** 🚧 PLANNED - Phase 2: Validate approach before implementation (catch issues
  early)
- **[IMPLEMENTATION_WORKFLOW.md](~/git/dotfiles/workflows/IMPLEMENTATION_WORKFLOW.md)** - Phase 3: Write code, tests, documentation
- **review-loop.md** 🚧 PLANNED - Phase 4: Multi-persona AI review with iterative refinement (max
  3x)
- **[retrospective.md](retrospective.md)** - Phase 5: Extract learnings, identify friction patterns
- **[verify-pr-completion.md](verify-pr-completion.md)** - Phase 6: Verify PR is ready for human
  review

**Current Flow**:

````

setup → discovery → implementation → refinement → retrospective → verify ↓ ↓ ↓ ↓ ↓ [find work]
[build it] [get reviews] [learn] [complete]

`````text

**Planned Enhancements** (🚧):

```bash

setup → discovery → pre-flight → implementation → review-loop → retrospective → verify ↓ ↓ ↓ ↓ ↓ ↓
[find work] [validate] [build it] [iterate 3x] [learn] [verify ready]

````bash

**Key Features**:

- **Stateful**: Persists phase across Cursor sessions via `.workflow-state`
- **Quality loops**: Pre-flight validation + review iterations improve output
- **Self-improving**: Retrospective identifies friction for continuous improvement
- **Verification**: Systematic check before human review

#### Standalone Workflows

- **[deep-review.md](deep-review.md)** - Deep code review (comprehensive security, persona-based, or standard)
- **[refinement.md](refinement.md)** - Issue refinement workflow (standalone)

### Issue Management

- **[get-next-work-item.md](get-next-work-item.md)** - Get next prioritized work item
- **[resolve-work-item.md](resolve-work-item.md)** ⭐ - **Consolidated work item resolution** -
  Complete workflow for simple and project-based issues
- **[quick-start-work.md](quick-start-work.md)** - Quick start on work items
- **[validate-old-issues.md](validate-old-issues.md)** ⭐ **NEW** - Validate and triage issues
  (oldest to newest)
- **[investigate-e2e-ci-failures.md](investigate-e2e-ci-failures.md)** ⭐ **NEW** - Investigate and
  fix E2E test failures in CI/CD

### Pull Request Workflows

- **[deep-review.md](deep-review.md)** - Deep code review (comprehensive, persona-based, or standard)
- **[address-comments.md](address-comments.md)** - Address PR review comments with threaded replies
- **[resolve-conflicts.md](resolve-conflicts.md)** - Resolve PR merge conflicts
- **[verify-pr-completion.md](verify-pr-completion.md)** - Verify PR completeness
- **[assess-pr-cleanliness.md](assess-pr-cleanliness.md)** - Assess PR quality

### Project Management

- **[create-github-project.md](create-github-project.md)** - Create GitHub project
- **[create-project-epic.md](create-project-epic.md)** - Create project epics
- **[create-project-tasks.md](create-project-tasks.md)** - Create project tasks
- **[move-issues-to-epic.md](move-issues-to-epic.md)** - Organize issues in epics
- **[update-project-status.md](update-project-status.md)** - Update project status
- **[project-cycle.md](project-cycle.md)** - Complete project cycle
- **[groom-backlog.md](groom-backlog.md)** - Backlog grooming

### Discovery & Analysis

- **[discovery.md](discovery.md)** - Codebase discovery
- **[friction-analysis.md](friction-analysis.md)** - Friction pattern analysis and retrospectives
- **[deep-review.md](deep-review.md)** - Deep code review (comprehensive, persona-based, or standard)
- **[generate-next-ideas.md](generate-next-ideas.md)** ⭐ **NEW** - Generate ambitious next steps
  for project advancement
- **[log-session.md](log-session.md)** ⭐ **NEW** - Log AI agent session for deep analysis in another
  agent

### AI Personas

See [personas/](personas/) directory for specialized AI review personas:

- Architect - Pattern consistency
- Simplifier - Reduce complexity
- Tester - Test coverage
- Medusa v2 Specialist - Medusa-specific guidance
- Next.js Specialist - Next.js best practices

---

## Quick Start

### Validate Old Issues

**Purpose**: Systematically review and validate old repository issues.

**Quick Start**:

```bash
# Get the oldest unvalidated issue
~/git/dotfiles/tools/gh-helpers/get-next-issue-to-validate.sh

# Interactively validate an issue
~/git/dotfiles/tools/gh-helpers/validate-issue-interactive.sh ISSUE_NUMBER

# Generate batch validation report
~/git/dotfiles/tools/gh-helpers/batch-validate-issues.py --count 10
`````

**See Also**:

- [Workflow: VALIDATE_OLD_ISSUES.md](~/git/dotfiles/workflows/VALIDATE_OLD_ISSUES.md)
- [Labels: STANDARD_LABELS.md](~/git/dotfiles/docs/STANDARD_LABELS.md)

---

## Usage Pattern

1. **Choose a prompt** based on your current task
2. **Run the prompt command**: `prompt <name>` (copies to clipboard)
3. **Follow the steps** outlined in the prompt
4. **Execute workflows when needed**: `prompt workflow "<NAME>.md"` (prints to stdout)
5. **Use helper scripts** when available
6. **Document outcomes** as specified

**Note**: Prompts provide instructions (~100 lines), workflows provide full procedural details
(~5000 lines max).

---

## Creating New Prompts

When creating new prompts:

1. **Clear objective** - State the purpose upfront
2. **Step-by-step process** - Break down the workflow
3. **Examples** - Show good and bad patterns
4. **Quality checks** - Include validation criteria
5. **Related docs** - Link to relevant documentation

---

## Related Documentation

- [Workflows](../workflows/) - Step-by-step workflow guides
- [Tools](../tools/gh-helpers/) - Helper scripts and automation
- [Docs](../docs/) - Reference documentation
