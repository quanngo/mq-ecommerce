# admin: MQ E-commerce Admin (frontend)

React 18 + Vite, plain JavaScript (JSX), React Router 6, CSS modules. For now it has
only the login page (Google + internal email/password) and a profile page
(ticket EC-1). The backend is `workspace/backend`.

## Run

```bash
npm install
npm run dev          # http://localhost:5173, proxies /api -> http://localhost:9091
```

To preview the UI with no backend or database:

```bash
npm run dev:mock     # in-memory fake of /api/v1/auth (mock/dev-backend.js) + Vite on :5173
                     # password login: demo@mq.com / demo-password
                     # Google button opens a mock consent page (allowed, not-allowed, gmail, cancel)
```

For `npm run dev`, the backend must be running (`cd workspace/backend && go run main.go`) with
`migrations/002_create_auth.sql` applied. To point the proxy at a different
backend, set `API_TARGET=http://host:port npm run dev`.

Create an internal user (the password is read from stdin, so it doesn't end up in shell history):

```bash
cd workspace/backend
echo 'a-long-password' | go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen" --password-stdin
go run main.go create-user --email an.nguyen@mq.com --name "An Nguyen"   # Google-only
```

## How auth works (don't break these)

- **Logout:** both buttons ask for confirmation first (`ConfirmDialog`).
  - "Đăng xuất" ends this browser's session.
  - "Đăng xuất tất cả thiết bị" (`POST /auth/logout-all`) ends every session of the
    user. If that call fails, the user stays signed in and sees an error.
- Logging in or out in one tab updates the other tabs at once. This uses a
  BroadcastChannel and sends only the strings `'login'`/`'logout'`, never user data.
- **"Ghi nhớ đăng nhập trong 7 ngày"** applies to both the password form and the
  Google button:
  - ticked: a 7-day session and a persistent cookie;
  - not ticked: a cookie that disappears when the browser closes (the server-side
    limit is 12 hours).

- The **backend** is the OAuth client and owns the session. The browser only holds the
  httpOnly `ecom_session` cookie. JS can't read it, and the app never stores
  tokens or user data in `localStorage`/`sessionStorage`. A test checks this.
- The FE and the API must be **same-origin** (Vite proxy in dev, a reverse proxy
  that serves `/` and `/api` from one host in other environments). The cookie is
  `SameSite=Lax`, and the backend rejects a POST whose `Origin` isn't `auth.frontend-url`.
- "Đăng nhập với Google" is a full-page link to `/api/v1/auth/google/login?next=…`,
  not `fetch`. Callback errors come back as `/login?error=CODE`. The messages are
  in `src/auth/errorMessages.js`.
- API responses are always HTTP 200 with `{code, message, data[]}`.
  `src/api/client.js` unwraps `data[0]` and throws `ApiError(code, errorCode)`.
  Any `UNAUTHENTICATED` drops the user back to the login page.

## Test

```bash
npm test             # Vitest + Testing Library + MSW (fake backend in src/test/server.js)
npm run build
```

### E2E (Playwright, against the real backend)

```bash
npx playwright install chromium            # once
E2E_EMAIL=e2e@mq.com E2E_PASSWORD='...' npm run e2e
```

These tests are skipped unless `E2E_EMAIL`/`E2E_PASSWORD` are set. Create that user
with `create-user` first. Google sign-in can't be automated because Google blocks
automated consent, so test it by hand in each environment. The backend covers it
with a fake identity provider (`component/services/google_oidc_test.go`).
