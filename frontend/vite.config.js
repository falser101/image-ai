import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') }
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    // Vite 默认只允许 localhost，外网域名会被拦（防止 DNS rebinding）。
    // 调试期间要加白名单；多域名用数组，加新域名直接 append。
    allowedHosts: ['image.falser101.xyz', 'localhost', '127.0.0.1'],
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/uploads': { target: 'http://localhost:8080', changeOrigin: true }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
