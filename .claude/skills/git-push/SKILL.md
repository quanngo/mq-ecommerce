---
name: git-push
description: Push the current local branch to its remote for one, several, or all frontend/backend projects defined in projects.env. Always previews what would be pushed and gets developer confirmation before actually pushing. Never force-pushes, never creates a new upstream, never touches a branch other than the current one. Use when asked to push/publish local commits across projects.
---

Pushing changes to a remote is visible to the rest of the team and affects
shared state — always preview before pushing, never push silently.

## Steps

1. **Preview first.** Run, from the workspace root:
   ```bash
   python3 .claude/skills/git-push/push_all.py [target ...]
   ```
   - **No arguments** → previews every project in `projects.env`.
   - **One or more project keys** (e.g. `api`, `web`) → previews only those.
   - **`frontend` or `backend`** as a target → previews every project of that
     type; can be combined with specific keys.

   If the developer named specific project(s), pass exactly those keys. If
   they didn't specify anything, preview all.

   This performs a read-only fetch per project and reports, per project:
   `nothing to push` (already up to date), `ready to push` (N commits ahead),
   `would be rejected` (remote has newer commits — needs a pull first), or
   `skipped` (detached HEAD / no upstream configured / fetch failed). It
   never pushes anything at this stage.

2. **Show the preview table to the developer and get explicit confirmation**
   before pushing anything — summarize which projects/branches are
   `ready to push` and how many commits each would push. If any project
   shows `would be rejected`, tell the developer it needs a pull (e.g. via
   the `git-pull` skill) before it can be pushed, and don't attempt to force
   past it.

3. **Only after the developer confirms**, re-run the same command with
   `--execute` appended (same target arguments as the preview):
   ```bash
   python3 .claude/skills/git-push/push_all.py [target ...] --execute
   ```
   This runs a plain `git push` (never `--force`, never `--force-with-lease`)
   on each project's current branch to its existing upstream only — it never
   creates a new upstream, switches branches, or touches any branch other
   than the current one.

4. **Show the execute results as-is.** Report each row's real status
   (`pushed` / `failed` / `nothing to push` / `skipped`) — don't paper over a
   `failed` row or retry with force; surface the failure reason to the
   developer instead.
