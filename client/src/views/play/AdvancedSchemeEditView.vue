<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog } from '@/utils/confirmDialog'
import { ApiError } from '@/api/client'
import { addSchemeToCloud, forkSchemeToCloud } from '@/api/schemes/addToCloud'
import {
  createScheme,
  fetchSchemeDefinitions,
  updateSchemeDefinition,
  deleteSchemeDefinition,
} from '@/api/schemes/definitions'
import type {
  SchemeJushuRow,
  SchemeTriggerBet,
  SchemeTriggerRow,
  SchemeHotColdWarm,
  SchemeRandomDraw,
  UpdateSchemeInput,
} from '@/api/schemes/definitions'
import { fetchSchemeFavorites, type SchemeFavoriteRow } from '@/api/schemes/favorites'
import { fetchGameDraws } from '@/api/games/detail'
import { parseSchemeKind } from '@/utils/schemeKind'
import DateTimePickerModal from '@/components/ui/DateTimePickerModal.vue'
import { BET_MODE_OPTIONS, betUnitFromSchemeConfig, normalizeBetUnitValue } from '@/constants/betModeOptions'
import SchemeGroupPickPanel from '@/components/schemes/SchemeGroupPickPanel.vue'
import {
  catalogFieldsFromPlayConfig,
  groupContentPlaceholder,
  playConfigSummary,
  validateGroupContent,
  validateSchemeGroups,
} from '@/utils/betPayload'
import { normalizeSchemeTimePairFromConfig, schemeTimeRangeError } from '@/utils/schemeDateTime'
import { usePublicLotteries } from '@/composables/usePublicLotteries'
import { usePlayTreeConfig } from '@/composables/usePlayTreeConfig'
import { longhuPickOptionsForConfig } from '@/utils/longhuPickOptions'
import { schemeGroupUsesPickPanel, textPickOptionsForConfig } from '@/utils/pickPanelOptions'
import {
  isLonghuPlayConfigLike,
  isPc28HezhiConfigLike,
  isPc28ModeConfigLike,
} from '@/utils/runTypeMatrix'
import {
  clearSchemeDraft,
  consumeSchemeEditRestoreSnapshot,
  draftMetaFromQuery,
  draftPatchFromSnapshot,
  isDraftSchemeId,
  loadSchemeDraft,
  saveSchemeDraft,
  saveSchemeEditRestoreSnapshot,
  type SchemeDraftSnapshot,
} from '@/utils/schemeDraftStorage'
import { syncDraftAdvancedTemplatesToServer } from '@/utils/draftAdvancedTemplates'
import { simBetFromSchemeConfig } from '@/utils/schemeSimBet'

const route = useRoute()
const router = useRouter()
const { load: loadLotteries, codeToLabel } = usePublicLotteries()

const schemeId = computed(() => String(route.params.schemeId ?? ''))
const isDraftScheme = computed(() => isDraftSchemeId(schemeId.value) || route.query.draft === '1')
const schemeKind = computed(() =>
  parseSchemeKind(route.query.kind ?? route.query.schemeKind),
)
const isCustomKind = computed(() => schemeKind.value === 'custom')

const BACK_ICO = '/images/lobby/icon-back.png'

/** false=正式运行，true=模拟运行 */
const simBet = ref(false)
const schemeName = ref(decodeURIComponent(String(route.query.title ?? '') || '精算方案-A01'))
const shareStatus = ref<'private' | 'public'>('private')
const shareLocked = ref(false)
const cloudBusy = ref(false)
const schemeFunds = ref('10000')
/** 开始/结束时间；两者均留空表示无限期运行 */
const startTime = ref('')
const endTime = ref('')
const stopLoss = ref('')
const takeProfit = ref('')
const multCoeff = ref('1')
const betUnit = ref('2')
/** 方案内容按组划分，默认一组 */
const schemeGroups = ref<string[]>([''])

const lotteryCode = ref(String(route.query.lottery ?? 'tron_ffc_1m'))
const playTypeId = ref(String(route.query.playType ?? route.query.playTypeId ?? ''))
const subPlayId = ref(String(route.query.subPlay ?? route.query.subPlayId ?? ''))

const { playConfig: schemePlayConfig, load: loadPlayTree } = usePlayTreeConfig(
  lotteryCode,
  playTypeId,
  subPlayId,
)

const playModeSummary = computed(() => playConfigSummary(schemePlayConfig.value))

/** 当前玩法用 chip 选号时不再展示 textarea（避免双轨编辑） */
const schemeUsesPickPanel = computed(() => schemeGroupUsesPickPanel(schemePlayConfig.value))

const groupInputPlaceholder = computed(() => groupContentPlaceholder(schemePlayConfig.value))

const gameNameDisplay = computed(() => {
  const id = String(route.query.lottery ?? '')
  const label = codeToLabel(id)
  if (label) return label
  return id || '—'
})

function groupBetUnits(raw: string): number {
  const r = validateGroupContent(schemePlayConfig.value, raw)
  return r.ok ? r.betUnits : 0
}

// ----- 运行类型（runTypeId）与七套方案内容面板 -----
const RUN_TYPE_IDS = [
  'fixed_rotate',
  'adv_fixed_rotate',
  'adv_trigger_bet',
  'hot_cold_warm',
  'random_draw',
  'builtin_plan',
  'fixed_number',
] as const
type RunTypeId = (typeof RUN_TYPE_IDS)[number]

const RUN_TYPE_LABELS: Record<RunTypeId, string> = {
  fixed_rotate: '定码轮换',
  adv_fixed_rotate: '高级定码轮换',
  adv_trigger_bet: '高级开某投某',
  hot_cold_warm: '冷热温出号',
  random_draw: '随机出号',
  builtin_plan: '内置计画',
  fixed_number: '固定号码',
}

/** batch_fixed / dynamic_chase / plan_follow 等废弃或未知值统一兜底为高级定码轮换 */
function normalizeRunTypeId(raw: unknown): RunTypeId {
  const v = String(Array.isArray(raw) ? raw[0] ?? '' : raw ?? '').trim()
  if ((RUN_TYPE_IDS as readonly string[]).includes(v)) return v as RunTypeId
  return 'adv_fixed_rotate'
}

/** 远端 def.config.runTypeId 为准，路由 query.runType 兜底 */
const runTypeId = ref<RunTypeId>(normalizeRunTypeId(route.query.runType))
const runTypeLabel = computed(() => RUN_TYPE_LABELS[runTypeId.value])

const POSITION_FALLBACK_LABELS = ['万位', '千位', '百位', '十位', '个位']
const ALL_DIGITS = Array.from({ length: 10 }, (_, i) => String(i))

/** 玩法号码池（P4 全模板）：PK10 1-10、11选5 01-11、K3 1-6、六合彩 1-49，缺省 0-9 */
const numberPoolTokens = computed<string[]>(() => {
  const min = schemePlayConfig.value.numberPoolMin
  const max = schemePlayConfig.value.numberPoolMax
  if (min != null && max != null && max >= min && (max > 9 || min > 0)) {
    const pad = max >= 11
    return Array.from({ length: max - min + 1 }, (_, i) => {
      const n = min + i
      return pad ? String(n).padStart(2, '0') : String(n)
    })
  }
  return [...ALL_DIGITS]
})

/** 把开奖球/输入值归一化为号码池 token（兼容 '07' 与 '7'） */
function normalizePoolToken(raw: string): string {
  const v = String(raw ?? '').trim()
  if (!/^\d{1,2}$/.test(v)) return ''
  const n = Number(v)
  for (const tk of numberPoolTokens.value) {
    if (Number(tk) === n) return tk
  }
  return ''
}

/** 玩法位数：定胆等单位玩法 = 1 位 */
const positionCount = computed(() => Math.max(1, schemePlayConfig.value.segmentLen || 1))
const positionLabels = computed(() =>
  Array.from(
    { length: positionCount.value },
    (_, i) => schemePlayConfig.value.segmentLabels[i] ?? POSITION_FALLBACK_LABELS[i] ?? `第 ${i + 1} 位`,
  ),
)
const isLonghuPlay = computed(() => isLonghuPlayConfigLike(schemePlayConfig.value))

function isPc28PlayLine(): boolean {
  return isPc28ModeConfigLike(schemePlayConfig.value)
}

/** 固定号码：仅展示并保存第 1 组 */
const displayedGroupIndexes = computed(() =>
  runTypeId.value === 'fixed_number' ? [0] : schemeGroups.value.map((_, i) => i),
)

// --- adv_fixed_rotate 高级定码轮换：局数列表 ---
const jushuList = ref<SchemeJushuRow[]>([])
const jushuDialogVisible = ref(false)
const jushuForm = ref<SchemeJushuRow>({ ju: 1, content: '', afterHit: 1, afterMiss: 1 })

function applyJushuFromConfig(raw: unknown): boolean {
  if (!Array.isArray(raw) || !raw.length) return false
  const rows: SchemeJushuRow[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const r = item as Record<string, unknown>
    const ju = Math.trunc(Number(r.ju))
    if (!Number.isInteger(ju) || ju < 1) continue
    rows.push({
      ju,
      content: String(r.content ?? ''),
      afterHit: Math.max(1, Math.trunc(Number(r.afterHit)) || 1),
      afterMiss: Math.max(1, Math.trunc(Number(r.afterMiss)) || 1),
    })
  }
  if (!rows.length) return false
  rows.sort((a, b) => a.ju - b.ju)
  jushuList.value = rows
  return true
}

/** 远端无 jushuList 时由现有 schemeGroups 生成预览行（局 i+1 = 第 i 组），保存后落 jushuList */
function seedJushuFromGroups(): void {
  const groups = schemeGroups.value.map((g) => g.trim()).filter(Boolean)
  if (!groups.length) return
  jushuList.value = groups.map((content, i) => ({ ju: i + 1, content, afterHit: 1, afterMiss: 1 }))
}

function openJushuDialog(): void {
  const maxJu = jushuList.value.reduce((m, r) => Math.max(m, r.ju), 0)
  jushuForm.value = { ju: maxJu + 1, content: '', afterHit: 1, afterMiss: 1 }
  jushuDialogVisible.value = true
}

function confirmJushuDialog(): void {
  const f = jushuForm.value
  if (!Number.isInteger(f.ju) || f.ju < 1) {
    ElMessage.warning('局数须为不小于 1 的整数')
    return
  }
  if (jushuList.value.some((r) => r.ju === f.ju)) {
    ElMessage.warning(`第 ${f.ju} 局已存在，局数不可重复`)
    return
  }
  const content = f.content.trim()
  if (!content) {
    ElMessage.warning('投注号码不能为空')
    return
  }
  jushuList.value = [
    ...jushuList.value,
    { ju: f.ju, content, afterHit: Math.max(1, f.afterHit), afterMiss: Math.max(1, f.afterMiss) },
  ].sort((a, b) => a.ju - b.ju)
  jushuDialogVisible.value = false
}

function removeJushuRow(idx: number): void {
  jushuList.value.splice(idx, 1)
}

// --- adv_trigger_bet 高级开某投某 ---
const PC28_HEZHI_VALUES = Array.from({ length: 28 }, (_, i) => String(i))
const longhuPickValues = computed(() => longhuPickOptionsForConfig(schemePlayConfig.value))
const triggerRows = ref<SchemeTriggerRow[]>([])
const triggerMode = ref<SchemeTriggerBet['mode']>('always_pos')
/** 远端已有配置时不随玩法解析过程重建行 */
let triggerRowsLocked = false
let lastTriggerPlayKey = ''

/** 玩法切换时允许重建开某投某映射行（保留远端已加载配置） */
watch(
  [
    () => schemePlayConfig.value.playTypeId,
    () => schemePlayConfig.value.catalogSubId,
    () => schemePlayConfig.value.subPlayId,
  ],
  () => {
    const key = `${schemePlayConfig.value.playTypeId}:${schemePlayConfig.value.catalogSubId ?? schemePlayConfig.value.subPlayId}`
    if (key !== lastTriggerPlayKey) {
      lastTriggerPlayKey = key
      triggerRowsLocked = false
    }
  },
)

const TRIGGER_MODE_OPTIONS = [
  { label: '一直正投', value: 'always_pos' },
  { label: '一直反投', value: 'always_neg' },
  { label: '前正后反', value: 'alt_pos_first' },
  { label: '前反后正', value: 'alt_neg_first' },
] as const

/** 正投/反投下拉选项（龙虎、PC28 大小单双/龙虎豹） */
const triggerBetOptions = computed<string[]>(() => {
  if (isLonghuPlay.value) return longhuPickValues.value
  const opts = textPickOptionsForConfig(schemePlayConfig.value)
  return opts.length ? opts : []
})

const isTriggerTextPlay = computed(() => triggerBetOptions.value.length > 0)

function triggerOpenValues(): string[] {
  if (isLonghuPlay.value) return longhuPickValues.value
  if (isPc28HezhiConfigLike(schemePlayConfig.value) && isPc28PlayLine()) {
    return [...PC28_HEZHI_VALUES]
  }
  const bm = schemePlayConfig.value.betMode ?? ''
  if (bm === 'hezhi' && isPc28PlayLine()) {
    return [...PC28_HEZHI_VALUES]
  }
  const textOpts = textPickOptionsForConfig(schemePlayConfig.value)
  if (textOpts.length) return textOpts
  return [...numberPoolTokens.value]
}

function ensureTriggerRows(): void {
  if (triggerRowsLocked && triggerRows.value.length) return
  const opens = triggerOpenValues()
  const cur = triggerRows.value
  if (cur.length === opens.length && cur.every((r, i) => r.open === opens[i])) return
  triggerRows.value = opens.map((open) => ({ enabled: true, open, pos: '', neg: '' }))
}

function applyTriggerBetFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const tb = raw as { rows?: unknown; mode?: unknown }
  if (Array.isArray(tb.rows) && tb.rows.length) {
    const rows: SchemeTriggerRow[] = []
    for (const item of tb.rows) {
      if (!item || typeof item !== 'object') continue
      const r = item as Record<string, unknown>
      rows.push({
        enabled: r.enabled !== false,
        open: String(r.open ?? ''),
        pos: String(r.pos ?? ''),
        neg: String(r.neg ?? ''),
      })
    }
    if (rows.length) {
      triggerRows.value = rows
      triggerRowsLocked = true
    }
  }
  const mode = String(tb.mode ?? '')
  if (mode === 'always_pos' || mode === 'always_neg' || mode === 'alt_pos_first' || mode === 'alt_neg_first') {
    triggerMode.value = mode
  }
}

function randomTriggerValue(): string {
  const pool = triggerOpenValues()
  return pool[Math.floor(Math.random() * pool.length)] ?? '0'
}

/** 「全部随机」：纯前端一次性填表辅助，引擎下注不涉及随机 */
function randomFillTrigger(): void {
  for (const row of triggerRows.value) {
    row.pos = randomTriggerValue()
    row.neg = randomTriggerValue()
  }
  ElMessage.success('已随机填充正投 / 反投号码')
}

function sanitizeTriggerDigit(v: string): string {
  const digits = String(v ?? '').replace(/\D/g, '').slice(0, 2)
  if (!digits) return ''
  const cfg = schemePlayConfig.value
  if (isPc28HezhiConfigLike(cfg) && isPc28PlayLine()) {
    const n = Math.min(27, Number(digits))
    return Number.isFinite(n) ? String(n) : ''
  }
  return normalizePoolToken(digits) || digits.slice(0, 1)
}

const triggerInputPlaceholder = computed(() => {
  if (isPc28HezhiConfigLike(schemePlayConfig.value) && isPc28PlayLine()) {
    return '0-27'
  }
  const bm = schemePlayConfig.value.betMode ?? ''
  if (bm === 'hezhi' && isPc28PlayLine()) {
    return '0-27'
  }
  const pool = numberPoolTokens.value
  return pool.length ? `${pool[0]}-${pool[pool.length - 1]}` : '0-9'
})

// --- hot_cold_warm 冷热温出号 ---
const hcwTotalPeriods = ref(100)
const hcwWinRotate = ref(false)
/** 每位一个已选号码数组 */
const hcwPools = ref<string[][]>([])
const hcwLoading = ref(false)
const hcwStatsReady = ref(false)
interface HcwTier {
  hot: string[]
  warm: string[]
  cold: string[]
}
const hcwTiers = ref<HcwTier[]>([])

function ensureHcwPools(): void {
  const n = positionCount.value
  while (hcwPools.value.length < n) hcwPools.value.push([])
}

function applyHotColdWarmFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const c = raw as Record<string, unknown>
  const tp = Math.trunc(Number(c.totalPeriods))
  if (Number.isFinite(tp) && tp >= 20 && tp <= 500) hcwTotalPeriods.value = tp
  if (typeof c.winRotate === 'boolean') hcwWinRotate.value = c.winRotate
  if (Array.isArray(c.pool)) {
    // 回填时玩法树可能尚未就绪（numberPoolTokens 不可依赖），原样保留数字 token，
    // 展示选中态与去重均按数值比较（poolHasToken / toggleHcwDigit）
    hcwPools.value = c.pool.map((line) =>
      String(line ?? '')
        .split(/[,，\s]+/)
        .map((s) => s.trim())
        .filter((s) => /^\d{1,2}$/.test(s)),
    )
  }
}

/** 多位玩法将位面板对齐到开奖球序列（后 X 取尾、中 X 取中、定胆按子玩法定位） */
function hcwPositionOffset(ballsLen: number): number {
  const segLen = positionCount.value
  if (ballsLen <= segLen) return 0
  if (segLen === 1) {
    const sub = (schemePlayConfig.value.catalogSubId ?? schemePlayConfig.value.subPlayId ?? '').toLowerCase()
    const map: Array<[string, number]> = [
      ['wan', 0],
      ['qian', 1],
      ['bai', 2],
      ['shi', 3],
      ['ge', 4],
    ]
    for (const [key, idx] of map) {
      if (sub.includes(key)) return Math.min(idx, ballsLen - 1)
    }
    return 0
  }
  const typeId = schemePlayConfig.value.playTypeId
  if (typeId.startsWith('hou')) return ballsLen - segLen
  if (typeId.startsWith('zhong')) return Math.floor((ballsLen - segLen) / 2)
  return 0
}

async function loadHcwStats(): Promise<void> {
  if (hcwLoading.value) return
  hcwLoading.value = true
  try {
    const res = await fetchGameDraws(lotteryCode.value, undefined, hcwTotalPeriods.value)
    const items = Array.isArray(res?.items) ? res.items : []
    const segLen = positionCount.value
    const pool = numberPoolTokens.value
    const freq: Array<Record<string, number>> = Array.from({ length: segLen }, () => ({}))
    let counted = 0
    for (const it of items) {
      const balls = Array.isArray(it?.balls) ? it.balls : []
      if (!balls.length) continue
      const offset = hcwPositionOffset(balls.length)
      for (let p = 0; p < segLen; p++) {
        const tk = normalizePoolToken(String(balls[offset + p] ?? ''))
        if (tk) {
          freq[p]![tk] = (freq[p]![tk] ?? 0) + 1
          counted += 1
        }
      }
    }
    if (!counted) {
      hcwStatsReady.value = false
      return
    }
    // 频次降序排序后按池长三等分：热 / 温 / 冷（与引擎 hotColdWarmTiers 口径一致）
    const third = Math.ceil(pool.length / 3)
    hcwTiers.value = freq.map((counts) => {
      const sorted = [...pool].sort((a, b) => {
        const diff = (counts[b] ?? 0) - (counts[a] ?? 0)
        return diff !== 0 ? diff : Number(a) - Number(b)
      })
      return {
        hot: sorted.slice(0, third),
        warm: sorted.slice(third, 2 * third),
        cold: sorted.slice(2 * third),
      }
    })
    hcwStatsReady.value = true
  } catch {
    hcwStatsReady.value = false
  } finally {
    hcwLoading.value = false
  }
}

/** 池内是否已含该号（数值比较，兼容 '07' 与 '7'） */
function poolHasToken(arr: string[] | undefined, token: string): boolean {
  if (!arr) return false
  const n = Number(token)
  return arr.some((t) => Number(t) === n)
}

function toggleHcwDigit(pos: number, digit: string): void {
  ensureHcwPools()
  const arr = hcwPools.value[pos]
  if (!arr) return
  const n = Number(digit)
  const i = arr.findIndex((t) => Number(t) === n)
  if (i >= 0) arr.splice(i, 1)
  else {
    arr.push(digit)
    arr.sort((a, b) => Number(a) - Number(b))
  }
}

/** 预估注数：每位选号数相加 */
const hcwEstimatedUnits = computed(() =>
  Array.from({ length: positionCount.value }, (_, i) => (hcwPools.value[i] ?? []).length).reduce(
    (sum, n) => sum + n,
    0,
  ),
)

// --- random_draw 随机出号 ---
const rdCounts = ref<number[]>([])
const rdStrategy = ref<SchemeRandomDraw['strategy']>('every')
const rdPreview = ref<string[][]>([])

const RD_STRATEGY_OPTIONS = [
  { label: '每期换', value: 'every' },
  { label: '不换号', value: 'keep' },
  { label: '中后换', value: 'after_hit' },
  { label: '挂后换', value: 'after_miss' },
] as const

function ensureRdCounts(): void {
  const n = positionCount.value
  while (rdCounts.value.length < n) rdCounts.value.push(1)
}

/** 各面板状态随玩法位数 / 运行类型就绪（须在 ensureHcwPools / ensureRdCounts 声明之后） */
watch(
  [
    positionCount,
    runTypeId,
    isLonghuPlay,
    () => schemePlayConfig.value.betMode,
    () => schemePlayConfig.value.subPlayId,
    () => schemePlayConfig.value.catalogSubId,
    () => schemePlayConfig.value.playMethodLabel,
  ],
  () => {
    if (runTypeId.value === 'adv_trigger_bet') ensureTriggerRows()
    if (runTypeId.value === 'hot_cold_warm') ensureHcwPools()
    if (runTypeId.value === 'random_draw') ensureRdCounts()
  },
  { immediate: true },
)

function applyRandomDrawFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const c = raw as Record<string, unknown>
  if (Array.isArray(c.counts) && c.counts.length) {
    rdCounts.value = c.counts.map((n) => Math.min(10, Math.max(1, Math.trunc(Number(n)) || 1)))
  }
  const s = String(c.strategy ?? '')
  if (s === 'every' || s === 'keep' || s === 'after_hit' || s === 'after_miss') rdStrategy.value = s
}

/** 仅本地预览；云端运行时每期由引擎按数量自动随机 */
function generateRdPreview(): void {
  ensureRdCounts()
  rdPreview.value = Array.from({ length: positionCount.value }, (_, i) => {
    const pool = [...numberPoolTokens.value]
    const count = Math.min(10, Math.max(1, rdCounts.value[i] ?? 1), pool.length)
    for (let j = pool.length - 1; j > 0; j--) {
      const k = Math.floor(Math.random() * (j + 1))
        ;[pool[j], pool[k]] = [pool[k]!, pool[j]!]
    }
    return pool.slice(0, count).sort((a, b) => Number(a) - Number(b))
  })
}

// --- builtin_plan 内置计画 ---
const favorites = ref<SchemeFavoriteRow[]>([])
const favoritesLoading = ref(false)
const favoritesLoaded = ref(false)
const favSelectedSnapshotId = ref('')
const builtinSnapshotId = ref('')
const builtinApplying = ref(false)
const builtinReselecting = ref(false)

const builtinChosenFavorite = computed(
  () => favorites.value.find((f) => f.snapshotId === builtinSnapshotId.value) ?? null,
)

async function loadFavorites(): Promise<void> {
  if (favoritesLoading.value) return
  favoritesLoading.value = true
  try {
    favorites.value = await fetchSchemeFavorites()
  } catch {
    favorites.value = []
  } finally {
    favoritesLoading.value = false
    favoritesLoaded.value = true
  }
}

function formatFavoredAt(raw: string): string {
  const t = new Date(raw)
  if (Number.isNaN(t.getTime())) return raw
  return t.toLocaleString('zh-CN', { hour12: false })
}

function startBuiltinReselect(): void {
  builtinReselecting.value = true
  favSelectedSnapshotId.value = builtinSnapshotId.value
  if (!favoritesLoaded.value) void loadFavorites()
}

async function confirmBuiltinPlan(): Promise<void> {
  if (builtinApplying.value) return
  if (!favSelectedSnapshotId.value) {
    ElMessage.warning('请先选择一个收藏方案')
    return
  }
  if (isDraftScheme.value) {
    builtinSnapshotId.value = favSelectedSnapshotId.value
    ElMessage.success('已选择收藏方案')
    builtinReselecting.value = false
    persistDraft()
    return
  }
  builtinApplying.value = true
  try {
    await updateSchemeDefinition(schemeId.value, {
      builtinPlan: { snapshotId: favSelectedSnapshotId.value },
    })
    ElMessage.success('已复制该方案配置')
    builtinReselecting.value = false
    await loadRemoteDefinition()
  } catch (err) {
    const message = err instanceof ApiError ? err.message : err instanceof Error ? err.message : '选择失败'
    ElMessage.warning(message)
  } finally {
    builtinApplying.value = false
  }
}

const shareOptions = [
  { label: '私密 (仅自己可见)', value: 'private' as const },
  { label: '公开 (允许他人跟单)', value: 'public' as const },
]

const showShareField = computed(() => isCustomKind.value && !shareLocked.value)

const remoteHasInstance = ref(false)
const remoteReady = ref(false)
let remotePersistTimer: ReturnType<typeof setTimeout> | null = null
const instanceStatusText = computed(() => (remoteHasInstance.value ? '待开启' : ''))
const canDeleteScheme = computed(() => isDraftScheme.value || !remoteHasInstance.value)
const hasCloudInstance = computed(() => remoteHasInstance.value)

/** 倍投设定 Tab 与中文名称（与 BetMultiplierSettingsView 一致） */
const BET_MULTIPLIER_KIND_LABELS: Record<string, string> = {
  '0': '小白倍投',
  '1': '一键倍投',
  '2': '简单倍投',
  '3': '高级倍投',
}

/** 从本页进入倍投设定再返回时恢复滚动（避免回到页面顶部） */
function scrollRestoreStorageKey(): string {
  return `advanced-scheme-edit:scrollY:${String(route.params.schemeId ?? '')}`
}

function readDocumentScrollY(): number {
  return window.scrollY || document.documentElement.scrollTop || 0
}

function applyDraftSnapshot(draft: SchemeDraftSnapshot): void {
  schemeName.value = draft.meta.schemeName
  simBet.value = draft.simBet
  schemeFunds.value = draft.schemeFunds
  startTime.value = draft.startTime
  endTime.value = draft.endTime
  schemeGroups.value = draft.schemeGroups.length ? [...draft.schemeGroups] : ['']
  stopLoss.value = draft.stopLoss
  takeProfit.value = draft.takeProfit
  betUnit.value = normalizeBetUnitValue(draft.betUnit ?? (draft as { betMode?: string }).betMode ?? '2')
  multCoeff.value = draft.multCoeff || '1'
  shareStatus.value = draft.shareStatus
  runTypeId.value = normalizeRunTypeId(draft.meta.runTypeId)
  lotteryCode.value = draft.meta.lotteryCode || lotteryCode.value
  playTypeId.value = draft.meta.playTypeId || playTypeId.value
  subPlayId.value = draft.meta.subPlayId || subPlayId.value
  if (draft.betMultiplierKind) betMultiplierKind.value = draft.betMultiplierKind
  if (draft.betMultiplier) applyBetMultiplierFromConfig(draft.betMultiplier)
  if (draft.builtinSnapshotId) builtinSnapshotId.value = draft.builtinSnapshotId
  if (draft.jushuList?.length) applyJushuFromConfig(draft.jushuList)
  if (draft.triggerBet) applyTriggerBetFromConfig(draft.triggerBet)
  if (draft.hotColdWarm) applyHotColdWarmFromConfig(draft.hotColdWarm)
  if (draft.randomDraw) applyRandomDrawFromConfig(draft.randomDraw)
}

function buildDraftSnapshot(): SchemeDraftSnapshot {
  const meta = draftMetaFromQuery(route.query as Record<string, unknown>)
  meta.schemeName = schemeName.value.trim() || meta.schemeName
  const existing = loadSchemeDraft()
  const rtFields = runTypeDraftFields()
  return {
    meta,
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    startTime: startTime.value,
    endTime: endTime.value,
    schemeGroups: [...schemeGroups.value],
    stopLoss: stopLoss.value,
    takeProfit: takeProfit.value,
    betUnit: betUnit.value,
    multCoeff: multCoeff.value,
    shareStatus: shareStatus.value,
    betMultiplierKind: betMultiplierKind.value,
    betMultiplier: existing?.betMultiplier,
    builtinSnapshotId: builtinSnapshotId.value || undefined,
    jushuList: rtFields.jushuList,
    triggerBet: rtFields.triggerBet,
    hotColdWarm: rtFields.hotColdWarm,
    randomDraw: rtFields.randomDraw,
  }
}

function syncRunTypePanelsAfterSnapshot(): void {
  if (runTypeId.value === 'adv_trigger_bet') ensureTriggerRows()
  if (runTypeId.value === 'hot_cold_warm') {
    ensureHcwPools()
    void loadHcwStats()
  }
  if (runTypeId.value === 'random_draw') ensureRdCounts()
  if (runTypeId.value === 'builtin_plan' && !favoritesLoaded.value) void loadFavorites()
  if (runTypeId.value === 'adv_fixed_rotate' && !jushuList.value.length) seedJushuFromGroups()
}

/** 从倍投设定等子页返回时，用离开前快照覆盖远端/草稿加载结果 */
function applyPendingRestoreSnapshot(): void {
  const restored = consumeSchemeEditRestoreSnapshot(schemeId.value)
  if (!restored) return
  applyDraftSnapshot(restored)
  const draft = loadSchemeDraft()
  if (draft?.betMultiplier) {
    applyBetMultiplierFromConfig(draft.betMultiplier)
    if (draft.betMultiplierKind) betMultiplierKind.value = draft.betMultiplierKind
  }
  const qk = route.query.bmsKind
  const kindFromQuery = String(Array.isArray(qk) ? qk[0] : qk ?? '')
  if (kindFromQuery === '0' || kindFromQuery === '1' || kindFromQuery === '2' || kindFromQuery === '3') {
    betMultiplierKind.value = kindFromQuery
  }
  syncRunTypePanelsAfterSnapshot()
}

async function loadRemoteDefinition() {
  if (isDraftScheme.value) {
    const draft = loadSchemeDraft()
    if (draft) {
      applyDraftSnapshot(draft)
    } else {
      const meta = draftMetaFromQuery(route.query as Record<string, unknown>)
      schemeName.value = meta.schemeName
      runTypeId.value = normalizeRunTypeId(meta.runTypeId)
      if (meta.lotteryCode) lotteryCode.value = meta.lotteryCode
      if (meta.playTypeId) playTypeId.value = meta.playTypeId
      if (meta.subPlayId) subPlayId.value = meta.subPlayId
    }
    remoteHasInstance.value = false
    shareLocked.value = false
    void loadPlayTree()
    remoteReady.value = true
    syncRunTypePanelsAfterSnapshot()
    applyPendingRestoreSnapshot()
    return
  }
  try {
    const { items } = await fetchSchemeDefinitions()
    const def = items.find((d) => d.id === schemeId.value)
    if (!def) return
    remoteHasInstance.value = def.hasInstance
    shareLocked.value = def.hasInstance
    schemeName.value = def.schemeName
    shareStatus.value = def.shareStatusLocked === 'public' ? 'public' : 'private'
    const cfg = def.config ?? {}
    simBet.value = simBetFromSchemeConfig(cfg as Record<string, unknown>)
    if (typeof cfg.schemeFunds === 'string' || typeof cfg.schemeFunds === 'number') {
      schemeFunds.value = String(cfg.schemeFunds)
    }
    const times = normalizeSchemeTimePairFromConfig(cfg.startTime, cfg.endTime)
    startTime.value = times.start
    endTime.value = times.end
    if (typeof cfg.lotteryCode === 'string' && cfg.lotteryCode) {
      lotteryCode.value = cfg.lotteryCode
    }
    if (typeof cfg.playTypeId === 'string' && cfg.playTypeId) {
      playTypeId.value = cfg.playTypeId
    } else if (typeof cfg.typeId === 'string' && cfg.typeId) {
      playTypeId.value = cfg.typeId
    }
    if (typeof cfg.subPlayId === 'string' && cfg.subPlayId) {
      subPlayId.value = cfg.subPlayId
    } else if (typeof cfg.subId === 'string' && cfg.subId) {
      subPlayId.value = cfg.subId
    }
    void loadPlayTree()
    if (Array.isArray(cfg.schemeGroups) && cfg.schemeGroups.length > 0) {
      schemeGroups.value = cfg.schemeGroups.map((g) => String(g))
    }
    if (typeof cfg.stopLoss === 'string' || typeof cfg.stopLoss === 'number') {
      stopLoss.value = String(cfg.stopLoss)
    }
    if (typeof cfg.takeProfit === 'string' || typeof cfg.takeProfit === 'number') {
      takeProfit.value = String(cfg.takeProfit)
    }
    betUnit.value = betUnitFromSchemeConfig(cfg)
    applyBetMultiplierFromConfig(cfg.betMultiplier)
    if (typeof cfg.runTypeId === 'string' && cfg.runTypeId.trim()) {
      runTypeId.value = normalizeRunTypeId(cfg.runTypeId)
    }
    applyJushuFromConfig(cfg.jushuList)
    applyTriggerBetFromConfig(cfg.triggerBet)
    applyHotColdWarmFromConfig(cfg.hotColdWarm)
    applyRandomDrawFromConfig(cfg.randomDraw)
    const bp = cfg.builtinPlan
    if (bp && typeof bp === 'object' && typeof (bp as { snapshotId?: unknown }).snapshotId === 'string') {
      builtinSnapshotId.value = (bp as { snapshotId: string }).snapshotId
    }
    if (runTypeId.value === 'adv_fixed_rotate' && !jushuList.value.length) {
      seedJushuFromGroups()
    }
  } catch {
    /* 列表加载失败时保留 query 默认值 */
  } finally {
    remoteReady.value = true
    syncRunTypePanelsAfterSnapshot()
    applyPendingRestoreSnapshot()
  }
}

onMounted(() => {
  const raw = sessionStorage.getItem(scrollRestoreStorageKey())
  if (raw != null) {
    sessionStorage.removeItem(scrollRestoreStorageKey())
    const y = Number(raw)
    if (Number.isFinite(y) && y >= 0) {
      nextTick(() => {
        requestAnimationFrame(() => {
          window.scrollTo(0, y)
          requestAnimationFrame(() => {
            window.scrollTo(0, y)
          })
        })
      })
    }
  }

  void loadLotteries()
  void loadPlayTree()
  void loadRemoteDefinition()
})

/** 按运行类型把对应的方案内容字段并入 PATCH */
function runTypeDraftFields(): Partial<UpdateSchemeInput> {
  switch (runTypeId.value) {
    case 'adv_fixed_rotate':
      return jushuList.value.length ? { jushuList: jushuList.value.map((r) => ({ ...r })) } : {}
    case 'adv_trigger_bet': {
      const triggerBet: SchemeTriggerBet = {
        rows: triggerRows.value.map((r) => ({ ...r })),
        mode: triggerMode.value,
      }
      return { triggerBet }
    }
    case 'hot_cold_warm': {
      const hotColdWarm: SchemeHotColdWarm = {
        totalPeriods: hcwTotalPeriods.value,
        pool: Array.from({ length: positionCount.value }, (_, i) => (hcwPools.value[i] ?? []).join(',')),
        winRotate: hcwWinRotate.value,
      }
      return { hotColdWarm }
    }
    case 'random_draw': {
      const randomDraw: SchemeRandomDraw = {
        counts: Array.from({ length: positionCount.value }, (_, i) =>
          Math.min(10, Math.max(1, rdCounts.value[i] ?? 1)),
        ),
        strategy: rdStrategy.value,
      }
      return { randomDraw }
    }
    default:
      return {}
  }
}

function buildRemoteDraftPatch(): UpdateSchemeInput {
  return {
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    startTime: startTime.value,
    endTime: endTime.value,
    // 内置计画配置只读（服务端物化），不回写 schemeGroups；固定号码仅保存单元素数组
    ...(runTypeId.value === 'builtin_plan'
      ? {}
      : {
        schemeGroups:
          runTypeId.value === 'fixed_number' ? [schemeGroups.value[0] ?? ''] : [...schemeGroups.value],
      }),
    betUnit: betUnit.value,
    ...catalogFieldsFromPlayConfig(schemePlayConfig.value),
    ...(stopLoss.value ? { stopLoss: stopLoss.value } : {}),
    ...(takeProfit.value ? { takeProfit: takeProfit.value } : {}),
    ...runTypeDraftFields(),
  }
}

function flushPersistDraft(): void {
  if (remotePersistTimer) {
    clearTimeout(remotePersistTimer)
    remotePersistTimer = null
  }
  if (!remoteReady.value) return
  if (isDraftScheme.value) {
    saveSchemeDraft(buildDraftSnapshot())
    return
  }
  void updateSchemeDefinition(schemeId.value, buildRemoteDraftPatch()).catch(() => { })
}

function persistDraft() {
  if (!remoteReady.value) return
  if (remotePersistTimer) clearTimeout(remotePersistTimer)
  remotePersistTimer = setTimeout(() => flushPersistDraft(), 600)
}

watch(
  [
    schemeName,
    simBet,
    schemeFunds,
    startTime,
    endTime,
    schemeGroups,
    shareStatus,
    betUnit,
    stopLoss,
    takeProfit,
    multCoeff,
  ],
  persistDraft,
  { deep: true },
)

/** 七套面板状态跟随现有防抖持久化机制 */
watch(
  [jushuList, triggerRows, triggerMode, hcwTotalPeriods, hcwPools, hcwWinRotate, rdCounts, rdStrategy],
  persistDraft,
  { deep: true },
)

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push({ name: 'custom-scheme-new' })
}

/** 倍投设定方式（0–3），须从倍投设定页确认后才有值 */
const betMultiplierKind = ref<'' | '0' | '1' | '2' | '3'>('')

/** 倍投设定页校验失败：query.bmsError；确认成功：query.bmsKind（0–3） */
const betMultiplierError = ref('')

const betMultiplierSelectedLabel = computed(() =>
  betMultiplierKind.value ? (BET_MULTIPLIER_KIND_LABELS[betMultiplierKind.value] ?? '') : '',
)

function applyBetMultiplierFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const kind = (raw as { kind?: string }).kind
  if (kind === '0' || kind === '1' || kind === '2' || kind === '3') {
    betMultiplierKind.value = kind
  }
}

watch(
  () => route.query.bmsKind,
  (k) => {
    if (k == null || k === '') return
    const id = String(Array.isArray(k) ? k[0] : k)
    if (id === '0' || id === '1' || id === '2' || id === '3') {
      betMultiplierKind.value = id
      betMultiplierError.value = ''
    }
    const nextQuery = { ...route.query } as Record<string, string | string[] | undefined>
    delete nextQuery.bmsKind
    void router.replace({ query: nextQuery })
  },
  { immediate: true }
)

watch(
  () => route.query.bmsError,
  (q) => {
    if (q == null || q === '') return
    const raw = String(Array.isArray(q) ? q[0] : q)
    try {
      betMultiplierError.value = decodeURIComponent(raw)
    } catch {
      betMultiplierError.value = raw
    }
    betMultiplierKind.value = ''
    const nextQuery = { ...route.query } as Record<string, string | string[] | undefined>
    delete nextQuery.bmsError
    delete nextQuery.activeTab
    void router.replace({ query: nextQuery })
  },
  { immediate: true }
)

function schemeRouteQueryExtras(): Record<string, string> {
  const q: Record<string, string> = {}
  if (isDraftScheme.value) q.draft = '1'
  if (route.query.kind != null && String(route.query.kind) !== '') q.kind = String(route.query.kind)
  if (route.query.runType != null && String(route.query.runType) !== '') q.runType = String(route.query.runType)
  if (route.query.playType != null && String(route.query.playType) !== '') q.playType = String(route.query.playType)
  if (route.query.subPlay != null && String(route.query.subPlay) !== '') q.subPlay = String(route.query.subPlay)
  return q
}

function goBetMultiplierSettings() {
  betMultiplierError.value = ''
  const snapshot = buildDraftSnapshot()
  saveSchemeEditRestoreSnapshot(schemeId.value, snapshot)
  flushPersistDraft()
  sessionStorage.setItem(scrollRestoreStorageKey(), String(readDocumentScrollY()))
  router.push({
    name: 'bet-multiplier-settings',
    query: {
      fromScheme: '1',
      schemeId: String(route.params.schemeId ?? ''),
      ...(betMultiplierKind.value ? { activeTab: betMultiplierKind.value } : {}),
      ...(route.query.title != null && route.query.title !== ''
        ? { title: String(route.query.title) }
        : {}),
      ...(route.query.lottery != null && String(route.query.lottery) !== ''
        ? { lottery: String(route.query.lottery) }
        : {}),
      ...schemeRouteQueryExtras(),
    },
  })
}

function onClearContent(groupIdx: number) {
  schemeGroups.value[groupIdx] = ''
  ElMessage.info('已清空')
}

async function onDeleteGroup(groupIdx: number) {
  if (schemeGroups.value.length <= 1) {
    const ok = await confirmDialog({
      title: '清空组',
      message: '仅剩一组，将清空该组内容？',
      tone: 'warning',
      confirmText: '清空',
    })
    if (ok) {
      schemeGroups.value[0] = ''
      ElMessage.success('已清空')
    }
    return
  }
  const ok = await confirmDialog({
    title: '删除组',
    message: '确定删除该分组？',
    tone: 'danger',
    confirmText: '删除',
  })
  if (ok) {
    schemeGroups.value.splice(groupIdx, 1)
    ElMessage.success('已删除')
  }
}

function onAddGroup() {
  schemeGroups.value.push('')
}

async function onSaveCloud() {
  if (cloudBusy.value) return

  const warn = (message: string) =>
    confirmDialog({
      title: '提示',
      message,
      tone: 'warning',
      confirmText: '我知道了',
      showCancel: false,
    })

  const name = schemeName.value.trim()
  const lottery = String(route.query.lottery ?? '').trim()
  const fundsRaw = schemeFunds.value.trim()
  const funds = Number(fundsRaw)
  const groups = schemeGroups.value.map((g) => g.trim())

  if (!name) {
    await warn('方案名称不能为空')
    return
  }
  if (!lottery) {
    await warn('游戏名称不能为空')
    return
  }
  if (!fundsRaw || !Number.isFinite(funds)) {
    await warn('方案资金不能为空')
    return
  }
  if (funds <= 0) {
    await warn('方案资金必须大于 0')
    return
  }
  const timeErr = schemeTimeRangeError(startTime.value, endTime.value)
  if (timeErr) {
    await warn(timeErr)
    return
  }
  if (!betMultiplierKind.value) {
    await warn('方案模式不能为空，请前往倍投设定页选择')
    return
  }
  const stopLossRaw = stopLoss.value.trim()
  if (!stopLossRaw) {
    await warn('止损金额不能为空')
    return
  }
  const stopLossNum = Number(stopLossRaw)
  if (!Number.isFinite(stopLossNum) || stopLossNum < 0) {
    await warn('止损金额不能小于 0')
    return
  }
  const takeProfitRaw = takeProfit.value.trim()
  if (!takeProfitRaw) {
    await warn('止盈金额不能为空')
    return
  }
  const takeProfitNum = Number(takeProfitRaw)
  if (!Number.isFinite(takeProfitNum) || takeProfitNum < 0) {
    await warn('止盈金额不能小于 0')
    return
  }
  const multCoeffRaw = multCoeff.value.trim()
  if (!multCoeffRaw) {
    await warn('倍数系数不能为空')
    return
  }
  const multCoeffNum = Number(multCoeffRaw)
  if (!Number.isFinite(multCoeffNum) || multCoeffNum < 0) {
    await warn('倍数系数不能小于 0')
    return
  }
  if (!Number.isInteger(multCoeffNum)) {
    await warn('倍数系数只能为整数')
    return
  }

  const rt = runTypeId.value
  if (rt === 'adv_fixed_rotate') {
    if (!jushuList.value.length) {
      await warn('请至少添加一局投注号码')
      return
    }
    // 与局数内容对齐，供仍读取 schemeGroups 的下游兜底
    schemeGroups.value = jushuList.value.map((r) => r.content)
  } else if (rt === 'adv_trigger_bet') {
    const filled = triggerRows.value.some(
      (r) => r.enabled && (String(r.pos).trim() !== '' || String(r.neg).trim() !== ''),
    )
    if (!filled) {
      await warn('请填写开某投某映射（可用「全部随机」）')
      return
    }
    const sample = triggerRows.value.find((r) => r.enabled && String(r.pos).trim())
    schemeGroups.value = [sample ? String(sample.pos).trim() : '0']
  } else if (rt === 'hot_cold_warm') {
    ensureHcwPools()
    if (hcwEstimatedUnits.value <= 0) {
      await warn('请至少选择一个冷热温号码')
      return
    }
    schemeGroups.value = Array.from(
      { length: positionCount.value },
      (_, i) => (hcwPools.value[i] ?? []).join(','),
    )
  } else if (rt === 'random_draw') {
    ensureRdCounts()
    if (!rdPreview.value.length || rdPreview.value.every((row) => !row.length)) {
      generateRdPreview()
    }
    schemeGroups.value = Array.from({ length: positionCount.value }, (_, i) => {
      const prev = rdPreview.value[i] ?? []
      if (prev.length) return prev.join(',')
      const count = Math.min(10, Math.max(1, rdCounts.value[i] ?? 1))
      return Array.from({ length: count }, (_, j) => String(j % 10)).join(',')
    })
  } else {
    if (groups.every((g) => g === '')) {
      await warn('方案内容不能为空')
      return
    }
    if (groups.some((g) => g === '')) {
      await warn('存在空的方案分组，请填写内容或删除该组')
      return
    }

    const groupCheck = validateSchemeGroups(schemePlayConfig.value, schemeGroups.value)
    if (!groupCheck.ok) {
      for (const idx of groupCheck.invalidIndexes) {
        schemeGroups.value[idx] = ''
      }
      await confirmDialog({
        title: '输入不合法',
        message: `${groupCheck.message}。请按「${playModeSummary.value}」规则重新填写。`,
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return
    }
    schemeGroups.value = groupCheck.normalized
  }

  cloudBusy.value = true
  flushPersistDraft()

  const cloudPayload = {
    kind: schemeKind.value,
    schemeName: schemeName.value.trim() || '未命名方案',
    lotteryCode: String(route.query.lottery ?? ''),
    shareStatus: (isCustomKind.value ? shareStatus.value : 'private') as 'private' | 'public',
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    startTime: startTime.value,
    endTime: endTime.value,
    schemeGroups: [...schemeGroups.value],
    stopLoss: stopLoss.value,
    takeProfit: takeProfit.value,
    betUnit: betUnit.value,
    ...catalogFieldsFromPlayConfig(schemePlayConfig.value),
  }

  try {
    if (isDraftScheme.value) {
      saveSchemeDraft(buildDraftSnapshot())
      const draft = loadSchemeDraft()
      if (!draft) {
        ElMessage.warning('方案草稿丢失，请返回重新新建')
        return
      }
      const meta = draft.meta
      let createdDefId: string | null = null
      try {
        const def = await createScheme({
          kind: meta.kind,
          schemeName: meta.schemeName,
          lotteryCode: meta.lotteryCode,
          runTypeId: meta.runTypeId,
          playTypeId: meta.playTypeId,
          subPlayId: meta.subPlayId,
        })
        createdDefId = def.id
        const patch = {
          ...draftPatchFromSnapshot(draft),
          ...catalogFieldsFromPlayConfig(schemePlayConfig.value),
        }
        const syncedBetMultiplier = await syncDraftAdvancedTemplatesToServer(def.id, draft)
        if (syncedBetMultiplier) {
          patch.betMultiplier = syncedBetMultiplier as unknown as Record<string, unknown>
        }
        await updateSchemeDefinition(def.id, patch)
        await addSchemeToCloud(def.id, cloudPayload)
        clearSchemeDraft()
        ElMessage.success('已添加至云端（待开启）')
        router.push({ name: 'cloud' })
      } catch (innerErr) {
        if (createdDefId) {
          try {
            await deleteSchemeDefinition(createdDefId)
          } catch {
            /* 回滚失败时保留定义，用户可在新建页「删除重建」 */
          }
        }
        throw innerErr
      }
      return
    }

    if (hasCloudInstance.value) {
      const forkResult = await forkSchemeToCloud(schemeId.value, cloudPayload)
      ElMessage.success(`已复制为「${forkResult.definition.schemeName}」并添加至云端（待开启）`)
      void router.replace({
        name: 'advanced-scheme-edit',
        params: { schemeId: forkResult.definition.id },
        query: { ...route.query, kind: schemeKind.value },
      })
      return
    }

    await addSchemeToCloud(schemeId.value, cloudPayload)
    shareLocked.value = true
    remoteHasInstance.value = true
    ElMessage.success('已添加至云端（待开启）')
    router.push({ name: 'cloud' })
  } catch (err) {
    const message = err instanceof ApiError ? err.message : err instanceof Error ? err.message : '添加失败'
    ElMessage.warning(message)
  } finally {
    setTimeout(() => {
      cloudBusy.value = false
    }, 1000)
  }
}

async function onDeleteScheme() {
  if (!canDeleteScheme.value) {
    ElMessage.warning('运行中的方案不可删除')
    return
  }
  if (isDraftScheme.value) {
    const ok = await confirmDialog({
      title: '放弃新建',
      message: '尚未添加至云端，确定放弃本次新建？',
      tone: 'danger',
      confirmText: '放弃',
    })
    if (!ok) return
    clearSchemeDraft()
    ElMessage.success('已放弃新建')
    goBack()
    return
  }
  const ok = await confirmDialog({
    title: '删除方案',
    message: '删除后将同时移除云端实例，确定继续？',
    tone: 'danger',
    confirmText: '删除',
  })
  if (!ok) return
  try {
    await deleteSchemeDefinition(schemeId.value)
    ElMessage.success('方案已删除')
    goBack()
  } catch (err) {
    const message = err instanceof ApiError ? err.message : err instanceof Error ? err.message : '删除失败'
    ElMessage.warning(message)
  }
}

// ----- 运行时段弹窗（滚轮 + 开始/结束切换） -----
const TW_ITEM_H = 44
const twHours24 = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'))
const twMinutes = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, '0'))

const timeDialogVisible = ref(false)
const timeActive = ref<'start' | 'end'>('start')
const pendingStart = ref('00:00')
const pendingEnd = ref('23:59')

const selHourIdx = ref(0)
const selMinIdx = ref(0)

const hourScrollRef = ref<HTMLElement | null>(null)
const minScrollRef = ref<HTMLElement | null>(null)

let twScrollTimer: ReturnType<typeof setTimeout> | null = null

function parseHm(s: string): { h: number; m: number } | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec((s ?? '').trim())
  if (!m) return null
  const h = Number(m[1])
  const mi = Number(m[2])
  if (Number.isNaN(h) || Number.isNaN(mi) || h < 0 || h > 23 || mi < 0 || mi > 59) return null
  return { h, m: mi }
}

function normalizeHm(s: string, fallback = '00:00'): string {
  const p = parseHm(s)
  if (!p) return fallback
  return `${String(p.h).padStart(2, '0')}:${String(p.m).padStart(2, '0')}`
}

/** 24h：小时 0–23 → selHourIdx 0–23 */
function hmToPickerParts(hm: string): { hi: number; mi: number } {
  const p = parseHm(hm) ?? { h: 0, m: 0 }
  return { hi: p.h, mi: p.m }
}

function pickerPartsToHm(hi: number, mi: number): string {
  const h = Math.max(0, Math.min(23, hi))
  const m = Math.max(0, Math.min(59, mi))
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

function hmFromPicker(): string {
  return pickerPartsToHm(selHourIdx.value, selMinIdx.value)
}

function loadPickerFromHm(hm: string) {
  const { hi, mi } = hmToPickerParts(normalizeHm(hm))
  selHourIdx.value = hi
  selMinIdx.value = mi
}

function snapScroll(el: HTMLElement | null, idx: number, maxIdx: number) {
  if (!el) return
  const i = Math.max(0, Math.min(maxIdx, idx))
  el.scrollTo({ top: i * TW_ITEM_H, behavior: 'auto' })
}

function snapAllScrolls() {
  snapScroll(hourScrollRef.value, selHourIdx.value, 23)
  snapScroll(minScrollRef.value, selMinIdx.value, 59)
}

function scheduleTwScrollSync(kind: 'h' | 'm') {
  if (twScrollTimer) clearTimeout(twScrollTimer)
  twScrollTimer = setTimeout(() => finalizeTwScroll(kind), 72)
}

function finalizeTwScroll(kind: 'h' | 'm') {
  if (kind === 'h' && hourScrollRef.value) {
    const idx = Math.round(hourScrollRef.value.scrollTop / TW_ITEM_H)
    selHourIdx.value = Math.max(0, Math.min(23, idx))
    snapScroll(hourScrollRef.value, selHourIdx.value, 23)
  }
  if (kind === 'm' && minScrollRef.value) {
    const idx = Math.round(minScrollRef.value.scrollTop / TW_ITEM_H)
    selMinIdx.value = Math.max(0, Math.min(59, idx))
    snapScroll(minScrollRef.value, selMinIdx.value, 59)
  }
}

function twSelectHour(idx: number) {
  selHourIdx.value = idx
  snapScroll(hourScrollRef.value, idx, 23)
}

function twSelectMin(idx: number) {
  selMinIdx.value = idx
  snapScroll(minScrollRef.value, idx, 59)
}

function setTimeActive(tab: 'start' | 'end') {
  if (tab === timeActive.value) return
  if (timeActive.value === 'start') pendingStart.value = hmFromPicker()
  else pendingEnd.value = hmFromPicker()
  timeActive.value = tab
  const hm = tab === 'start' ? pendingStart.value : pendingEnd.value
  loadPickerFromHm(hm)
  nextTick(() => snapAllScrolls())
}

function formatHm24Label(hm: string): string {
  return normalizeHm(hm)
}

const displayStartSummary = computed(() => formatHm24Label(pendingStart.value))
const displayEndSummary = computed(() => formatHm24Label(pendingEnd.value))

function confirmTimeDialog() {
  if (timeActive.value === 'start') pendingStart.value = hmFromPicker()
  else pendingEnd.value = hmFromPicker()

  startTime.value = normalizeHm(pendingStart.value)
  endTime.value = normalizeHm(pendingEnd.value)
  timeDialogVisible.value = false
}

const displayMainStart = computed(() => startTime.value || '选择时间')
const displayMainEnd = computed(() => endTime.value || '选择时间')

// 日期时间选择弹窗（先选日期再选时间）
const dtpVisible = ref(false)
const dtpField = ref<'start' | 'end'>('start')
const dtpValue = ref('')

function openTimePicker(field: 'start' | 'end') {
  dtpField.value = field
  dtpValue.value = field === 'start' ? startTime.value : endTime.value
  dtpVisible.value = true
}

function onTimePicked(dt: string) {
  if (dtpField.value === 'start') startTime.value = dt
  else endTime.value = dt
}

function onTimeDialogOpened() {
  nextTick(() => snapAllScrolls())
}
</script>

<template>
  <div class="scf">
    <header class="scf-header">
      <button type="button" class="scf-back" aria-label="返回" @click="goBack">
        <img :src="BACK_ICO" alt="" width="24" height="24" class="scf-back-img" decoding="async" />
      </button>
      <h1 class="scf-title">方案配置</h1>
      <div class="scf-header-right">
        <span v-if="instanceStatusText" class="scf-inst-badge">{{ instanceStatusText }}</span>
        <button type="button" class="scf-del-btn" :disabled="!canDeleteScheme" @click="onDeleteScheme">
          删除
        </button>
      </div>
    </header>

    <main class="scf-main">
      <section class="scf-section">
        <div class="scf-section-head">
          <h2 class="scf-section-title">基础设置</h2>
        </div>
        <div class="scf-card scf-stack">
          <div class="scf-field">
            <span class="scf-lbl">运行模式</span>
            <div class="scf-seg" role="group" aria-label="运行模式">
              <button type="button" class="scf-seg-btn" :class="{ 'is-active': !simBet }" @click="simBet = false">
                正式运行
              </button>
              <button type="button" class="scf-seg-btn" :class="{ 'is-active': simBet }" @click="simBet = true">
                模拟运行
              </button>
            </div>
          </div>
          <div class="scf-field">
            <label class="scf-lbl" for="scf-name">方案名称</label>
            <el-input id="scf-name" v-model="schemeName" size="large" class="scf-el-inp" placeholder="方案名称" />
          </div>
          <div v-if="showShareField" class="scf-field">
            <span class="scf-lbl">分享状态</span>
            <el-select v-model="shareStatus" class="scf-el-select" size="large" placeholder="选择">
              <el-option v-for="o in shareOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </div>
          <div v-else-if="isCustomKind && shareLocked" class="scf-field">
            <span class="scf-lbl">分享状态</span>
            <div class="scf-readonly">
              {{ shareStatus === 'public' ? '公开 (已锁定)' : '私密 (已锁定)' }}
            </div>
          </div>
          <div class="scf-grid2">
            <div class="scf-field">
              <label class="scf-lbl" for="scf-funds">方案资金</label>
              <div class="scf-suffix-wrap">
                <el-input id="scf-funds" v-model="schemeFunds" size="large" class="scf-el-inp scf-el-inp--suffix"
                  type="number" />
              </div>
            </div>
            <div class="scf-field">
              <span class="scf-lbl">游戏名称</span>
              <div class="scf-readonly">{{ gameNameDisplay }}</div>
            </div>
          </div>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head scf-section-head--plain">
          <h2 class="scf-section-title">运行逻辑</h2>
        </div>
        <div class="scf-card scf-stack">
          <div class="scf-tip">
            <p>
              提示：方案保存后将自动同步至精算云中心。开始时间与结束时间须同时填写，或同时留空表示无限期运行。
            </p>
          </div>
          <div class="scf-grid2">
            <div class="scf-field">
              <span class="scf-lbl">开始时间</span>
              <button type="button" class="scf-time-hit" aria-haspopup="dialog" @click="openTimePicker('start')">
                <span class="scf-time-hit-val">{{ displayMainStart }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">schedule</span>
              </button>
            </div>
            <div class="scf-field">
              <span class="scf-lbl">结束时间</span>
              <button type="button" class="scf-time-hit" aria-haspopup="dialog" @click="openTimePicker('end')">
                <span class="scf-time-hit-val">{{ displayMainEnd }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">schedule</span>
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head scf-section-head--plain">
          <h2 class="scf-section-title">风险控制</h2>
        </div>
        <div class="scf-card scf-grid2">
          <div class="scf-field">
            <label class="scf-lbl" for="scf-sl">止损金额</label>
            <el-input id="scf-sl" v-model="stopLoss" size="large" class="scf-el-inp scf-el-inp--danger"
              placeholder="0.00" type="number" />
          </div>
          <div class="scf-field">
            <label class="scf-lbl" for="scf-tp">止盈金额</label>
            <el-input id="scf-tp" v-model="takeProfit" size="large" class="scf-el-inp scf-el-inp--profit"
              placeholder="0.00" type="number" />
          </div>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head scf-section-head--plain">
          <h2 class="scf-section-title">投注参数</h2>
        </div>
        <div class="scf-card scf-stack">
          <div class="scf-grid2">
            <div class="scf-field">
              <label class="scf-lbl" for="scf-mult">倍数系数</label>
              <el-input id="scf-mult" v-model="multCoeff" size="large" class="scf-el-inp" type="number" :min="0"
                :step="1" />
            </div>
            <div class="scf-field">
              <span class="scf-lbl">投注单位</span>
              <el-select v-model="betUnit" class="scf-el-select" size="large">
                <el-option v-for="o in BET_MODE_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </div>
          </div>
          <button type="button" class="scf-mode-card" @click="goBetMultiplierSettings">
            <div class="scf-mode-left">
              <span class="scf-mode-ico-bg" aria-hidden="true">
                <span class="scf-ms scf-ms--white">analytics</span>
              </span>
              <div class="scf-mode-texts">
                <p class="scf-mode-title">方案模式设置</p>
              </div>
            </div>
            <div class="scf-mode-right">
              <span class="scf-ms scf-ms--primary scf-mode-gear" aria-hidden="true">settings</span>
              <p v-if="betMultiplierError" class="scf-mode-err" role="alert">
                {{ betMultiplierError }}
              </p>
              <p v-else-if="betMultiplierSelectedLabel" class="scf-mode-selected">
                {{ betMultiplierSelectedLabel }}
              </p>
              <p v-else class="scf-mode-unset">未设置，请点击进入倍投设定</p>
            </div>
          </button>
          <p class="scf-mode-share-hint">倍投设定与轮次设置共用同一份倍率配置，后保存者覆盖先前配置</p>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head">
          <div class="scf-section-head-left">
            <h2 class="scf-section-title">方案内容</h2>
            <p class="scf-play-hint">{{ runTypeLabel }} · {{ playModeSummary }}</p>
          </div>
          <button v-if="runTypeId === 'fixed_rotate'" type="button" class="scf-add-btn" @click="onAddGroup">
            <span class="scf-ms scf-ms--sm" aria-hidden="true">add</span>
            <span>新增</span>
          </button>
          <button v-else-if="runTypeId === 'adv_fixed_rotate'" type="button" class="scf-add-btn"
            @click="openJushuDialog">
            <span class="scf-ms scf-ms--sm" aria-hidden="true">add</span>
            <span>添加局数</span>
          </button>
          <button v-else-if="runTypeId === 'adv_trigger_bet'" type="button" class="scf-add-btn"
            @click="randomFillTrigger">
            <span class="scf-ms scf-ms--sm" aria-hidden="true">casino</span>
            <span>全部随机</span>
          </button>
        </div>

        <!-- 1/2. 定码轮换（多分组） / 固定号码（单组） -->
        <div v-if="runTypeId === 'fixed_rotate' || runTypeId === 'fixed_number'" class="scf-groups-stack">
          <p v-if="runTypeId === 'fixed_number'" class="scf-run-tip scf-run-tip--banner">
            固定号码：仅需设置一注号码，每期原样复投
          </p>
          <div v-for="idx in displayedGroupIndexes" :key="idx" class="scf-content-card">
            <div class="scf-group-bar">
              <h3 class="scf-group-title">
                {{ runTypeId === 'fixed_number' ? '固定号码' : `第 ${idx + 1} 组` }}
              </h3>
              <div class="scf-content-toolbar scf-content-toolbar--group" role="toolbar"
                :aria-label="`第 ${idx + 1} 组操作`">
                <button type="button" class="scf-tb-btn scf-tb-btn--muted" @click="onClearContent(idx)">
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">backspace</span>
                  <span>清空</span>
                </button>
                <button v-if="runTypeId === 'fixed_rotate'" type="button" class="scf-tb-btn scf-tb-btn--danger"
                  @click="onDeleteGroup(idx)">
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">delete</span>
                  <span>删除组</span>
                </button>
              </div>
            </div>
            <div class="scf-textarea-wrap">
              <SchemeGroupPickPanel v-if="schemeUsesPickPanel" v-model="schemeGroups[idx]" :config="schemePlayConfig" />
              <el-input v-if="!schemeUsesPickPanel" v-model="schemeGroups[idx]" type="textarea" :rows="5" resize="none"
                class="scf-area" :placeholder="groupInputPlaceholder" />
              <div class="scf-area-meta">
                <span v-if="schemeUsesPickPanel">
                  {{
                    schemePlayConfig.inputMode === 'multiline'
                      ? '按位选号，每位多选以逗号保存'
                      : '点击号码选择，多选以逗号保存'
                  }}
                </span>
                <span v-else>每注以逗号分隔</span>
                <span>注数: {{ groupBetUnits(schemeGroups[idx] ?? '') }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 3. 高级定码轮换：局数列表 -->
        <div v-else-if="runTypeId === 'adv_fixed_rotate'" class="scf-content-card scf-panel">
          <p class="scf-run-tip">跳转到不存在的局数时，自动回到第 1 局</p>
          <el-empty v-if="!jushuList.length" description="暂无局数，点击右上角「添加局数」录入" :image-size="56" />
          <ul v-else class="scf-jushu-list">
            <li v-for="(row, idx) in jushuList" :key="row.ju" class="scf-jushu-row">
              <div class="scf-jushu-main">
                <span class="scf-jushu-no">第 {{ row.ju }} 局</span>
                <span class="scf-jushu-content">{{ row.content }}</span>
              </div>
              <div class="scf-jushu-side">
                <span class="scf-jushu-jump">中后 → 第 {{ row.afterHit }} 局</span>
                <span class="scf-jushu-jump">挂后 → 第 {{ row.afterMiss }} 局</span>
                <button type="button" class="scf-jushu-del" :aria-label="`删除第 ${row.ju} 局`"
                  @click="removeJushuRow(idx)">
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">delete</span>
                </button>
              </div>
            </li>
          </ul>
        </div>

        <!-- 4. 高级开某投某：映射表 + 投向模式 -->
        <div v-else-if="runTypeId === 'adv_trigger_bet'" class="scf-content-card scf-panel">
          <div class="scf-trig-grid scf-trig-grid--head" aria-hidden="true">
            <span>启用</span>
            <span>开出</span>
            <span>正投</span>
            <span>反投</span>
          </div>
          <div v-for="row in triggerRows" :key="row.open" class="scf-trig-grid" :class="{ 'is-off': !row.enabled }">
            <el-switch v-model="row.enabled" size="small" :aria-label="`启用开出 ${row.open} 的映射`" />
            <span class="scf-trig-open">{{ row.open }}</span>
            <template v-if="isTriggerTextPlay">
              <el-select v-model="row.pos" size="small" placeholder="正投" :disabled="!row.enabled">
                <el-option v-for="v in triggerBetOptions" :key="v" :label="v" :value="v" />
              </el-select>
              <el-select v-model="row.neg" size="small" placeholder="反投" :disabled="!row.enabled">
                <el-option v-for="v in triggerBetOptions" :key="`neg-${v}`" :label="v" :value="v" />
              </el-select>
            </template>
            <template v-else>
              <el-input :model-value="row.pos" size="small" maxlength="2" :placeholder="triggerInputPlaceholder"
                inputmode="numeric" :disabled="!row.enabled"
                @update:model-value="row.pos = sanitizeTriggerDigit($event)" />
              <el-input :model-value="row.neg" size="small" maxlength="2" :placeholder="triggerInputPlaceholder"
                inputmode="numeric" :disabled="!row.enabled"
                @update:model-value="row.neg = sanitizeTriggerDigit($event)" />
            </template>
          </div>
          <div class="scf-field scf-panel-field">
            <span class="scf-lbl">投向模式</span>
            <el-radio-group v-model="triggerMode" class="scf-radio-wrap">
              <el-radio v-for="o in TRIGGER_MODE_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</el-radio>
            </el-radio-group>
          </div>
          <p class="scf-run-tip">上期开出号码无启用映射时，按启用行第 1 行的正投下注</p>
        </div>

        <!-- 5. 冷热温出号 -->
        <div v-else-if="runTypeId === 'hot_cold_warm'" class="scf-content-card scf-panel">
          <div class="scf-hcw-bar">
            <div class="scf-hcw-ctrl">
              <span class="scf-lbl">总期数</span>
              <el-input-number v-model="hcwTotalPeriods" :min="20" :max="500" :step="10" size="small" />
              <el-button size="small" type="primary" plain :loading="hcwLoading" @click="loadHcwStats">
                刷新统计
              </el-button>
            </div>
            <div class="scf-hcw-ctrl">
              <span class="scf-lbl">中奖轮换</span>
              <el-switch v-model="hcwWinRotate" />
            </div>
          </div>
          <div v-for="(label, pi) in positionLabels" :key="pi" class="scf-hcw-pos">
            <p class="scf-hcw-pos-name">{{ label }}</p>
            <template v-if="hcwStatsReady">
              <div class="scf-hcw-tier scf-hcw-tier--hot">
                <span class="scf-hcw-tier-lbl">热</span>
                <div class="scf-hcw-chips">
                  <button v-for="d in hcwTiers[pi]?.hot ?? []" :key="d" type="button" class="scf-hcw-chip"
                    :class="{ 'is-on': poolHasToken(hcwPools[pi], d) }" @click="toggleHcwDigit(pi, d)">
                    {{ d }}
                  </button>
                </div>
              </div>
              <div class="scf-hcw-tier scf-hcw-tier--warm">
                <span class="scf-hcw-tier-lbl">温</span>
                <div class="scf-hcw-chips">
                  <button v-for="d in hcwTiers[pi]?.warm ?? []" :key="d" type="button" class="scf-hcw-chip"
                    :class="{ 'is-on': poolHasToken(hcwPools[pi], d) }" @click="toggleHcwDigit(pi, d)">
                    {{ d }}
                  </button>
                </div>
              </div>
              <div class="scf-hcw-tier scf-hcw-tier--cold">
                <span class="scf-hcw-tier-lbl">冷</span>
                <div class="scf-hcw-chips">
                  <button v-for="d in hcwTiers[pi]?.cold ?? []" :key="d" type="button" class="scf-hcw-chip"
                    :class="{ 'is-on': poolHasToken(hcwPools[pi], d) }" @click="toggleHcwDigit(pi, d)">
                    {{ d }}
                  </button>
                </div>
              </div>
            </template>
            <template v-else>
              <p class="scf-run-tip">暂无开奖统计，可直接手动选号</p>
              <div class="scf-hcw-tier scf-hcw-tier--plain">
                <div class="scf-hcw-chips">
                  <button v-for="d in numberPoolTokens" :key="d" type="button" class="scf-hcw-chip"
                    :class="{ 'is-on': poolHasToken(hcwPools[pi], d) }" @click="toggleHcwDigit(pi, d)">
                    {{ d }}
                  </button>
                </div>
              </div>
            </template>
          </div>
          <div class="scf-hcw-pool">
            <p class="scf-hcw-pool-title">
              <span>已选号码池</span>
              <span class="scf-hcw-pool-units">预估 {{ hcwEstimatedUnits }} 注</span>
            </p>
            <p v-for="(label, pi) in positionLabels" :key="pi" class="scf-hcw-pool-line">
              <span class="scf-hcw-pool-pos">{{ label }}</span>
              <span class="scf-hcw-pool-vals">{{ (hcwPools[pi] ?? []).join(',') || '未选' }}</span>
            </p>
          </div>
        </div>

        <!-- 6. 随机出号 -->
        <div v-else-if="runTypeId === 'random_draw'" class="scf-content-card scf-panel">
          <div v-for="(label, pi) in positionLabels" :key="pi" class="scf-rd-row">
            <span class="scf-rd-pos">{{ label }}</span>
            <el-input-number v-model="rdCounts[pi]" :min="1" :max="10" size="small" />
            <span v-if="(rdPreview[pi] ?? []).length" class="scf-rd-preview">
              预览：{{ (rdPreview[pi] ?? []).join(',') }}
            </span>
          </div>
          <div class="scf-rd-actions">
            <el-button type="primary" plain size="small" @click="generateRdPreview">生成预览</el-button>
          </div>
          <div class="scf-field scf-panel-field">
            <span class="scf-lbl">换号策略</span>
            <el-radio-group v-model="rdStrategy" class="scf-radio-wrap">
              <el-radio v-for="o in RD_STRATEGY_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</el-radio>
            </el-radio-group>
          </div>
          <p class="scf-run-tip">云端运行时每期由引擎按数量自动随机，实际号码见投注明细</p>
        </div>

        <!-- 7. 内置计画 -->
        <div v-else-if="runTypeId === 'builtin_plan'" class="scf-content-card scf-panel">
          <template v-if="builtinSnapshotId && !builtinReselecting">
            <div class="scf-bp-summary">
              <div class="scf-bp-summary-main">
                <p class="scf-bp-summary-title">
                  已跟随：{{ builtinChosenFavorite?.schemeName ?? schemeName }} ·
                  {{ builtinChosenFavorite?.playMethod ?? playModeSummary }}
                </p>
                <p class="scf-run-tip">内置计画配置只读，与收藏计划保持一致</p>
              </div>
              <el-button size="small" plain @click="startBuiltinReselect">重新选择</el-button>
            </div>
          </template>
          <template v-else>
            <el-empty v-if="favoritesLoaded && !favorites.length" description="暂无收藏方案，先去跟单大厅收藏方案" :image-size="64" />
            <template v-else>
              <div class="scf-bp-list">
                <button v-for="f in favorites" :key="f.snapshotId" type="button" class="scf-bp-item"
                  :class="{ 'is-sel': favSelectedSnapshotId === f.snapshotId }"
                  @click="favSelectedSnapshotId = f.snapshotId">
                  <span class="scf-bp-radio" :class="{ 'is-on': favSelectedSnapshotId === f.snapshotId }"
                    aria-hidden="true" />
                  <span class="scf-bp-info">
                    <span class="scf-bp-name">{{ f.schemeName }}</span>
                    <span class="scf-bp-meta">
                      {{ f.lotteryLabel }} · {{ f.playMethod }} · 收藏于 {{ formatFavoredAt(f.favoredAt) }}
                    </span>
                  </span>
                </button>
              </div>
              <div class="scf-bp-actions">
                <el-button type="primary" :loading="builtinApplying" :disabled="!favSelectedSnapshotId"
                  @click="confirmBuiltinPlan">
                  确认选择
                </el-button>
                <el-button v-if="builtinReselecting" plain @click="builtinReselecting = false">取消</el-button>
              </div>
            </template>
          </template>
        </div>
      </section>

      <div class="scf-main-pad" aria-hidden="true" />
    </main>

    <DateTimePickerModal v-model="dtpVisible" :value="dtpValue" :title="dtpField === 'start' ? '开始时间' : '结束时间'"
      @confirm="onTimePicked" />

    <el-dialog v-model="timeDialogVisible" title="运行时段" width="min(22rem, calc(100vw - 2rem))" class="scf-tw-dialog"
      modal-class="scf-tw-overlay" append-to-body align-center destroy-on-close @opened="onTimeDialogOpened">
      <div class="scf-tw">
        <div class="scf-tw-wheel-wrap">
          <div class="scf-tw-highlight" aria-hidden="true" />
          <div class="scf-tw-row">
            <div class="scf-tw-mask scf-tw-mask--hour">
              <div ref="hourScrollRef" class="scf-tw-scroll" role="listbox" aria-label="小时（24 小时制）"
                @scroll.passive="scheduleTwScrollSync('h')">
                <div class="scf-tw-spacer" aria-hidden="true" />
                <div v-for="(h, idx) in twHours24" :key="'h' + h" class="scf-tw-cell"
                  :class="{ 'is-sel': selHourIdx === idx }" role="option" :aria-selected="selHourIdx === idx"
                  @click="twSelectHour(idx)">
                  {{ h }}
                </div>
                <div class="scf-tw-spacer" aria-hidden="true" />
              </div>
            </div>
            <span class="scf-tw-colon" aria-hidden="true">:</span>
            <div class="scf-tw-mask scf-tw-mask--min">
              <div ref="minScrollRef" class="scf-tw-scroll" role="listbox" aria-label="分钟"
                @scroll.passive="scheduleTwScrollSync('m')">
                <div class="scf-tw-spacer" aria-hidden="true" />
                <div v-for="(n, idx) in twMinutes" :key="'m' + n" class="scf-tw-cell"
                  :class="{ 'is-sel': selMinIdx === idx }" role="option" :aria-selected="selMinIdx === idx"
                  @click="twSelectMin(idx)">
                  {{ n }}
                </div>
                <div class="scf-tw-spacer" aria-hidden="true" />
              </div>
            </div>
          </div>
        </div>

        <div class="scf-tw-summary">
          <button type="button" class="scf-tw-sum-half" :class="{ 'is-active': timeActive === 'start' }"
            @click="setTimeActive('start')">
            <span class="scf-tw-sum-lbl">开始时间</span>
            <span class="scf-tw-sum-val">{{ displayStartSummary }}</span>
          </button>
          <button type="button" class="scf-tw-sum-half" :class="{ 'is-active': timeActive === 'end' }"
            @click="setTimeActive('end')">
            <span class="scf-tw-sum-lbl">结束时间</span>
            <span class="scf-tw-sum-val">{{ displayEndSummary }}</span>
          </button>
        </div>

        <el-button type="primary" class="scf-tw-confirm" size="large" @click="confirmTimeDialog">
          <span>确认选择</span>
          <span class="scf-tw-check" aria-hidden="true">
            <span class="scf-ms scf-ms--fill scf-ms--white scf-tw-check-ico">check</span>
          </span>
        </el-button>
      </div>
    </el-dialog>

    <el-dialog v-model="jushuDialogVisible" title="添加局数" width="min(24rem, calc(100vw - 2rem))" append-to-body
      align-center destroy-on-close class="scf-jushu-dialog">
      <div class="scf-jushu-form">
        <div class="scf-field">
          <span class="scf-lbl">局数</span>
          <el-input-number v-model="jushuForm.ju" :min="1" :step="1" step-strictly class="scf-jushu-num" />
        </div>
        <div class="scf-field">
          <span class="scf-lbl">投注号码</span>
          <SchemeGroupPickPanel v-if="schemeUsesPickPanel" v-model="jushuForm.content" :config="schemePlayConfig" />
          <el-input v-if="!schemeUsesPickPanel" v-model="jushuForm.content" type="textarea" :rows="4" resize="none"
            class="scf-area" :placeholder="groupInputPlaceholder" />
        </div>
        <div class="scf-grid2">
          <div class="scf-field">
            <span class="scf-lbl">中后跳转局</span>
            <el-input-number v-model="jushuForm.afterHit" :min="1" :step="1" step-strictly class="scf-jushu-num" />
          </div>
          <div class="scf-field">
            <span class="scf-lbl">挂后跳转局</span>
            <el-input-number v-model="jushuForm.afterMiss" :min="1" :step="1" step-strictly class="scf-jushu-num" />
          </div>
        </div>
        <p class="scf-run-tip">跳转到不存在的局数时，自动回到第 1 局</p>
      </div>
      <template #footer>
        <el-button @click="jushuDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmJushuDialog">确认添加</el-button>
      </template>
    </el-dialog>

    <footer class="scf-footer">
      <p v-if="hasCloudInstance" class="scf-fork-hint">
        该方案已有云端实例；再次添加将自动复制新方案（默认私密）。
      </p>
      <el-button type="primary" class="scf-cloud-btn" size="large" :loading="cloudBusy" :disabled="cloudBusy"
        @click="onSaveCloud">
        <span class="scf-ms scf-ms--fill scf-cloud-ico" aria-hidden="true">cloud_upload</span>
        添加至云端
      </el-button>
    </footer>
  </div>
</template>

<style scoped>
.scf {
  --scf-surface: #f7f9fb;
  --scf-primary: #0050cb;
  --scf-primary-strong: #0066ff;
  --scf-on-variant: #424656;
  --scf-outline: #c2c6d8;
  --scf-error: #ba1a1a;
  --scf-tertiary: #a33200;
  --scf-secondary-container: #9bb4fe;
  --scf-on-secondary-container: #f8f7ff;
  --scf-error-container: #ffdad6;
  min-height: 100dvh;
  background: var(--scf-surface);
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: env(safe-area-inset-bottom);
}

.scf-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.375rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  vertical-align: middle;
  user-select: none;
}

.scf-ms--sm {
  font-size: 1.25rem;
}

.scf-ms--primary {
  color: var(--scf-primary-strong);
}

.scf-ms--white {
  color: #fff;
}

.scf-ms--fill {
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}

.scf-header {
  position: sticky;
  top: 0;
  z-index: 50;
  flex-shrink: 0;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 0.5rem;
  padding: max(0.75rem, env(safe-area-inset-top)) 0.75rem 0.875rem;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  box-shadow: 0 8px 32px rgba(25, 28, 30, 0.06);
}

.scf-back {
  justify-self: start;
  width: 2.25rem;
  height: 2.25rem;
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.scf-back-img {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
  display: block;
  pointer-events: none;
}

.scf-back:focus-visible {
  outline: 2px solid var(--scf-primary-strong);
  outline-offset: 2px;
}

.scf-title {
  margin: 0;
  justify-self: center;
  text-align: center;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.scf-header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  justify-self: end;
  min-width: 0;
  gap: 0.5rem;
}

.scf-inst-badge {
  font-size: 11px;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
  background: rgba(0, 80, 203, 0.08);
  color: var(--scf-primary);
}

.scf-del-btn {
  border: none;
  background: transparent;
  color: var(--scf-error);
  font-size: 13px;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
}

.scf-del-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.scf-fork-hint {
  margin: 0 0 0.75rem;
  font-size: 12px;
  color: var(--scf-on-variant);
  text-align: center;
}

.scf-main {
  padding: 1.25rem 1rem 0;
  max-width: 32rem;
  margin: 0 auto;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.scf-main-pad {
  height: 6rem;
}

.scf-section {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.scf-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0.25rem;
}

.scf-section-head--plain {
  justify-content: flex-start;
}

.scf-section-head-left {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.scf-play-hint {
  margin: 0;
  font-size: 0.75rem;
  color: #64748b;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
}

.scf-section-title {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scf-on-variant);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.scf-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
  background: var(--scf-secondary-container);
  color: var(--scf-on-secondary-container);
}

.scf-card {
  background: #fff;
  border-radius: 0.875rem;
  padding: 1.15rem 1rem;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scf-stack {
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}

.scf-grid2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

@media (max-width: 380px) {
  .scf-grid2 {
    grid-template-columns: 1fr;
  }
}

.scf-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.scf-lbl {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--scf-on-variant);
  padding-left: 0.15rem;
}

.scf-seg {
  display: flex;
  gap: 0.25rem;
  padding: 0.25rem;
  background: #f2f4f6;
  border-radius: 0.5rem;
}

.scf-seg-btn {
  flex: 1;
  border: none;
  border-radius: 0.375rem;
  padding: 0.5rem 0.35rem;
  font-size: 0.875rem;
  font-weight: 600;
  font-family: inherit;
  color: var(--scf-on-variant);
  background: transparent;
  cursor: pointer;
  transition:
    background 0.15s,
    box-shadow 0.15s,
    color 0.15s;
}

.scf-seg-btn:hover {
  background: rgba(255, 255, 255, 0.55);
}

.scf-seg-btn.is-active {
  background: #fff;
  color: var(--scf-primary-strong);
  box-shadow: 0 1px 4px rgba(25, 28, 30, 0.08);
}

.scf-el-inp :deep(.el-input__wrapper) {
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  padding-left: 0.9rem;
}

.scf-el-inp :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px rgba(0, 102, 255, 0.35) inset;
}

.scf-time-hit {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 2.5rem;
  padding: 0.55rem 0.9rem;
  border: none;
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition:
    box-shadow 0.15s,
    background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-time-hit:hover {
  background: rgba(242, 244, 246, 0.85);
}

.scf-time-hit:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.28);
}

.scf-time-hit-val {
  font-size: 0.9375rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--scf-primary-strong);
}

.scf-time-hit-ico {
  flex-shrink: 0;
  opacity: 0.65;
  color: var(--scf-primary-strong);
}

.scf-el-inp--danger :deep(.el-input__inner) {
  color: var(--scf-error);
  font-weight: 700;
}

.scf-el-inp--profit :deep(.el-input__inner) {
  color: var(--scf-tertiary);
  font-weight: 700;
}

.scf-el-select {
  width: 100%;
}

.scf-el-select :deep(.el-select__wrapper) {
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  min-height: 2.5rem;
}

.scf-suffix-wrap {
  position: relative;
}

.scf-el-inp--suffix :deep(.el-input__wrapper) {
  padding-right: 3rem;
}

.scf-suffix {
  position: absolute;
  right: 0.85rem;
  top: 50%;
  transform: translateY(-50%);
  font-size: 0.8125rem;
  font-weight: 700;
  color: #727687;
  pointer-events: none;
}

.scf-readonly {
  min-height: 2.5rem;
  padding: 0.55rem 0.9rem;
  border-radius: 0.5rem;
  background: rgba(230, 232, 234, 0.35);
  border: 1px solid rgba(194, 198, 216, 0.35);
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--scf-on-variant);
  display: flex;
  align-items: center;
}

.scf-tip {
  border-left: 4px solid var(--scf-error);
  border-radius: 0 0.5rem 0.5rem 0;
  padding: 0.65rem 0.75rem;
  background: rgba(255, 218, 214, 0.45);
}

.scf-tip p {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 500;
  line-height: 1.55;
  color: var(--scf-error);
}

.scf-mode-card {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.95rem 1rem;
  border: none;
  border-radius: 0.875rem;
  background: rgba(0, 102, 255, 0.06);
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-mode-card:hover {
  background: rgba(0, 102, 255, 0.1);
}

.scf-mode-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
  flex: 1 1 0;
}

.scf-mode-right {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.35rem;
  max-width: min(11rem, 42%);
  padding-top: 0.1rem;
}

.scf-mode-err {
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--scf-error);
  text-align: right;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.scf-mode-selected {
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--el-color-primary, #0050cb);
  text-align: right;
}

.scf-mode-unset {
  margin: 0;
  font-size: 11px;
  font-weight: 500;
  line-height: 1.35;
  color: #94a3b8;
  text-align: right;
}

.scf-mode-share-hint {
  margin: 0.375rem 0 0;
  font-size: 11px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.scf-mode-ico-bg {
  flex-shrink: 0;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 999px;
  background: var(--scf-primary-strong);
  display: flex;
  align-items: center;
  justify-content: center;
}

.scf-mode-texts {
  text-align: left;
  min-width: 0;
}

.scf-mode-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--scf-primary-strong);
}

.scf-mode-sub {
  margin: 0.15rem 0 0;
  font-size: 11px;
  color: var(--scf-on-variant);
  opacity: 0.78;
}

.scf-mode-gear {
  flex-shrink: 0;
  transition: transform 0.15s;
}

.scf-mode-card:hover .scf-mode-gear {
  transform: translateX(3px);
}

.scf-add-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
  padding: 0.35rem 0.65rem;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: var(--scf-primary-strong);
  font-size: 0.8125rem;
  font-weight: 700;
  font-family: inherit;
  cursor: pointer;
  transition: background 0.15s;
}

.scf-add-btn:hover {
  background: rgba(0, 80, 203, 0.06);
}

.scf-groups-stack {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.scf-group-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  flex-wrap: wrap;
  padding: 0.65rem 1rem;
  border-bottom: 1px solid rgba(194, 198, 216, 0.2);
  background: #fff;
  min-width: 0;
}

.scf-group-title {
  margin: 0;
  flex-shrink: 0;
  font-size: 0.875rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.01em;
  color: var(--scf-primary-strong);
}

.scf-content-toolbar--group {
  flex: 1;
  display: flex;
  justify-content: flex-end;
  align-items: stretch;
  align-self: stretch;
  min-width: min(12rem, 100%);
  border-bottom: none;
}

.scf-content-toolbar--group .scf-tb-btn {
  flex: 0 1 auto;
  padding: 0.5rem 0.55rem;
}

.scf-content-card {
  background: #fff;
  border-radius: 0.875rem;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scf-content-toolbar {
  display: flex;
  border-bottom: 1px solid rgba(194, 198, 216, 0.2);
}

.scf-tb-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.75rem 0.25rem;
  border: none;
  border-right: 1px solid rgba(194, 198, 216, 0.2);
  background: #fff;
  font-size: 0.75rem;
  font-weight: 700;
  font-family: inherit;
  color: var(--scf-primary-strong);
  cursor: pointer;
  transition: background 0.15s;
}

.scf-tb-btn:last-child {
  border-right: none;
}

.scf-tb-btn:hover {
  background: #f2f4f6;
}

.scf-tb-btn--muted {
  color: var(--scf-on-variant);
}

.scf-tb-btn--danger {
  color: var(--scf-error);
}

.scf-textarea-wrap {
  padding: 1rem;
}

.scf-area :deep(.el-textarea__inner) {
  border: none;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  padding: 1rem;
  font-size: 0.9375rem;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  line-height: 1.55;
  box-shadow: none;
}

.scf-area :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.scf-area-meta {
  margin-top: 0.65rem;
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 11px;
  color: #727687;
  padding: 0 0.2rem;
}

.scf-footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 50;
  padding: 0.85rem 1rem max(1rem, env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  box-shadow: 0 -10px 40px rgba(25, 28, 30, 0.06);
}

.scf-cloud-btn {
  width: 100%;
  height: 3.25rem;
  margin: 0;
  border-radius: 0.75rem;
  font-weight: 700;
  font-size: 1rem;
  border: none;
  box-shadow: 0 8px 24px rgba(0, 102, 255, 0.22);
}

.scf-cloud-ico {
  margin-right: 0.35rem;
  font-size: 1.35rem;
  vertical-align: -0.15em;
}

/* ----- 运行时段弹窗（滚轮） ----- */
.scf-tw {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding-bottom: 0.15rem;
}

.scf-tw-wheel-wrap {
  position: relative;
  padding: 0.35rem 0 0.15rem;
}

.scf-tw-highlight {
  position: absolute;
  left: 0.3rem;
  right: 0.3rem;
  top: 50%;
  transform: translateY(-50%);
  height: 44px;
  border-radius: 0.5rem;
  background: rgba(0, 102, 255, 0.09);
  pointer-events: none;
  z-index: 0;
}

.scf-tw-row {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: stretch;
  justify-content: center;
  gap: 0.2rem;
}

.scf-tw-mask {
  position: relative;
  flex: 1 1 0;
  min-width: 0;
  border-radius: 0.65rem;
  background: rgba(247, 249, 251, 0.92);
  -webkit-mask-image: linear-gradient(to bottom, transparent 0%, #000 14%, #000 86%, transparent 100%);
  mask-image: linear-gradient(to bottom, transparent 0%, #000 14%, #000 86%, transparent 100%);
}

.scf-tw-mask--hour {
  max-width: 5rem;
}

.scf-tw-mask--min {
  max-width: 4.35rem;
}

.scf-tw-scroll {
  height: 220px;
  overflow-y: auto;
  scroll-snap-type: y mandatory;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.scf-tw-scroll::-webkit-scrollbar {
  width: 0;
  height: 0;
}

.scf-tw-spacer {
  height: 88px;
  flex-shrink: 0;
}

.scf-tw-cell {
  height: 44px;
  scroll-snap-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.0625rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: #9499ae;
  cursor: pointer;
  user-select: none;
  transition:
    color 0.12s,
    font-weight 0.12s,
    transform 0.12s;
}

.scf-tw-cell.is-sel {
  color: var(--scf-primary-strong);
  font-weight: 800;
  font-size: 1.125rem;
}

.scf-tw-colon {
  align-self: center;
  font-weight: 800;
  font-size: 1.25rem;
  color: var(--scf-primary-strong);
  padding: 0 0.05rem;
  line-height: 1;
}

.scf-tw-summary {
  display: flex;
  gap: 0.65rem;
  padding: 0.65rem;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.98);
}

.scf-tw-sum-half {
  flex: 1;
  min-width: 0;
  padding: 0.55rem 0.6rem;
  border: none;
  border-radius: 0.55rem;
  background: transparent;
  cursor: pointer;
  text-align: center;
  font-family: inherit;
  transition:
    background 0.15s,
    box-shadow 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-tw-sum-half:hover {
  background: rgba(255, 255, 255, 0.55);
}

.scf-tw-sum-half.is-active {
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 4px 22px rgba(25, 28, 30, 0.06);
}

.scf-tw-sum-half:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.35);
  outline-offset: 2px;
}

.scf-tw-sum-lbl {
  display: block;
  font-size: 11px;
  font-weight: 600;
  color: #727687;
  margin-bottom: 0.25rem;
}

.scf-tw-sum-val {
  display: block;
  font-size: 1rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
}

.scf-tw-confirm {
  width: 100%;
  margin: 0;
  height: 3rem;
  border-radius: 0.75rem;
  font-weight: 700;
  font-size: 1rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: none;
  box-shadow: 0 8px 24px rgba(0, 102, 255, 0.22);
}

.scf-tw-check {
  width: 1.4rem;
  height: 1.4rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.28);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.scf-tw-check-ico {
  font-size: 1rem !important;
}

/* ----- 运行类型方案内容面板 ----- */
.scf-panel {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.scf-panel-field {
  padding-top: 0.15rem;
}

.scf-run-tip {
  margin: 0;
  font-size: 11px;
  font-weight: 500;
  line-height: 1.6;
  color: #727687;
}

.scf-run-tip--banner {
  padding: 0.65rem 1rem;
  border-radius: 0.75rem;
  background: rgba(0, 80, 203, 0.06);
  color: var(--scf-primary);
}

.scf-radio-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 0.15rem 1.1rem;
}

/* 局数列表 */
.scf-jushu-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.scf-jushu-row {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  padding: 0.75rem 0.85rem;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
}

.scf-jushu-main {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
  min-width: 0;
}

.scf-jushu-no {
  flex-shrink: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scf-primary-strong);
}

.scf-jushu-content {
  min-width: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  color: #191c1e;
  word-break: break-all;
  white-space: pre-wrap;
}

.scf-jushu-side {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.scf-jushu-jump {
  font-size: 11px;
  font-weight: 600;
  color: var(--scf-on-variant);
}

.scf-jushu-del {
  margin-left: auto;
  border: none;
  background: transparent;
  color: var(--scf-error);
  cursor: pointer;
  padding: 0.2rem 0.3rem;
  border-radius: 0.4rem;
  line-height: 0;
}

.scf-jushu-del:hover {
  background: rgba(186, 26, 26, 0.08);
}

.scf-jushu-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.scf-jushu-num {
  width: 100%;
}

/* 开某投某映射表 */
.scf-trig-grid {
  display: grid;
  grid-template-columns: 3rem 3rem 1fr 1fr;
  align-items: center;
  gap: 0.6rem;
}

.scf-trig-grid--head span {
  font-size: 11px;
  font-weight: 700;
  color: var(--scf-on-variant);
  letter-spacing: 0.02em;
}

.scf-trig-grid.is-off .scf-trig-open {
  opacity: 0.35;
}

.scf-trig-open {
  font-size: 0.9375rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--scf-primary-strong);
  text-align: center;
  padding: 0.25rem 0;
  border-radius: 0.45rem;
  background: rgba(0, 80, 203, 0.06);
}

/* 冷热温 */
.scf-hcw-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.scf-hcw-ctrl {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.scf-hcw-pos {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem 0.85rem;
  border-radius: 0.75rem;
  background: rgba(247, 249, 251, 0.9);
}

.scf-hcw-pos-name {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scf-on-variant);
}

.scf-hcw-tier {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.scf-hcw-tier-lbl {
  flex-shrink: 0;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 0.45rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
}

.scf-hcw-tier--hot .scf-hcw-tier-lbl {
  background: var(--el-color-primary, #0050cb);
  color: #fff;
}

.scf-hcw-tier--warm .scf-hcw-tier-lbl {
  background: var(--el-color-primary-light-5, #80a7e5);
  color: #fff;
}

.scf-hcw-tier--cold .scf-hcw-tier-lbl {
  background: var(--el-color-primary-light-9, #ecf2fc);
  color: var(--el-color-primary, #0050cb);
}

.scf-hcw-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.scf-hcw-chip {
  width: 2.1rem;
  height: 2.1rem;
  border: none;
  border-radius: 0.55rem;
  font-size: 0.9375rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  font-family: inherit;
  cursor: pointer;
  background: #f2f4f6;
  color: var(--scf-on-variant);
  transition:
    box-shadow 0.15s,
    background 0.15s,
    color 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-hcw-tier--hot .scf-hcw-chip {
  background: rgba(0, 80, 203, 0.14);
  color: var(--el-color-primary, #0050cb);
}

.scf-hcw-tier--warm .scf-hcw-chip {
  background: var(--el-color-primary-light-5, #80a7e5);
  color: #fff;
}

.scf-hcw-tier--cold .scf-hcw-chip {
  background: var(--el-color-primary-light-9, #ecf2fc);
  color: var(--el-color-primary, #0050cb);
}

.scf-hcw-chip.is-on {
  background: var(--el-color-primary, #0050cb);
  color: #fff;
  box-shadow: 0 4px 14px rgba(0, 80, 203, 0.28);
}

.scf-hcw-pool {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  padding: 0.75rem 0.85rem;
  border-radius: 0.75rem;
  background: rgba(0, 80, 203, 0.05);
}

.scf-hcw-pool-title {
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scf-primary);
}

.scf-hcw-pool-units {
  font-size: 11px;
  font-weight: 600;
  color: var(--scf-on-variant);
}

.scf-hcw-pool-line {
  margin: 0;
  display: flex;
  gap: 0.5rem;
  font-size: 0.875rem;
  line-height: 1.6;
}

.scf-hcw-pool-pos {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 700;
  color: var(--scf-on-variant);
  align-self: center;
}

.scf-hcw-pool-vals {
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  font-weight: 600;
  color: #191c1e;
  word-break: break-all;
}

/* 随机出号 */
.scf-rd-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.scf-rd-pos {
  flex-shrink: 0;
  min-width: 3.2rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scf-on-variant);
}

.scf-rd-preview {
  font-size: 0.8125rem;
  font-weight: 600;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  color: var(--scf-primary-strong);
  word-break: break-all;
}

.scf-rd-actions {
  display: flex;
  justify-content: flex-start;
}

/* 内置计画 */
.scf-bp-summary {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.85rem 1rem;
  border-radius: 0.75rem;
  background: rgba(0, 80, 203, 0.06);
}

.scf-bp-summary-main {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}

.scf-bp-summary-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  line-height: 1.6;
  color: var(--scf-primary);
  word-break: break-all;
}

.scf-bp-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.scf-bp-item {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  width: 100%;
  padding: 0.8rem 0.9rem;
  border: none;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition:
    background 0.15s,
    box-shadow 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-bp-item:hover {
  background: rgba(242, 244, 246, 0.95);
}

.scf-bp-item.is-sel {
  background: rgba(0, 80, 203, 0.07);
  box-shadow: 0 0 0 1.5px rgba(0, 80, 203, 0.45) inset;
}

.scf-bp-radio {
  flex-shrink: 0;
  width: 1.05rem;
  height: 1.05rem;
  border-radius: 999px;
  box-shadow: 0 0 0 1.5px var(--scf-outline) inset;
  background: #fff;
  transition: box-shadow 0.15s;
}

.scf-bp-radio.is-on {
  box-shadow: 0 0 0 5px var(--el-color-primary, #0050cb) inset;
}

.scf-bp-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.scf-bp-name {
  font-size: 0.9375rem;
  font-weight: 700;
  color: #191c1e;
  word-break: break-all;
}

.scf-bp-meta {
  font-size: 11px;
  font-weight: 500;
  line-height: 1.6;
  color: var(--scf-on-variant);
}

.scf-bp-actions {
  display: flex;
  gap: 0.5rem;
}
</style>

<style>
/* Teleport 遮罩需全局类名（modal-class） */
.scf-tw-overlay.el-overlay {
  background-color: rgba(15, 23, 42, 0.42);
  backdrop-filter: blur(26px);
  -webkit-backdrop-filter: blur(26px);
}

.scf-tw-dialog.el-dialog {
  padding: 0;
  border-radius: 1rem;
  overflow: hidden;
  box-shadow: 0 28px 56px rgba(25, 28, 30, 0.08);
}

.scf-tw-dialog .el-dialog__header {
  padding: 1rem 1rem 0.25rem;
  margin-right: 0;
}

.scf-tw-dialog .el-dialog__title {
  font-family:
    'Plus Jakarta Sans',
    'Noto Sans SC',
    system-ui,
    sans-serif;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.scf-tw-dialog .el-dialog__body {
  padding: 0 1rem 1.15rem;
}

@media (max-width: 420px) {
  .scf-tw-dialog.el-dialog {
    width: calc(100vw - 1.5rem) !important;
    max-width: 22rem;
    margin-left: auto;
    margin-right: auto;
  }
}
</style>
