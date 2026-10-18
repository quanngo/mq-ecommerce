import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation, useParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { ordersApi } from '../api/orders';
import { orderErrorMessage } from '../auth/errorMessages';
import {
  formatMoney,
  leadStatusLabel,
  marketPlaceLabel,
  orderLabel,
  paymentMethodLabel,
  saleOrderStatusLabel,
} from './orderFormat';
import { formatDateTime } from '../i18n/format';
import { Spinner } from './Spinner';
import styles from './OrderDetailPage.module.css';
import ui from './ui.module.css';

/** One order with its items (orders.detail). */
export function OrderDetailPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const location = useLocation();
  // Back to the list with the filters the user came from.
  const backTo = `/orders${location.state?.from ? `?${location.state.from}` : ''}`;

  const [order, setOrder] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    setOrder(null);
    setError(null);
    ordersApi
      .get(id)
      .then((o) => !cancelled && setOrder(o))
      .catch((err) => !cancelled && setError(err instanceof ApiError ? err : 'INTERNAL_ERROR'));
    return () => {
      cancelled = true;
    };
  }, [id]);

  const back = (
    <Link to={backTo} className={styles.back}>
      {t('orderDetail.back')}
    </Link>
  );

  if (error) {
    return (
      <main className={ui.main}>
        {back}
        <div className={ui.error} role="alert">
          {orderErrorMessage(error)}
        </div>
      </main>
    );
  }
  if (!order) return <Spinner label={t('orderDetail.loading')} />;

  const subtotal = order.items.reduce((sum, it) => sum + it.price * it.quantity, 0);
  const address = [order.address, order.neighborhood, order.sub_district, order.district, order.province].filter(Boolean).join(', ');

  return (
    <main className={ui.main}>
      {back}
      <div className={styles.header}>
        <h1 className={ui.title}>{t('orderDetail.title', { label: orderLabel(order) })}</h1>
        <span className={order.lead_status === 3 || order.lead_status === 4 ? ui.badgeDanger : ui.badge}>
          {leadStatusLabel(order.lead_status)}
        </span>
      </div>

      <div className={styles.grid}>
        <section className={ui.card} aria-labelledby="order-info">
          <h2 id="order-info" className={ui.sectionTitle}>
            {t('orderDetail.info')}
          </h2>
          <dl className={styles.fields}>
            <Field label={t('orderDetail.marketPlace')}>{marketPlaceLabel(order.market_place)}</Field>
            <Field label={t('orderDetail.shop')}>{order.shop_id}</Field>
            <Field label={t('orderDetail.orderId')}>{order.order_id}</Field>
            <Field label={t('orderDetail.tracking')}>{order.tracking_code}</Field>
            <Field label={t('orderDetail.doStatus')}>{order.do_status}</Field>
            <Field label={t('orderDetail.soStatus')}>{saleOrderStatusLabel(order.so_status)}</Field>
            <Field label={t('orderDetail.payment')}>{paymentMethodLabel(order.payment_method)}</Field>
            <Field label={t('orderDetail.warehouse')}>{order.warehouse}</Field>
            <Field label={t('orderDetail.created')}>{formatDateTime(order.created_at)}</Field>
            <Field label={t('orderDetail.updated')}>{formatDateTime(order.updated_at)}</Field>
          </dl>
        </section>

        <section className={ui.card} aria-labelledby="order-customer">
          <h2 id="order-customer" className={ui.sectionTitle}>
            {t('orderDetail.customer')}
          </h2>
          <dl className={styles.fields}>
            <Field label={t('orderDetail.name')}>{order.customer_name}</Field>
            <Field label={t('orderDetail.phone')}>{order.phone}</Field>
            <Field label={t('orderDetail.address')}>{address}</Field>
            <Field label={t('orderDetail.comment')}>{order.comment}</Field>
          </dl>
        </section>
      </div>

      <section className={`${ui.card} ${styles.itemsCard}`} aria-labelledby="order-items">
        <h2 id="order-items" className={`${ui.sectionTitle} ${styles.itemsTitle}`}>
          {t('orderDetail.items', { count: order.items.length })}
        </h2>
        {order.items.length === 0 ? (
          <p className={`${ui.muted} ${styles.itemsTitle}`}>{t('orderDetail.noItems')}</p>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">{t('orderDetail.colProduct')}</th>
                  <th scope="col">{t('orderDetail.colSku')}</th>
                  <th scope="col" className={styles.num}>
                    {t('orderDetail.colQty')}
                  </th>
                  <th scope="col" className={styles.num}>
                    {t('orderDetail.colPrice')}
                  </th>
                  <th scope="col" className={styles.num}>
                    {t('orderDetail.colAmount')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {order.items.map((it) => (
                  <tr key={it.id}>
                    <td>{it.name || '—'}</td>
                    <td className={styles.sku}>{it.sku || '—'}</td>
                    <td className={styles.num}>{it.quantity}</td>
                    <td className={styles.num}>{formatMoney(it.price)}</td>
                    <td className={styles.num}>{formatMoney(it.price * it.quantity)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <dl className={styles.totals}>
          <div>
            <dt>{t('orderDetail.subtotal')}</dt>
            <dd>{formatMoney(subtotal)}</dd>
          </div>
          <div>
            <dt>{t('orderDetail.discount')}</dt>
            <dd>{order.discount ? `−${formatMoney(order.discount)}` : formatMoney(0)}</dd>
          </div>
          <div>
            <dt>{t('orderDetail.shippingFee')}</dt>
            <dd>{formatMoney(order.shipping_fee)}</dd>
          </div>
          <div className={styles.grandTotal}>
            <dt>{t('orderDetail.total')}</dt>
            <dd>{formatMoney(order.total_amount)}</dd>
          </div>
        </dl>
      </section>
    </main>
  );
}

function Field({ label, children }) {
  return (
    <div className={styles.field}>
      <dt>{label}</dt>
      <dd>{children || '—'}</dd>
    </div>
  );
}
