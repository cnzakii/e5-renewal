import { defineConfig } from 'vite'
// @ts-expect-error vite plugin types are resolved by Vite at runtime
import vue from '@vitejs/plugin-vue'
// @ts-expect-error vite plugin types are resolved by Vite at runtime
import tailwindcss from '@tailwindcss/vite'


export default defineConfig({
  base: './',
  plugins: [vue(), tailwindcss()],
  // @ts-expect-error vitest config is accepted by Vite
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
