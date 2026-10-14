// Labels and formatting for orders. Codes mirror workspace/backend/utils/constants.go.

/** Filterable marketplaces (utils.Plaform). */
export const MARKET_PLACES = [
  { value: 'Tiktok', label: 'TikTok Shop' },
  { value: 'Lazada', label: 'Lazada' },
  { value: 'Shopee', label: 'Shopee' },
];

/** utils.LeadStatus */
export const LEAD_STATUS_LABELS = {
  1: 'Chờ hàng',
  2: 'Đã tạo đơn',
  3: 'Hoàn tiền',
  4: 'Đã huỷ',
};

/** utils.SaleOrderStatus */
const SALE_ORDER_STATUS_LABELS = {
  41: 'Mới',
  42: 'Chờ xử lý',
  43: 'Đã xác nhận',
  44: 'Đã huỷ',
  45: 'Thành công',
  46: 'Chưa phân công',
  48: 'Tạo vận đơn lỗi',
  49: 'Hoàn tất',
  50: 'Hoàn tiền',
  357: 'Chờ xác nhận',
};

/** utils.PAYMENT_METHOD_* */
const PAYMENT_METHOD_LABELS = {
  1: 'COD',
  8: 'Thanh toán qua sàn',
};

/** @param {string} code */
export function marketPlaceLabel(code) {
  return MARKET_PLACES.find((m) => m.value === code)?.label ?? (code || '—');
}

/** @param {number} status */
export function leadStatusLabel(status) {
  return LEAD_STATUS_LABELS[status] ?? `Trạng thái ${status}`;
}

/** @param {number | null} status */
export function saleOrderStatusLabel(status) {
  if (status === null || status === undefined) return '—';
  return SALE_ORDER_STATUS_LABELS[status] ?? String(status);
}

/** @param {number} method */
export function paymentMethodLabel(method) {
  return PAYMENT_METHOD_LABELS[method] ?? (method ? String(method) : '—');
}

const vnd = new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND', maximumFractionDigits: 0 });

/** @param {number} amount */
export function formatMoney(amount) {
  return vnd.format(Number(amount) || 0);
}

/** The marketplace order id, or the internal id for sheet imports without one. */
export function orderLabel(order) {
  return order.order_id || `#${order.id}`;
}
