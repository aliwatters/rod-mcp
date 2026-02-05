# Create Project Epic

**Execution Mode**: 🤖 **Autonomous AI Agent** **Purpose**: Following the Epic Management Workflow
from ~/git/dotfiles/workflows/EPIC_MANAGEMENT_WORKFLOW.md

## For AI Agents

When invoked, you should:

- ✅ Execute all steps automatically
- ✅ Make decisions based on the epic management requirements
- ✅ Create and configure the epic
- ✅ Break down tasks and set dependencies
- ✅ Complete the epic setup from definition to validation

Do NOT just provide instructions - actually perform the work.

**Note**: This prompt is designed for autonomous AI agent execution, not manual human workflow
steps. For reference documentation, see `~/git/dotfiles/workflows/EPIC_MANAGEMENT_WORKFLOW.md`.

---

1. I will check for repo-specific overrides first - look for repository-specific workflows:
   - Check for {current_repo}/workflows/ directory
   - Check for ~/git/dotfiles/workflows/EPIC_MANAGEMENT_WORKFLOW.md
   - Check for ~/git/dotfiles/docs/DEVELOPMENT.md or CONTRIBUTING.md
   - Use repo-specific overrides if found, otherwise use default workflow

2. I will read the docs - understand epic structure and task breakdown process:
   - If repo-specific: Use current repo's workflow files
   - If not found: Use ~/git/dotfiles/workflows/EPIC_MANAGEMENT_WORKFLOW.md

3. I will define epic scope, objectives, and success criteria
4. I will create epic-level project item with comprehensive description
5. I will break down epic into logical task groups and individual tasks
6. I will estimate effort and complexity for each task
7. I will define task dependencies and work order
8. I will assign tasks to appropriate team members
9. I will set epic timeline and milestone dates
10. I will create task items in project with proper field values
11. I will link tasks to epic using project relationships
12. I will validate epic structure and task completeness
13. I will update project roadmap and sprint planning

MANDATORY: Always check for repository-specific overrides first. Ensure epic has clear success
criteria and all tasks are properly linked. Validate work order and dependencies before marking
complete.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
