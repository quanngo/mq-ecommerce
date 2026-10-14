#!/usr/bin/env python3
"""Push the current local branch to its remote for projects declared in
projects.env. Defaults to a PREVIEW (no push performed) showing what would
be pushed; pass --execute to actually push. Never force-pushes, never
creates a new upstream, never touches a branch other than the current one.
No third-party dependencies.
"""
import re
import subprocess
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[3]
PROJECTS_ENV = ROOT_DIR / "projects.env"
GIT_TIMEOUT = 30  # seconds, guards against a hung push/fetch (no network / bad auth)

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

    return [p for p in projects if p["key"] in targets or p["type"] in targets]


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


def inspect_for_push(project_dir: Path):
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
                 "detail": "no upstream configured (won't auto-create one)"}
    upstream = upstream_result.stdout.strip()
    remote = upstream.split("/", 1)[0]

    fetch_result = run_git(["fetch", "--quiet", remote], project_dir)
    if isinstance(fetch_result, Exception) or fetch_result.returncode != 0:
        return {"branch": branch, "status": "skipped",
                 "detail": "fetch failed, could not verify remote state"}

    counts = run_git(
        ["rev-list", "--left-right", "--count", f"{upstream}...HEAD"], project_dir
    )
    if isinstance(counts, Exception) or counts.returncode != 0:
        return {"branch": branch, "status": "error",
                 "detail": "could not compare local/remote history"}
    behind_str, ahead_str = counts.stdout.split()
    behind, ahead = int(behind_str), int(ahead_str)

    if ahead == 0:
        return {"branch": branch, "status": "nothing to push",
                 "detail": f"up to date with {upstream}"}
    if behind > 0:
        return {"branch": branch, "status": "would be rejected",
                 "detail": f"{ahead} ahead, {behind} behind {upstream} — pull first"}
    return {"branch": branch, "status": "ready to push",
             "detail": f"{ahead} commit(s) ahead of {upstream}"}


def push_project(project_dir: Path, preview: dict):
    if preview["status"] != "ready to push":
        return preview

    push_result = run_git(["push"], project_dir)
    if isinstance(push_result, subprocess.TimeoutExpired):
        return {**preview, "status": "failed", "detail": "timed out"}
    if isinstance(push_result, OSError):
        return {**preview, "status": "failed", "detail": str(push_result)}

    if push_result.returncode == 0:
        return {**preview, "status": "pushed", "detail": preview["detail"]}

    reason = (push_result.stderr or push_result.stdout).strip().splitlines()
    reason = reason[-1] if reason else "push failed"
    return {**preview, "status": "failed", "detail": reason}


def main():
    if not PROJECTS_ENV.exists():
        print(f"error: {PROJECTS_ENV} not found")
        return

    args = sys.argv[1:]
    execute = "--execute" in args
    targets = [a for a in args if a != "--execute"]

    env = parse_env(PROJECTS_ENV)
    all_projects = list_projects(env)
    projects = select_projects(all_projects, targets)
    if projects is None:
        return
    if not projects:
        print("no matching projects to push")
        return

    rows = []
    skipped_not_cloned = []

    for project in projects:
        project_dir = project["dir"]
        if not (project_dir / ".git").exists():
            skipped_not_cloned.append(project["key"])
            continue
        preview = inspect_for_push(project_dir)
        info = push_project(project_dir, preview) if execute else preview
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

    if not execute:
        print()
        print("Preview only — nothing was pushed. Re-run with --execute to push "
              "the rows marked 'ready to push'.")


if __name__ == "__main__":
    main()
