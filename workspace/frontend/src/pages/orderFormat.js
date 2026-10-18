// Labels and formatting for orders. Codes mirror workspace/backend/utils/constants.go;
// the wording is in src/locales/*.json (orders.leadStatus, saleOrderStatus, paymentMethod).
// Call these during render in a component that uses useTranslation().

import i18n from '../i18n';

export { formatMoney } from '../i18n/format';

/** Filterable marketplaces (utils.Plaform). Brand names: not translated. */
export const MARKET_PLACES = [
  { value: 'Tiktok', label: 'TikTok Shop' },
  { value: 'Lazada', label: 'Lazada' },
  { value: 'Shopee', label: 'Shopee' },
];

/** utils.LeadStatus values, in filter order. */
export const LEAD_STATUSES = [1, 2, 3, 4];

/** @param {string} code */
export function marketPlaceLabel(code) {
  return MARKET_PLACES.find((m) => m.value === code)?.label ?? (code || '—');
}

/** @param {number} status */
export function leadStatusLabel(status) {
  const key = `orders.leadStatus.${status}`;
  return i18n.exists(key) ? i18n.t(key) : i18n.t('orders.leadStatus.other', { status });
}

/** utils.SaleOrderStatus @param {number | null} status */
export function saleOrderStatusLabel(status) {
  if (status === null || status === undefined) return '—';
  return i18n.t(`orders.saleOrderStatus.${status}`, { defaultValue: String(status) });
}

/** utils.PAYMENT_METHOD_* @param {number} method */
export function paymentMethodLabel(method) {
  if (!method) return '—';
  return i18n.t(`orders.paymentMethod.${method}`, { defaultValue: String(method) });
}

/** The marketplace order id, or the internal id for sheet imports without one. */
export function orderLabel(order) {
  return order.order_id || `#${order.id}`;
}
