// UI preview without the Go backend or a database:
//
//   npm run dev:mock   ->  http://localhost:8088
//
// Starts an in-memory fake of the backend's /api/v1/auth, /users, /roles and /orders contract (same
// HTTP-200 {code, message, data[]} envelope, same cookie, same error codes)
// on :9099 and runs Vite with /api proxied to it. Google sign-in is replaced
// by a local "consent" page. Dev only: nothing here is used by the real app.

import { randomBytes } from 'node:crypto';
import { readFileSync } from 'node:fs';
import http from 'node:http';
import { fileURLToPath } from 'node:url';
import { createServer as createVite } from 'vite';

const MOCK_PORT = 9099;
const COOKIE = 'ecom_session';
const MAX_FAILED = 5;
const LOCK_MS = 15 * 60 * 1000;

// Same roles/permissions as migrations/004 + 005 in the backend.
const ROLES = [
  { code: 'admin', name: 'Quản trị viên', description: 'Toàn quyền, gồm quản lý người dùng', is_system: true, permissions: ['orders.detail', 'orders.read', 'orders.write', 'users.manage', 'users.read'] },
  { code: 'operator', name: 'Vận hành', description: 'Xem và xử lý đơn hàng', is_system: true, permissions: ['orders.detail', 'orders.read', 'orders.write'] },
  { code: 'viewer', name: 'Chỉ xem', description: 'Chỉ xem dữ liệu', is_system: true, permissions: ['orders.detail', 'orders.read'] },
];
const MIN_PASSWORD = 10;

// Role and permission names in the user's language, from the backend's own
// catalogs (workspace/backend/i18n/locales), like the real API (EC-7).
const CATALOGS = Object.fromEntries(
  ['vi', 'en'].map((l) => [l, JSON.parse(readFileSync(new URL(`../../backend/i18n/locales/${l}.json`, import.meta.url), 'utf8'))]),
);
const tr = (lang, key, fallback) => CATALOGS[lang]?.[key] ?? CATALOGS.vi[key] ?? fallback;
const roleRef = (role, lang) => ({ code: role.code, name: tr(lang, `role.${role.code}.name`, role.name) });
const permissionNames = (codes, lang) => Object.fromEntries(codes.map((c) => [c, tr(lang, `permission.${c}`, c)]));
const localizedRoles = (lang) =>
  ROLES.map((r) => ({
    ...r,
    ...roleRef(r, lang),
    description: tr(lang, `role.${r.code}.description`, r.description),
    permission_names: permissionNames(r.permissions, lang),
  }));

let nextId = 4;
const users = [
  // Default admin seeded by migrations/004 (same credentials as the real backend).
  { id: 1, email: 'admin@mq.com', password: '123456', display_name: 'Administrator', role: 'admin', google: false },
  { id: 2, email: 'demo@mq.com', password: 'demo-password', display_name: 'Demo User', role: 'operator', google: false },
  { id: 3, email: 'google.user@mq.com', password: null, display_name: 'Nguyễn Văn Google', role: 'viewer', google: true },
].map((u) => ({ ...u, avatar_url: null, status: 'active', locale: 'vi', failed: 0, lockedUntil: 0, last_login_at: null, created_at: new Date().toISOString() }));

const sessions = new Map(); // token -> { userId, method }

// 57 demo orders over the last ~30 days, newest first (public.ecom_order + items).
const NAMES = ['Nguyễn Văn An', 'Trần Thị Bình', 'Lê Hoàng Cường', 'Phạm Minh Dũng', 'Võ Thị Em', 'Đặng Quốc Phong'];
const PRODUCTS = [
  ['Serum dưỡng da', 'SR-01', 189000],
  ['Toner hoa hồng', 'TN-01', 129000],
  ['Kem chống nắng', 'KCN-50', 259000],
];
const orders = Array.from({ length: 57 }, (_, i) => {
  const id = 57 - i;
  const mp = ['Tiktok', 'Lazada', 'Shopee'][id % 3];
  const items = PRODUCTS.slice(0, 1 + (id % 3)).map(([name, sku, price], k) => ({ id: id * 10 + k, name, sku, quantity: 1 + ((id + k) % 2), price }));
  const subtotal = items.reduce((sum, it) => sum + it.price * it.quantity, 0);
  const discount = id % 4 === 0 ? 20000 : 0;
  const shippingFee = 15000;
  const created = new Date(Date.now() - i * 12.5 * 3600 * 1000).toISOString();
  return {
    id,
    market_place: mp,
    shop_id: `${mp.toLowerCase()}-shop-01`,
    order_id: `${mp.slice(0, 2).toUpperCase()}${580000000 + id * 7919}`,
    tracking_code: id % 5 === 0 ? '' : `SPX${100000 + id}`,
    lead_status: id % 11 === 0 ? 4 : id % 13 === 0 ? 1 : id % 17 === 0 ? 3 : 2,
    do_status: ['', 'READY_TO_SHIP', 'IN_TRANSIT', 'DELIVERED'][id % 4],
    so_status: id % 3 === 0 ? 45 : null,
    customer_name: NAMES[id % NAMES.length],
    phone: `09${String(10000000 + id * 104729).slice(0, 8)}`,
    total_amount: subtotal - discount + shippingFee,
    payment_method: id % 2 ? 1 : 8,
    item_count: items.length,
    created_at: created,
    updated_at: created,
    address: `${10 + id} Lê Lợi`,
    province: 'TP. Hồ Chí Minh',
    district: 'Quận 1',
    sub_district: 'Bến Nghé',
    neighborhood: '',
    warehouse: 'Kho HCM',
    shipping_label: '',
    comment: id % 6 === 0 ? 'Giao giờ hành chính' : '',
    discount,
    shipping_fee: shippingFee,
    items,
  };
});
/** created_at as a YYYY-MM-DD in Vietnam time, like the backend's from/to filter. */
const vnDate = (iso) => new Date(new Date(iso).getTime() + 7 * 3600 * 1000).toISOString().slice(0, 10);
const orderSummary = ({ address, province, district, sub_district, neighborhood, warehouse, shipping_label, comment, discount, shipping_fee, items, ...rest }) => rest;

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
  const user = s && users.find((u) => u.id === s.userId);
  return user && user.status === 'active' ? { token: m[1], user, method: s.method } : null;
}

function startSession(user, method) {
  const token = randomBytes(32).toString('base64url');
  user.last_login_at = new Date().toISOString();
  user.failed = 0;
  user.lockedUntil = 0;
  sessions.set(token, { userId: user.id, method });
  return token;
}

const roleOf = (user) => ROLES.find((r) => r.code === user.role);
const userResponse = (user, method) => ({
  id: user.id,
  email: user.email,
  display_name: user.display_name,
  avatar_url: user.avatar_url,
  auth_method: method,
  last_login_at: user.last_login_at,
  role: roleRef(roleOf(user), user.locale),
  permissions: roleOf(user).permissions,
  permission_names: permissionNames(roleOf(user).permissions, user.locale),
  locale: user.locale,
});
// lang: the signed-in viewer's language (role names are translated for them).
const managed = (u, lang = 'vi') => ({
  id: u.id, email: u.email, display_name: u.display_name, avatar_url: u.avatar_url, status: u.status,
  role: roleRef(roleOf(u), lang), has_password: !!u.password, google_linked: u.google,
  locked_until: u.lockedUntil > Date.now() ? new Date(u.lockedUntil).toISOString() : null,
  last_login_at: u.last_login_at, created_at: u.created_at,
});
const revokeAll = (userId, keep) => {
  for (const [token, sess] of sessions) if (sess.userId === userId && token !== keep) sessions.delete(token);
};
const managersLeft = (excludeId) =>
  users.filter((u) => u.id !== excludeId && u.status === 'active' && roleOf(u).permissions.includes('users.manage')).length;

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
  let route = `${req.method} ${url.pathname}`;
  const orderMatch = url.pathname.match(/^\/api\/v1\/orders\/([^/]+)$/);
  if (orderMatch) route = `${req.method} /api/v1/orders/:id`;
  const idMatch = url.pathname.match(/^\/api\/v1\/users\/(\d+)(\/reset-password)?$/);
  const targetId = idMatch ? Number(idMatch[1]) : 0;
  if (idMatch) route = `${req.method} /api/v1/users/:id${idMatch[2] ?? ''}`;
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
      if (!user || !user.password || user.status !== 'active') return fail(res, 401, 'INVALID_CREDENTIALS');
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

    case 'PUT /api/v1/auth/me/locale': {
      const s = sessionUser(req);
      if (!s) return fail(res, 401, 'UNAUTHENTICATED');
      const body = (await readJson(req)) ?? {};
      if (!['vi', 'en'].includes(body.locale)) return fail(res, 400, 'INVALID_REQUEST');
      s.user.locale = body.locale;
      return ok(res, userResponse(s.user, s.method));
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

    case 'POST /api/v1/auth/change-password': {
      const s = sessionUser(req);
      if (!s) return fail(res, 401, 'UNAUTHENTICATED');
      const body = (await readJson(req)) ?? {};
      if (s.user.password && body.current_password !== s.user.password) return fail(res, 400, 'WRONG_PASSWORD');
      if (!body.new_password || body.new_password.length < MIN_PASSWORD || body.new_password === body.current_password) {
        return fail(res, 400, 'WEAK_PASSWORD');
      }
      s.user.password = body.new_password;
      revokeAll(s.user.id, s.token);
      return ok(res);
    }

    case 'GET /api/v1/users':
    case 'GET /api/v1/roles':
    case 'POST /api/v1/users':
    case 'PATCH /api/v1/users/:id':
    case 'POST /api/v1/users/:id/reset-password': {
      const s = sessionUser(req);
      if (!s) return fail(res, 401, 'UNAUTHENTICATED');
      const perms = roleOf(s.user).permissions;
      const write = req.method !== 'GET';
      if (!perms.includes(write ? 'users.manage' : 'users.read')) return fail(res, 403, 'FORBIDDEN');
      if (route === 'GET /api/v1/roles') return ok(res, { roles: localizedRoles(s.user.locale) });
      if (route === 'GET /api/v1/users') return ok(res, { users: [...users].reverse().map((u) => managed(u, s.user.locale)) });

      const body = (await readJson(req)) ?? {};
      if (route === 'POST /api/v1/users') {
        const email = String(body.email ?? '').trim().toLowerCase();
        if (!email.includes('@') || !body.role) return fail(res, 400, 'INVALID_REQUEST');
        if (!ROLES.some((r) => r.code === body.role)) return fail(res, 400, 'INVALID_ROLE');
        if (users.some((u) => u.email === email)) return fail(res, 409, 'EMAIL_TAKEN');
        if ('password' in body && (body.password ?? '').length < MIN_PASSWORD) return fail(res, 400, 'WEAK_PASSWORD');
        const u = {
          id: nextId++, email, password: body.password ?? null, display_name: (body.display_name ?? '').trim(), role: body.role, google: false,
          avatar_url: null, status: 'active', locale: 'vi', failed: 0, lockedUntil: 0, last_login_at: null, created_at: new Date().toISOString(),
        };
        users.push(u);
        return ok(res, managed(u, s.user.locale));
      }

      const target = users.find((u) => u.id === targetId);
      if (!target) return fail(res, 404, 'NOT_FOUND');
      if (route.endsWith('/reset-password')) {
        if (target.id === s.user.id) return fail(res, 409, 'CANNOT_MODIFY_SELF');
        if ((body.password ?? '').length < MIN_PASSWORD) return fail(res, 400, 'WEAK_PASSWORD');
        target.password = body.password;
        target.failed = 0;
        target.lockedUntil = 0;
        revokeAll(target.id);
        return ok(res);
      }
      const roleChanges = body.role && body.role !== target.role;
      const statusChanges = body.status && body.status !== target.status;
      if ((roleChanges || statusChanges) && target.id === s.user.id) return fail(res, 409, 'CANNOT_MODIFY_SELF');
      if (roleChanges && !ROLES.some((r) => r.code === body.role)) return fail(res, 400, 'INVALID_ROLE');
      const targetManages = target.status === 'active' && roleOf(target).permissions.includes('users.manage');
      if (targetManages && (roleChanges || body.status === 'disabled') && managersLeft(target.id) === 0) {
        return fail(res, 409, 'LAST_ADMIN');
      }
      if (typeof body.display_name === 'string') target.display_name = body.display_name.trim();
      if (roleChanges) target.role = body.role;
      if (statusChanges) {
        target.status = body.status;
        if (body.status === 'disabled') revokeAll(target.id);
      }
      return ok(res, managed(target, s.user.locale));
    }

    case 'GET /api/v1/orders':
    case 'GET /api/v1/orders/:id': {
      const s = sessionUser(req);
      if (!s) return fail(res, 401, 'UNAUTHENTICATED');
      const perms = roleOf(s.user).permissions;
      if (route === 'GET /api/v1/orders/:id') {
        if (!perms.includes('orders.detail')) return fail(res, 403, 'FORBIDDEN');
        const order = orders.find((o) => String(o.id) === orderMatch[1]);
        return order ? ok(res, { order }) : fail(res, 404, 'NOT_FOUND');
      }
      if (!perms.includes('orders.read')) return fail(res, 403, 'FORBIDDEN');
      const get = (k) => (url.searchParams.get(k) ?? '').trim();
      const [q, mp, ls, from, to] = ['q', 'market_place', 'lead_status', 'from', 'to'].map(get);
      const page = Number(get('page') || 1);
      const size = Number(get('page_size') || 20);
      const dateOk = (d) => !d || /^\d{4}-\d{2}-\d{2}$/.test(d);
      const invalid =
        (mp && !['Tiktok', 'Lazada', 'Shopee'].includes(mp)) ||
        (ls && !['1', '2', '3', '4'].includes(ls)) ||
        !dateOk(from) ||
        !dateOk(to) ||
        (from && to && from > to) ||
        !(Number.isInteger(page) && page >= 1) ||
        !(Number.isInteger(size) && size >= 1 && size <= 100) ||
        q.length > 100;
      if (invalid) return fail(res, 400, 'INVALID_REQUEST');
      const needle = q.toLowerCase();
      const matches = orders.filter(
        (o) =>
          (!needle || [o.order_id, o.tracking_code, o.customer_name, o.phone].some((v) => v.toLowerCase().includes(needle))) &&
          (!mp || o.market_place === mp) &&
          (!ls || o.lead_status === Number(ls)) &&
          (!from || vnDate(o.created_at) >= from) &&
          (!to || vnDate(o.created_at) <= to),
      );
      return ok(res, { orders: matches.slice((page - 1) * size, page * size).map(orderSummary), total: matches.length, page, page_size: size });
    }

    default:
      return fail(res, 404, 'NOT_FOUND');
  }
});

mock.listen(MOCK_PORT, async () => {
  process.env.API_TARGET = `http://localhost:${MOCK_PORT}`;
  const vite = await createVite({ configFile: fileURLToPath(new URL('../vite.config.js', import.meta.url)) });
  await vite.listen();
  console.log('\n  MQ Admin (mock backend) -> http://localhost:8088');
  console.log('  Admin          : admin@mq.com / 123456   (all permissions, user management)');
  console.log('  Operator       : demo@mq.com / demo-password');
  console.log('  Google login   : click the Google button -> pick an account on the mock consent page\n');
});
