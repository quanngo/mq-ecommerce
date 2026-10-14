import { useCallback, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { PERMISSION_LABELS } from '../auth/errorMessages';
import { ConfirmDialog } from './ConfirmDialog';
import styles from './ProfilePage.module.css';
import ui from './ui.module.css';

const METHOD_LABEL = {
  google: 'Google',
  password: 'Email & mật khẩu',
};

export function ProfilePage() {
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
      setError(
        err instanceof ApiError && err.errorCode === 'NETWORK_ERROR'
          ? 'Không kết nối được máy chủ. Các thiết bị khác vẫn đang đăng nhập — vui lòng thử lại.'
          : 'Không thể đăng xuất khỏi các thiết bị khác lúc này. Vui lòng thử lại.',
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className={ui.main}>
      <h1 className={ui.title}>Xin chào, {name}</h1>

      {error && (
        <div className={ui.error} role="alert">
          {error}
        </div>
      )}

      <section className={ui.card} aria-label="Thông tin tài khoản">
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
            <dt>Vai trò</dt>
            <dd>
              <span className={ui.badge}>{user.role?.name || user.role?.code || '—'}</span>
            </dd>
          </div>
          <div>
            <dt>Phương thức đăng nhập</dt>
            <dd>{METHOD_LABEL[user.auth_method] ?? user.auth_method}</dd>
          </div>
          <div>
            <dt>Đăng nhập lúc</dt>
            <dd>{formatDateTime(user.last_login_at)}</dd>
          </div>
        </dl>

        <div className={styles.permissions}>
          <div className={styles.permissionsLabel}>Quyền</div>
          {permissions.length === 0 ? (
            <p className={ui.muted}>Vai trò này chưa có quyền nào.</p>
          ) : (
            <ul className={ui.chips} aria-label="Quyền của bạn">
              {permissions.map((p) => (
                <li key={p} className={ui.badgeMuted} title={p}>
                  {PERMISSION_LABELS[p] ?? p}
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      <section className={ui.card} aria-labelledby="security-title">
        <h2 id="security-title" className={ui.sectionTitle}>
          Bảo mật
        </h2>
        <div className={styles.securityRow}>
          <p className={styles.securityText}>Đổi mật khẩu đăng nhập bằng email. Các thiết bị khác sẽ bị đăng xuất.</p>
          <Link to="/account/password" className={ui.secondaryButton}>
            Đổi mật khẩu
          </Link>
        </div>
        <div className={styles.securityRow}>
          <p className={styles.securityText}>
            Nghi ngờ tài khoản đang đăng nhập ở nơi khác, hoặc quên đăng xuất trên máy dùng chung? Kết thúc mọi phiên đăng nhập của bạn.
          </p>
          <button className={ui.dangerButton} type="button" onClick={() => setConfirming(true)}>
            Đăng xuất tất cả thiết bị
          </button>
        </div>
      </section>

      <ConfirmDialog
        open={confirming}
        title="Đăng xuất khỏi tất cả thiết bị?"
        message="Mọi phiên đăng nhập của bạn trên mọi thiết bị và trình duyệt sẽ bị kết thúc, kể cả phiên này."
        confirmLabel={busy ? 'Đang đăng xuất…' : 'Đăng xuất tất cả'}
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

export function formatDateTime(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return new Intl.DateTimeFormat('vi-VN', { dateStyle: 'medium', timeStyle: 'short' }).format(d);
}
