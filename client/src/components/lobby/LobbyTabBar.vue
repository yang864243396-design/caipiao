<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useLayoutMode } from '@/composables/useLayoutMode'
import { demoAppBrand } from '@/demo/demoAccount'

/** 底栏三项图标 */
const TAB_ICONS = {
  lobby: '/images/lobby/tab-lobby.png',
  cloud: '/images/lobby/tab-cloud.png',
  member: '/images/lobby/tab-member.png',
} as const

const route = useRoute()
const { isWeb } = useLayoutMode()

/**
 * H5：仅主 Tab 展示底栏。
 * Web：对齐第三方桌面顶栏，登录页外常驻。
 */
const visible = computed(() => {
  if (route.path === '/login') return false
  if (isWeb.value) return true
  const path = route.path
  return path === '/' || path === '/cloud' || path === '/member'
})

const key = computed(() => {
  if (route.path.startsWith('/cloud')) return 'cloud'
  if (route.path.startsWith('/member')) return 'member'
  if (route.path === '/' || route.path.startsWith('/copy') || route.path.startsWith('/scheme') || route.path.startsWith('/play')) {
    return 'lobby'
  }
  return 'lobby'
})

const navLabel = computed(() => (isWeb.value ? '主导航' : '底部导航'))
</script>

<template>
  <nav v-if="visible" class="bottom" :class="{ 'bottom--web': isWeb }" :aria-label="navLabel">
    <div class="nav-inner">
      <div v-if="isWeb" class="nav-brand" aria-hidden="true">
        <span class="nav-brand-mark">{{ demoAppBrand.slice(0, 1) }}</span>
        <span class="nav-brand-name">{{ demoAppBrand }}</span>
      </div>
      <div class="nav-links">
        <RouterLink
          to="/"
          class="nav-item"
          :class="{ active: key === 'lobby' }"
        >
          <img
            :src="TAB_ICONS.lobby"
            alt=""
            width="24"
            height="24"
            class="nav-ico"
            decoding="async"
          />
          <span class="nav-lbl">游戏大厅</span>
        </RouterLink>
        <RouterLink
          to="/cloud"
          class="nav-item"
          :class="{ active: key === 'cloud' }"
        >
          <img
            :src="TAB_ICONS.cloud"
            alt=""
            width="24"
            height="24"
            class="nav-ico"
            decoding="async"
          />
          <span class="nav-lbl">云端中心</span>
        </RouterLink>
      </div>
      <RouterLink
        to="/member"
        class="nav-item nav-item--member"
        :class="{ active: key === 'member' }"
      >
        <img
          :src="TAB_ICONS.member"
          alt=""
          width="24"
          height="24"
          class="nav-ico"
          decoding="async"
        />
        <span class="nav-lbl">会员中心</span>
      </RouterLink>
    </div>
  </nav>
</template>

<style scoped>
.bottom {
  position: fixed;
  bottom: 0;
  left: 0;
  z-index: 50;
  width: 100%;
  display: flex;
  justify-content: space-around;
  align-items: center;
  padding: 0.3rem 0.75rem calc(0.45rem + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 -8px 30px rgba(0, 0, 0, 0.04);
  border-radius: 0.85rem 0.85rem 0 0;
}
.nav-inner {
  display: contents;
}
.nav-brand {
  display: none;
}
.nav-links {
  display: contents;
}
.nav-ico {
  width: 1.75rem;
  height: 1.75rem;
  object-fit: contain;
  display: block;
  flex-shrink: 0;
  transition: opacity 0.2s;
}
.nav-item:not(.active) .nav-ico {
  opacity: 0.55;
}
.nav-item.active .nav-ico {
  opacity: 1;
}
.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0.2rem 0.4rem;
  min-width: 3.75rem;
  color: #94a3b8;
  text-decoration: none;
  font-size: 12px;
  font-family: 'Noto Sans SC', sans-serif;
  font-weight: 600;
  border-radius: 0.65rem;
  transition:
    color 0.2s,
    background 0.2s,
    transform 0.2s;
}
.nav-item:hover {
  color: #3b82f6;
}
.nav-item:active {
  transform: scale(0.9);
}
.nav-item.active {
  color: #2563eb;
  background: rgba(239, 246, 255, 0.5);
  padding: 0.2rem 0.55rem 0.25rem;
}
.nav-lbl {
  margin-top: 0.12rem;
  line-height: 1.15;
}

/**
 * Web 顶栏：背景通栏，菜单内容与页面壳同宽居中
 */
html.layout-web .bottom,
html.layout-web .bottom--web {
  top: 0;
  bottom: auto;
  left: 0;
  right: 0;
  width: 100%;
  height: var(--layout-web-nav-height, 3.75rem);
  flex-direction: row;
  justify-content: center;
  align-items: center;
  gap: 0;
  padding: 0 var(--layout-web-gutter, 1.5rem);
  border-radius: 0;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(28px);
  -webkit-backdrop-filter: blur(28px);
  box-shadow: 0 10px 36px rgba(15, 23, 42, 0.06);
}
html.layout-web .nav-inner {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  width: 100%;
  max-width: var(--layout-web-shell, 75rem);
  height: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}
html.layout-web .nav-brand {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0;
  margin: 0 1.75rem 0 0;
  flex-shrink: 0;
}
html.layout-web .nav-brand-mark {
  width: 2.1rem;
  height: 2.1rem;
  border-radius: 0.65rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 800;
  font-size: 0.95rem;
  color: #fff;
  background: linear-gradient(180deg, #0066ff 0%, #0050cb 100%);
  box-shadow: 0 10px 24px -12px rgba(0, 80, 203, 0.55);
}
html.layout-web .nav-brand-name {
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.05rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #0f172a;
  line-height: 1.2;
}
html.layout-web .nav-links {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  min-width: 0;
}
html.layout-web .nav-item {
  flex-direction: row;
  justify-content: center;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  height: 2.5rem;
  padding: 0 1rem;
  font-size: 0.875rem;
  font-weight: 600;
  border-radius: 0.75rem;
}
html.layout-web .nav-item:active {
  transform: none;
}
html.layout-web .nav-item.active {
  padding: 0 1rem;
  background: rgba(0, 80, 203, 0.08);
  color: #0050cb;
}
html.layout-web .nav-item--member {
  margin-left: auto;
}
html.layout-web .nav-lbl {
  margin-top: 0;
}
html.layout-web .nav-ico {
  width: 1.2rem;
  height: 1.2rem;
}
</style>
