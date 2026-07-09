import { createRouter, createWebHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useAdminRolesStore } from '@/stores/adminRoles'
import { canAccessPath } from '@/utils/menuRbac'

const LoginView = () => import('@/views/LoginView.vue')
const AdminLayout = () => import('@/layouts/AdminLayout.vue')
const DashboardView = () => import('@/views/DashboardView.vue')
const RolesPlaceholderView = () => import('@/views/system/RolesPlaceholderView.vue')
const AdminUsersView = () => import('@/views/system/AdminUsersView.vue')
const BannerManageView = () => import('@/views/content/BannerManageView.vue')
const AnnouncementListView = () => import('@/views/content/AnnouncementListView.vue')
const FaqManageView = () => import('@/views/content/FaqManageView.vue')
const MemberListView = () => import('@/views/members/MemberListView.vue')
const MemberDetailView = () => import('@/views/members/MemberDetailView.vue')
const SchemeMonitorView = () => import('@/views/schemes/SchemeMonitorView.vue')
const LotteryCatalogView = () => import('@/views/games/LotteryCatalogView.vue')
const CopyHallOpsView = () => import('@/views/games/CopyHallOpsView.vue')
const GlobalSchemeDefaultsView = () => import('@/views/games/GlobalSchemeDefaultsView.vue')
const BetOrdersView = () => import('@/views/orders/BetOrdersView.vue')
const LedgerView = () => import('@/views/orders/LedgerView.vue')
const LotteryStatReportView = () => import('@/views/reports/LotteryStatReportView.vue')
const AuditLogView = () => import('@/views/system/AuditLogView.vue')
const MaintenanceView = () => import('@/views/operations/MaintenanceView.vue')
const CustomerServiceSettingsView = () => import('@/views/service/CustomerServiceSettingsView.vue')
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      meta: {},
      children: [
        { path: 'dashboard', name: 'dashboard', meta: { title: '仪表盘' }, component: DashboardView },
        { path: 'members', name: 'member-list', meta: { title: '会员查询' }, component: MemberListView },
        { path: 'members/:id', name: 'member-detail', meta: { title: '会员详情' }, component: MemberDetailView },
        {
          path: 'schemes/monitor',
          name: 'scheme-monitor',
          meta: { title: '全站方案监控' },
          component: SchemeMonitorView,
        },
        {
          path: 'games/lottery-catalog',
          name: 'lottery-catalog',
          meta: { title: '彩种目录' },
          component: LotteryCatalogView,
        },
        {
          path: 'games/copy-hall',
          name: 'games-copy-hall',
          meta: { title: '跟单大厅运营' },
          component: CopyHallOpsView,
        },
        {
          path: 'games/scheme-defaults',
          name: 'games-scheme-defaults',
          meta: { title: '方案模板库' },
          component: GlobalSchemeDefaultsView,
        },
        {
          path: 'orders/bets',
          name: 'orders-bets',
          meta: { title: '投注与追号' },
          component: BetOrdersView,
        },
        {
          path: 'orders/ledger',
          name: 'orders-ledger',
          meta: { title: '帐变流水' },
          component: LedgerView,
        },
        {
          path: 'reports/lottery-stat',
          name: 'report-lottery-stat',
          meta: { title: '经营报表' },
          component: LotteryStatReportView,
        },
        // 旧「盈亏报表」已并入经营报表，保留路径重定向避免书签/角色菜单失效
        { path: 'reports/pnl', redirect: '/reports/lottery-stat' },
        {
          path: 'operations/maintenance',
          name: 'operations-maintenance',
          meta: { title: '系统维护' },
          component: MaintenanceView,
        },
        {
          path: 'content/banners',
          name: 'content-banners',
          meta: { title: 'Banner 管理' },
          component: BannerManageView,
        },
        { path: 'content/lobby', redirect: '/content/banners' },
        {
          path: 'content/announcements',
          name: 'content-announcements',
          meta: { title: '公告管理' },
          component: AnnouncementListView,
        },
        { path: 'content/faq', name: 'content-faq', meta: { title: '常见问题' }, component: FaqManageView },
        // 旧「站点品牌」已下线，保留路径重定向避免书签/角色菜单失效
        { path: 'content/site-brand', redirect: '/content/banners' },
        {
          path: 'service/customer-service',
          name: 'service-customer-service',
          meta: { title: '客服设置' },
          component: CustomerServiceSettingsView,
        },
        { path: 'system/roles', name: 'roles', meta: { title: '角色管理' }, component: RolesPlaceholderView },
        { path: 'system/admin-users', name: 'admin-users', meta: { title: 'Admin 账号' }, component: AdminUsersView },
        { path: 'system/audit', name: 'audit-log', meta: { title: '操作审计' }, component: AuditLogView },
        // 旧「上线备忘」已下线，保留路径重定向避免书签/角色菜单失效
        { path: 'system/go-live-memo', redirect: '/dashboard' },
      ],
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore()

  if (to.meta.public) {
    if (to.path === '/login' && auth.isAuthenticated) {
      next(typeof to.query.redirect === 'string' ? to.query.redirect : '/dashboard')
      return
    }
    next()
    return
  }

  if (!auth.isAuthenticated) {
    auth.logout()
    ElMessage.warning('登录已失效，请重新登录')
    next({ path: '/login', query: { redirect: to.fullPath } })
    return
  }

  const rolesStore = useAdminRolesStore()
  await rolesStore.hydrate()
  const role = rolesStore.roles.find((r) => r.id === auth.adminRoleId)
  const menuPaths = role?.menuPaths ?? ['/']
  if (!canAccessPath(to.path, menuPaths)) {
    ElMessage.warning('当前角色无权访问该页面')
    next('/dashboard')
    return
  }

  next()
})
