#!/usr/bin/env bash
# Shared helpers for setup.sh / start.sh. Source this file, don't execute it.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# CLAUDE_ALLOWED_TOOLS: tools the deliver.sh pipeline's headless `claude -p`
# phases may call (--allowedTools). Kept narrow on purpose since these run
# unattended off ticket content the pipeline hasn't vetted. Extend this list
# when Stage 2/3 phases (backend/frontend/qa/report) are implemented — they
# will need Edit/Bash/git and, for the report phase, addCommentToJiraIssue.
if [[ -z "${CLAUDE_ALLOWED_TOOLS+x}" ]]; then
  CLAUDE_ALLOWED_TOOLS=(
    Read
    Write
    "mcp__claude_ai_Atlassian__getJiraIssue"
    "mcp__claude_ai_Atlassian__getAccessibleAtlassianResources"
    "mcp__claude_ai_Atlassian__searchJiraIssuesUsingJql"
  )
fi

# Reads projects.env into the current shell (FRONTEND_COUNT, FRONTEND_KEY_1, ...)
# and validates the schema.
load_projects() {
  local env_file="$ROOT_DIR/projects.env"
  if [[ ! -f "$env_file" ]]; then
    echo "error: $env_file not found" >&2
    exit 1
  fi
  # shellcheck disable=SC1090
  source "$env_file"

  FRONTEND_COUNT="${FRONTEND_COUNT:-0}"
  BACKEND_COUNT="${BACKEND_COUNT:-0}"

  local seen_keys=()
  local type i key_var key
  for type in FRONTEND BACKEND; do
    local count_var="${type}_COUNT"
    local count="${!count_var}"
    for ((i = 1; i <= count; i++)); do
      key_var="${type}_KEY_${i}"
      key="${!key_var}"
      if [[ -z "$key" ]]; then
        echo "error: ${key_var} is not set (projects.env)" >&2
        exit 1
      fi
      for seen in "${seen_keys[@]:-}"; do
        if [[ "$seen" == "$key" ]]; then
          echo "error: duplicate project key '$key' in projects.env (keys must be unique across frontend+backend)" >&2
          exit 1
        fi
      done
      seen_keys+=("$key")
    done
  done
}

# warn_if_untrusted <ticket>
# Non-blocking safety reminder shown before deliver.sh starts its pipeline.
# Every phase after this point is a `claude -p ... --permission-mode
# acceptEdits` call seeded from content fetched off Jira (ticket description,
# comments) that nothing here has vetted — auto-approved edits driven by
# text from an external ticket is a prompt-injection surface, so only run
# this against tickets from a Jira project/instance you trust.
warn_if_untrusted() {
  local ticket="$1"
  echo "warning: deliver.sh runs multiple 'claude -p --permission-mode acceptEdits' phases" >&2
  echo "  seeded from this ticket's Jira content (description/comments), which is not" >&2
  echo "  reviewed before those phases act on it. Only proceed if you trust the source." >&2
  if [[ "$ticket" == *"://"* || "$ticket" == *"/"* ]]; then
    echo "warning: '$ticket' looks like a URL, not a bare ticket key." >&2
    echo "  Pass just the key (e.g. SA-19) — a URL will break the .workflow/<TICKET>/ path." >&2
  fi
}

# project_dir <type: frontend|backend, lowercase> <key>
# Default: workspace/<type>/<key>. An optional <TYPE>_DIR_<N> in projects.env
# overrides it, relative to workspace/<type>/ ("." = workspace/<type> itself,
# for a project that lives directly in this repo).
project_dir() {
  local type_lower="$1" key="$2"
  local type_upper count_var i key_var dir_var
  type_upper="$(echo "$type_lower" | tr '[:lower:]' '[:upper:]')"
  count_var="${type_upper}_COUNT"
  for ((i = 1; i <= ${!count_var:-0}; i++)); do
    key_var="${type_upper}_KEY_${i}"
    dir_var="${type_upper}_DIR_${i}"
    if [[ "${!key_var}" == "$key" && -n "${!dir_var:-}" ]]; then
      if [[ "${!dir_var}" == "." ]]; then
        echo "$ROOT_DIR/workspace/${type_lower}"
      else
        echo "$ROOT_DIR/workspace/${type_lower}/${!dir_var}"
      fi
      return
    fi
  done
  echo "$ROOT_DIR/workspace/${type_lower}/${key}"
}

# for_each_project <TYPE: FRONTEND|BACKEND> <callback>
# callback is invoked as: callback <key> <name> <repo> <start> <type_lower> <init>
# <init> is the optional *_REPO_INIT_<N> command (empty string if unset).
for_each_project() {
  local type="$1" callback="$2"
  local count_var="${type}_COUNT"
  local count="${!count_var}"
  local type_lower
  type_lower="$(echo "$type" | tr '[:upper:]' '[:lower:]')"

  local i key_var name_var repo_var start_var init_var
  for ((i = 1; i <= count; i++)); do
    key_var="${type}_KEY_${i}"
    name_var="${type}_NAME_${i}"
    repo_var="${type}_REPO_${i}"
    start_var="${type}_START_${i}"
    init_var="${type}_REPO_INIT_${i}"
    "$callback" "${!key_var}" "${!name_var}" "${!repo_var}" "${!start_var}" "$type_lower" "${!init_var:-}"
  done
}
