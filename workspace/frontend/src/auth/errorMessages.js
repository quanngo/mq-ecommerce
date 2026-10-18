// Backend error codes (services/auth_errors.go) -> what the user reads, in the
// current UI language. Wording lives in src/locales/*.json under errors.<context>.
//
// Lookup order: this page's own wording (errors.<context>.<CODE>), then the
// shared action wording, then the text the backend sent in the user's
// language (`error_text`, see ApiError.errorText), then a generic message.
// Call these during render (not when the error happens), so a language
// switch re-translates a message that is already on screen.

import i18n from '../i18n';

/**
 * @param {unknown} err  ApiError, an error code string, or null
 * @returns {{ code: string | null, backendText: string | null }}
 */
function parse(err) {
  if (typeof err === 'string') return { code: err, backendText: null };
  if (err && typeof err === 'object') return { code: err.errorCode ?? null, backendText: err.errorText || null };
  return { code: null, backendText: null };
}

/**
 * @param {('login' | 'action' | 'order')[]} contexts  most specific first
 * @param {unknown} err
 */
function describe(contexts, err) {
  const { code, backendText } = parse(err);
  if (code) {
    for (const ctx of contexts) {
      const key = `errors.${ctx}.${code}`;
      if (i18n.exists(key)) return i18n.t(key);
    }
  }
  return backendText ?? i18n.t('errors.generic');
}

/**
 * Message for a login error (password form or ?error=CODE from the Google callback).
 * @param {unknown} err  ApiError or error code
 * @returns {string | null} null when there is nothing to show
 */
export function authErrorMessage(err) {
  const { code } = parse(err);
  if (!code || code === 'UNAUTHENTICATED') return null;
  return describe(['login'], err);
}

/**
 * Message for an error from a non-login action (user management, password change).
 * @param {unknown} err  ApiError, error code or null
 * @returns {string}
 */
export function actionErrorMessage(err) {
  return describe(['action'], err);
}

/**
 * Message for an error from the Orders pages.
 * @param {unknown} err  ApiError, error code or null
 * @returns {string}
 */
export function orderErrorMessage(err) {
  return describe(['order', 'action'], err);
}

/**
 * Label of a permission code (models.Perm* in the backend). The backend sends
 * translated names (`permission_names` on /auth/me and /roles); the local
 * catalog covers anything it didn't send.
 * @param {string} code  e.g. "orders.read"
 * @param {Record<string, string> | undefined} [names]
 */
export function permissionLabel(code, names) {
  return names?.[code] || i18n.t(`permissions.${code}`, { defaultValue: code });
}
