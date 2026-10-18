# CLAUDE.md: admin (frontend)

MQ E-commerce Admin. React 18 + Vite 6, **plain JavaScript (`.js` / `.jsx`), not
TypeScript**. That was the developer's decision, so don't add TypeScript, `tsconfig`
or `.ts`/`.tsx` files. React Router 6, CSS modules, no component or state library.
UI text is in Vietnamese (default) and English (EC-7, react-i18next): every string lives in
`src/locales/vi.json` and `src/locales/en.json`.

## Commands

```bash
npm install
npm run dev        # :8088 (fixed), proxies /api -> http://localhost:9091 (API_TARGET=... to change)
npm test           # Vitest + Testing Library + MSW
npm run build
npm run e2e        # Playwright against the real backend; needs E2E_EMAIL/E2E_PASSWORD (see README)
```

npm 11 blocks install scripts (esbuild, msw postinstall). The build and tests
work without them, so don't approve them unless something actually breaks.

## Layout

```
src/
  main.jsx               BrowserRouter (ROUTER_FUTURE flags) > AuthProvider > App
  App.jsx                all routes; protected pages go inside <Route element={<ProtectedRoute />}>
  routerFuture.js        React Router v7 future flags; pass to every Router (tests too)
  api/client.js          apiRequest(method, path, body): the only place that calls fetch
  api/auth.js            authApi + googleLoginUrl + safeNextPath; one module per backend area
  api/users.js, api/orders.js   user management; orders list/detail (query built from ORDER_QUERY_KEYS)
  auth/AuthContext.jsx   useAuth() -> { state: {status, user}, login, logout, logoutAll, can, setLanguage }
  auth/ProtectedRoute.jsx
  auth/errorMessages.js  backend error code -> message in the current language; permissionLabel
  i18n/index.js          i18next setup, LANGUAGES, language detection (account > localStorage > browser > vi)
  i18n/format.js         formatDateTime (Vietnam time) / formatNumber / formatMoney (VND) for the current language
  locales/vi.json, en.json   all UI text, same keys in both (a test checks)
  pages/*.jsx + *.module.css   one page = one component + its CSS module
  pages/orderFormat.js   order labels (marketplace, LeadStatus, SaleOrderStatus, payment), translated
  pages/LanguageSwitcher.jsx   VI / EN toggle (header and login page)
  styles/global.css      design tokens (:root vars, dark mode) and resets only
  test/server.js         MSW fake of the Go backend + mutable `backend` state
  test/setup.js          jest-dom, MSW lifecycle, relative-fetch shim, storage spy
  test/renderApp.jsx     renderApp(path): full app in a MemoryRouter + location probe
e2e/*.spec.js            Playwright
```

## Conventions

- **API calls**: add a function to `src/api/<area>.js` that calls
  `apiRequest('GET'|'POST', '/path', body)`. Paths are relative to `/api/v1`.
  The backend always answers HTTP 200 with `{code, message, data[]}`.
  `apiRequest` returns `data[0]` and throws `ApiError(code, errorCode)`
  otherwise. `UNAUTHENTICATED` automatically logs the user out of the UI. Don't
  call `fetch` anywhere else, and don't add axios.
- **Same origin only**: always use relative `/api/...` URLs. No base-URL env var,
  no CORS, no `credentials: 'include'`.
- **Auth**: the session is an httpOnly cookie. Never use `localStorage`/`sessionStorage`
  for auth or user data (a test enforces this; the UI language code, key
  `mq-admin-lang`, is the only thing stored), and never put secrets in `VITE_*`
  vars. Read the user with `useAuth().state.user`. See the `admin-auth` skill.
- **Types**: describe data shapes with JSDoc `@typedef` (see `api/auth.js`), not TypeScript.
- **Styling**: CSS modules per component. Use the `var(--…)` tokens from
  `global.css` (they cover light and dark), not hard-coded colours. Layouts must
  work at phone width.
- **Text**: never hard-code UI strings. Add the key to both locale files and use
  `const { t } = useTranslation()`. Format dates/numbers/money with `i18n/format.js`.
  Role and permission names come translated from the backend (`role.name`,
  `permission_names`); use `permissionLabel(code, names)`.
- **Errors shown to users**: keep the `ApiError` (or code) in state and translate
  at render with `authErrorMessage` / `actionErrorMessage` / `orderErrorMessage`
  (wording in `errors.<context>.<CODE>`; then the backend's `error_text`; then a
  generic message), so switching language re-translates messages on screen.
- **Permissions**: guard routes with `<ProtectedRoute permission="…">` and hide
  menu items/links with `useAuth().can('…')`; the backend enforces the same
  permission. List pages keep filters and page in the URL (`useSearchParams`),
  as `OrdersPage` does.
- **Accessibility**: real `<label>`s, `role="alert"` for errors, `role="status"`
  for loading. Tests query by role and label, so keep them meaningful.

## Tests

- Unit and component tests live next to the code as `*.test.js(x)`.
- Add new backend endpoints to `src/test/server.js` (the same envelope as the real
  backend), and drive scenarios through the exported `backend` object.
  `onUnhandledRequest: 'error'` fails any test that calls an endpoint that isn't mocked.
- Test through the UI with `renderApp('/path')` + `screen.findByRole(...)` +
  `userEvent`. Assert redirects with `screen.getByTestId('location')` (a plain
  `div`, so it never collides with `findByRole('status')`).
- Permissions in tests: `backend.role = 'viewer'` picks a seeded role;
  `backend.withoutPermissions = ['orders.detail']` takes permissions away from it.
- E2E specs in `e2e/` run against the real backend, and skip themselves when their
  env vars are missing.
