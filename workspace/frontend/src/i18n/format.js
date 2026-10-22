// Locale-aware formatting for the current UI language. Call these during
// render in a component that uses useTranslation(), so they re-run when the
// language changes. Times are shown in Vietnam time, whatever the browser's zone.

import { currentIntlLocale } from './index';

export const TIME_ZONE = 'Asia/Ho_Chi_Minh';

/** @param {string | null | undefined} iso RFC3339 */
export function formatDateTime(iso) {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return new Intl.DateTimeFormat(currentIntlLocale(), { dateStyle: 'medium', timeStyle: 'short', timeZone: TIME_ZONE }).format(d);
}

/** @param {number} n */
export function formatNumber(n) {
  return new Intl.NumberFormat(currentIntlLocale()).format(Number(n) || 0);
}

/** Amounts are always VND; only the number format follows the language. */
export function formatMoney(amount) {
  return new Intl.NumberFormat(currentIntlLocale(), { style: 'currency', currency: 'VND', maximumFractionDigits: 0 }).format(Number(amount) || 0);
}

/** File size for people: "850 KB", "1,2 MB" / "1.2 MB". @param {number} bytes */
export function formatBytes(bytes) {
  const n = Number(bytes) || 0;
  const nf = (v, digits) => new Intl.NumberFormat(currentIntlLocale(), { maximumFractionDigits: digits }).format(v);
  if (n < 1024) return `${nf(n, 0)} B`;
  if (n < 1024 * 1024) return `${nf(n / 1024, 0)} KB`;
  return `${nf(n / (1024 * 1024), 1)} MB`;
}
