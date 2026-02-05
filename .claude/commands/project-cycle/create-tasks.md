# Create Project Tasks

**Execution Mode**: 🤖 **Autonomous AI Agent** **Purpose**: Following the Task Management Workflow
from ~/git/dotfiles/workflows/TASK_MANAGEMENT_WORKFLOW.md

## For AI Agents

When invoked, you should:

- ✅ Execute all steps automatically
- ✅ Make decisions based on the task management requirements
- ✅ Create and configure the tasks
- ✅ Set up assignments and dependencies
- ✅ Complete the task setup from definition to validation

Do NOT just provide instructions - actually perform the work.

**Note**: This prompt is designed for autonomous AI agent execution, not manual human workflow
steps. For reference documentation, see `~/git/dotfiles/workflows/TASK_MANAGEMENT_WORKFLOW.md`.

---

1. I will check for repo-specific overrides first - look for repository-specific workflows:
   - Check for {current_repo}/workflows/ directory
   - Check for ~/git/dotfiles/workflows/TASK_MANAGEMENT_WORKFLOW.md
   - Check for ~/git/dotfiles/docs/DEVELOPMENT.md or CONTRIBUTING.md
   - Use repo-specific overrides if found, otherwise use default workflow

2. I will read the docs - understand task structure and field requirements:
   - If repo-specific: Use current repo's workflow files
   - If not found: Use ~/git/dotfiles/workflows/TASK_MANAGEMENT_WORKFLOW.md

3. I will define task scope, acceptance criteria, and deliverables
4. I will create task items in project with proper field values
5. I will set task priority, complexity, and effort estimates
6. I will assign tasks to appropriate team members
7. I will define task dependencies and blockers
8. I will set task due dates and milestone targets
9. I will add task labels and categorization
10. I will link tasks to parent epic or feature
11. I will validate task completeness and field accuracy
12. I will update project views and sprint planning
13. I will notify assignees and stakeholders

MANDATORY: Always check for repository-specific overrides first. Ensure all tasks have clear
acceptance criteria and proper field values. Validate task assignments and dependencies before
marking complete.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
