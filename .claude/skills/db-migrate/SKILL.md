---
name: db-migrate
description: Apply the backend's SQL migrations (workspace/backend/migrations/*.sql) to the Postgres database configured in config/application.yml, tracking applied files in public.ecom_schema_migrations so each runs exactly once. Always dry-runs and gets confirmation before executing. Also covers writing a new migration file. Use when asked to run/apply migrations, create tables, check migration status, or add a schema change to the backend.
---

Applying a migration changes a shared database and cannot be undone by
re-running anything, so always preview it, confirm it, then execute.

## Running migrations

1. **Dry run** from the workspace root:
   ```bash
   bash .claude/skills/db-migrate/migrate.sh
   ```
   It prints the target (`user@host:port/db`, never the password), the files
   already applied, and the pending files. Nothing is written.
   - If it cannot connect, report the real `psql` error. Don't edit
     `config/application.yml` to work around it unless the developer asks.
   - `--config <file>` reads the `database:` block from a different YAML,
     for example a staging config.

2. **Show the developer the target and pending list, and ask for explicit
   confirmation.** Point out if the target looks like a shared or production
   host rather than `127.0.0.1`/`localhost`.

3. **Only after confirmation**:
   ```bash
   bash .claude/skills/db-migrate/migrate.sh --execute
   ```
   Each file runs in a single transaction together with its
   `public.ecom_schema_migrations` row. On failure, that file is rolled back,
   later files are not attempted, and the next run retries it.

4. **Report the result as printed.** On failure, quote the SQL error and fix
   the migration file. It was rolled back, so editing it is safe.

## What's in migrations/

- `000_baseline.sql` covers the service's original tables (`market_token`,
  `market_place_mapping`, `log_market_place_order`, `log_order_status`,
  `ecom_statement_transaction`), which all live in `public`.
  - Step 1 moves them out of the old `ecommerce` schema with
    `ALTER TABLE … SET SCHEMA public`, which keeps data, indexes and
    sequences. It aborts, and moves nothing, if a table exists in both
    schemas. **This breaks any code or other system still querying
    `ecommerce.*`**, so deploy the matching backend at the same time. Before
    running it on a shared DB, tell the developer the old schema will be
    emptied.
  - Step 2 creates the tables if they are missing, using `IF NOT EXISTS`
    with inline constraints. The column definitions were reconstructed from
    the code, not dumped from a live DB.
- `001_create_ecom_order.sql` creates `public.ecom_order` and `public.ecom_order_item`.
- `002`–`005` are the admin app's auth and RBAC (EC-1, EC-3, EC-4): `app_user`,
  `user_session`, `oauth_login_attempt`, `role`, `permission`, `role_permission`,
  the default admin and `orders.detail`. See the `admin-auth` skill.
- `006_user_config.sql` creates `public.user_config`, per-user settings 1-1 with
  `app_user` (`user_id` is PK and FK, `ON DELETE CASCADE`), starting with
  `locale` (`vi`|`en`, EC-7), and backfills a row for every existing user. Code
  reads a missing row as `vi`.
- `dev/connector_tables.sql` holds minimal stand-ins for the Connector's
  `public.od_ecom_order`, `od_do_new` and `od_sale_order`. It is **local
  only**: `migrate.sh` never applies subdirectories. Load it with `psql -f`
  into a throwaway DB.

## Testing against a throwaway Postgres

The repository integration tests
(`component/repositories/repositories_integration_test.go`) run every repo
query against a real schema. They are skipped unless `ECOM_TEST_DATABASE_URL`
is set, and they refuse to run unless the DB name contains `test`, because
they TRUNCATE tables. No existing server or password is needed:

```bash
PG="/c/Program Files/PostgreSQL/18/bin"; D=<scratch dir>
"$PG/initdb.exe" -D "$D/pgdata" -U postgres -A trust -E UTF8
"$PG/pg_ctl.exe" -D "$D/pgdata" -o "-p 55432 -c listen_addresses=127.0.0.1" -l "$D/pg.log" -w start
"$PG/psql.exe" -h 127.0.0.1 -p 55432 -U postgres -c "create database mq_test"
"$PG/psql.exe" -h 127.0.0.1 -p 55432 -U postgres -d mq_test -f workspace/backend/migrations/dev/connector_tables.sql
printf 'database:\n  host: 127.0.0.1\n  port: 55432\n  user: postgres\n  password:\n  dbname: mq_test\n  sslmode: disable\n' > "$D/test.yml"
bash .claude/skills/db-migrate/migrate.sh --config "$D/test.yml" --execute
cd workspace/backend && ECOM_TEST_DATABASE_URL="postgres://postgres@127.0.0.1:55432/mq_test?sslmode=disable" go test ./component/repositories/ -v
"$PG/pg_ctl.exe" -D "$D/pgdata" stop      # when done
```

Run this after any migration or repository query change. For a schema
change, also extend the matching test.

## Writing a new migration

- **Name:** `NNN_short_description.sql` in `workspace/backend/migrations/`,
  with NNN one higher than the current highest (`006_user_config.sql`
  today). Files apply in filename order. Only letters, digits, `_ . -` are
  accepted.
- **Never edit a file that has been applied.** Its name is already recorded,
  so the change would never run. Write a new file instead.
- Make statements re-runnable where Postgres allows it (`CREATE TABLE IF NOT
  EXISTS`, `CREATE INDEX IF NOT EXISTS`, `ADD COLUMN IF NOT EXISTS`), so a
  database that was patched by hand doesn't fail.
- Don't put `BEGIN`/`COMMIT` in the file. The runner already wraps it.
  `CREATE INDEX CONCURRENTLY` cannot run inside a transaction, so put a
  normal `CREATE INDEX` in the migration and tell the developer if the table
  is large.
- Every table lives in `public`. Don't create anything in the old
  `ecommerce` schema. Prefix new tables with `ecom_` (`ecom_order`,
  `ecom_order_item`, `ecom_schema_migrations`). The admin app's account tables
  (`app_user`, `user_session`, `oauth_login_attempt`, `role`, `permission`,
  `role_permission`, `user_config`) are the existing exception; new per-user
  settings go in `user_config` as columns, not in a new table. `public` is shared with the
  Connector (formerly TMS) system, so check that the name is free
  (`\dt public.<name>`) and never alter its `od_*` tables. Always
  schema-qualify names (`public.x`) in migrations and repository SQL.
- Add a header comment saying what the migration is for. Then update the
  repo in `component/repositories/` and the model in `component/models/`
  that read or write the table, and run the `backend-check` skill.
