import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../auth/AuthContext';
import { currentLanguage, LANGUAGES } from '../i18n';
import styles from './LanguageSwitcher.module.css';

/**
 * VI / EN toggle. Signed in, the choice is saved on the account (useAuth().setLanguage);
 * on the login page it is remembered in this browser only.
 *
 * @param {object} props
 * @param {(failed: boolean) => void} [props.onSaveResult]  told whether saving to the account failed
 * @param {string} [props.className]
 */
export function LanguageSwitcher({ onSaveResult, className }) {
  const { t } = useTranslation();
  const { setLanguage } = useAuth();
  const [busy, setBusy] = useState(false);
  const active = currentLanguage();

  async function choose(code) {
    if (code === active || busy) return;
    setBusy(true);
    try {
      await setLanguage(code);
      onSaveResult?.(false);
    } catch {
      onSaveResult?.(true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={`${styles.switcher} ${className ?? ''}`} role="group" aria-label={t('common.language')} aria-busy={busy}>
      {LANGUAGES.map((l) => (
        <button
          key={l.code}
          type="button"
          lang={l.code}
          className={l.code === active ? `${styles.option} ${styles.active}` : styles.option}
          aria-pressed={l.code === active}
          aria-label={l.label}
          title={l.label}
          disabled={busy}
          onClick={() => choose(l.code)}
        >
          {l.short}
        </button>
      ))}
    </div>
  );
}
