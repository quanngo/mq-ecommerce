#!/usr/bin/env bash
# Pre-commit gate for the Go backend: deps, format, build, vet, tests, secrets.
#   check.sh            run every check, report all failures, exit 1 if any failed
#   check.sh --no-test  skip go test (e.g. while iterating on a compile error)
# BACKEND_DIR overrides the backend location (default: workspace/backend).
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
BACKEND="${BACKEND_DIR:-$ROOT/workspace/backend}"
RUN_TESTS=1
[[ "${1:-}" == "--no-test" ]] && RUN_TESTS=0

if [[ ! -f "$BACKEND/go.mod" ]]; then
  echo "error: no go.mod in $BACKEND" >&2
  exit 1
fi
cd "$BACKEND"

FAILED=()
step() { echo; echo "=== $1 ==="; }
fail() { FAILED+=("$1"); echo "FAIL: $1"; }

step "go mod tidy (go.sum is gitignored, so it must be regenerated locally)"
go mod tidy || fail "go mod tidy"
if ! git diff --quiet --ignore-cr-at-eol -- go.mod 2>/dev/null; then
  echo "note: go mod tidy changed go.mod content — review and commit it:"
  git --no-pager diff --ignore-cr-at-eol --stat -- go.mod
fi

step "gofmt (changed .go files only; CRLF stripped before checking)"
mapfile -t changed < <( { git diff --name-only --relative HEAD -- '*.go'; git ls-files --others --exclude-standard -- '*.go'; } 2>/dev/null | sort -u)
unformatted=()
for f in "${changed[@]}"; do
  [[ -f "$f" ]] || continue
  # gofmt -l prints "<standard input>" when unformatted, or an error on bad syntax.
  [[ -n "$(tr -d '\r' < "$f" | gofmt -l 2>&1)" ]] && unformatted+=("$f")
done
if [[ ${#unformatted[@]} -gt 0 ]]; then
  printf '  %s\n' "${unformatted[@]}"
  fail "gofmt: ${#unformatted[@]} file(s) need formatting"
else
  echo "ok (${#changed[@]} changed file(s))"
fi

step "go build ./..."
go build ./... || fail "go build"

step "go vet ./..."
go vet ./... || fail "go vet"

if [[ "$RUN_TESTS" -eq 1 ]]; then
  step "go test ./..."
  go test ./... -count=1 2>&1 | grep -v "no test files"
  [[ "${PIPESTATUS[0]}" -eq 0 ]] || fail "go test"
fi

step "secrets"
fails_before=${#FAILED[@]}
for f in config/application.yml config/service-account.json config/.env; do
  if git ls-files --error-unmatch "$f" >/dev/null 2>&1; then
    fail "secret file is tracked by git: $f (git rm --cached it)"
  elif [[ -e "$f" ]] && ! git check-ignore -q "$f"; then
    fail "secret file is not gitignored: $f"
  fi
done
# Real values must never appear in tracked or staged files. Secrets normally
# live in config/.env (KEY=VALUE, referenced from the yml as "${KEY}"); a
# literal secret still left in application.yml is checked too.
values=()
if [[ -f config/.env ]]; then
  # Only credential-like names: user names and IDs (e.g. DB_USER=postgres) are
  # ordinary words that legitimately appear in code.
  mapfile -t -O "${#values[@]}" values < <(awk -F'=' '!/^[[:space:]]*(#|$)/ && $1 ~ /(PASSWORD|SECRET|TOKEN|KEY|AUTHENTICATOR)/ {sub(/^[^=]*=/, ""); gsub(/^["'"'"']|["'"'"']$/, ""); if (length($0) >= 6) print}' config/.env)
fi
if [[ -f config/application.yml ]]; then
  mapfile -t -O "${#values[@]}" values < <(awk -F': *' '/(password|secret|key|token)[a-z_-]*:/ && NF>1 && $2 !~ /^ *$/ && $2 !~ /\$\{/ {v=$2; sub(/ *#.*/,"",v); if (length(v) >= 8) print v}' config/application.yml)
fi
for v in "${values[@]}"; do
  if git grep -qIF -- "$v" -- . ':!config/application.yml' ':!config/.env' 2>/dev/null || git diff --cached | grep -qF -- "$v"; then
    fail "a secret value from config/.env or config/application.yml appears in a tracked/staged file"
    break
  fi
done
if [[ -f config/application.yml ]] && awk -F': *' '/(password|secret|token|app_key|partner_key|client-id|authenticator)[a-z_-]*:/ && NF>1 && $2 !~ /^ *("?\$\{[A-Za-z0-9_]+\}"?)? *(#.*)?$/ {found=1} END {exit !found}' config/application.yml; then
  fail "config/application.yml holds a literal secret; move it to config/.env and use \"\${VAR}\""
fi
if git grep -qIE -- "BEGIN (RSA |EC )?PRIVATE KEY|gh[pousr]_[A-Za-z0-9]{30,}" -- . 2>/dev/null; then
  fail "private key or GitHub token found in a tracked file"
fi
[[ ${#FAILED[@]} -eq $fails_before ]] && echo "ok"

echo
if [[ ${#FAILED[@]} -gt 0 ]]; then
  echo "--- ${#FAILED[@]} check(s) failed ---"
  printf '  - %s\n' "${FAILED[@]}"
  exit 1
fi
echo "--- all checks passed ---"
