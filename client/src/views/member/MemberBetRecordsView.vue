<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import DateRangePickerField from '@/components/ui/DateRangePickerField.vue'
import { fetchBetOrders, toBetDisplayRow } from '@/api/orders/bets'
import { fetchMemberLotteryFilterOptions, fetchPublicLotteries } from '@/api/games/lotteries'
import { fetchRunningSchemes } from '@/api/cloud/center'
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

const MAX_QUERY_DAYS = 3
const PAGE_SIZE = 20
const ALL_SCHEMES = 'all'

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

const gameId = ref('')
const schemeId = ref('')
const orderNo = ref('')
const dateRange = ref<[string, string] | null>(todayRange())

const gameOptions = ref<SelectOption[]>([])
const cloudSchemes = ref<CloudSchemeOption[]>([])

const schemeOptions = computed<SelectOption[]>(() => {
  const opts = cloudSchemes.value
    .filter((s) => s.lotteryCode === gameId.value)
    .reduce<SelectOption[]>((acc, s) => {
      if (acc.some((o) => o.value === s.definitionId)) return acc
      acc.push({ value: s.definitionId, label: s.schemeName })
      return acc
    }, [])
  return [{ value: ALL_SCHEMES, label: '全部方案' }, ...opts]
})

const ready = ref(false)
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const rows = ref<BetRow[]>([])
const loadSentinel = ref<HTMLElement | null>(null)

let loadObserver: IntersectionObserver | null = null

watch(gameId, () => {
  syncSchemeToGame()
})

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

function pickDefaultGameId(): string {
  const codesWithSchemes = new Set(cloudSchemes.value.map((s) => s.lotteryCode))
  const preferred = gameOptions.value.find((g) => codesWithSchemes.has(g.value))
  return preferred?.value ?? gameOptions.value[0]?.value ?? ''
}

function applyDefaultFilters(): void {
  if (!gameId.value) {
    gameId.value = pickDefaultGameId()
  }
  syncSchemeToGame()
}

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

async function fetchBetPage(cursor?: string, append = false): Promise<void> {
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) return
  const result = await fetchBetOrders({
    dateFrom: dateRange.value[0],
    dateTo: dateRange.value[1],
    gameCode: gameId.value,
    schemeDefinitionId: schemeId.value,
    orderNo: orderNo.value.trim() || undefined,
    cursor,
    limit: PAGE_SIZE,
  })
  const mapped = result.items.map(toBetDisplayRow)
  rows.value = append ? [...rows.value, ...mapped] : mapped
  hasMore.value = result.page.hasMore
  nextCursor.value = result.page.nextCursor ?? null
}

async function runSearch(auto = false): Promise<void> {
  if (!gameId.value) {
    if (!auto) ElMessage.warning('请选择彩种')
    resetPagination()
    rows.value = []
    ready.value = true
    return
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
  } finally {
    loading.value = false
    ready.value = true
  }
}

async function onSearch(): Promise<void> {
  await runSearch(false)
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
}

onMounted(() => {
  void loadFilters()
  setupLoadObserver()
})

onUnmounted(() => {
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
          <span class="mss-ms" aria-hidden="true">arrow_back</span>
        </button>
        <h1 class="mss-title">投注记录</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="mbr-main">
      <section class="mbr-card mbr-filters">
        <div class="mbr-field mbr-field--inline">
          <div class="mbr-lbl">
            <span class="mbr-lbl-bar" aria-hidden="true" />
            <span>彩种</span>
          </div>
          <el-select v-model="gameId" size="large" class="mbr-select" placeholder="请选择彩种">
            <el-option v-for="o in gameOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="mbr-field mbr-field--inline">
          <div class="mbr-lbl">
            <span class="mbr-lbl-bar" aria-hidden="true" />
            <span>方案</span>
          </div>
          <el-select v-model="schemeId" size="large" class="mbr-select">
            <el-option v-for="o in schemeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="mbr-field mbr-field--inline">
          <div class="mbr-lbl">
            <span class="mbr-lbl-bar" aria-hidden="true" />
            <span>时间</span>
          </div>
          <DateRangePickerField v-model="dateRange" size="large" class="mbr-drp" :max-days="MAX_QUERY_DAYS" />
        </div>
        <div class="mbr-field mbr-field--inline">
          <div class="mbr-lbl">
            <span class="mbr-lbl-bar" aria-hidden="true" />
            <span>注单编号</span>
          </div>
          <el-input v-model="orderNo" clearable size="large" placeholder="输入第三方注单编号" class="mbr-input" />
        </div>

        <div class="mbr-actions">
          <el-button type="primary" size="large" round class="mbr-query" :loading="loading" @click="onSearch">
            查询
          </el-button>
        </div>
      </section>

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
  padding: 1rem 1.15rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mbr-card {
  background: var(--mbr-card);
  border-radius: 1.25rem;
  padding: 1.15rem;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

.mbr-filters {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.mbr-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.mbr-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 0.65rem;
}

.mbr-field--inline .mbr-lbl {
  flex: 0 0 4.75rem;
  white-space: nowrap;
}

.mbr-field--inline .mbr-select,
.mbr-field--inline .mbr-drp {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
}

.mbr-lbl {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--mbr-on);
  letter-spacing: 0.02em;
}

.mbr-lbl-bar {
  width: 3px;
  height: 1rem;
  border-radius: 999px;
  background: rgba(0, 80, 203, 0.35);
}

.mbr-select {
  width: 100%;
}

.mbr-select :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mbr-field--inline .mbr-input {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
}

.mbr-drp {
  width: 100%;
  min-width: 0;
}

.mbr-input :deep(.el-input__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mbr-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-top: 0.125rem;
}

.mbr-query {
  font-weight: 800;
  letter-spacing: 0.03em;
  padding-left: 1.5rem;
  padding-right: 1.5rem;
  box-shadow: 0 14px 32px -16px rgba(0, 80, 203, 0.55);
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

