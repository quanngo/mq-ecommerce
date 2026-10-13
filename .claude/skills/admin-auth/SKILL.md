---
name: admin-auth
description: Work with admin-app login (EC-1). The Go backend runs Google OIDC plus internal email/password with server-side sessions in an httpOnly cookie, and the React (JS) app lives in workspace/frontend/admin. Covers protecting a new backend endpoint with RequireAuth, reading the signed-in user, creating internal users with the create-user CLI, configuring Google per environment (allowed-domains / auto-provision), adding an auth error code end to end, the security rules that must not regress, and how to test. Use when asked to require login on an endpoint, add or disable a user, set up Google sign-in for an environment, change who may log in, debug a login failure, or touch any auth code.
---

Login is a security boundary. Read the files named below before changing
anything, and keep every rule in **Rules**. Each one is a requirement of
the EC-1 acceptance criteria, and most are enforced by a test.

## Where things are

| Concern | File (relative to `workspace/backend/` unless noted) |
|---|---|
| Schema | `migrations/002_create_auth.sql`: `app_user`, `user_session`, `oauth_login_attempt`; `003_auth_remember_me.sql`: `oauth_login_attempt.remember_me` |
| Config | `utils/config.go` `AuthConfig` / `GoogleAuthConfig` + `WithDefaults()`; documented in `config/application.yml.example` (`auth:` block) |
| Business logic | `component/services/auth_service.go` (`AuthService`), `auth_errors.go` (stable error codes), `google_oidc.go` (Google behind the `OIDCClient` interface) |
| SQL | `component/repositories/user_repo.go`, `session_repo.go`, `login_attempt_repo.go` |
| HTTP | `component/handlers/rest/auth_handler.go`, mounted in `rest.go` under `/api/v1/auth` |
| Middleware | `middleware/require_auth.go`: `RequireAuth`, `CurrentUser`, `RequireSameOrigin` |
| Wiring | `server/base.go` `Initialize()` (`newAuthService`, `newGoogleClient`), `InitializeWorkers()`; `server/create_user.go`; `main.go` |
| Cleanup job | `cleanupAuthSessions` in `server/workers.go` + `config/workers.yml` |
| Frontend | `workspace/frontend/admin/src/`: `api/client.js`, `api/auth.js`, `auth/AuthContext.jsx` (`login`/`logout`/`logoutAll`), `auth/authChannel.js` (cross-tab sync), `auth/ProtectedRoute.jsx`, `auth/errorMessages.js`, `pages/LoginPage.jsx`, `pages/ProfilePage.jsx`, `pages/ConfirmDialog.jsx` |
| Design | `.workflow/EC-1/design.md` (if present locally; `.workflow/` is gitignored) |

Endpoints: `GET providers`, `GET google/login[?next=&remember=1]`, `GET google/callback`,
`POST login` (`{email, password, remember_me}`), `GET me` (needs a session),
`POST logout` (this browser), and `POST logout-all` (needs a session; revokes every
session of the user and returns `{revoked_sessions}`). All are under
`/api/v1/auth`. JSON responses follow the repo convention: always HTTP 200,
the real status in `code`, the error code in `message`, and `data` as an array.

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
For a new group, do the same in `APIRoutes`:

```go
ordersHandler.Router(apiGroup.Group("/orders", requireAuth))
```

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

There is no admin UI. Run these from `workspace/backend/` with the target
environment's `config/application.yml`:

```bash
# internal password user (password is read from stdin: never pass it as an argument)
echo 'a-long-password' | go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen" --password-stdin
# Google-only user (no password; must sign in with Google)
go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen"
```

Passwords must be 10–72 bytes. Emails are lower-cased. To disable or enable
a user, confirm with the developer first and then use SQL. Disabling takes
effect on the user's next request, because `SessionRepo.GetValid` joins on
`status = 'active'`:

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
   keeps working, and `GET /api/v1/auth/providers` returns `google: false`.
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

### Add or change an auth error code

1. Add the sentinel in `component/services/auth_errors.go` (`Code` is stable and
   upper snake case; `Status` goes into the body `code`). Return it, wrapped with
   `wrapAuth(Err…, cause)` when there is a cause worth logging.
2. Add the user-facing text (Vietnamese) to `MESSAGES` in
   `workspace/frontend/admin/src/auth/errorMessages.js`. Unknown codes fall back
   to a generic message, so the FE won't break if you forget, but the user gets a vague message.
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
  spies on `Storage.prototype.setItem`), no tokens in JS, and no secrets in
  `VITE_*` variables.
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
ECOM_TEST_DATABASE_URL=postgres://…/mq_test?sslmode=disable go test ./component/repositories/ -run TestAuthRepos

cd workspace/frontend/admin
npm test && npm run build
E2E_EMAIL=… E2E_PASSWORD=… npm run e2e        # real backend; after `npx playwright install chromium`
```

Google sign-in can't be automated end to end. The backend tests cover it with a
fake identity provider (`google_oidc_test.go`: discovery, JWKS, a signed ID token,
and bad signature, audience, issuer and expiry cases). Test the real flow by hand
once per environment after setup. Finish with the `backend-check` skill.
