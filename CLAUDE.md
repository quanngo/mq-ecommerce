# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A fullstack workspace template (bash scripts + Claude Code skills) wrapped around MQ's e-commerce integration service. Projects are declared in `projects.env`, cloned under `workspace/frontend/<key>/` and `workspace/backend/<key>/` (or placed by `*_DIR_<N>`), and run through PM2.

> `./setup.sh` step 3 **regenerates this file** from the per-project `CLAUDE.md`s and `.claude/rules/`. Anything you add here by hand will be overwritten on the next setup run. Put lasting conventions in `.claude/rules/` instead.

### Current state

- Both projects live **directly in this repo**, not as clones: `projects.env` declares `admin` → `workspace/frontend/` and `api` → `workspace/backend/` with `*_DIR_<N>=.` and an empty `*_REPO_<N>`. `project_dir` (bash) and the git-pull/push/current-branch Python scripts honour `*_DIR_<N>`; `setup.sh` skips cloning a project with no repo. Neither folder has its own `.git`, so the git-* skills report them as "not a git repo" (commit at the workspace root instead).
- `workspace/frontend/` is the admin app (React 18 + Vite, plain JavaScript/JSX, created for EC-1). It has login (Google + email/password; the Google button is always shown, greyed out until `auth.google` is configured), profile, change password, user management (`/users`) the Orders menu (`/orders` list + `/orders/:id` detail, EC-4) and Product images (`/product-images`, EC-10: upload images straight to TikTok/Lazada, keep the returned URLs). The UI is in Vietnamese and English (EC-7: `react-i18next`, `src/locales/{vi,en}.json`, VI/EN toggle; the user's choice is saved in `user_config.locale`, a 1-1 settings table next to `app_user`). It talks to the backend through the Vite `/api` proxy. Run it with `npm install && npm run dev` on the **fixed port :8088** (must match `auth.frontend-url`) and test with `npm test`. See its README and the `admin-auth` skill.
- The backend has no `CLAUDE.md` and an empty README. Apply `migrations/` with the `db-migrate` skill before starting it. `000_baseline.sql` moves the legacy `ecommerce.*` tables into `public` (see the `db-migrate` skill before running it on a shared DB), and its column definitions are reconstructed from code, not dumped from a live DB.

## Workspace commands

```bash
./setup.sh                    # clone/pull projects, run /init per project, split into .claude/rules/, regenerate root CLAUDE.md
./start.sh [frontend|backend|<key>]   # writes ecosystem.config.js from projects.env, then pm2 start
pm2 list | pm2 logs <key> | pm2 restart <key> | pm2 delete all

./deliver.sh <TICKET>         # Jira ticket -> plan pipeline (headless claude -p per phase)
./deliver.sh <TICKET> --status | --resume | --from plan | --yes
```

- `scripts/lib.sh` is shared by all three scripts: `load_projects` (sources and validates `projects.env`, keys must be unique across frontend and backend), `for_each_project`, `project_dir`, and `CLAUDE_ALLOWED_TOOLS`, the deliberately narrow tool allowlist for `deliver.sh` phases.
- Optional per-project `FRONTEND_REPO_INIT_<N>` / `BACKEND_REPO_INIT_<N>` commands run after clone or pull.
- `deliver.sh` phases are `ticket → plan → [approval gate] → backend → frontend → qa → report`. Only `ticket` (`/jira-ticket`) and `plan` (`/plan-ticket`) are implemented. The rest return "not implemented (Stage 2/3)". Phase state is kept in `.workflow/<TICKET>/progress.json` and is read and written **only** through `scripts/workflow_state.py`; bash never parses that JSON itself. When you add a phase, update `PHASE_NAMES`/`PHASE_SKILLS` in `deliver.sh`, `PHASES` in `workflow_state.py`, and widen `CLAUDE_ALLOWED_TOOLS`.
- Workspace skills live in `.claude/skills/` (admin-auth, write-code-full/fe/be, backend-api-docs, git-pull/push/current-branch, jira-ticket, plan-ticket, workspace-init). The backend-side skills expect each backend to keep an `API_DOCUMENT.md`.

## Backend (`workspace/backend/`): Go marketplace order sync service

Go 1.24, Gin, sqlx + lib/pq (Postgres), robfig/cron, logrus, swaggo. The module is named `main`, so imports look like `main/component/services`.

```bash
cd workspace/backend
go mod tidy                  # required first: go.sum is gitignored, and the build fails with "missing go.sum entry" until you run this
go run main.go               # API server on :9091 plus the cron workers in the same process
go run main.go run-job       # one-shot: refresh Lazada/TikTok tokens, then run the Google Sheet import
go build ./... && go vet ./...
go test ./... -run TestName  # unit tests: component/services, server
ECOM_TEST_DATABASE_URL=postgres://postgres@127.0.0.1:55432/mq_test?sslmode=disable go test ./component/repositories/  # repo integration tests; see .claude/skills/db-migrate (skipped without the var)
swag init                    # regenerate docs/ (swagger), served at /swagger/index.html
```

Running it needs a reachable Postgres (`config/application.yml` → `database`, default `127.0.0.1:5555`). `internal.ConnectPgDatabase` panics if the ping fails.

### Big picture

The service polls the marketplaces (TikTok Shop, Lazada; Shopee is partial) for shop orders and stores them as orders in `public.ecom_order` / `ecom_order_item` (schema in `migrations/`). With `connector.enable: true` it also forwards them to the external **Connector** (formerly TMS) order/delivery system. It also syncs status changes, refunds, to-ship/AWB, payments, and settlement statements back into Postgres. **Most of the work is done by cron workers, not by HTTP endpoints.**

- **Entry point.** `main.go` uses `server.CommandManager` and dispatches on `os.Args[1]`: `""` starts the server and workers, `run-job` runs the one-shot job.
- **Wiring.** `server/base.go` holds `Initialize()` (API), `InitializeWorkers()` (cron) and `InitializeJob()` (one-shot). Each one wires config → DB → repos → `ApiMiddleware` → services by hand. The three functions duplicate this wiring, so a new dependency must be added to each one that needs it.
- **Config.** `initConfig()` decodes **every `*.yml` in `./config`** into one `utils.Config`, stored in the global `utils.AppConfig`. That is how `workers.yml` supplies `Config.Workers`. Paths are relative to the cwd, so run the service from `workspace/backend/`. **Secrets are not written in the yml.** A value like `"${TIKTOK_APP_SECRET}"` is filled in after decoding from an environment variable, or from `config/.env` (gitignored; template in `config/.env.example`). Real environment variables win over `config/.env`. See `server/config_env.go`. Add any new secret the same way. `config/.env` and `config/service-account.json` contain live credentials. Don't print or copy them.
- **Workers.** In `server/workers.go`, `workers.yml` entries (`name`, `time-run` cron/`@every`, `function`, `enable`) map by the `function` string to methods in `mappingWorkers()`. To add a job, implement it on a service, register it in `mappingWorkers()`, and add a `workers.yml` entry. Most of the jobs are in `services/platform_service.go`, which fans out per shop to the TikTok and Lazada services.
- **The `middleware` package is mostly outbound API clients, not Gin middleware.** `ApiMiddleware` (`middleware/base.go`) holds config and `TokenRepo`. `tiktok_auth.go`, `lazada_auth.go`, `shopee_auth.go` and `connector_auth.go` implement request signing, token refresh and typed calls to each external API (`DoTiktokPost/GET`, `DoLazadaPost/Get`, `DoConnectorPOST`). The only real Gin middleware is `require_auth.go` (`RequireAuth`, `RequireSameOrigin`). The old no-op `AuthMiddleWare()` was removed.
- **Layers** (`component/`):
  - `services/`: interface plus `*Impl` per domain. `tiktok`, `lazada`, `platform` (orchestration), `connector` (`CreateLead` / `UpdateDOStatus`: always write `public.ecom_order` via `EcomOrderRepo`, then call the Connector host/callback only if `connector.enable`), `logistic` (inventory check, gated by `logistic.enable`), `import` (reads a Google Sheet via the service account and creates leads).
  - `repositories/`: raw SQL with sqlx, all against `public`: the service's own tables (`market_token`, `market_place_mapping`, `log_market_place_order`, `log_order_status`, `ecom_statement_transaction`, `ecom_order`, `ecom_order_item`) and the Connector's `od_ecom_order` / `od_do_new` / `od_sale_order` (read, plus payment columns updated). These were previously in an `ecommerce` schema.
  - `models/`: shared DTOs, plus per-platform packages (`tiktok_model`, `lazada_model`, `shopee_model`).
  - `auth` (EC-1): `AuthService` handles login to the admin app (Google OIDC plus internal email/password) and server-side sessions in the httpOnly `ecom_session` cookie. Routes are under `/api/v1/auth`, `middleware.RequireAuth` protects `/tiktok` and `/lazada`, and users are created with `go run main.go create-user`. Roles and permissions are stored in the DB (`role`, `permission`, `role_permission`, one role per user). Guard routes with `middleware.RequirePermission(models.Perm…)` and never check role names. `UserAdminService` serves `/api/v1/users` and `/api/v1/roles`. `OrderService` (+ `OrderQueryRepo`, read-only) serves `GET /api/v1/orders` (`orders.read`) and `GET /api/v1/orders/:id` (`orders.detail`, migration `005`). The full API is in `workspace/backend/API_DOCUMENT.md`. Migration `004` seeds a default admin, **`admin@mq.com` / `123456`**, which must be changed after the first login (the backend logs a warning until it is). Read the `admin-auth` skill before touching any of this.
  - Product images (EC-10): `ProductImageService` + `ProductImageRepo` + `middleware/product_image.go` (multipart upload to TikTok `/product/202309/images/upload` and Lazada `/image/upload` with the shop token from `market_token`). `GET/POST /api/v1/product-images`, `GET /api/v1/product-images/shops`, permission `orders.write`. The file is not stored; `public.ecom_product_image` (migration `007`) keeps the returned `url` (+ TikTok `uri`).
  - `i18n` (EC-7, package `main/i18n`): `vi`/`en` catalogs in `i18n/locales/*.json`. `middleware.Locale()` reads `Accept-Language`, `RequireAuth` switches to the user's `locale` (`user_config`, migration `006`, `PUT /api/v1/auth/me/locale`). Error responses add `error_text` in that language; role/permission names are translated. Error codes in `message` never change.
  - `handlers/rest/`: routes under `/api/v1/{tiktok,lazada}`. Their `Router()`s currently register nothing. `ShopeeHandler` (POST `webhook`) exists but isn't mounted in `rest.APIRoutes`, and no `ShopeeService` is constructed.
- **Domain enums** live in `utils/constants.go`: `Plaform` (sic) is `"Lazada" | "Tiktok" | "Shopee"`, `LeadStatus` is 1–4, `DeliveryOrderStatus` is 51–60+. Use these rather than literals.
- **Response convention** (`component/api/response.go`): `api.Ok` / `api.Error` / `api.ErrorWithMessage` always reply **HTTP 200** and put the real status in the body `{code, message, data}`. `api.Ok` is variadic, so `data` is serialized as an array.
- **Logging**: use `utils.ShowInfoLogs` / `utils.ShowErrorLogs` (logrus).
- **Kafka**: `internal/kafka.go` is fully commented out and `kafka.enabled: false`.
- **Dockerfile** is stale. It uses `golang:1.19` while `go.mod` needs 1.24, and it copies `public/`, `view/` and `files/` directories that don't exist.
