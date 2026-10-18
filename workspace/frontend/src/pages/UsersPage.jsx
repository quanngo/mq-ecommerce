import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError } from '../api/client';
import { usersApi } from '../api/users';
import { useAuth } from '../auth/AuthContext';
import { actionErrorMessage, permissionLabel } from '../auth/errorMessages';
import { formatDateTime } from '../i18n/format';
import { MIN_PASSWORD_LENGTH } from './ChangePasswordPage';
import { Modal } from './Modal';
import { PasswordField } from './PasswordField';
import { Spinner } from './Spinner';
import styles from './UsersPage.module.css';
import ui from './ui.module.css';

// Errors and notices are kept untranslated (ApiError / {key, params}) and
// translated at render, so switching language re-translates them.
const asError = (err) => (err instanceof ApiError ? err : 'INTERNAL_ERROR');

export function UsersPage() {
  const { t, i18n } = useTranslation();
  const { state, can } = useAuth();
  const canManage = can('users.manage');
  const me = state.user?.id;

  const [users, setUsers] = useState(null);
  const [roles, setRoles] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null); // { key, params } | null
  const [savingId, setSavingId] = useState(null);
  const [creating, setCreating] = useState(false);
  const [resetting, setResetting] = useState(null); // ManagedUser | null

  const load = useCallback(async () => {
    try {
      const [u, r] = await Promise.all([usersApi.list(), usersApi.roles()]);
      setUsers(u);
      setRoles(r);
    } catch (err) {
      setError(asError(err));
      setUsers([]);
    }
  }, []);

  // Role and permission names come translated from the backend: reload them in the new language.
  useEffect(() => {
    load();
  }, [load, i18n.language]);

  const replaceUser = (updated) => setUsers((list) => list.map((u) => (u.id === updated.id ? updated : u)));

  async function change(user, changes, successNotice) {
    setError(null);
    setNotice(null);
    setSavingId(user.id);
    try {
      replaceUser(await usersApi.update(user.id, changes));
      setNotice(successNotice);
    } catch (err) {
      setError(asError(err));
    } finally {
      setSavingId(null);
    }
  }

  const closeCreate = useCallback(() => setCreating(false), []);
  const closeReset = useCallback(() => setResetting(null), []);

  if (users === null) return <Spinner label={t('users.loading')} />;

  return (
    <main className={ui.main}>
      <div className={styles.header}>
        <h1 className={ui.title}>{t('users.title')}</h1>
        {canManage && (
          <button className={ui.primaryButton} type="button" onClick={() => setCreating(true)}>
            {t('users.create')}
          </button>
        )}
      </div>

      {error && (
        <div className={ui.error} role="alert">
          {actionErrorMessage(error)}
        </div>
      )}
      {notice && (
        <div className={ui.success} role="status">
          {t(notice.key, notice.params)}
        </div>
      )}

      <section className={`${ui.card} ${styles.tableCard}`}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th scope="col">{t('users.colUser')}</th>
              <th scope="col">{t('users.colRole')}</th>
              <th scope="col">{t('users.colMethod')}</th>
              <th scope="col">{t('users.colStatus')}</th>
              <th scope="col">{t('users.colLastLogin')}</th>
              {canManage && <th scope="col" aria-label={t('users.colActions')} />}
            </tr>
          </thead>
          <tbody>
            {users.map((u) => {
              const isMe = u.id === me;
              const saving = savingId === u.id;
              return (
                <tr key={u.id} aria-busy={saving}>
                  <td>
                    <div className={styles.userName}>
                      {u.display_name || '—'} {isMe && <span className={ui.badgeMuted}>{t('users.you')}</span>}
                    </div>
                    <div className={styles.userEmail}>{u.email}</div>
                  </td>
                  <td>
                    {canManage && !isMe ? (
                      <select
                        className={ui.select}
                        aria-label={t('users.roleOf', { email: u.email })}
                        value={u.role.code}
                        disabled={saving}
                        onChange={(e) => change(u, { role: e.target.value }, { key: 'users.roleChanged', params: { email: u.email } })}
                      >
                        {roles.map((r) => (
                          <option key={r.code} value={r.code}>
                            {r.name}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <span className={ui.badge}>{u.role.name || u.role.code}</span>
                    )}
                  </td>
                  <td>
                    <div className={styles.methods}>
                      {u.has_password && <span className={ui.badgeMuted}>{t('users.methodPassword')}</span>}
                      {u.google_linked && <span className={ui.badgeMuted}>{t('users.methodGoogle')}</span>}
                      {!u.has_password && !u.google_linked && <span className={ui.badgeMuted}>{t('users.methodGooglePending')}</span>}
                    </div>
                  </td>
                  <td>
                    {u.status === 'active' ? (
                      <span className={ui.badgeMuted}>{t('users.active')}</span>
                    ) : (
                      <span className={ui.badgeDanger}>{t('users.disabled')}</span>
                    )}
                    {u.locked_until && <div className={styles.lockNote}>{t('users.lockedUntil', { time: formatDateTime(u.locked_until) })}</div>}
                  </td>
                  <td className={styles.when}>{formatDateTime(u.last_login_at)}</td>
                  {canManage && (
                    <td>
                      <div className={styles.rowActions}>
                        {!isMe && (
                          <>
                            <button
                              type="button"
                              className={ui.secondaryButton}
                              disabled={saving}
                              onClick={() =>
                                change(
                                  u,
                                  { status: u.status === 'active' ? 'disabled' : 'active' },
                                  { key: u.status === 'active' ? 'users.disabledNotice' : 'users.enabledNotice', params: { email: u.email } },
                                )
                              }
                            >
                              {u.status === 'active' ? t('users.disable') : t('users.enable')}
                            </button>
                            <button type="button" className={ui.secondaryButton} disabled={saving} onClick={() => setResetting(u)}>
                              {t('users.resetPassword')}
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
      </section>

      <RolesLegend roles={roles} />

      {canManage && (
        <CreateUserDialog
          open={creating}
          roles={roles}
          onClose={closeCreate}
          onCreated={(u) => {
            setUsers((list) => [u, ...list]);
            setNotice({ key: 'users.created', params: { email: u.email } });
            setError(null);
            setCreating(false);
          }}
        />
      )}
      {canManage && (
        <ResetPasswordDialog
          user={resetting}
          onClose={closeReset}
          onDone={(u) => {
            setNotice({ key: 'users.resetDone', params: { email: u.email } });
            setError(null);
            setResetting(null);
            load();
          }}
        />
      )}
    </main>
  );
}

function RolesLegend({ roles }) {
  const { t } = useTranslation();
  if (roles.length === 0) return null;
  return (
    <section className={ui.card} aria-labelledby="roles-title">
      <h2 id="roles-title" className={ui.sectionTitle}>
        {t('users.rolesTitle')}
      </h2>
      <dl className={styles.roles}>
        {roles.map((r) => (
          <div key={r.code}>
            <dt>
              <span className={ui.badge}>{r.name}</span> <span className={styles.roleDesc}>{r.description}</span>
            </dt>
            <dd>
              <ul className={ui.chips}>
                {r.permissions.map((p) => (
                  <li key={p} className={ui.badgeMuted} title={p}>
                    {permissionLabel(p, r.permission_names)}
                  </li>
                ))}
              </ul>
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

function CreateUserDialog({ open, roles, onClose, onCreated }) {
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [role, setRole] = useState('viewer');
  const [withPassword, setWithPassword] = useState(true);
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (open) {
      setEmail('');
      setName('');
      setRole(roles.some((r) => r.code === 'viewer') ? 'viewer' : (roles[0]?.code ?? ''));
      setWithPassword(true);
      setPassword('');
      setError(null);
    }
    // Only when the dialog opens: roles reloading on a language switch must not wipe the form.
  }, [open]);

  const tooShort = withPassword && password.length > 0 && password.length < MIN_PASSWORD_LENGTH;
  const canSubmit = !busy && email && role && (!withPassword || (password && !tooShort));

  async function onSubmit(e) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const body = { email: email.trim(), display_name: name.trim(), role };
      if (withPassword) body.password = password;
      onCreated(await usersApi.create(body));
    } catch (err) {
      setError(asError(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title={t('users.createTitle')} busy={busy} onClose={onClose}>
      <form className={ui.form} onSubmit={onSubmit} noValidate>
        {error && (
          <div className={ui.error} role="alert">
            {actionErrorMessage(error)}
          </div>
        )}
        <label className={ui.label}>
          {t('users.email')}
          <input className={ui.input} type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder={t('login.emailPlaceholder')} />
        </label>
        <label className={ui.label}>
          {t('users.displayName')}
          <input className={ui.input} value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className={ui.label}>
          {t('users.role')}
          <select className={ui.select} value={role} onChange={(e) => setRole(e.target.value)}>
            {roles.map((r) => (
              <option key={r.code} value={r.code}>
                {r.name}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.checkbox}>
          <input type="checkbox" checked={withPassword} onChange={(e) => setWithPassword(e.target.checked)} />
          {t('users.withPassword')}
        </label>
        {withPassword && (
          <PasswordField
            label={t('users.password')}
            hint={t('users.passwordHint', { count: MIN_PASSWORD_LENGTH })}
            className={ui.label}
            inputClassName={ui.input}
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        )}
        <div className={ui.actions}>
          <button type="button" className={ui.secondaryButton} onClick={onClose} disabled={busy}>
            {t('common.cancel')}
          </button>
          <button type="submit" className={ui.primaryButton} disabled={!canSubmit}>
            {busy ? t('users.creating') : t('users.createSubmit')}
          </button>
        </div>
      </form>
    </Modal>
  );
}

function ResetPasswordDialog({ user, onClose, onDone }) {
  const { t } = useTranslation();
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    setPassword('');
    setError(null);
  }, [user]);

  const tooShort = password.length > 0 && password.length < MIN_PASSWORD_LENGTH;

  async function onSubmit(e) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await usersApi.resetPassword(user.id, password);
      onDone(user);
    } catch (err) {
      setError(asError(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={user !== null} title={t('users.resetTitle', { email: user?.email ?? '' })} busy={busy} onClose={onClose}>
      <form className={ui.form} onSubmit={onSubmit} noValidate>
        {error && (
          <div className={ui.error} role="alert">
            {actionErrorMessage(error)}
          </div>
        )}
        <p className={ui.muted}>{t('users.resetText')}</p>
        <PasswordField
          label={t('users.newPassword')}
          hint={t('password.minLength', { count: MIN_PASSWORD_LENGTH })}
          className={ui.label}
          inputClassName={ui.input}
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <div className={ui.actions}>
          <button type="button" className={ui.secondaryButton} onClick={onClose} disabled={busy}>
            {t('common.cancel')}
          </button>
          <button type="submit" className={ui.primaryButton} disabled={busy || !password || tooShort}>
            {busy ? t('common.saving') : t('users.resetSubmit')}
          </button>
        </div>
      </form>
    </Modal>
  );
}
