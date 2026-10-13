// UI preview without the Go backend or a database:
//
//   npm run dev:mock   ->  http://localhost:5173
//
// Starts an in-memory fake of the backend's /api/v1/auth contract (same
// HTTP-200 {code, message, data[]} envelope, same cookie, same error codes)
// on :9099 and runs Vite with /api proxied to it. Google sign-in is replaced
// by a local "consent" page. Dev only: nothing here is used by the real app.

import { randomBytes } from 'node:crypto';
import http from 'node:http';
import { fileURLToPath } from 'node:url';
import { createServer as createVite } from 'vite';

const MOCK_PORT = 9099;
const COOKIE = 'ecom_session';
const MAX_FAILED = 5;
const LOCK_MS = 15 * 60 * 1000;

const users = [
  { id: 1, email: 'demo@mq.com', password: 'demo-password', display_name: 'Demo User', avatar_url: null },
  { id: 2, email: 'google.user@mq.com', password: null, display_name: 'Nguyễn Văn Google', avatar_url: null },
].map((u) => ({ ...u, failed: 0, lockedUntil: 0, last_login_at: null }));

const sessions = new Map(); // token -> { userId, method }

const send = (res, status, body, headers = {}) => {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8', 'Cache-Control': 'no-store', ...headers });
  res.end(JSON.stringify(body));
};
const ok = (res, data, headers) => send(res, 200, { code: 200, message: 'Success', data: data === undefined ? [] : [data] }, headers);
const fail = (res, code, message, headers) => send(res, 200, { code, message }, headers);
const redirect = (res, location, headers = {}) => {
  res.writeHead(302, { Location: location, 'Cache-Control': 'no-store', ...headers });
  res.end();
};

// Like the Go backend: "remember me" = 7-day Max-Age, otherwise a browser-session cookie.
const setCookie = (token, remember) => `${COOKIE}=${token}; Path=/;${remember ? ' Max-Age=604800;' : ''} HttpOnly; SameSite=Lax`;
const clearCookie = `${COOKIE}=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax`;

function sessionUser(req) {
  const m = (req.headers.cookie ?? '').match(new RegExp(`(?:^|;\\s*)${COOKIE}=([^;]+)`));
  const s = m && sessions.get(m[1]);
  return s ? { token: m[1], user: users.find((u) => u.id === s.userId), method: s.method } : null;
}

function startSession(user, method) {
  const token = randomBytes(32).toString('base64url');
  user.last_login_at = new Date().toISOString();
  user.failed = 0;
  user.lockedUntil = 0;
  sessions.set(token, { userId: user.id, method });
  return token;
}

const userResponse = (user, method) => ({
  id: user.id,
  email: user.email,
  display_name: user.display_name,
  avatar_url: user.avatar_url,
  auth_method: method,
  last_login_at: user.last_login_at,
});

const safeNext = (n) => (n && n.startsWith('/') && !n.startsWith('//') && !n.includes('://') && !n.startsWith('/login') ? n : '/profile');

const readJson = (req) =>
  new Promise((resolve) => {
    let raw = '';
    req.on('data', (c) => (raw += c));
    req.on('end', () => {
      try {
        resolve(JSON.parse(raw));
      } catch {
        resolve(null);
      }
    });
  });

function consentPage(next, remember) {
  const q = (p) => `/api/v1/auth/google/mock-callback?${new URLSearchParams({ ...p, ...(remember ? { remember: '1' } : {}) })}`;
  const accounts = [
    ['google.user@mq.com', 'Tài khoản @mq.com đã được cấp quyền'],
    ['demo@mq.com', 'Tài khoản @mq.com đã được cấp quyền (có cả mật khẩu)'],
    ['stranger@mq.com', '@mq.com nhưng chưa có trong app_user → USER_NOT_ALLOWED'],
    ['someone@gmail.com', 'Gmail cá nhân → DOMAIN_NOT_ALLOWED'],
  ];
  return `<!doctype html><html lang="vi"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Mock Google sign-in</title><style>
body{font-family:system-ui,sans-serif;background:#f1f3f4;margin:0;display:flex;min-height:100vh;align-items:center;justify-content:center;padding:16px;box-sizing:border-box}
.c{background:#fff;border-radius:12px;padding:28px;max-width:420px;width:100%;box-shadow:0 1px 3px rgba(0,0,0,.15)}
h1{font-size:20px;margin:0 0 4px}p{color:#5f6368;margin:0 0 20px;font-size:14px}
a{display:block;padding:12px 14px;border:1px solid #dadce0;border-radius:8px;margin-bottom:10px;color:#202124;text-decoration:none}
a:hover{background:#f8f9fa}small{display:block;color:#5f6368;margin-top:2px}.deny{color:#b3261e;text-align:center}
.tag{display:inline-block;background:#fef7e0;color:#7a5900;font-size:12px;padding:2px 8px;border-radius:99px;margin-bottom:12px}
</style></head><body><div class="c"><span class="tag">MOCK — không phải Google thật</span>
<h1>Chọn tài khoản</h1><p>để tiếp tục tới MQ E-commerce Admin</p>
${accounts.map(([email, hint]) => `<a href="${q({ email, next })}">${email}<small>${hint}</small></a>`).join('')}
<a class="deny" href="${q({ error: 'access_denied', next })}">Huỷ (từ chối quyền)</a></div></body></html>`;
}

const mock = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://localhost');
  const route = `${req.method} ${url.pathname}`;
  console.log(`[mock] ${req.method} ${url.pathname}`);

  switch (route) {
    case 'GET /api/v1/auth/providers':
      return ok(res, { google: true, password: true });

    case 'GET /api/v1/auth/google/login':
      return redirect(res, `/api/v1/auth/google/mock-consent?${url.searchParams}`);

    case 'GET /api/v1/auth/google/mock-consent':
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      return res.end(consentPage(safeNext(url.searchParams.get('next')), url.searchParams.get('remember') === '1'));

    case 'GET /api/v1/auth/google/mock-callback': {
      if (url.searchParams.get('error')) return redirect(res, '/login?error=CONSENT_DENIED');
      const email = url.searchParams.get('email') ?? '';
      if (!email.endsWith('@mq.com')) return redirect(res, '/login?error=DOMAIN_NOT_ALLOWED');
      const user = users.find((u) => u.email === email);
      if (!user) return redirect(res, '/login?error=USER_NOT_ALLOWED');
      return redirect(res, safeNext(url.searchParams.get('next')), { 'Set-Cookie': setCookie(startSession(user, 'google'), url.searchParams.get('remember') === '1') });
    }

    case 'POST /api/v1/auth/login': {
      const body = await readJson(req);
      if (!body || typeof body.email !== 'string' || !body.email.includes('@') || !body.password) {
        return fail(res, 400, 'INVALID_REQUEST');
      }
      const user = users.find((u) => u.email === body.email.trim().toLowerCase());
      if (!user || !user.password) return fail(res, 401, 'INVALID_CREDENTIALS');
      if (user.lockedUntil > Date.now()) return fail(res, 423, 'ACCOUNT_LOCKED');
      if (user.password !== body.password) {
        user.failed += 1;
        if (user.failed >= MAX_FAILED) {
          user.lockedUntil = Date.now() + LOCK_MS;
          return fail(res, 423, 'ACCOUNT_LOCKED');
        }
        return fail(res, 401, 'INVALID_CREDENTIALS');
      }
      const token = startSession(user, 'password');
      return ok(res, userResponse(user, 'password'), { 'Set-Cookie': setCookie(token, body.remember_me === true) });
    }

    case 'GET /api/v1/auth/me': {
      const s = sessionUser(req);
      return s ? ok(res, userResponse(s.user, s.method)) : fail(res, 401, 'UNAUTHENTICATED');
    }

    case 'POST /api/v1/auth/logout-all': {
      const s = sessionUser(req);
      if (!s) return fail(res, 401, 'UNAUTHENTICATED');
      let revoked = 0;
      for (const [token, sess] of sessions) {
        if (sess.userId === s.user.id) {
          sessions.delete(token);
          revoked++;
        }
      }
      return ok(res, { revoked_sessions: revoked }, { 'Set-Cookie': clearCookie });
    }

    case 'POST /api/v1/auth/logout': {
      const s = sessionUser(req);
      if (s) sessions.delete(s.token);
      return s ? ok(res, undefined, { 'Set-Cookie': clearCookie }) : fail(res, 401, 'UNAUTHENTICATED', { 'Set-Cookie': clearCookie });
    }

    default:
      return fail(res, 404, 'NOT_FOUND');
  }
});

mock.listen(MOCK_PORT, async () => {
  process.env.API_TARGET = `http://localhost:${MOCK_PORT}`;
  const vite = await createVite({ configFile: fileURLToPath(new URL('../vite.config.js', import.meta.url)) });
  await vite.listen();
  console.log('\n  MQ Admin (mock backend) -> http://localhost:5173');
  console.log('  Password login : demo@mq.com / demo-password');
  console.log('  Google login   : click the Google button -> pick an account on the mock consent page\n');
});
