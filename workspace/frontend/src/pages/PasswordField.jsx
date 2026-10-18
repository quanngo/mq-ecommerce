import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import styles from './PasswordField.module.css';

/**
 * A labelled password input with a show/hide button.
 *
 * The button sits outside the <label>, so the field's accessible name stays
 * exactly `label` (tests and screen readers find it by that); the hint is
 * linked with aria-describedby. Showing the password only flips the input
 * type: the value never leaves the input.
 *
 * @param {object} props
 * @param {string} props.label
 * @param {import('react').ReactNode} [props.hint]
 * @param {string} [props.className]       wrapper (the page's label style: column, gap)
 * @param {string} [props.inputClassName]  the page's input style
 * @param {string} [props.describedBy]     extra aria-describedby id (e.g. an error below the field)
 * All other props (value, onChange, autoComplete, name, required, aria-invalid…) go to the <input>.
 */
export function PasswordField({ label, hint, className, inputClassName, describedBy, ...inputProps }) {
  const { t } = useTranslation();
  const id = useId();
  const hintId = `${id}-hint`;
  const [visible, setVisible] = useState(false);
  const described = [hint ? hintId : null, describedBy].filter(Boolean).join(' ') || undefined;

  return (
    <div className={className}>
      <label htmlFor={id}>{label}</label>
      {hint && (
        <span id={hintId} className={styles.hint}>
          {hint}
        </span>
      )}
      <div className={styles.wrap}>
        <input
          {...inputProps}
          id={id}
          className={`${inputClassName ?? ''} ${styles.input}`}
          type={visible ? 'text' : 'password'}
          aria-describedby={described}
          autoCapitalize="off"
          autoCorrect="off"
          spellCheck={false}
        />
        <button
          type="button"
          className={styles.toggle}
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? t('password.hide') : t('password.show')}
          aria-controls={id}
          title={visible ? t('password.hide') : t('password.show')}
        >
          {visible ? <EyeOffIcon /> : <EyeIcon />}
        </button>
      </div>
    </div>
  );
}

function EyeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function EyeOffIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M10.6 5.1A10.4 10.4 0 0 1 12 5c6.5 0 10 7 10 7a17.6 17.6 0 0 1-2.6 3.6" />
      <path d="M6.6 6.6C3.6 8.5 2 12 2 12s3.5 7 10 7a9.7 9.7 0 0 0 5.4-1.6" />
      <path d="M9.9 9.9a3 3 0 0 0 4.2 4.2" />
      <path d="M2 2l20 20" />
    </svg>
  );
}
