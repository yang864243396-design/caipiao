import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 单独一份配置，不动 vite.config.ts：
// 跑单测不需要 Element Plus 的按需自动引入，少一层插件也少一类构建期的干扰。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@shared': fileURLToPath(new URL('../shared', import.meta.url)),
    },
  },
  test: {
    // sessionStorage、document 等浏览器 API 由 jsdom 提供
    environment: 'jsdom',
    include: ['src/**/*.spec.ts'],
    restoreMocks: true,
    coverage: {
      provider: 'v8',
      include: ['src/utils/**', 'src/composables/**'],
    },
  },
})
