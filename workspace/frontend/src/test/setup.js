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

// Node's fetch (and MSW's, which wraps it) doesn't understand jsdom's FormData and
// would send "[object FormData]" as text, so uploads are encoded here the way a
// browser would. Installed after server.listen(), on top of MSW's fetch.
function encodeFormDataUploads() {
  const inner = globalThis.fetch;
  globalThis.fetch = async (input, init) => {
    if (init?.body instanceof FormData) {
      const { body, contentType } = await encodeMultipart(init.body);
      init = { ...init, body, headers: { ...init.headers, 'Content-Type': contentType } };
    }
    return inner(input, init);
  };
}

async function encodeMultipart(form) {
  const boundary = `----vitest${Math.random().toString(16).slice(2)}`;
  const enc = new TextEncoder();
  const parts = [];
  for (const [name, value] of form.entries()) {
    if (typeof value === 'string') {
      parts.push(enc.encode(`--${boundary}\r\nContent-Disposition: form-data; name="${name}"\r\n\r\n${value}\r\n`));
    } else {
      const bytes = new Uint8Array(await readBlob(value));
      parts.push(
        enc.encode(
          `--${boundary}\r\nContent-Disposition: form-data; name="${name}"; filename="${value.name}"\r\nContent-Type: ${value.type || 'application/octet-stream'}\r\n\r\n`,
        ),
        bytes,
        enc.encode('\r\n'),
      );
    }
  }
  parts.push(enc.encode(`--${boundary}--\r\n`));
  const body = new Uint8Array(parts.reduce((n, p) => n + p.length, 0));
  let offset = 0;
  for (const p of parts) {
    body.set(p, offset);
    offset += p.length;
  }
  return { body, contentType: `multipart/form-data; boundary=${boundary}` };
}

function readBlob(blob) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result);
    reader.onerror = () => reject(reader.error);
    reader.readAsArrayBuffer(blob);
  });
}

// Deterministic in-memory stand-in for cross-tab messaging.
globalThis.BroadcastChannel = FakeBroadcastChannel;

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
  encodeFormDataUploads();
});
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
