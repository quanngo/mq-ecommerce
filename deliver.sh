#!/usr/bin/env bash
# Ticket-to-delivery orchestrator.
#
#   ./deliver.sh <TICKET>              run the pipeline for a Jira ticket
#   ./deliver.sh <TICKET> --yes        skip the plan-approval gate
#   ./deliver.sh <TICKET> --resume     restart at the first unfinished phase
#   ./deliver.sh <TICKET> --from plan  restart at a specific phase
#   ./deliver.sh <TICKET> --status     show phase progress and exit
#
# Each phase is a separate `claude -p` call seeded only from
# .workflow/<TICKET>/, so no single context window has to hold the whole
# pipeline. Phase logic lives in skills (.claude/skills/<name>/SKILL.md) so
# each phase stays usable standalone as /<skill-name>.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib.sh
source "$SCRIPT_DIR/scripts/lib.sh"

STATE="python3 $ROOT_DIR/scripts/workflow_state.py"

# phase name | skill to invoke | human label   ("" skill = not implemented yet)
PHASE_NAMES=(ticket plan backend frontend qa report)
PHASE_SKILLS=(jira-ticket plan-ticket "" "" "" "")
PHASE_LABELS=(
  "Read the Jira ticket"
  "Plan the implementation"
  "Implement backend (Stage 2)"
  "Implement frontend (Stage 2)"
  "Integration + browser QA, fix loop (Stage 3)"
  "Summary report + Jira comment (Stage 3)"
)

usage() {
  # Print the leading comment block (skipping the shebang), stopping at the
  # first line that isn't a comment.
  awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
}

TICKET=""
SKIP_GATE=0
RESUME=0
FROM_PHASE=""
SHOW_STATUS=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes|-y)   SKIP_GATE=1; shift ;;
    --resume)   RESUME=1; shift ;;
    --from)     FROM_PHASE="${2:-}"; shift 2 ;;
    --status)   SHOW_STATUS=1; shift ;;
    -h|--help)  usage; exit 0 ;;
    -*)         echo "error: unknown option '$1'" >&2; usage >&2; exit 1 ;;
    *)
      if [[ -n "$TICKET" ]]; then
        echo "error: unexpected argument '$1' (ticket already set to '$TICKET')" >&2
        exit 1
      fi
      TICKET="$1"; shift ;;
  esac
done

if [[ -z "$TICKET" ]]; then
  echo "error: no ticket key given" >&2
  usage >&2
  exit 1
fi

WORKFLOW_DIR="$ROOT_DIR/.workflow/$TICKET"

if [[ "$SHOW_STATUS" -eq 1 ]]; then
  $STATE show "$TICKET"
  exit 0
fi

if ! command -v claude >/dev/null 2>&1; then
  echo "error: 'claude' CLI not found on PATH. Install Claude Code first." >&2
  exit 1
fi

phase_index() {
  local wanted="$1" i
  for i in "${!PHASE_NAMES[@]}"; do
    if [[ "${PHASE_NAMES[$i]}" == "$wanted" ]]; then
      echo "$i"
      return 0
    fi
  done
  return 1
}

warn_if_untrusted "$TICKET"

mkdir -p "$WORKFLOW_DIR/logs" "$WORKFLOW_DIR/reports"
$STATE init "$TICKET" >/dev/null

# Decide where to start.
START_INDEX=0
if [[ -n "$FROM_PHASE" ]]; then
  if ! START_INDEX="$(phase_index "$FROM_PHASE")"; then
    echo "error: unknown phase '$FROM_PHASE' (valid: ${PHASE_NAMES[*]})" >&2
    exit 1
  fi
elif [[ "$RESUME" -eq 1 ]]; then
  next_phase="$($STATE next "$TICKET")"
  if [[ -z "$next_phase" ]]; then
    echo "All phases already complete for $TICKET. Nothing to resume."
    $STATE show "$TICKET"
    exit 0
  fi
  START_INDEX="$(phase_index "$next_phase")"
  echo "Resuming $TICKET at phase '$next_phase'."
fi

run_phase() {
  local index="$1"
  local name="${PHASE_NAMES[$index]}"
  local skill="${PHASE_SKILLS[$index]}"
  local label="${PHASE_LABELS[$index]}"
  local num
  num="$(printf '%02d' "$((index + 1))")"

  echo
  echo "=== Phase $num/${#PHASE_NAMES[@]}: $label ==="

  if [[ -z "$skill" ]]; then
    echo "Phase '$name' is not implemented yet (Stage 2/3)."
    echo "Stage 1 covers the Jira read and the plan. See README.md."
    $STATE set "$TICKET" "$name" pending --note "not implemented yet"
    return 10
  fi

  local log="$WORKFLOW_DIR/logs/${num}-${name}.log"
  $STATE set "$TICKET" "$name" running

  local rc=0
  if claude -p "/$skill $TICKET" --permission-mode acceptEdits \
       --allowedTools "${CLAUDE_ALLOWED_TOOLS[@]}" 2>&1 | tee "$log"; then
    rc=0
  else
    rc=${PIPESTATUS[0]}
  fi

  if [[ "$rc" -ne 0 ]]; then
    $STATE set "$TICKET" "$name" failed --exit "$rc" --note "see logs/${num}-${name}.log"
    echo "Phase '$name' failed (exit $rc). Log: $log" >&2
    return "$rc"
  fi

  $STATE set "$TICKET" "$name" done --exit 0 --note ""
  echo "Phase '$name' complete."
  return 0
}

plan_gate() {
  local plan_md="$WORKFLOW_DIR/plan.md"

  echo
  echo "=== Plan approval gate ==="
  if [[ -f "$plan_md" ]]; then
    echo "--- $plan_md ---"
    cat "$plan_md"
    echo "--- end of plan ---"
  else
    echo "warning: $plan_md was not written by the plan phase." >&2
  fi

  if [[ "$SKIP_GATE" -eq 1 ]]; then
    echo "(--yes given, proceeding without approval)"
    return 0
  fi
  if [[ ! -t 0 ]]; then
    echo "(non-interactive run and no --yes: stopping here for human review)"
    echo "Review the plan above, then continue with:"
    echo "  ./deliver.sh $TICKET --resume"
    return 10
  fi

  local reply
  read -r -p "Approve this plan and continue with implementation? [y/N] " reply
  case "$reply" in
    [yY][eE][sS]|[yY]) return 0 ;;
    *)
      echo "Stopped at the plan gate. Edit $plan_md if needed, then re-run:"
      echo "  ./deliver.sh $TICKET --resume"
      return 10 ;;
  esac
}

for ((i = START_INDEX; i < ${#PHASE_NAMES[@]}; i++)); do
  rc=0
  run_phase "$i" || rc=$?
  if [[ "$rc" -eq 10 ]]; then
    # Phase intentionally stopped the run (not implemented yet).
    break
  elif [[ "$rc" -ne 0 ]]; then
    echo
    echo "Pipeline stopped. Fix the issue then resume with:" >&2
    echo "  ./deliver.sh $TICKET --resume" >&2
    exit "$rc"
  fi

  # Human gate sits between the plan phase and implementation.
  if [[ "${PHASE_NAMES[$i]}" == "plan" ]]; then
    gate_rc=0
    plan_gate || gate_rc=$?
    if [[ "$gate_rc" -ne 0 ]]; then
      exit 0
    fi
  fi
done

echo
echo "--- Progress ---"
$STATE show "$TICKET"
echo
echo "State directory: $WORKFLOW_DIR"
