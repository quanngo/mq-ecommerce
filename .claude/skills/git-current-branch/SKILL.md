---
name: git-current-branch
description: Show a table of every frontend and backend project's git branch status — current branch, last local update time, last remote update time, and who last updated the remote branch. Use when asked to check branch status across the workspace, "what branch is everyone on", or before starting/syncing work.
---

Run the bundled script from the workspace root:

```bash
python3 .claude/skills/git-current-branch/check_branches.py
```

It reads `projects.env`, inspects every cloned frontend/backend project's git
state (current branch, local `HEAD` commit time, and — after a read-only
`git fetch` — the remote branch's commit time and author), and prints a
ready-to-show Markdown table plus notes for any project that isn't cloned yet
or has uncommitted local changes.

Show its output to the developer as-is — don't re-derive the table by hand,
and don't edit or fabricate any value it reports (e.g. if it says
`fetch failed` or `no upstream configured`, report that verbatim rather than
guessing a timestamp or author).
