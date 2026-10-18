import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, beforeEach, vi } from 'vitest';
import { server, resetBackend, storageWrites } from './server';
import { FakeBroadcastChannel } from './fakeBroadcastChannel';
import i18n, { LANGUAGE_STORAGE_KEY } from '../i18n';

// jsdom's fetch needs absolute URLs; the app uses same-origin paths like /api/v1/...
const ORIGIN = 'http://localhost:5173';
const realFetch = globalThis.fetch;
globalThis.fetch = (input, init) =>
  realFetch(typeof input === 'string' && input.startsWith('/') ? ORIGIN + input : input, init);

// Deterministic in-memory stand-in for cross-tab messaging.
globalThis.BroadcastChannel = FakeBroadcastChannel;

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
beforeEach(async () => {
  // Tests assert Vietnamese text: pin the UI language (jsdom's browser language is en-US).
  await i18n.changeLanguage('vi');
  resetBackend();
  FakeBroadcastChannel.reset();
  storageWrites.count = 0;
  storageWrites.language = null;
  // The UI language code is the only thing the app may store in the browser.
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation((key, value) => {
    if (key === LANGUAGE_STORAGE_KEY) storageWrites.language = value;
    else storageWrites.count++;
  });
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => null);
});
afterEach(() => {
  cleanup();
  server.resetHandlers();
});
afterAll(() => server.close());
