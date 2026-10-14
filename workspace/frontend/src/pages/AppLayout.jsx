import { useCallback, useState } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ConfirmDialog } from './ConfirmDialog';
import styles from './AppLayout.module.css';

/** Top bar + navigation for every signed-in page. Menu items follow permissions. */
export function AppLayout() {
  const { state, logout, can } = useAuth();
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const close = useCallback(() => setConfirming(false), []);

  async function onLogout() {
    setBusy(true);
    try {
      await logout();
    } finally {
      setBusy(false);
      navigate('/login', { replace: true });
    }
  }

  const linkClass = ({ isActive }) => (isActive ? `${styles.link} ${styles.active}` : styles.link);

  return (
    <div className={styles.layout}>
      <header className={styles.topbar}>
        <span className={styles.brand}>MQ Admin</span>
        <nav className={styles.nav} aria-label="Điều hướng chính">
          <NavLink to="/profile" className={linkClass}>
            Hồ sơ
          </NavLink>
          {can('orders.read') && (
            <NavLink to="/orders" className={linkClass}>
              Đơn hàng
            </NavLink>
          )}
          {can('users.read') && (
            <NavLink to="/users" className={linkClass}>
              Người dùng
            </NavLink>
          )}
        </nav>
        <div className={styles.right}>
          <span className={styles.who} title={state.user?.email}>
            {state.user?.display_name || state.user?.email}
          </span>
          <button className={styles.logout} type="button" onClick={() => setConfirming(true)}>
            Đăng xuất
          </button>
        </div>
      </header>

      <Outlet />

      <ConfirmDialog
        open={confirming}
        title="Đăng xuất?"
        message="Bạn sẽ đăng xuất khỏi trình duyệt này (mọi tab đang mở)."
        confirmLabel={busy ? 'Đang đăng xuất…' : 'Đăng xuất'}
        busy={busy}
        onConfirm={onLogout}
        onCancel={close}
      />
    </div>
  );
}
