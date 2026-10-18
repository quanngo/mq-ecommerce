import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

// A fake of the Go backend's /api/v1 contract (HTTP 200 + {code, message, data[]}).

export const ROLES = [
  { code: 'admin', name: 'Quản trị viên', description: 'Toàn quyền', is_system: true, permissions: ['orders.detail', 'orders.read', 'orders.write', 'users.manage', 'users.read'] },
  { code: 'operator', name: 'Vận hành', description: 'Xử lý đơn hàng', is_system: true, permissions: ['orders.detail', 'orders.read', 'orders.write'] },
  { code: 'viewer', name: 'Chỉ xem', description: 'Chỉ xem dữ liệu', is_system: true, permissions: ['orders.detail', 'orders.read'] },
];
const roleRef = (code) => {
  const r = ROLES.find((x) => x.code === code);
  return { code: r.code, name: r.name };
};

// What the backend's i18n catalog returns for the signed-in user's locale (EC-7).
const EN_ROLE_NAMES = { admin: 'Administrator', operator: 'Operator', viewer: 'Viewer' };
const PERMISSION_NAMES = {
  vi: { 'users.read': 'Xem người dùng', 'users.manage': 'Quản lý người dùng', 'orders.read': 'Xem đơn hàng', 'orders.detail': 'Xem chi tiết đơn hàng', 'orders.write': 'Cập nhật đơn hàng' },
  en: { 'users.read': 'View users', 'users.manage': 'Manage users', 'orders.read': 'View orders', 'orders.detail': 'View order details', 'orders.write': 'Update orders' },
};
const localizedRef = (ref) => (backend.locale === 'en' ? { ...ref, name: EN_ROLE_NAMES[ref.code] ?? ref.name } : ref);
const permissionNames = (codes) => Object.fromEntries(codes.map((c) => [c, PERMISSION_NAMES[backend.locale][c] ?? c]));
const localizedRoles = () =>
  ROLES.map((r) => ({ ...r, ...localizedRef({ code: r.code, name: r.name }), permission_names: permissionNames(r.permissions) }));

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
  role: 'viewer', // role of testUser
  withoutPermissions: [], // permissions taken away from testUser's role (e.g. ['orders.detail'])
  google: true,
  loginError: null,
  logoutCalls: 0,
  logoutAllCalls: 0,
  logoutAllError: null, // e.g. 'INTERNAL_ERROR', or 'NETWORK' for a dropped connection
  lastLoginBody: null,
  changePasswordBody: null,
  changePasswordError: null,
  users: [],
  lastUserRequest: null, // { method, path, body }
  userError: null, // error code returned by the next user-management write
  orders: [],
  lastOrdersQuery: null, // URLSearchParams object of the last GET /orders, as a plain object
  ordersError: null, // error code returned by GET /orders and /orders/:id
  locale: 'vi', // testUser's saved UI language (user_config.locale)
  localeError: null, // error code returned by PUT /auth/me/locale
  localeCalls: [], // bodies of PUT /auth/me/locale
  acceptLanguage: null, // Accept-Language of the last request
};

/**
 * localStorage/sessionStorage writes. `count`: anything other than the UI
 * language, which the app must never make; `language`: the last language code stored.
 */
export const storageWrites = { count: 0, language: null };

export function resetBackend() {
  Object.assign(backend, {
    signedIn: false,
    role: 'viewer',
    withoutPermissions: [],
    google: true,
    loginError: null,
    logoutCalls: 0,
    logoutAllCalls: 0,
    logoutAllError: null,
    lastLoginBody: null,
    changePasswordBody: null,
    changePasswordError: null,
    lastUserRequest: null,
    userError: null,
    orders: makeOrders(),
    lastOrdersQuery: null,
    ordersError: null,
    locale: 'vi',
    localeError: null,
    localeCalls: [],
    acceptLanguage: null,
    users: [
      managed({ id: 7, email: 'an.nguyen@mq.com', display_name: 'An Nguyen', role: 'admin' }),
      managed({ id: 8, email: 'op@mq.com', display_name: 'Operator', role: 'operator' }),
      managed({ id: 9, email: 'google.only@mq.com', display_name: 'G', role: 'viewer', has_password: false, google_linked: true }),
    ],
  });
}

function managed({ id, email, display_name, role, status = 'active', has_password = true, google_linked = false }) {
  return { id, email, display_name, avatar_url: null, status, role: roleRef(role), has_password, google_linked, locked_until: null, last_login_at: null, created_at: '2026-10-01T00:00:00Z' };
}

const me = () => {
  const permissions = ROLES.find((r) => r.code === backend.role).permissions.filter((p) => !backend.withoutPermissions.includes(p));
  return {
    ...testUser,
    role: localizedRef(roleRef(backend.role)),
    permissions,
    permission_names: permissionNames(permissions),
    locale: backend.locale,
  };
};
const can = (perm) => backend.signedIn && me().permissions.includes(perm);

/**
 * 25 orders, newest first (ids 25..1): enough for two pages of 20.
 * Order 25: TikTok, created; 24: Lazada, cancelled; the rest alternate.
 */
function makeOrders() {
  const orders = [];
  for (let id = 25; id >= 1; id--) {
    const tiktok = id % 2 === 1;
    orders.push({
      id,
      market_place: tiktok ? 'Tiktok' : 'Lazada',
      shop_id: tiktok ? 'tt-shop' : 'lz-shop',
      order_id: `${tiktok ? 'TT' : 'LZ'}-${1000 + id}`,
      tracking_code: `TRK${id}`,
      lead_status: id === 24 ? 4 : 2,
      do_status: id === 25 ? 'IN_TRANSIT' : '',
      so_status: id === 25 ? 45 : null,
      customer_name: id === 25 ? 'Nguyễn Văn An' : `Khách ${id}`,
      phone: `09000000${String(id).padStart(2, '0')}`,
      total_amount: 100000 + id * 1000,
      payment_method: 1,
      item_count: 2,
      created_at: `2026-10-${String(Math.min(id, 28)).padStart(2, '0')}T03:00:00Z`,
      updated_at: '2026-10-14T03:00:00Z',
      address: '12 Lê Lợi',
      province: 'TP. Hồ Chí Minh',
      district: 'Quận 1',
      sub_district: 'Bến Nghé',
      neighborhood: '',
      warehouse: 'Kho HCM',
      shipping_label: '',
      comment: id === 25 ? 'Giao giờ hành chính' : '',
      discount: 10000,
      shipping_fee: 15000,
      items: [
        { id: id * 10 + 1, name: 'Serum dưỡng da', sku: 'SR-01', quantity: 2, price: 50000 },
        { id: id * 10 + 2, name: 'Toner', sku: 'TN-01', quantity: 1, price: 30000 },
      ],
    });
  }
  return orders;
}

const summary = ({ address, province, district, sub_district, neighborhood, warehouse, shipping_label, comment, discount, shipping_fee, items, ...rest }) => rest;

function listOrders(url) {
  const p = url.searchParams;
  backend.lastOrdersQuery = Object.fromEntries(p);
  const leadStatus = p.get('lead_status');
  if ((leadStatus && !['1', '2', '3', '4'].includes(leadStatus)) || (p.get('from') && p.get('to') && p.get('from') > p.get('to'))) {
    return null;
  }
  const q = (p.get('q') ?? '').toLowerCase();
  const matches = backend.orders.filter(
    (o) =>
      (!q || [o.order_id, o.tracking_code, o.customer_name, o.phone].some((v) => v.toLowerCase().includes(q))) &&
      (!p.get('market_place') || o.market_place === p.get('market_place')) &&
      (!leadStatus || o.lead_status === Number(leadStatus)) &&
      (!p.get('from') || o.created_at.slice(0, 10) >= p.get('from')) &&
      (!p.get('to') || o.created_at.slice(0, 10) <= p.get('to')),
  );
  const page = Number(p.get('page') || 1);
  const size = Number(p.get('page_size') || 20);
  return { orders: matches.slice((page - 1) * size, page * size).map(summary), total: matches.length, page, page_size: size };
}

const ok = (data) => HttpResponse.json({ code: 200, message: 'Success', data: data === undefined ? [] : [data] });
const fail = (code, message) => HttpResponse.json({ code, message });
const guard = (perm) => (!backend.signedIn ? fail(401, 'UNAUTHENTICATED') : !can(perm) ? fail(403, 'FORBIDDEN') : null);

export const server = setupServer(
  http.all('*/api/v1/*', ({ request }) => {
    backend.acceptLanguage = request.headers.get('Accept-Language');
    // no response: fall through to the matching handler below
  }),
  http.get('*/api/v1/auth/providers', () => ok({ google: backend.google, password: true })),
  http.put('*/api/v1/auth/me/locale', async ({ request }) => {
    if (!backend.signedIn) return fail(401, 'UNAUTHENTICATED');
    const body = await request.json();
    backend.localeCalls.push(body);
    if (backend.localeError) return fail(500, backend.localeError);
    if (!['vi', 'en'].includes(body.locale)) return fail(400, 'INVALID_REQUEST');
    backend.locale = body.locale;
    return ok(me());
  }),
  http.get('*/api/v1/auth/me', () => (backend.signedIn ? ok(me()) : fail(401, 'UNAUTHENTICATED'))),
  http.post('*/api/v1/auth/login', async ({ request }) => {
    backend.lastLoginBody = await request.json();
    if (backend.loginError) {
      return fail(backend.loginError === 'ACCOUNT_LOCKED' ? 423 : 401, backend.loginError);
    }
    backend.signedIn = true;
    return ok(me());
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
  http.post('*/api/v1/auth/change-password', async ({ request }) => {
    if (!backend.signedIn) return fail(401, 'UNAUTHENTICATED');
    backend.changePasswordBody = await request.json();
    if (backend.changePasswordError) return fail(400, backend.changePasswordError);
    return ok();
  }),

  http.get('*/api/v1/orders', ({ request }) => {
    const denied = guard('orders.read');
    if (denied) return denied;
    if (backend.ordersError) return fail(500, backend.ordersError);
    const res = listOrders(new URL(request.url));
    return res ? ok(res) : fail(400, 'INVALID_REQUEST');
  }),
  http.get('*/api/v1/orders/:id', ({ params }) => {
    const denied = guard('orders.detail');
    if (denied) return denied;
    if (backend.ordersError) return fail(500, backend.ordersError);
    const order = backend.orders.find((o) => o.id === Number(params.id));
    return order ? ok({ order }) : fail(404, 'NOT_FOUND');
  }),

  http.get('*/api/v1/roles', () => guard('users.read') ?? ok({ roles: localizedRoles() })),
  http.get('*/api/v1/users', () => guard('users.read') ?? ok({ users: backend.users.map((u) => ({ ...u, role: localizedRef(u.role) })) })),
  http.post('*/api/v1/users', async ({ request }) => {
    const denied = guard('users.manage');
    if (denied) return denied;
    const body = await request.json();
    backend.lastUserRequest = { method: 'POST', path: '/users', body };
    if (backend.userError) return fail(409, backend.userError);
    const u = managed({ id: 100 + backend.users.length, email: body.email, display_name: body.display_name, role: body.role, has_password: 'password' in body });
    backend.users.unshift(u);
    return ok(u);
  }),
  http.patch('*/api/v1/users/:id', async ({ params, request }) => {
    const denied = guard('users.manage');
    if (denied) return denied;
    const body = await request.json();
    backend.lastUserRequest = { method: 'PATCH', path: `/users/${params.id}`, body };
    if (backend.userError) return fail(409, backend.userError);
    const u = backend.users.find((x) => x.id === Number(params.id));
    if (body.role) u.role = roleRef(body.role);
    if (body.status) u.status = body.status;
    return ok({ ...u });
  }),
  http.post('*/api/v1/users/:id/reset-password', async ({ params, request }) => {
    const denied = guard('users.manage');
    if (denied) return denied;
    backend.lastUserRequest = { method: 'POST', path: `/users/${params.id}/reset-password`, body: await request.json() };
    if (backend.userError) return fail(400, backend.userError);
    return ok();
  }),
);
