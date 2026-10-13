# Workspace-level skills

Put skills that apply across the whole fullstack workspace here (e.g. "sync API
types between backend and frontend", "generate an end-to-end feature scaffold").
Per-repo skills that only make sense inside one project belong in that project's
own `.claude/skills/` instead (e.g. `workspace/backend/api/.claude/skills/`).

Layout Claude Code expects:

```
.claude/skills/<skill-name>/SKILL.md
```

`SKILL.md` needs YAML frontmatter with `name` and `description`, followed by the
skill's instructions. See the Claude Code docs for the full format.
