import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, beforeEach, vi } from 'vitest';
import { server, resetBackend, storageWrites } from './server';
import { FakeBroadcastChannel } from './fakeBroadcastChannel';

// jsdom's fetch needs absolute URLs; the app uses same-origin paths like /api/v1/...
const ORIGIN = 'http://localhost:5173';
const realFetch = globalThis.fetch;
globalThis.fetch = (input, init) =>
  realFetch(typeof input === 'string' && input.startsWith('/') ? ORIGIN + input : input, init);

// Deterministic in-memory stand-in for cross-tab messaging.
globalThis.BroadcastChannel = FakeBroadcastChannel;

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
beforeEach(() => {
  resetBackend();
  FakeBroadcastChannel.reset();
  storageWrites.count = 0;
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
    storageWrites.count++;
  });
});
afterEach(() => {
  cleanup();
  server.resetHandlers();
});
afterAll(() => server.close());
