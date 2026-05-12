import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import GameLobbyView from '@/views/lobby/GameLobbyView.vue'
import TabPlaceholderView from '@/views/TabPlaceholderView.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'lobby',
    component: GameLobbyView,
    meta: { title: '游戏大厅' },
  },
  {
    path: '/copy-hall',
    name: 'copy-hall',
    component: () => import('@/views/copy/CopyHallView.vue'),
    meta: { title: '跟单大厅' },
  },
  {
    path: '/play/detail',
    name: 'play-detail',
    component: () => import('@/views/play/GameDetailView.vue'),
    meta: { title: '游戏详情' },
  },
  {
    path: '/play/custom-scheme/new',
    name: 'custom-scheme-new',
    component: () => import('@/views/play/CustomSchemeNewView.vue'),
    meta: { title: '新增方案' },
  },
  {
    path: '/play/bet-multiplier-settings',
    name: 'bet-multiplier-settings',
    component: () => import('@/views/play/BetMultiplierSettingsView.vue'),
    meta: { title: '倍投设定' },
  },
  {
    path: '/play/bet-multiplier/advanced-scheme/:schemeId',
    name: 'advanced-scheme-edit',
    component: () => import('@/views/play/AdvancedSchemeEditView.vue'),
    meta: { title: '方案配置' },
  },
  {
    path: '/play/bet-multiplier/advanced-scheme/:schemeId/rounds',
    name: 'advanced-scheme-rounds',
    component: () => import('@/views/play/AdvancedSchemeRoundsView.vue'),
    meta: { title: '方案模式' },
  },
  {
    path: '/announcement/:id',
    name: 'announcement-detail',
    component: () => import('@/views/announcement/AnnouncementDetailView.vue'),
    meta: { title: '公告详情' },
  },
  {
    path: '/cloud',
    name: 'cloud',
    component: () => import('@/views/cloud/CloudCenterView.vue'),
    meta: { title: '云端中心' },
  },
  {
    path: '/bet-records',
    name: 'bet-records',
    component: () => import('@/views/cloud/BetRecordsView.vue'),
    meta: { title: '最近三日投注记录' },
  },
  {
    path: '/member',
    name: 'member',
    component: TabPlaceholderView,
    props: { title: '会员中心', imageSrc: '/images/lobby/nav-member.png' },
    meta: { title: '会员中心' },
  },
]
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.afterEach((to) => {
  if (to.name === 'play-detail') return
  const t = to.meta?.title
  document.title = typeof t === 'string' ? `${t} · 精密终端` : '精密终端'
})
