#!/usr/bin/env python3
"""Print a Markdown table of git branch status for every project declared in
projects.env (frontend + backend). No third-party dependencies.
"""
import re
import subprocess
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[3]
PROJECTS_ENV = ROOT_DIR / "projects.env"
GIT_TIMEOUT = 15  # seconds, guards against a hung fetch (no network / bad auth)

ENV_LINE_RE = re.compile(r'^([A-Z_][A-Z0-9_]*)=(.*)$')


def parse_env(path: Path) -> dict:
    env = {}
    for raw_line in path.read_text().splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        m = ENV_LINE_RE.match(line)
        if not m:
            continue
        key, value = m.group(1), m.group(2).strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
            value = value[1:-1]
        env[key] = value
    return env


def list_projects(env: dict):
    projects = []
    for type_upper in ("FRONTEND", "BACKEND"):
        count = int(env.get(f"{type_upper}_COUNT", "0") or 0)
        type_lower = type_upper.lower()
        for i in range(1, count + 1):
            key = env.get(f"{type_upper}_KEY_{i}")
            if not key:
                continue
            projects.append({
                "key": key,
                "type": type_lower,
                "dir": ROOT_DIR / "workspace" / type_lower / key,
            })
    return projects


def run_git(args, cwd):
    try:
        result = subprocess.run(
            ["git", *args],
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=GIT_TIMEOUT,
        )
    except (subprocess.TimeoutExpired, OSError):
        return None
    if result.returncode != 0:
        return None
    return result.stdout.strip()


def inspect_project(project_dir: Path):
    row = {
        "current_branch": "-",
        "latest_local_update": "-",
        "latest_remote_update": "-",
        "latest_remote_user": "-",
        "dirty": False,
    }

    branch = run_git(["branch", "--show-current"], project_dir)
    row["current_branch"] = branch if branch else "(detached HEAD)"

    local_date = run_git(
        ["log", "-1", "--date=iso-strict", "--format=%cd"], project_dir
    )
    row["latest_local_update"] = local_date or "no commits"

    status = run_git(["status", "--porcelain"], project_dir)
    row["dirty"] = bool(status)

    upstream = run_git(
        ["rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"], project_dir
    )
    if not upstream:
        row["latest_remote_update"] = "no upstream configured"
        row["latest_remote_user"] = "no upstream configured"
        return row

    remote = upstream.split("/", 1)[0]
    fetch_ok = True
    try:
        subprocess.run(
            ["git", "fetch", "--quiet", remote],
            cwd=project_dir,
            capture_output=True,
            text=True,
            timeout=GIT_TIMEOUT,
            check=True,
        )
    except (subprocess.TimeoutExpired, OSError, subprocess.CalledProcessError):
        fetch_ok = False

    if not fetch_ok:
        row["latest_remote_update"] = "fetch failed"
        row["latest_remote_user"] = "fetch failed"
        return row

    remote_date = run_git(
        ["log", "-1", "--date=iso-strict", "--format=%cd", upstream], project_dir
    )
    remote_user = run_git(["log", "-1", "--format=%an", upstream], project_dir)
    row["latest_remote_update"] = remote_date or "no commits"
    row["latest_remote_user"] = remote_user or "unknown"
    return row


def main():
    if not PROJECTS_ENV.exists():
        print(f"error: {PROJECTS_ENV} not found")
        return

    env = parse_env(PROJECTS_ENV)
    projects = list_projects(env)

    rows = []
    skipped = []
    dirty_notes = []

    for project in projects:
        project_dir = project["dir"]
        if not (project_dir / ".git").exists():
            skipped.append(project["key"])
            continue
        info = inspect_project(project_dir)
        rows.append((
            project["key"],
            info["current_branch"],
            info["latest_local_update"],
            info["latest_remote_update"],
            info["latest_remote_user"],
        ))
        if info["dirty"]:
            dirty_notes.append(project["key"])

    header = ("project", "current_branch", "latest_local_update",
              "latest_remote_update", "latest_remote_user")
    print("| " + " | ".join(header) + " |")
    print("|" + "|".join(["---"] * len(header)) + "|")
    for row in rows:
        print("| " + " | ".join(row) + " |")

    if skipped:
        print()
        print("Not cloned yet (run ./setup.sh): " + ", ".join(skipped))

    if dirty_notes:
        print()
        print("Uncommitted local changes: " + ", ".join(dirty_notes))


if __name__ == "__main__":
    main()
