import { apiRequest } from './client';

/**
 * models.ManagedUser in the backend.
 * @typedef {object} ManagedUser
 * @property {number} id
 * @property {string} email
 * @property {string} display_name
 * @property {string | null} avatar_url
 * @property {'active' | 'disabled'} status
 * @property {{ code: string, name: string }} role
 * @property {boolean} has_password
 * @property {boolean} google_linked
 * @property {string | null} locked_until
 * @property {string | null} last_login_at
 * @property {string} created_at
 */

/**
 * @typedef {object} Role
 * @property {string} code
 * @property {string} name
 * @property {string} description
 * @property {boolean} is_system
 * @property {string[]} permissions
 */

export const usersApi = {
  /** @returns {Promise<ManagedUser[]>} */
  list: async () => (await apiRequest('GET', '/users'))?.users ?? [],

  /** @returns {Promise<Role[]>} */
  roles: async () => (await apiRequest('GET', '/roles'))?.roles ?? [],

  /**
   * @param {{ email: string, display_name?: string, role: string, password?: string }} user
   *   leave password out for a Google-only account
   * @returns {Promise<ManagedUser>}
   */
  create: (user) => apiRequest('POST', '/users', user),

  /**
   * @param {number} id
   * @param {{ display_name?: string, role?: string, status?: 'active' | 'disabled' }} changes
   * @returns {Promise<ManagedUser>}
   */
  update: (id, changes) => apiRequest('PATCH', `/users/${id}`, changes),

  /** Sets a new password and signs the user out everywhere. */
  resetPassword: async (id, password) => {
    await apiRequest('POST', `/users/${id}/reset-password`, { password });
  },
};
