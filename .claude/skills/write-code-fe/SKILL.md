---
name: write-code-fe
description: Implement a feature on the frontend only, in the admin app (workspace/frontend/admin, React 18 + Vite, plain JavaScript/JSX, not TypeScript). Read the app's conventions and the existing backend API contract it will call, ask the developer about anything ambiguous, build the page/route/components, wire them to the real backend endpoints through src/api, add Vitest + MSW tests and, where it fits, a Playwright e2e spec. Does not modify backend code. Use when asked to build or change UI on the frontend only, or when the backend API already exists.
---

Implement a feature on the **frontend only**, grounded in what the code
actually shows. Never invent an API shape or a convention the project doesn't
use. Ask the developer rather than guess. This skill never writes backend code.

## Steps

1. **Restate the requirement** in your own words as a first check that you understood it.

2. **Read the frontend.** Read `workspace/frontend/admin/CLAUDE.md` first (stack,
   layout, conventions, tests), then the files you'll touch. The project is
   **plain JavaScript (`.js`/`.jsx`)**:
   - never add TypeScript, `tsconfig`, or `.ts`/`.tsx` files;
   - never "upgrade" a file to TypeScript while you're editing it;
   - describe data shapes with JSDoc `@typedef`.

   If another frontend project ever appears under `workspace/frontend/` and it's
   unclear which one the feature belongs to, ask.

3. **Read the backend contract (read-only).** Find the exact endpoint(s) in
   `workspace/backend/component/handlers/rest/*.go`, mounted in `rest.go`, and
   in `API_DOCUMENT.md` if present. Note the method, path, request body, the
   `data[0]` shape (the Go DTO's `json` tags) and the possible error codes in
   `message`.
   - Every endpoint answers HTTP 200 with `{code, message, data[]}`.
   - Routes in a group built with `requireAuth` need a session. The app sends
     the cookie automatically because it is same-origin.
   - If the endpoint doesn't exist or doesn't return what the UI needs, **stop**
     and tell the developer to run `/write-code-be` or `/write-code-full`. Don't
     mock it in the app or invent a path.

4. **Ask before coding** if any of these are unclear:
   - where it lives (new route or existing page; whether it needs login);
   - what each state looks like: loading, empty, error, success;
   - which endpoint(s) to call;
   - the Vietnamese wording of labels and errors, when it matters.

   Once those are answered, go straight to implementation. There is no separate design sign-off.

5. **Backend: not applicable.** Don't edit anything under `workspace/backend/`.

6. **Implement**, following `admin/CLAUDE.md` exactly:
   - **API**: add functions to `src/api/<area>.js` that call `apiRequest`. Never
     call `fetch` or add axios.
   - **Route**: register it in `src/App.jsx`. A page that needs login goes inside
     the `<Route element={<ProtectedRoute />}>` block.
   - **Page**: a component in `src/pages/<Name>Page.jsx` with
     `<Name>Page.module.css`. Use the `var(--…)` tokens from `styles/global.css`
     so light and dark both work, and make sure it works at phone width.
   - **State**: local `useState`/`useEffect`, plus `useAuth()` for the user. Don't
     add a state library, a component library or a CSS framework.
   - **Errors**: map backend error codes to Vietnamese text in a `*Messages.js`
     map with a generic fallback, and show them with `role="alert"`. Show loading
     with the `Spinner` component (`role="status"`).
   - **Storage**: nothing auth-related or user-related goes in `localStorage` or
     `sessionStorage`.

7. **Integrate.** Check field names against the Go DTO's `json` tags, which are
   snake_case (for example `display_name`). Remember that the backend's `data`
   is always an array, so `apiRequest` returns `data[0]`.

8. **Unit tests** (Vitest + Testing Library + MSW, the existing setup; don't add
   another tool):
   - Add the new endpoint(s) to `src/test/server.js` with the real envelope,
     and add any toggles to the exported `backend` object. Reset them in `resetBackend()`.
   - Test through the UI. `renderApp('/route')` renders the whole app with the
     router and auth; then use `findByRole`/`findByLabelText` and `userEvent`.
     Assert redirects via `getByTestId('location')`.
   - Cover success, every error code the UI handles, the loading and empty
     states, and the anonymous-visitor redirect for protected pages.

9. **API docs: not applicable** (no backend change).

10. **E2E.** For a user-visible flow, add `e2e/<feature>.spec.js` (Playwright)
    against the real backend. Like `e2e/auth.spec.js`, it should skip itself when
    the env vars it needs are missing. Say in the report that this covers the
    frontend side only.

11. **Verify, then report.** From `workspace/frontend/admin/` run
    `npm test && npm run build`, and quote the real result. On Windows, Node is
    at `C:\Program Files\nodejs`, so refresh `PATH` if `node` isn't found. If you
    didn't run e2e, say so. Then summarise:
    - the files changed;
    - the tests added and their result;
    - any open questions;
    - whether a backend counterpart is still needed.
