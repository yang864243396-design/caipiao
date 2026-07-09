<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

/** 底栏三项图标 */
const TAB_ICONS = {
  lobby: '/images/lobby/tab-lobby.png',
  cloud: '/images/lobby/tab-cloud.png',
  member: '/images/lobby/tab-member.png',
} as const

const route = useRoute()

/** 仅主 Tab 页展示底栏；会员中心次级页隐藏 */
const visible = computed(() => {
  const path = route.path
  if (path === '/' || path === '/cloud' || path === '/member') return true
  return false
})

const key = computed(() => {
  if (route.path.startsWith('/cloud')) return 'cloud'
  if (route.path.startsWith('/member')) return 'member'
  return 'lobby'
})
</script>

<template>
  <nav v-if="visible" class="bottom" aria-label="底部导航">
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
    <RouterLink
      to="/member"
      class="nav-item"
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
  padding: 0.75rem 1rem calc(1.5rem + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 -8px 30px rgba(0, 0, 0, 0.04);
  border-radius: 1rem 1rem 0 0;
}
.nav-ico {
  width: 1.5rem;
  height: 1.5rem;
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
  padding: 0.5rem 0.5rem;
  min-width: 4rem;
  color: #94a3b8;
  text-decoration: none;
  font-size: 11px;
  font-family: 'Noto Sans SC', sans-serif;
  font-weight: 500;
  border-radius: 0.75rem;
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
  padding: 0.375rem 0.75rem 0.5rem;
}
.nav-lbl {
  margin-top: 0.25rem;
  line-height: 1.2;
}
</style>
