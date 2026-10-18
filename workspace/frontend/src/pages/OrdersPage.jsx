import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useSearchParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { ordersApi } from '../api/orders';
import { useAuth } from '../auth/AuthContext';
import { orderErrorMessage } from '../auth/errorMessages';
import { formatDateTime, formatNumber } from '../i18n/format';
import { LEAD_STATUSES, MARKET_PLACES, formatMoney, leadStatusLabel, marketPlaceLabel, orderLabel } from './orderFormat';
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
  const { t } = useTranslation();
  const { can } = useAuth();
  const canDetail = can('orders.detail');
  const [params, setParams] = useSearchParams();
  const search = params.toString();

  const [form, setForm] = useState(() => filtersFrom(params));
  const [result, setResult] = useState(null); // { orders, total, page, page_size } | null while loading
  const [error, setError] = useState(null);

  // Keep the form in step with the URL (Back/Forward, "Clear filters").
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
        setError(err instanceof ApiError ? err : 'INTERNAL_ERROR');
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
      <h1 className={ui.title}>{t('orders.title')}</h1>

      <form className={`${ui.card} ${styles.filters}`} onSubmit={onSubmit} role="search" aria-label={t('orders.filters')}>
        <label className={`${ui.label} ${styles.search}`}>
          {t('orders.search')}
          <input
            className={ui.input}
            type="search"
            value={form.q}
            onChange={set('q')}
            placeholder={t('orders.searchPlaceholder')}
            maxLength={100}
          />
        </label>
        <label className={ui.label}>
          {t('orders.marketPlace')}
          <select className={ui.select} value={form.market_place} onChange={set('market_place')}>
            <option value="">{t('common.all')}</option>
            {MARKET_PLACES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </label>
        <label className={ui.label}>
          {t('orders.status')}
          <select className={ui.select} value={form.lead_status} onChange={set('lead_status')}>
            <option value="">{t('common.all')}</option>
            {LEAD_STATUSES.map((value) => (
              <option key={value} value={value}>
                {leadStatusLabel(value)}
              </option>
            ))}
          </select>
        </label>
        <label className={ui.label}>
          {t('orders.from')}
          <input className={ui.input} type="date" value={form.from} onChange={set('from')} max={form.to || undefined} />
        </label>
        <label className={ui.label}>
          {t('orders.to')}
          <input className={ui.input} type="date" value={form.to} onChange={set('to')} min={form.from || undefined} />
        </label>
        <div className={styles.filterActions}>
          <button className={ui.primaryButton} type="submit">
            {t('orders.filter')}
          </button>
          {hasFilters && (
            <button className={ui.secondaryButton} type="button" onClick={() => apply(EMPTY_FILTERS)}>
              {t('orders.clear')}
            </button>
          )}
        </div>
      </form>

      {error && (
        <div className={ui.error} role="alert">
          {orderErrorMessage(error)}
        </div>
      )}

      {result === null ? (
        <Spinner label={t('orders.loading')} />
      ) : (
        <>
          <p className={styles.summary} role="status">
            {result.total === 0 ? t('orders.empty') : t('orders.count', { count: result.total, total: formatNumber(result.total) })}
          </p>

          {result.orders.length > 0 && (
            <section className={`${ui.card} ${styles.tableCard}`}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th scope="col">{t('orders.colOrder')}</th>
                    <th scope="col">{t('orders.colMarketPlace')}</th>
                    <th scope="col">{t('orders.colCustomer')}</th>
                    <th scope="col">{t('orders.colPhone')}</th>
                    <th scope="col" className={styles.num}>
                      {t('orders.colTotal')}
                    </th>
                    <th scope="col">{t('orders.colStatus')}</th>
                    <th scope="col">{t('orders.colCreated')}</th>
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
                        {o.tracking_code && <div className={styles.sub}>{t('orders.tracking', { code: o.tracking_code })}</div>}
                      </td>
                      <td>
                        <div>{marketPlaceLabel(o.market_place)}</div>
                        {o.shop_id && <div className={styles.sub}>{t('orders.shop', { id: o.shop_id })}</div>}
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
            <nav className={styles.pager} aria-label={t('orders.pagination')}>
              <button className={ui.secondaryButton} type="button" disabled={page <= 1} onClick={() => apply(filtersFrom(params), page - 1)}>
                {t('orders.prev')}
              </button>
              <span className={styles.pageInfo}>
                {t('orders.page', { page, pages })}
              </span>
              <button className={ui.secondaryButton} type="button" disabled={page >= pages} onClick={() => apply(filtersFrom(params), page + 1)}>
                {t('orders.next')}
              </button>
            </nav>
          )}
        </>
      )}
    </main>
  );
}
