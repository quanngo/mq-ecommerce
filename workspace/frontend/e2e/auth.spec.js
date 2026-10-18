import { expect, test } from '@playwright/test';

// Needs a password user in the backend DB:
//   echo "$E2E_PASSWORD" | go run main.go create-user --email "$E2E_EMAIL" --name "E2E User" --password-stdin
const EMAIL = process.env.E2E_EMAIL;
const PASSWORD = process.env.E2E_PASSWORD;

test.skip(!EMAIL || !PASSWORD, 'set E2E_EMAIL and E2E_PASSWORD (see README)');

async function login(page, password = PASSWORD) {
  await page.goto('/login');
  await page.getByLabel('Email').fill(EMAIL);
  await page.getByLabel('Mật khẩu', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Đăng nhập' }).click();
}

test('password login -> profile -> logout', async ({ page, context }) => {
  await login(page);
  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.getByRole('heading', { name: /Xin chào/ })).toBeVisible();
  await expect(page.getByText(EMAIL)).toBeVisible();

  const cookie = (await context.cookies()).find((c) => c.name === 'ecom_session');
  expect(cookie?.httpOnly).toBe(true);
  expect(cookie?.sameSite).toBe('Lax');
  // Only the UI language code may be stored in the browser.
  expect(await page.evaluate(() => [Object.keys(localStorage), sessionStorage.length])).toEqual([['mq-admin-lang'], 0]);

  // Top-bar button (exact: the profile page also has "Đăng xuất tất cả thiết bị"), then confirm.
  await page.getByRole('button', { name: 'Đăng xuất', exact: true }).click();
  await page.getByRole('alertdialog', { name: 'Đăng xuất?' }).getByRole('button', { name: 'Đăng xuất', exact: true }).click();
  // ProtectedRoute redirects as soon as the session is gone, so `next` may still point at /profile.
  await expect(page).toHaveURL(/\/login(\?next=%2Fprofile)?$/);
  await page.goto('/profile');
  await expect(page).toHaveURL(/\/login\?next=%2Fprofile$/);
});

test('wrong password shows an error and sets no cookie', async ({ page, context }) => {
  await login(page, 'definitely-wrong-password');
  await expect(page.getByRole('alert')).toHaveText('Email hoặc mật khẩu không đúng.');
  expect((await context.cookies()).some((c) => c.name === 'ecom_session')).toBe(false);
});

test('anonymous visit to a protected page redirects to login', async ({ page }) => {
  await page.goto('/profile');
  await expect(page).toHaveURL(/\/login\?next=%2Fprofile$/);
});

test('Google callback errors are shown on the login page', async ({ page }) => {
  await page.goto('/login?error=CONSENT_DENIED');
  await expect(page.getByRole('alert')).toHaveText('Bạn đã huỷ đăng nhập bằng Google.');
  await expect(page).toHaveURL(/\/login$/);
});

test('a session revoked server-side sends the user back to login', async ({ page, context }) => {
  await login(page);
  await expect(page).toHaveURL(/\/profile$/);
  // Simulate expiry/revocation: the browser still has a cookie the backend no longer accepts.
  await context.addCookies([{ name: 'ecom_session', value: 'revoked-token', url: page.url() }]);
  await page.reload();
  await expect(page).toHaveURL(/\/login\?next=%2Fprofile$/);
});
