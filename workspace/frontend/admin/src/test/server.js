import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

// A fake of the Go backend's /api/v1/auth contract (HTTP 200 + {code, message, data[]}).

export const testUser = {
  id: 7,
  email: 'an.nguyen@mq.com',
  display_name: 'An Nguyen',
  avatar_url: null,
  auth_method: 'password',
  last_login_at: '2026-10-14T02:30:00Z',
};

export const backend = {
  signedIn: false,
  google: true,
  loginError: null,
  logoutCalls: 0,
  logoutAllCalls: 0,
  logoutAllError: null, // e.g. 'INTERNAL_ERROR', or 'NETWORK' for a dropped connection
  lastLoginBody: null,
};

/** Counts localStorage/sessionStorage writes; the app must never make any. */
export const storageWrites = { count: 0 };

export function resetBackend() {
  backend.signedIn = false;
  backend.google = true;
  backend.loginError = null;
  backend.logoutCalls = 0;
  backend.logoutAllCalls = 0;
  backend.logoutAllError = null;
  backend.lastLoginBody = null;
}

const ok = (data) => HttpResponse.json({ code: 200, message: 'Success', data: data === undefined ? [] : [data] });
const fail = (code, message) => HttpResponse.json({ code, message });

export const server = setupServer(
  http.get('*/api/v1/auth/providers', () => ok({ google: backend.google, password: true })),
  http.get('*/api/v1/auth/me', () => (backend.signedIn ? ok(testUser) : fail(401, 'UNAUTHENTICATED'))),
  http.post('*/api/v1/auth/login', async ({ request }) => {
    backend.lastLoginBody = await request.json();
    if (backend.loginError) {
      return fail(backend.loginError === 'ACCOUNT_LOCKED' ? 423 : 401, backend.loginError);
    }
    backend.signedIn = true;
    return ok(testUser);
  }),
  http.post('*/api/v1/auth/logout', () => {
    backend.logoutCalls++;
    if (!backend.signedIn) return fail(401, 'UNAUTHENTICATED');
    backend.signedIn = false;
    return ok();
  }),
  http.post('*/api/v1/auth/logout-all', () => {
    backend.logoutAllCalls++;
    if (backend.logoutAllError === 'NETWORK') return HttpResponse.error();
    if (backend.logoutAllError) return fail(500, backend.logoutAllError);
    if (!backend.signedIn) return fail(401, 'UNAUTHENTICATED');
    backend.signedIn = false;
    return ok({ revoked_sessions: 3 });
  }),
);
