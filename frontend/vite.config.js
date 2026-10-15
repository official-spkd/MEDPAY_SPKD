import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// base './' agar SPA dapat disajikan dari path mana pun (Go static + gateway)
export default defineConfig({
  base: './',
  plugins: [vue()],
  build: { outDir: 'dist', chunkSizeWarningLimit: 1200 },
  server: {
    port: 5173,
    // Target backend bisa diganti lewat env VITE_PROXY_TARGET (default Go server lokal :8080).
    proxy: { '/api': { target: process.env.VITE_PROXY_TARGET || 'http://127.0.0.1:8080', changeOrigin: true } },
  },
})
