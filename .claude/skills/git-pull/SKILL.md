---
name: git-pull
description: Pull the latest remote code into the current local branch for one, several, or all frontend/backend projects defined in projects.env. Fast-forward only — never rewrites history or discards local work. Use when asked to update/sync/pull all projects, or a specific project, before starting new work.
---

Run the bundled script from the workspace root:

```bash
python3 .claude/skills/git-pull/pull_all.py [target ...]
```

- **No arguments** → pulls every project declared in `projects.env`.
- **One or more project keys** (the `KEY` from `projects.env`, e.g. `api`,
  `web`) → pulls only those projects.
- **`frontend` or `backend`** as a target → pulls every project of that type.
  Can be combined with specific keys in the same call.

If the developer names specific project(s), pass exactly those keys as
arguments — don't pull everything and then just report on the ones they
asked about. If the developer doesn't specify anything, run it with no
arguments.

For every targeted project that's cloned, it runs `git pull --ff-only` on
that project's **current** local branch — it never checks out a different
branch, never rebases, never force-pulls, and never discards local work: git
itself refuses the fast-forward if the branch has diverged or would
overwrite uncommitted changes, and the script reports that as a failure
rather than forcing past it.

It prints a Markdown table (`project | branch | status | detail`) plus a
separate note for any targeted project that isn't cloned yet. If an unknown
project key or group is passed, it prints an error listing the valid keys
instead of guessing — relay that error to the developer rather than picking
a close match yourself.

Show its output to the developer as-is — don't re-run individual `git pull`
commands yourself, and don't paper over a `failed` or `skipped` row; if a
project shows a conflict, divergence, or "no upstream configured", surface
that to the developer instead of attempting to resolve it automatically
(e.g. never merge, rebase, or force anything on their behalf).
