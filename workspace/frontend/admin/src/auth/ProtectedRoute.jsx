import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { Spinner } from '../pages/Spinner';

/** Renders child routes for signed-in users; sends everyone else to /login?next=<here>. */
export function ProtectedRoute() {
  const { state } = useAuth();
  const location = useLocation();

  if (state.status === 'loading') return <Spinner label="Đang kiểm tra phiên đăng nhập…" />;
  if (state.status === 'anonymous') {
    const next = location.pathname + location.search;
    return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />;
  }
  return <Outlet />;
}
