import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { backend } from '../test/server';
import { renderApp } from '../test/renderApp';
import { FakeBroadcastChannel } from '../test/fakeBroadcastChannel';

const CHANNEL = 'mq-admin-auth';

async function openProfile() {
  backend.signedIn = true;
  renderApp('/profile');
  await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' });
  return userEvent.setup();
}

describe('ProtectedRoute + ProfilePage', () => {
  it('sends anonymous visitors to /login with the page they wanted', async () => {
    renderApp('/profile');
    expect(await screen.findByRole('heading', { name: 'Đăng nhập' })).toBeInTheDocument();
    expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Fprofile');
  });

  it('shows the signed-in user', async () => {
    await openProfile();
    expect(within(screen.getByRole('main')).getByText('an.nguyen@mq.com')).toBeInTheDocument(); // the sidebar shows it too
    expect(screen.getByText('Email & mật khẩu')).toBeInTheDocument();
    expect(screen.getByText('AN')).toBeInTheDocument(); // initials, no avatar
  });

  it('redirects unknown paths to the profile', async () => {
    backend.signedIn = true;
    renderApp('/does-not-exist');
    expect(await screen.findByRole('heading', { name: /Xin chào/ })).toBeInTheDocument();
  });
});

describe('logout (this browser)', () => {
  it('asks for confirmation, then revokes the session and returns to /login', async () => {
    const user = await openProfile();
    await user.click(screen.getByRole('button', { name: 'Đăng xuất' }));

    const dialog = screen.getByRole('alertdialog', { name: 'Đăng xuất?' });
    expect(backend.logoutCalls).toBe(0); // nothing happens before confirming
    expect(within(dialog).getByRole('button', { name: 'Huỷ' })).toHaveFocus();

    await user.click(within(dialog).getByRole('button', { name: 'Đăng xuất' }));
    expect(await screen.findByRole('heading', { name: 'Đăng nhập' })).toBeInTheDocument();
    expect(backend.logoutCalls).toBe(1);
    expect(backend.signedIn).toBe(false);
  });

  it('cancel and Escape keep the user signed in', async () => {
    const user = await openProfile();
    await user.click(screen.getByRole('button', { name: 'Đăng xuất' }));
    await user.click(screen.getByRole('button', { name: 'Huỷ' }));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Đăng xuất' }));
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();

    expect(backend.logoutCalls).toBe(0);
    expect(screen.getByRole('heading', { name: /Xin chào/ })).toBeInTheDocument();
  });

  it('still lands on /login when the session had already expired', async () => {
    const user = await openProfile();
    backend.signedIn = false; // expired server-side while the page was open
    await user.click(screen.getByRole('button', { name: 'Đăng xuất' }));
    await user.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Đăng xuất' }));
    expect(await screen.findByRole('heading', { name: 'Đăng nhập' })).toBeInTheDocument();
  });

  it('tells the other tabs', async () => {
    const user = await openProfile();
    await user.click(screen.getByRole('button', { name: 'Đăng xuất' }));
    await user.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Đăng xuất' }));
    await screen.findByRole('heading', { name: 'Đăng nhập' });
    expect(FakeBroadcastChannel.sent).toContainEqual({ name: CHANNEL, data: 'logout' });
  });
});

describe('logout on all devices', () => {
  it('asks for confirmation, revokes every session and returns to /login', async () => {
    const user = await openProfile();
    await user.click(screen.getByRole('button', { name: 'Đăng xuất tất cả thiết bị' }));

    const dialog = screen.getByRole('alertdialog', { name: 'Đăng xuất khỏi tất cả thiết bị?' });
    expect(backend.logoutAllCalls).toBe(0);
    await user.click(within(dialog).getByRole('button', { name: 'Đăng xuất tất cả' }));

    expect(await screen.findByRole('heading', { name: 'Đăng nhập' })).toBeInTheDocument();
    expect(backend.logoutAllCalls).toBe(1);
    expect(backend.logoutCalls).toBe(0);
    expect(FakeBroadcastChannel.sent).toContainEqual({ name: CHANNEL, data: 'logout' });
  });

  it.each([
    ['INTERNAL_ERROR', 'Không thể đăng xuất khỏi các thiết bị khác lúc này.'],
    ['NETWORK', 'Không kết nối được máy chủ. Các thiết bị khác vẫn đang đăng nhập'],
  ])('on %s it stays signed in and says the other devices are still signed in', async (failure, text) => {
    backend.logoutAllError = failure;
    const user = await openProfile();
    await user.click(screen.getByRole('button', { name: 'Đăng xuất tất cả thiết bị' }));
    await user.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Đăng xuất tất cả' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(text);
    expect(screen.getByRole('heading', { name: /Xin chào/ })).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(FakeBroadcastChannel.sent).not.toContainEqual({ name: CHANNEL, data: 'logout' });
  });
});

describe('cross-tab sync', () => {
  it('goes to /login when another tab logs out', async () => {
    await openProfile();
    backend.signedIn = false;
    FakeBroadcastChannel.fromOtherTab(CHANNEL, 'logout');
    expect(await screen.findByRole('heading', { name: 'Đăng nhập' })).toBeInTheDocument();
    expect(screen.getByTestId('location')).toHaveTextContent('/login?next=%2Fprofile');
  });

  it('signs in when another tab logs in', async () => {
    renderApp('/login');
    await screen.findByRole('heading', { name: 'Đăng nhập' });
    backend.signedIn = true; // the other tab's login set the shared cookie
    FakeBroadcastChannel.fromOtherTab(CHANNEL, 'login');
    expect(await screen.findByRole('heading', { name: 'Xin chào, An Nguyen' })).toBeInTheDocument();
  });

  it('ignores unknown messages', async () => {
    await openProfile();
    FakeBroadcastChannel.fromOtherTab(CHANNEL, { type: 'logout', user: 'x' });
    FakeBroadcastChannel.fromOtherTab('some-other-channel', 'logout');
    await waitFor(() => expect(screen.getByRole('heading', { name: /Xin chào/ })).toBeInTheDocument());
  });
});
