import { afterEach, describe, expect, it, vi } from 'vitest';
import en from '../locales/en.json';
import viCatalog from '../locales/vi.json';
import i18n, { detectLanguage, LANGUAGE_STORAGE_KEY, normalizeLanguage } from './index';
import { formatDateTime, formatMoney, formatNumber } from './format';
import { actionErrorMessage, authErrorMessage, orderErrorMessage, permissionLabel } from '../auth/errorMessages';
import { ApiError } from '../api/client';
import { leadStatusLabel, paymentMethodLabel, saleOrderStatusLabel } from '../pages/orderFormat';

/** Flattened keys, with plural suffixes (_one, _other…) folded into the base key. */
function keys(obj, prefix = '') {
  return Object.entries(obj).flatMap(([k, v]) =>
    v && typeof v === 'object' ? keys(v, `${prefix}${k}.`) : [`${prefix}${k}`.replace(/_(zero|one|two|few|many|other)$/, '')],
  );
}

describe('catalogs', () => {
  it('vi.json and en.json have the same keys', () => {
    expect([...new Set(keys(en))].sort()).toEqual([...new Set(keys(viCatalog))].sort());
  });

  it('has no empty strings', () => {
    const empty = (obj) => Object.values(obj).some((v) => (v && typeof v === 'object' ? empty(v) : v === ''));
    expect(empty(en)).toBe(false);
    expect(empty(viCatalog)).toBe(false);
  });
});

describe('language detection', () => {
  afterEach(() => vi.unstubAllGlobals());

  it.each([
    ['en', 'en'],
    ['en-US', 'en'],
    ['VI_vn', 'vi'],
    ['fr', null],
    ['', null],
    [null, null],
  ])('normalizeLanguage(%s) = %s', (tag, want) => {
    expect(normalizeLanguage(tag)).toBe(want);
  });

  it('prefers the stored choice, then the browser language, then Vietnamese', () => {
    const getItem = vi.spyOn(Storage.prototype, 'getItem');

    getItem.mockImplementation((key) => (key === LANGUAGE_STORAGE_KEY ? 'en' : null));
    vi.stubGlobal('navigator', { languages: ['vi-VN'], language: 'vi-VN' });
    expect(detectLanguage()).toBe('en');

    getItem.mockImplementation(() => null);
    vi.stubGlobal('navigator', { languages: ['fr-FR', 'en-GB'], language: 'fr-FR' });
    expect(detectLanguage()).toBe('en');

    vi.stubGlobal('navigator', { languages: ['fr-FR', 'de'], language: 'fr-FR' });
    expect(detectLanguage()).toBe('vi');

    getItem.mockImplementation(() => {
      throw new Error('SecurityError'); // storage blocked
    });
    vi.stubGlobal('navigator', { languages: undefined, language: 'en-US' });
    expect(detectLanguage()).toBe('en');
  });
});

describe('formatting follows the language', () => {
  afterEach(() => i18n.changeLanguage('vi'));

  it('formats dates in Vietnam time, numbers and VND', async () => {
    const iso = '2026-10-14T17:30:00Z'; // 00:30 on the 15th in Vietnam
    expect(formatDateTime(iso)).toMatch(/15.*10.*2026|15 thg 10, 2026/);
    expect(formatNumber(1234567)).toBe('1.234.567');
    expect(formatMoney(150000)).toMatch(/150\.000\s?₫/);

    await i18n.changeLanguage('en');
    expect(formatDateTime(iso)).toMatch(/Oct 15, 2026/);
    expect(formatNumber(1234567)).toBe('1,234,567');
    expect(formatMoney(150000)).toMatch(/₫\s?150,000/);
    expect(formatDateTime(null)).toBe('—');
    expect(formatDateTime('not a date')).toBe('—');
  });

  it('translates order labels', async () => {
    expect(leadStatusLabel(4)).toBe('Đã huỷ');
    expect(leadStatusLabel(9)).toBe('Trạng thái 9');
    await i18n.changeLanguage('en');
    expect(leadStatusLabel(4)).toBe('Cancelled');
    expect(leadStatusLabel(9)).toBe('Status 9');
    expect(saleOrderStatusLabel(45)).toBe('Successful');
    expect(saleOrderStatusLabel(999)).toBe('999');
    expect(saleOrderStatusLabel(null)).toBe('—');
    expect(paymentMethodLabel(1)).toBe('Cash on delivery');
    expect(paymentMethodLabel(0)).toBe('—');
  });
});

describe('error messages', () => {
  afterEach(() => i18n.changeLanguage('vi'));

  it('uses the page wording, then the shared wording, then the backend text, then a generic message', async () => {
    await i18n.changeLanguage('en');
    expect(authErrorMessage('INVALID_CREDENTIALS')).toBe('Incorrect email or password.');
    expect(authErrorMessage('UNAUTHENTICATED')).toBeNull();
    expect(authErrorMessage(null)).toBeNull();
    expect(orderErrorMessage(new ApiError(404, 'NOT_FOUND'))).toBe('Order not found.');
    expect(orderErrorMessage(new ApiError(409, 'EMAIL_TAKEN'))).toBe('An account with this email already exists.');
    expect(actionErrorMessage(new ApiError(418, 'BRAND_NEW_CODE', 'Text from the backend.'))).toBe('Text from the backend.');
    expect(actionErrorMessage(new ApiError(418, 'BRAND_NEW_CODE'))).toBe('Something went wrong. Please try again.');
    expect(actionErrorMessage(null)).toBe('Something went wrong. Please try again.');
  });

  it('labels permissions from the backend names, falling back to the catalog and then the code', async () => {
    expect(permissionLabel('orders.read', { 'orders.read': 'From backend' })).toBe('From backend');
    expect(permissionLabel('orders.read')).toBe('Xem đơn hàng');
    await i18n.changeLanguage('en');
    expect(permissionLabel('orders.detail', {})).toBe('View order details');
    expect(permissionLabel('reports.export')).toBe('reports.export');
  });
});
