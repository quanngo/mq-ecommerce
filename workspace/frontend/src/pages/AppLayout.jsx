import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ConfirmDialog } from './ConfirmDialog';
import { LanguageSwitcher } from './LanguageSwitcher';
import styles from './AppLayout.module.css';
import ui from './ui.module.css';

/**
 * Shell of every signed-in page: the main menu in a sidebar on the left (menu
 * items follow permissions; language, user and logout at its foot), the page on
 * the right. Below 768px the sidebar becomes a bar above the page.
 */
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
      <aside className={styles.sidebar}>
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
          {can('orders.write') && (
            <NavLink to="/product-images" className={linkClass}>
              {t('nav.productImages')}
            </NavLink>
          )}
          {can('users.read') && (
            <NavLink to="/users" className={linkClass}>
              {t('nav.users')}
            </NavLink>
          )}
        </nav>
        <div className={styles.footer}>
          <LanguageSwitcher onSaveResult={setLanguageSaveFailed} />
          <div className={styles.who} title={state.user?.email}>
            <span className={styles.whoName}>{state.user?.display_name || state.user?.email}</span>
            {state.user?.display_name && <span className={styles.whoEmail}>{state.user.email}</span>}
          </div>
          <button className={styles.logout} type="button" onClick={() => setConfirming(true)}>
            {t('nav.logout')}
          </button>
        </div>
      </aside>

      <div className={styles.content}>
        {languageSaveFailed && (
          <div className={`${ui.error} ${styles.banner}`} role="alert">
            {t('common.languageSaveFailed')}
          </div>
        )}

        <Outlet />
      </div>

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
