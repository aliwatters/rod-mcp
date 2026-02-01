# Create GitHub Project

**Execution Mode**: 🤖 **Autonomous AI Agent** **Purpose**: Following the Project Management
Workflow from ~/git/dotfiles/workflows/PROJECT_MANAGEMENT_WORKFLOW.md

## For AI Agents

When invoked, you should:

- ✅ Execute all steps automatically
- ✅ Make decisions based on the project management requirements
- ✅ Create and configure the project
- ✅ Set up automation and validation
- ✅ Complete the project setup from definition to validation

Do NOT just provide instructions - actually perform the work.

**Note**: This prompt is designed for autonomous AI agent execution, not manual human workflow
steps. For reference documentation, see `~/git/dotfiles/workflows/PROJECT_MANAGEMENT_WORKFLOW.md`.

---

1. I will check for repo-specific overrides first - look for repository-specific workflows:
   - Check for {current_repo}/workflows/ directory
   - Check for ~/git/dotfiles/workflows/PROJECT_MANAGEMENT_WORKFLOW.md
   - Check for ~/git/dotfiles/docs/DEVELOPMENT.md or CONTRIBUTING.md
   - Use repo-specific overrides if found, otherwise use default workflow

2. I will read the docs - understand project structure and field requirements:
   - If repo-specific: Use current repo's workflow files
   - If not found: Use ~/git/dotfiles/workflows/PROJECT_MANAGEMENT_WORKFLOW.md

3. I will define project scope and objectives using project creation template
4. I will set up project fields (Status, Priority, Epic, Sprint, Assignee, Labels)
5. I will configure project views (Backlog, In Progress, Review, Done)
6. I will link project to relevant repositories and teams
7. I will create initial epic-level items for major features
8. I will set up project automation rules and workflows
9. I will validate project structure and field configurations
10. I will test project item creation and field updates
11. I will document project setup and share with team
12. I will configure project notifications and access controls
13. I will run project validation checklist

MANDATORY: Always check for repository-specific overrides first. Use standardized project fields and
views. Ensure all team members have appropriate access. Validate project structure before marking
complete.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
