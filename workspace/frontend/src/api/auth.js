import { apiRequest } from './client';

/**
 * models.UserResponse in the backend.
 * @typedef {object} User
 * @property {number} id
 * @property {string} email
 * @property {string} display_name
 * @property {string | null} avatar_url
 * @property {'google' | 'password'} auth_method
 * @property {string | null} last_login_at  RFC3339
 * @property {{ code: string, name: string }} role
 * @property {string[]} permissions  e.g. "users.manage"; check these, not the role name
 */

/**
 * @typedef {object} Providers
 * @property {boolean} google
 * @property {boolean} password
 */

export const authApi = {
  /** @returns {Promise<Providers>} */
  providers: async () => (await apiRequest('GET', '/auth/providers')) ?? { google: false, password: true },

  /** @returns {Promise<User>} */
  me: async () => {
    const user = await apiRequest('GET', '/auth/me');
    if (!user) throw new Error('empty /auth/me response');
    return user;
  },

  /**
   * @param {boolean} [rememberMe] long session (7 days by default) + cookie that survives a browser restart
   * @returns {Promise<User>}
   */
  login: async (email, password, rememberMe = false) => {
    const user = await apiRequest('POST', '/auth/login', { email, password, remember_me: rememberMe });
    if (!user) throw new Error('empty /auth/login response');
    return user;
  },

  /** Ends this browser's session. @returns {Promise<void>} */
  logout: async () => {
    await apiRequest('POST', '/auth/logout');
  },

  /**
   * Changes the signed-in user's password; their other sessions are signed out.
   * currentPassword may be '' for a Google-only account setting its first password.
   */
  changePassword: async (currentPassword, newPassword) => {
    await apiRequest('POST', '/auth/change-password', { current_password: currentPassword, new_password: newPassword });
  },

  /**
   * Ends every session of the signed-in user, on all devices.
   * @returns {Promise<number>} how many sessions were revoked
   */
  logoutAll: async () => {
    const res = await apiRequest('POST', '/auth/logout-all');
    return res?.revoked_sessions ?? 0;
  },
};

/**
 * Google sign-in is a full-page navigation (not fetch): the backend 302s to
 * Google and Google 302s back to the backend callback, which sets the cookie.
 * @param {string} next
 * @param {boolean} [rememberMe]
 */
export function googleLoginUrl(next, rememberMe = false) {
  const params = new URLSearchParams({ next });
  if (rememberMe) params.set('remember', '1');
  return `/api/v1/auth/google/login?${params}`;
}

/**
 * Same rule as the backend's SanitizeRedirectPath: same-site absolute paths only.
 * @param {string | null | undefined} next
 */
export function safeNextPath(next) {
  if (!next || next.length > 512 || !next.startsWith('/')) return '/profile';
  if (next.startsWith('//') || next.startsWith('/\\') || next.includes('://') || /[\\\r\n\t]/.test(next)) return '/profile';
  if (next.startsWith('/login')) return '/profile';
  return next;
}
