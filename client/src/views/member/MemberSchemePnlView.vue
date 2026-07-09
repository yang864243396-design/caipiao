<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import DateRangePickerField from '@/components/ui/DateRangePickerField.vue'
import { fetchBetRecordGroups } from '@/api/cloud/betRecords'
import { fetchMemberLotteryFilterOptions, fetchPublicLotteries } from '@/api/games/lotteries'
import { fetchRunningSchemes } from '@/api/cloud/center'
import { buildLotteryNameMap, lotteryFilterLabel } from '@/utils/lotteryDisplayName'

/** 会员中心 · 方案盈亏（§22.2 丙） */

interface CloudSchemeOption {
  definitionId: string
  instanceId: string
  lotteryCode: string
  schemeName: string
}

interface SelectOption {
  value: string
  label: string
}

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

const router = useRouter()

const ALL_SCHEMES = 'all'

const gameId = ref('')
const schemeId = ref('')
const dateRange = ref<[string, string] | null>(todayRange())
const gameOptions = ref<SelectOption[]>([])
const cloudSchemes = ref<CloudSchemeOption[]>([])

const schemeOptions = computed<SelectOption[]>(() => {
  const seen = new Set<string>()
  const opts: SelectOption[] = []
  for (const s of cloudSchemes.value) {
    if (s.lotteryCode !== gameId.value || seen.has(s.definitionId)) continue
    seen.add(s.definitionId)
    opts.push({ value: s.definitionId, label: s.schemeName })
  }
  return [{ value: ALL_SCHEMES, label: '全部方案' }, ...opts]
})

const ready = ref(false)
const loading = ref(false)

interface SchemePnlSummary {
  totalBet: number
  totalPrize: number
  dayPnl: number
}

const summary = ref<SchemePnlSummary | null>(null)

const hasPnlData = computed(() => {
  const s = summary.value
  if (!s) return false
  return s.totalBet !== 0 || s.totalPrize !== 0 || s.dayPnl !== 0
})

const metricRows = computed(() => {
  const s = summary.value
  if (!s) return []
  return [
    { key: 'bet', label: '投注', value: money(s.totalBet), tone: '' },
    { key: 'prize', label: '奖金', value: money(s.totalPrize), tone: '' },
    { key: 'pnl', label: '盈亏', value: signed(s.dayPnl), tone: s.dayPnl > 0 ? 'up' : s.dayPnl < 0 ? 'down' : '' },
  ]
})

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

function instanceIdsForGame(): Set<string> {
  return new Set(
    cloudSchemes.value.filter((s) => s.lotteryCode === gameId.value).map((s) => s.instanceId),
  )
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

function instanceIdsForDefinition(definitionId: string): Set<string> {
  return new Set(
    cloudSchemes.value.filter((s) => s.definitionId === definitionId).map((s) => s.instanceId),
  )
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

function money(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function signed(n: number): string {
  const v = money(Math.abs(n))
  if (!Number.isFinite(n) || n === 0) return v
  return n > 0 ? `+${v}` : `-${v}`
}

async function runSearch(auto = false): Promise<void> {
  if (!gameId.value) {
    if (!auto) ElMessage.warning('请选择彩种')
    summary.value = null
    ready.value = true
    return
  }
  if (!schemeId.value) {
    schemeId.value = ALL_SCHEMES
  }
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) {
    if (!auto) ElMessage.warning('请选择时间区间')
    return
  }
  loading.value = true
  try {
    const data = await fetchBetRecordGroups({
      mode: 'real',
      dateFrom: dateRange.value[0],
      dateTo: dateRange.value[1],
      lotteryCode: gameId.value,
      limit: 200,
    })
    const ids =
      schemeId.value === ALL_SCHEMES
        ? instanceIdsForGame()
        : instanceIdsForDefinition(schemeId.value)
    const matched = (data.groups?.items ?? []).filter((g) => ids.has(g.schemeId))
    summary.value = matched.reduce<SchemePnlSummary>(
      (acc, g) => ({
        totalBet: acc.totalBet + (g.totalBet ?? 0),
        totalPrize: acc.totalPrize + (g.totalPrize ?? 0),
        dayPnl: acc.dayPnl + (g.dayPnl ?? 0),
      }),
      { totalBet: 0, totalPrize: 0, dayPnl: 0 },
    )
  } catch (e) {
    if (!auto) ElMessage.error(e instanceof Error ? e.message : '查询失败')
    summary.value = null
  } finally {
    loading.value = false
    ready.value = true
  }
}

async function onSearch(): Promise<void> {
  await runSearch(false)
}

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
    cloudSchemes.value = [...real, ...sim]
      .map((row) => ({
        definitionId: row.definitionId?.trim() ?? '',
        instanceId: row.id,
        lotteryCode: row.lotteryCode?.trim() ?? '',
        schemeName: row.schemeName,
      }))
      .filter((s) => s.definitionId && s.instanceId)
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
})
</script>

<template>
  <div class="sp member-subpage" data-page="member-scheme-pnl">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back</span>
        </button>
        <h1 class="mss-title">方案盈亏</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="sp-main">
      <section class="sp-card sp-filters">
        <div class="sp-field sp-field--inline">
          <div class="sp-lbl">
            <span class="sp-lbl-bar" aria-hidden="true" />
            <span>彩种</span>
          </div>
          <el-select v-model="gameId" size="large" class="sp-select" placeholder="请选择彩种">
            <el-option v-for="o in gameOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="sp-field sp-field--inline">
          <div class="sp-lbl">
            <span class="sp-lbl-bar" aria-hidden="true" />
            <span>方案</span>
          </div>
          <el-select v-model="schemeId" size="large" class="sp-select">
            <el-option v-for="o in schemeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="sp-field sp-field--inline">
          <div class="sp-lbl">
            <span class="sp-lbl-bar" aria-hidden="true" />
            <span>时间</span>
          </div>
          <DateRangePickerField v-model="dateRange" size="large" class="sp-drp" />
        </div>

        <div class="sp-actions">
          <el-button type="primary" size="large" round class="sp-query" :loading="loading" @click="onSearch">
            查询
          </el-button>
        </div>
      </section>

      <section class="sp-results" aria-live="polite">
        <el-skeleton v-if="loading && !ready" animated :rows="3" />
        <div v-else-if="!hasPnlData" class="sp-empty">
          <span class="mss-ms sp-empty-ico" aria-hidden="true">monitoring</span>
          <p class="sp-empty-title">暂无方案盈亏</p>
        </div>
        <div v-else class="sp-metrics-card">
          <div v-for="row in metricRows" :key="row.key" class="sp-metric-row">
            <span class="sp-metric-lbl">{{ row.label }}</span>
            <span class="sp-metric-val" :class="row.tone">{{ row.value }}</span>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.sp {
  min-height: 100dvh;
  background: #f7f9fb;
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
}

.sp-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1rem 1.15rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.sp-card {
  background: #fff;
  border-radius: 1.25rem;
  padding: 1.15rem;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

.sp-filters {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.sp-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.sp-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 0.65rem;
}

.sp-field--inline .sp-lbl {
  flex: 0 0 4.75rem;
  white-space: nowrap;
}

.sp-field--inline .sp-select,
.sp-field--inline .sp-drp {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
}

.sp-lbl {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.8125rem;
  font-weight: 800;
  color: #191c1e;
  letter-spacing: 0.02em;
}

.sp-lbl-bar {
  width: 3px;
  height: 1rem;
  border-radius: 999px;
  background: rgba(0, 80, 203, 0.35);
}

.sp-select {
  width: 100%;
}

.sp-select :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.sp-drp {
  width: 100%;
  min-width: 0;
}

.sp-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 0.125rem;
}

.sp-query {
  font-weight: 800;
  letter-spacing: 0.03em;
  padding-left: 1.5rem;
  padding-right: 1.5rem;
  box-shadow: 0 14px 32px -16px rgba(0, 80, 203, 0.55);
}

.sp-results {
  min-height: 12rem;
}

.sp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 2.5rem 1rem;
  gap: 0.35rem;
}

.sp-empty-ico {
  font-size: 2.25rem;
  color: rgba(0, 80, 203, 0.35);
}

.sp-empty-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: #424656;
}

.sp-metrics-card {
  background: #fff;
  border-radius: 1rem;
  overflow: hidden;
  box-shadow: 0 12px 30px -24px rgba(15, 23, 42, 0.35);
}

.sp-metric-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.95rem 1.15rem;
  font-size: 0.9375rem;
}

.sp-metric-row + .sp-metric-row {
  border-top: 1px solid #f1f5f9;
}

.sp-metric-lbl {
  color: #424656;
  font-weight: 650;
}

.sp-metric-val {
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
}

.up {
  color: #1f9d63;
}

.down {
  color: #ba1a1a;
}
</style>
