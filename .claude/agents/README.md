# Workspace-level subagents

Put custom subagent definitions that operate across the workspace here (e.g. a
`backend-reviewer` that only looks at `workspace/backend/`, or a
`fullstack-integrator` that checks frontend/backend API contracts stay in sync).

Layout Claude Code expects:

```
.claude/agents/<agent-name>.md
```

Each file needs YAML frontmatter (`name`, `description`, optionally `tools`)
followed by the agent's system prompt. See the Claude Code docs for the full
format.
