<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

/** 与方案配置等页一致的返回图标占位，可替换为专用资源 */
const BACK_ICO = '/images/lobby/icon-placeholder.png'

type BetTab = 'real' | 'sim'
type RecordStatus = 'hit' | 'miss'

interface BetRecordRow {
  period: string
  playType: string
  multiplier: string
  round: string
  amount: string
  pnl: string
  pnlPositive: boolean
  status: RecordStatus
}

/** 真实记录：演示数据（与 Stitch 原型一致，接口对接后替换） */
const MOCK_REAL: BetRecordRow[] = [
  { period: '20240310031', playType: '万位定位', multiplier: '1.5', round: '1/3', amount: '10.00', pnl: '+5.00', pnlPositive: true, status: 'hit' },
  { period: '20240310030', playType: '千位定位', multiplier: '3.0', round: '2/3', amount: '20.00', pnl: '-20.00', pnlPositive: false, status: 'miss' },
  { period: '20220523029', playType: '个位定位', multiplier: '1.5', round: '1/5', amount: '5.00', pnl: '+2.50', pnlPositive: true, status: 'hit' },
  { period: '20240310028', playType: '百位定位', multiplier: '2.2', round: '3/3', amount: '50.00', pnl: '+60.00', pnlPositive: true, status: 'hit' },
  { period: '20240310027', playType: '万位定位', multiplier: '1.0', round: '1/3', amount: '100.00', pnl: '-100.00', pnlPositive: false, status: 'miss' },
  { period: '20240310026', playType: '十位定位', multiplier: '1.5', round: '1/3', amount: '10.00', pnl: '+5.00', pnlPositive: true, status: 'hit' },
  { period: '20240310025', playType: '万位定位', multiplier: '1.5', round: '2/3', amount: '30.00', pnl: '+15.00', pnlPositive: true, status: 'hit' },
  { period: '20240310024', playType: '个位定位', multiplier: '1.5', round: '1/3', amount: '10.00', pnl: '+5.00', pnlPositive: true, status: 'hit' },
]

/** 模拟记录：空列表演示「暂无数据」（截图空态）；有数据时可与真实同源 */
const MOCK_SIM: BetRecordRow[] = []

const router = useRouter()
const activeTab = ref<BetTab>('real')
const loading = ref(false)

const realRows = ref<BetRecordRow[]>([...MOCK_REAL])
const simRows = ref<BetRecordRow[]>([...MOCK_SIM])

const displayRows = computed(() => (activeTab.value === 'real' ? realRows.value : simRows.value))

function parseAmount(s: string): number {
  const n = parseFloat(String(s).replace(/[^\d.-]/g, ''))
  return Number.isFinite(n) ? n : 0
}

const summary = computed(() => {
  const rows = displayRows.value
  if (!rows.length) {
    return { totalBet: 0, dayPnl: 0, winRate: 0 }
  }
  let totalBet = 0
  let dayPnl = 0
  let hits = 0
  for (const r of rows) {
    totalBet += parseAmount(r.amount)
    dayPnl += parseAmount(r.pnl)
    if (r.status === 'hit') hits += 1
  }
  const winRate = (hits / rows.length) * 100
  return { totalBet, dayPnl, winRate }
})

function formatMoney(n: number, signed = false): string {
  const abs = Math.abs(n).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  if (!signed) return `¥${abs}`
  if (n > 0) return `+¥${abs}`
  if (n < 0) return `-¥${abs}`
  return `¥${abs}`
}

function goBack() {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'cloud' })
}

function onRefresh() {
  loading.value = true
  window.setTimeout(() => {
    realRows.value = [...MOCK_REAL]
    simRows.value = [...MOCK_SIM]
    loading.value = false
    ElMessage.success('已刷新')
  }, 280)
}
</script>

<template>
  <div class="br" data-page="bet-records">
    <header class="br-head" role="banner">
      <button type="button" class="br-back-btn br-back" aria-label="返回" @click="goBack">
        <img :src="BACK_ICO" alt="" width="24" height="24" class="br-back-img" decoding="async" />
      </button>
      <h1 class="br-title">最近三日投注记录</h1>
      <button
        type="button"
        class="br-icon-btn br-refresh"
        aria-label="刷新"
        :disabled="loading"
        @click="onRefresh"
      >
        <span class="br-ms" aria-hidden="true">refresh</span>
      </button>
    </header>

    <main class="br-main">
      <section class="br-tabs-wrap" aria-label="记录类型">
        <div class="br-tabs">
          <button
            type="button"
            class="br-tab"
            :class="{ 'br-tab--active': activeTab === 'real' }"
            @click="activeTab = 'real'"
          >
            真实记录
          </button>
          <button
            type="button"
            class="br-tab"
            :class="{ 'br-tab--active': activeTab === 'sim' }"
            @click="activeTab = 'sim'"
          >
            模拟记录
          </button>
        </div>
      </section>

      <section class="br-summary-wrap">
        <div class="br-summary">
          <div class="br-sum-cell">
            <span class="br-sum-lbl">总投注额</span>
            <span class="br-sum-val">{{ formatMoney(summary.totalBet) }}</span>
          </div>
          <div class="br-sum-divider" aria-hidden="true" />
          <div class="br-sum-cell">
            <span class="br-sum-lbl">当日盈亏</span>
            <div class="br-sum-pnl">
              <span
                class="br-sum-val br-sum-val--pnl"
                :class="{
                  'is-pos': summary.dayPnl > 0,
                  'is-neg': summary.dayPnl < 0,
                  'is-zero': summary.dayPnl === 0,
                }"
              >
                {{ formatMoney(summary.dayPnl, true) }}
              </span>
              <span
                v-if="summary.dayPnl > 0"
                class="br-ms br-sum-trend"
                aria-hidden="true"
              >trending_up</span>
            </div>
          </div>
          <div class="br-sum-divider" aria-hidden="true" />
          <div class="br-sum-cell br-sum-cell--end">
            <span class="br-sum-lbl">胜率</span>
            <span class="br-sum-val">{{ displayRows.length ? `${summary.winRate.toFixed(1)}%` : '—' }}</span>
          </div>
        </div>
      </section>

      <section class="br-table-sec">
        <div class="br-table-card">
          <el-table
            :data="displayRows"
            class="br-el-table"
            size="small"
            stripe
            fit
            empty-text="暂无数据"
            :style="{ width: '100%' }"
          >
            <el-table-column
              prop="period"
              label="期数"
              :min-width="88"
              align="center"
              class-name="br-cell-period"
              label-class-name="br-head-period"
            >
              <template #default="{ row }">
                <span class="br-td-muted">{{ row.period }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="playType" label="玩法" :min-width="48" />
            <el-table-column prop="multiplier" label="倍数" :min-width="34" align="center" />
            <el-table-column prop="round" label="轮次" :min-width="34" align="center" />
            <el-table-column prop="amount" label="金额" :min-width="36" align="right">
              <template #default="{ row }">
                <span class="br-td-num">{{ row.amount }}</span>
              </template>
            </el-table-column>
            <el-table-column label="盈亏" :min-width="40" align="right">
              <template #default="{ row }">
                <span
                  class="br-td-pl"
                  :class="{
                    'br-td-pl--gain': row.pnlPositive,
                    'br-td-pl--loss': !row.pnlPositive,
                  }"
                >
                  {{ row.pnl }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="状态" :min-width="40" align="center">
              <template #default="{ row }">
                <el-tag
                  :type="row.status === 'hit' ? 'primary' : 'danger'"
                  effect="light"
                  size="small"
                  class="br-status-tag"
                >
                  {{ row.status === 'hit' ? '中' : '挂' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.br {
  --br-primary: #0050cb;
  --br-primary-strong: #0066ff;
  --br-surface: #f7f9fb;
  --br-surface-low: #f2f4f6;
  --br-on: #191c1e;
  --br-on-var: #424656;
  --br-outline: #c2c6d8;
  --br-error: #ba1a1a;
  min-height: 100dvh;
  background: var(--br-surface-low);
  color: var(--br-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  display: flex;
  flex-direction: column;
  padding-bottom: env(safe-area-inset-bottom);
}

.br-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.5rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.br-head {
  position: sticky;
  top: 0;
  z-index: 50;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 0.5rem;
  padding: max(0.75rem, env(safe-area-inset-top)) 0.75rem 0.875rem;
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(28px);
  -webkit-backdrop-filter: blur(28px);
  box-shadow: 0 8px 32px rgba(25, 28, 30, 0.06);
}

.br-back {
  justify-self: start;
}

.br-refresh {
  justify-self: end;
}

.br-back-btn {
  width: 2.25rem;
  height: 2.25rem;
  padding: 0;
  border: none;
  border-radius: 0.75rem;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
  flex-shrink: 0;
  transition:
    background 0.2s,
    transform 0.2s;
}

.br-back-btn:hover {
  background: rgba(0, 80, 203, 0.06);
}

.br-back-btn:active {
  transform: scale(0.94);
}

.br-back-btn:focus-visible {
  outline: 2px solid var(--br-primary-strong);
  outline-offset: 2px;
}

.br-back-img {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
  display: block;
  pointer-events: none;
}

.br-icon-btn {
  width: 2.25rem;
  height: 2.25rem;
  border: none;
  border-radius: 0.75rem;
  background: transparent;
  color: var(--br-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition:
    background 0.2s,
    transform 0.2s;
  flex-shrink: 0;
}

.br-icon-btn:hover:not(:disabled) {
  background: rgba(0, 80, 203, 0.06);
}

.br-icon-btn:active:not(:disabled) {
  transform: scale(0.94);
}

.br-icon-btn:focus-visible {
  outline: 2px solid var(--br-primary-strong);
  outline-offset: 2px;
}

.br-icon-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.br-title {
  margin: 0;
  justify-self: center;
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: clamp(0.9375rem, 3.4vw, 1.0625rem);
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--br-on);
  line-height: 1.25;
  min-width: 0;
  padding: 0 0.25rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.br-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.br-tabs-wrap {
  padding: 1rem;
  background: var(--br-surface-low);
}

.br-tabs {
  display: flex;
  align-items: center;
  padding: 0.25rem;
  background: #fff;
  border-radius: 999px;
  border: 1px solid rgba(194, 198, 216, 0.45);
  box-shadow: 0 2px 12px rgba(15, 23, 42, 0.06);
  gap: 0.125rem;
}

.br-tab {
  flex: 1;
  border: none;
  border-radius: 999px;
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  background: transparent;
  color: var(--br-on-var);
  transition:
    color 0.2s,
    background 0.2s,
    box-shadow 0.2s;
}

.br-tab:hover:not(.br-tab--active) {
  color: var(--br-primary-strong);
}

.br-tab--active {
  color: #fff;
  font-weight: 800;
  background: linear-gradient(180deg, #4d8dff 0%, var(--br-primary-strong) 100%);
  box-shadow: 0 4px 14px rgba(0, 80, 203, 0.22);
}

.br-summary-wrap {
  padding: 0 1rem 1rem;
}

.br-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  background: #fff;
  border-radius: 0.75rem;
  padding: 1rem;
  border: 1px solid rgba(194, 198, 216, 0.35);
  box-shadow: 0 4px 20px rgba(15, 23, 42, 0.06);
}

.br-sum-cell {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.br-sum-cell--end {
  align-items: flex-end;
  text-align: right;
}

.br-sum-lbl {
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: rgba(66, 70, 86, 0.55);
}

.br-sum-val {
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--br-on);
}

.br-sum-val--pnl.is-pos {
  color: var(--br-primary-strong);
}

.br-sum-val--pnl.is-neg {
  color: var(--br-error);
}

.br-sum-val--pnl.is-zero {
  color: var(--br-on-var);
}

.br-sum-pnl {
  display: flex;
  align-items: center;
  gap: 0.2rem;
}

.br-sum-trend {
  font-size: 1rem;
  color: var(--br-primary-strong);
}

.br-sum-divider {
  width: 1px;
  height: 2rem;
  flex-shrink: 0;
  background: rgba(194, 198, 216, 0.45);
}

.br-table-sec {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--br-surface-low);
  min-height: 12rem;
  padding: 0 1rem 1.5rem;
}

.br-table-card {
  background: #fff;
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.35);
  box-shadow: 0 4px 20px rgba(15, 23, 42, 0.06);
  overflow: hidden;
}

/* 与 GameDetailView「detail-bet-table」一致：透明纵线、min-width 列、单元格换行、无横滚（见 el-table-layout.css） */
.br-el-table :deep(.el-table) {
  --el-table-border-color: transparent;
  --el-table-bg-color: transparent;
  --el-table-header-bg-color: #f8fafc;
}

.br-el-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.br-el-table :deep(.el-table__header th) {
  font-size: 10px;
  font-weight: 700;
  color: #64748b !important;
  letter-spacing: 0.02em;
}

.br-el-table :deep(.el-table__header th .cell) {
  line-height: 1.35;
}

.br-el-table :deep(.el-table__body .br-cell-period .cell),
.br-el-table :deep(.el-table__header .br-head-period .cell) {
  padding-left: 6px;
  padding-right: 6px;
  line-height: 1.4;
}

.br-el-table :deep(.br-cell-period .cell) {
  word-break: break-all;
  overflow-wrap: anywhere;
}

.br-el-table :deep(.el-table__body .el-table__cell) {
  font-size: 11px;
  padding: 10px 4px;
}

.br-td-muted {
  color: var(--br-on-var);
  font-weight: 500;
}

.br-td-num {
  font-weight: 700;
  color: var(--br-on);
  font-variant-numeric: tabular-nums;
}

.br-td-pl {
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.br-td-pl--gain {
  color: var(--br-primary-strong);
}

.br-td-pl--loss {
  color: var(--br-error);
}

.br-status-tag {
  font-weight: 700;
  border: none;
}

.br-el-table :deep(.el-table__empty-text) {
  font-size: 0.875rem;
  color: rgba(66, 70, 86, 0.45);
  padding: 2.5rem 1rem;
}
</style>
