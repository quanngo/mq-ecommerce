import { useEffect, useId, useRef } from 'react';
import styles from './ConfirmDialog.module.css';

/** Generic modal (same look as ConfirmDialog). Escape / backdrop close unless busy. */
export function Modal({ open, title, busy = false, onClose, children }) {
  const titleId = useId();
  const boxRef = useRef(null);

  useEffect(() => {
    if (!open) return undefined;
    const previous = document.activeElement;
    boxRef.current?.querySelector('input, select, button')?.focus();
    const onKey = (e) => {
      if (e.key === 'Escape' && !busy) onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('keydown', onKey);
      if (previous instanceof HTMLElement) previous.focus();
    };
  }, [open, busy, onClose]);

  if (!open) return null;
  return (
    <div className={styles.backdrop} onMouseDown={(e) => e.target === e.currentTarget && !busy && onClose()}>
      <div ref={boxRef} className={styles.dialog} role="dialog" aria-modal="true" aria-labelledby={titleId}>
        <h2 id={titleId} className={styles.title}>
          {title}
        </h2>
        {children}
      </div>
    </div>
  );
}
