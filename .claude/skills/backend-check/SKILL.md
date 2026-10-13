---
name: backend-check
description: Run the Go backend's pre-commit gate — go mod tidy, gofmt on changed files, go build, go vet, go test, and a secret scan that fails if config/application.yml, config/.env or service-account.json are tracked, if application.yml holds a literal secret instead of a "${VAR}" placeholder, or if any real credential value from config/.env appears in a tracked/staged file. Use before committing or pushing backend changes, after finishing a backend change, or when asked to "check", "verify", or "make sure the backend builds".
---

Run every check, then report the real result. Never mark the backend as
ready while any check fails, and never "fix" a failure by weakening the check.

## Steps

1. **Run the gate** from the workspace root:
   ```bash
   bash .claude/skills/backend-check/check.sh            # all checks
   bash .claude/skills/backend-check/check.sh --no-test  # skip go test
   ```
   It runs all checks even after one fails and exits 1 if any failed.

2. **Fix what it reports, in this order** (earlier failures cause later ones):
   - **go mod tidy**: `go.sum` is gitignored, so a fresh clone fails to build
     with "missing go.sum entry" until tidy runs. If tidy changes `go.mod`
     *content* (the script prints it), that change belongs in the commit.
   - **gofmt**: only files changed vs `HEAD` are checked, because the repo
     uses CRLF line endings and plain `gofmt -l .` flags every file. Format
     with `tr -d '\r' < f | gofmt > f.tmp && mv f.tmp f`. Git restores CRLF
     on checkout.
   - **build / vet / test**: `go test` runs a subset of vet (including
     `printf`). A vet error anywhere in a package stops that package's tests
     from running at all.
   - **secrets**: if a secret file is tracked, untrack it with
     `git rm --cached <file>` and keep it gitignored. If a real value leaked
     into another file, replace it with an empty or placeholder value (see
     `config/application.yml.example`). If it was ever pushed, tell the
     developer that the credential must be **rotated**, because rewriting
     history is not enough.

3. **Re-run until it prints `all checks passed`.** Then report which checks
   failed initially and what you changed. Quote the real output for anything
   you could not fix.

## Notes

- `BACKEND_DIR=<path>` points the script at another backend, for example
  `workspace/backend/<key>` once `projects.env` matches the layout.
- The secret scan does three things:
  - It compares every credential-like value in the local `config/.env` (names
    containing PASSWORD, SECRET, TOKEN, KEY or AUTHENTICATOR, 6 characters or longer),
    plus any literal 8+ character `password`/`secret`/`key`/`token` value still in
    `config/application.yml`, against tracked and staged files.
  - It fails if `application.yml` holds a literal secret instead of a `"${VAR}"`
    placeholder, or if `config/.env` is tracked or not ignored.
  - It looks for PEM private keys and GitHub tokens in tracked files.
