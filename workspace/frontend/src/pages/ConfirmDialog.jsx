import { useEffect, useId, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import styles from './ConfirmDialog.module.css';

/**
 * Modal confirmation. Escape or clicking the backdrop cancels; focus starts on
 * Cancel so an accidental Enter doesn't confirm a destructive action.
 */
export function ConfirmDialog({ open, title, message, confirmLabel, cancelLabel, busy = false, danger = false, onConfirm, onCancel }) {
  const { t } = useTranslation();
  const titleId = useId();
  const messageId = useId();
  const cancelRef = useRef(null);

  useEffect(() => {
    if (!open) return undefined;
    const previous = document.activeElement;
    cancelRef.current?.focus();
    const onKey = (e) => {
      if (e.key === 'Escape' && !busy) onCancel();
    };
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('keydown', onKey);
      if (previous instanceof HTMLElement) previous.focus();
    };
  }, [open, busy, onCancel]);

  if (!open) return null;

  return (
    <div className={styles.backdrop} onMouseDown={(e) => e.target === e.currentTarget && !busy && onCancel()}>
      <div className={styles.dialog} role="alertdialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={messageId}>
        <h2 id={titleId} className={styles.title}>
          {title}
        </h2>
        <p id={messageId} className={styles.message}>
          {message}
        </p>
        <div className={styles.actions}>
          <button ref={cancelRef} type="button" className={styles.cancel} onClick={onCancel} disabled={busy}>
            {cancelLabel ?? t('common.cancel')}
          </button>
          <button type="button" className={danger ? styles.danger : styles.confirm} onClick={onConfirm} disabled={busy}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
