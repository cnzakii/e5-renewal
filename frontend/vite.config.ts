import { defineConfig } from 'vite'
// @ts-ignore
import vue from '@vitejs/plugin-vue'
// @ts-ignore
import tailwindcss from '@tailwindcss/vite'


export default defineConfig({
  base: './',
  plugins: [vue(), tailwindcss()],
  // @ts-ignore
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/__tests__/**/*.spec.ts']
  },
  server: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
})
