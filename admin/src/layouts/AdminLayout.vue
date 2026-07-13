<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, RouterView, useRouter } from 'vue-router'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'
import {
  Cpu,
  DataAnalysis,
  DocumentCopy,
  House,
  Monitor,
  Setting,
  ShoppingCart,
  SwitchButton,
  Trophy,
  UserFilled,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useAdminRbac } from '@/composables/useAdminRbac'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { activeRole, canAccess, canAccessSome } = useAdminRbac()

/** 子路由高亮：会员详情仍归在「会员与用户」下 */
const activeMenu = computed(() => {
  const p = route.path
  if (p.startsWith('/members')) return '/members'
  return p
})

async function onLogout() {
  const ok = await adminConfirmDialog({
    title: '退出登录',
    message: '确认退出当前账号？',
    tone: 'primary',
  })
  if (!ok) return
  auth.logout()
  await router.replace('/login')
}
</script>

<template>
  <div class="admin-shell">
    <el-container class="admin-shell-inner" direction="horizontal">
      <el-aside class="admin-aside">
        <div class="admin-brand">管理后台</div>
        <el-menu :default-active="activeMenu" unique-opened router class="admin-el-menu">
          <el-menu-item v-if="canAccess('/dashboard')" index="/dashboard">
            <el-icon>
              <House />
            </el-icon>
            <span>仪表盘</span>
          </el-menu-item>

          <el-sub-menu v-if="canAccess('/operations/maintenance')" index="/operations">
            <template #title>
              <el-icon>
                <Setting />
              </el-icon>
              <span>运维</span>
            </template>
            <el-menu-item index="/operations/maintenance">系统维护</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAccessSome(['/members', '/schemes/monitor'])" index="members-group">
            <template #title>
              <el-icon>
                <UserFilled />
              </el-icon>
              <span>会员与用户</span>
            </template>
            <el-menu-item v-if="canAccess('/members')" index="/members">会员查询</el-menu-item>
            <el-menu-item v-if="canAccess('/schemes/monitor')" index="/schemes/monitor">全站方案监控</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAccessSome(['/games/lottery-catalog', '/games/copy-hall', '/games/scheme-defaults'])"
            index="games-group">
            <template #title>
              <el-icon>
                <Trophy />
              </el-icon>
              <span>游戏与玩法</span>
            </template>
            <el-menu-item v-if="canAccess('/games/lottery-catalog')" index="/games/lottery-catalog">彩种目录</el-menu-item>
            <el-menu-item v-if="canAccess('/games/copy-hall')" index="/games/copy-hall">跟单大厅运营</el-menu-item>
            <el-menu-item v-if="canAccess('/games/scheme-defaults')" index="/games/scheme-defaults">方案模板库</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAccessSome(['/orders/bets', '/orders/ledger'])" index="orders-group">
            <template #title>
              <el-icon>
                <ShoppingCart />
              </el-icon>
              <span>订单与帐变</span>
            </template>
            <el-menu-item v-if="canAccess('/orders/bets')" index="/orders/bets">投注与追号</el-menu-item>
            <el-menu-item v-if="canAccess('/orders/ledger')" index="/orders/ledger">帐变流水</el-menu-item>
          </el-sub-menu>

          <el-menu-item v-if="canAccess('/reports/lottery-stat')" index="/reports/lottery-stat">
            <el-icon>
              <DataAnalysis />
            </el-icon>
            <span>经营报表</span>
          </el-menu-item>

          <el-sub-menu
            v-if="canAccessSome(['/content/banners', '/content/lobby', '/content/announcements', '/content/faq'])"
            index="content-group">
            <template #title>
              <el-icon>
                <DocumentCopy />
              </el-icon>
              <span>站点与内容</span>
            </template>
            <el-menu-item v-if="canAccessSome(['/content/banners', '/content/lobby'])" index="/content/banners">
              Banner 管理
            </el-menu-item>
            <el-menu-item v-if="canAccess('/content/announcements')" index="/content/announcements">公告管理</el-menu-item>
            <el-menu-item v-if="canAccess('/content/faq')" index="/content/faq">常见问题</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAccess('/service/customer-service')" index="service-group">
            <template #title>
              <el-icon>
                <Monitor />
              </el-icon>
              <span>客服</span>
            </template>
            <el-menu-item index="/service/customer-service">客服设置</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAccessSome(['/system/roles', '/system/admin-users', '/system/audit'])"
            index="system-group">
            <template #title>
              <el-icon>
                <Cpu />
              </el-icon>
              <span>系统</span>
            </template>
            <el-menu-item v-if="canAccess('/system/roles')" index="/system/roles">角色管理</el-menu-item>
            <el-menu-item v-if="canAccess('/system/admin-users')" index="/system/admin-users">Admin 账号</el-menu-item>
            <el-menu-item v-if="canAccess('/system/audit')" index="/system/audit">操作审计</el-menu-item>
          </el-sub-menu>
        </el-menu>

        <div class="aside-foot">
          <div class="aside-foot-muted">当前角色：<strong>{{ activeRole?.name ?? '—' }}</strong><br />菜单按 menuPaths 前缀过滤
          </div>
        </div>
      </el-aside>

      <el-container class="admin-body">
        <el-header class="admin-header" height="56px">
          <div class="header-title">{{ (route.meta.title as string) || '工作台' }}</div>
          <div class="header-actions">
            <el-tag v-if="activeRole" type="info" effect="plain" size="small">
              {{ activeRole.name }}
            </el-tag>
            <el-button type="primary" plain :icon="SwitchButton" @click="onLogout">
              退出
            </el-button>
          </div>
        </el-header>

        <el-main class="admin-main">
          <RouterView />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<style scoped>
.admin-shell {
  min-height: 100dvh;
  width: 100%;
  max-width: 100%;
  background: var(--admin-surface-bg);
}

.admin-shell-inner {
  min-height: 100dvh;
  width: 100%;
  max-width: 100%;
  flex-direction: row;
}

/** 右侧主栏：占满剩余宽度，避免 flex 子项默认 min-width:auto 把主区挤窄 */
.admin-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.admin-aside {
  width: var(--admin-aside-width) !important;
  flex-shrink: 0;
  background: linear-gradient(180deg, #ffffff 0%, #fafbfd 98%);
  display: flex;
  flex-direction: column;
  box-shadow: 4px 0 24px rgb(26 62 138 / 4%);
}

.admin-brand {
  padding: 1.25rem 1rem;
  font-family: var(--admin-font-display);
  font-weight: 700;
  font-size: 0.9375rem;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
}

.admin-el-menu {
  border-right: none;
  flex: 1;
  background: transparent;
}

.aside-foot {
  padding: 1rem;
  margin-top: auto;
}

.aside-foot-muted {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  padding: 0 var(--admin-main-pad-inline);
  background: #ffffffcc;
  backdrop-filter: blur(24px);
  border-bottom: 1px solid rgb(148 163 184 / 12%);
}

.header-title {
  font-weight: 600;
  font-size: 1rem;
  font-family: var(--admin-font-display);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.admin-main {
  flex: 1;
  min-height: 0;
  min-width: 0;
  padding: var(--admin-main-pad-block) var(--admin-main-pad-inline);
  box-sizing: border-box;
  overflow: auto;
}

@media (max-width: 768px) {
  .admin-aside {
    width: 72px !important;
  }

  .admin-brand,
  .aside-foot {
    display: none;
  }
}
</style>
