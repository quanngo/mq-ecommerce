import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { backend } from '../test/server';
import { renderApp } from '../test/renderApp';

function signIn(role = 'viewer', withoutPermissions = []) {
  backend.signedIn = true;
  backend.role = role;
  backend.withoutPermissions = withoutPermissions;
}

async function openOrders(path = '/orders', ...signInArgs) {
  signIn(...signInArgs);
  renderApp(path);
  await screen.findByRole('heading', { name: 'Đơn hàng' });
  await screen.findByRole('table');
  return userEvent.setup();
}

const rowOf = (text) => screen.getByText(text).closest('tr');

describe('navigation and access', () => {
  it('shows the "Đơn hàng" menu with orders.read', async () => {
    signIn('viewer');
    renderApp('/profile');
    await screen.findByRole('heading', { name: /Xin chào/ });
    expect(screen.getByRole('link', { name: 'Đơn hàng' })).toHaveAttribute('href', '/orders');
  });

  it('hides the menu and blocks /orders without orders.read', async () => {
    signIn('viewer', ['orders.read']);
    renderApp('/orders');
    expect(await screen.findByRole('heading', { name: 'Không có quyền truy cập' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Đơn hàng' })).not.toBeInTheDocument();
    expect(backend.lastOrdersQuery).toBeNull();
  });

  it('anonymous visitors go to /login?next=/orders', async () => {
    renderApp('/orders');
    await screen.findByRole('heading', { name: 'Đăng nhập' });
    expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Forders');
  });
});

describe('OrdersPage', () => {
  it('lists the first page, newest first, with phone and total', async () => {
    await openOrders();
    expect(screen.getByRole('status')).toHaveTextContent('25 đơn hàng');
    const rows = within(screen.getByRole('table')).getAllByRole('row');
    expect(rows).toHaveLength(21); // header + 20
    const first = rowOf('TT-1025');
    expect(within(first).getByText('Nguyễn Văn An')).toBeInTheDocument();
    expect(within(first).getByText('0900000025')).toBeInTheDocument();
    expect(within(first).getByText('TikTok Shop')).toBeInTheDocument();
    expect(within(first).getByText(/125\.000/)).toBeInTheDocument();
    expect(within(rowOf('LZ-1024')).getByText('Đã huỷ')).toBeInTheDocument();
    expect(screen.getByText('Trang 1 / 2')).toBeInTheDocument();
  });

  it('applies filters through the URL and the API query', async () => {
    const user = await openOrders();
    await user.type(screen.getByRole('searchbox', { name: 'Tìm kiếm' }), 'Nguyễn');
    await user.selectOptions(screen.getByRole('combobox', { name: 'Sàn' }), 'Tiktok');
    await user.selectOptions(screen.getByRole('combobox', { name: 'Trạng thái' }), '2');
    await user.type(screen.getByLabelText('Từ ngày'), '2026-10-01');
    await user.click(screen.getByRole('button', { name: 'Lọc' }));

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('1 đơn hàng'));
    expect(backend.lastOrdersQuery).toEqual({ q: 'Nguyễn', market_place: 'Tiktok', lead_status: '2', from: '2026-10-01' });
    expect(screen.getByTestId('location')).toHaveTextContent(
      '/orders?q=Nguy%E1%BB%85n&market_place=Tiktok&lead_status=2&from=2026-10-01',
    );

    await user.click(screen.getByRole('button', { name: 'Xoá lọc' }));
    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('25 đơn hàng'));
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/orders$/);
    expect(screen.getByRole('searchbox', { name: 'Tìm kiếm' })).toHaveValue('');
  });

  it('reads filters from the URL on load', async () => {
    await openOrders('/orders?market_place=Lazada&q=LZ-1024');
    expect(screen.getByRole('combobox', { name: 'Sàn' })).toHaveValue('Lazada');
    expect(screen.getByRole('searchbox', { name: 'Tìm kiếm' })).toHaveValue('LZ-1024');
    expect(screen.getByRole('status')).toHaveTextContent('1 đơn hàng');
  });

  it('pages through results', async () => {
    const user = await openOrders();
    await user.click(screen.getByRole('button', { name: 'Sau ›' }));
    await waitFor(() => expect(screen.getByText('Trang 2 / 2')).toBeInTheDocument());
    expect(backend.lastOrdersQuery).toEqual({ page: '2' });
    expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(6); // header + 5
    expect(screen.getByRole('button', { name: 'Sau ›' })).toBeDisabled();
  });

  it('shows an empty state', async () => {
    signIn();
    renderApp('/orders?q=khong-co-don');
    expect(await screen.findByText('Không có đơn hàng nào.')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('explains an invalid filter', async () => {
    signIn();
    renderApp('/orders?from=2026-10-20&to=2026-10-01');
    expect(await screen.findByRole('alert')).toHaveTextContent('Bộ lọc không hợp lệ');
  });

  it('links to the detail page only with orders.detail', async () => {
    await openOrders('/orders', 'viewer', ['orders.detail']);
    expect(screen.queryByRole('link', { name: 'TT-1025' })).not.toBeInTheDocument();
    expect(screen.getByText('TT-1025')).toBeInTheDocument();
  });
});

describe('OrderDetailPage', () => {
  it('opens from the list and shows the order, customer and items', async () => {
    const user = await openOrders('/orders?market_place=Tiktok');
    await user.click(screen.getByRole('link', { name: 'TT-1025' }));

    expect(await screen.findByRole('heading', { name: 'Đơn hàng TT-1025' })).toBeInTheDocument();
    expect(screen.getByTestId('location')).toHaveTextContent('/orders/25');
    expect(screen.getByText('0900000025')).toBeInTheDocument();
    expect(screen.getByText('12 Lê Lợi, Bến Nghé, Quận 1, TP. Hồ Chí Minh')).toBeInTheDocument();
    expect(screen.getByText('Giao giờ hành chính')).toBeInTheDocument();
    expect(screen.getByText('Thành công')).toBeInTheDocument(); // so_status 45
    expect(screen.getByText('COD')).toBeInTheDocument();

    const items = within(screen.getByRole('region', { name: 'Sản phẩm (2)' }));
    expect(items.getByText('Serum dưỡng da')).toBeInTheDocument();
    expect(items.getByText('SR-01')).toBeInTheDocument();
    expect(items.getAllByText(/100\.000/).length).toBeGreaterThan(0); // 2 × 50.000

    // Back keeps the list filters.
    await user.click(screen.getByRole('link', { name: '‹ Quay lại danh sách' }));
    await screen.findByRole('table');
    expect(screen.getByTestId('location')).toHaveTextContent('/orders?market_place=Tiktok');
  });

  it('shows "not found" for an unknown order', async () => {
    signIn();
    renderApp('/orders/999');
    expect(await screen.findByRole('alert')).toHaveTextContent('Không tìm thấy đơn hàng.');
    expect(screen.getByRole('link', { name: '‹ Quay lại danh sách' })).toHaveAttribute('href', '/orders');
  });

  it('is blocked without orders.detail', async () => {
    signIn('viewer', ['orders.detail']);
    renderApp('/orders/25');
    expect(await screen.findByRole('heading', { name: 'Không có quyền truy cập' })).toBeInTheDocument();
  });

  it('shows a generic error when the backend fails', async () => {
    backend.ordersError = 'INTERNAL_ERROR';
    signIn();
    renderApp('/orders/25');
    expect(await screen.findByRole('alert')).toHaveTextContent('Đã có lỗi xảy ra');
  });
});
