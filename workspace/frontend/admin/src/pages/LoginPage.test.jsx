import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { backend, storageWrites } from '../test/server';
import { renderApp } from '../test/renderApp';
import { FakeBroadcastChannel } from '../test/fakeBroadcastChannel';

async function fillAndSubmit(email, password) {
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText('Email'), email);
  await user.type(screen.getByLabelText('Mật khẩu'), password);
  await user.click(screen.getByRole('button', { name: 'Đăng nhập' }));
}

describe('LoginPage', () => {
  it('logs in with email and password and lands on the requested page', async () => {
    renderApp('/login?next=%2Fprofile');
    await fillAndSubmit('An.Nguyen@mq.com ', 'correct-horse-1');

    expect(await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' })).toBeInTheDocument();
    expect(screen.getByTestId('location')).toHaveTextContent('/profile');
    expect(backend.lastLoginBody).toEqual({ email: 'An.Nguyen@mq.com', password: 'correct-horse-1', remember_me: false });
    expect(storageWrites.count).toBe(0);
  });

  it('shows the invalid-credentials message and clears the password', async () => {
    backend.loginError = 'INVALID_CREDENTIALS';
    renderApp('/login');
    await fillAndSubmit('an.nguyen@mq.com', 'wrong-password');

    expect(await screen.findByRole('alert')).toHaveTextContent('Email hoặc mật khẩu không đúng.');
    expect(screen.getByLabelText('Mật khẩu')).toHaveValue('');
    expect(screen.getByTestId('location')).toHaveTextContent('/login');
  });

  it('shows the lockout message', async () => {
    backend.loginError = 'ACCOUNT_LOCKED';
    renderApp('/login');
    await fillAndSubmit('an.nguyen@mq.com', 'wrong-password');
    expect(await screen.findByRole('alert')).toHaveTextContent('Tài khoản tạm khoá');
  });

  it('disables submit until both fields are filled', async () => {
    renderApp('/login');
    const submit = await screen.findByRole('button', { name: 'Đăng nhập' });
    expect(submit).toBeDisabled();
    await userEvent.setup().type(screen.getByLabelText('Email'), 'a@mq.com');
    expect(submit).toBeDisabled();
  });

  it.each([
    ['CONSENT_DENIED', 'Bạn đã huỷ đăng nhập bằng Google.'],
    ['DOMAIN_NOT_ALLOWED', 'Chỉ tài khoản Google @mq.com được phép đăng nhập.'],
    ['USER_NOT_ALLOWED', 'Tài khoản chưa được cấp quyền truy cập.'],
    ['INVALID_STATE', 'Phiên đăng nhập đã hết hạn.'],
    ['SOMETHING_NEW', 'Đã có lỗi xảy ra.'],
  ])('shows the message for ?error=%s from the Google callback and strips it from the URL', async (code, text) => {
    renderApp(`/login?error=${code}&next=%2Fprofile`);
    expect(await screen.findByRole('alert')).toHaveTextContent(text);
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Fprofile'));
  });

  it('links the Google button to the backend login endpoint with a safe next path', async () => {
    renderApp('/login?next=%2F%2Fevil.example');
    const link = await screen.findByRole('link', { name: /Đăng nhập với Google/ });
    expect(link).toHaveAttribute('href', '/api/v1/auth/google/login?next=%2Fprofile');
  });

  it('"remember me" is off by default and applies to both password and Google login', async () => {
    renderApp('/login');
    const user = userEvent.setup();
    const remember = await screen.findByRole('checkbox', { name: 'Ghi nhớ đăng nhập trong 7 ngày' });
    const google = await screen.findByRole('link', { name: /Đăng nhập với Google/ });
    expect(remember).not.toBeChecked();
    expect(google.getAttribute('href')).not.toContain('remember');

    await user.click(remember);
    expect(google).toHaveAttribute('href', '/api/v1/auth/google/login?next=%2Fprofile&remember=1');

    await fillAndSubmit('an.nguyen@mq.com', 'correct-horse-1');
    await screen.findByRole('heading', { name: /Xin chào/ });
    expect(backend.lastLoginBody).toMatchObject({ remember_me: true });
  });

  it('tells other tabs after a successful login', async () => {
    renderApp('/login');
    await fillAndSubmit('an.nguyen@mq.com', 'correct-horse-1');
    await screen.findByRole('heading', { name: /Xin chào/ });
    expect(FakeBroadcastChannel.sent).toContainEqual({ name: 'mq-admin-auth', data: 'login' });
  });

  it('hides the Google button when the backend has Google disabled', async () => {
    backend.google = false;
    renderApp('/login');
    await screen.findByLabelText('Email');
    await waitFor(() => expect(screen.queryByRole('link', { name: /Google/ })).not.toBeInTheDocument());
  });

  it('redirects an already signed-in user away from /login', async () => {
    backend.signedIn = true;
    renderApp('/login');
    expect(await screen.findByRole('heading', { name: /Xin chào/ })).toBeInTheDocument();
  });
});
