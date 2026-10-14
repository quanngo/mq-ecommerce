#!/usr/bin/env bash
# Applies workspace/backend/migrations/*.sql to the database in config/application.yml.
#   migrate.sh                 dry run: show target DB, applied and pending files
#   migrate.sh --execute       apply pending files in filename order
#   migrate.sh --config <yml>  read the `database:` block from another file
# Applied files are recorded in public.ecom_schema_migrations. Each file runs in
# its own transaction together with its bookkeeping row, so a failed file
# leaves no partial changes and is retried on the next run.
# BACKEND_DIR overrides the backend location (default: workspace/backend).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
BACKEND="${BACKEND_DIR:-$ROOT/workspace/backend}"
CONFIG="$BACKEND/config/application.yml"
MIG_DIR="$BACKEND/migrations"
EXECUTE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --execute) EXECUTE=1; shift ;;
    --config)  CONFIG="${2:?--config needs a path}"; shift 2 ;;
    -h|--help) awk 'NR==1{next} /^#/{sub(/^# ?/,""); print; next} {exit}' "$0"; exit 0 ;;
    *) echo "error: unknown argument '$1'" >&2; exit 1 ;;
  esac
done

PSQL="$(command -v psql 2>/dev/null || true)"
if [[ -z "$PSQL" ]]; then
  PSQL="$(ls -1 /c/Program\ Files/PostgreSQL/*/bin/psql.exe 2>/dev/null | sort -V | tail -1 || true)"
fi
if [[ -z "$PSQL" ]]; then
  echo "error: psql not found (not on PATH, not under C:/Program Files/PostgreSQL/*/bin)" >&2
  exit 1
fi
[[ -f "$CONFIG" ]] || { echo "error: config not found: $CONFIG (copy application.yml.example)" >&2; exit 1; }
[[ -d "$MIG_DIR" ]] || { echo "error: no migrations directory at $MIG_DIR" >&2; exit 1; }

# "${VAR}" placeholders are filled like server/config_env.go does: a real
# environment variable wins, otherwise config/.env (next to the yml), else "".
# Values are never printed.
ENV_FILE="$(dirname "$CONFIG")/.env"
dotenv_get() {
  [[ -f "$ENV_FILE" ]] || return 0
  awk -v want="$1" '
    { sub(/\r$/, ""); sub(/^\xef\xbb\xbf/, ""); line = $0
      sub(/^[ \t]+/, "", line); if (line == "" || line ~ /^#/) next
      sub(/^export[ \t]+/, "", line)
      i = index(line, "="); if (i == 0) next
      k = substr(line, 1, i - 1); gsub(/[ \t]+$/, "", k); if (k != want) next
      v = substr(line, i + 1); gsub(/^[ \t]+|[ \t]+$/, "", v)
      if (length(v) >= 2 && (v ~ /^".*"$/ || v ~ /^'\''.*'\''$/)) v = substr(v, 2, length(v) - 2)
      print v; exit }' "$ENV_FILE"
}
expand_placeholders() {
  local v="$1" name val n=0
  while [[ $v =~ \$\{([A-Za-z_][A-Za-z0-9_]*)\} ]] && (( n++ < 20 )); do
    name="${BASH_REMATCH[1]}"
    if [[ -n "${!name+x}" ]]; then val="${!name}"; else val="$(dotenv_get "$name")"; fi
    v="${v//\$\{$name\}/$val}"
  done
  printf '%s' "$v"
}

# Read the top-level `database:` block the way the app's YAML decoder does:
# drop a trailing " #comment" and one pair of surrounding quotes.
while IFS='=' read -r k v; do
  v="$(expand_placeholders "$v")"
  case "$k" in
    host) export PGHOST="$v" ;;
    port) export PGPORT="$v" ;;
    user) export PGUSER="$v" ;;
    password) export PGPASSWORD="$v" ;;
    dbname) export PGDATABASE="$v" ;;
    sslmode) export PGSSLMODE="$v" ;;
  esac
done < <(awk '
  /^database:/ { inblock = 1; next }
  /^[^ #]/     { inblock = 0 }
  inblock && match($0, /^  (host|port|user|password|dbname|sslmode):[ ]*/) {
    key = $0; sub(/^  /, "", key); sub(/:.*/, "", key)
    val = substr($0, RLENGTH + 1); sub(/\r$/, "", val)
    if (val ~ /^".*"$/ || val ~ /^'\''.*'\''$/) { val = substr(val, 2, length(val) - 2) }
    else { sub(/[ \t]+#.*$/, "", val); sub(/[ ]+$/, "", val) }
    print key "=" val
  }' "$CONFIG")

for var in PGHOST PGPORT PGUSER PGDATABASE; do
  [[ -n "${!var:-}" ]] || { echo "error: database.${var#PG} missing in $CONFIG" >&2; exit 1; }
done
export PGCONNECT_TIMEOUT=10

psqlq() { "$PSQL" -X -q -v ON_ERROR_STOP=1 "$@"; }

echo "Target:     $PGUSER@$PGHOST:$PGPORT/$PGDATABASE (password from config, not shown)"
echo "Migrations: $MIG_DIR"
if ! psqlq -tAc "select 1" >/dev/null; then
  echo "error: cannot connect to the database above" >&2
  exit 1
fi

applied=()
if [[ "$(psqlq -tAc "select to_regclass('public.ecom_schema_migrations') is not null")" == "t" ]]; then
  mapfile -t applied < <(psqlq -tAc "select filename from public.ecom_schema_migrations order by filename" | tr -d '\r')
fi

pending=()
while IFS= read -r f; do
  name="$(basename "$f")"
  if [[ ! "$name" =~ ^[0-9A-Za-z_.-]+\.sql$ ]]; then
    echo "error: unsafe migration filename '$name' (use letters, digits, _ . -)" >&2
    exit 1
  fi
  if [[ " ${applied[*]-} " != *" $name "* ]]; then
    pending+=("$f")
  fi
done < <(ls -1 "$MIG_DIR"/*.sql 2>/dev/null | sort)

echo
echo "Applied (${#applied[@]}):"
[[ ${#applied[@]} -gt 0 ]] && printf '  %s\n' "${applied[@]}" || echo "  (none)"
echo "Pending (${#pending[@]}):"
if [[ ${#pending[@]} -eq 0 ]]; then
  echo "  (none) — database is up to date"
  exit 0
fi
for f in "${pending[@]}"; do echo "  $(basename "$f")"; done

if [[ "$EXECUTE" -ne 1 ]]; then
  echo
  echo "Dry run only. Re-run with --execute to apply the pending files above."
  exit 0
fi

echo
psqlq -c "SET client_min_messages = warning;
          CREATE TABLE IF NOT EXISTS public.ecom_schema_migrations (
            filename   TEXT PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
          );"

for f in "${pending[@]}"; do
  name="$(basename "$f")"
  echo "==> applying $name"
  if ! psqlq --single-transaction -f "$f" \
       -c "INSERT INTO public.ecom_schema_migrations (filename) VALUES ('$name')"; then
    echo "FAILED: $name — rolled back; later files were not applied." >&2
    exit 1
  fi
  echo "    done"
done
echo
echo "Applied ${#pending[@]} migration(s)."
