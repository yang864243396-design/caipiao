import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'
import { demoAppBrand } from '@/demo/demoAccount'
import { getAccessToken } from '@/api/client'
import {
  lotteryRouteToastMessage,
  resolveRouteLotteryBlock,
  routeNeedsLotteryGuard,
} from '@/composables/useLotteryRouteGuard'
import {
  consumeGuajiGateToast,
  guajiGateToastMessage,
  guajiRouteRedirect,
  resolveGuajiAuthStatus,
  routeNeedsGuajiGuard,
  setPendingGuajiGateToast,
} from '@/composables/useGuajiAuthGuard'
import GameLobbyView from '@/views/lobby/GameLobbyView.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: { title: '登录', public: true },
  },
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
    path: '/scheme-download',
    name: 'scheme-download',
    component: () => import('@/views/scheme/SchemeDownloadView.vue'),
    meta: { title: '方案下载' },
  },
  {
    path: '/play/detail',
    name: 'play-detail',
    component: () => import('@/views/play/GameDetailView.vue'),
    meta: { title: '游戏详情' },
  },
  {
    path: '/play/custom-scheme/new',
    redirect: {
      name: 'advanced-scheme-edit',
      params: { schemeId: 'new' },
      query: { draft: '1', kind: 'custom', fresh: '1' },
    },
  },
  {
    path: '/play/scheme-detail/:definitionId',
    name: 'scheme-detail',
    component: () => import('@/views/play/SchemeDetailView.vue'),
    meta: { title: '方案详情' },
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
    path: '/bet-records/:schemeId',
    name: 'bet-records-scheme',
    component: () => import('@/views/cloud/BetRecordsSchemeDetailView.vue'),
    meta: { title: '方案投注明细' },
  },
  {
    path: '/member/auth/bind',
    name: 'member-auth-bind',
    component: () => import('@/views/member/auth/GuajiBindView.vue'),
    meta: { title: '绑定授权账号' },
  },
  {
    path: '/member/auth/list',
    name: 'member-auth-list',
    component: () => import('@/views/member/auth/GuajiListView.vue'),
    meta: { title: '授权账号' },
  },
  {
    path: '/member',
    name: 'member',
    component: () => import('@/views/member/MemberCenterView.vue'),
    meta: { title: '会员中心' },
  },
  {
    path: '/member/fund-records',
    name: 'member-fund-records',
    component: () => import('@/views/member/FundRecordsView.vue'),
    meta: { title: '资金记录' },
  },
  {
    // 修改资料子页面下线（余额/资料以第三方为准），重定向会员中心
    path: '/member/profile',
    redirect: '/member',
  },
  {
    path: '/member/feedback',
    name: 'member-feedback',
    component: () => import('@/views/member/MemberFeedbackView.vue'),
    meta: { title: '意见回馈' },
  },
  {
    path: '/member/announcements',
    name: 'member-announcements',
    component: () => import('@/views/member/MemberAnnouncementsView.vue'),
    meta: { title: '平台公告' },
  },
  {
    // 彩种统计下线，重定向会员中心
    path: '/member/lottery-stat',
    redirect: '/member',
  },
  {
    // §22.3：帐变记录下线，重定向会员中心
    path: '/member/ledger',
    redirect: '/member',
  },
  {
    path: '/member/bet-records',
    name: 'member-bet-records',
    component: () => import('@/views/member/MemberBetRecordsView.vue'),
    meta: { title: '投注记录' },
  },
  {
    path: '/member/scheme-pnl',
    name: 'member-scheme-pnl',
    component: () => import('@/views/member/MemberSchemePnlView.vue'),
    meta: { title: '方案盈亏' },
  },
  {
    // §22.3：追号记录下线，重定向会员中心
    path: '/member/chase-records',
    redirect: '/member',
  },
  {
    path: '/member/faq',
    name: 'member-faq',
    component: () => import('@/views/member/MemberFaqView.vue'),
    meta: { title: '常见问题' },
  },
  {
    path: '/member/faq/:id',
    name: 'member-faq-detail',
    component: () => import('@/views/member/MemberFaqDetailView.vue'),
    meta: { title: '问题详情' },
  },
  {
    // 帮助中心下线，重定向会员中心
    path: '/member/help',
    redirect: '/member',
  },
  {
    // 盈亏报表下线，重定向会员中心
    path: '/member/pnl-report',
    redirect: '/member',
  },
  {
    // 聊天室下线，重定向会员中心
    path: '/member/chat/system',
    redirect: '/member',
  },
  {
    path: '/member/chat',
    redirect: '/member',
  },
  {
    path: '/member/chat/:peerId',
    redirect: '/member',
  },
]
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.beforeEach(async (to) => {
  const authed = !!getAccessToken()
  // 已登录访问登录页 → 跳回目标或大厅
  if (to.meta.public) {
    if (to.name === 'login' && authed) {
      const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : '/'
      return redirect
    }
    return true
  }
  // 未登录访问受保护页 → 跳登录页并带回跳地址
  if (!authed) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (routeNeedsGuajiGuard(to)) {
    try {
      const guajiStatus = await resolveGuajiAuthStatus()
      if (!guajiStatus.hasActiveGuajiAuth) {
        const redirect = guajiRouteRedirect(to, guajiStatus)
        setPendingGuajiGateToast(guajiGateToastMessage(guajiStatus))
        return redirect
      }
    } catch {
      // 后端未就绪时不阻断（开发降级）
    }
  }
  if (routeNeedsLotteryGuard(to)) {
    const block = await resolveRouteLotteryBlock(to)
    if (block) {
      return { name: 'lobby', state: { lotteryToast: block } }
    }
  }
  return true
})

router.afterEach((to) => {
  const state = history.state as { lotteryToast?: 'offline' | 'maintenance' } | null
  const toast = state?.lotteryToast
  if (to.name === 'lobby' && toast) {
    const msg = lotteryRouteToastMessage(toast)
    if (msg) ElMessage.warning({ message: msg, duration: 3000 })
  }
  if (to.name === 'member-auth-bind' || to.name === 'member-auth-list') {
    const guajiMsg = consumeGuajiGateToast()
    if (guajiMsg) ElMessage.warning({ message: guajiMsg, duration: 3000 })
  }
  if (to.name === 'play-detail') return
  const t = to.meta?.title
  document.title = typeof t === 'string' ? `${t} · ${demoAppBrand}` : demoAppBrand
})
