import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { backend } from '../test/server';
import { renderApp } from '../test/renderApp';

async function openUsers(role = 'admin') {
  backend.signedIn = true;
  backend.role = role;
  renderApp('/users');
  await screen.findByRole('heading', { name: 'Người dùng' });
  return userEvent.setup();
}

const row = (email) => within(screen.getByRole('table')).getByText(email).closest('tr');

describe('navigation and access', () => {
  it('shows the "Người dùng" menu only with users.read', async () => {
    backend.signedIn = true;
    backend.role = 'viewer';
    renderApp('/profile');
    await screen.findByRole('heading', { name: /Xin chào/ });
    expect(screen.queryByRole('link', { name: 'Người dùng' })).not.toBeInTheDocument();
  });

  it('a viewer opening /users sees "no access", not the list', async () => {
    backend.signedIn = true;
    backend.role = 'viewer';
    renderApp('/users');
    expect(await screen.findByRole('heading', { name: 'Không có quyền truy cập' })).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('anonymous visitors go to /login?next=/users', async () => {
    renderApp('/users');
    await screen.findByRole('heading', { name: 'Đăng nhập' });
    expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Fusers');
  });
});

describe('UsersPage (admin)', () => {
  it('lists users with role, login methods and status, and marks yourself', async () => {
    await openUsers();
    expect(screen.getByRole('link', { name: 'Người dùng' })).toBeInTheDocument();
    const mine = row('an.nguyen@mq.com');
    expect(within(mine).getByText('Bạn')).toBeInTheDocument();
    // You can't change your own role/status or reset your own password here.
    expect(within(mine).queryByRole('combobox')).not.toBeInTheDocument();
    expect(within(mine).queryByRole('button', { name: 'Khoá' })).not.toBeInTheDocument();

    const g = row('google.only@mq.com');
    expect(within(g).getByText('Google')).toBeInTheDocument();
    expect(within(g).queryByText('Mật khẩu')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Vai trò và quyền' })).toBeInTheDocument();
  });

  it('changes a role', async () => {
    const user = await openUsers();
    await user.selectOptions(screen.getByRole('combobox', { name: 'Vai trò của op@mq.com' }), 'viewer');
    expect(await screen.findByRole('status')).toHaveTextContent('Đã đổi vai trò của op@mq.com.');
    expect(backend.lastUserRequest).toEqual({ method: 'PATCH', path: '/users/8', body: { role: 'viewer' } });
  });

  it('disables and re-enables a user', async () => {
    const user = await openUsers();
    await user.click(within(row('op@mq.com')).getByRole('button', { name: 'Khoá' }));
    expect(await screen.findByRole('status')).toHaveTextContent('Đã khoá op@mq.com.');
    expect(within(row('op@mq.com')).getByText('Đã khoá')).toBeInTheDocument();
    expect(backend.lastUserRequest.body).toEqual({ status: 'disabled' });

    await user.click(within(row('op@mq.com')).getByRole('button', { name: 'Mở khoá' }));
    await waitFor(() => expect(backend.lastUserRequest.body).toEqual({ status: 'active' }));
  });

  it('shows the backend reason when a change is refused', async () => {
    backend.userError = 'LAST_ADMIN';
    const user = await openUsers();
    await user.click(within(row('op@mq.com')).getByRole('button', { name: 'Khoá' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('quản trị viên cuối cùng');
  });

  it('creates a password user', async () => {
    const user = await openUsers();
    await user.click(screen.getByRole('button', { name: '+ Tạo người dùng' }));
    const dialog = screen.getByRole('dialog', { name: 'Tạo người dùng' });
    await user.type(within(dialog).getByLabelText('Email'), 'new@mq.com');
    await user.type(within(dialog).getByLabelText('Tên hiển thị'), 'New Person');
    await user.selectOptions(within(dialog).getByLabelText('Vai trò'), 'operator');

    const submit = within(dialog).getByRole('button', { name: 'Tạo' });
    await user.type(within(dialog).getByLabelText(/^Mật khẩu/), 'short');
    expect(submit).toBeDisabled(); // under 10 characters
    await user.type(within(dialog).getByLabelText(/^Mật khẩu/), '-but-now-long');
    await user.click(submit);

    expect(await screen.findByRole('status')).toHaveTextContent('Đã tạo tài khoản new@mq.com.');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(backend.lastUserRequest.body).toEqual({ email: 'new@mq.com', display_name: 'New Person', role: 'operator', password: 'short-but-now-long' });
    expect(row('new@mq.com')).toBeInTheDocument();
  });

  it('creates a Google-only user without sending a password', async () => {
    const user = await openUsers();
    await user.click(screen.getByRole('button', { name: '+ Tạo người dùng' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Email'), 'g2@mq.com');
    await user.click(within(dialog).getByRole('checkbox', { name: /Đặt mật khẩu/ }));
    await user.click(within(dialog).getByRole('button', { name: 'Tạo' }));
    await screen.findByRole('status');
    expect(backend.lastUserRequest.body).not.toHaveProperty('password');
  });

  it('keeps the dialog open and explains a duplicate email', async () => {
    backend.userError = 'EMAIL_TAKEN';
    const user = await openUsers();
    await user.click(screen.getByRole('button', { name: '+ Tạo người dùng' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Email'), 'op@mq.com');
    await user.click(within(dialog).getByRole('checkbox', { name: /Đặt mật khẩu/ }));
    await user.click(within(dialog).getByRole('button', { name: 'Tạo' }));
    expect(await within(dialog).findByRole('alert')).toHaveTextContent('Email này đã có tài khoản.');
  });

  it('resets another user\'s password', async () => {
    const user = await openUsers();
    await user.click(within(row('op@mq.com')).getByRole('button', { name: 'Đặt lại mật khẩu' }));
    const dialog = screen.getByRole('dialog', { name: 'Đặt lại mật khẩu cho op@mq.com' });
    await user.type(within(dialog).getByLabelText(/Mật khẩu mới/), 'brand-new-password');
    await user.click(within(dialog).getByRole('button', { name: 'Đặt lại' }));
    expect(await screen.findByRole('status')).toHaveTextContent('Đã đặt lại mật khẩu cho op@mq.com');
    expect(backend.lastUserRequest).toEqual({ method: 'POST', path: '/users/8/reset-password', body: { password: 'brand-new-password' } });
  });
});
