<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import ContentDialog from '@/components/ui/ContentDialog.vue'
import { startCloudRunningSync, cloudRunningPollMs } from '@/composables/useCloudRunningPoll'
import { ApiError } from '@/api/client'
import { deleteSchemeDefinition, getSchemeDefinition } from '@/api/schemes/definitions'
import { confirmDialog } from '@/utils/confirmDialog'
import { normalizeSchemeTimePairFromConfig, schemeStartTimeOpenError } from '@/utils/schemeDateTime'
import { minBetOpenMessage, schemeMinBetOpenError } from '@/utils/schemeMinBet'
import { fetchPrimaryCurrency } from '@/api/guaji/accounts'
import { redirectToGuajiAuthIfNeeded, isGuajiAuthRequiredError } from '@/composables/useGuajiAuthGuard'
import {
  fetchCloudGlobalSettings,
  fetchCloudCenterStats,
  formatCloudStatAmount,
  emptyCloudCenterStats,
  fetchLookbackSettings,
  fetchRunningSchemesPage,
  CLOUD_SCHEME_PAGE_SIZE,
  formatRunTime,
  schemeCardDisplayStatus,
  globalSettingsFromUi,
  globalSettingsToUi,
  instanceToDisplay,
  lookbackFromUi,
  lookbackSummaryFromUi,
  lookbackToUi,
  type LookbackJudgment,
  mergeCloudSchemesStable,
  mergeSchemeCountdownOnPoll,
  schemeCountdownDisplayFields,
  SCHEME_COUNTDOWN_WAITING_LABEL,
  tickSchemeRunTimeSec,
  schemeCountdownText,
  stopCloudInstance,
  startCloudInstance,
  saveCloudGlobalSettings,
  saveLookbackSettings,
  saveCloudInstanceMultiplier,
  saveCloudInstanceSimBet,
  normalizeSchemeMultiplier,
  type CloudSchemeCard,
} from '@/api/cloud/center'

const router = useRouter()

const totalStopLoss = ref('0')
const totalTakeProfit = ref('0')
const planMultiplier = ref('1')
const breakPeriodStop = ref(false)
const lookbackSummary = ref('无')
const pageLoading = ref(false)

const runningSchemes = ref<CloudSchemeCard[]>([])
const listNextCursor = ref<string | undefined>()
const listHasMore = ref(true)
const listTotal = ref(0)
const listLoadingMore = ref(false)
const loadMoreSentinel = ref<HTMLElement | null>(null)
/** 倒计时本地 1s tick；列表同步见 useCloudRunningPoll（WS 事件 + 15s REST 轮询） */
/** 各方案倍数系数上次已保存值，用于弹窗确定时判断是否需要提交 */
const multiplierSaved = ref<Record<string, string>>({})
const multiplierSavingId = ref<string | null>(null)
const simBetSavingId = ref<string | null>(null)
const multiplierDialogVisible = ref(false)
const multiplierEditScheme = ref<CloudSchemeCard | null>(null)
const multiplierDraft = ref('1')

const searchDialogVisible = ref(false)
const searchDraft = ref('')
const schemeSearchKeyword = ref('')

const displayedSchemes = computed(() => {
  const q = schemeSearchKeyword.value.trim().toLowerCase()
  if (!q) return runningSchemes.value
  return runningSchemes.value.filter((s) => {
    const hay = `${s.schemeName} ${s.lotteryName} ${s.definitionId} ${s.id}`.toLowerCase()
    return hay.includes(q)
  })
})

const lookbackDialogVisible = ref(false)

const lookback = reactive({
  runModeSim: false,
  runModeProd: false,
  judgment: '' as LookbackJudgment,
  singleProfitThreshold: '100.00',
  singleLossThreshold: '0.00',
  overallProfitThreshold: '',
  overallLossThreshold: '',
  schemeWinsMin: '',
  schemeWinsMax: '',
  periodProfit: '',
  periodLoss: '',
})

const globalSaving = ref(false)
const enableAllBusy = ref(false)
const centerStats = ref(emptyCloudCenterStats())

let stopCloudPoll: (() => void) | null = null
let cloudRefresh: (() => void) | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null
let loadMoreObserver: IntersectionObserver | null = null
let lastWaitingRefreshAt = 0

function setupLoadMoreObserver() {
  loadMoreObserver?.disconnect()
  if (!loadMoreSentinel.value) return
  loadMoreObserver = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        void loadSchemePage(false)
      }
    },
    { root: null, rootMargin: '120px', threshold: 0 },
  )
  loadMoreObserver.observe(loadMoreSentinel.value)
}

function applyRunningSchemes(cards: CloudSchemeCard[], preserveOrder = false) {
  const prev = runningSchemes.value
  const merged = preserveOrder ? mergeCloudSchemesStable(prev, cards) : cards
  runningSchemes.value = merged
  const next: Record<string, string> = { ...multiplierSaved.value }
  for (const s of merged) {
    next[s.id] = s.multiplier
  }
  multiplierSaved.value = next
}

function appendRunningSchemes(cards: CloudSchemeCard[]) {
  if (cards.length === 0) return
  const seen = new Set(runningSchemes.value.map((s) => s.id))
  const merged = [...runningSchemes.value]
  for (const c of cards) {
    if (!seen.has(c.id)) {
      merged.push(c)
      seen.add(c.id)
      multiplierSaved.value[c.id] = c.multiplier
    }
  }
  runningSchemes.value = merged
}

async function loadSchemePage(reset = false) {
  if (listLoadingMore.value) return
  if (!reset && !listHasMore.value) return
  if (reset) {
    listNextCursor.value = undefined
    listHasMore.value = true
  }
  if (reset) {
    pageLoading.value = true
  } else {
    listLoadingMore.value = true
  }
  try {
    const res = await fetchRunningSchemesPage({
      limit: CLOUD_SCHEME_PAGE_SIZE,
      cursor: reset ? undefined : listNextCursor.value,
    })
    const cards = res.items.map(instanceToDisplay)
    listTotal.value = res.total ?? cards.length
    listHasMore.value = res.page?.hasMore ?? false
    listNextCursor.value = res.page?.nextCursor
    if (reset) {
      applyRunningSchemes(cards)
    } else {
      appendRunningSchemes(cards)
    }
  } catch (e) {
    if (reset) {
      ElMessage.error(e instanceof Error ? e.message : '加载失败')
    }
  } finally {
    if (reset) {
      pageLoading.value = false
    }
    listLoadingMore.value = false
    await nextTick()
    setupLoadMoreObserver()
  }
}

async function refreshCloudStats() {
  try {
    centerStats.value = await fetchCloudCenterStats()
  } catch {
    /* 保留上次有效数据 */
  }
}

async function loadCloudData() {
  try {
    const [lb, global, stats] = await Promise.all([
      fetchLookbackSettings(),
      fetchCloudGlobalSettings(),
      fetchCloudCenterStats().catch(() => emptyCloudCenterStats()),
    ])
    centerStats.value = stats
    await loadSchemePage(true)
    Object.assign(lookback, lookbackToUi(lb))
    lookbackSummary.value = lookbackSummaryFromUi(lookback)
    const g = globalSettingsToUi(global)
    totalStopLoss.value = g.totalStopLoss
    totalTakeProfit.value = g.totalTakeProfit
    planMultiplier.value = g.planMultiplier
    breakPeriodStop.value = g.breakPeriodStop
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

async function persistGlobalSettings() {
  if (globalSaving.value) return
  globalSaving.value = true
  try {
    const saved = await saveCloudGlobalSettings(
      globalSettingsFromUi({
        totalStopLoss: totalStopLoss.value,
        totalTakeProfit: totalTakeProfit.value,
        planMultiplier: planMultiplier.value,
        breakPeriodStop: breakPeriodStop.value,
      }),
    )
    const g = globalSettingsToUi(saved)
    totalStopLoss.value = g.totalStopLoss
    totalTakeProfit.value = g.totalTakeProfit
    planMultiplier.value = g.planMultiplier
    breakPeriodStop.value = g.breakPeriodStop
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存全局规则失败')
  } finally {
    globalSaving.value = false
  }
}

function tickSchemeLiveFields() {
  let periodEnded = false
  let needsWaitingRefresh = false
  for (const s of runningSchemes.value) {
    s.runTimeSec = tickSchemeRunTimeSec(s)

    if (!s.countdownEndTime) continue

    const prev = s.countdownSec
    const display = schemeCountdownDisplayFields(s)
    s.countdownSec = display.countdownSec
    if (s.status === 'running') {
      s.countdownLabel = display.countdownLabel
    }
    if (prev > 0 && s.countdownSec === 0) {
      periodEnded = true
    }
    if (
      s.status === 'running'
      && s.countdownSec <= 0
      && (s.countdownLabel === SCHEME_COUNTDOWN_WAITING_LABEL || display.countdownLabel === SCHEME_COUNTDOWN_WAITING_LABEL)
    ) {
      needsWaitingRefresh = true
    }
  }
  if (periodEnded) {
    void cloudRefresh?.()
    return
  }
  if (needsWaitingRefresh) {
    const now = Date.now()
    const waitMs = Math.max(5_000, Math.floor(cloudRunningPollMs() / 2))
    if (now - lastWaitingRefreshAt >= waitMs) {
      lastWaitingRefreshAt = now
      void cloudRefresh?.()
    }
  }
}

function statPnlClass(n: number): string {
  return n < 0 ? 'cc-stat-em cc-stat-em--loss' : 'cc-stat-em'
}

onMounted(async () => {
  const sync = startCloudRunningSync(
    () => runningSchemes.value.map((s) => s.id),
    (cards) => applyRunningSchemes(cards, true),
  )
  stopCloudPoll = sync.stop
  cloudRefresh = async () => {
    await sync.refresh()
    await refreshCloudStats()
  }
  countdownTimer = window.setInterval(tickSchemeLiveFields, 1000)
  await loadCloudData()
  void sync.refresh()
})

onUnmounted(() => {
  loadMoreObserver?.disconnect()
  loadMoreObserver = null
  if (countdownTimer) {
    window.clearInterval(countdownTimer)
    countdownTimer = null
  }
  stopCloudPoll?.()
  stopCloudPoll = null
  cloudRefresh = null
})

function openLookbackDialog() {
  lookback.schemeWinsMin = toPositiveIntString(lookback.schemeWinsMin)
  lookback.schemeWinsMax = toPositiveIntString(lookback.schemeWinsMax)
  lookbackDialogVisible.value = true
}

function cancelLookback() {
  lookbackDialogVisible.value = false
}

/** 仅保留正整数（1, 2, 3…），非法输入清空 */
function toPositiveIntString(v: string | number): string {
  const digits = String(v ?? '').replace(/[^\d]/g, '')
  if (!digits) return ''
  const n = parseInt(digits, 10)
  return n >= 1 ? String(n) : ''
}

function onSchemeWinsMinChange(v: string | number) {
  lookback.schemeWinsMin = toPositiveIntString(v)
}

function onSchemeWinsMaxChange(v: string | number) {
  lookback.schemeWinsMax = toPositiveIntString(v)
}

function openMultiplierDialog(s: CloudSchemeCard) {
  if (multiplierSavingId.value !== null) return
  multiplierEditScheme.value = s
  multiplierDraft.value = s.multiplier
  multiplierDialogVisible.value = true
}

function cancelMultiplierDialog() {
  multiplierDialogVisible.value = false
}

watch(multiplierDialogVisible, (open) => {
  if (!open) multiplierEditScheme.value = null
})

/** 编辑中只保留数字，允许清空以便改成其他正整数；确认时再规范 */
function onMultiplierDraftChange(v: string | number) {
  multiplierDraft.value = String(v ?? '').replace(/[^\d]/g, '')
}

async function confirmMultiplierDialog() {
  const s = multiplierEditScheme.value
  if (!s || multiplierSavingId.value !== null) return

  const digits = multiplierDraft.value.replace(/[^\d]/g, '')
  if (!digits || Number.parseInt(digits, 10) < 1) {
    ElMessage.warning('请输入不小于 1 的正整数')
    return
  }
  const normalized = normalizeSchemeMultiplier(digits)
  multiplierDraft.value = normalized
  const baseline = multiplierSaved.value[s.id] ?? s.multiplier
  if (normalized === baseline) {
    multiplierDialogVisible.value = false
    return
  }

  multiplierSavingId.value = s.id
  try {
    const row = await saveCloudInstanceMultiplier(s.id, Number(normalized))
    patchSchemeCard(row)
    multiplierSaved.value = {
      ...multiplierSaved.value,
      [s.id]: normalizeSchemeMultiplier(row.multiplier),
    }
    multiplierDialogVisible.value = false
    ElMessage.success('倍数系数已更新')
  } catch (e) {
    ElMessage.error(e instanceof ApiError ? e.message : '倍数系数保存失败')
  } finally {
    multiplierSavingId.value = null
  }
}

async function toggleSimBet(s: CloudSchemeCard, simBet: boolean) {
  if (simBetSavingId.value !== null) return
  const prev = s.simBet
  simBetSavingId.value = s.id
  try {
    const row = await saveCloudInstanceSimBet(s.id, simBet)
    patchSchemeCard(row)
    ElMessage.success(simBet ? '已开启模拟投注' : '已关闭模拟投注')
  } catch (e) {
    const idx = runningSchemes.value.findIndex((x) => x.id === s.id)
    if (idx >= 0) runningSchemes.value[idx] = { ...runningSchemes.value[idx], simBet: prev }
    ElMessage.error(e instanceof ApiError ? e.message : '模拟投注设置保存失败')
  } finally {
    simBetSavingId.value = null
  }
}

function toggleLookbackJudgment(mode: Exclude<LookbackJudgment, ''>) {
  lookback.judgment = lookback.judgment === mode ? '' : mode
}

function toggleLookbackRunMode(mode: 'sim' | 'prod') {
  if (mode === 'sim') lookback.runModeSim = !lookback.runModeSim
  else lookback.runModeProd = !lookback.runModeProd
}

function validateSchemeWinsRange(): string | null {
  if (lookback.judgment !== 'overall') return null
  const minS = lookback.schemeWinsMin.trim()
  const maxS = lookback.schemeWinsMax.trim()
  if (!minS && !maxS) return null
  if (!minS || !maxS) return '请填写方案几回头的最小与最大次数'
  const min = Number(minS)
  const max = Number(maxS)
  if (!Number.isInteger(min) || min < 1) return '最小次数须为正整数'
  if (!Number.isInteger(max) || max < 1) return '最大次数须为正整数'
  if (min >= max) return '最小次数须小于最大次数'
  return null
}

async function confirmLookback() {
  const winsErr = validateSchemeWinsRange()
  if (winsErr) {
    ElMessage.warning(winsErr)
    return
  }
  try {
    const saved = await saveLookbackSettings(lookbackFromUi(lookback))
    Object.assign(lookback, lookbackToUi(saved))
    lookbackSummary.value = lookbackSummaryFromUi(lookback)
    lookbackDialogVisible.value = false
    ElMessage.success('已保存回头设置')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

function onHeaderSearch() {
  searchDraft.value = schemeSearchKeyword.value
  searchDialogVisible.value = true
}

function applySchemeSearch() {
  schemeSearchKeyword.value = searchDraft.value.trim()
  searchDialogVisible.value = false
  if (schemeSearchKeyword.value && displayedSchemes.value.length === 0) {
    ElMessage.info('未找到匹配的运行中方案')
  }
}

function clearSchemeSearch() {
  schemeSearchKeyword.value = ''
  searchDraft.value = ''
}

function onHeaderAdd() {
  void router.push({
    name: 'advanced-scheme-edit',
    params: { schemeId: 'new' },
    query: { draft: '1', kind: 'custom', fresh: '1' },
  })
}

async function enableAllSchemes() {
  if (enableAllBusy.value) return
  const targets = runningSchemes.value.filter(
    (s) => (s.status === 'pending' || s.status === 'paused') && s.statusReason !== 'maintenance',
  )
  if (targets.length === 0) {
    ElMessage.info('当前没有待开启的方案')
    return
  }
  const ok = await confirmDialog({
    title: '一键开启方案',
    message: `将开启 ${targets.length} 个待开启方案。是否继续？`,
    confirmText: '全部开启',
    tone: 'primary',
  })
  if (!ok) return

  enableAllBusy.value = true
  let okCount = 0
  const failed: string[] = []
  try {
    for (const s of targets) {
      if (!(await validateBeforeOpen(s))) continue
      try {
        const updated = await startCloudInstance(s.id)
        patchSchemeCard(updated)
        okCount++
      } catch (e) {
        if (isGuajiAuthRequiredError(e)) {
          await redirectToGuajiAuthIfNeeded(e, (path) => router.push(path))
          return
        }
        const msg =
          e instanceof ApiError && e.message.includes('预计开启时间')
            ? START_TIME_OPEN_MSG
            : e instanceof ApiError && (e.message.includes('单次投注金额') || e.message.includes('低于0.1'))
              ? minBetOpenMessage('CNY')
              : e instanceof Error
                ? e.message
                : '失败'
        failed.push(`${s.schemeName}：${msg}`)
      }
    }
    if (okCount > 0) {
      ElMessage.success(`已成功开启 ${okCount} 个方案`)
      void refreshCloudStats()
    }
    if (failed.length > 0) {
      const hint = failed.slice(0, 2).join('；')
      ElMessage.warning(
        failed.length === targets.length ? `全部开启失败：${hint}` : `部分失败（${failed.length}/${targets.length}）：${hint}`,
      )
    }
  } finally {
    enableAllBusy.value = false
  }
}

function openBetRecords() {
  void router.push({ name: 'bet-records' })
}

/** 点击运行中方案 → 查看方案详情（合并新增方案 + 方案配置） */
function openSchemeDetail(s: CloudSchemeCard) {
  if (!s.definitionId) {
    ElMessage.info('该方案暂无可查看的配置详情')
    return
  }
  void router.push({
    name: 'scheme-detail',
    params: { definitionId: s.definitionId },
    query: {
      turnover: s.turnover,
      sessionPnl: s.sessionPnl,
      multiplier: s.multiplier,
      status: s.status,
    },
  })
}

const START_TIME_OPEN_MSG = '预计开启时间小于现在时间 请修改后再执行开启'

async function validateBeforeOpen(s: CloudSchemeCard): Promise<boolean> {
  if (!s.definitionId) return true
  try {
    const def = await getSchemeDefinition(s.definitionId)
    const times = normalizeSchemeTimePairFromConfig(
      def.config?.startTime,
      def.config?.endTime,
    )
    const timeErr = schemeStartTimeOpenError(times.start)
    if (timeErr) {
      await confirmDialog({
        title: '无法开启',
        message: timeErr,
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return false
    }
    const currency = await fetchPrimaryCurrency().catch(() => 'CNY')
    const minBetErr = schemeMinBetOpenError(
      (def.config ?? {}) as Record<string, unknown>,
      s.multiplier,
      currency,
    )
    if (minBetErr) {
      await confirmDialog({
        title: '无法开启',
        message: minBetErr,
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return false
    }
    return true
  } catch {
    return true
  }
}

function showStartOpenError(e: unknown): void {
  void (async () => {
    if (await redirectToGuajiAuthIfNeeded(e, (path) => router.push(path))) return
    const raw = e instanceof Error ? e.message : '操作失败'
    const isStartTime =
      (e instanceof ApiError && raw.includes('预计开启时间')) || raw.includes('预计开启时间')
    const isMinBet =
      (e instanceof ApiError && (raw.includes('单次投注金额') || raw.includes('低于'))) ||
      raw.includes('单次投注金额')
    if (isStartTime) {
      void confirmDialog({
        title: '无法开启',
        message: START_TIME_OPEN_MSG,
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return
    }
    if (isMinBet) {
      void confirmDialog({
        title: '无法开启',
        message: raw.includes('单次投注金额') ? raw : minBetOpenMessage('CNY'),
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return
    }
    ElMessage.error(raw)
  })()
}


async function startScheme(s: CloudSchemeCard) {
  if (!(await validateBeforeOpen(s))) return
  try {
    const updated = await startCloudInstance(s.id)
    patchSchemeCard(updated)
    void refreshCloudStats()
    ElMessage.success(`已开启：${s.schemeName}`)
  } catch (e) {
    showStartOpenError(e)
  }
}

async function stopScheme(s: CloudSchemeCard) {
  try {
    const updated = await stopCloudInstance(s.id)
    patchSchemeCard(updated)
    void refreshCloudStats()
    ElMessage.success(`已停止：${s.schemeName}`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '停止失败')
  }
}

function canStartScheme(s: CloudSchemeCard): boolean {
  return s.status === 'pending' || s.status === 'paused'
}

function patchSchemeCard(updated: Awaited<ReturnType<typeof startCloudInstance>>) {
  const idx = runningSchemes.value.findIndex((x) => x.id === updated.id)
  const card = instanceToDisplay(updated)
  if (idx >= 0) {
    runningSchemes.value[idx] = mergeSchemeCountdownOnPoll(runningSchemes.value[idx], card)
  }
}

async function removeScheme(card: CloudSchemeCard) {
  if (!card.definitionId) {
    ElMessage.error('该方案缺少配置信息，无法删除')
    return
  }
  const ok = await confirmDialog({
    title: '删除方案',
    message: `确定删除方案「${card.schemeName}」？删除后云端实例与方案配置将一并移除，不可恢复。`,
    tone: 'danger',
    confirmText: '删除',
    cancelText: '取消',
  })
  if (!ok) return
  try {
    await deleteSchemeDefinition(card.definitionId)
    runningSchemes.value = runningSchemes.value.filter((x) => x.id !== card.id)
    listTotal.value = Math.max(0, listTotal.value - 1)
    ElMessage.success('方案已删除')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

const schemeCount = computed(() => (listTotal.value > 0 ? listTotal.value : runningSchemes.value.length))
const displayedSchemeCount = computed(() => displayedSchemes.value.length)

function schemeCardClass(s: CloudSchemeCard): string[] {
  const classes = ['cc-card', 'cc-card--clickable']
  if (s.status === 'running') classes.push('cc-card--running')
  return classes
}

function statusBadgeClass(s: CloudSchemeCard): string {
  const reason = schemeCardDisplayStatus(s).reason
  if (s.status === 'running') {
    switch (reason) {
      case 'await_next_bet':
        return 'cc-badge--info'
      case 'cloud_active':
        return 'cc-badge--active'
      default:
        return 'cc-badge--info'
    }
  }
  if (s.status === 'pending' || s.status === 'paused') {
    switch (s.statusReason) {
      case 'insufficient_funds':
        return 'cc-badge--warn'
      case 'bet_failed':
        return 'cc-badge--warn'
      case 'maintenance':
        return 'cc-badge--muted'
      case 'end_time':
        return 'cc-badge--info'
      case 'scheme_stop_loss':
      case 'scheme_take_profit':
      case 'total_stop_loss':
      case 'total_take_profit':
        return 'cc-badge--warn'
      default:
        return ''
    }
  }
  return ''
}
</script>

<template>
  <div class="cc" data-page="cloud-center">
    <header class="cc-head" role="banner">
      <div class="cc-head-top">
        <h1 class="cc-title">云端中心</h1>
        <div class="cc-head-actions">
          <button type="button" class="cc-icon-btn" aria-label="搜索方案" @click="onHeaderSearch">
            <span class="cc-ms" aria-hidden="true">search</span>
          </button>
          <button type="button" class="cc-icon-btn" aria-label="新增方案" @click="onHeaderAdd">
            <span class="cc-ms" aria-hidden="true">add_circle</span>
          </button>
        </div>
      </div>

      <div class="cc-stats">
        <div class="cc-stat-col">
          <h3 class="cc-stat-h">正式运行</h3>
          <div class="cc-stat-rows">
            <div class="cc-stat-row">
              <span>总投注</span>
              <span>{{ formatCloudStatAmount(centerStats.formal.totalTurnover) }}</span>
            </div>
            <div class="cc-stat-row">
              <span>总盈亏</span>
              <span :class="statPnlClass(centerStats.formal.totalSessionPnl)">{{
                formatCloudStatAmount(centerStats.formal.totalSessionPnl) }}</span>
            </div>
            <div class="cc-stat-row cc-stat-row--pill">
              <span>运行中盈亏</span>
              <span :class="statPnlClass(centerStats.formal.runningSessionPnl)">{{
                formatCloudStatAmount(centerStats.formal.runningSessionPnl) }}</span>
            </div>
          </div>
        </div>
        <div class="cc-stat-divider" aria-hidden="true" />
        <div class="cc-stat-col">
          <h3 class="cc-stat-h">模拟运行</h3>
          <div class="cc-stat-rows">
            <div class="cc-stat-row">
              <span>总投注</span>
              <span>{{ formatCloudStatAmount(centerStats.sim.totalTurnover) }}</span>
            </div>
            <div class="cc-stat-row">
              <span>总盈亏</span>
              <span :class="statPnlClass(centerStats.sim.totalSessionPnl)">{{
                formatCloudStatAmount(centerStats.sim.totalSessionPnl) }}</span>
            </div>
            <div class="cc-stat-row cc-stat-row--pill">
              <span>运行中盈亏</span>
              <span :class="statPnlClass(centerStats.sim.runningSessionPnl)">{{
                formatCloudStatAmount(centerStats.sim.runningSessionPnl) }}</span>
            </div>
          </div>
        </div>
      </div>
    </header>

    <main class="cc-main">
      <section class="cc-panel">
        <div class="cc-panel-grid">
          <div class="cc-field">
            <label class="cc-lbl">
              总止损
              <span class="cc-ms cc-lbl-ico" aria-hidden="true">info</span>
            </label>
            <el-input v-model="totalStopLoss" type="number" size="default" class="cc-el-inp"
              @change="persistGlobalSettings" />
          </div>
          <div class="cc-field">
            <label class="cc-lbl">
              总止盈
              <span class="cc-ms cc-lbl-ico" aria-hidden="true">trending_up</span>
            </label>
            <el-input v-model="totalTakeProfit" type="number" size="default" class="cc-el-inp"
              @change="persistGlobalSettings" />
          </div>
        </div>

        <div class="cc-field">
          <label class="cc-lbl">方案倍数系数</label>
          <div class="cc-mult-wrap">
            <div class="cc-mult-prefix" aria-hidden="true">乘</div>
            <el-input v-model="planMultiplier" type="number" size="default" class="cc-el-inp cc-el-inp--grow"
              @change="persistGlobalSettings" />
          </div>
        </div>

        <div class="cc-row-between">
          <div class="cc-hint">
            <span class="cc-ms cc-hint-ico" aria-hidden="true">history</span>
            <span class="cc-hint-txt">目前回头设置：{{ lookbackSummary }}</span>
          </div>
          <div class="cc-switch-row">
            <span class="cc-switch-lbl">断期停投</span>
            <el-switch v-model="breakPeriodStop" size="small" @change="persistGlobalSettings" />
          </div>
        </div>

        <div class="cc-actions">
          <el-button type="primary" size="default" round class="cc-btn cc-btn--primary" :loading="enableAllBusy"
            :disabled="enableAllBusy" @click="enableAllSchemes">
            一键开启方案
          </el-button>
          <el-button size="default" round class="cc-btn cc-btn--outline" @click="openLookbackDialog">
            回头设置
          </el-button>
          <el-button size="default" round class="cc-btn cc-btn--outline" @click="openBetRecords">
            投注记录
          </el-button>
        </div>
      </section>

      <section class="cc-list-sec">
        <div class="cc-list-head">
          <h2 class="cc-list-h2">运行中方案</h2>
          <span class="cc-list-meta">
            共 {{ schemeCount }} 个方案
            <template v-if="schemeSearchKeyword">
              ，筛选 {{ displayedSchemeCount }} 个
              <button type="button" class="cc-search-clear" @click="clearSchemeSearch">清除</button>
            </template>
          </span>
        </div>

        <p v-if="displayedSchemes.length === 0" class="cc-list-empty">
          {{ schemeSearchKeyword ? '未找到匹配的运行中方案' : '暂无运行中的方案' }}
        </p>

        <div v-for="s in displayedSchemes" :key="s.id" :class="schemeCardClass(s)" role="button" tabindex="0"
          @click="openSchemeDetail(s)" @keyup.enter="openSchemeDetail(s)">
          <div class="cc-card-hd">
            <div class="cc-card-title-row">
              <h3 class="cc-card-h3">{{ s.lotteryName }}</h3>
              <el-tag v-if="s.runTypeLabel" size="small" type="info" effect="plain" class="cc-runtype-tag">
                {{ s.runTypeLabel }}
              </el-tag>
              <span class="cc-ms cc-card-title-arrow" aria-hidden="true">chevron_right</span>
            </div>
            <span class="cc-badge" :class="statusBadgeClass(s)" :title="schemeCardDisplayStatus(s).label">{{
              schemeCardDisplayStatus(s).label }}</span>
          </div>

          <div class="cc-kv-grid">
            <div class="cc-kv">
              <span class="cc-k">方案名称</span>
              <span class="cc-v cc-v--ellipsis" :title="s.schemeName">{{ s.schemeName }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">投注流水</span>
              <span class="cc-v">{{ s.turnover }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">倒计时</span>
              <span class="cc-v cc-v--primary cc-v--mono">{{ schemeCountdownText(s) }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">本次盈亏</span>
              <span class="cc-v cc-v--error">{{ s.sessionPnl }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">运行时间</span>
              <span class="cc-v cc-v--mono">{{ formatRunTime(s.runTimeSec) }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">回头盈亏</span>
              <span class="cc-v cc-v--error">{{ s.lookbackPnl }}</span>
            </div>
            <div class="cc-kv cc-kv--last cc-kv--mult" @click.stop>
              <span class="cc-k">倍数系数</span>
              <el-input :model-value="s.multiplier" readonly size="small"
                class="cc-el-inp cc-el-inp--mult cc-el-inp--mult-trigger" :disabled="multiplierSavingId === s.id"
                @click="openMultiplierDialog(s)" />
            </div>
          </div>

          <div class="cc-card-foot">
            <div class="cc-foot-left">
              <el-button v-if="canStartScheme(s)" type="primary" round class="cc-start-btn"
                @click.stop="startScheme(s)">
                开启方案
              </el-button>
              <el-button v-else-if="s.status === 'running'" round class="cc-start-btn" @click.stop="stopScheme(s)">
                停止
              </el-button>
              <el-button class="cc-del-btn" round @click.stop="removeScheme(s)" aria-label="删除">
                <span class="cc-ms cc-ms--sm" aria-hidden="true">delete</span>
              </el-button>
            </div>
            <div class="cc-foot-right" @click.stop>
              <span class="cc-sim-lbl">模拟投注</span>
              <el-switch :model-value="s.simBet" :disabled="simBetSavingId === s.id || s.status === 'running'"
                @update:model-value="(v: boolean) => toggleSimBet(s, v)" />
            </div>
          </div>
        </div>

        <div v-if="listHasMore && displayedSchemes.length > 0 && !schemeSearchKeyword" ref="loadMoreSentinel"
          class="cc-list-sentinel" aria-hidden="true" />
        <p v-if="listLoadingMore" class="cc-list-more">加载中…</p>
        <p v-else-if="!listHasMore && runningSchemes.length > 0 && !schemeSearchKeyword"
          class="cc-list-more cc-list-more--done">
          已加载全部方案
        </p>
      </section>
    </main>

    <el-dialog v-model="searchDialogVisible" title="搜索方案" width="min(22rem, 88vw)" align-center append-to-body
      class="cc-search-dialog">
      <el-input v-model="searchDraft" clearable size="large" placeholder="方案名称、彩种或方案 ID"
        @keyup.enter="applySchemeSearch" />
      <template #footer>
        <el-button @click="searchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="applySchemeSearch">搜索</el-button>
      </template>
    </el-dialog>

    <ContentDialog v-model="multiplierDialogVisible" title="修改倍数系数" icon="tune" confirm-text="确定" show-cancel
      :confirm-loading="multiplierSavingId !== null" :auto-close-on-confirm="false" @confirm="confirmMultiplierDialog"
      @cancel="cancelMultiplierDialog">
      <el-input :model-value="multiplierDraft" inputmode="numeric" maxlength="6" size="large" placeholder="请输入正整数"
        class="cc-mult-inp" @update:model-value="onMultiplierDraftChange" @keyup.enter="confirmMultiplierDialog" />
    </ContentDialog>

    <el-dialog v-model="lookbackDialogVisible" class="cc-lookback-dialog" width="min(32rem, 92vw)" align-center
      destroy-on-close :show-close="false" append-to-body>
      <template #header>
        <div class="lb-dlg-head">
          <h2 class="lb-dlg-title">回头设置</h2>
          <button type="button" class="lb-dlg-close" aria-label="关闭" @click="cancelLookback">
            <span class="cc-ms lb-dlg-close-ico" aria-hidden="true">close</span>
          </button>
        </div>
      </template>

      <div class="lb-body">
        <!-- 运行模式选择 -->
        <section class="lb-section">
          <div class="lb-section-head">
            <span class="cc-ms lb-section-ico" aria-hidden="true">play_arrow</span>
            <span class="lb-section-title">运行模式选择</span>
          </div>
          <div class="lb-run-grid" role="group" aria-label="运行模式（可多选）">
            <label class="lb-run-opt" :class="{ 'is-active': lookback.runModeSim }"
              @click.prevent="toggleLookbackRunMode('sim')">
              <input type="checkbox" class="lb-sr-only" tabindex="-1" :checked="lookback.runModeSim"
                aria-hidden="true" />
              <span class="lb-run-card">模拟运行</span>
            </label>
            <label class="lb-run-opt" :class="{ 'is-active': lookback.runModeProd }"
              @click.prevent="toggleLookbackRunMode('prod')">
              <input type="checkbox" class="lb-sr-only" tabindex="-1" :checked="lookback.runModeProd"
                aria-hidden="true" />
              <span class="lb-run-card">正式运行</span>
            </label>
          </div>
        </section>

        <!-- 回头条件逻辑配置 -->
        <section class="lb-section lb-section--logic">
          <div class="lb-section-head">
            <span class="cc-ms lb-section-ico" aria-hidden="true">settings_suggest</span>
            <span class="lb-section-title">回头条件逻辑配置</span>
          </div>

          <div class="lb-judge-list">
            <!-- 个别判断：可与整体判断同时不选；勾选时互斥 -->
            <div class="lb-judge-section">
              <label class="lb-judge-top" :class="{ 'is-active': lookback.judgment === 'individual' }"
                @click.prevent="toggleLookbackJudgment('individual')">
                <input type="checkbox" class="lb-sr-only" tabindex="-1" :checked="lookback.judgment === 'individual'"
                  aria-hidden="true" />
                <span class="lb-judge-check" aria-hidden="true" />
                <span class="lb-judge-label-txt">个别判断</span>
              </label>
              <div class="lb-judge-indent" :class="{ 'lb-judge-panel--inactive': lookback.judgment !== 'individual' }">
                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">单方案盈亏回头</div>
                  <div class="lb-row2">
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-sp">盈利阈值</label>
                      <el-input id="lb-sp" v-model="lookback.singleProfitThreshold" type="number" size="small"
                        class="lb-inp" :disabled="lookback.judgment !== 'individual'" />
                    </div>
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-sl">亏损阈值</label>
                      <el-input id="lb-sl" v-model="lookback.singleLossThreshold" type="number" size="small"
                        class="lb-inp" placeholder="0.00" :disabled="lookback.judgment !== 'individual'" />
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 整体判断：可与个别判断同时不选；勾选时互斥 -->
            <div class="lb-judge-section">
              <label class="lb-judge-top" :class="{ 'is-active': lookback.judgment === 'overall' }"
                @click.prevent="toggleLookbackJudgment('overall')">
                <input type="checkbox" class="lb-sr-only" tabindex="-1" :checked="lookback.judgment === 'overall'"
                  aria-hidden="true" />
                <span class="lb-judge-check" aria-hidden="true" />
                <span class="lb-judge-label-txt">整体判断</span>
              </label>
              <div class="lb-judge-indent" :class="{ 'lb-judge-panel--inactive': lookback.judgment !== 'overall' }">
                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">整体盈亏回头</div>
                  <div class="lb-row2">
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-op">盈利阈值</label>
                      <el-input id="lb-op" v-model="lookback.overallProfitThreshold" type="number" size="small"
                        class="lb-inp" placeholder="盈利阈值" :disabled="lookback.judgment !== 'overall'" />
                    </div>
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-ol">亏损阈值</label>
                      <el-input id="lb-ol" v-model="lookback.overallLossThreshold" type="number" size="small"
                        class="lb-inp" placeholder="亏损阈值" :disabled="lookback.judgment !== 'overall'" />
                    </div>
                  </div>
                </div>

                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">方案中局几回头</div>
                  <div class="lb-wins-inline">
                    <span class="lb-wins-txt lb-wins-op">{{ '>=' }}</span>
                    <el-input v-model="lookback.schemeWinsMin" inputmode="numeric" size="small"
                      class="lb-inp lb-inp--wins" placeholder="最小" :disabled="lookback.judgment !== 'overall'"
                      @update:model-value="onSchemeWinsMinChange" />
                    <span class="lb-wins-txt lb-wins-op">{{ '<=' }}</span>
                        <el-input v-model="lookback.schemeWinsMax" inputmode="numeric" size="small"
                          class="lb-inp lb-inp--wins" placeholder="最大" :disabled="lookback.judgment !== 'overall'"
                          @update:model-value="onSchemeWinsMaxChange" />
                        <span class="lb-wins-txt">次即回头</span>
                  </div>
                </div>

                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">单期盈亏回头</div>
                  <div class="lb-row2">
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-pp">盈利</label>
                      <el-input id="lb-pp" v-model="lookback.periodProfit" type="number" size="small" class="lb-inp"
                        placeholder="0.00" :disabled="lookback.judgment !== 'overall'" />
                    </div>
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-ploss">亏损</label>
                      <el-input id="lb-ploss" v-model="lookback.periodLoss" type="number" size="small" class="lb-inp"
                        placeholder="0.00" :disabled="lookback.judgment !== 'overall'" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <div class="lb-alert" role="alert">
          <span class="cc-ms lb-alert-ico" aria-hidden="true">error_outline</span>
          <p class="lb-alert-txt">
            注意：配置变更将重置相关方案的所有递增步长。请确保阈值设置合理，避免频繁重置影响最终收益。
          </p>
        </div>
      </div>

      <template #footer>
        <div class="lb-footer">
          <button type="button" class="lb-footer-cancel" @click="cancelLookback">取消</button>
          <el-button type="primary" class="lb-footer-save" size="large" round @click="confirmLookback">
            确认并保存设置
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.cc {
  --cc-primary: #0050cb;
  --cc-primary-strong: #0066ff;
  --cc-surface: #f7f9fb;
  --cc-card: #ffffff;
  --cc-on: #191c1e;
  --cc-on-var: #424656;
  --cc-container: #f1f5f9;
  --cc-variant: #f8fafc;
  --cc-outline: rgba(226, 232, 240, 0.85);
  --cc-error: #ba1a1a;
  min-height: 100dvh;
  background: var(--cc-surface);
  color: var(--cc-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: calc(5.5rem + env(safe-area-inset-bottom));
  -webkit-font-smoothing: antialiased;
}

.cc-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.375rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.cc-ms--sm {
  font-size: 1.125rem;
}

/* ===== Header ===== */
.cc-head {
  background: linear-gradient(180deg, var(--cc-primary-strong) 0%, var(--cc-primary) 100%);
  color: #fff;
  padding: max(1.75rem, env(safe-area-inset-top)) 1.25rem 3.75rem;
  border-radius: 0 0 2rem 2rem;
  box-shadow: 0 20px 40px -24px rgba(0, 80, 203, 0.45);
}

.cc-head-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.cc-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.cc-head-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-icon-btn {
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  border: none;
  border-radius: 0.75rem;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.cc-icon-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.cc-icon-btn .cc-ms {
  font-size: var(--page-titlebar-icon-size);
}

.cc-stats {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: stretch;
  gap: 0;
}

.cc-stat-col {
  padding: 0 0.35rem;
  text-align: center;
}

.cc-stat-h {
  margin: 0 0 0.75rem;
  font-size: 0.8125rem;
  font-weight: 700;
}

.cc-stat-rows {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.cc-stat-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.6875rem;
  opacity: 0.92;
  padding: 0 0.35rem;
}

.cc-stat-em {
  font-weight: 700;
  opacity: 1;
}

.cc-stat-em--loss {
  color: #ffb4ab;
}

.cc-stat-row--pill {
  font-weight: 700;
  opacity: 1;
  background: rgba(255, 255, 255, 0.12);
  border-radius: 0.5rem;
  padding: 0.4rem 0.45rem;
  margin-top: 0.15rem;
}

.cc-stat-divider {
  width: 1px;
  align-self: stretch;
  min-height: 6rem;
  background: linear-gradient(180deg,
      transparent,
      rgba(255, 255, 255, 0.22) 15%,
      rgba(255, 255, 255, 0.22) 85%,
      transparent);
}

/* ===== Main ===== */
.cc-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 0 1.15rem 2rem;
  margin-top: -1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}

.cc-panel {
  background: var(--cc-card);
  border-radius: 1rem;
  padding: 1rem;
  box-shadow: 0 24px 48px -28px rgba(15, 23, 42, 0.12), 0 4px 16px -8px rgba(15, 23, 42, 0.06);
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.cc-panel-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.65rem;
}

.cc-field {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.cc-lbl {
  font-size: 0.625rem;
  font-weight: 700;
  color: var(--cc-on-var);
  display: inline-flex;
  align-items: center;
  gap: 0.15rem;
  line-height: 1.3;
}

.cc-lbl-ico {
  font-size: 0.75rem;
  opacity: 0.85;
}

.cc-mult-wrap {
  display: flex;
  align-items: stretch;
  gap: 0.5rem;
}

.cc-mult-prefix {
  flex-shrink: 0;
  min-width: 2.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 0.75rem;
  font-weight: 700;
  border-radius: 0.5rem;
  font-family: 'Noto Sans SC', sans-serif;
}

.cc-panel :deep(.el-input__wrapper) {
  min-height: 2rem;
  padding-top: 0;
  padding-bottom: 0;
}

.cc-panel :deep(.el-input__inner) {
  height: 2rem;
  line-height: 2rem;
  font-size: 0.8125rem;
}

.cc-el-inp--grow {
  flex: 1;
  min-width: 0;
}

.cc-el-inp--w {
  width: 5rem;
  max-width: 40vw;
}

.cc-row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.cc-hint {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.6875rem;
  color: var(--cc-on-var);
  min-width: 0;
  line-height: 1.3;
}

.cc-hint-ico {
  font-size: 1rem;
  color: var(--cc-primary-strong);
}

.cc-hint-txt {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cc-hint--switch {
  margin: 0.35rem 0 0;
  line-height: 1.5;
  white-space: normal;
}

.cc-switch-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-switch-lbl {
  font-size: 0.6875rem;
  color: var(--cc-on-var);
}

.cc-actions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.4rem;
  align-items: stretch;
  margin-top: 0.1rem;
}

.cc-btn {
  font-size: 0.625rem;
  font-weight: 700;
  padding: 0.45rem 0.2rem;
  height: 2rem;
  margin: 0;
  width: 100%;
  min-width: 0;
  border: none;
  white-space: nowrap;
}

.cc-actions :deep(.el-button > span) {
  white-space: nowrap;
}

.cc-btn--primary {
  box-shadow: 0 8px 20px -8px rgba(0, 80, 203, 0.45);
}

.cc-btn--outline {
  background: #fff;
  color: var(--el-color-primary);
  border: 1px solid rgba(0, 80, 203, 0.35);
}

.cc-btn--outline:hover {
  background: rgba(0, 102, 255, 0.06);
}

/* ===== List ===== */
.cc-list-sec {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cc-list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 0 0.15rem;
}

.cc-list-h2 {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: -0.01em;
}

.cc-list-meta {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--cc-on-var);
}

.cc-search-clear {
  margin-left: 0.35rem;
  padding: 0;
  border: none;
  background: none;
  color: var(--cc-primary, #0066ff);
  font-size: inherit;
  font-weight: 700;
  cursor: pointer;
}

.cc-list-empty {
  margin: 0.5rem 0 1rem;
  padding: 1.25rem;
  text-align: center;
  font-size: 0.875rem;
  color: var(--cc-on-var);
  background: var(--cc-card);
  border-radius: 1rem;
}

.cc-list-sentinel {
  width: 100%;
  height: 1px;
}

.cc-list-more {
  margin: 0.75rem 0 1rem;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--cc-on-var);
}

.cc-list-more--done {
  opacity: 0.72;
}

.cc-card {
  min-width: 0;
  background: var(--cc-card);
  border-radius: 1.25rem;
  padding: 1.25rem;
  box-shadow: 0 12px 32px -20px rgba(15, 23, 42, 0.14);
}

.cc-card-hd {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0.45rem;
  margin-bottom: 1.15rem;
}

.cc-card-title-row {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
  width: 100%;
}

.cc-card-hd>.cc-badge {
  align-self: flex-end;
  max-width: 100%;
}

.cc-card--clickable {
  cursor: pointer;
  transition: box-shadow 0.2s ease, transform 0.2s ease;
}

.cc-card--clickable:hover {
  box-shadow: 0 18px 40px -24px rgba(0, 80, 203, 0.35);
}

.cc-card--running {
  background:
    linear-gradient(145deg, rgba(0, 102, 255, 0.1) 0%, rgba(0, 80, 203, 0.04) 52%, #ffffff 100%);
  box-shadow: 0 16px 36px -18px rgba(0, 80, 203, 0.28);
}

.cc-card--running:hover {
  box-shadow: 0 20px 44px -20px rgba(0, 80, 203, 0.38);
}

.cc-card--running .cc-card-h3 {
  color: var(--cc-primary);
}

.cc-card--running .cc-v--primary {
  color: var(--cc-primary-strong);
  font-weight: 700;
}

.cc-card-title-arrow {
  font-size: 1.05rem;
  color: #94a3b8;
}

.cc-card-h3 {
  margin: 0;
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.9375rem;
  font-weight: 800;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
}

.cc-runtype-tag {
  flex-shrink: 0;
  font-size: 11px;
}

.cc-badge {
  flex-shrink: 0;
  max-width: 100%;
  text-align: right;
  line-height: 1.35;
  word-break: break-word;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--cc-on-var);
  background: var(--cc-variant);
  padding: 0.2rem 0.45rem;
  border-radius: 0.35rem;
}

.cc-badge--warn {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.cc-badge--muted {
  color: #64748b;
  background: rgba(100, 116, 139, 0.12);
}

.cc-badge--info {
  color: #0050cb;
  background: rgba(0, 102, 255, 0.1);
}

.cc-badge--active {
  color: #047857;
  background: rgba(16, 185, 129, 0.12);
}

.cc-kv-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0.65rem 1.25rem;
  min-width: 0;
}

.cc-kv {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  padding-bottom: 0.35rem;
  border-bottom: 1px solid rgba(226, 232, 240, 0.65);
}

.cc-k {
  flex: 0 0 auto;
  font-size: 0.75rem;
  color: var(--cc-on-var);
}

.cc-v {
  flex: 1 1 auto;
  font-size: 0.75rem;
  font-weight: 600;
  text-align: right;
  min-width: 0;
}

.cc-v--ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cc-v--primary {
  color: var(--cc-primary-strong);
  font-weight: 800;
}

.cc-v--error {
  color: var(--cc-error);
  font-weight: 800;
}

.cc-v--mono {
  font-family: Inter, ui-monospace, monospace;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.cc-kv--last {
  border-bottom: none;
  padding-bottom: 0;
}

.cc-kv--mult {
  justify-content: space-between;
}

.cc-el-inp--mult {
  width: 3.25rem;
}

.cc-el-inp--mult-trigger :deep(.el-input__wrapper) {
  cursor: pointer;
}

.cc-el-inp--mult-trigger :deep(.el-input__inner) {
  cursor: pointer;
  text-align: center;
}

.cc-mult-inp {
  width: 100%;
}

.cc-mult-inp :deep(.el-input__inner) {
  text-align: center;
  font-variant-numeric: tabular-nums;
}

.cc-mult-dialog-hint {
  margin: 0 0 0.75rem;
  font-size: 0.8125rem;
  color: var(--cc-on-var, #424656);
  line-height: 1.5;
}

.cc-card-foot {
  margin-top: 0.85rem;
  padding-top: 1.1rem;
  border-top: 1px solid rgba(226, 232, 240, 0.65);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.cc-foot-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-start-btn {
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.55rem 1.1rem;
}

.cc-del-btn {
  padding: 0.5rem 0.65rem;
  min-height: auto;
  color: var(--cc-on-var);
  background: var(--cc-variant);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

.cc-foot-right {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.cc-sim-lbl {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--cc-on-var);
}
</style>

<style>
/* 对话框：覆盖 Element Plus，对齐「数字精算主义」与运营中心回头弹窗设计 */
.cc-lookback-dialog.el-dialog {
  border-radius: 1.5rem;
  overflow: hidden;
  padding: 0;
  box-shadow: 0 20px 50px rgba(0, 80, 203, 0.15);
}

.cc-lookback-dialog .el-dialog__header {
  margin: 0;
  padding: 0;
  border-bottom: none;
}

.cc-lookback-dialog .el-dialog__body {
  padding: 0 1.5rem 1.25rem;
  max-height: min(75dvh, 34rem);
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.cc-lookback-dialog .el-dialog__body::-webkit-scrollbar {
  display: none;
}

.cc-lookback-dialog .el-dialog__footer {
  margin: 0;
  padding: 0;
  border-top: none;
}

.lb-dlg-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem 1.5rem 0.75rem;
}

.lb-dlg-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #191c1e;
}

.lb-dlg-close {
  width: 2rem;
  height: 2rem;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: #727687;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.lb-dlg-close:hover {
  background: #e6e8ea;
}

.lb-dlg-close-ico {
  font-size: 1.25rem;
}

.lb-body {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  min-width: 0;
}

.lb-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-width: 0;
}

.lb-section-head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.lb-section-ico {
  font-size: 1.25rem;
  color: #0050cb;
}

.lb-section-title {
  font-size: 0.875rem;
  font-weight: 700;
  color: #191c1e;
}

.lb-section--logic {
  gap: 1rem;
}

.lb-judge-list {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 1.25rem;
  width: 100%;
}

.lb-judge-section {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  min-width: 0;
}

.lb-judge-top {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  width: fit-content;
  cursor: pointer;
  user-select: none;
}

.lb-judge-check {
  width: 1.125rem;
  height: 1.125rem;
  border-radius: 0.25rem;
  border: 1.5px solid #c4c7cf;
  background: #fff;
  flex-shrink: 0;
  transition:
    border-color 0.15s ease,
    background 0.15s ease,
    box-shadow 0.15s ease;
}

.lb-judge-top.is-active .lb-judge-check {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: inset 0 0 0 2px #fff;
}

.lb-judge-top.is-active .lb-judge-label-txt {
  color: var(--el-color-primary);
}

.lb-judge-label-txt {
  font-size: 0.9375rem;
  font-weight: 700;
  color: #191c1e;
  line-height: 1.3;
}

.lb-judge-indent {
  padding-left: 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-width: 0;
  transition:
    opacity 0.25s ease,
    filter 0.25s ease;
}

.lb-judge-panel--inactive {
  opacity: 0.4;
  pointer-events: none;
  filter: grayscale(0.45);
}

.lb-wins-inline {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.3rem;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  transition: opacity 0.15s;
}

.lb-wins-txt {
  font-size: 0.75rem;
  color: #424656;
  flex-shrink: 0;
}

.lb-wins-op {
  font-family: Inter, ui-monospace, monospace;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.lb-inp--wins .el-input__inner {
  padding-left: 0.25rem;
  padding-right: 0.25rem;
}

.lb-inp--wins .el-input__wrapper {
  justify-content: center;
  padding-left: 0.15rem;
  padding-right: 0.15rem;
  min-width: 0;
}

.lb-run-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.lb-run-opt {
  cursor: pointer;
  margin: 0;
}

.lb-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.lb-run-card {
  display: block;
  text-align: center;
  padding: 0.75rem 0.5rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.85);
  background: #f2f4f6;
  font-size: 0.875rem;
  font-weight: 600;
  color: #424656;
  transition:
    border-color 0.15s,
    background 0.15s,
    color 0.15s;
}

.lb-run-opt.is-active .lb-run-card {
  border-color: #0050cb;
  background: rgba(0, 102, 255, 0.08);
  color: #0050cb;
}

.lb-logic-card {
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.55);
  background: rgba(242, 244, 246, 0.45);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
}

.lb-logic-card-h {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: #191c1e;
  line-height: 1.35;
}

.lb-row2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  transition: opacity 0.15s;
}

.lb-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.lb-field-lbl {
  font-size: 0.625rem;
  font-weight: 700;
  color: #727687;
  letter-spacing: 0.02em;
  padding-left: 0.15rem;
}

.lb-inp.el-input {
  width: 100%;
}

.lb-inp.el-input.lb-inp--wins {
  width: auto;
  flex: 0 1 2.125rem;
  min-width: 1.75rem;
  max-width: 2.625rem;
}

.lb-alert {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  padding: 0.75rem;
  border-radius: 0.75rem;
  background: rgba(255, 218, 214, 0.22);
  border: 1px solid rgba(186, 26, 26, 0.14);
}

.lb-alert-ico {
  font-size: 1.125rem;
  color: #ba1a1a;
  flex-shrink: 0;
  margin-top: 0.05rem;
}

.lb-alert-txt {
  margin: 0;
  font-size: 0.6875rem;
  line-height: 1.65;
  font-weight: 500;
  color: #424656;
}

.lb-footer {
  display: flex;
  align-items: stretch;
  gap: 0.75rem;
  width: 100%;
  padding: 1rem 1.5rem calc(1rem + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid rgba(194, 198, 216, 0.25);
}

.lb-footer-cancel {
  flex: 1;
  border: none;
  border-radius: 0.75rem;
  background: transparent;
  font-size: 0.875rem;
  font-weight: 700;
  color: #424656;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s;
}

.lb-footer-cancel:hover {
  background: #f2f4f6;
}

.lb-footer-save {
  flex: 2;
  margin: 0;
  border: none;
  font-weight: 700;
  box-shadow: 0 8px 16px rgba(0, 80, 203, 0.25);
  background: linear-gradient(180deg, #0066ff 0%, #0050cb 100%);
}
</style>
