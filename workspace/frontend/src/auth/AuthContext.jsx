import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { authApi } from '../api/auth';
import { ApiError, onUnauthorized } from '../api/client';
import { createAuthChannel } from './authChannel';

// Auth state lives in memory only. The session itself is an httpOnly cookie
// the JS can't read; nothing auth-related is ever put in local/sessionStorage.
//
// state.status: 'loading' | 'authenticated' | 'anonymous'
// state.user:   the User from /auth/me when authenticated, otherwise null
//
// Tabs of the same browser share the cookie, so they are kept in step through
// a BroadcastChannel: logging out (or in) in one tab updates every other tab
// immediately instead of on its next API call.

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [state, setState] = useState({ status: 'loading', user: null });
  const channelRef = useRef(null);

  const refreshFromServer = useCallback(() => {
    return authApi
      .me()
      .then((user) => setState({ status: 'authenticated', user }))
      .catch(() => setState({ status: 'anonymous', user: null }));
  }, []);

  useEffect(() => {
    let cancelled = false;
    authApi
      .me()
      .then((user) => !cancelled && setState({ status: 'authenticated', user }))
      .catch(() => !cancelled && setState({ status: 'anonymous', user: null }));

    const unsubscribe = onUnauthorized(() => setState({ status: 'anonymous', user: null }));

    const channel = createAuthChannel((message) => {
      if (message === 'logout') setState({ status: 'anonymous', user: null });
      // Another tab signed in (possibly as someone else): ask the server who we are now.
      if (message === 'login') refreshFromServer();
    });
    channelRef.current = channel;

    return () => {
      cancelled = true;
      unsubscribe();
      channel.close();
      channelRef.current = null;
    };
  }, [refreshFromServer]);

  const login = useCallback(async (email, password, rememberMe = false) => {
    const user = await authApi.login(email, password, rememberMe);
    setState({ status: 'authenticated', user });
    channelRef.current?.post('login');
    return user;
  }, []);

  const signedOut = useCallback(() => {
    setState({ status: 'anonymous', user: null });
    channelRef.current?.post('logout');
  }, []);

  /** Ends this browser's session (all tabs). Always signs out locally, even if the request fails. */
  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch (err) {
      // Already logged out server-side is fine; anything else is still a local logout.
      if (!(err instanceof ApiError)) throw err;
    } finally {
      signedOut();
    }
  }, [signedOut]);

  /**
   * Ends every session of this user on every device. Unlike logout, a failure
   * is rethrown and the user stays signed in here: pretending it worked would
   * leave the other devices signed in while the user believes they are not.
   */
  const logoutAll = useCallback(async () => {
    try {
      await authApi.logoutAll();
    } catch (err) {
      if (err instanceof ApiError && err.errorCode === 'UNAUTHENTICATED') {
        signedOut(); // our own session was already gone
        return;
      }
      throw err;
    }
    signedOut();
  }, [signedOut]);

  /** True when the signed-in user's role grants the permission (e.g. 'users.manage'). */
  const can = useCallback((permission) => state.status === 'authenticated' && (state.user.permissions ?? []).includes(permission), [state]);

  const value = useMemo(() => ({ state, login, logout, logoutAll, can }), [state, login, logout, logoutAll, can]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}
