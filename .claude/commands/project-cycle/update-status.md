# Update Project Status

**Execution Mode**: 🤖 **Autonomous AI Agent** **Purpose**: Following the Project Status Management
Workflow from ~/git/dotfiles/workflows/PROJECT_STATUS_MANAGEMENT_WORKFLOW.md

## For AI Agents

When invoked, you should:

- ✅ Execute all steps automatically
- ✅ Make decisions based on the status management requirements
- ✅ Update project status and metrics
- ✅ Generate reports and identify risks
- ✅ Complete the project status update from review to validation

Do NOT just provide instructions - actually perform the work.

**Note**: This prompt is designed for autonomous AI agent execution, not manual human workflow
steps. For reference documentation, see
`~/git/dotfiles/workflows/PROJECT_STATUS_MANAGEMENT_WORKFLOW.md`.

---

1. I will check for repo-specific overrides first - look for repository-specific workflows:
   - Check for {current_repo}/workflows/ directory
   - Check for ~/git/dotfiles/workflows/PROJECT_STATUS_MANAGEMENT_WORKFLOW.md
   - Check for ~/git/dotfiles/docs/DEVELOPMENT.md or CONTRIBUTING.md
   - Use repo-specific overrides if found, otherwise use default workflow

2. I will read the docs - understand status management and project tracking process:
   - If repo-specific: Use current repo's workflow files
   - If not found: Use ~/git/dotfiles/workflows/PROJECT_STATUS_MANAGEMENT_WORKFLOW.md

3. I will review current project status and progress metrics
4. I will identify completed, in-progress, and blocked items
5. I will update item status fields based on current state
6. I will resolve blockers and update dependency status
7. I will reorder items based on priority and readiness
8. I will update project views and sprint planning
9. I will generate project status report and metrics
10. I will identify risks and mitigation strategies
11. I will update stakeholder communications
12. I will plan next sprint and priority adjustments
13. I will validate project health and progress

MANDATORY: Always check for repository-specific overrides first. Ensure accurate status updates and
proper item ordering. Validate project health metrics before marking complete.

## Related Documentation

- **Architecture**: ~/git/dotfiles/docs/COMPOSABLE_AI_WORKFLOWS_ARCHITECTURE.md
- **Quality Principles**: ~/git/dotfiles/docs/WORKFLOW_QUALITY_PRINCIPLES.md
- **Agent Boundaries**: ~/git/dotfiles/docs/AGENT_BOUNDARIES.md
- **Automation Policy**: ~/git/dotfiles/docs/AUTOMATION_POLICY.md
