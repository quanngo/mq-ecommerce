import { useEffect, useState } from 'react';
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
import { formatDateTime } from './ProfilePage';
import { Spinner } from './Spinner';
import styles from './OrderDetailPage.module.css';
import ui from './ui.module.css';

/** One order with its items (orders.detail). */
export function OrderDetailPage() {
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
      .catch((err) => !cancelled && setError(orderErrorMessage(err instanceof ApiError ? err.errorCode : null)));
    return () => {
      cancelled = true;
    };
  }, [id]);

  const back = (
    <Link to={backTo} className={styles.back}>
      ‹ Quay lại danh sách
    </Link>
  );

  if (error) {
    return (
      <main className={ui.main}>
        {back}
        <div className={ui.error} role="alert">
          {error}
        </div>
      </main>
    );
  }
  if (!order) return <Spinner label="Đang tải đơn hàng…" />;

  const subtotal = order.items.reduce((sum, it) => sum + it.price * it.quantity, 0);
  const address = [order.address, order.neighborhood, order.sub_district, order.district, order.province].filter(Boolean).join(', ');

  return (
    <main className={ui.main}>
      {back}
      <div className={styles.header}>
        <h1 className={ui.title}>Đơn hàng {orderLabel(order)}</h1>
        <span className={order.lead_status === 3 || order.lead_status === 4 ? ui.badgeDanger : ui.badge}>
          {leadStatusLabel(order.lead_status)}
        </span>
      </div>

      <div className={styles.grid}>
        <section className={ui.card} aria-labelledby="order-info">
          <h2 id="order-info" className={ui.sectionTitle}>
            Thông tin đơn
          </h2>
          <dl className={styles.fields}>
            <Field label="Sàn">{marketPlaceLabel(order.market_place)}</Field>
            <Field label="Shop">{order.shop_id}</Field>
            <Field label="Mã đơn sàn">{order.order_id}</Field>
            <Field label="Mã vận đơn">{order.tracking_code}</Field>
            <Field label="Trạng thái giao hàng">{order.do_status}</Field>
            <Field label="Trạng thái đơn bán">{saleOrderStatusLabel(order.so_status)}</Field>
            <Field label="Thanh toán">{paymentMethodLabel(order.payment_method)}</Field>
            <Field label="Kho">{order.warehouse}</Field>
            <Field label="Ngày tạo">{formatDateTime(order.created_at)}</Field>
            <Field label="Cập nhật">{formatDateTime(order.updated_at)}</Field>
          </dl>
        </section>

        <section className={ui.card} aria-labelledby="order-customer">
          <h2 id="order-customer" className={ui.sectionTitle}>
            Khách hàng
          </h2>
          <dl className={styles.fields}>
            <Field label="Tên">{order.customer_name}</Field>
            <Field label="SĐT">{order.phone}</Field>
            <Field label="Địa chỉ">{address}</Field>
            <Field label="Ghi chú">{order.comment}</Field>
          </dl>
        </section>
      </div>

      <section className={`${ui.card} ${styles.itemsCard}`} aria-labelledby="order-items">
        <h2 id="order-items" className={`${ui.sectionTitle} ${styles.itemsTitle}`}>
          Sản phẩm ({order.items.length})
        </h2>
        {order.items.length === 0 ? (
          <p className={`${ui.muted} ${styles.itemsTitle}`}>Đơn hàng không có sản phẩm.</p>
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Sản phẩm</th>
                  <th scope="col">SKU</th>
                  <th scope="col" className={styles.num}>
                    SL
                  </th>
                  <th scope="col" className={styles.num}>
                    Đơn giá
                  </th>
                  <th scope="col" className={styles.num}>
                    Thành tiền
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
            <dt>Tiền hàng</dt>
            <dd>{formatMoney(subtotal)}</dd>
          </div>
          <div>
            <dt>Giảm giá</dt>
            <dd>{order.discount ? `−${formatMoney(order.discount)}` : formatMoney(0)}</dd>
          </div>
          <div>
            <dt>Phí vận chuyển</dt>
            <dd>{formatMoney(order.shipping_fee)}</dd>
          </div>
          <div className={styles.grandTotal}>
            <dt>Tổng tiền</dt>
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
