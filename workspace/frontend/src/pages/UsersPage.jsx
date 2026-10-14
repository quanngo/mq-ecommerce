import { useCallback, useEffect, useState } from 'react';
import { ApiError } from '../api/client';
import { usersApi } from '../api/users';
import { useAuth } from '../auth/AuthContext';
import { actionErrorMessage, PERMISSION_LABELS } from '../auth/errorMessages';
import { MIN_PASSWORD_LENGTH } from './ChangePasswordPage';
import { Modal } from './Modal';
import { PasswordField } from './PasswordField';
import { formatDateTime } from './ProfilePage';
import { Spinner } from './Spinner';
import styles from './UsersPage.module.css';
import ui from './ui.module.css';

const errorText = (err) => actionErrorMessage(err instanceof ApiError ? err.errorCode : null);

export function UsersPage() {
  const { state, can } = useAuth();
  const canManage = can('users.manage');
  const me = state.user?.id;

  const [users, setUsers] = useState(null);
  const [roles, setRoles] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [savingId, setSavingId] = useState(null);
  const [creating, setCreating] = useState(false);
  const [resetting, setResetting] = useState(null); // ManagedUser | null

  const load = useCallback(async () => {
    try {
      const [u, r] = await Promise.all([usersApi.list(), usersApi.roles()]);
      setUsers(u);
      setRoles(r);
    } catch (err) {
      setError(errorText(err));
      setUsers([]);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const replaceUser = (updated) => setUsers((list) => list.map((u) => (u.id === updated.id ? updated : u)));

  async function change(user, changes, successText) {
    setError(null);
    setNotice(null);
    setSavingId(user.id);
    try {
      replaceUser(await usersApi.update(user.id, changes));
      setNotice(successText);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setSavingId(null);
    }
  }

  const closeCreate = useCallback(() => setCreating(false), []);
  const closeReset = useCallback(() => setResetting(null), []);

  if (users === null) return <Spinner label="Đang tải danh sách người dùng…" />;

  return (
    <main className={ui.main}>
      <div className={styles.header}>
        <h1 className={ui.title}>Người dùng</h1>
        {canManage && (
          <button className={ui.primaryButton} type="button" onClick={() => setCreating(true)}>
            + Tạo người dùng
          </button>
        )}
      </div>

      {error && (
        <div className={ui.error} role="alert">
          {error}
        </div>
      )}
      {notice && (
        <div className={ui.success} role="status">
          {notice}
        </div>
      )}

      <section className={`${ui.card} ${styles.tableCard}`}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th scope="col">Người dùng</th>
              <th scope="col">Vai trò</th>
              <th scope="col">Đăng nhập bằng</th>
              <th scope="col">Trạng thái</th>
              <th scope="col">Đăng nhập gần nhất</th>
              {canManage && <th scope="col" aria-label="Thao tác" />}
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
                      {u.display_name || '—'} {isMe && <span className={ui.badgeMuted}>Bạn</span>}
                    </div>
                    <div className={styles.userEmail}>{u.email}</div>
                  </td>
                  <td>
                    {canManage && !isMe ? (
                      <select
                        className={ui.select}
                        aria-label={`Vai trò của ${u.email}`}
                        value={u.role.code}
                        disabled={saving}
                        onChange={(e) => change(u, { role: e.target.value }, `Đã đổi vai trò của ${u.email}.`)}
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
                    {u.has_password && <span className={ui.badgeMuted}>Mật khẩu</span>}
                    {u.google_linked && <span className={ui.badgeMuted}>Google</span>}
                    {!u.has_password && !u.google_linked && <span className={ui.badgeMuted}>Google (chưa đăng nhập)</span>}
                    </div>
                  </td>
                  <td>
                    {u.status === 'active' ? <span className={ui.badgeMuted}>Hoạt động</span> : <span className={ui.badgeDanger}>Đã khoá</span>}
                    {u.locked_until && <div className={styles.lockNote}>Tạm khoá đến {formatDateTime(u.locked_until)}</div>}
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
                                u.status === 'active' ? `Đã khoá ${u.email}.` : `Đã mở khoá ${u.email}.`,
                              )
                            }
                          >
                            {u.status === 'active' ? 'Khoá' : 'Mở khoá'}
                          </button>
                          <button type="button" className={ui.secondaryButton} disabled={saving} onClick={() => setResetting(u)}>
                            Đặt lại mật khẩu
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
            setNotice(`Đã tạo tài khoản ${u.email}.`);
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
            setNotice(`Đã đặt lại mật khẩu cho ${u.email}. Người dùng đã bị đăng xuất khỏi mọi thiết bị.`);
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
  if (roles.length === 0) return null;
  return (
    <section className={ui.card} aria-labelledby="roles-title">
      <h2 id="roles-title" className={ui.sectionTitle}>
        Vai trò và quyền
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
                    {PERMISSION_LABELS[p] ?? p}
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
  }, [open, roles]);

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
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} title="Tạo người dùng" busy={busy} onClose={onClose}>
      <form className={ui.form} onSubmit={onSubmit} noValidate>
        {error && (
          <div className={ui.error} role="alert">
            {error}
          </div>
        )}
        <label className={ui.label}>
          Email
          <input className={ui.input} type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="ten@mq.com" />
        </label>
        <label className={ui.label}>
          Tên hiển thị
          <input className={ui.input} value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className={ui.label}>
          Vai trò
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
          Đặt mật khẩu (bỏ chọn nếu người dùng chỉ đăng nhập bằng Google)
        </label>
        {withPassword && (
          <PasswordField
            label="Mật khẩu"
            hint={`Tối thiểu ${MIN_PASSWORD_LENGTH} ký tự. Gửi riêng cho người dùng và nhắc họ đổi sau khi đăng nhập.`}
            className={ui.label}
            inputClassName={ui.input}
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        )}
        <div className={ui.actions}>
          <button type="button" className={ui.secondaryButton} onClick={onClose} disabled={busy}>
            Huỷ
          </button>
          <button type="submit" className={ui.primaryButton} disabled={!canSubmit}>
            {busy ? 'Đang tạo…' : 'Tạo'}
          </button>
        </div>
      </form>
    </Modal>
  );
}

function ResetPasswordDialog({ user, onClose, onDone }) {
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
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={user !== null} title={`Đặt lại mật khẩu cho ${user?.email ?? ''}`} busy={busy} onClose={onClose}>
      <form className={ui.form} onSubmit={onSubmit} noValidate>
        {error && (
          <div className={ui.error} role="alert">
            {error}
          </div>
        )}
        <p className={ui.muted}>Người dùng sẽ bị đăng xuất khỏi mọi thiết bị và phải dùng mật khẩu mới.</p>
        <PasswordField
          label="Mật khẩu mới"
          hint={`Tối thiểu ${MIN_PASSWORD_LENGTH} ký tự.`}
          className={ui.label}
          inputClassName={ui.input}
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <div className={ui.actions}>
          <button type="button" className={ui.secondaryButton} onClick={onClose} disabled={busy}>
            Huỷ
          </button>
          <button type="submit" className={ui.primaryButton} disabled={busy || !password || tooShort}>
            {busy ? 'Đang lưu…' : 'Đặt lại'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
