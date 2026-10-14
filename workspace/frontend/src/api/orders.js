import { apiRequest } from './client';

/**
 * models.OrderSummary in the backend: a row of GET /orders.
 * @typedef {object} OrderSummary
 * @property {number} id             ecom_order id (use this in /orders/:id)
 * @property {string} market_place   'Tiktok' | 'Lazada' | 'Shopee' (sheet imports may hold other values)
 * @property {string} shop_id
 * @property {string} order_id       marketplace order id; '' for some sheet imports
 * @property {string} tracking_code
 * @property {number} lead_status    1-4, see LEAD_STATUS_LABELS
 * @property {string} do_status      latest delivery status from the marketplace
 * @property {number | null} so_status
 * @property {string} customer_name
 * @property {string} phone
 * @property {number} total_amount
 * @property {number} payment_method
 * @property {number} item_count
 * @property {string} created_at
 * @property {string} updated_at
 */

/**
 * models.OrderDetail: GET /orders/:id.
 * @typedef {OrderSummary & {
 *   address: string, province: string, district: string, sub_district: string, neighborhood: string,
 *   warehouse: string, shipping_label: string, comment: string, discount: number, shipping_fee: number,
 *   items: { id: number, name: string, sku: string, quantity: number, price: number }[]
 * }} OrderDetail
 */

/**
 * Filters as GET /orders takes them; empty values are left out.
 * @typedef {object} OrderQuery
 * @property {string} [q]
 * @property {string} [market_place]
 * @property {string} [lead_status]
 * @property {string} [from]       YYYY-MM-DD
 * @property {string} [to]         YYYY-MM-DD
 * @property {string | number} [page]
 * @property {string | number} [page_size]
 */

/** Query-string keys GET /orders understands, in URL order. */
export const ORDER_QUERY_KEYS = ['q', 'market_place', 'lead_status', 'from', 'to', 'page', 'page_size'];

/** @param {OrderQuery} query */
function toQueryString(query) {
  const params = new URLSearchParams();
  for (const key of ORDER_QUERY_KEYS) {
    const v = query[key];
    if (v !== undefined && v !== null && String(v).trim() !== '') params.set(key, String(v).trim());
  }
  const s = params.toString();
  return s ? `?${s}` : '';
}

export const ordersApi = {
  /**
   * @param {OrderQuery} query
   * @returns {Promise<{ orders: OrderSummary[], total: number, page: number, page_size: number }>}
   */
  list: async (query = {}) => {
    const res = await apiRequest('GET', `/orders${toQueryString(query)}`);
    return { orders: res?.orders ?? [], total: res?.total ?? 0, page: res?.page ?? 1, page_size: res?.page_size ?? 20 };
  },

  /**
   * @param {number | string} id
   * @returns {Promise<OrderDetail>}
   */
  get: async (id) => (await apiRequest('GET', `/orders/${encodeURIComponent(id)}`))?.order,
};
