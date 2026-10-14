import { expect, test } from '@playwright/test';

// Orders menu (EC-4) against the real backend. Needs:
//   - migrations 004 + 005 applied (orders.read / orders.detail);
//   - a password user whose role has both permissions (every seeded role does):
//       echo "$E2E_PASSWORD" | go run main.go create-user --email "$E2E_EMAIL" --name "E2E User" --role viewer --password-stdin
//   - at least one row in public.ecom_order (let the TikTok/Lazada workers run, or insert one by hand), and
//     E2E_ORDER_QUERY set to something that finds it (its order_id, tracking code or customer phone).
const EMAIL = process.env.E2E_EMAIL;
const PASSWORD = process.env.E2E_PASSWORD;
const ORDER_QUERY = process.env.E2E_ORDER_QUERY;

test.skip(!EMAIL || !PASSWORD || !ORDER_QUERY, 'set E2E_EMAIL, E2E_PASSWORD and E2E_ORDER_QUERY (see the header of this file)');

test.beforeEach(async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('Email').fill(EMAIL);
  await page.getByLabel('Mật khẩu', { exact: true }).fill(PASSWORD);
  await page.getByRole('button', { name: 'Đăng nhập' }).click();
  await expect(page).toHaveURL(/\/profile$/);
});

test('menu -> filtered list -> detail -> back keeps the filter', async ({ page }) => {
  await page.getByRole('link', { name: 'Đơn hàng' }).click();
  await expect(page).toHaveURL(/\/orders$/);
  await expect(page.getByRole('heading', { name: 'Đơn hàng' })).toBeVisible();

  await page.getByRole('searchbox', { name: 'Tìm kiếm' }).fill(ORDER_QUERY);
  await page.getByRole('button', { name: 'Lọc' }).click();
  await expect(page).toHaveURL(new RegExp(`/orders\\?q=${encodeURIComponent(ORDER_QUERY)}$`));
  const table = page.getByRole('table');
  await expect(table).toBeVisible();

  await table.getByRole('row').nth(1).getByRole('link').first().click();
  await expect(page).toHaveURL(/\/orders\/\d+$/);
  await expect(page.getByRole('heading', { name: /^Đơn hàng / })).toBeVisible();
  await expect(page.getByRole('region', { name: /^Sản phẩm/ })).toBeVisible();

  await page.getByRole('link', { name: '‹ Quay lại danh sách' }).click();
  await expect(page).toHaveURL(new RegExp(`/orders\\?q=${encodeURIComponent(ORDER_QUERY)}$`));
  expect(await page.evaluate(() => [localStorage.length, sessionStorage.length])).toEqual([0, 0]);
});

test('an invalid date range is explained, an unknown order says not found', async ({ page }) => {
  await page.goto('/orders?from=2026-10-20&to=2026-10-01');
  await expect(page.getByRole('alert')).toContainText('Bộ lọc không hợp lệ');

  await page.goto('/orders/999999999');
  await expect(page.getByRole('alert')).toHaveText('Không tìm thấy đơn hàng.');
});
