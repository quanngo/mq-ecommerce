---
name: write-code-full
description: Implement a fullstack feature end to end in this workspace. The Go backend is in workspace/backend (Gin + sqlx, migrations, the always-HTTP-200 response envelope); the admin app is in workspace/frontend (React 18 + Vite, plain JavaScript/JSX, not TypeScript). Clarify schema, contract and UI placement, then implement both sides, wire the frontend to the real contract, write unit tests on both sides, update API_DOCUMENT.md, add e2e coverage, and run the checks. Use when asked to build a feature or "add X to the app" spanning backend and frontend.
---

Implement the feature end to end, grounded in what the code actually shows.
Never invent schema, an API shape or a convention. Ask the developer rather than guess.

## Steps

1. **Restate the requirement** in your own words as a first check that you understood it.

2. **Read both sides.**
   - Root `CLAUDE.md`: architecture, the backend's layering, the response convention.
   - `workspace/frontend/CLAUDE.md`: frontend stack, layout, conventions and
     tests. The frontend is **plain JavaScript**. Never add TypeScript or
     `.ts`/`.tsx` files.
   - The backend code lives **directly in `workspace/backend/`**, not
     `workspace/backend/<key>/`, whatever `projects.env` says. Read the
     handler, service and repository files near the feature, and `API_DOCUMENT.md`
     if it exists.
   - If the feature touches login or permissions, also read the `admin-auth` skill.

3. **Ask before coding** if any of these are unclear:
   - where it lives in the UI (new or existing page; whether it needs login);
   - schema changes (tables, columns, nullability, constraints, indexes);
   - the API contract (method, path, request and `data[0]` shape, validation,
     error codes, auth);
   - the technical approach, when the existing architecture allows more than one.

   Once those are answered, go straight to implementation.

4. **Schema**: write a new numbered file in `workspace/backend/migrations/`
   (`NNN_<what>.sql`, `CREATE … IF NOT EXISTS`, commented like the existing ones),
   following the `db-migrate` skill. Never edit a migration that has already been
   applied. Don't apply it to a shared DB without the developer's confirmation
   (the skill dry-runs first). Non-pointer Go fields need `NOT NULL`
   (see the header of `000_baseline.sql`).

5. **Backend** (`workspace/backend/`), matching the existing layers:
   - **Model**: `component/models/<area>_model.go`, with `db` and `json` tags
     (snake_case JSON). Response DTOs never carry secrets (`json:"-"`).
   - **Repository**: a concrete struct over `*sqlx.DB` with raw SQL against `public`.
     A missing row returns `(nil, nil)`.
   - **Service**: an interface plus `*Impl`. It depends on small private store
     interfaces, not concrete repos (as `ecomOrderStore` does in
     `connector_service.go`), so it can be tested with fakes. Errors carry a
     stable code where the frontend needs to tell them apart (pattern:
     `services/auth_errors.go`).
   - **Handler**: `component/handlers/rest/<area>_handler.go`, mounted in
     `rest.go`. Respond only with `api.Ok` / `api.ErrorWithMessage` (HTTP 200, real
     status in `code`, error code in `message`; `error_text` is added from the
     `error.<CODE>` keys in `i18n/locales/{vi,en}.json`, so add both for a new
     code). Text the user reads (names, labels) goes through `i18n.T(i18n.Locale(c), key)`. Never echo `err.Error()` for a bind
     error. Put routes that need login in a group built with `requireAuth`, and add
     `sameOrigin` to state-changing routes.
   - **Wiring**: `server/base.go`. Add the new dependency to each of `Initialize()`,
     `InitializeWorkers()` and `InitializeJob()` that needs it. A cron job goes
     through the `add-worker` skill.
   - **Logging**: `utils.ShowInfoLogs`/`ShowErrorLogs`. Never log tokens, passwords
     or whole token or credential structs.

6. **Frontend** (`workspace/frontend/`), as in `write-code-fe` step 6:
   - API functions go in `src/api/<area>.js` and call `apiRequest`.
   - The route goes in `App.jsx`, inside `ProtectedRoute` if it needs login.
   - The page goes in `src/pages/<Name>Page.jsx` with a CSS module that uses the
     `var(--…)` tokens.
   - All UI text goes in `src/locales/vi.json` **and** `en.json` (`t(...)`);
     error codes map to messages there (`errors.<context>.<CODE>`).
   - Backend text the user reads (new error codes, role/permission names) goes
     in `i18n/locales/vi.json` **and** `en.json`; never return translated text
     in place of a stable error code.
   - Nothing goes in `localStorage`/`sessionStorage` except the UI language code.

7. **Integrate**: match the frontend to the backend you just wrote, verified
   against the code, not assumed:
   - the path is relative to `/api/v1`;
   - field names are the Go `json` tags;
   - `data` is always an array, so `apiRequest` returns `data[0]`;
   - every error `message` code the backend can return is handled.

8. **Unit tests on both sides** (existing tools only):
   - **Backend**: `go test`. Write service tests with fake stores (style:
     `connector_service_test.go`, `auth_service_test.go`), and handler tests with
     gin and `httptest` through `APIRoutes`, asserting the body `code`/`message`.
     Repository SQL gets a case in `repositories_integration_test.go` (gated by
     `ECOM_TEST_DATABASE_URL`; add new tables to `allTables`).
   - **Frontend**: Vitest + MSW. Mirror the new endpoint in `src/test/server.js`
     with the real envelope, and test through `renderApp('/route')`.

9. **API docs**: follow `.claude/skills/backend-api-docs/SKILL.md`. Re-scan the
   routes and create or update `workspace/backend/API_DOCUMENT.md`, keeping
   hand-written prose. Add swag annotations on new handlers like the ones in
   `auth_handler.go`.

10. **E2E**: add `workspace/frontend/e2e/<feature>.spec.js` (Playwright)
    for the main user flow against the real backend. It should skip itself when
    the env vars or test data it needs are missing, and its header comment should
    say how to seed them.

11. **Verify, then report the real results** (never claim a check you didn't run):
    - Backend: `bash .claude/skills/backend-check/check.sh` (tidy, gofmt on
      changed files, build, vet, test, secret scan).
    - Frontend: `npm test && npm run build` in `workspace/frontend/`.
    - DB integration and e2e tests: say whether they ran. They need a test
      Postgres and a running stack.

    Then summarise:
    - the files changed per side;
    - any migration still to apply;
    - the tests added and their results;
    - whether `API_DOCUMENT.md` changed;
    - open questions.
