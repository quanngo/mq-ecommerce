import { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { ConfirmDialog } from './ConfirmDialog';
import styles from './ProfilePage.module.css';

const METHOD_LABEL = {
  google: 'Google',
  password: 'Email & mật khẩu',
};

const CONFIRM = {
  logout: {
    title: 'Đăng xuất?',
    message: 'Bạn sẽ đăng xuất khỏi trình duyệt này (mọi tab đang mở).',
    confirmLabel: 'Đăng xuất',
    danger: false,
  },
  logoutAll: {
    title: 'Đăng xuất khỏi tất cả thiết bị?',
    message: 'Mọi phiên đăng nhập của bạn trên mọi thiết bị và trình duyệt sẽ bị kết thúc, kể cả phiên này.',
    confirmLabel: 'Đăng xuất tất cả',
    danger: true,
  },
};

export function ProfilePage() {
  const { state, logout, logoutAll } = useAuth();
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(null); // null | 'logout' | 'logoutAll'
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  const closeDialog = useCallback(() => setConfirming(null), []);

  if (state.status !== 'authenticated') return null; // ProtectedRoute guarantees this
  const user = state.user;
  const name = user.display_name || user.email;

  async function onConfirm() {
    setBusy(true);
    setError(null);
    try {
      if (confirming === 'logoutAll') {
        await logoutAll();
      } else {
        await logout();
      }
      navigate('/login', { replace: true });
    } catch (err) {
      setConfirming(null);
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
    <div className={styles.layout}>
      <header className={styles.topbar}>
        <span className={styles.brand}>MQ Admin</span>
        <button className={styles.logout} type="button" onClick={() => setConfirming('logout')}>
          Đăng xuất
        </button>
      </header>

      <main className={styles.main}>
        <h1 className={styles.greeting}>Xin chào, {name}</h1>

        {error && (
          <div className={styles.error} role="alert">
            {error}
          </div>
        )}

        <section className={styles.card} aria-label="Thông tin tài khoản">
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
              <dt>Phương thức đăng nhập</dt>
              <dd>{METHOD_LABEL[user.auth_method] ?? user.auth_method}</dd>
            </div>
            <div>
              <dt>Đăng nhập lúc</dt>
              <dd>{formatDateTime(user.last_login_at)}</dd>
            </div>
          </dl>
        </section>

        <section className={styles.card} aria-labelledby="security-title">
          <h2 id="security-title" className={styles.sectionTitle}>
            Bảo mật
          </h2>
          <div className={styles.securityRow}>
            <p className={styles.securityText}>
              Nghi ngờ tài khoản đang đăng nhập ở nơi khác, hoặc quên đăng xuất trên máy dùng chung? Kết thúc mọi phiên đăng nhập của bạn.
            </p>
            <button className={styles.dangerButton} type="button" onClick={() => setConfirming('logoutAll')}>
              Đăng xuất tất cả thiết bị
            </button>
          </div>
        </section>
      </main>

      <ConfirmDialog
        open={confirming !== null}
        title={CONFIRM[confirming]?.title}
        message={CONFIRM[confirming]?.message}
        confirmLabel={busy ? 'Đang đăng xuất…' : CONFIRM[confirming]?.confirmLabel}
        danger={CONFIRM[confirming]?.danger}
        busy={busy}
        onConfirm={onConfirm}
        onCancel={closeDialog}
      />
    </div>
  );
}

function initials(name) {
  const parts = name.replace(/@.*/, '').split(/[\s._-]+/).filter(Boolean);
  return ((parts[0]?.[0] ?? '') + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase() || '?';
}

function formatDateTime(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return new Intl.DateTimeFormat('vi-VN', { dateStyle: 'medium', timeStyle: 'short' }).format(d);
}
