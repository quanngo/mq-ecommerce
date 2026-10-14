import { describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { safeNextPath, googleLoginUrl } from './auth';
import { apiRequest, ApiError, onUnauthorized } from './client';
import { authErrorMessage } from '../auth/errorMessages';
import { server } from '../test/server';

describe('safeNextPath', () => {
  it.each([
    [null, '/profile'],
    ['', '/profile'],
    ['/orders?page=2', '/orders?page=2'],
    ['//evil.example', '/profile'],
    ['/\\evil.example', '/profile'],
    ['https://evil.example', '/profile'],
    ['/x/https://evil', '/profile'],
    ['evil', '/profile'],
    ['/login?next=/x', '/profile'],
  ])('%s -> %s', (input, expected) => {
    expect(safeNextPath(input)).toBe(expected);
  });

  it('encodes next into the Google login URL', () => {
    expect(googleLoginUrl('/orders?page=2')).toBe('/api/v1/auth/google/login?next=%2Forders%3Fpage%3D2');
    expect(googleLoginUrl('/profile', true)).toBe('/api/v1/auth/google/login?next=%2Fprofile&remember=1');
    expect(googleLoginUrl('/profile', false)).toBe('/api/v1/auth/google/login?next=%2Fprofile');
  });
});

describe('apiRequest', () => {
  it('unwraps data[0]', async () => {
    server.use(http.get('*/api/v1/thing', () => HttpResponse.json({ code: 200, message: 'Success', data: [{ a: 1 }] })));
    await expect(apiRequest('GET', '/thing')).resolves.toEqual({ a: 1 });
  });

  it('turns a non-200 body code into ApiError with the error code', async () => {
    server.use(http.get('*/api/v1/thing', () => HttpResponse.json({ code: 423, message: 'ACCOUNT_LOCKED' })));
    const err = await apiRequest('GET', '/thing').catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ code: 423, errorCode: 'ACCOUNT_LOCKED' });
  });

  it('notifies listeners on UNAUTHENTICATED', async () => {
    server.use(http.get('*/api/v1/thing', () => HttpResponse.json({ code: 401, message: 'UNAUTHENTICATED' })));
    const listener = vi.fn();
    const off = onUnauthorized(listener);
    await apiRequest('GET', '/thing').catch(() => undefined);
    off();
    expect(listener).toHaveBeenCalledOnce();
  });

  it('reports network failures as NETWORK_ERROR', async () => {
    server.use(http.get('*/api/v1/thing', () => HttpResponse.error()));
    await expect(apiRequest('GET', '/thing')).rejects.toMatchObject({ errorCode: 'NETWORK_ERROR' });
  });
});

describe('authErrorMessage', () => {
  it('has no message for UNAUTHENTICATED or nothing', () => {
    expect(authErrorMessage(null)).toBeNull();
    expect(authErrorMessage('UNAUTHENTICATED')).toBeNull();
  });
  it('falls back for unknown codes', () => {
    expect(authErrorMessage('NOPE')).toBe('Đã có lỗi xảy ra. Vui lòng thử lại.');
  });
});
