import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// The browser only ever talks to this origin; /api is proxied to the Go
// backend, so the session cookie is first-party and no CORS is needed.
const API_TARGET = process.env.API_TARGET ?? 'http://localhost:9091';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': { target: API_TARGET, changeOrigin: false },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.js'],
    include: ['src/**/*.test.{js,jsx}'],
    restoreMocks: true,
  },
});
