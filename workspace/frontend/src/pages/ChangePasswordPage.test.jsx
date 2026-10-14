import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { backend } from '../test/server';
import { renderApp } from '../test/renderApp';

async function openPage() {
  backend.signedIn = true;
  renderApp('/account/password');
  await screen.findByRole('heading', { name: 'Đổi mật khẩu' });
  return userEvent.setup();
}

describe('ChangePasswordPage', () => {
  it('is reachable from the profile', async () => {
    backend.signedIn = true;
    renderApp('/profile');
    const user = userEvent.setup();
    await user.click(await screen.findByRole('link', { name: 'Đổi mật khẩu' }));
    expect(await screen.findByRole('heading', { name: 'Đổi mật khẩu' })).toBeInTheDocument();
  });

  it('changes the password and clears the form', async () => {
    const user = await openPage();
    await user.type(screen.getByLabelText('Mật khẩu hiện tại'), 'old-password-1');
    await user.type(screen.getByLabelText(/^Mật khẩu mới/), 'new-password-123');
    await user.type(screen.getByLabelText('Nhập lại mật khẩu mới'), 'new-password-123');
    await user.click(screen.getByRole('button', { name: 'Đổi mật khẩu' }));

    expect(await screen.findByRole('status')).toHaveTextContent('Đã đổi mật khẩu.');
    expect(backend.changePasswordBody).toEqual({ current_password: 'old-password-1', new_password: 'new-password-123' });
    expect(screen.getByLabelText(/^Mật khẩu mới/)).toHaveValue('');
  });

  it('blocks a short or mismatched new password before calling the backend', async () => {
    const user = await openPage();
    const submit = screen.getByRole('button', { name: 'Đổi mật khẩu' });
    await user.type(screen.getByLabelText('Mật khẩu hiện tại'), 'old-password-1');
    await user.type(screen.getByLabelText(/^Mật khẩu mới/), 'short');
    await user.type(screen.getByLabelText('Nhập lại mật khẩu mới'), 'short');
    expect(submit).toBeDisabled();

    await user.clear(screen.getByLabelText(/^Mật khẩu mới/));
    await user.type(screen.getByLabelText(/^Mật khẩu mới/), 'long-enough-1');
    await user.clear(screen.getByLabelText('Nhập lại mật khẩu mới'));
    await user.type(screen.getByLabelText('Nhập lại mật khẩu mới'), 'long-enough-2');
    expect(screen.getByText('Mật khẩu nhập lại không khớp.')).toBeInTheDocument();
    expect(submit).toBeDisabled();
    expect(backend.changePasswordBody).toBeNull();
  });

  it('shows "wrong current password" and clears only that field', async () => {
    backend.changePasswordError = 'WRONG_PASSWORD';
    const user = await openPage();
    await user.type(screen.getByLabelText('Mật khẩu hiện tại'), 'not-my-password');
    await user.type(screen.getByLabelText(/^Mật khẩu mới/), 'new-password-123');
    await user.type(screen.getByLabelText('Nhập lại mật khẩu mới'), 'new-password-123');
    await user.click(screen.getByRole('button', { name: 'Đổi mật khẩu' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('Mật khẩu hiện tại không đúng.');
    expect(screen.getByLabelText('Mật khẩu hiện tại')).toHaveValue('');
    expect(screen.getByLabelText(/^Mật khẩu mới/)).toHaveValue('new-password-123');
  });
});

describe('ProfilePage role display', () => {
  it('shows the role and its permissions', async () => {
    backend.signedIn = true;
    backend.role = 'operator';
    renderApp('/profile');
    await screen.findByRole('heading', { name: /Xin chào/ });
    expect(screen.getByText('Vận hành')).toBeInTheDocument();
    const perms = screen.getByRole('list', { name: 'Quyền của bạn' });
    expect(perms).toHaveTextContent('Xem đơn hàng');
    expect(perms).toHaveTextContent('Cập nhật đơn hàng');
    expect(perms).not.toHaveTextContent('Quản lý người dùng');
  });
});
