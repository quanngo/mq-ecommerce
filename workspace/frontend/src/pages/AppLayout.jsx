import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ConfirmDialog } from './ConfirmDialog';
import { LanguageSwitcher } from './LanguageSwitcher';
import styles from './AppLayout.module.css';
import ui from './ui.module.css';

/** Top bar + navigation for every signed-in page. Menu items follow permissions. */
export function AppLayout() {
  const { t } = useTranslation();
  const { state, logout, can } = useAuth();
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [languageSaveFailed, setLanguageSaveFailed] = useState(false);
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
        <span className={styles.brand}>{t('common.brand')}</span>
        <nav className={styles.nav} aria-label={t('nav.main')}>
          <NavLink to="/profile" className={linkClass}>
            {t('nav.profile')}
          </NavLink>
          {can('orders.read') && (
            <NavLink to="/orders" className={linkClass}>
              {t('nav.orders')}
            </NavLink>
          )}
          {can('users.read') && (
            <NavLink to="/users" className={linkClass}>
              {t('nav.users')}
            </NavLink>
          )}
        </nav>
        <div className={styles.right}>
          <LanguageSwitcher onSaveResult={setLanguageSaveFailed} />
          <span className={styles.who} title={state.user?.email}>
            {state.user?.display_name || state.user?.email}
          </span>
          <button className={styles.logout} type="button" onClick={() => setConfirming(true)}>
            {t('nav.logout')}
          </button>
        </div>
      </header>

      {languageSaveFailed && (
        <div className={`${ui.error} ${styles.banner}`} role="alert">
          {t('common.languageSaveFailed')}
        </div>
      )}

      <Outlet />

      <ConfirmDialog
        open={confirming}
        title={t('nav.logoutTitle')}
        message={t('nav.logoutMessage')}
        confirmLabel={busy ? t('nav.loggingOut') : t('nav.logout')}
        busy={busy}
        onConfirm={onLogout}
        onCancel={close}
      />
    </div>
  );
}
