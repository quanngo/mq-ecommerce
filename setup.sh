#!/usr/bin/env bash
# Clones/updates every project in projects.env, bootstraps a per-project
# CLAUDE.md via `claude /init` where missing, splits that CLAUDE.md into
# topic rule files under .claude/rules/ where none exist yet, then
# synthesizes a root CLAUDE.md describing the whole workspace. Safe to
# re-run after editing projects.env or adding a new project.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/scripts/lib.sh"

if ! command -v claude >/dev/null 2>&1; then
  echo "error: 'claude' CLI not found on PATH. Install Claude Code first." >&2
  exit 1
fi
if ! command -v git >/dev/null 2>&1; then
  echo "error: 'git' not found on PATH." >&2
  exit 1
fi

load_projects

SUMMARY=()

clone_or_update() {
  local key="$1" name="$2" repo="$3" _start="$4" type_lower="$5" init="$6"
  local dir
  dir="$(project_dir "$type_lower" "$key")"

  if [[ -d "$dir/.git" ]]; then
    echo "==> [$type_lower/$key] updating existing clone ($name)"
    if ! git -C "$dir" pull --ff-only; then
      echo "warning: fast-forward pull failed for $key, leaving local state untouched" >&2
    fi
  else
    echo "==> [$type_lower/$key] cloning $repo -> workspace/$type_lower/$key ($name)"
    mkdir -p "$(dirname "$dir")"
    git clone "$repo" "$dir"
  fi

  if [[ -n "$init" ]]; then
    echo "==> [$type_lower/$key] running init command: $init"
    if ! (cd "$dir" && eval "$init"); then
      echo "warning: init command failed for $key ($init)" >&2
    fi
  fi
}

init_claude_md() {
  local key="$1" name="$2" _repo="$3" _start="$4" type_lower="$5" _init="$6"
  local dir
  dir="$(project_dir "$type_lower" "$key")"
  local claude_md_status="already initialized"

  if [[ ! -f "$dir/CLAUDE.md" ]]; then
    echo "==> [$type_lower/$key] no CLAUDE.md yet, running claude /init ($name)"
    (cd "$dir" && claude -p "/init" --permission-mode acceptEdits)
    claude_md_status="generated"
  else
    echo "==> [$type_lower/$key] CLAUDE.md already exists, skipping /init"
  fi

  local rules_dir="$dir/.claude/rules"
  local rules_status="already present"
  if [[ ! -d "$rules_dir" || -z "$(ls -A "$rules_dir" 2>/dev/null)" ]]; then
    echo "==> [$type_lower/$key] no rules yet, splitting CLAUDE.md into .claude/rules/"
    # .claude/ is a hardcoded protected path in Claude Code: writes there are never
    # auto-approved by --permission-mode/allow rules, only by bypassPermissions.
    # Since this runs headless (-p), there's no one to answer that prompt, so we
    # need --dangerously-skip-permissions for this one invocation.
    (cd "$dir" && claude -p "Read ./CLAUDE.md and split its content into multiple \
topic-specific rule files under {$rules_dir} (create the directory if needed). \
Pick topics that actually match this project's real content (for example: \
architecture, API conventions, database schema, testing conventions, coding \
conventions — only the ones that apply here, don't force a fixed list). Each rule \
file is plain markdown covering just that one topic in depth. Preserve all \
information from the original CLAUDE.md — reorganize it, don't drop any of it. \
Then rewrite ./CLAUDE.md into a concise overview that references each new rule \
file using Claude Code's @path import syntax (e.g. '@.claude/rules/architecture.md' \
on its own line), so the full detail still loads as context, organized by topic." \
      --dangerously-skip-permissions)
    rules_status="generated"
  else
    echo "==> [$type_lower/$key] .claude/rules/ already has content, skipping split"
  fi

  SUMMARY+=("$key|$type_lower|$dir|$claude_md_status|$rules_status")
}

echo "--- Step 1: clone/update projects ---"
for_each_project FRONTEND clone_or_update
for_each_project BACKEND clone_or_update

echo
echo "--- Step 2: bootstrap per-project CLAUDE.md + .claude/rules/ ---"
for_each_project FRONTEND init_claude_md
for_each_project BACKEND init_claude_md

echo
echo "--- Step 3: synthesize root CLAUDE.md ---"
(
  cd "$ROOT_DIR"
  claude -p "Synthesize a root CLAUDE.md describing this fullstack workspace. \
Read workspace/frontend/*/CLAUDE.md and workspace/backend/*/CLAUDE.md (and any \
ARCHITECTURE.md files in those project directories), and any files under .claude/rules/, \
and produce an overview covering: overall architecture, how the frontend project(s) and \
backend project(s) relate to each other, key API integration points between them, and a \
linked index of each project's own CLAUDE.md (relative links like \
workspace/frontend/<key>/CLAUDE.md). Write the result to ./CLAUDE.md." \
    --permission-mode acceptEdits
)

echo
echo "--- Summary ---"
printf "%-14s %-10s %-45s %-18s %s\n" "KEY" "TYPE" "PATH" "CLAUDE.md" "rules/"
for row in "${SUMMARY[@]}"; do
  IFS='|' read -r key type_lower dir claude_md_status rules_status <<<"$row"
  printf "%-14s %-10s %-45s %-18s %s\n" "$key" "$type_lower" "$dir" "$claude_md_status" "$rules_status"
done

echo
echo "Setup complete. Run ./start.sh to launch everything (see README.md)."
