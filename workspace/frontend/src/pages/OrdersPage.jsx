import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { ordersApi } from '../api/orders';
import { useAuth } from '../auth/AuthContext';
import { orderErrorMessage } from '../auth/errorMessages';
import { LEAD_STATUS_LABELS, MARKET_PLACES, formatMoney, leadStatusLabel, marketPlaceLabel, orderLabel } from './orderFormat';
import { formatDateTime } from './ProfilePage';
import { Spinner } from './Spinner';
import styles from './OrdersPage.module.css';
import ui from './ui.module.css';

const FILTER_KEYS = ['q', 'market_place', 'lead_status', 'from', 'to'];
const EMPTY_FILTERS = { q: '', market_place: '', lead_status: '', from: '', to: '' };

const filtersFrom = (params) => Object.fromEntries(FILTER_KEYS.map((k) => [k, params.get(k) ?? '']));

/**
 * Orders list (orders.read). Filters and page live in the URL, so Back and
 * shared links keep them. The order id links to the detail page only with
 * orders.detail.
 */
export function OrdersPage() {
  const { can } = useAuth();
  const canDetail = can('orders.detail');
  const [params, setParams] = useSearchParams();
  const search = params.toString();

  const [form, setForm] = useState(() => filtersFrom(params));
  const [result, setResult] = useState(null); // { orders, total, page, page_size } | null while loading
  const [error, setError] = useState(null);

  // Keep the form in step with the URL (Back/Forward, "Xoá lọc").
  useEffect(() => {
    setForm(filtersFrom(new URLSearchParams(search)));
  }, [search]);

  useEffect(() => {
    let cancelled = false;
    setResult(null);
    setError(null);
    ordersApi
      .list(Object.fromEntries(new URLSearchParams(search)))
      .then((res) => !cancelled && setResult(res))
      .catch((err) => {
        if (cancelled) return;
        setError(orderErrorMessage(err instanceof ApiError ? err.errorCode : null));
        setResult({ orders: [], total: 0, page: 1, page_size: 20 });
      });
    return () => {
      cancelled = true;
    };
  }, [search]);

  function apply(filters, page = 1) {
    const next = new URLSearchParams();
    for (const k of FILTER_KEYS) {
      const v = (filters[k] ?? '').trim();
      if (v) next.set(k, v);
    }
    if (page > 1) next.set('page', String(page));
    setParams(next);
  }

  function onSubmit(e) {
    e.preventDefault();
    apply(form);
  }

  const set = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }));
  const hasFilters = FILTER_KEYS.some((k) => params.get(k));

  const pages = result ? Math.max(1, Math.ceil(result.total / result.page_size)) : 1;
  const page = result?.page ?? 1;

  return (
    <main className={`${ui.main} ${styles.wide}`}>
      <h1 className={ui.title}>Đơn hàng</h1>

      <form className={`${ui.card} ${styles.filters}`} onSubmit={onSubmit} role="search" aria-label="Lọc đơn hàng">
        <label className={`${ui.label} ${styles.search}`}>
          Tìm kiếm
          <input
            className={ui.input}
            type="search"
            value={form.q}
            onChange={set('q')}
            placeholder="Mã đơn, mã vận đơn, tên khách, SĐT"
            maxLength={100}
          />
        </label>
        <label className={ui.label}>
          Sàn
          <select className={ui.select} value={form.market_place} onChange={set('market_place')}>
            <option value="">Tất cả</option>
            {MARKET_PLACES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </label>
        <label className={ui.label}>
          Trạng thái
          <select className={ui.select} value={form.lead_status} onChange={set('lead_status')}>
            <option value="">Tất cả</option>
            {Object.entries(LEAD_STATUS_LABELS).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label className={ui.label}>
          Từ ngày
          <input className={ui.input} type="date" value={form.from} onChange={set('from')} max={form.to || undefined} />
        </label>
        <label className={ui.label}>
          Đến ngày
          <input className={ui.input} type="date" value={form.to} onChange={set('to')} min={form.from || undefined} />
        </label>
        <div className={styles.filterActions}>
          <button className={ui.primaryButton} type="submit">
            Lọc
          </button>
          {hasFilters && (
            <button className={ui.secondaryButton} type="button" onClick={() => apply(EMPTY_FILTERS)}>
              Xoá lọc
            </button>
          )}
        </div>
      </form>

      {error && (
        <div className={ui.error} role="alert">
          {error}
        </div>
      )}

      {result === null ? (
        <Spinner label="Đang tải danh sách đơn hàng…" />
      ) : (
        <>
          <p className={styles.summary} role="status">
            {result.total === 0 ? 'Không có đơn hàng nào.' : `${result.total.toLocaleString('vi-VN')} đơn hàng`}
          </p>

          {result.orders.length > 0 && (
            <section className={`${ui.card} ${styles.tableCard}`}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th scope="col">Mã đơn</th>
                    <th scope="col">Sàn</th>
                    <th scope="col">Khách hàng</th>
                    <th scope="col">SĐT</th>
                    <th scope="col" className={styles.num}>
                      Tổng tiền
                    </th>
                    <th scope="col">Trạng thái</th>
                    <th scope="col">Ngày tạo</th>
                  </tr>
                </thead>
                <tbody>
                  {result.orders.map((o) => (
                    <tr key={o.id}>
                      <td>
                        {canDetail ? (
                          <Link className={styles.orderLink} to={`/orders/${o.id}`} state={{ from: search }}>
                            {orderLabel(o)}
                          </Link>
                        ) : (
                          <span className={styles.orderId}>{orderLabel(o)}</span>
                        )}
                        {o.tracking_code && <div className={styles.sub}>Vận đơn: {o.tracking_code}</div>}
                      </td>
                      <td>
                        <div>{marketPlaceLabel(o.market_place)}</div>
                        {o.shop_id && <div className={styles.sub}>Shop {o.shop_id}</div>}
                      </td>
                      <td>{o.customer_name || '—'}</td>
                      <td className={styles.nowrap}>{o.phone || '—'}</td>
                      <td className={`${styles.num} ${styles.nowrap}`}>{formatMoney(o.total_amount)}</td>
                      <td>
                        <span className={o.lead_status === 4 || o.lead_status === 3 ? ui.badgeDanger : ui.badgeMuted}>
                          {leadStatusLabel(o.lead_status)}
                        </span>
                      </td>
                      <td className={styles.when}>{formatDateTime(o.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>
          )}

          {pages > 1 && (
            <nav className={styles.pager} aria-label="Phân trang">
              <button className={ui.secondaryButton} type="button" disabled={page <= 1} onClick={() => apply(filtersFrom(params), page - 1)}>
                ‹ Trước
              </button>
              <span className={styles.pageInfo}>
                Trang {page} / {pages}
              </span>
              <button className={ui.secondaryButton} type="button" disabled={page >= pages} onClick={() => apply(filtersFrom(params), page + 1)}>
                Sau ›
              </button>
            </nav>
          )}
        </>
      )}
    </main>
  );
}
