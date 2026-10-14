#!/usr/bin/env python3
"""Fast-forward-pull projects declared in projects.env on their current local
branch. With no arguments, pulls every project. Pass one or more project
keys (and/or "frontend" / "backend" as a group filter) to pull only those.
Never rewrites history, never discards local work: uses `git pull --ff-only`,
which git itself refuses if it would overwrite uncommitted changes or the
branch has diverged. No third-party dependencies.
"""
import re
import subprocess
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[3]
PROJECTS_ENV = ROOT_DIR / "projects.env"
GIT_TIMEOUT = 30  # seconds, guards against a hung pull (no network / bad auth)

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


def project_dir(env: dict, type_upper: str, i: int, key: str) -> Path:
    """workspace/<type>/<key>, or <TYPE>_DIR_<N> from projects.env ("." = workspace/<type>)."""
    base = ROOT_DIR / "workspace" / type_upper.lower()
    override = env.get(f"{type_upper}_DIR_{i}", "").strip()
    if override == ".":
        return base
    return base / (override or key)


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
                "dir": project_dir(env, type_upper, i, key),
            })
    return projects


def run_git(args, cwd, timeout=GIT_TIMEOUT):
    try:
        return subprocess.run(
            ["git", *args],
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except (subprocess.TimeoutExpired, OSError) as exc:
        return exc


def pull_project(project_dir: Path):
    result = run_git(["branch", "--show-current"], project_dir)
    if isinstance(result, Exception) or result.returncode != 0:
        return {"status": "error", "detail": "could not read current branch"}
    branch = result.stdout.strip()
    if not branch:
        return {"branch": "(detached HEAD)", "status": "skipped",
                 "detail": "detached HEAD, skipped"}

    upstream_result = run_git(
        ["rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"],
        project_dir,
    )
    if isinstance(upstream_result, Exception) or upstream_result.returncode != 0:
        return {"branch": branch, "status": "skipped",
                 "detail": "no upstream configured"}

    pull_result = run_git(["pull", "--ff-only"], project_dir)
    if isinstance(pull_result, subprocess.TimeoutExpired):
        return {"branch": branch, "status": "failed", "detail": "timed out"}
    if isinstance(pull_result, OSError):
        return {"branch": branch, "status": "failed", "detail": str(pull_result)}

    if pull_result.returncode == 0:
        stdout = pull_result.stdout.strip()
        if "Already up to date" in stdout:
            return {"branch": branch, "status": "up to date", "detail": "-"}
        first_line = stdout.splitlines()[-1] if stdout else "fast-forwarded"
        return {"branch": branch, "status": "pulled", "detail": first_line}

    reason = (pull_result.stderr or pull_result.stdout).strip().splitlines()
    reason = reason[-1] if reason else "pull failed"
    return {"branch": branch, "status": "failed", "detail": reason}


def select_projects(projects, targets):
    if not targets:
        return projects

    valid_keys = {p["key"] for p in projects}
    unknown = [t for t in targets if t not in valid_keys and t not in ("frontend", "backend")]
    if unknown:
        print(f"error: unknown project/group: {', '.join(unknown)}")
        print(f"valid project keys: {', '.join(sorted(valid_keys)) or '(none defined)'}")
        print("valid groups: frontend, backend")
        return None

    selected = []
    for project in projects:
        if project["key"] in targets or project["type"] in targets:
            selected.append(project)
    return selected


def main():
    if not PROJECTS_ENV.exists():
        print(f"error: {PROJECTS_ENV} not found")
        return

    env = parse_env(PROJECTS_ENV)
    all_projects = list_projects(env)

    targets = sys.argv[1:]
    projects = select_projects(all_projects, targets)
    if projects is None:
        return
    if not projects:
        print("no matching projects to pull")
        return

    rows = []
    skipped_not_cloned = []

    for project in projects:
        project_dir = project["dir"]
        if not (project_dir / ".git").exists():
            skipped_not_cloned.append(project["key"])
            continue
        info = pull_project(project_dir)
        rows.append((
            project["key"],
            info.get("branch", "-"),
            info["status"],
            info["detail"],
        ))

    header = ("project", "branch", "status", "detail")
    print("| " + " | ".join(header) + " |")
    print("|" + "|".join(["---"] * len(header)) + "|")
    for row in rows:
        print("| " + " | ".join(row) + " |")

    if skipped_not_cloned:
        print()
        print("Not cloned yet (run ./setup.sh): " + ", ".join(skipped_not_cloned))


if __name__ == "__main__":
    main()
