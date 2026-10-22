# Fullstack Claude Workspace Template

A workspace template for fullstack teams: declare your frontend + backend
repos in one file, clone and bootstrap them with Claude Code, and run
everything locally through PM2 with a single command.

## Layout

```
projects.env              # you edit this — declares every project
setup.sh                  # clone/update projects + generate CLAUDE.md docs
start.sh                  # run projects via PM2
CLAUDE.md                 # auto-generated overview of the whole workspace
.claude/
  settings.json            # project-level Claude Code settings (MCP servers, etc.)
  skills/                   # workspace-wide skills
  agents/                   # workspace-wide custom subagents
  rules/                    # house conventions, linked into the generated CLAUDE.md
workspace/
  frontend/<key>/           # cloned frontend repos
  backend/<key>/             # cloned backend repos
```

## Prerequisites

- `git`
- `node` + `npm` (only used to install/run `pm2`)
- [Claude Code CLI](https://docs.claude.com/claude-code) (`claude`), logged in
- `pm2` — `start.sh` will offer to install it for you if missing (`npm install -g pm2`)

## 1. Declare your projects

Edit `projects.env`. Each project needs 4 lines under its type
(`FRONTEND_*` / `BACKEND_*`), numbered starting at 1, plus a `_COUNT`:

```bash
FRONTEND_COUNT=1

FRONTEND_KEY_2=admin
FRONTEND_NAME_2="Admin Dashboard"
FRONTEND_REPO_2=git@github.com:org/admin.git
FRONTEND_START_2="npm run dev"

BACKEND_COUNT=1

BACKEND_KEY_1=api
BACKEND_NAME_1="Core API"
BACKEND_REPO_1=git@github.com:org/api.git
BACKEND_START_1="go run main.go"
```

- `KEY` must be unique across **all** frontend + backend projects. It's used
  as the clone folder name, the PM2 process name, and the argument to
  `./start.sh <key>`.
- `START` is the exact command used to run the project locally.
- Optional `<TYPE>_DIR_<N>`: folder relative to `workspace/<type>/` to use
  instead of `<key>`; `.` means `workspace/<type>` itself.
- `REPO` may be left empty for a project that lives in this repo; setup then
  skips the clone/pull. This workspace's own `projects.env` does that for
  both apps:

  ```bash
  FRONTEND_KEY_1=admin
  FRONTEND_REPO_1=
  FRONTEND_DIR_1=.            # workspace/frontend (dev server :8088)
  FRONTEND_START_1="npm run dev"

  BACKEND_KEY_1=api
  BACKEND_REPO_1=
  BACKEND_DIR_1=.             # workspace/backend (:9091)
  BACKEND_START_1="go run main.go"
  ```

## 2. Run setup

```bash
./setup.sh
```

This does three things:

1. **Clone/update** — clones each repo into `workspace/frontend/<key>` or
   `workspace/backend/<key>`. If a repo is already cloned, it runs
   `git pull --ff-only` instead (local changes are never clobbered — a failed
   fast-forward just prints a warning and moves on).
2. **Per-project docs** — for any project that doesn't yet have a `CLAUDE.md`,
   runs Claude Code's `/init` inside it to generate one (architecture, API
   surface, DB schema for backends, etc). Projects that already have a
   `CLAUDE.md` are left untouched.
3. **Root synthesis** — regenerates the root `CLAUDE.md`, which reads every
   project's `CLAUDE.md`/`ARCHITECTURE.md` plus anything in `.claude/rules/`
   and produces a linked overview of the whole system.

**Whenever you edit `projects.env`** (add, remove, or rename a project),
just re-run `./setup.sh` — it's idempotent and only does the work that's
actually needed (new clones, missing `CLAUDE.md`s), then refreshes the root
overview.

## 3. Run everything

```bash
./start.sh              # start every project
./start.sh frontend     # start only frontend projects
./start.sh backend      # start only backend projects
./start.sh web          # start a single project by its key
```

All projects run under [PM2](https://pm2.keymetrics.io/) in the background.
If `pm2` isn't installed, `start.sh` will ask before installing it globally.

Useful follow-ups:

```bash
pm2 list             # what's running
pm2 logs             # tail all logs (pm2 logs <key> for one project)
pm2 restart <key>    # restart a project
pm2 stop all         # stop everything
pm2 delete all       # remove all processes from pm2
```

## Extending Claude Code for this workspace

- `.claude/settings.json` — register MCP servers shared across the whole
  workspace (databases, ticketing systems, etc).
- `.claude/skills/` — skills that operate across frontend + backend (see the
  README inside for the expected layout).
- `.claude/agents/` — custom subagents scoped to this workspace.
- `.claude/rules/` — house conventions/standards docs; anything dropped here
  gets folded into the root `CLAUDE.md` the next time `./setup.sh` runs.

Each cloned project under `workspace/` can also have its own
`.claude/skills`, `.claude/agents`, etc. for repo-specific needs.

### Available skills

Skills live at `.claude/skills/<name>/SKILL.md` and are auto-discovered by
Claude Code from the workspace root — no setup needed. Run one either by
typing its slash form in a Claude Code session started at the workspace root,
or just ask in natural language (Claude matches your request against each
skill's description and picks it automatically):

- **`/workspace-init`** — orients you fast: reads `projects.env` and every
  project's `CLAUDE.md`, then produces a system overview, a project table,
  integration points, and quick-start commands. Use this right after cloning
  the workspace, or whenever you ask "what's in this workspace" / "get me
  oriented".
- **`/backend-api-docs`** — scans every project under `workspace/backend/`
  and creates or refreshes its `API_DOCUMENT.md` from the actual
  route/handler code (not guesses). Use this to document backend APIs for
  the first time, or to bring the docs back in sync after endpoints change.
- **`/write-code-full`** — implements a fullstack feature end-to-end: reads
  the relevant frontend/backend projects, asks you to clarify anything
  ambiguous (UI placement, technical design, DB schema, API contract) before
  writing any code, implements both sides, wires the frontend to the real
  backend contract, writes unit tests for both, updates the backend
  `API_DOCUMENT.md` (via the same approach as `backend-api-docs`), and
  adds/runs integration tests. Use this when asked to build or add a feature
  spanning both frontend and backend.
- **`/write-code-fe`** — the frontend-only half of `write-code-full`: reads
  the relevant frontend project and the *existing* backend API contract it
  needs to call, clarifies anything ambiguous, implements the frontend side,
  wires it to that existing API, and writes frontend unit + integration
  tests. Never touches backend code — if the API doesn't exist yet, it tells
  you to run `/write-code-be` or `/write-code-full` first.
- **`/write-code-be`** — the backend-only half of `write-code-full`: reads
  the relevant backend project, clarifies anything ambiguous (DB schema, API
  contract, auth), implements the endpoint(s)/schema changes, writes backend
  unit + integration tests, and updates `API_DOCUMENT.md`. Never touches
  frontend code — use this to build/expose an API for the frontend to
  consume later (with `/write-code-fe`).
- **`/git-current-branch`** — shows a table of every project's git status:
  current branch, last local commit time, last remote commit time, and who
  made that last remote update. Use before starting work or syncing, to see
  what branch everything is on at a glance.
- **`/git-pull`** — fast-forward-pulls the current local branch of every
  project from its remote. Never rewrites history or discards local work —
  git itself refuses if a branch has diverged or has conflicting uncommitted
  changes, and that's reported rather than forced past. Use to sync all
  projects before starting work.
- **`/git-push`** — pushes the current local branch of one, several, or all
  projects to its remote. Always previews what would be pushed (commits
  ahead/behind per project) and waits for your confirmation before actually
  pushing. Never force-pushes and never creates a new upstream branch on its
  own. Use to publish local commits across projects.
- **`/backend-check`**: pre-commit gate for the Go backend. Runs
  `go mod tidy`, gofmt on changed files, build, vet and tests, and fails if
  `config/application.yml` / `service-account.json` are tracked or any real
  credential from the local config appears in a tracked or staged file.
  Script: `bash .claude/skills/backend-check/check.sh`.
- **`/db-migrate`**: applies `workspace/backend/migrations/*.sql` to the DB in
  `config/application.yml`, recording each file in
  `public.ecom_schema_migrations` so it runs once. It dry-runs first and only
  writes with `--execute` after you confirm. Also documents how to write a
  new migration.
- **`/add-worker`**: adds or reschedules a cron worker
  (`workers.yml` → `mappingWorkers()` → service method), including the valid
  schedule syntax and the mistakes that silently disable a job or crash the
  process.
- **`/add-marketplace`**: adds a new sales channel, or a new flow to an
  existing one, in the same shape as TikTok/Lazada (API client, models,
  service, `platform_service` dispatch, wiring, required DB rows), and lists
  the existing client bugs not to copy.

Add new workspace-wide skills the same way: create
`.claude/skills/<name>/SKILL.md` with YAML frontmatter (`name`,
`description`) and instructions in the body — see the existing skills above
as examples.
