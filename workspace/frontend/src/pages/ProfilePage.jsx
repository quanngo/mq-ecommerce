import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { permissionLabel } from '../auth/errorMessages';
import { formatDateTime } from '../i18n/format';
import { ConfirmDialog } from './ConfirmDialog';
import styles from './ProfilePage.module.css';
import ui from './ui.module.css';

export function ProfilePage() {
  const { t } = useTranslation();
  const { state, logoutAll } = useAuth();
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  const closeDialog = useCallback(() => setConfirming(false), []);

  if (state.status !== 'authenticated') return null; // ProtectedRoute guarantees this
  const user = state.user;
  const name = user.display_name || user.email;
  const permissions = user.permissions ?? [];

  async function onLogoutAll() {
    setBusy(true);
    setError(null);
    try {
      await logoutAll();
      navigate('/login', { replace: true });
    } catch (err) {
      setConfirming(false);
      setError(err instanceof ApiError && err.errorCode === 'NETWORK_ERROR' ? 'profile.logoutAllNetwork' : 'profile.logoutAllFailed');
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className={ui.main}>
      <h1 className={ui.title}>{t('profile.greeting', { name })}</h1>

      {error && (
        <div className={ui.error} role="alert">
          {t(error)}
        </div>
      )}

      <section className={ui.card} aria-label={t('profile.account')}>
        <div className={styles.identity}>
          {user.avatar_url ? (
            <img className={styles.avatar} src={user.avatar_url} alt="" referrerPolicy="no-referrer" />
          ) : (
            <span className={styles.avatarFallback} aria-hidden="true">
              {initials(name)}
            </span>
          )}
          <div>
            <div className={styles.name}>{name}</div>
            <div className={styles.email}>{user.email}</div>
          </div>
        </div>

        <dl className={styles.details}>
          <div>
            <dt>{t('profile.role')}</dt>
            <dd>
              <span className={ui.badge}>{user.role?.name || user.role?.code || '—'}</span>
            </dd>
          </div>
          <div>
            <dt>{t('profile.method')}</dt>
            <dd>{t(`profile.methods.${user.auth_method}`, { defaultValue: user.auth_method })}</dd>
          </div>
          <div>
            <dt>{t('profile.lastLogin')}</dt>
            <dd>{formatDateTime(user.last_login_at)}</dd>
          </div>
        </dl>

        <div className={styles.permissions}>
          <div className={styles.permissionsLabel}>{t('profile.permissions')}</div>
          {permissions.length === 0 ? (
            <p className={ui.muted}>{t('profile.noPermissions')}</p>
          ) : (
            <ul className={ui.chips} aria-label={t('profile.yourPermissions')}>
              {permissions.map((p) => (
                <li key={p} className={ui.badgeMuted} title={p}>
                  {permissionLabel(p, user.permission_names)}
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      <section className={ui.card} aria-labelledby="security-title">
        <h2 id="security-title" className={ui.sectionTitle}>
          {t('profile.security')}
        </h2>
        <div className={styles.securityRow}>
          <p className={styles.securityText}>{t('profile.changePasswordText')}</p>
          <Link to="/account/password" className={ui.secondaryButton}>
            {t('profile.changePassword')}
          </Link>
        </div>
        <div className={styles.securityRow}>
          <p className={styles.securityText}>{t('profile.logoutAllText')}</p>
          <button className={ui.dangerButton} type="button" onClick={() => setConfirming(true)}>
            {t('profile.logoutAll')}
          </button>
        </div>
      </section>

      <ConfirmDialog
        open={confirming}
        title={t('profile.logoutAllTitle')}
        message={t('profile.logoutAllMessage')}
        confirmLabel={busy ? t('nav.loggingOut') : t('profile.logoutAllConfirm')}
        danger
        busy={busy}
        onConfirm={onLogoutAll}
        onCancel={closeDialog}
      />
    </main>
  );
}

function initials(name) {
  const parts = name.replace(/@.*/, '').split(/[\s._-]+/).filter(Boolean);
  return ((parts[0]?.[0] ?? '') + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase() || '?';
}
