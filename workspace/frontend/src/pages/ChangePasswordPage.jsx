import { useState } from 'react';
import { Link } from 'react-router-dom';
import { authApi } from '../api/auth';
import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { actionErrorMessage } from '../auth/errorMessages';
import { PasswordField } from './PasswordField';
import ui from './ui.module.css';

export const MIN_PASSWORD_LENGTH = 10; // auth.min-password-length on the backend (default)

export function ChangePasswordPage() {
  const { state } = useAuth();
  // A Google sign-in may belong to an account without a password yet; the
  // backend then accepts an empty current password and sets the first one.
  const googleSession = state.user?.auth_method === 'google';

  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);
  const [done, setDone] = useState(false);

  const mismatch = confirm !== '' && next !== confirm;
  const tooShort = next !== '' && next.length < MIN_PASSWORD_LENGTH;
  const canSubmit = !busy && next && confirm && !mismatch && !tooShort && (googleSession || current);

  async function onSubmit(e) {
    e.preventDefault();
    setError(null);
    setDone(false);
    setBusy(true);
    try {
      await authApi.changePassword(current, next);
      setDone(true);
      setCurrent('');
      setNext('');
      setConfirm('');
    } catch (err) {
      setError(actionErrorMessage(err instanceof ApiError ? err.errorCode : null));
      setCurrent('');
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className={`${ui.main} ${ui.narrow}`}>
      <h1 className={ui.title}>Đổi mật khẩu</h1>

      {done && (
        <div className={ui.success} role="status">
          Đã đổi mật khẩu. Các thiết bị khác đã bị đăng xuất.
        </div>
      )}
      {error && (
        <div className={ui.error} role="alert">
          {error}
        </div>
      )}

      <form className={`${ui.card} ${ui.form}`} onSubmit={onSubmit} noValidate>
        <PasswordField
          label="Mật khẩu hiện tại"
          hint={googleSession ? 'Bỏ trống nếu tài khoản của bạn chưa từng có mật khẩu (chỉ đăng nhập Google).' : null}
          className={ui.label}
          inputClassName={ui.input}
          autoComplete="current-password"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
        />
        <PasswordField
          label="Mật khẩu mới"
          hint={`Tối thiểu ${MIN_PASSWORD_LENGTH} ký tự.`}
          className={ui.label}
          inputClassName={ui.input}
          autoComplete="new-password"
          value={next}
          onChange={(e) => setNext(e.target.value)}
          aria-invalid={tooShort}
        />
        <PasswordField
          label="Nhập lại mật khẩu mới"
          className={ui.label}
          inputClassName={ui.input}
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          aria-invalid={mismatch}
          describedBy={mismatch ? 'confirm-mismatch' : undefined}
        />
        {/* Outside the field, so its accessible name stays "Nhập lại mật khẩu mới". */}
        {mismatch && (
          <span id="confirm-mismatch" className={ui.hint}>
            Mật khẩu nhập lại không khớp.
          </span>
        )}
        <div className={ui.actions}>
          <Link to="/profile" className={ui.secondaryButton}>
            Quay lại
          </Link>
          <button className={ui.primaryButton} type="submit" disabled={!canSubmit}>
            {busy ? 'Đang lưu…' : 'Đổi mật khẩu'}
          </button>
        </div>
      </form>
    </main>
  );
}
