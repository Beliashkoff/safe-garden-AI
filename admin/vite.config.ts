import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Dev: the Go API runs on :8080 — proxy /admin/v1 so cookies stay same-origin.
// Prod: Caddy on admin.<domain> routes /admin/* to the api container and
// everything else to the static files this build produces.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/admin/v1': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
});
