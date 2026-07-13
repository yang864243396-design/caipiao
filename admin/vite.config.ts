import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

const adminSrc = fileURLToPath(new URL('./src', import.meta.url))
const clientSrc = fileURLToPath(new URL('../client/src', import.meta.url))
const sharedSrc = fileURLToPath(new URL('../shared', import.meta.url))

/** 会员端 `@/…` 路径：供 admin 复用的 client 模块内部 import 回落到 client/src */
const clientScopedAliases: Array<{ find: string | RegExp; replacement: string }> = [
  { find: '@/constants/lhcPlay', replacement: `${clientSrc}/constants/lhcPlay` },
  { find: '@/constants/betModeOptions', replacement: `${clientSrc}/constants/betModeOptions` },
  { find: '@/types/playCatalog', replacement: `${clientSrc}/types/playCatalog` },
  { find: '@/utils/betPayload', replacement: `${clientSrc}/utils/betPayload` },
  { find: '@/utils/pickPanelOptions', replacement: `${clientSrc}/utils/pickPanelOptions` },
  { find: '@/utils/playConfig', replacement: `${clientSrc}/utils/playConfig` },
  { find: '@/utils/playInputProfile', replacement: `${clientSrc}/utils/playInputProfile` },
  { find: '@/utils/runTypeMatrix', replacement: `${clientSrc}/utils/runTypeMatrix` },
  { find: '@/utils/longhuPickOptions', replacement: `${clientSrc}/utils/longhuPickOptions` },
  { find: '@/utils/playTypeLabels', replacement: `${clientSrc}/utils/playTypeLabels` },
  { find: '@/utils/betMultiplierPlan', replacement: `${clientSrc}/utils/betMultiplierPlan` },
]

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver({ importStyle: 'css' })],
    }),
  ],
  resolve: {
    alias: [
      ...clientScopedAliases,
      { find: '@shared', replacement: sharedSrc },
      { find: '@client', replacement: clientSrc },
      { find: '@', replacement: adminSrc },
    ],
  },
  server: { port: 5174, host: true },
})
