<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { logoutClient } from '@/api/auth'
import { fetchMemberProfile } from '@/api/member/profile'
import {
  fetchGuajiAuthStatus,
  fetchGuajiBalance,
  fetchPrimaryCurrency,
  setPrimaryCurrency,
  PRIMARY_CURRENCIES,
  type GuajiBalance,
  type PrimaryCurrency,
} from '@/api/guaji/accounts'
import { demoUser } from '@/demo/demoAccount'
import { confirmDialog } from '@/utils/confirmDialog'
import {
  guajiAmountsEqual,
  readGuajiBalanceCache,
  writeGuajiBalanceCache,
} from '@/utils/guajiBalanceCache'
import {
  fetchCustomerServiceAgents,
  normalizeTgHref,
  tgDisplayLabel,
  type CustomerServiceAgent,
} from '@/api/customerService'

/**
 * 会员中心首页；「卡片管理」跳转至独立银行卡 / 渠道页
 * 严格遵循 client/DESIGN.md「数字精算主义」：
 * - 渐变蓝色头部 + 白色卡片下沉
 * - 大模糊低不透明阴影，避免 1px 实线切分
 * - Material Symbols 图标 + Plus Jakarta Sans 标题
 */

interface FeatureItem {
  /** 用作 key */
  id: string
  /** Material Symbols Outlined 名称 */
  icon: string
  /** 主标题（中文） */
  label: string
  /** 可选副标题（一句话说明） */
  hint?: string
  /** 图标主色：默认主色蓝；可指定其它点缀色 */
  tone?: 'primary' | 'success' | 'amber' | 'magenta' | 'cyan' | 'indigo'
  /** 可选徽标：例如「NEW」「3」 */
  badge?: string
}

const user = ref({
  account: demoUser.account as string,
  memberId: 0,
  platform: demoUser.platform,
})

const AVATAR_IMG = '/images/lobby/avatar-user.png'

const stats = ref({
  betting: '0.00',
  pnl: '0.00',
  balance: '',
  currency: '¥',
})

const memberIdText = computed(() => {
  const id = user.value.memberId
  return id > 0 ? String(id) : '—'
})

const router = useRouter()

const accountFeatures: FeatureItem[] = [
  { id: 'auth', icon: 'verified_user', label: '授权账号', tone: 'indigo' },
]

// §4.5：钱包流水（fund-records）保留；帐变(ledger)/追号(chase) 下线
const systemFeatures: FeatureItem[] = [
  { id: 'bet', icon: 'list_alt', label: '投注纪录', tone: 'primary' },
  { id: 'scheme-pnl', icon: 'monitoring', label: '方案盈亏', tone: 'success' },
  { id: 'wallet', icon: 'receipt_long', label: '钱包流水', tone: 'cyan' },
  { id: 'faq', icon: 'help', label: '常见问题', tone: 'amber' },
  { id: 'notice', icon: 'campaign', label: '公告', tone: 'magenta' },
  { id: 'contact-service', icon: 'headset_mic', label: '联系客服', tone: 'magenta' },
  { id: 'feedback', icon: 'forum', label: '意见回馈', tone: 'cyan' },
]

const refreshing = ref(false)
const activeGuajiUsername = ref('')
const primaryCurrency = ref<PrimaryCurrency>('CNY')
const switchingCurrency = ref(false)
const currencyOptions = PRIMARY_CURRENCIES

/** 头部齿轮：账户设置 / 登出面板 */
const settingsOpen = ref(false)

const csDialogVisible = ref(false)
const csLoading = ref(false)
const csAgents = ref<CustomerServiceAgent[]>([])

async function openCustomerService(): Promise<void> {
  csDialogVisible.value = true
  csLoading.value = true
  try {
    csAgents.value = await fetchCustomerServiceAgents()
  } catch {
    csAgents.value = []
    ElMessage.error('加载客服信息失败')
  } finally {
    csLoading.value = false
  }
}

async function copyTgLink(link: string): Promise<void> {
  const href = normalizeTgHref(link)
  if (!href) return
  try {
    await navigator.clipboard.writeText(href)
    ElMessage.success('已复制 Telegram 链接')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

function openTgLink(link: string): void {
  const href = normalizeTgHref(link)
  if (!href) return
  window.open(href, '_blank', 'noopener,noreferrer')
}

const formattedBalance = computed(() => {
  const sym = stats.value.currency || '¥'
  if (!stats.value.balance) return `${sym} --`
  return `${sym} ${formatMoney(parseFloat(stats.value.balance))}`
})
const formattedBetting = computed(() => formatMoney(parseFloat(stats.value.betting)))
const pnlValue = computed(() => parseFloat(stats.value.pnl))
const formattedPnl = computed(() => {
  const v = pnlValue.value
  const abs = formatMoney(Math.abs(v))
  if (!Number.isFinite(v) || v === 0) return abs
  return v > 0 ? `+${abs}` : `-${abs}`
})
const pnlTone = computed<'flat' | 'up' | 'down'>(() => {
  const v = pnlValue.value
  if (!Number.isFinite(v) || v === 0) return 'flat'
  return v > 0 ? 'up' : 'down'
})

function formatMoney(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

/** 币种符号：CNY ¥ / USDT ₮（Tether 通用符）/ TRX Ŧ（无官方 Unicode 符号，取近似字符） */
function currencySymbol(code: string): string {
  switch (code) {
    case 'CNY':
      return '¥'
    case 'USDT':
      return '₮'
    case 'TRX':
      return 'Ŧ'
    default:
      return code
  }
}

function todo(label: string): void {
  ElMessage.info(`${label}：二级页面待对接`)
}

/** 先读本地缓存（按授权账号 + 币种），供切换账号时立即展示 */
function applyCachedBalance(username: string, currency: string): boolean {
  const cached = readGuajiBalanceCache(username, currency)
  if (!cached) return false
  stats.value = {
    ...stats.value,
    balance: String(cached.amount),
    currency: currencySymbol(cached.currency),
  }
  return true
}

/**
 * 用第三方余额更新展示：默认仅在与当前显示不同时重绘；成功后写入本地缓存。
 * @returns 是否更新了界面上的余额
 */
function syncBalanceFromRemote(bal: GuajiBalance, force = false): boolean {
  const username = bal.username ?? activeGuajiUsername.value
  const sym = currencySymbol(bal.currency)
  const prevAmount = parseFloat(stats.value.balance)
  const changed =
    force ||
    !stats.value.balance ||
    stats.value.currency !== sym ||
    !guajiAmountsEqual(prevAmount, bal.amount)

  if (changed) {
    stats.value = {
      ...stats.value,
      balance: String(bal.amount),
      currency: sym,
    }
  }

  if (username) {
    writeGuajiBalanceCache({
      username,
      currency: bal.currency,
      amount: bal.amount,
      updatedAt: Date.now(),
    })
    activeGuajiUsername.value = username
  }
  return changed
}

/** §20.1：顶栏余额展示第三方实账；手动刷新时强制更新 */
async function refreshBalance(): Promise<void> {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const bal = await fetchGuajiBalance()
    syncBalanceFromRemote(bal, true)
    ElMessage.success('余额已刷新')
  } catch {
    ElMessage.error('余额刷新失败')
  } finally {
    refreshing.value = false
  }
}

async function loadMemberCenter(): Promise<void> {
  try {
    const profile = await fetchMemberProfile()
    user.value = {
      ...user.value,
      account: profile.account,
      memberId: Number(profile.memberId) || 0,
    }
    // 余额仅来自第三方（下方 fetchGuajiBalance）；资料接口只提供币种兜底符号
    stats.value = {
      ...stats.value,
      currency: currencySymbol(profile.currency),
    }
  } catch {
    ElMessage.error('会员资料加载失败')
  }
  try {
    const status = await fetchGuajiAuthStatus()
    if (status.activeUsername) activeGuajiUsername.value = status.activeUsername
  } catch {
    // 第三方未启用时忽略，沿用本地展示
  }
  try {
    primaryCurrency.value = await fetchPrimaryCurrency()
  } catch {
    // 第三方未启用时默认 CNY
  }
  // 切换授权账号后：先展示该账号上次记录的余额，再静默拉第三方对比更新
  if (activeGuajiUsername.value) {
    applyCachedBalance(activeGuajiUsername.value, primaryCurrency.value)
  }
  try {
    const bal = await fetchGuajiBalance()
    syncBalanceFromRemote(bal)
  } catch {
    // 拉取失败时保留缓存值；无缓存则继续显示 --
  }
}

/** §4.4：切换主币种与切换授权同逻辑——弹窗确认后全部 running/pending 方案暂停 */
async function onSwitchCurrency(next: PrimaryCurrency): Promise<void> {
  if (next === primaryCurrency.value || switchingCurrency.value) return
  const ok = await confirmDialog({
    title: '切换主币种',
    message: `切换主币种为 ${next} 将停止全部挂机方案，确定继续？`,
    confirmText: '切换',
    cancelText: '取消',
  })
  if (!ok) return
  switchingCurrency.value = true
  try {
    primaryCurrency.value = await setPrimaryCurrency(next)
    ElMessage.success('主币种已切换；方案已暂停，请到云端中心开启')
    await refreshBalance()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '切换主币种失败')
  } finally {
    switchingCurrency.value = false
  }
}

onMounted(() => {
  void loadMemberCenter()
})

async function onLogout(): Promise<void> {
  settingsOpen.value = false
  const ok = await confirmDialog({
    title: '登出确认',
    message: '确认登出当前会话？再次登入需输入授权账号。',
    icon: 'logout',
    confirmText: '登出',
    cancelText: '取消',
  })
  if (!ok) return
  logoutClient()
  ElMessage.success('已退出登录')
  void router.replace({ name: 'login' })
}

function onPickFeature(item: FeatureItem): void {
  if (item.id === 'auth') {
    void router.push({ name: 'member-auth-list' })
    return
  }
  if (item.id === 'wallet') {
    void router.push({ name: 'member-fund-records' })
    return
  }
  if (item.id === 'bet') {
    void router.push({ name: 'member-bet-records' })
    return
  }
  if (item.id === 'scheme-pnl') {
    void router.push({ name: 'member-scheme-pnl' })
    return
  }
  if (item.id === 'notice') {
    void router.push({ name: 'member-announcements' })
    return
  }
  if (item.id === 'feedback') {
    void router.push({ name: 'member-feedback' })
    return
  }
  if (item.id === 'faq') {
    void router.push({ name: 'member-faq' })
    return
  }
  if (item.id === 'contact-service') {
    void openCustomerService()
    return
  }
  todo(item.label)
}

const personalFeaturesAll = computed(() => [...accountFeatures, ...systemFeatures])

const personalFeatureCount = computed(() => personalFeaturesAll.value.length)
</script>

<template>
  <div class="mc" data-page="member-center">
    <!-- ===== 头部：身份卡 + 当前下注 / 当前盈亏 ===== -->
    <header class="mc-head" role="banner">
      <div class="mc-head-deco" aria-hidden="true" />
      <div class="mc-head-inner">
        <div class="mc-id">
          <div class="mc-avatar-wrap">
            <img
              :src="AVATAR_IMG"
              alt="用户头像"
              width="72"
              height="72"
              class="mc-avatar-img"
              decoding="async"
            />
          </div>
          <div class="mc-id-meta">
            <h1 class="mc-id-name">{{ user.account }}（id：{{ memberIdText }}）</h1>
            <p class="mc-id-line">
              <span class="mc-id-key">授权账号</span>
              <span class="mc-id-val">{{ activeGuajiUsername || user.account }}</span>
            </p>
            <p class="mc-id-line">
              <span class="mc-id-key">授权平台</span>
              <span class="mc-id-val">{{ user.platform }}</span>
            </p>
          </div>
          <el-popover v-model:visible="settingsOpen" placement="bottom-end" :width="288" trigger="click"
            popper-class="mc-settings-popper" :show-arrow="false">
            <template #reference>
              <button type="button" class="mc-id-edit" aria-label="账户设置与登出" :aria-expanded="settingsOpen"
                aria-haspopup="dialog">
                <span class="mc-ms" aria-hidden="true">settings</span>
              </button>
            </template>
            <div class="mc-settings-panel">
              <p class="mc-settings-title">账户</p>
              <el-button type="primary" size="large" round class="mc-logout" @click="onLogout">
                <span class="mc-ms mc-logout-ico" aria-hidden="true">logout</span>
                <span>登&nbsp;&nbsp;出</span>
              </el-button>
              <p class="mc-foot-note mc-foot-note--in-popover">
                本次登入会话由「{{ user.platform }}」授权 · 安全离线
              </p>
            </div>
          </el-popover>
        </div>

        <div class="mc-kpi">
          <div class="mc-kpi-col">
            <div class="mc-kpi-val">{{ formattedBetting }}</div>
            <div class="mc-kpi-lbl">当前下注</div>
          </div>
          <div class="mc-kpi-divider" aria-hidden="true" />
          <div class="mc-kpi-col">
            <div class="mc-kpi-val" :class="{ 'is-up': pnlTone === 'up', 'is-down': pnlTone === 'down' }">
              {{ formattedPnl }}
            </div>
            <div class="mc-kpi-lbl">当前盈亏</div>
          </div>
        </div>
      </div>
    </header>

    <main class="mc-main">
      <!-- ===== 余额卡：充值 / 提现 / 客服 + 个人 / 团队 ===== -->
      <section class="mc-card mc-balance">
        <div class="mc-balance-head">
          <div class="mc-balance-meta">
            <span class="mc-balance-lbl">帐户余额</span>
            <span class="mc-balance-amt">{{ formattedBalance }}</span>
          </div>
          <button type="button" class="mc-refresh" :class="{ 'is-loading': refreshing }" aria-label="刷新余额"
            @click="refreshBalance">
            <span class="mc-ms" aria-hidden="true">refresh</span>
          </button>
        </div>

        <div class="mc-currency" role="group" aria-label="主币种切换">
          <span class="mc-currency-lbl">主币种</span>
          <div class="mc-currency-seg">
            <button v-for="c in currencyOptions" :key="c" type="button" class="mc-currency-opt"
              :class="{ 'is-active': primaryCurrency === c }" :disabled="switchingCurrency"
              @click="onSwitchCurrency(c)">
              {{ c }}
            </button>
          </div>
        </div>

      </section>

      <!-- ===== 个人中心：帐户 + 系统功能合并展示 ===== -->
      <section class="mc-card mc-group" role="tabpanel" aria-label="个人中心功能">
        <header class="mc-group-head">
          <h2 class="mc-group-title">个人中心</h2>
          <span class="mc-group-meta">{{ personalFeatureCount }} 项</span>
        </header>
        <ul class="mc-grid mc-grid--3" role="list">
          <li v-for="it in personalFeaturesAll" :key="it.id">
            <button type="button" class="mc-cell" :data-tone="it.tone || 'primary'" @click="onPickFeature(it)">
              <span class="mc-cell-ico" aria-hidden="true">
                <span class="mc-ms">{{ it.icon }}</span>
              </span>
              <span class="mc-cell-lbl">{{ it.label }}</span>
              <span v-if="it.hint" class="mc-cell-hint">{{ it.hint }}</span>
              <span v-if="it.badge" class="mc-cell-badge">{{ it.badge }}</span>
            </button>
          </li>
        </ul>
      </section>
    </main>

    <el-dialog
      v-model="csDialogVisible"
      title="联系客服"
      width="min(92vw, 24rem)"
      class="mc-cs-dialog"
      destroy-on-close
      append-to-body
    >
      <el-skeleton v-if="csLoading" animated :rows="3" />
      <el-empty v-else-if="!csAgents.length" description="暂未配置客服，请稍后再试" />
      <ul v-else class="mc-cs-list" role="list">
        <li v-for="agent in csAgents" :key="agent.id" class="mc-cs-item">
          <div class="mc-cs-name">{{ agent.name }}</div>
          <div class="mc-cs-row">
            <span class="mc-cs-key">Telegram</span>
            <button type="button" class="mc-cs-tg" @click="openTgLink(agent.tgLink)">
              {{ tgDisplayLabel(agent.tgLink) }}
            </button>
            <button type="button" class="mc-cs-copy" aria-label="复制 Telegram 链接" @click="copyTgLink(agent.tgLink)">
              <span class="mc-ms" aria-hidden="true">content_copy</span>
            </button>
          </div>
          <div v-if="agent.workHours" class="mc-cs-row">
            <span class="mc-cs-key">上班时间</span>
            <span class="mc-cs-val">{{ agent.workHours }}</span>
          </div>
        </li>
      </ul>
    </el-dialog>
  </div>
</template>

<style scoped>
.mc {
  --mc-primary: #0050cb;
  --mc-primary-strong: #0066ff;
  --mc-primary-soft: rgba(0, 102, 255, 0.08);
  --mc-surface: #f7f9fb;
  --mc-card: #ffffff;
  --mc-container: #f1f5f9;
  --mc-variant: #f8fafc;
  --mc-on: #191c1e;
  --mc-on-var: #424656;
  --mc-on-mute: #727687;
  --mc-success: #1f9d63;
  --mc-error: #ba1a1a;
  --mc-cyan: #0aa6c4;
  --mc-amber: #d97706;
  --mc-magenta: #c63a8b;
  --mc-indigo: #4f46e5;
  --mc-outline: rgba(226, 232, 240, 0.85);
  min-height: 100dvh;
  background: var(--mc-surface);
  color: var(--mc-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: calc(5.5rem + env(safe-area-inset-bottom));
  -webkit-font-smoothing: antialiased;
}

.mc-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.375rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

/* =====================================================
   Header（渐变蓝 + 圆角下沉，与云端中心一致的视觉语言）
   ===================================================== */
.mc-head {
  position: relative;
  background: linear-gradient(180deg, var(--mc-primary-strong) 0%, var(--mc-primary) 100%);
  color: #fff;
  padding: max(1.75rem, env(safe-area-inset-top)) 1.25rem 5rem;
  border-radius: 0 0 2rem 2rem;
  box-shadow: 0 20px 40px -24px rgba(0, 80, 203, 0.45);
  overflow: hidden;
}

/* 装饰：右上角弥散光斑（数字精算主义「无线化」分层语言） */
.mc-head-deco {
  position: absolute;
  inset: -30% -20% auto auto;
  width: 18rem;
  height: 18rem;
  background: radial-gradient(closest-side, rgba(255, 255, 255, 0.18), rgba(255, 255, 255, 0) 70%);
  pointer-events: none;
}

.mc-head-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* ---- 身份卡 ---- */
.mc-id {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 0.95rem;
}

.mc-avatar-wrap {
  width: 4.5rem;
  height: 4.5rem;
  border-radius: 999px;
  background: var(--c-surface-c-high, #eef2f7);
  overflow: hidden;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.4),
    0 8px 22px -10px rgba(0, 0, 0, 0.4);
}

.mc-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mc-id-meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.mc-id-name {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 800;
  letter-spacing: -0.01em;
  font-size: 1.25rem;
  line-height: 1.2;
}

.mc-id-line {
  margin: 0;
  display: flex;
  align-items: baseline;
  gap: 0.45rem;
  font-size: 0.75rem;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.92);
}

.mc-id-key {
  font-size: 0.6875rem;
  color: rgba(255, 255, 255, 0.7);
  letter-spacing: 0.02em;
  flex-shrink: 0;
}

.mc-id-val {
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mc-id-edit {
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.75rem;
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  align-self: flex-start;
  margin-top: 0.15rem;
}

.mc-id-edit:hover {
  background: rgba(255, 255, 255, 0.22);
}

.mc-id-edit .mc-ms {
  font-size: 1.25rem;
}

/* ---- KPI（当前下注 / 当前盈亏） ---- */
.mc-kpi {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: stretch;
  margin-top: 0.25rem;
}

.mc-kpi-col {
  text-align: center;
  padding: 0 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.mc-kpi-val {
  font-family: 'Plus Jakarta Sans', 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
  line-height: 1.05;
}

.mc-kpi-val.is-up {
  color: #b9f5d0;
}

.mc-kpi-val.is-down {
  color: #ffd7d1;
}

.mc-kpi-lbl {
  font-size: 0.75rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.82);
  letter-spacing: 0.02em;
}

.mc-kpi-divider {
  width: 1px;
  align-self: stretch;
  min-height: 3.25rem;
  background: linear-gradient(180deg,
      transparent,
      rgba(255, 255, 255, 0.22) 15%,
      rgba(255, 255, 255, 0.22) 85%,
      transparent);
}

/* =====================================================
   Main 容器
   ===================================================== */
.mc-main {
  max-width: 40rem;
  margin: -3rem auto 0;
  padding: 0 1.15rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
  position: relative;
}

.mc-card {
  background: var(--mc-card);
  border-radius: 1.25rem;
  padding: 1.25rem;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

/* =====================================================
   余额卡
   ===================================================== */
.mc-balance {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
}

.mc-balance-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.mc-balance-meta {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
  min-width: 0;
}

.mc-balance-lbl {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--mc-on-var);
  letter-spacing: 0.01em;
}

.mc-balance-amt {
  font-family: 'Plus Jakarta Sans', 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.35rem;
  font-weight: 800;
  color: var(--mc-primary);
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.mc-refresh {
  width: 2.1rem;
  height: 2.1rem;
  border-radius: 999px;
  background: var(--mc-primary-soft);
  color: var(--mc-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.mc-refresh:hover {
  background: rgba(0, 102, 255, 0.14);
}

.mc-refresh .mc-ms {
  font-size: 1.125rem;
  transition: transform 0.45s ease;
}

.mc-refresh.is-loading .mc-ms {
  animation: mc-spin 0.8s linear infinite;
}

@keyframes mc-spin {
  to {
    transform: rotate(360deg);
  }
}

.mc-currency {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.mc-currency-lbl {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--mc-on-var);
}

.mc-currency-seg {
  display: inline-flex;
  padding: 0.2rem;
  border-radius: 999px;
  background: var(--mc-container);
  gap: 0.15rem;
}

.mc-currency-opt {
  min-width: 3.4rem;
  padding: 0.4rem 0.75rem;
  border-radius: 999px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--mc-on-mute);
  font-variant-numeric: tabular-nums;
  transition: background 0.15s, color 0.15s;
}

.mc-currency-opt.is-active {
  background: var(--mc-card);
  color: var(--mc-primary);
  box-shadow: 0 4px 12px -6px rgba(0, 80, 203, 0.35);
}

.mc-currency-opt:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.mc-balance-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding-top: 0.9rem;
  border-top: 1px solid var(--mc-outline);
}

.mc-balance-hint {
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--mc-on-mute);
}

/* 快捷三项：充值 / 提现 / 客服 —— 不用 1px 分割线，用浅底+悬停色阶 */
.mc-quick {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.6rem;
  background: var(--mc-variant);
  border-radius: 1rem;
  padding: 0.85rem 0.5rem;
}

.mc-quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.25rem;
  border-radius: 0.85rem;
  background: transparent;
  color: var(--mc-on);
  transition: background 0.15s, transform 0.15s;
}

.mc-quick-item:hover {
  background: rgba(255, 255, 255, 0.85);
  transform: translateY(-1px);
}

.mc-quick-item:active {
  transform: scale(0.97);
}

.mc-quick-ico {
  width: 2.85rem;
  height: 2.85rem;
  border-radius: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: linear-gradient(180deg, var(--mc-primary-strong) 0%, var(--mc-primary) 100%);
  box-shadow: 0 10px 20px -10px rgba(0, 80, 203, 0.55);
}

.mc-quick-ico .mc-ms {
  font-size: 1.45rem;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 24;
}

.mc-quick-item[data-tone='cyan'] .mc-quick-ico {
  background: linear-gradient(180deg, #22b3cf 0%, #0aa6c4 100%);
  box-shadow: 0 10px 20px -10px rgba(10, 166, 196, 0.55);
}

.mc-quick-item[data-tone='magenta'] .mc-quick-ico {
  background: linear-gradient(180deg, #e35aa6 0%, #c63a8b 100%);
  box-shadow: 0 10px 20px -10px rgba(198, 58, 139, 0.5);
}

.mc-quick-lbl {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--mc-on);
  letter-spacing: 0.01em;
}

/* 个人中心 / 团队中心 — 轨道式双态（选中 = 主色渐变填充） */
.mc-balance-cta {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.65rem;
}

.mc-cta-toggle {
  font: inherit;
  font-size: 0.875rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  padding: 0.75rem 0.6rem;
  margin: 0;
  width: 100%;
  border-radius: 999px;
  cursor: pointer;
  border: 1px solid rgba(0, 80, 203, 0.28);
  background: var(--mc-card);
  color: var(--mc-primary);
  box-shadow: 0 4px 14px -8px rgba(15, 23, 42, 0.12);
  transition:
    background 0.15s,
    color 0.15s,
    border-color 0.15s,
    box-shadow 0.15s;
}

.mc-cta-toggle:hover:not(.is-active) {
  background: rgba(0, 102, 255, 0.06);
  border-color: rgba(0, 80, 203, 0.38);
}

.mc-cta-toggle.is-active {
  border-color: transparent;
  color: #fff;
  background: linear-gradient(180deg, var(--mc-primary-strong) 0%, var(--mc-primary) 100%);
  box-shadow: 0 12px 26px -12px rgba(0, 80, 203, 0.5);
}

.mc-cta-toggle:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.45);
  outline-offset: 2px;
}

/* =====================================================
   功能分组（个人 / 团队）
   ===================================================== */
.mc-group {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.mc-group-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0 0.15rem;
}

.mc-group-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  position: relative;
  padding-left: 0.85rem;
}

/* 标题左侧的小立柱：替代「分割线」的层级语言 */
.mc-group-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 0.25rem;
  height: 1rem;
  border-radius: 999px;
  background: linear-gradient(180deg, var(--mc-primary-strong), var(--mc-primary));
}

.mc-group-meta {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--mc-on-mute);
  letter-spacing: 0.02em;
}

.mc-grid {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.6rem;
}

.mc-grid--2 {
  grid-template-columns: repeat(2, 1fr);
}

.mc-grid--3 {
  grid-template-columns: repeat(3, 1fr);
}

@media (min-width: 480px) {
  .mc-grid--3 {
    grid-template-columns: repeat(4, 1fr);
  }
}

/* 功能单元：用色阶分层，不使用 1px 实线 */
.mc-cell {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 0.95rem 0.5rem 0.85rem;
  border-radius: 1rem;
  background: var(--mc-variant);
  color: var(--mc-on);
  text-align: center;
  transition:
    background 0.15s,
    transform 0.15s,
    box-shadow 0.2s;
}

.mc-cell:hover {
  background: #eef3f9;
  transform: translateY(-1px);
  box-shadow: 0 10px 24px -18px rgba(15, 23, 42, 0.35);
}

.mc-cell:active {
  transform: scale(0.97);
}

.mc-cell-ico {
  width: 2.65rem;
  height: 2.65rem;
  border-radius: 0.85rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.95), rgba(255, 255, 255, 0.65));
  box-shadow:
    inset 0 0 0 1px rgba(0, 80, 203, 0.06),
    0 6px 14px -10px rgba(0, 80, 203, 0.35);
  color: var(--mc-primary);
}

.mc-cell-ico .mc-ms {
  font-size: 1.5rem;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
}

.mc-cell[data-tone='success'] .mc-cell-ico {
  color: var(--mc-success);
  box-shadow:
    inset 0 0 0 1px rgba(31, 157, 99, 0.08),
    0 6px 14px -10px rgba(31, 157, 99, 0.35);
}

.mc-cell[data-tone='amber'] .mc-cell-ico {
  color: var(--mc-amber);
  box-shadow:
    inset 0 0 0 1px rgba(217, 119, 6, 0.08),
    0 6px 14px -10px rgba(217, 119, 6, 0.35);
}

.mc-cell[data-tone='magenta'] .mc-cell-ico {
  color: var(--mc-magenta);
  box-shadow:
    inset 0 0 0 1px rgba(198, 58, 139, 0.08),
    0 6px 14px -10px rgba(198, 58, 139, 0.35);
}

.mc-cell[data-tone='cyan'] .mc-cell-ico {
  color: var(--mc-cyan);
  box-shadow:
    inset 0 0 0 1px rgba(10, 166, 196, 0.08),
    0 6px 14px -10px rgba(10, 166, 196, 0.35);
}

.mc-cell[data-tone='indigo'] .mc-cell-ico {
  color: var(--mc-indigo);
  box-shadow:
    inset 0 0 0 1px rgba(79, 70, 229, 0.08),
    0 6px 14px -10px rgba(79, 70, 229, 0.35);
}

.mc-cell-lbl {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--mc-on);
  letter-spacing: 0.01em;
  line-height: 1.2;
}

.mc-cell-hint {
  font-size: 0.625rem;
  color: var(--mc-on-mute);
  line-height: 1.4;
  letter-spacing: 0.02em;
}

.mc-cell-badge {
  position: absolute;
  top: 0.45rem;
  right: 0.45rem;
  padding: 0.05rem 0.4rem;
  font-size: 0.5625rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  color: #fff;
  background: linear-gradient(180deg, #ff5b8a, #c63a8b);
  border-radius: 999px;
  line-height: 1.4;
}

/* =====================================================
   设置面板内 · 登出
   ===================================================== */
.mc-settings-panel {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.15rem 0.05rem 0;
}

.mc-settings-title {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  color: var(--mc-on-mute);
  text-transform: uppercase;
}

.mc-logout {
  width: 100%;
  max-width: none;
  margin: 0;
  height: 44px;
  font-size: 0.9375rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  /* 对齐「数字精算主义」：使用全局主色，去重渐变；阴影低不透明度、大模糊 */
  box-shadow: 0 8px 20px -10px rgba(0, 80, 203, 0.28);
}

.mc-logout:hover {
  box-shadow: 0 10px 24px -10px rgba(0, 80, 203, 0.36);
}

.mc-logout :deep(.el-icon) {
  margin-right: 0;
}

.mc-logout .mc-logout-ico {
  margin-right: 0.45rem;
  font-size: 1.1rem;
  vertical-align: middle;
}

.mc-foot-note {
  margin: 0;
  font-size: 0.6875rem;
  color: var(--mc-on-mute);
  letter-spacing: 0.02em;
  text-align: center;
}

.mc-foot-note--in-popover {
  text-align: left;
  line-height: 1.45;
}

.mc-cs-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.mc-cs-item {
  padding: 0.85rem 1rem;
  border-radius: 0.85rem;
  background: #f7f9fb;
}

.mc-cs-name {
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', sans-serif;
  font-weight: 600;
  font-size: 1rem;
  color: #191c1e;
  margin-bottom: 0.5rem;
}

.mc-cs-row {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.875rem;
  line-height: 1.5;
}

.mc-cs-row + .mc-cs-row {
  margin-top: 0.35rem;
}

.mc-cs-key {
  flex: 0 0 4.5rem;
  color: #727687;
  font-size: 0.8125rem;
}

.mc-cs-val {
  color: #424656;
}

.mc-cs-tg {
  flex: 1;
  min-width: 0;
  padding: 0;
  border: none;
  background: none;
  color: #0066ff;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mc-cs-copy {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border: none;
  border-radius: 0.5rem;
  background: rgba(0, 102, 255, 0.08);
  color: #0066ff;
  cursor: pointer;
}

.mc-cs-copy .mc-ms {
  font-size: 1.1rem;
}
</style>

<!-- popper 挂载到 body，需非 scoped -->
<style>
.mc-settings-popper.el-popover.el-popper {
  padding: 1rem 1.1rem;
  border-radius: 1rem;
  border: none;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.2),
    0 8px 24px -12px rgba(15, 23, 42, 0.12);
}

.mc-cs-dialog.el-dialog {
  max-height: 50dvh;
  display: flex;
  flex-direction: column;
}

.mc-cs-dialog .el-dialog__header {
  flex-shrink: 0;
}

.mc-cs-dialog .el-dialog__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.mc-cs-dialog .el-dialog__body::-webkit-scrollbar {
  display: none;
}
</style>
