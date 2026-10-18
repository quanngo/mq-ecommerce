import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom';
import { authApi, googleLoginUrl, safeNextPath } from '../api/auth';
import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { authErrorMessage } from '../auth/errorMessages';
import { LanguageSwitcher } from './LanguageSwitcher';
import { PasswordField } from './PasswordField';
import { Spinner } from './Spinner';
import styles from './LoginPage.module.css';

export function LoginPage() {
  const { t } = useTranslation();
  const { state, login } = useAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const next = safeNextPath(searchParams.get('next'));

  const [providers, setProviders] = useState(null);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  // ?error=CODE comes from the Google callback redirect; read it once, then drop it from the URL.
  // The code (or ApiError) is kept and translated at render, so switching language re-translates it.
  const [error, setError] = useState(() => searchParams.get('error'));

  useEffect(() => {
    if (searchParams.has('error')) {
      const cleaned = new URLSearchParams(searchParams);
      cleaned.delete('error');
      setSearchParams(cleaned, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  useEffect(() => {
    authApi
      .providers()
      .then(setProviders)
      .catch(() => setProviders({ google: false, password: true }));
  }, []);

  if (state.status === 'loading') return <Spinner label={t('common.loading')} />;
  if (state.status === 'authenticated') return <Navigate to={next} replace />;

  async function onSubmit(e) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await login(email.trim(), password, rememberMe);
      navigate(next, { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err : 'INTERNAL_ERROR');
      setPassword('');
    } finally {
      setSubmitting(false);
    }
  }

  const errorMessage = authErrorMessage(error);

  return (
    <main className={styles.page}>
      <LanguageSwitcher className={styles.language} />
      <section className={styles.card} aria-labelledby="login-title">
        <header className={styles.header}>
          <span className={styles.logo} aria-hidden="true">
            MQ
          </span>
          <h1 id="login-title" className={styles.title}>
            {t('login.title')}
          </h1>
          <p className={styles.subtitle}>{t('common.appName')}</p>
        </header>

        {errorMessage && (
          <div className={styles.error} role="alert">
            {errorMessage}
          </div>
        )}

        <form className={styles.form} onSubmit={onSubmit} noValidate>
          <label className={styles.label}>
            {t('login.email')}
            <input
              className={styles.input}
              type="email"
              name="email"
              autoComplete="username"
              inputMode="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('login.emailPlaceholder')}
            />
          </label>
          <PasswordField
            label={t('login.password')}
            className={styles.label}
            inputClassName={styles.input}
            name="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          {/* Applies to both the password form and the Google button below. */}
          <label className={styles.remember}>
            <input type="checkbox" checked={rememberMe} onChange={(e) => setRememberMe(e.target.checked)} />
            {t('login.remember')}
          </label>
          <button className={styles.submit} type="submit" disabled={submitting || !email || !password}>
            {submitting ? t('login.submitting') : t('login.submit')}
          </button>
        </form>

        {/* Google (OIDC) is always shown once providers are known; it is only
            usable when the backend has it configured (auth.google + credentials). */}
        {providers && (
          <>
            <div className={styles.divider}>
              <span>{t('login.or')}</span>
            </div>
            {providers.google ? (
              <a className={styles.google} href={googleLoginUrl(next, rememberMe)}>
                <GoogleIcon />
                {t('login.google')}
              </a>
            ) : (
              <>
                <button className={styles.google} type="button" disabled aria-describedby="google-off">
                  <GoogleIcon />
                  {t('login.google')}
                </button>
                <p id="google-off" className={styles.googleOff}>
                  {t('login.googleOff')}
                </p>
              </>
            )}
          </>
        )}
      </section>
    </main>
  );
}

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 48 48" aria-hidden="true">
      <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3C33.7 32.7 29.2 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.3-.1-2.4-.4-3.5z" />
      <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.7 15.1 19 12 24 12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 16.3 4 9.7 8.3 6.3 14.7z" />
      <path fill="#4CAF50" d="M24 44c5.2 0 9.9-2 13.4-5.2l-6.2-5.2C29.2 35.1 26.7 36 24 36c-5.2 0-9.6-3.3-11.3-8l-6.5 5C9.5 39.6 16.2 44 24 44z" />
      <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.2-2.2 4.2-4.1 5.6l6.2 5.2C37 39.2 44 34 44 24c0-1.3-.1-2.4-.4-3.5z" />
    </svg>
  );
}
