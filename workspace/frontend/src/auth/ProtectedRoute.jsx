import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { Spinner } from '../pages/Spinner';
import { ForbiddenPage } from '../pages/ForbiddenPage';

/**
 * Renders child routes for signed-in users; sends everyone else to /login?next=<here>.
 * With `permission`, signed-in users without it see a "no access" page. The
 * backend enforces the same permission, so this only keeps the UI honest.
 */
export function ProtectedRoute({ permission }) {
  const { state, can } = useAuth();
  const location = useLocation();

  if (state.status === 'loading') return <Spinner label="Đang kiểm tra phiên đăng nhập…" />;
  if (state.status === 'anonymous') {
    const next = location.pathname + location.search;
    return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />;
  }
  if (permission && !can(permission)) return <ForbiddenPage />;
  return <Outlet />;
}
