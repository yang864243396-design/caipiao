<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import DateRangePickerField from '@/components/ui/DateRangePickerField.vue'
import {
  fetchBetOrders,
  formatBetAmount,
  formatBetPnl,
  toBetDisplayRow,
  type BetCurrencySummary,
} from '@/api/orders/bets'
import { fetchMemberLotteryFilterOptions, fetchPublicLotteries } from '@/api/games/lotteries'
import { fetchRunningSchemes } from '@/api/cloud/center'
import { PRIMARY_CURRENCIES, type PrimaryCurrency } from '@/api/guaji/accounts'
import { currencySymbol } from '@/utils/currencyDisplay'
import { buildLotteryNameMap, lotteryFilterLabel } from '@/utils/lotteryDisplayName'

/** 会员中心 · 投注记录（对接 route） */

interface CloudSchemeOption {
  definitionId: string
  lotteryCode: string
  schemeName: string
}

interface SelectOption {
  value: string
  label: string
}

type BetCurrencyFilter = 'all' | PrimaryCurrency

const MAX_QUERY_DAYS = 3
const PAGE_SIZE = 20
const ALL_SCHEMES = 'all'
const ALL_LOTTERIES = 'all'

function ymd(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function todayRange(): [string, string] {
  const s = ymd(new Date())
  return [s, s]
}

/** 近 n 天（含今天），用于注单号检索扩窗 */
function lastNDaysRange(n: number): [string, string] {
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - Math.max(0, n - 1))
  return [ymd(start), ymd(end)]
}

function rangeSpanDays(from: string, to: string): number {
  const a = new Date(from.replace(/-/g, '/'))
  const b = new Date(to.replace(/-/g, '/'))
  const lo = Math.min(a.getTime(), b.getTime())
  const hi = Math.max(a.getTime(), b.getTime())
  return Math.round((hi - lo) / 86_400_000) + 1
}

interface BetRow {
  time: string
  game: string
  orderId: string
  amount: string
  returnAmount: string
  status: string
}

const router = useRouter()

const gameId = ref(ALL_LOTTERIES)
const schemeId = ref(ALL_SCHEMES)
const currency = ref<BetCurrencyFilter>('all')
const orderNo = ref('')
const dateRange = ref<[string, string] | null>(todayRange())

const gameOptions = ref<SelectOption[]>([])
const cloudSchemes = ref<CloudSchemeOption[]>([])

const lotteryOptions = computed<SelectOption[]>(() => [
  { value: ALL_LOTTERIES, label: '全部彩种' },
  ...gameOptions.value,
])

const currencyOptions = computed<SelectOption[]>(() => [
  { value: 'all', label: '全部币种' },
  ...PRIMARY_CURRENCIES.map((c) => ({ value: c, label: c })),
])

const schemeOptions = computed<SelectOption[]>(() => {
  const opts = cloudSchemes.value
    .filter((s) => gameId.value === ALL_LOTTERIES || s.lotteryCode === gameId.value)
    .reduce<SelectOption[]>((acc, s) => {
      if (acc.some((o) => o.value === s.definitionId)) return acc
      acc.push({ value: s.definitionId, label: s.schemeName })
      return acc
    }, [])
  return [{ value: ALL_SCHEMES, label: '全部方案' }, ...opts]
})

const ready = ref(false)
const filtersReady = ref(false)
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const rows = ref<BetRow[]>([])
const summaryRows = ref<BetCurrencySummary[]>(
  PRIMARY_CURRENCIES.map((c) => ({ currency: c, orderCount: 0, validAmount: 0, pnl: 0 })),
)
const loadSentinel = ref<HTMLElement | null>(null)

const summaryDisplay = computed(() =>
  summaryRows.value.map((row) => ({
    currency: row.currency,
    symbol: currencySymbol(row.currency),
    validAmount: formatBetAmount(row.validAmount),
    pnlText: formatBetPnl(row.pnl),
    pnlTone: row.pnl > 0 ? 'up' : row.pnl < 0 ? 'down' : 'flat',
  })),
)

const INPUT_DEBOUNCE_MS = 400

let loadObserver: IntersectionObserver | null = null
let searchQueued = false
let orderNoTimer: ReturnType<typeof setTimeout> | null = null

function syncSchemeToGame(): void {
  const opts = schemeOptions.value
  if (!opts.length) {
    schemeId.value = ALL_SCHEMES
    return
  }
  if (!opts.some((o) => o.value === schemeId.value)) {
    schemeId.value = ALL_SCHEMES
  }
}

function applyDefaultFilters(): void {
  if (!gameId.value) {
    gameId.value = ALL_LOTTERIES
  }
  if (!schemeId.value) {
    schemeId.value = ALL_SCHEMES
  }
  syncSchemeToGame()
}

function requestSearch(): void {
  if (!filtersReady.value || searchQueued) return
  searchQueued = true
  queueMicrotask(() => {
    searchQueued = false
    void runSearch(true)
  })
}

watch(gameId, () => {
  syncSchemeToGame()
  requestSearch()
})

watch([schemeId, currency, dateRange], () => {
  requestSearch()
})

function scheduleOrderNoSearch(): void {
  if (!filtersReady.value) return
  if (orderNoTimer) clearTimeout(orderNoTimer)
  orderNoTimer = setTimeout(() => {
    orderNoTimer = null
    const q = orderNo.value.trim()
    if (q) {
      // 有注单号时扩到最大可查天数，并回退「全部方案」——已删方案按 definition 联表会漏单
      dateRange.value = lastNDaysRange(MAX_QUERY_DAYS)
      schemeId.value = ALL_SCHEMES
    }
    void runSearch(true)
  }, INPUT_DEBOUNCE_MS)
}

watch(orderNo, () => {
  scheduleOrderNoSearch()
})

watch(dateRange, (v) => {
  if (!v || !v[0] || !v[1]) {
    dateRange.value = todayRange()
  }
})

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member' })
}

function resetPagination(): void {
  nextCursor.value = null
  hasMore.value = false
}

function applySummary(items?: BetCurrencySummary[]): void {
  const byCur = new Map((items ?? []).map((r) => [r.currency.toUpperCase(), r]))
  summaryRows.value = PRIMARY_CURRENCIES.map((c) => {
    const hit = byCur.get(c)
    return {
      currency: c,
      orderCount: hit?.orderCount ?? 0,
      validAmount: hit?.validAmount ?? 0,
      pnl: hit?.pnl ?? 0,
    }
  })
}

async function fetchBetPage(cursor?: string, append = false): Promise<void> {
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) return
  const result = await fetchBetOrders({
    dateFrom: dateRange.value[0],
    dateTo: dateRange.value[1],
    gameCode: gameId.value,
    schemeDefinitionId: schemeId.value,
    currency: currency.value,
    orderNo: orderNo.value.trim() || undefined,
    cursor,
    limit: PAGE_SIZE,
  })
  const mapped = result.items.map(toBetDisplayRow)
  rows.value = append ? [...rows.value, ...mapped] : mapped
  hasMore.value = result.page.hasMore
  nextCursor.value = result.page.nextCursor ?? null
  if (!append) {
    applySummary(result.summary)
  }
}

async function runSearch(auto = false): Promise<void> {
  if (!gameId.value) {
    gameId.value = ALL_LOTTERIES
  }
  if (!schemeId.value) {
    schemeId.value = ALL_SCHEMES
  }
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) {
    if (!auto) ElMessage.warning('请选择日期区间')
    return
  }
  if (rangeSpanDays(dateRange.value[0], dateRange.value[1]) > MAX_QUERY_DAYS) {
    if (!auto) ElMessage.warning(`查询区间不能超过 ${MAX_QUERY_DAYS} 天`)
    return
  }
  loading.value = true
  resetPagination()
  try {
    await fetchBetPage(undefined, false)
  } catch {
    if (!auto) ElMessage.error('加载投注记录失败')
    rows.value = []
    applySummary([])
  } finally {
    loading.value = false
    ready.value = true
  }
}

async function loadMore(): Promise<void> {
  if (!ready.value || !hasMore.value || !nextCursor.value || loading.value || loadingMore.value) return
  loadingMore.value = true
  try {
    await fetchBetPage(nextCursor.value, true)
  } catch {
    ElMessage.error('加载更多失败')
  } finally {
    loadingMore.value = false
  }
}

function setupLoadObserver(): void {
  loadObserver?.disconnect()
  loadObserver = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) void loadMore()
    },
    { root: null, rootMargin: '120px 0px' },
  )
  if (loadSentinel.value) loadObserver.observe(loadSentinel.value)
}

watch(loadSentinel, (el) => {
  if (!loadObserver) return
  loadObserver.disconnect()
  if (el) loadObserver.observe(el)
})

watch(ready, (v) => {
  if (v) setupLoadObserver()
})

async function loadGameOptions() {
  try {
    const [publicRows, items] = await Promise.all([
      fetchPublicLotteries().catch(() => []),
      fetchMemberLotteryFilterOptions(),
    ])
    const nameMap = buildLotteryNameMap(publicRows, items)
    gameOptions.value = items.map((item) => ({
      value: item.code,
      label: lotteryFilterLabel(item.code, item.saleStatus, nameMap),
    }))
  } catch {
    gameOptions.value = []
  }
}

async function loadCloudSchemeOptions() {
  try {
    const [real, sim] = await Promise.all([fetchRunningSchemes('real'), fetchRunningSchemes('sim')])
    const seen = new Set<string>()
    const items: CloudSchemeOption[] = []
    for (const row of [...real, ...sim]) {
      const definitionId = row.definitionId?.trim() ?? ''
      if (!definitionId || seen.has(definitionId)) continue
      seen.add(definitionId)
      items.push({
        definitionId,
        lotteryCode: row.lotteryCode?.trim() ?? '',
        schemeName: row.schemeName,
      })
    }
    cloudSchemes.value = items
  } catch {
    cloudSchemes.value = []
  }
}

async function loadFilters() {
  await Promise.all([loadGameOptions(), loadCloudSchemeOptions()])
  applyDefaultFilters()
  await runSearch(true)
  filtersReady.value = true
}

onMounted(() => {
  void loadFilters()
  setupLoadObserver()
})

onUnmounted(() => {
  if (orderNoTimer) clearTimeout(orderNoTimer)
  orderNoTimer = null
  loadObserver?.disconnect()
  loadObserver = null
})
</script>

<template>
  <div class="mbr member-subpage" data-page="member-bet-records">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">投注记录</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="mbr-main">
      <div class="mbr-top">
        <section class="mbr-card mbr-filters">
          <div class="mbr-filter-grid mbr-filter-grid--3">
            <el-select v-model="gameId" class="mbr-select" placeholder="全部彩种">
              <el-option v-for="o in lotteryOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <el-select v-model="schemeId" class="mbr-select" placeholder="全部方案">
              <el-option v-for="o in schemeOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <el-select v-model="currency" class="mbr-select" placeholder="全部币种">
              <el-option v-for="o in currencyOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </div>
          <div class="mbr-filter-grid mbr-filter-grid--2">
            <el-input
              v-model="orderNo"
              clearable
              placeholder="注单编号（平台/第三方）"
              class="mbr-input"
              @keyup.enter="scheduleOrderNoSearch"
              @clear="scheduleOrderNoSearch"
            />
            <DateRangePickerField v-model="dateRange" class="mbr-drp" :max-days="MAX_QUERY_DAYS" />
          </div>
        </section>

        <section class="mbr-summary" aria-label="币种投注汇总">
          <div v-for="row in summaryDisplay" :key="row.currency" class="mbr-summary-row">
            <div class="mbr-summary-col mbr-summary-cur">
              <span class="mbr-summary-ico" :data-cur="row.currency" aria-hidden="true">{{ row.symbol }}</span>
              <span class="mbr-summary-cur-lbl">{{ row.currency }}</span>
            </div>
            <div class="mbr-summary-col mbr-summary-metric">
              <span class="mbr-summary-lbl">有效投注</span>
              <span class="mbr-summary-val">{{ row.validAmount }}</span>
            </div>
            <div class="mbr-summary-col mbr-summary-metric">
              <span class="mbr-summary-lbl">输赢总计</span>
              <span class="mbr-summary-val" :class="row.pnlTone">{{ row.pnlText }}</span>
            </div>
          </div>
        </section>
      </div>

      <section class="mbr-results" aria-live="polite">
        <el-skeleton v-if="loading && !ready" animated :rows="5" />
        <div v-else-if="!rows.length" class="mbr-empty">
          <span class="mbr-ms mbr-empty-ico" aria-hidden="true">list_alt</span>
          <p class="mbr-empty-title">暂无投注记录</p>
        </div>
        <template v-else>
          <el-table :data="rows" stripe size="small" class="mbr-table member-list-table" style="width: 100%">
            <el-table-column prop="time" label="时间" :min-width="44" />
            <el-table-column prop="game" label="彩种" :min-width="40" />
            <el-table-column prop="orderId" label="单号" :min-width="44" />
            <el-table-column prop="amount" label="投注金额" :min-width="36" />
            <el-table-column prop="returnAmount" label="返奖金额" :min-width="36" />
            <el-table-column prop="status" label="状态" :min-width="36" />
          </el-table>
          <div v-if="hasMore || loadingMore" ref="loadSentinel" class="mbr-load-sentinel" aria-hidden="true" />
          <p v-if="loadingMore" class="mbr-load-hint">加载中…</p>
          <p v-else-if="rows.length && !hasMore" class="mbr-load-hint">已加载全部</p>
          <p v-if="false" class="mbr-footnote">数据仅供参考，以第三方平台为准</p>
        </template>
      </section>
    </main>
  </div>
</template>

<style scoped>
.mbr {
  --mbr-primary: #0050cb;
  --mbr-primary-strong: #0066ff;
  --mbr-surface: #f7f9fb;
  --mbr-tonal: #eef2f7;
  --mbr-card: #ffffff;
  --mbr-on: #191c1e;
  --mbr-on-var: #424656;
  --mbr-on-mute: #727687;
  min-height: 100dvh;
  background: var(--mbr-surface);
  color: var(--mbr-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mbr-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.35rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.mbr-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1rem var(--page-gutter) 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mbr-top {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.mbr-card {
  background: var(--mbr-card);
  border-radius: 1.25rem;
  padding: var(--card-pad);
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

.mbr-filters {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: var(--card-pad);
}

.mbr-filter-grid {
  display: grid;
  gap: 0.35rem;
}

.mbr-filter-grid--3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.mbr-filter-grid--2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.mbr-filter-grid > * {
  min-width: 0;
  width: 100%;
}

.mbr-summary {
  background: #f1f5f9;
  border: 1px solid rgba(194, 198, 216, 0.45);
  border-radius: 0.75rem;
  padding: 0.55rem var(--card-pad);
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.mbr-summary-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.mbr-summary-col {
  min-width: 0;
}

.mbr-summary-cur {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.mbr-summary-ico {
  width: 1.35rem;
  height: 1.35rem;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 0.7rem;
  font-weight: 800;
  color: #fff;
  line-height: 1;
}

.mbr-summary-ico[data-cur='USDT'] {
  background: #26a17b;
}

.mbr-summary-ico[data-cur='TRX'] {
  background: #ef0027;
}

.mbr-summary-ico[data-cur='CNY'] {
  background: #e11d48;
}

.mbr-summary-cur-lbl {
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--mbr-on);
  letter-spacing: 0.01em;
}

.mbr-summary-metric {
  display: inline-flex;
  flex-direction: row;
  align-items: baseline;
  gap: 0.3rem;
  justify-content: flex-start;
}

.mbr-summary-lbl {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--mbr-on-mute);
  line-height: 1.2;
  white-space: nowrap;
  flex-shrink: 0;
}

.mbr-summary-val {
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--mbr-primary);
  font-variant-numeric: tabular-nums;
  line-height: 1.25;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mbr-summary-val.up {
  color: #16a34a;
}

.mbr-summary-val.down {
  color: #dc2626;
}

.mbr-summary-val.flat {
  color: #dc2626;
}

.mbr-select {
  width: 100%;
}

.mbr-select :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mbr-drp {
  width: 100%;
  min-width: 0;
}

.mbr-input :deep(.el-input__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mbr-results {
  min-height: 12rem;
  min-width: 0;
  max-width: 100%;
  overflow-x: hidden;
}

.mbr-table {
  width: 100%;
  background: var(--mbr-card);
  border-radius: 1.25rem;
  overflow: hidden;
  box-shadow: 0 18px 40px -28px rgba(15, 23, 42, 0.12);
  --el-table-border-color: transparent;
  --el-table-header-bg-color: #f8fafc;
  --el-table-bg-color: var(--mbr-card);
}

.mbr-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.mbr-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 2.5rem 1rem;
  gap: 0.35rem;
}

.mbr-empty-ico {
  font-size: 2.25rem;
  color: rgba(0, 80, 203, 0.35);
}

.mbr-empty-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--mbr-on-var);
}

.mbr-load-sentinel {
  height: 1px;
}

.mbr-load-hint {
  margin: 0.65rem 0 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--mbr-on-mute);
  line-height: 1.5;
}

.mbr-footnote {
  margin: 0.75rem 0 0;
  font-size: 0.6875rem;
  line-height: 1.5;
  color: var(--mbr-on-mute);
  text-align: center;
}
</style>

