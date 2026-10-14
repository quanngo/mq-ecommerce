// Thin fetch wrapper for the Go backend.
//
// Backend convention (workspace/backend/component/api/response.go): every
// response is HTTP 200 with body {code, message, data}; `code` is the real
// status, `message` carries the stable error code on failure, and `data` is
// always an array.

export class ApiError extends Error {
  /**
   * @param {number} code      status from the response body (0 = network failure)
   * @param {string} errorCode stable backend code, e.g. INVALID_CREDENTIALS
   */
  constructor(code, errorCode) {
    super(errorCode);
    this.name = 'ApiError';
    this.code = code;
    this.errorCode = errorCode;
  }
}

const unauthorizedListeners = new Set();

/**
 * Called whenever any request comes back UNAUTHENTICATED (e.g. the session expired).
 * @param {() => void} listener
 * @returns {() => void} unsubscribe
 */
export function onUnauthorized(listener) {
  unauthorizedListeners.add(listener);
  return () => unauthorizedListeners.delete(listener);
}

/**
 * @param {'GET' | 'POST'} method
 * @param {string} path  relative to /api/v1, e.g. "/auth/me"
 * @param {unknown} [body]
 * @returns {Promise<any>} data[0] of the response, or undefined
 */
export async function apiRequest(method, path, body) {
  const headers = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  let res;
  try {
    res = await fetch(`/api/v1${path}`, {
      method,
      credentials: 'same-origin',
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'NETWORK_ERROR');
  }

  let envelope;
  try {
    envelope = await res.json();
  } catch {
    throw new ApiError(res.status, 'INTERNAL_ERROR');
  }

  if (envelope.code !== 200) {
    if (envelope.message === 'UNAUTHENTICATED') {
      unauthorizedListeners.forEach((l) => l());
    }
    throw new ApiError(envelope.code, envelope.message || 'INTERNAL_ERROR');
  }
  return envelope.data?.[0];
}
