import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { backend, storageWrites } from '../test/server';
import { renderApp } from '../test/renderApp';
import { FakeBroadcastChannel } from '../test/fakeBroadcastChannel';

const english = () => screen.getByRole('button', { name: 'English' });
const vietnamese = () => screen.getByRole('button', { name: 'Tiếng Việt' });

describe('language switcher', () => {
  it('switches the login page to English and remembers it in this browser only', async () => {
    renderApp('/login');
    const user = userEvent.setup();
    await screen.findByRole('heading', { name: 'Đăng nhập' });
    expect(vietnamese()).toHaveAttribute('aria-pressed', 'true');

    await user.click(english());

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Show password' })).toBeInTheDocument();
    expect(english()).toHaveAttribute('aria-pressed', 'true');
    expect(document.documentElement.lang).toBe('en');
    expect(storageWrites.language).toBe('en');
    expect(storageWrites.count).toBe(0); // nothing else is stored
    expect(backend.localeCalls).toEqual([]); // not signed in: nothing saved on an account
  });

  it('re-translates an error that is already on screen and sends the language to the backend', async () => {
    backend.loginError = 'INVALID_CREDENTIALS';
    renderApp('/login');
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText('Email'), 'an.nguyen@mq.com');
    await user.type(screen.getByLabelText('Mật khẩu'), 'wrong-password');
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Email hoặc mật khẩu không đúng.');
    expect(backend.acceptLanguage).toBe('vi');

    await user.click(english());
    expect(await screen.findByRole('alert')).toHaveTextContent('Incorrect email or password.');

    await user.type(screen.getByLabelText('Password'), 'again');
    await user.click(screen.getByRole('button', { name: 'Sign in' }));
    await waitFor(() => expect(backend.acceptLanguage).toBe('en'));
  });

  it('applies the language saved on the account after login', async () => {
    backend.locale = 'en';
    renderApp('/login?next=%2Fprofile');
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText('Email'), 'an.nguyen@mq.com');
    await user.type(screen.getByLabelText('Mật khẩu'), 'correct-horse-1');
    await user.click(screen.getByRole('button', { name: 'Đăng nhập' }));

    expect(await screen.findByRole('heading', { name: 'Hello, An Nguyen' })).toBeInTheDocument();
    expect(screen.getByText('Viewer')).toBeInTheDocument(); // role name from the backend
    expect(screen.getByText('View order details')).toBeInTheDocument(); // permission name from the backend
    expect(storageWrites.language).toBe('en');
    expect(storageWrites.count).toBe(0);
  });

  it('saves a new choice on the account, shows the backend names in that language and tells other tabs', async () => {
    backend.signedIn = true;
    renderApp('/profile');
    const user = userEvent.setup();
    expect(await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' })).toBeInTheDocument();
    expect(screen.getByText('Chỉ xem')).toBeInTheDocument();

    await user.click(english());

    expect(await screen.findByRole('heading', { name: 'Hello, An Nguyen' })).toBeInTheDocument();
    expect(backend.localeCalls).toEqual([{ locale: 'en' }]);
    expect(screen.getByText('Viewer')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Orders' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument();
    expect(FakeBroadcastChannel.sent).toContainEqual({ name: 'mq-admin-auth', data: 'locale' });

    await user.click(vietnamese());
    expect(await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' })).toBeInTheDocument();
    expect(backend.localeCalls).toEqual([{ locale: 'en' }, { locale: 'vi' }]);
  });

  it('follows a language change made in another tab', async () => {
    backend.signedIn = true;
    renderApp('/profile');
    expect(await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' })).toBeInTheDocument();

    backend.locale = 'en'; // the other tab saved it
    FakeBroadcastChannel.fromOtherTab('mq-admin-auth', 'locale');

    expect(await screen.findByRole('heading', { name: 'Hello, An Nguyen' })).toBeInTheDocument();
  });

  it('still switches this browser when saving to the account fails, and says so', async () => {
    backend.signedIn = true;
    backend.localeError = 'INTERNAL_ERROR';
    renderApp('/profile');
    const user = userEvent.setup();
    await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' });

    await user.click(english());

    expect(await screen.findByRole('heading', { name: 'Hello, An Nguyen' })).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't save the language to your account.");
    expect(backend.locale).toBe('vi');
  });

  it('reloads roles and users in the new language on the users page', async () => {
    backend.signedIn = true;
    backend.role = 'admin';
    renderApp('/users');
    const user = userEvent.setup();
    expect(await screen.findByRole('heading', { name: 'Người dùng' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Vai trò và quyền' })).toBeInTheDocument();
    expect(screen.getAllByText('Quản lý người dùng').length).toBeGreaterThan(0);

    await user.click(english());

    expect(await screen.findByRole('heading', { name: 'Roles and permissions' })).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByText('Manage users').length).toBeGreaterThan(0));
    expect(screen.getByRole('combobox', { name: 'Role of op@mq.com' })).toHaveDisplayValue('Operator');
  });
});
