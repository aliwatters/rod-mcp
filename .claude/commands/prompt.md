# prompt

Select and execute a prompt from the prompt CLI.

## Usage

```bash
/prompt                    # List available prompts and ask which to execute
/prompt <name>             # Execute specific prompt directly
/prompt dev-cycle          # Example: run dev-cycle prompt
/prompt grooming-cycle     # Example: run grooming-cycle prompt
```

## Description

This command provides a unified interface to all prompts managed by the `prompt` CLI. It can:

1. List all available prompts when called without arguments
2. Execute a specific prompt when given a name argument

## Instructions

If the user provided a prompt name argument:

- Execute `prompt <name>` directly to run that prompt

If no argument was provided:

- Run `prompt --list` to show available prompts
- Ask the user which prompt they would like to execute
- Once they select one, execute `prompt <selected-name>`

## Available Prompt Categories

Common prompts include:

- **dev-cycle**: Complete development cycle from issue to PR
- **grooming-cycle**: Issue grooming and backlog maintenance
- **pr-ready-cycle**: Prepare PRs for human review
- **resume-cycle**: Resume any active workflow cycle
- **session-harvest**: Analyze session for automation opportunities

Run `prompt --list` for the complete list with descriptions.
