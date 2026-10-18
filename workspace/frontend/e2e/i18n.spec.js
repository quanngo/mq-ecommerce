import { expect, test } from '@playwright/test';

// EC-7: Vietnamese / English. Same E2E user as auth.spec.js; the spec puts the
// account back to Vietnamese at the end so the other specs keep their wording.
const EMAIL = process.env.E2E_EMAIL;
const PASSWORD = process.env.E2E_PASSWORD;

test.skip(!EMAIL || !PASSWORD, 'set E2E_EMAIL and E2E_PASSWORD (see README)');

const pick = (page, label) => page.getByRole('group', { name: /Ngôn ngữ|Language/ }).getByRole('button', { name: label });

test('switch to English, keep it across reload and login, switch back', async ({ page, context }) => {
  // Signed out: the login page switches and remembers the choice in this browser.
  await page.goto('/login');
  await pick(page, 'English').click();
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible();
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible();

  // Signed in: the account's saved language applies (Vietnamese unless changed before).
  await page.getByLabel('Email').fill(EMAIL);
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByRole('heading', { name: /Xin chào|Hello/ })).toBeVisible();

  // Save English on the account; the backend answers in it too.
  const saved = page.waitForResponse((r) => r.url().endsWith('/api/v1/auth/me/locale'));
  await pick(page, 'English').click();
  const body = await (await saved).json();
  expect(body.code).toBe(200);
  expect(body.data[0].locale).toBe('en');
  await expect(page.getByRole('heading', { name: /^Hello/ })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Profile' })).toBeVisible();

  // A brand-new browser context (no localStorage) still gets English after login.
  const other = await context.browser().newContext({ locale: 'vi-VN' });
  const second = await other.newPage();
  await second.goto('/login');
  await second.getByLabel('Email').fill(EMAIL);
  await second.getByLabel('Mật khẩu', { exact: true }).fill(PASSWORD);
  await second.getByRole('button', { name: 'Đăng nhập' }).click();
  await expect(second.getByRole('heading', { name: /^Hello/ })).toBeVisible();
  await other.close();

  // An error from the backend comes with text in the account's language.
  const err = await page.evaluate(async () => {
    const res = await fetch('/api/v1/auth/me/locale', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'Accept-Language': 'vi' },
      body: JSON.stringify({ locale: 'fr' }),
    });
    return res.json();
  });
  expect(err.message).toBe('INVALID_REQUEST'); // the code never changes
  expect(err.error_text).toBe('Invalid data. Please check and try again.'); // the saved language beats Accept-Language

  // Back to Vietnamese for the other specs.
  await pick(page, 'Tiếng Việt').click();
  await expect(page.getByRole('heading', { name: /^Xin chào/ })).toBeVisible();
});
