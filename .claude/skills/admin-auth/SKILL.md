---
name: admin-auth
description: Work with admin-app login (EC-1) and roles/permissions. The Go backend runs Google OIDC plus internal email/password with server-side sessions in an httpOnly cookie, and the React (JS) app lives in workspace/frontend. Covers protecting a new backend endpoint with RequireAuth / RequirePermission, reading the signed-in user and their permissions, the role model (admin/operator/viewer, default admin@mq.com) and permission codes (users.read/manage, orders.read/detail/write), the user-management page and API, the Orders menu as the reference permission-guarded feature, creating users with the create-user CLI, configuring Google per environment (allowed-domains / auto-provision), the per-user UI language (vi/en in user_config, PUT /auth/me/locale, translated error_text and role/permission names, EC-7), adding an auth error code end to end in both languages, the security rules that must not regress, and how to test. Use when asked to require login or a permission on an endpoint, add a permission or role, add, disable or reset a user, set up Google sign-in for an environment, change who may log in, debug a login failure, change how the UI language is saved, or touch any auth code.
---

Login is a security boundary. Read the files named below before changing
anything, and keep every rule in **Rules**. Each one is a requirement of
the EC-1 acceptance criteria, and most are enforced by a test.

## Where things are

| Concern | File (relative to `workspace/backend/` unless noted) |
|---|---|
| Schema | `migrations/002_create_auth.sql`: `app_user`, `user_session`, `oauth_login_attempt`; `003_auth_remember_me.sql`: `oauth_login_attempt.remember_me`; `004_rbac_and_default_admin.sql`: `role`, `permission`, `role_permission`, `app_user.role_id`, default admin; `005_orders_detail_permission.sql`: `orders.detail` for every seeded role; `006_user_config.sql`: `user_config` (1-1 with `app_user`, `locale`, EC-7) |
| Roles & users | `component/services/user_admin_service.go` (`UserAdminService`), `component/repositories/role_repo.go`, `component/handlers/rest/user_admin_handler.go` (`/api/v1/users`, `/api/v1/roles`), permission codes in `component/models/auth_model.go` (`PermUsersRead`…) |
| Config | `utils/config.go` `AuthConfig` / `GoogleAuthConfig` + `WithDefaults()`; documented in `config/application.yml.example` (`auth:` block) |
| Business logic | `component/services/auth_service.go` (`AuthService`), `auth_errors.go` (stable error codes), `google_oidc.go` (Google behind the `OIDCClient` interface) |
| SQL | `component/repositories/user_repo.go` (`appUserColumns` also reads `user_config.locale`; `SetLocale` upserts it), `session_repo.go`, `login_attempt_repo.go` |
| Language (EC-7) | `i18n/i18n.go` + `i18n/locales/{vi,en}.json` (error texts, role/permission names), `component/api/response.go` (`error_text`), `component/handlers/rest/localize.go` |
| HTTP | `component/handlers/rest/auth_handler.go`, mounted in `rest.go` under `/api/v1/auth` (the whole `/api/v1` group runs `middleware.Locale()`) |
| Middleware | `middleware/require_auth.go`: `RequireAuth` (also sets the user's locale), `RequirePermission`, `CurrentUser`, `RequireSameOrigin`, `Locale` |
| Wiring | `server/base.go` `Initialize()` (`newAuthService`, `newGoogleClient`), `InitializeWorkers()`; `server/create_user.go`; `main.go` |
| Cleanup job | `cleanupAuthSessions` in `server/workers.go` + `config/workers.yml` |
| Frontend | `workspace/frontend/src/`: `api/client.js`, `api/auth.js`, `api/users.js`, `auth/AuthContext.jsx` (`login`/`logout`/`logoutAll`/`can`/`setLanguage`), `auth/authChannel.js` (cross-tab sync), `auth/ProtectedRoute.jsx` (`permission` prop), `auth/errorMessages.js` (`authErrorMessage`/`actionErrorMessage`/`orderErrorMessage`/`permissionLabel`), `i18n/` + `locales/{vi,en}.json` (EC-7), `pages/LanguageSwitcher.jsx`, `pages/AppLayout.jsx` (top bar, menu, logout), `pages/LoginPage.jsx`, `pages/ProfilePage.jsx`, `pages/ChangePasswordPage.jsx`, `pages/UsersPage.jsx`, `pages/ConfirmDialog.jsx`, `pages/Modal.jsx` |
| Permission-guarded feature (reference) | Orders menu (EC-4): `component/handlers/rest/order_handler.go` (per-route `RequirePermission`), `component/services/order_service.go`, `component/repositories/order_query_repo.go`; FE `api/orders.js`, `pages/OrdersPage.jsx`, `pages/OrderDetailPage.jsx`, routes in `App.jsx`, menu item in `AppLayout.jsx`. API: `workspace/backend/API_DOCUMENT.md` |
| Design | `.workflow/EC-1/design.md` (if present locally; `.workflow/` is gitignored) |

Endpoints: `GET providers`, `GET google/login[?next=&remember=1]`, `GET google/callback`,
`POST login` (`{email, password, remember_me}`), `GET me` (needs a session),
`POST logout` (this browser), `POST logout-all` (needs a session; revokes every
session of the user and returns `{revoked_sessions}`), and `POST change-password`
(`{current_password, new_password}`; signs out the user's other sessions), and
`PUT me/locale` (`{locale: "vi"|"en"}`; session, same origin). All are under
`/api/v1/auth`. JSON responses follow the repo convention: always HTTP 200,
the real status in `code`, the error code in `message`, `error_text` (the error in
the request's language) on failures, and `data` as an array.
`/auth/me` and the login response include `role {code, name}`, `permissions []`,
`permission_names {code: name}` and `locale`; names are in the user's language.

User management, all needing a session:

| Method | Path | Permission | Notes |
|---|---|---|---|
| GET | `/api/v1/users` | `users.read` | Returns `{users}`. |
| GET | `/api/v1/roles` | `users.read` | Returns `{roles}` with their permissions. |
| POST | `/api/v1/users` | `users.manage` + same origin | Body `{email, display_name, role, password?}`. Leave out `password` for a Google-only account. |
| PATCH | `/api/v1/users/:id` | `users.manage` + same origin | Body `{display_name?, role?, status?}`. |
| POST | `/api/v1/users/:id/reset-password` | `users.manage` + same origin | Body `{password}`. Signs the user out everywhere. |

### Roles and permissions

- **Model:** each user has exactly one role (`app_user.role_id`). A role grants
  permission codes through `role_permission`. **Code checks permissions, never role
  names:**
  - backend: `middleware.RequirePermission(models.PermUsersManage)` or `user.Can(...)`;
  - frontend: `useAuth().can('users.manage')` or `<ProtectedRoute permission=…>`.
- **Permission codes** (`models.Perm…` in `auth_model.go`; names in `permission.<code>` of the backend catalogs `workspace/backend/i18n/locales/*.json`, sent as `permission_names`, and in `permissions.<area>.<action>` of the FE `src/locales/*.json` as fallback):
  | Code | Constant | Guards |
  |---|---|---|
  | `users.read` | `PermUsersRead` | `GET /users`, `GET /roles`; FE menu "Người dùng", `/users` |
  | `users.manage` | `PermUsersManage` | `POST /users`, `PATCH /users/:id`, `POST /users/:id/reset-password`; FE create/role/lock/reset controls |
  | `orders.read` | `PermOrdersRead` | `GET /orders` (list); FE menu "Đơn hàng", `/orders` |
  | `orders.detail` | `PermOrdersDetail` | `GET /orders/:id` (detail, with phone/address/items); FE `/orders/:id` and the order-id link in the list |
  | `orders.write` | `PermOrdersWrite` | nothing yet (reserved for order updates) |
- **Seeded** by `004` + `005`, all `is_system`:
  | Role | Permissions |
  |---|---|
  | `admin` (Quản trị viên) | all five |
  | `operator` (Vận hành) | `orders.read`, `orders.detail`, `orders.write` |
  | `viewer` (Chỉ xem) | `orders.read`, `orders.detail` |

  `orders.detail` is separate from `orders.read` so a role can see the list
  without opening details; today every seeded role has both (developer's
  choice, EC-4). To take it from a role, delete its `role_permission` row.
- **Adding a permission:**
  1. Write a new migration that inserts into `permission` and `role_permission`
     (`005_orders_detail_permission.sql` is the template).
  2. Add a `Perm…` constant in `auth_model.go`.
  3. Add its name in both languages: `permission.<code>` in `workspace/backend/i18n/locales/{vi,en}.json` and `permissions.<area>.<action>` in `workspace/frontend/src/locales/{vi,en}.json` (key-parity tests fail otherwise).
  4. Guard the route with `RequirePermission` and the FE route with
     `<ProtectedRoute permission=…>`; hide links/menu items with `can(…)`.
  5. Update the role counts in `TestRBACRepos`, the role lists in
     `src/test/server.js` and `mock/dev-backend.js`, and the tables above.

  There is no UI for editing roles. Change them with SQL or a migration.
- **Rules enforced by `UserAdminService`:**
  - Nobody can change their own role or status, or reset their own password,
    through user management (`CANNOT_MODIFY_SELF`). Use "Đổi mật khẩu" for your
    own password.
  - The last active user with `users.manage` can't be demoted or disabled (`LAST_ADMIN`).
  - Disabling a user or resetting their password revokes all their sessions.
- **Default role** for auto-provisioned Google users and `create-user` without
  `--role` is `auth.default-role` (default `viewer`). Users that existed before `004`
  became `viewer`.
- **Default admin:** `004` seeds `admin@mq.com` / `123456` in **every** environment
  (requested by the developer). It is not created again if the account already
  exists. Every start logs `[SECURITY] admin@mq.com still uses the default password…`
  until the password is changed. Tell whoever deploys to change it right after the
  first login.
- **Password policy:** `auth.min-password-length` (default 10, up to 72 bytes) applies
  to every new or changed password. The seeded admin is exempt only because its hash
  is written directly by the migration.

### Session lifetime and logout behaviour

- **Remember me** (`remember_me` on password login, `remember=1` on
  `google/login`):
  - When ticked, the session lasts `auth.remember-me-days` (default 7) and the cookie
    gets a matching `Max-Age`.
  - Otherwise the session lasts `auth.session-ttl-hours` (default 12) server-side, and
    the cookie has **no** `Max-Age`, so it dies when the browser closes.
  - For Google, the choice is stored on `oauth_login_attempt.remember_me`, so the
    callback request can't change it.
- **Logout** always signs out locally, even if the request fails.
- **Logout-all** rethrows on failure and keeps the user signed in, so nobody believes
  the other devices were signed out when they weren't. Only on success does it clear
  this browser's cookie.
- **Cross-tab:** `authChannel.js` posts only the bare strings `'login'` / `'logout'`
  on the BroadcastChannel `mq-admin-auth`. Never put user data or tokens on it. On
  `login`, a receiving tab re-asks `/auth/me`.
- Both logout buttons go through `ConfirmDialog`, which focuses **Cancel** first.

## Tasks

### Require login on a backend endpoint

Routes inside the `/api/v1/tiktok` and `/api/v1/lazada` groups are already
protected, because `rest.go` builds them with `apiGroup.Group("/tiktok", requireAuth)`.
For a new group, do the same in `APIRoutes`, and put the permission on each
route (as `/orders` does):

```go
// rest.go
ordersHandler.Router(apiGroup.Group("/orders", requireAuth))

// order_handler.go
r.GET("", middleware.RequirePermission(models.PermOrdersRead), h.List)
r.GET("/:id", middleware.RequirePermission(models.PermOrdersDetail), h.Get)
```

On the FE, wrap the route in `<ProtectedRoute permission="…">` inside the
signed-in `<Route element={<AppLayout />}>` block of `App.jsx`, and show the
menu item only when `can('…')`. Test both: a handler test through `APIRoutes`
with sessions that have and lack the permission (`order_handler_test.go`), and
a Vitest case that removes the permission with `backend.withoutPermissions`
(`OrdersPage.test.jsx`).

Inside a handler, read the user with `middleware.CurrentUser(c)` (`*models.SessionUser`:
`ID`, `Email`, `DisplayName`, `AuthMethod`, …). For a state-changing route
(POST/PUT/PATCH/DELETE), also add `sameOrigin` as a handler middleware, the way
`/auth/login` does.

Keep these public: `/swagger/*any`, `/api/v1/auth/{providers,google/*,login,logout}`,
and any marketplace **webhook**, since a marketplace can't send our cookie.
Cron workers don't go through HTTP, so they need nothing.

Gin only runs group middleware for routes that are registered. A protection
test therefore has to hit a real route, not a made-up path (a made-up path just returns 404).

### Create, disable or re-enable a user

Normally this is done in the UI: **Người dùng** (`/users`). It is visible with
`users.read`; `users.manage` is needed to create users, change role or status, and
reset passwords.

Without the UI (bootstrap, scripts), run these from `workspace/backend/`:

```bash
# internal password user (password is read from stdin: never pass it as an argument)
echo 'a-long-password' | go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen" --role admin --password-stdin
# Google-only user (no password; must sign in with Google); role = auth.default-role
go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen"
```

Passwords must be at least `auth.min-password-length` (default 10) and at most
72 bytes. Emails are lower-cased. To disable or enable a user without the UI,
confirm with the developer first and then use SQL. Disabling takes effect on the
user's next request, because `SessionRepo.GetValid` joins on `status = 'active'`:

```sql
UPDATE public.app_user SET status = 'disabled', updated_at = NOW() WHERE email = 'x@mq.com';
-- optional: also kill open sessions right away
UPDATE public.user_session SET revoked_at = NOW() WHERE user_id = (SELECT id FROM public.app_user WHERE email = 'x@mq.com') AND revoked_at IS NULL;
```

To clear a lockout: `UPDATE public.app_user SET failed_login_count = 0, locked_until = NULL WHERE email = '…';`

Never print, copy or commit `password_hash`, `user_session.token_hash`, or anything from `config/application.yml`.

### Set up Google sign-in for an environment

1. In Google Cloud Console, create an OAuth 2.0 Client of type **Web application**.
   Add `<api-origin>/api/v1/auth/google/callback` as an **Authorized redirect URI**.
   It must match `auth.google.redirect-uri` exactly, including scheme, host, port and path.
2. Put `AUTH_GOOGLE_CLIENT_ID` and `AUTH_GOOGLE_CLIENT_SECRET` in that
   environment's `config/.env`, or in real environment variables. Never write them
   in the yml, which only holds the `"${…}"` placeholders. In `config/application.yml`,
   set `auth.google.redirect-uri` and `enable: true`. Set
   `auth.frontend-url` to the FE origin, and set `cookie-secure: true` whenever the
   environment uses https.
3. Restart the service. On startup `newGoogleClient` runs OIDC discovery. If it fails,
   or config is missing, the log shows `Google login disabled: …`, password login
   keeps working, and `GET /api/v1/auth/providers` returns `google: false`. The
   login page still shows "Đăng nhập với Google", greyed out with "Đăng nhập Google
   chưa được cấu hình" under it; it becomes a working link once `google: true`.
   Look for that log line before debugging anything else.

The FE and API must be served from **one origin** (Vite proxy in dev, a reverse
proxy elsewhere). Don't add a CORS middleware. If that ever becomes unavoidable,
use an explicit origin with `AllowCredentials: true`, never `*`.

### Change who may sign in with Google

Only config changes are needed:

| Goal | `allowed-domains` | `auto-provision` |
|---|---|---|
| Must be `@mq.com` **and** pre-created (default) | `["mq.com"]` | `false` |
| Any `@mq.com` account, created on first login | `["mq.com"]` | `true` |
| Any pre-created user, any domain | `[]` | `false` |

The domain check requires **both** the `hd` claim and the email suffix to match,
so a personal Gmail can never get through it. Don't weaken this to checking the
email suffix only.

### UI language (EC-7)

- Supported: `vi` (default) and `en`. Each user's choice is `user_config.locale`
  (table `user_config`, 1-1 with `app_user`, migration `006`), changed with `PUT /auth/me/locale` and returned as
  `locale` on `/auth/me` and login. `create-user --locale en` sets it up front.
- Backend: `middleware.Locale()` (on all of `/api/v1`) picks the language from
  `Accept-Language`; `RequireAuth` then switches to the user's saved locale.
  `i18n.Locale(c)` reads it. It only translates `error_text` and role/permission
  names (`rest/localize.go`); **error codes never change**, so code and tests
  keep switching on `message`.
- Frontend: `useAuth().setLanguage(code)` saves the choice on the account, then
  switches the UI and tells other tabs (`'locale'` on the auth channel). Signed
  out, the choice is kept in `localStorage` (`mq-admin-lang`, the language code
  only). Login applies the account's saved locale.
- New language: `i18n/locales/<code>.json` + `i18n.Supported` (backend), a
  migration widening `ck_user_config_locale`, `src/locales/<code>.json` +
  `LANGUAGES` in `src/i18n/index.js` (frontend).

### Add or change an auth error code

1. Add the sentinel in `component/services/auth_errors.go` (`Code` is stable and
   upper snake case; `Status` goes into the body `code`). Return it, wrapped with
   `wrapAuth(Err…, cause)` when there is a cause worth logging.
2. Add the user-facing text in **both languages**:
   - backend `error.<CODE>` in `workspace/backend/i18n/locales/{vi,en}.json`
     (becomes `error_text` in the response);
   - frontend `errors.login.<CODE>` (or `errors.action.<CODE>`) in
     `workspace/frontend/src/locales/{vi,en}.json`.
   The FE prefers its own wording, then the backend's `error_text`, then a
   generic message, so it won't break if you forget one, but the user gets vaguer text.
   Both sides have a test that the `vi` and `en` catalogs have the same keys.
3. Cover the new code with tests: the backend service test plus a case in
   `LoginPage.test.jsx` (`it.each` over `?error=`).

### Debug "I can't log in"

Read the backend log around the attempt. Every failure logs its code and a
cause, and never logs secrets.

- `INVALID_STATE`: the login took more than 10 minutes, or the callback was
  opened twice, or the request reached a different DB than the one that started it.
- `USER_NOT_ALLOWED`: there's no `app_user` for that email (create it), the user
  is disabled, or the row is linked to a different Google account (`google_sub`).
- `DOMAIN_NOT_ALLOWED`: the account isn't in `allowed-domains`, or it's a
  personal account with no `hd` claim.
- `EXCHANGE_FAILED`: wrong client secret or redirect URI, usually `invalid_client`
  or `redirect_uri_mismatch` in the cause.
- `ACCOUNT_LOCKED`: 5 wrong passwords. It clears itself after 15 minutes, or you
  can clear it with the SQL above.
- Cookie not sticking: `cookie-secure: true` on plain http, or the FE and API
  are on different origins.

## Rules (do not regress)

- **No secrets in logs or responses.** Never log or return passwords, session tokens
  or their hashes, the OAuth `code`, `state`, `nonce`, the PKCE verifier, Google tokens,
  `client-secret`, or a provider's raw error body. `server/base.go` `requestLogger()`
  strips the query string, because the callback URL carries the code. Keep it and
  don't switch back to `gin.Logger()`. On a bind error, `/login` returns only
  `INVALID_REQUEST`, never `err.Error()`. Tests: `TestAuthFlows_DoNotLogSecrets`,
  `TestRequestLogger_DropsQueryString`, `TestGoogleOIDC_Exchange`.
- **Store only hashes.** Sessions are stored as `sha256(token)` and passwords as
  bcrypt (cost 12). Google access and refresh tokens are discarded.
- **Each OAuth state is used once and expires.** `LoginAttemptRepo.Consume` is one
  `DELETE … RETURNING`, and the service consumes the state *before* it looks at the
  provider's `error` param. The ID token must pass go-oidc verification (signature,
  `iss`, `aud`, `exp`), then a `nonce` match, then `email_verified`.
- **No open redirects.** `next` goes through `SanitizeRedirectPath` (backend) and
  `safeNextPath` (FE). Change both together, and keep their tests in sync.
- **Password errors don't reveal whether an account exists.** An unknown email, a
  wrong password, a Google-only account and a disabled account all return
  `INVALID_CREDENTIALS`, and the unknown-user path still runs a dummy bcrypt
  compare so the timing matches.
- **The FE never stores auth data.** No `localStorage`/`sessionStorage` (a test
  spies on `Storage.prototype.setItem`; the only allowed key is `mq-admin-lang`,
  the UI language code), no tokens in JS, and no secrets in `VITE_*` variables.
- **Error codes stay language-neutral.** Translated text only goes in
  `error_text`; never put it in `message` or branch on it.
- **No typed-nil Google client.** When Google is off, pass an untyped `nil`
  `OIDCClient`. A nil `*googleOIDC` stored in the interface makes `s.google == nil`
  false and causes a panic.
- **Import direction:** `services` imports `middleware`, so `middleware` must not
  import `services`. `RequireAuth` takes the one-method `Authenticator` interface instead.
- **Migrations:** auth schema changes go in a new numbered file under `migrations/`
  (use the `db-migrate` skill). Never edit `002_create_auth.sql` after it has been
  applied anywhere.

## Testing

```bash
cd workspace/backend
go test ./component/services/ ./component/handlers/rest/ ./server/
ECOM_TEST_DATABASE_URL=postgres://…/mq_test?sslmode=disable go test ./component/repositories/ -run 'TestAuthRepos|TestRBACRepos|TestOrderQueryRepo'

cd workspace/frontend
npm test && npm run build
E2E_EMAIL=… E2E_PASSWORD=… npm run e2e        # real backend; after `npx playwright install chromium`
```

Google sign-in can't be automated end to end. The backend tests cover it with a
fake identity provider (`google_oidc_test.go`: discovery, JWKS, a signed ID token,
and bad signature, audience, issuer and expiry cases). Test the real flow by hand
once per environment after setup. Finish with the `backend-check` skill.
