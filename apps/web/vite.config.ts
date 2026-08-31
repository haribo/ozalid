import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// Where the API listens in development. Off :8080 on purpose — see
// apps/server/cmd/server/main.go — and overridable for anyone whose machine
// disagrees.
const API = process.env.OZALID_API ?? 'http://localhost:8090'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    // The web client only ever talks to the API (backend ADR 0002), so the dev
    // server forwards it rather than the client knowing an origin.
    proxy: { '/api': API },
  },
  // The end-to-end suite runs against the built client rather than the dev
  // server, so what CI ships is what it watches. `preview` needs the proxy of
  // its own — it does not inherit the one above.
  preview: {
    port: Number(process.env.OZALID_E2E_WEB_PORT ?? 4174),
    proxy: { '/api': API },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.spec.ts'],
  },
})
