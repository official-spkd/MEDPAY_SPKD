import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// base './' agar SPA dapat disajikan dari path mana pun (Go static + gateway)
export default defineConfig({
  base: './',
  plugins: [vue()],
  build: { outDir: 'dist', chunkSizeWarningLimit: 1200 },
  server: {
    port: 5173,
    proxy: { '/api': { target: 'http://localhost:8080', changeOrigin: true } },
  },
})
