#!/usr/bin/env python3
"""Owns .workflow/<TICKET>/progress.json so bash never hand-parses JSON.

Usage:
  workflow_state.py init  <TICKET>
  workflow_state.py get   <TICKET> <phase>
  workflow_state.py set   <TICKET> <phase> <status> [--exit N] [--attempt N] [--note TEXT]
  workflow_state.py next  <TICKET>          # first phase not yet "done" (empty if all done)
  workflow_state.py show  <TICKET>
"""
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[1]
WORKFLOW_DIR = ROOT_DIR / ".workflow"

# Phase order is the pipeline definition. Phases 3-6 land in Stage 2/3.
PHASES = ["ticket", "plan", "backend", "frontend", "qa", "report"]
VALID_STATUS = {"pending", "running", "done", "failed", "skipped"}


def state_path(ticket: str) -> Path:
    return WORKFLOW_DIR / ticket / "progress.json"


def now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def load(ticket: str) -> dict:
    path = state_path(ticket)
    if not path.exists():
        return {}
    try:
        return json.loads(path.read_text())
    except json.JSONDecodeError:
        return {}


def save(ticket: str, state: dict) -> None:
    path = state_path(ticket)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(state, indent=2) + "\n")


def cmd_init(ticket: str) -> int:
    state = load(ticket)
    if state.get("phases"):
        # Already initialised — don't clobber progress on a re-run/resume.
        print(f"{state_path(ticket)} (existing)")
        return 0
    state = {
        "ticket": ticket,
        "created_at": now(),
        "updated_at": now(),
        "phases": {
            name: {"status": "pending", "attempt": 0, "exit_code": None,
                   "note": "", "updated_at": None}
            for name in PHASES
        },
    }
    save(ticket, state)
    print(state_path(ticket))
    return 0


def cmd_get(ticket: str, phase: str) -> int:
    state = load(ticket)
    entry = state.get("phases", {}).get(phase)
    if entry is None:
        print("unknown", file=sys.stderr)
        return 1
    print(entry["status"])
    return 0


def cmd_set(ticket: str, phase: str, status: str, exit_code, attempt, note) -> int:
    if phase not in PHASES:
        print(f"error: unknown phase '{phase}' (valid: {', '.join(PHASES)})", file=sys.stderr)
        return 1
    if status not in VALID_STATUS:
        print(f"error: unknown status '{status}' (valid: {', '.join(sorted(VALID_STATUS))})",
              file=sys.stderr)
        return 1

    state = load(ticket)
    if not state.get("phases"):
        cmd_init(ticket)
        state = load(ticket)

    entry = state["phases"][phase]
    entry["status"] = status
    entry["updated_at"] = now()
    if exit_code is not None:
        entry["exit_code"] = int(exit_code)
    if attempt is not None:
        entry["attempt"] = int(attempt)
    elif status == "running":
        entry["attempt"] = int(entry.get("attempt", 0)) + 1
    if note is not None:
        entry["note"] = note

    state["updated_at"] = now()
    save(ticket, state)
    return 0


def cmd_next(ticket: str) -> int:
    state = load(ticket)
    phases = state.get("phases", {})
    for name in PHASES:
        if phases.get(name, {}).get("status") != "done":
            print(name)
            return 0
    print("")  # all phases done
    return 0


def cmd_show(ticket: str) -> int:
    state = load(ticket)
    if not state:
        print(f"no workflow state for {ticket}", file=sys.stderr)
        return 1
    print(f"ticket: {state.get('ticket', ticket)}   updated: {state.get('updated_at', '-')}")
    print(f"{'phase':<12} {'status':<10} {'attempt':<8} {'exit':<6} note")
    for name in PHASES:
        e = state.get("phases", {}).get(name, {})
        exit_code = e.get("exit_code")
        print(f"{name:<12} {e.get('status', '-'):<10} {e.get('attempt', 0):<8} "
              f"{('-' if exit_code is None else exit_code):<6} {e.get('note', '')}")
    return 0


def parse_flag(args, flag):
    if flag in args:
        i = args.index(flag)
        if i + 1 < len(args):
            value = args[i + 1]
            del args[i:i + 2]
            return value
        print(f"error: {flag} requires a value", file=sys.stderr)
        sys.exit(1)
    return None


def main() -> int:
    args = sys.argv[1:]
    if not args:
        print(__doc__.strip())
        return 1

    command = args.pop(0)

    if command == "init" and len(args) == 1:
        return cmd_init(args[0])
    if command == "get" and len(args) == 2:
        return cmd_get(args[0], args[1])
    if command == "next" and len(args) == 1:
        return cmd_next(args[0])
    if command == "show" and len(args) == 1:
        return cmd_show(args[0])
    if command == "set":
        exit_code = parse_flag(args, "--exit")
        attempt = parse_flag(args, "--attempt")
        note = parse_flag(args, "--note")
        if len(args) == 3:
            return cmd_set(args[0], args[1], args[2], exit_code, attempt, note)

    print(__doc__.strip())
    return 1


if __name__ == "__main__":
    sys.exit(main())
