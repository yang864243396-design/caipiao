<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog } from '@/utils/confirmDialog'
import { ApiError } from '@/api/client'
import { addSchemeToCloud } from '@/api/schemes/addToCloud'
import {
  checkSchemeNameAvailable,
  createScheme,
  fetchSchemeDefinitions,
  updateSchemeDefinition,
  deleteSchemeDefinition,
  fetchHotColdWarmTiers,
} from '@/api/schemes/definitions'
import type {
  SchemeJushuRow,
  SchemeTriggerBet,
  SchemeTriggerRow,
  SchemeHotColdWarm,
  SchemeHotColdPickType,
  SchemeRotateStrategy,
  SchemeRandomDraw,
  UpdateSchemeInput,
} from '@/api/schemes/definitions'
import { fetchSchemeFavorites, type SchemeFavoriteRow } from '@/api/schemes/favorites'
import { fetchLotterySchemeOptions } from '@/api/schemes/schemeOptions'
import { fetchGameDraws } from '@/api/games/detail'
import { fetchPlayTree } from '@/api/games/lotteries'
import { parseSchemeKind } from '@/utils/schemeKind'
import DateTimePickerModal from '@/components/ui/DateTimePickerModal.vue'
import OptionPickerModal from '@/components/ui/OptionPickerModal.vue'
import type { OptionPickerItem } from '@/components/ui/OptionPickerModal.vue'
import { BET_MODE_OPTIONS, betUnitFromSchemeConfig, normalizeBetUnitValue } from '@/constants/betModeOptions'
import SchemeGroupPickPanel from '@/components/schemes/SchemeGroupPickPanel.vue'
import SchemeGroupInputPanel from '@/components/schemes/SchemeGroupInputPanel.vue'
import SchemeRenxuanDanshiPanel from '@/components/schemes/SchemeRenxuanDanshiPanel.vue'
import {
  adaptSchemeGroupContentForPlay,
  catalogFieldsFromPlayConfig,
  countBetUnits,
  groupContentPlaceholder,
  isYixingDingweiPlayConfig,
  isMaxBetUnitsExceededMessage,
  WEISHU_MAX_BET_UNITS,
  YIXING_MAX_PICKS_MSG,
  YIXING_MAX_PICKS_PER_POS,
  isRenxuanPositionDanshiConfig,
  playConfigSummary,
  validateGroupContent,
  validateSchemeGroups,
  schemeSoloBaoziError,
} from '@/utils/betPayload'
import { defaultPlaySelection, formatSubPlayLabel } from '@/utils/playConfig'
import { normalizeSchemeTimePairFromConfig, schemeTimeRangeError } from '@/utils/schemeDateTime'
import { usePublicLotteries } from '@/composables/usePublicLotteries'
import { usePlayTreeConfig } from '@/composables/usePlayTreeConfig'
import { longhuPickOptionsForConfig } from '@/utils/longhuPickOptions'
import {
  schemeGroupContentToInputBox,
  schemeGroupUsesDigitInput,
  schemeGroupUsesPickPanel,
  textPickOptionsForConfig,
} from '@/utils/pickPanelOptions'
import {
  filterPlayTypesForRunType,
  filterSubPlaysForRunType,
  isLonghuPlayConfigLike,
  isPc28HezhiConfigLike,
  isPc28ModeConfigLike,
  lotteryHasAdvTriggerPlay,
  supportsAdvTriggerPerPosColumns,
  supportsAdvTriggerPositionPicker,
  syncRunTypePlaySelection,
  validateRunTypePlaySelection,
} from '@/utils/runTypeMatrix'
import type { PlayTypeNode } from '@/types/playCatalog'
import {
  clearSchemeDraft,
  consumeSchemeEditRestoreSnapshot,
  draftMetaFromQuery,
  draftPatchFromSnapshot,
  isDraftSchemeId,
  loadSchemeDraft,
  saveSchemeDraft,
  saveSchemeEditRestoreSnapshot,
  type SchemeDraftMeta,
  type SchemeDraftSnapshot,
} from '@/utils/schemeDraftStorage'
import { syncDraftAdvancedTemplatesToServer } from '@/utils/draftAdvancedTemplates'
import { simBetFromSchemeConfig } from '@/utils/schemeSimBet'
import { PRIMARY_CURRENCIES, type PrimaryCurrency } from '@/api/guaji/accounts'

const route = useRoute()
const router = useRouter()
const { lotteries, load: loadLotteries, codeToLabel } = usePublicLotteries()

const SCHEME_CURRENCY_OPTIONS = PRIMARY_CURRENCIES

function normalizeSchemeCurrency(raw: unknown): PrimaryCurrency {
  const c = String(raw ?? '').trim().toUpperCase()
  if (c === 'TRX' || c === 'CNY') return c
  return 'USDT'
}

const schemeId = computed(() => String(route.params.schemeId ?? ''))
const isDraftScheme = computed(() => isDraftSchemeId(schemeId.value) || route.query.draft === '1')
const schemeKind = computed(() =>
  parseSchemeKind(route.query.kind ?? route.query.schemeKind),
)
const isCustomKind = computed(() => schemeKind.value === 'custom')
/** 新建草稿可改彩种/运行类型/玩法；已落库方案不可改（服务端 forbidden） */
const identityEditable = computed(() => isDraftScheme.value)

/** 开始/结束时间说明（气泡展示，不占文档流） */
const TIME_RANGE_HINT =
  '方案保存后将自动同步至精算云中心。开始时间与结束时间须同时填写，或同时留空表示无限期运行。'
/** 方案模式说明 */
const BET_MODE_HINT = '倍投设定与轮次设置共用同一份倍率配置，后保存者覆盖先前配置。'

/** false=正式运行，true=模拟运行 */
const simBet = ref(false)
const titleFromQuery = String(route.query.title ?? '')
const schemeName = ref(titleFromQuery ? decodeURIComponent(titleFromQuery) : '')
const shareStatus = ref<'private' | 'public'>('private')
const shareLocked = ref(false)
const cloudBusy = ref(false)
const schemeFunds = ref('10000')
/** 方案币种；历史未填默认 USDT */
const schemeCurrency = ref<PrimaryCurrency>('USDT')
/** 开始/结束时间；两者均留空表示无限期运行 */
const startTime = ref('')
const endTime = ref('')
const stopLoss = ref('')
const takeProfit = ref('')
const multCoeff = ref('1')
const betUnit = ref('2')
/** 方案内容按组划分，默认一组 */
const schemeGroups = ref<string[]>([''])

const lotteryCode = ref(String(route.query.lottery ?? ''))
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
/** 数字玩法方案内容改用输入框录入（对齐第三方，不点选） */
const schemeUsesDigitInput = computed(() => schemeGroupUsesDigitInput(schemePlayConfig.value))
const schemeUsesRenxuanDanshi = computed(() => isRenxuanPositionDanshiConfig(schemePlayConfig.value))

const groupInputPlaceholder = computed(() => groupContentPlaceholder(schemePlayConfig.value))

const gameNameDisplay = computed(() => {
  const id = lotteryCode.value
  const label = codeToLabel(id)
  if (label) return label
  return id || '—'
})

// ----- 顶部身份字段：方案名称 / 彩种 / 运行类型 / 玩法类型 / 子玩法 -----
type IdentityPickerKind = 'lottery' | 'runType' | 'playType' | 'subPlay'
const DEPRECATED_RUN_TYPES = new Set(['batch_fixed', 'dynamic_chase', 'plan_follow'])

const lotteryOptions = computed<OptionPickerItem[]>(() =>
  lotteries.value.map((l) => ({ label: l.displayName, value: l.code })),
)

const runTypeOptions = ref<OptionPickerItem[]>([
  { label: '定码轮换', value: 'fixed_rotate' },
  { label: '高级定码轮换', value: 'adv_fixed_rotate' },
  { label: '高级开某投某', value: 'adv_trigger_bet' },
  { label: '冷热出号', value: 'hot_cold_warm' },
  { label: '随机出号', value: 'random_draw' },
  { label: '内置计划', value: 'builtin_plan' },
  { label: '固定号码', value: 'fixed_number' },
])
const playTypeOptions = ref<OptionPickerItem[]>([])
const subPlayOptions = ref<OptionPickerItem[]>([])
const playTreeTypes = ref<PlayTypeNode[]>([])

const identityPickerOpen = ref(false)
const identityPickerKind = ref<IdentityPickerKind | null>(null)

function groupBetUnits(raw: string): number {
  const cfg = schemePlayConfig.value
  const r = validateGroupContent(cfg, raw)
  if (r.ok) return r.betUnits
  // 输入过程中校验未通过时仍按玩法计注（单式会去重），避免短暂显示原始逗号段数
  return countBetUnits(cfg, raw)
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
  hot_cold_warm: '冷热出号',
  random_draw: '随机出号',
  builtin_plan: '内置计划',
  fixed_number: '固定号码',
}

/** batch_fixed / dynamic_chase / plan_follow 等废弃或未知值统一兜底为定码轮换 */
function normalizeRunTypeId(raw: unknown): RunTypeId {
  const v = String(Array.isArray(raw) ? raw[0] ?? '' : raw ?? '').trim()
  if ((RUN_TYPE_IDS as readonly string[]).includes(v)) return v as RunTypeId
  return 'fixed_rotate'
}

/** 远端 def.config.runTypeId 为准，路由 query.runType 兜底 */
const runTypeId = ref<RunTypeId>(normalizeRunTypeId(route.query.runType))
const runTypeLabel = computed(() => RUN_TYPE_LABELS[runTypeId.value])
const isBuiltinPlan = computed(() => runTypeId.value === 'builtin_plan')

const availableRunTypeOptions = computed<OptionPickerItem[]>(() => {
  let opts = runTypeOptions.value.filter((o) => !DEPRECATED_RUN_TYPES.has(String(o.value)))
  if (playTreeTypes.value.length > 0 && !lotteryHasAdvTriggerPlay(playTreeTypes.value)) {
    opts = opts.filter((o) => String(o.value) !== 'adv_trigger_bet')
  }
  return opts
})

const filteredPlayTypeOptions = computed<OptionPickerItem[]>(() =>
  filterPlayTypesForRunType(runTypeId.value, playTypeOptions.value, playTreeTypes.value),
)

const filteredSubPlayOptions = computed<OptionPickerItem[]>(() => {
  const typeNode = playTreeTypes.value.find((t) => t.typeId === playTypeId.value)
  const fromTree = (typeNode?.subPlays ?? []).map((s) => ({
    label: formatSubPlayLabel(s.label),
    value: s.subId,
  }))
  const source = fromTree.length > 0 ? fromTree : subPlayOptions.value
  return filterSubPlaysForRunType(
    runTypeId.value,
    source,
    playTypeId.value,
    playTreeTypes.value,
  )
})

const playTypeLabelDisplay = computed(() => {
  const fromOpts = filteredPlayTypeOptions.value.find((o) => String(o.value) === playTypeId.value)?.label
  if (fromOpts) return fromOpts
  return schemePlayConfig.value.playTypeLabel || playTypeId.value || '—'
})

const subPlayLabelDisplay = computed(() => {
  const fromOpts = filteredSubPlayOptions.value.find((o) => String(o.value) === subPlayId.value)?.label
  if (fromOpts) return fromOpts
  return schemePlayConfig.value.playMethodLabel || subPlayId.value || '—'
})

function applyRunTypePlaySync() {
  if (isBuiltinPlan.value || !playTreeTypes.value.length) return
  const synced = syncRunTypePlaySelection({
    runTypeId: runTypeId.value,
    playTypeId: playTypeId.value,
    subPlayId: subPlayId.value,
    playTreeTypes: playTreeTypes.value,
    playTypeOptions: playTypeOptions.value,
    subPlayOptions: subPlayOptions.value,
  })
  runTypeId.value = normalizeRunTypeId(synced.runTypeId)
  playTypeId.value = synced.playTypeId
  subPlayId.value = synced.subPlayId
}

function ensureSelectedInOptions(
  options: OptionPickerItem[],
  selected: { value: string },
  fallback: string,
) {
  if (options.length === 0) return
  if (!options.some((o) => String(o.value) === selected.value)) {
    selected.value = String(options[0]?.value ?? fallback)
  }
}

async function loadRunTypeOptions(code: string) {
  if (!code) return
  try {
    const data = await fetchLotterySchemeOptions(code)
    const fresh = data.runTypes
      .filter((o) => !DEPRECATED_RUN_TYPES.has(String(o.value)))
      .map((o) => {
        const value = String(o.value)
        const local = RUN_TYPE_LABELS[value as RunTypeId]
        return local ? { ...o, value, label: local } : { ...o, value }
      })
    if (fresh.length) runTypeOptions.value = fresh
  } catch {
    /* 保留默认运行类型 */
  }
}

async function loadIdentityPlayTree(code: string) {
  if (!code) return
  try {
    const tree = await fetchPlayTree(code)
    playTreeTypes.value = tree.playTypes
    playTypeOptions.value = tree.playTypes.map((t) => ({
      label: t.label,
      value: t.typeId,
    }))
    const hasType = tree.playTypes.some((t) => t.typeId === playTypeId.value)
    if (!hasType) {
      const def = defaultPlaySelection(tree)
      playTypeId.value = def.typeId
      subPlayId.value = def.subId
    }
    const typeNode = tree.playTypes.find((t) => t.typeId === playTypeId.value)
    subPlayOptions.value = (typeNode?.subPlays ?? []).map((s) => ({
      label: formatSubPlayLabel(s.label),
      value: s.subId,
    }))
    ensureSelectedInOptions(subPlayOptions.value, subPlayId, subPlayId.value)
    applyRunTypePlaySync()
  } catch (e) {
    playTreeTypes.value = []
    playTypeOptions.value = []
    subPlayOptions.value = []
    if (identityEditable.value) {
      ElMessage.error(e instanceof ApiError ? e.message : '加载玩法树失败')
    }
  }
}

const identityPickerTitle = computed(() => {
  switch (identityPickerKind.value) {
    case 'lottery':
      return '选择彩种'
    case 'runType':
      return '运行类型'
    case 'playType':
      return '玩法类型'
    case 'subPlay':
      return '子玩法'
    default:
      return ''
  }
})

const identityPickerOptions = computed<OptionPickerItem[]>(() => {
  switch (identityPickerKind.value) {
    case 'lottery':
      return lotteryOptions.value
    case 'runType':
      return availableRunTypeOptions.value
    case 'playType':
      return filteredPlayTypeOptions.value
    case 'subPlay':
      return filteredSubPlayOptions.value
    default:
      return []
  }
})

const identityPickerSelectedValue = computed(() => {
  switch (identityPickerKind.value) {
    case 'lottery':
      return lotteryCode.value
    case 'runType':
      return runTypeId.value
    case 'playType':
      return playTypeId.value
    case 'subPlay':
      return subPlayId.value
    default:
      return ''
  }
})

function openIdentityPicker(k: IdentityPickerKind) {
  if (!identityEditable.value) return
  identityPickerKind.value = k
  identityPickerOpen.value = true
}

function onIdentityPickerConfirm(val: string | number) {
  const v = String(val)
  const k = identityPickerKind.value
  if (k === 'lottery') {
    lotteryCode.value = v
    void loadRunTypeOptions(v)
    void loadIdentityPlayTree(v)
  } else if (k === 'runType') {
    runTypeId.value = normalizeRunTypeId(v)
    applyRunTypePlaySync()
  } else if (k === 'playType') {
    playTypeId.value = v
    const local = playTreeTypes.value.find((t) => t.typeId === v)
    if (local) {
      subPlayOptions.value = (local.subPlays ?? []).map((s) => ({
        label: formatSubPlayLabel(s.label),
        value: s.subId,
      }))
    }
    applyRunTypePlaySync()
  } else if (k === 'subPlay') {
    subPlayId.value = v
  }
  identityPickerKind.value = null
}

function onIdentityPickerCancel() {
  identityPickerKind.value = null
}

function labelOf(list: OptionPickerItem[] | readonly OptionPickerItem[], id: string) {
  return list.find((o) => String(o.value) === id)?.label ?? ''
}

watch(availableRunTypeOptions, (opts) => {
  if (!identityEditable.value || !opts.length) return
  if (!opts.some((o) => String(o.value) === runTypeId.value)) {
    runTypeId.value = 'fixed_rotate'
  }
})

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

/** 固定取码：仅展示并保存第 1 组 */
const displayedGroupIndexes = computed(() =>
  runTypeId.value === 'fixed_number' ? [0] : schemeGroups.value.map((_, i) => i),
)

// --- adv_fixed_rotate 高级定码轮换：局数列表 ---
const jushuList = ref<SchemeJushuRow[]>([])
const jushuDialogVisible = ref(false)
/** null=添加；非 null=编辑列表下标 */
const jushuEditIdx = ref<number | null>(null)
const jushuForm = ref<SchemeJushuRow>({ ju: 1, content: '', afterHit: 1, afterMiss: 1 })
const jushuDialogTitle = computed(() => (jushuEditIdx.value != null ? '编辑局数' : '添加局数'))
const jushuDialogConfirmLabel = computed(() => (jushuEditIdx.value != null ? '保存修改' : '确认添加'))

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
  const groups = schemeGroups.value
    .map((g) => String(g ?? '').replace(/\r/g, ''))
    .filter((g) => g.trim())
  if (!groups.length) return
  jushuList.value = groups.map((content, i) => ({ ju: i + 1, content, afterHit: 1, afterMiss: 1 }))
}

function openJushuDialog(): void {
  jushuEditIdx.value = null
  const maxJu = jushuList.value.reduce((m, r) => Math.max(m, r.ju), 0)
  jushuForm.value = { ju: maxJu + 1, content: '', afterHit: 1, afterMiss: 1 }
  jushuDialogVisible.value = true
}

function openJushuEditDialog(idx: number): void {
  const row = jushuList.value[idx]
  if (!row) return
  jushuEditIdx.value = idx
  jushuForm.value = {
    ju: row.ju,
    content: String(row.content ?? '').replace(/\r/g, ''),
    afterHit: Math.max(1, Math.trunc(Number(row.afterHit)) || 1),
    afterMiss: Math.max(1, Math.trunc(Number(row.afterMiss)) || 1),
  }
  jushuDialogVisible.value = true
}

function closeJushuDialog(): void {
  jushuDialogVisible.value = false
  jushuEditIdx.value = null
}

function confirmJushuDialog(): void {
  const f = jushuForm.value
  if (!Number.isInteger(f.ju) || f.ju < 1) {
    ElMessage.warning('局数须为不小于 1 的整数')
    return
  }
  const editIdx = jushuEditIdx.value
  if (jushuList.value.some((r, i) => r.ju === f.ju && i !== editIdx)) {
    ElMessage.warning(`第 ${f.ju} 局已存在，局数不可重复`)
    return
  }
  // 定位胆多位含前导空行，禁止 trim
  const content = String(f.content ?? '').replace(/\r/g, '')
  if (!content.trim()) {
    ElMessage.warning('投注号码不能为空')
    return
  }
  const contentCheck = validateGroupContent(schemePlayConfig.value, content)
  if (!contentCheck.ok) {
    ElMessage.warning(contentCheck.message)
    return
  }
  const nextRow: SchemeJushuRow = {
    ju: f.ju,
    content: contentCheck.normalized,
    afterHit: Math.max(1, f.afterHit),
    afterMiss: Math.max(1, f.afterMiss),
  }
  if (editIdx != null && editIdx >= 0 && editIdx < jushuList.value.length) {
    const next = jushuList.value.slice()
    next[editIdx] = nextRow
    jushuList.value = next.sort((a, b) => a.ju - b.ju)
  } else {
    jushuList.value = [...jushuList.value, nextRow].sort((a, b) => a.ju - b.ju)
  }
  closeJushuDialog()
}

function removeJushuRow(idx: number): void {
  jushuList.value.splice(idx, 1)
}

/** 局数列表展示：按录入框原版（压缩）格式，不把引擎换行渲染成多行 */
function formatJushuContentDisplay(content: string): string {
  const raw = String(content ?? '').replace(/\r/g, '')
  if (!raw.trim()) return ''
  if (schemeUsesDigitInput.value) {
    const box = schemeGroupContentToInputBox(raw, schemePlayConfig.value)
    if (box) return box
  }
  if (raw.includes('\n')) {
    return raw
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean)
      .join(', ')
  }
  return raw
}

// --- adv_trigger_bet 高级开某投某 ---
const PC28_HEZHI_VALUES = Array.from({ length: 28 }, (_, i) => String(i))
const longhuPickValues = computed(() => longhuPickOptionsForConfig(schemePlayConfig.value))
const triggerRows = ref<SchemeTriggerRow[]>([])
const triggerMode = ref<SchemeTriggerBet['mode']>('always_pos')
/** 「全部随机」每格随机出号个数（位数），由工具条步进器控制 */
const triggerRandomCount = ref(1)
/** 定位胆投注位（可多选）：0=万 … 4=个（统一一星定位胆时必选） */
const triggerPositionIdxs = ref<number[]>([0])
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
      // 换玩法后默认勾选段内全部投注位（如前三：万/千/百）
      if (runTypeId.value === 'adv_trigger_bet') {
        triggerPositionIdxs.value = defaultTriggerPositionIdxs()
      }
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

/** 一星/定位胆：展示投注位芯片 */
const showTriggerPositionPicker = computed(() => {
  if (runTypeId.value !== 'adv_trigger_bet') return false
  if (isLonghuPlay.value || isTriggerTextPlay.value) return false
  return supportsAdvTriggerPositionPicker(schemePlayConfig.value)
})

/** 前三直选复式等：按位分列正投/反投（不展示投注位） */
const showTriggerPerPosColumns = computed(() => {
  if (runTypeId.value !== 'adv_trigger_bet') return false
  if (isLonghuPlay.value || isTriggerTextPlay.value) return false
  return supportsAdvTriggerPerPosColumns(schemePlayConfig.value)
})

/** 位名展示：万 → 万位 */
function triggerPosName(posLabel: string): string {
  const base = String(posLabel ?? '')
    .trim()
    .replace(/位$/, '')
  return `${base || '位'}位`
}

/** pos/neg 按行存「位1\n位2\n位3」；旧单值兼容展开到各位 */
function triggerFieldParts(raw: string, len: number): string[] {
  const n = Math.max(1, len)
  const text = String(raw ?? '')
  if (!text.includes('\n') && !text.includes('\r')) {
    const one = text.trim()
    return Array.from({ length: n }, () => one)
  }
  const parts = text.split(/\r?\n/).map((s) => s.trim())
  return Array.from({ length: n }, (_, i) => parts[i] ?? '')
}

function getTriggerFieldCell(row: SchemeTriggerRow, field: 'pos' | 'neg', idx: number): string {
  return triggerFieldParts(row[field], positionCount.value)[idx] ?? ''
}

function writeTriggerFieldCell(
  row: SchemeTriggerRow,
  field: 'pos' | 'neg',
  idx: number,
  raw: string,
): void {
  const parts = triggerFieldParts(row[field], positionCount.value)
  parts[idx] = String(raw ?? '')
  row[field] = parts.join('\n')
}

function commitTriggerFieldCell(row: SchemeTriggerRow, field: 'pos' | 'neg', idx: number): void {
  const parts = triggerFieldParts(row[field], positionCount.value)
  parts[idx] = sanitizeTriggerBetContent(parts[idx] ?? '')
  row[field] = parts.join('\n')
}

function defaultTriggerPositionIdxs(): number[] {
  const n = Math.max(1, positionCount.value)
  return Array.from({ length: n }, (_, i) => i)
}

function ensureTriggerPositions(): void {
  if (!showTriggerPositionPicker.value) return
  const n = Math.max(1, positionCount.value)
  const cur = triggerPositionIdxs.value.filter((i) => Number.isInteger(i) && i >= 0 && i < n)
  triggerPositionIdxs.value = cur.length ? cur : defaultTriggerPositionIdxs()
}

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

function normalizeTriggerPositionIdxs(raw: unknown, maxExclusive = 10): number[] {
  const max = Math.max(1, maxExclusive) - 1
  const out: number[] = []
  const seen = new Set<number>()
  const push = (n: number) => {
    if (!Number.isFinite(n)) return
    const idx = Math.min(max, Math.max(0, Math.trunc(n)))
    if (seen.has(idx)) return
    seen.add(idx)
    out.push(idx)
  }
  if (Array.isArray(raw)) {
    for (const item of raw) {
      if (typeof item === 'number') push(item)
      else if (typeof item === 'string' && /^\d+$/.test(item.trim())) push(Number(item.trim()))
    }
  } else if (typeof raw === 'number') {
    push(raw)
  } else if (typeof raw === 'string' && /^\d+$/.test(raw.trim())) {
    push(Number(raw.trim()))
  }
  out.sort((a, b) => a - b)
  return out.length ? out : [0]
}

function toggleTriggerPosition(idx: number): void {
  const cur = [...triggerPositionIdxs.value]
  const at = cur.indexOf(idx)
  if (at >= 0) {
    if (cur.length <= 1) return // 至少保留一位
    cur.splice(at, 1)
  } else {
    cur.push(idx)
    cur.sort((a, b) => a - b)
  }
  triggerPositionIdxs.value = cur
}

function applyTriggerBetFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const tb = raw as { rows?: unknown; mode?: unknown; positionIdx?: unknown; positionIdxs?: unknown }
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
  if (tb.positionIdxs != null || tb.positionIdx != null) {
    const n = Math.max(1, positionCount.value)
    const idxs = Array.isArray(tb.positionIdxs)
      ? normalizeTriggerPositionIdxs(tb.positionIdxs, n)
      : normalizeTriggerPositionIdxs(tb.positionIdx, n)
    triggerPositionIdxs.value = idxs
  }
}

/** 随机出号个数上限 = 号池大小（至少 1） */
const triggerRandomMax = computed(() => Math.max(1, triggerOpenValues().length))

/** 取 count 个不重复随机号码，逗号拼接（count 由「随机出号」步进器决定） */
function randomTriggerMultiValue(count: number): string {
  const pool = [...triggerOpenValues()]
  if (!pool.length) return '0'
  const n = Math.min(pool.length, Math.max(1, Math.trunc(count) || 1))
  for (let i = pool.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[pool[i], pool[j]] = [pool[j]!, pool[i]!]
  }
  return pool.slice(0, n).join(',')
}

/** 「全部随机」：纯前端一次性填表辅助，引擎下注不涉及随机；每格取「随机出号」个号 */
function randomFillTrigger(): void {
  const count = Math.min(triggerRandomMax.value, Math.max(1, Math.trunc(triggerRandomCount.value) || 1))
  triggerRandomCount.value = count
  const posN = Math.max(1, positionCount.value)
  for (const row of triggerRows.value) {
    if (showTriggerPerPosColumns.value) {
      row.pos = Array.from({ length: posN }, () => randomTriggerMultiValue(count)).join('\n')
      row.neg = Array.from({ length: posN }, () => randomTriggerMultiValue(count)).join('\n')
    } else {
      row.pos = randomTriggerMultiValue(count)
      row.neg = randomTriggerMultiValue(count)
    }
  }
  ElMessage.success(`已随机填充正投 / 反投号码（每格 ${count} 个号）`)
}

/** 规范化按位正投/反投（换行分位，每位内可逗号多号） */
function sanitizeTriggerPerPosField(raw: string): string {
  const n = Math.max(1, positionCount.value)
  return triggerFieldParts(raw, n)
    .map((cell) => sanitizeTriggerBetContent(cell))
    .join('\n')
}

/** 规范化单个投注 token（数字池 / PC28 和值） */
function sanitizeOneTriggerToken(raw: string): string {
  const p = String(raw ?? '').trim()
  if (!p) return ''
  const cfg = schemePlayConfig.value
  if (isPc28HezhiConfigLike(cfg) && isPc28PlayLine()) {
    const digits = p.replace(/\D/g, '')
    if (!digits) return ''
    const n = Number(digits)
    if (!Number.isFinite(n) || n < 0) return ''
    return String(Math.min(27, n))
  }
  const digits = p.replace(/\D/g, '')
  return normalizePoolToken(digits) || normalizePoolToken(p) || ''
}

/**
 * 正投/反投内容：支持逗号分隔多个号码（如 1,3,5）。
 * 中文逗号会归一为英文逗号；去重保序。
 */
function sanitizeTriggerBetContent(v: string): string {
  const raw = String(v ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!raw) return ''
  const parts = raw.split(',').map((s) => s.trim()).filter(Boolean)
  const out: string[] = []
  const seen = new Set<string>()
  for (const p of parts) {
    const tok = sanitizeOneTriggerToken(p)
    if (!tok || seen.has(tok)) continue
    seen.add(tok)
    out.push(tok)
  }
  return out.join(',')
}

/** 文字玩法：字符串 ↔ 多选数组 */
function triggerTextTokens(v: string): string[] {
  return String(v ?? '')
    .replace(/，/g, ',')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function setTriggerTextField(row: SchemeTriggerRow, field: 'pos' | 'neg', vals: string[]): void {
  const allow = new Set(triggerBetOptions.value)
  const out: string[] = []
  const seen = new Set<string>()
  for (const v of vals ?? []) {
    const t = String(v ?? '').trim()
    if (!t || !allow.has(t) || seen.has(t)) continue
    seen.add(t)
    out.push(t)
  }
  row[field] = out.join(',')
}

const triggerInputPlaceholder = computed(() => {
  if (isPc28HezhiConfigLike(schemePlayConfig.value) && isPc28PlayLine()) {
    return '如 1,2,15'
  }
  const bm = schemePlayConfig.value.betMode ?? ''
  if (bm === 'hezhi' && isPc28PlayLine()) {
    return '如 1,2,15'
  }
  const pool = numberPoolTokens.value
  if (!pool.length) return '如 1,3,5'
  if (pool.length <= 10) return `如 ${pool[0]},${pool[1] ?? pool[0]},${pool[2] ?? pool[0]}`
  return `如 ${pool[0]},${pool[1]},${pool[2]}`
})

// --- hot_cold_warm 冷热出号（v6 仅冷/热） ---
const hcwTotalPeriods = ref(20)
/** 容错=起点偏移：在「最热→最冷」排序上跳过该端最极端的前 N 名（0-9，0=不跳过）。 */
const hcwFaultCount = ref(0)
/** 名次个数：从起点偏移处连续取几个号（1-10，默认 1）。 */
const hcwPickCount = ref(1)
/** 出号类型：hot / cold（可多选；空=纯手动覆盖模式，仅用下方网格选号）。 */
const hcwPickTypes = ref<SchemeHotColdPickType[]>([])
const hcwStrategy = ref<SchemeRotateStrategy>('keep')
const HCW_STRATEGY_OPTIONS: Array<{ label: string; value: SchemeRotateStrategy }> = [
  { label: '每期换', value: 'every' },
  { label: '不换号', value: 'keep' },
  { label: '中后换', value: 'after_hit' },
  { label: '挂后换', value: 'after_miss' },
]
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
/** 每位/单档：选项 → 最近统计命中次数 */
const hcwFreq = ref<Array<Record<string, number>>>([])
interface HcwCell {
  token: string
  count: number | null
  tier: 'hot' | 'cold' | 'none'
}

/**
 * 号码整体频次模式（组选家族/不定位/包胆）：单档选号池（跨位合并频次），
 * 区别于按位型（每位一档）。
 */
const hcwDigitOverall = computed(() => {
  const cfg = schemePlayConfig.value as { betMode?: string; subPlayId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (['zu3', 'zu6', 'zu24', 'zu12', 'zu60', 'zu30', 'zu120', 'budingwei', 'baodan'].includes(bm)) return true
  const sub = `${String(cfg.subPlayId ?? '')}`.toLowerCase()
  if (/zuxuan_fs|zu3|zu6|zu24|zu12|zu60|zu30|zu120|budingwei|baodan/.test(sub)) return true
  const label = String(cfg.playMethodLabel ?? '')
  if (label.includes('单式')) return false
  return /组三|组六|组选|不定位|包胆/.test(label)
})

/**
 * 属性/聚合家族（大小单双/龙虎/庄闲/特殊号/和值/跨度）：单档「选项池」，
 * 分档频次由服务端复用权威判定计算（避免前端重复实现各彩种大小阈值/和值/跨度/龙虎口径）。
 */
const hcwAttribute = computed(() => {
  const cfg = schemePlayConfig.value
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (['daxiao', 'danshuang', 'dxds', 'zhuangxian', 'longhu', 'longhuhe', 'longhubao', 'teshu', 'hezhi', 'kuadu', 'weishu'].includes(bm)) {
    return true
  }
  const label = String(cfg.playMethodLabel ?? '')
  return /特殊号|大小单双|庄闲|龙虎豹|直选和值|组选和值|和值尾数|跨度/.test(label)
    || (label.includes('龙虎') && !label.includes('龙虎豹'))
    || (label === '和值' || (label.includes('和值') && !/单双|大小|尾数/.test(label)))
})

/** 单档布局（整体频次 或 属性选项池），区别于按位型每位一档 */
const hcwSingleGroup = computed(() => hcwDigitOverall.value || hcwAttribute.value)

/** 属性选项宇宙（服务端回填；无统计时用本地宇宙兜底，确保特殊号始终显示豹子/对子/顺子） */
const hcwAttrUniverse = ref<string[]>([])

/** 本地属性选项宇宙：特殊号→豹子/对子/顺子；和值等→数字池 */
function hcwLocalAttrUniverse(): string[] {
  const cfg = schemePlayConfig.value
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (bm === 'hezhi' || bm === 'kuadu' || bm === 'weishu') {
    return [...numberPoolTokens.value]
  }
  const textOpts = textPickOptionsForConfig(cfg)
  if (textOpts.length) return [...textOpts]
  // betMode 未就绪时按文案兜底（前三特殊号）
  const label = String(cfg.playMethodLabel ?? '')
  if (label.includes('特殊号')) {
    return cfg.playTemplate === 'pc28_std'
      ? ['豹子', '对子', '顺子', '极大', '极小']
      : ['豹子', '对子', '顺子']
  }
  return []
}

function ensureHcwPools(): void {
  const n = hcwSingleGroup.value ? 1 : positionCount.value
  while (hcwPools.value.length < n) hcwPools.value.push([])
}

/** 冷热分档分组：属性=单档「选项池」；号码整体频次=单档「号码池」；按位=每位一档 */
const hcwGroupLabels = computed(() =>
  hcwAttribute.value ? ['选项池'] : hcwDigitOverall.value ? ['号码池'] : positionLabels.value,
)

/** 无统计时的兜底可选项：属性优先本地宇宙（豹子/对子/顺子），再回退服务端回填 */
const hcwFallbackOptions = computed(() => {
  if (!hcwAttribute.value) return numberPoolTokens.value
  const local = hcwLocalAttrUniverse()
  if (local.length) return local
  return hcwAttrUniverse.value
})

function applyHotColdWarmFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const c = raw as Record<string, unknown>
  const tp = Math.trunc(Number(c.totalPeriods))
  if (Number.isFinite(tp) && tp >= 20 && tp <= 100) hcwTotalPeriods.value = tp
  else if (Number.isFinite(tp) && tp > 100) hcwTotalPeriods.value = 100
  // 容错=起点偏移（0-9），允许显式 0
  const fc = Math.trunc(Number(c.faultCount))
  if (Number.isFinite(fc)) hcwFaultCount.value = Math.min(9, Math.max(0, fc))
  const pc = Math.trunc(Number(c.pickCount))
  if (Number.isFinite(pc) && pc >= 1) hcwPickCount.value = Math.min(10, pc)
  if (Array.isArray(c.pickTypes)) {
    hcwPickTypes.value = c.pickTypes
      .map((t) => String(t ?? '').toLowerCase())
      .filter((t): t is SchemeHotColdPickType => t === 'hot' || t === 'cold')
  }
  const st = String(c.strategy ?? '')
  if (st === 'every' || st === 'keep' || st === 'after_hit' || st === 'after_miss') {
    hcwStrategy.value = st
  } else if (typeof c.winRotate === 'boolean') {
    hcwStrategy.value = c.winRotate ? 'after_hit' : 'keep'
  }
  if (Array.isArray(c.pool)) {
    // 回填时玩法树可能尚未就绪：保留任意非空 token（数字或属性文字如 大/小/龙/虎），
    // 展示选中态与去重经 tokenEq（数字按值、文字按串）比较
    hcwPools.value = c.pool.map((line) =>
      String(line ?? '')
        .split(/[,，\s]+/)
        .map((s) => s.trim())
        .filter((s) => s !== ''),
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

/** 冷热统计请求代数：玩法异步回填时 watch 会连触发，只采纳最后一次结果，避免中途 return 卡住「—」 */
let hcwLoadSeq = 0

/** 属性家族分档：调用服务端接口（复用权威 evaluatePlayHit 计频），单档选项池 */
async function loadHcwAttrStats(seq: number): Promise<void> {
  const cfg = schemePlayConfig.value
  const localUni = hcwLocalAttrUniverse()
  // 先铺本地宇宙，避免接口失败/延迟时选项池空白（前三特殊号须始终可见豹子/对子/顺子）
  if (localUni.length) hcwAttrUniverse.value = localUni
  // 特殊号/和值/跨度/尾数：playConfig.segmentLen=1 仅表示单档选项池，不是开奖截取长度。
  // 勿传 segmentLen，否则后端若覆盖 resolve 的 3 位，跨度恒为 0、次数全堆在「0」。
  const bm = String(cfg.betMode || (localUni.includes('豹子') ? 'teshu' : '')).toLowerCase()
  const res = await fetchHotColdWarmTiers({
    lotteryCode: lotteryCode.value,
    playTypeId: cfg.playTypeId,
    subPlayId: cfg.subPlayId,
    playTemplate: cfg.playTemplate || 'ssc_std',
    betMode: bm || cfg.betMode,
    catalogSubId: cfg.catalogSubId,
    playMethodLabel: cfg.playMethodLabel,
    numberPoolMin: cfg.numberPoolMin,
    numberPoolMax: cfg.numberPoolMax,
    periods: hcwTotalPeriods.value,
  })
  if (seq !== hcwLoadSeq) return
  if (res.mode !== 'attribute' || !Array.isArray(res.universe) || res.universe.length === 0) {
    // 接口未识别时仍展示本地选项，频次显示为 —
    hcwStatsReady.value = false
    hcwTiers.value = localUni.length
      ? [{ hot: [], warm: [], cold: [...localUni] }]
      : []
    hcwFreq.value = []
    return
  }
  const uni = res.universe
  const rawCounts = res.counts && typeof res.counts === 'object' ? res.counts : {}
  // 未命中选项后端可能省略 key；补 0，避免误以为「没下发次数」
  const counts: Record<string, number> = {}
  for (const opt of uni) counts[opt] = Number(rawCounts[opt]) || 0
  hcwAttrUniverse.value = uni
  hcwTiers.value = [
    { hot: res.hot ?? [], warm: res.warm ?? [], cold: res.cold ?? [] },
  ]
  hcwFreq.value = [counts]
  hcwStatsReady.value = true
}

async function loadHcwStats(): Promise<void> {
  const seq = ++hcwLoadSeq
  hcwLoading.value = true
  try {
    if (hcwAttribute.value) {
      await loadHcwAttrStats(seq)
      return
    }
    const res = await fetchGameDraws(lotteryCode.value, undefined, hcwTotalPeriods.value)
    if (seq !== hcwLoadSeq) return
    const items = Array.isArray(res?.items) ? res.items : []
    const segLen = positionCount.value
    const pool = numberPoolTokens.value
    // 号码整体频次：所有位合并统计为单档；按位型：每位一档
    const dims = hcwDigitOverall.value ? 1 : segLen
    const freq: Array<Record<string, number>> = Array.from({ length: dims }, () => ({}))
    let counted = 0
    for (const it of items) {
      const balls = Array.isArray(it?.balls) ? it.balls : []
      if (!balls.length) continue
      const offset = hcwPositionOffset(balls.length)
      for (let p = 0; p < segLen; p++) {
        const tk = normalizePoolToken(String(balls[offset + p] ?? ''))
        if (tk) {
          const d = hcwDigitOverall.value ? 0 : p
          freq[d]![tk] = (freq[d]![tk] ?? 0) + 1
          counted += 1
        }
      }
    }
    if (!counted) {
      hcwStatsReady.value = false
      hcwFreq.value = []
      return
    }
    // 频次降序排序后按池长二等分：热 / 冷（对齐 v6，无温档）
    const half = Math.ceil(pool.length / 2)
    hcwTiers.value = freq.map((counts) => {
      const sorted = [...pool].sort((a, b) => {
        const diff = (counts[b] ?? 0) - (counts[a] ?? 0)
        return diff !== 0 ? diff : Number(a) - Number(b)
      })
      return {
        hot: sorted.slice(0, half),
        warm: [],
        cold: sorted.slice(half),
      }
    })
    hcwFreq.value = freq.map((counts) => ({ ...counts }))
    hcwStatsReady.value = true
  } catch {
    if (seq !== hcwLoadSeq) return
    hcwStatsReady.value = false
    hcwFreq.value = []
  } finally {
    if (seq === hcwLoadSeq) hcwLoading.value = false
  }
}

/** token 相等：数字按数值比较（兼容 '07'/'7'），属性文字按字符串比较 */
function tokenEq(a: string, b: string): boolean {
  const na = Number(a)
  const nb = Number(b)
  if (a.trim() !== '' && b.trim() !== '' && Number.isFinite(na) && Number.isFinite(nb)) {
    return na === nb
  }
  return a === b
}

/** 池内是否已含该号（数字按值、属性文字按串） */
function poolHasToken(arr: string[] | undefined, token: string): boolean {
  if (!arr) return false
  return arr.some((t) => tokenEq(t, token))
}

function hcwPosPickCap(): number | null {
  if (hcwAttribute.value) return null
  if (isYixingDingweiPlayConfig(schemePlayConfig.value)) return YIXING_MAX_PICKS_PER_POS
  return null
}

function toggleHcwDigit(pos: number, digit: string): void {
  ensureHcwPools()
  const arr = hcwPools.value[pos]
  if (!arr) return
  const i = arr.findIndex((t) => tokenEq(t, digit))
  if (i >= 0) {
    arr.splice(i, 1)
    return
  }
  const cap = hcwPosPickCap()
  if (cap != null && arr.length >= cap) {
    ElMessage.warning(YIXING_MAX_PICKS_MSG)
    return
  }
  arr.push(digit)
  // 数字池升序；属性文字保持选择顺序（Number 比较对文字为 NaN，稳定不变序）
  arr.sort((a, b) => {
    const na = Number(a)
    const nb = Number(b)
    if (Number.isFinite(na) && Number.isFinite(nb)) return na - nb
    return 0
  })
}

function sortHcwTokens(tokens: string[]): string[] {
  return [...tokens].sort((a, b) => {
    const na = Number(a)
    const nb = Number(b)
    if (Number.isFinite(na) && Number.isFinite(nb)) return na - nb
    // 属性选项：按本地宇宙顺序（豹子→对子→顺子），勿因比较返回 0 打乱
    const uni = hcwLocalAttrUniverse()
    const ia = uni.indexOf(a)
    const ib = uni.indexOf(b)
    if (ia >= 0 || ib >= 0) return (ia < 0 ? 999 : ia) - (ib < 0 ? 999 : ib)
    return a.localeCompare(b, 'zh')
  })
}

/** 快捷选号目标：冷/热取分档；全取当前可见全集 */
function hcwQuickTargets(pos: number, kind: 'cold' | 'hot' | 'all'): string[] {
  if (kind === 'all') {
    if (hcwStatsReady.value) {
      const t = hcwTiers.value[pos]
      return [...(t?.hot ?? []), ...(t?.cold ?? [])]
    }
    return [...hcwFallbackOptions.value]
  }
  if (!hcwStatsReady.value) return []
  const t = hcwTiers.value[pos]
  return kind === 'hot' ? [...(t?.hot ?? [])] : [...(t?.cold ?? [])]
}

function applyHcwQuick(pos: number, kind: 'cold' | 'hot' | 'all' | 'clear'): void {
  ensureHcwPools()
  if (kind === 'clear') {
    hcwPools.value[pos] = []
    return
  }
  let targets = hcwQuickTargets(pos, kind)
  if (!targets.length) return
  const cap = hcwPosPickCap()
  if (cap != null && targets.length > cap) {
    targets = sortHcwTokens(targets).slice(0, cap)
    ElMessage.warning(YIXING_MAX_PICKS_MSG)
  }
  hcwPools.value[pos] = sortHcwTokens(targets)
}

/** 快捷钮高亮：当前池与该快捷目标完全一致（「清」仅动作、不高亮） */
function hcwQuickActive(pos: number, kind: 'cold' | 'hot' | 'all'): boolean {
  const pool = hcwPools.value[pos] ?? []
  const targets = hcwQuickTargets(pos, kind)
  if (!targets.length) return false
  if (pool.length !== targets.length) return false
  return targets.every((t) => poolHasToken(pool, t))
}

function hcwLookupCount(pos: number, token: string): number {
  const counts = hcwFreq.value[pos] ?? {}
  for (const [k, v] of Object.entries(counts)) {
    if (tokenEq(k, token)) return Number(v) || 0
  }
  return 0
}

function hcwTokenTier(pos: number, token: string): 'hot' | 'cold' | 'none' {
  if (!hcwStatsReady.value) return 'none'
  const t = hcwTiers.value[pos]
  if ((t?.hot ?? []).some((x) => tokenEq(x, token))) return 'hot'
  if ((t?.cold ?? []).some((x) => tokenEq(x, token))) return 'cold'
  return 'none'
}

/** 每位展示：0–9（或选项宇宙）升序；下方带频次；热/冷着色 */
function hcwDisplayCells(pos: number): HcwCell[] {
  let tokens: string[]
  if (hcwStatsReady.value) {
    const t = hcwTiers.value[pos]
    const seen: string[] = []
    for (const d of [...(t?.hot ?? []), ...(t?.cold ?? [])]) {
      if (!seen.some((x) => tokenEq(x, d))) seen.push(d)
    }
    tokens = seen.length ? seen : [...hcwFallbackOptions.value]
  } else {
    tokens = [...hcwFallbackOptions.value]
  }
  return sortHcwTokens(tokens).map((token) => ({
    token,
    count: hcwStatsReady.value ? hcwLookupCount(pos, token) : null,
    tier: hcwTokenTier(pos, token),
  }))
}

/** 按分组缓存单元格，避免模板重复计算 */
const hcwCellsByPos = computed(() =>
  hcwGroupLabels.value.map((_, pi) => hcwDisplayCells(pi)),
)

/** 预估注数：与随机出号一致，走 countBetUnits（位积/组合×段长/任选 C(n,k)） */
const hcwEstimatedUnits = computed(() => {
  // 属性/聚合家族（和值/跨度/大小单双/龙虎等）：统一走 countBetUnits
  // 和值/跨度按组合数×段倍乘（前中后三组选和值 2,6,13,17,24 → 38×3=114），勿按「选几个算几注」
  if (hcwAttribute.value) {
    const line = (hcwPools.value[0] ?? []).filter((t) => t.trim() !== '').join(',')
    return line ? countBetUnits(schemePlayConfig.value, line) : 0
  }
  // 号码整体频次：单档选号池，按组选/不定位/包胆口径算注数
  if (hcwDigitOverall.value) {
    const line = (hcwPools.value[0] ?? []).join(',')
    return line.trim() ? countBetUnits(schemePlayConfig.value, line) : 0
  }
  const n = positionCount.value
  if (n <= 0) return 0
  const lines = Array.from({ length: n }, (_, i) => (hcwPools.value[i] ?? []).join(','))
  if (lines.every((x) => !x.trim())) return 0
  return countBetUnits(schemePlayConfig.value, lines.join('\n'))
})

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
  if (rdCounts.value.length === 0) rdCounts.value.push(1)
}

/** 单式/组选单式：整注随机（仅需注数 rdCounts[0]），非按位产号 */
const rdWholeTicket = computed(() => {
  const cfg = schemePlayConfig.value as { betMode?: string; subPlayId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  const sub = String(cfg.subPlayId ?? '').toLowerCase()
  if (['danshi', 'zhixuan_ds', 'zuxuan_ds', 'hunhe'].includes(bm)) return true
  if (['zhixuan_ds', 'zuxuan_ds'].includes(sub)) return true
  const label = String(cfg.playMethodLabel ?? '')
  return label.includes('单式') || label.includes('混合')
})

/** 单式整注随机的本地预览注单 */
const rdWholePreview = ref<string[]>([])

/** 组三/组六/组选N/组选复式：号码池随机（选 K 个号），非按位、非整注。包胆属属性单选，勿因文案含「组选」误入。 */
const rdZuxuanPool = computed(() => {
  if (rdWholeTicket.value) return false
  const cfg = schemePlayConfig.value as { betMode?: string; subPlayId?: string; catalogSubId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  const label = String(cfg.playMethodLabel ?? '')
  if (bm === 'baodan' || /包胆/.test(label)) return false
  if (['zu3', 'zu6', 'zu24', 'zu12', 'zu60', 'zu30', 'zu120'].includes(bm)) return true
  const cat = `${String(cfg.subPlayId ?? '')} ${String(cfg.catalogSubId ?? '')}`.toLowerCase()
  if (/baodan|_bd\b|包胆/.test(`${cat} ${label}`)) return false
  if (/zuxuan_fs|zuxuan|zu3|zu6|zu24|zu12|zu60|zu30|zu120/.test(cat)) return true
  return /组三|组六|组选/.test(label)
})

/** 属性/聚合家族（大小单双/龙虎/特殊号/庄闲/和值/跨度/不定位/包胆）：从选项宇宙随机抽 K 个 */
const rdAttribute = computed(() => {
  if (rdWholeTicket.value || rdZuxuanPool.value) return false
  const bm = String(schemePlayConfig.value.betMode ?? '').toLowerCase()
  return ['daxiao', 'danshuang', 'dxds', 'zhuangxian', 'longhu', 'longhuhe', 'longhubao', 'teshu', 'hezhi', 'kuadu', 'budingwei', 'baodan'].includes(bm)
})

/** 随机出号是否用"单一数量输入"（单式注数 / 组选选码个数 / 属性选项个数） */
const rdSingleCountMode = computed(() => rdWholeTicket.value || rdZuxuanPool.value || rdAttribute.value)
const rdSingleCountLabel = computed(() => {
  if (rdWholeTicket.value) return '注数'
  if (rdZuxuanPool.value) return '选码个数'
  return '选项个数'
})

/** 属性/聚合玩法选项宇宙（特殊号=豹子/对子/顺子，大小单双=大/小/单/双，和值=号池等） */
function rdAttributeUniverse(): string[] {
  const cfg = schemePlayConfig.value
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (bm === 'baodan') return [...numberPoolTokens.value]
  if (bm === 'weishu' || bm === 'hezhi' || bm === 'kuadu' || bm === 'budingwei') {
    return [...numberPoolTokens.value]
  }
  const textOpts = textPickOptionsForConfig(cfg)
  if (textOpts.length) return [...textOpts]
  return [...numberPoolTokens.value]
}

/** 属性/聚合玩法选项宇宙大小（特殊号=3、大小单双=4、和值=号池长度等） */
function rdAttributeUniverseMax(): number {
  const bm = String(schemePlayConfig.value.betMode ?? '').toLowerCase()
  if (bm === 'baodan') return 1
  if (bm === 'weishu') return Math.min(WEISHU_MAX_BET_UNITS, numberPoolTokens.value.length || 10)
  return Math.max(1, rdAttributeUniverse().length)
}

const rdSingleCountMax = computed(() => {
  if (rdWholeTicket.value) return 200
  if (rdZuxuanPool.value) return Math.max(3, numberPoolTokens.value.length)
  if (rdAttribute.value) return rdAttributeUniverseMax()
  return 10
})
const rdSingleCountMin = computed(() => {
  if (rdWholeTicket.value) return 1
  if (rdZuxuanPool.value) return Math.max(2, positionCount.value)
  return 1
})

/** 玩法切换后把选项个数钳到当前宇宙上限（避免特殊号仍显示 >3） */
watch(
  [rdSingleCountMax, rdSingleCountMin, rdSingleCountMode],
  ([max, min, single]) => {
    if (!single) return
    const cur = Math.trunc(Number(rdCounts.value[0]) || 0)
    const next = Math.min(max, Math.max(min, cur || min))
    if (cur !== next) rdCounts.value = [next, ...rdCounts.value.slice(1)]
  },
)

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
    if (runTypeId.value === 'adv_trigger_bet') {
      ensureTriggerRows()
      ensureTriggerPositions()
    }
    if (runTypeId.value === 'hot_cold_warm') {
      ensureHcwPools()
      // 玩法家族切换后旧分档失效（数字↔属性宇宙不同），重置后按当前总期数直接统计
      hcwStatsReady.value = false
      hcwAttrUniverse.value = []
      void loadHcwStats()
    }
    if (runTypeId.value === 'random_draw') {
      ensureRdCounts()
      // 切换玩法后清空预览，避免前三残留整注 tag、或一星残留错位内容
      rdPreview.value = []
      rdWholePreview.value = []
    }
  },
  { immediate: true },
)

/** 子玩法切换：复式按位内容 ↔ 单式整注互不适配时转换/清空，避免 `5,5,5` 复式存成 `5\\n5\\n5` 后切入单式仍保留 */
let playContentAdaptReady = false
watch(
  () =>
    [
      schemePlayConfig.value.playTypeId,
      schemePlayConfig.value.catalogSubId,
      schemePlayConfig.value.subPlayId,
      schemePlayConfig.value.betMode,
      schemePlayConfig.value.inputMode,
      schemePlayConfig.value.segmentLen,
    ].join('|'),
  () => {
    if (!playContentAdaptReady) {
      playContentAdaptReady = true
      return
    }
    const cfg = schemePlayConfig.value
    let changed = false
    const nextGroups = schemeGroups.value.map((g) => adaptSchemeGroupContentForPlay(g, cfg))
    if (nextGroups.some((g, i) => g !== schemeGroups.value[i])) {
      schemeGroups.value = nextGroups
      changed = true
    }
    if (jushuList.value.length) {
      const nextJushu = jushuList.value.map((row) => ({
        ...row,
        content: adaptSchemeGroupContentForPlay(row.content, cfg),
      }))
      if (nextJushu.some((r, i) => r.content !== jushuList.value[i]?.content)) {
        jushuList.value = nextJushu
        changed = true
      }
    }
    if (changed) persistDraft()
  },
)

function applyRandomDrawFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const c = raw as Record<string, unknown>
  if (Array.isArray(c.counts) && c.counts.length) {
    const max = rdSingleCountMax.value
    const min = rdSingleCountMin.value
    rdCounts.value = c.counts.map((n) =>
      Math.min(max, Math.max(min, Math.trunc(Number(n)) || min)),
    )
  }
  const s = String(c.strategy ?? '')
  if (s === 'every' || s === 'keep' || s === 'after_hit' || s === 'after_miss') rdStrategy.value = s
}

function shuffleInPlace<T>(arr: T[]): T[] {
  for (let j = arr.length - 1; j > 0; j--) {
    const k = Math.floor(Math.random() * (j + 1))
    ;[arr[j], arr[k]] = [arr[k]!, arr[j]!]
  }
  return arr
}

/** 一星每位最多 9 个号；其它按位玩法最多 10 */
const rdPerPosMax = computed(() =>
  isYixingDingweiPlayConfig(schemePlayConfig.value) ? YIXING_MAX_PICKS_PER_POS : 10,
)

/** 本地生成预览号码（含属性家族选项抽样） */
function generateRdPreview(): void {
  ensureRdCounts()
  if (rdWholeTicket.value) {
    const pool = [...numberPoolTokens.value]
    const positions = positionCount.value
    const n = Math.min(200, Math.max(1, rdCounts.value[0] ?? 1))
    const seen = new Set<string>()
    const out: string[] = []
    for (let a = 0; out.length < n && a < n * 100 + 100; a++) {
      const digits = Array.from({ length: positions }, () => pool[Math.floor(Math.random() * pool.length)] ?? '0')
      const key = digits.join('')
      if (seen.has(key)) continue
      seen.add(key)
      out.push(key)
    }
    rdWholePreview.value = out
    rdPreview.value = []
    return
  }
  if (rdZuxuanPool.value) {
    const pool = shuffleInPlace([...numberPoolTokens.value])
    const k = Math.min(pool.length, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value))
    rdWholePreview.value = [pool.slice(0, k).sort((a, b) => Number(a) - Number(b)).join(',')]
    rdPreview.value = []
    return
  }
  if (rdAttribute.value) {
    // 属性家族：从选项宇宙抽 K 个本地预览（特殊号→豹子/对子/顺子）
    const universe = shuffleInPlace(rdAttributeUniverse())
    const k = Math.min(
      universe.length,
      rdSingleCountMax.value,
      Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value),
    )
    const picks = universe.slice(0, k)
    const textOpts = textPickOptionsForConfig(schemePlayConfig.value)
    if (textOpts.length) {
      const order = new Map(textOpts.map((t, i) => [t, i]))
      picks.sort((a, b) => (order.get(a) ?? 0) - (order.get(b) ?? 0))
    } else {
      picks.sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
    }
    rdWholePreview.value = picks
    rdPreview.value = []
    return
  }
  // 按位玩法（一星/前三/前二/五星复式等）：按位预览，每个号一枚蓝色 tag，不展开整注
  const perPosMax = rdPerPosMax.value
  const pools = Array.from({ length: positionCount.value }, (_, i) => {
    const pool = shuffleInPlace([...numberPoolTokens.value])
    const count = Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1), pool.length)
    return pool.slice(0, count).sort((a, b) => Number(a) - Number(b))
  })
  rdPreview.value = pools
  rdWholePreview.value = []
}

interface RdPreviewTag {
  key: string
  /** 展示文案 */
  label: string
  kind: 'whole' | 'pos'
  /** 按位：位下标；整注/号池：条目下标 */
  index: number
  /** 按位：该 tag 对应的单个号码（关闭时只删这一号） */
  digit?: string
}

/**
 * 预览 tag：
 * - 按位玩法（一星/前三/前二/五星复式等）：每位每个号一枚蓝色 tag（三位各 1 号 → 3 枚）
 * - 单式整注 / 组选号池：一注或号池条目一枚
 */
const rdPreviewTags = computed<RdPreviewTag[]>(() => {
  // 按位模式：只用 rdPreview，禁止整注笛卡尔残留盖住按位 tag
  if (!rdSingleCountMode.value) {
    const rows = rdPreview.value
    if (!rows.length) return []
    const out: RdPreviewTag[] = []
    rows.forEach((row, index) => {
      if (!row?.length) return
      row.forEach((digit, di) => {
        out.push({
          key: `p-${index}-${di}-${digit}`,
          label: digit,
          kind: 'pos' as const,
          index,
          digit,
        })
      })
    })
    return out
  }
  if (!rdWholePreview.value.length) return []
  return rdWholePreview.value.map((ticket, index) => ({
    key: `w-${index}-${ticket}`,
    label: ticket.includes(',')
      ? ticket.split(/[,，]/).filter(Boolean).join('\u2009')
      : ticket,
    kind: 'whole' as const,
    index,
  }))
})

function removeRdPreviewTag(tag: RdPreviewTag): void {
  if (tag.kind === 'whole') {
    rdWholePreview.value = rdWholePreview.value.filter((_, i) => i !== tag.index)
    return
  }
  const next = [...rdPreview.value]
  const row = [...(next[tag.index] ?? [])]
  if (tag.digit != null) {
    const at = row.indexOf(tag.digit)
    if (at >= 0) row.splice(at, 1)
  } else {
    row.length = 0
  }
  next[tag.index] = row
  rdPreview.value = next
}

/** 预估注数：按预览（或每位数量占位）走同一套 countBetUnits，含直选组合×段长 */
const rdEstimatedUnits = computed(() => {
  // 单式整注随机：注数即 rdCounts[0]
  if (rdWholeTicket.value) return Math.min(200, Math.max(1, rdCounts.value[0] ?? 1))
  // 组选号码池随机：按选中号码池走 countBetUnits（组选口径）
  if (rdZuxuanPool.value) {
    const pool = [...numberPoolTokens.value]
    const k = Math.min(pool.length, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value))
    return countBetUnits(schemePlayConfig.value, pool.slice(0, k).join(','))
  }
  // 属性家族：注数=选项个数，且不超过宇宙上限（特殊号最多 3）
  if (rdAttribute.value) {
    return Math.min(rdSingleCountMax.value, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? 1))
  }
  const n = positionCount.value
  if (n <= 0) return 0
  const perPosMax = rdPerPosMax.value
  const lines = Array.from({ length: n }, (_, i) => {
    const prev = rdPreview.value[i] ?? []
    if (prev.length) return prev.join(',')
    const count = Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1))
    return Array.from({ length: count }, (_, j) => String(j % 10)).join(',')
  })
  return countBetUnits(schemePlayConfig.value, lines.join('\n'))
})

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

/** 草稿占位名不回填到输入框，避免设置方案模式后名称被自动写成「未命名方案」 */
function schemeNameFromDraftMeta(raw: string): string {
  const name = String(raw ?? '').trim()
  return name === '未命名方案' ? '' : name
}

function applyDraftSnapshot(draft: SchemeDraftSnapshot): void {
  schemeName.value = schemeNameFromDraftMeta(draft.meta.schemeName)
  simBet.value = draft.simBet
  schemeFunds.value = draft.schemeFunds
  schemeCurrency.value = normalizeSchemeCurrency(draft.schemeCurrency)
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

function buildLiveDraftMeta(): SchemeDraftMeta {
  return {
    kind: schemeKind.value,
    // 草稿保留空名称；提交上云前仍校验必填，不在此处写占位名
    schemeName: schemeName.value.trim(),
    lotteryCode: isBuiltinPlan.value ? '' : lotteryCode.value.trim(),
    runTypeId: runTypeId.value,
    playTypeId: isBuiltinPlan.value ? '' : playTypeId.value.trim(),
    subPlayId: isBuiltinPlan.value ? '' : subPlayId.value.trim(),
  }
}

function buildDraftSnapshot(): SchemeDraftSnapshot {
  const existing = loadSchemeDraft()
  const rtFields = runTypeDraftFields()
  return {
    meta: buildLiveDraftMeta(),
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    schemeCurrency: schemeCurrency.value,
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
    const fresh = route.query.fresh === '1'
    if (fresh) {
      clearSchemeDraft()
      const nextQuery = { ...route.query } as Record<string, string | string[] | undefined>
      delete nextQuery.fresh
      void router.replace({ query: nextQuery })
    }
    const draft = fresh ? null : loadSchemeDraft()
    if (draft) {
      applyDraftSnapshot(draft)
    } else {
      const meta = draftMetaFromQuery(route.query as Record<string, unknown>)
      schemeName.value = schemeNameFromDraftMeta(meta.schemeName)
      runTypeId.value = normalizeRunTypeId(meta.runTypeId || 'fixed_rotate')
      if (meta.lotteryCode) lotteryCode.value = meta.lotteryCode
      if (meta.playTypeId) playTypeId.value = meta.playTypeId
      if (meta.subPlayId) subPlayId.value = meta.subPlayId
    }
    remoteHasInstance.value = false
    shareLocked.value = false
    await loadLotteries()
    if (!lotteryCode.value && lotteries.value.length && !isBuiltinPlan.value) {
      lotteryCode.value = lotteries.value[0].code
    }
    if (lotteryCode.value) {
      await loadRunTypeOptions(lotteryCode.value)
      await loadIdentityPlayTree(lotteryCode.value)
    }
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
    schemeCurrency.value = normalizeSchemeCurrency(cfg.schemeCurrency)
    if (cfg.multCoeff != null && String(cfg.multCoeff).trim() !== '') {
      multCoeff.value = String(cfg.multCoeff).trim()
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
    if (lotteryCode.value) {
      void loadRunTypeOptions(lotteryCode.value)
      void loadIdentityPlayTree(lotteryCode.value)
    }
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
      const textPlay = isTriggerTextPlay.value
      const perPos = showTriggerPerPosColumns.value
      const triggerBet: SchemeTriggerBet = {
        rows: triggerRows.value.map((r) => ({
          ...r,
          pos: textPlay
            ? triggerTextTokens(r.pos)
                .filter((t) => triggerBetOptions.value.includes(t))
                .join(',')
            : perPos
              ? sanitizeTriggerPerPosField(r.pos)
              : sanitizeTriggerBetContent(r.pos),
          neg: textPlay
            ? triggerTextTokens(r.neg)
                .filter((t) => triggerBetOptions.value.includes(t))
                .join(',')
            : perPos
              ? sanitizeTriggerPerPosField(r.neg)
              : sanitizeTriggerBetContent(r.neg),
        })),
        mode: triggerMode.value,
      }
      if (showTriggerPositionPicker.value) {
        triggerBet.positionIdxs = normalizeTriggerPositionIdxs(
          triggerPositionIdxs.value,
          positionCount.value || 10,
        )
      }
      return { triggerBet }
    }
    case 'hot_cold_warm': {
      const hotColdWarm: SchemeHotColdWarm = {
        totalPeriods: Math.min(100, Math.max(20, Math.trunc(Number(hcwTotalPeriods.value) || 20))),
        // 属性选项池 / 号码整体频次：单档（单元素）；按位：每位一行
        pool: hcwSingleGroup.value
          ? [(hcwPools.value[0] ?? []).join(',')]
          : Array.from({ length: positionCount.value }, (_, i) => (hcwPools.value[i] ?? []).join(',')),
        strategy: hcwStrategy.value,
        // 出号类型：hot/cold（可多选；空则退化为纯手动覆盖）
        pickTypes: [...hcwPickTypes.value],
        // 容错=起点偏移（0-9）
        faultCount: Math.min(9, Math.max(0, Math.trunc(Number(hcwFaultCount.value) || 0))),
        // 名次个数（1-10）
        pickCount: Math.min(10, Math.max(1, Math.trunc(Number(hcwPickCount.value) || 1))),
        // 兼容旧字段：中后换 ≈ 原中奖轮换开
        winRotate: hcwStrategy.value === 'after_hit',
      }
      return { hotColdWarm }
    }
    case 'random_draw': {
      // 单式=注数 / 组选=选码个数 → counts=[K]；按位型 → 每位号码数量
      const counts = rdSingleCountMode.value
        ? [Math.min(rdSingleCountMax.value, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value))]
        : Array.from({ length: positionCount.value }, (_, i) => Math.min(10, Math.max(1, rdCounts.value[i] ?? 1)))
      const randomDraw: SchemeRandomDraw = { counts, strategy: rdStrategy.value }
      return { randomDraw }
    }
    case 'fixed_number':
      // 固定取码：内容在 schemeGroups[0]，每期原样复投
      return {}
    default:
      return {}
  }
}

function buildRemoteDraftPatch(): UpdateSchemeInput {
  return {
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    schemeCurrency: schemeCurrency.value,
    multCoeff: multCoeff.value.trim() || '1',
    startTime: startTime.value,
    endTime: endTime.value,
    // 内置计画配置只读（服务端物化），不回写 schemeGroups；固定取码仅保存单元素数组
    ...(runTypeId.value === 'builtin_plan'
      ? {}
      : {
        schemeGroups:
          runTypeId.value === 'fixed_number' ? [schemeGroups.value[0] ?? ''] : [...schemeGroups.value],
      }),
    betUnit: betUnit.value,
    ...catalogFieldsFromPlayConfig(schemePlayConfig.value),
    stopLoss: stopLoss.value,
    takeProfit: takeProfit.value,
    ...runTypeDraftFields(),
  }
}

function navigateAfterCloudSave(): void {
  const returnName = String(route.query.returnName ?? '')
  if (returnName === 'scheme-detail') {
    const q: Record<string, string> = {}
    for (const key of ['turnover', 'sessionPnl', 'multiplier', 'status'] as const) {
      const raw = route.query[key]
      if (raw != null && String(raw) !== '') q[key] = String(Array.isArray(raw) ? raw[0] : raw)
    }
    void router.replace({
      name: 'scheme-detail',
      params: { definitionId: schemeId.value },
      query: q,
    })
    return
  }
  void router.push({ name: 'cloud' })
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
    schemeCurrency,
    startTime,
    endTime,
    schemeGroups,
    shareStatus,
    betUnit,
    stopLoss,
    takeProfit,
    multCoeff,
    lotteryCode,
    runTypeId,
    playTypeId,
    subPlayId,
  ],
  persistDraft,
  { deep: true },
)

/** 七套面板状态跟随现有防抖持久化机制 */
watch(
  [
    jushuList,
    triggerRows,
    triggerMode,
    hcwTotalPeriods,
    hcwPools,
    hcwStrategy,
    hcwFaultCount,
    hcwPickCount,
    hcwPickTypes,
    rdCounts,
    rdStrategy,
  ],
  persistDraft,
  { deep: true },
)

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push({ name: 'cloud' })
}

/** 倍投设定方式（0–3），须从倍投设定页确认后才有值 */
const betMultiplierKind = ref<'' | '0' | '1' | '2' | '3'>('')

/** 倍投设定页校验失败：query.bmsError；确认成功：query.bmsKind（0–3） */
const betMultiplierError = ref('')

const betMultiplierSelectedLabel = computed(() =>
  betMultiplierKind.value ? (BET_MULTIPLIER_KIND_LABELS[betMultiplierKind.value] ?? '') : '',
)

const betMultiplierFieldText = computed(() => {
  if (betMultiplierError.value) return betMultiplierError.value
  if (betMultiplierSelectedLabel.value) return betMultiplierSelectedLabel.value
  return '未设置，请选择'
})

const betMultiplierFieldTone = computed(() => {
  if (betMultiplierError.value) return 'danger'
  if (betMultiplierSelectedLabel.value) return 'normal'
  return 'muted'
})

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
  if (runTypeId.value) q.runType = runTypeId.value
  if (playTypeId.value) q.playType = playTypeId.value
  if (subPlayId.value) q.subPlay = subPlayId.value
  if (lotteryCode.value) q.lottery = lotteryCode.value
  return q
}

function goBetMultiplierSettings() {
  betMultiplierError.value = ''
  const snapshot = buildDraftSnapshot()
  saveSchemeEditRestoreSnapshot(schemeId.value, snapshot)
  flushPersistDraft()
  sessionStorage.setItem(scrollRestoreStorageKey(), String(readDocumentScrollY()))
  const cfg = schemePlayConfig.value
  // 回显：旧 kind 0/1 在无自动算表玩法下落到简单；新保存只写 2/3
  const activeTab =
    betMultiplierKind.value === '3'
      ? '3'
      : betMultiplierKind.value === '0' || betMultiplierKind.value === '1'
        ? betMultiplierKind.value
        : '2'
  router.push({
    name: 'bet-multiplier-settings',
    query: {
      fromScheme: '1',
      schemeId: String(route.params.schemeId ?? ''),
      activeTab,
      ...(schemeName.value.trim()
        ? { title: encodeURIComponent(schemeName.value.trim()) }
        : {}),
      ...(lotteryCode.value ? { lottery: lotteryCode.value } : {}),
      playType: playTypeId.value || cfg.playTypeId || '',
      subPlay: subPlayId.value || cfg.subPlayId || '',
      betMode: cfg.betMode || '',
      playTypeLabel: cfg.playTypeLabel || '',
      subPlayLabel: cfg.playMethodLabel || '',
      playTemplate: cfg.playTemplate || '',
      ...(cfg.segmentLen ? { segmentLen: String(cfg.segmentLen) } : {}),
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
  const lottery = lotteryCode.value.trim()
  const fundsRaw = schemeFunds.value.trim()
  const funds = Number(fundsRaw)
  // 定位胆多位内容含前导空行，禁止 trim（否则 ,,12,, 会压成万位）
  const groups = schemeGroups.value.map((g) => String(g ?? '').replace(/\r/g, ''))
  const groupHasContent = (g: string) => Boolean(g.trim())

  if (!name) {
    await warn('方案名称不能为空')
    return
  }
  if (!isBuiltinPlan.value && (!lottery || !playTypeId.value || !subPlayId.value)) {
    await warn('请选择彩种与玩法')
    return
  }
  if (!isBuiltinPlan.value) {
    const matrixErr = validateRunTypePlaySelection(
      runTypeId.value,
      playTypeId.value,
      subPlayId.value,
      playTreeTypes.value,
    )
    if (matrixErr) {
      await warn(matrixErr)
      return
    }
  }
  if (!fundsRaw || !Number.isFinite(funds)) {
    await warn('方案资金不能为空')
    return
  }
  if (funds <= 0) {
    await warn('方案资金必须大于 0')
    return
  }
  if (!SCHEME_CURRENCY_OPTIONS.includes(schemeCurrency.value)) {
    await warn('请选择方案币种')
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
  // 止损/止盈非必填：空或 0 表示无上限（运行时 schemelimits 仅在 >0 时生效）
  const stopLossRaw = stopLoss.value.trim()
  if (stopLossRaw) {
    const stopLossNum = Number(stopLossRaw)
    if (!Number.isFinite(stopLossNum) || stopLossNum < 0) {
      await warn('止损金额不能小于 0')
      return
    }
  }
  const takeProfitRaw = takeProfit.value.trim()
  if (takeProfitRaw) {
    const takeProfitNum = Number(takeProfitRaw)
    if (!Number.isFinite(takeProfitNum) || takeProfitNum < 0) {
      await warn('止盈金额不能小于 0')
      return
    }
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
    if (showTriggerPerPosColumns.value) {
      const n = Math.max(1, positionCount.value)
      const incomplete = triggerRows.value.find((r) => {
        if (!r.enabled) return false
        const posParts = triggerFieldParts(r.pos, n)
        const negParts = triggerFieldParts(r.neg, n)
        return posParts.some((c) => !c) || negParts.some((c) => !c)
      })
      if (incomplete) {
        const posNames = positionLabels.value.map((l) => triggerPosName(l)).join('、')
        await warn(`启用行须填齐各位正投与反投（${posNames}）`)
        return
      }
      const anyEnabled = triggerRows.value.some((r) => r.enabled)
      if (!anyEnabled) {
        await warn('请至少启用一行开某投某映射')
        return
      }
    } else {
      const filled = triggerRows.value.some(
        (r) => r.enabled && (String(r.pos).trim() !== '' || String(r.neg).trim() !== ''),
      )
      if (!filled) {
        await warn('请填写开某投某映射（可用「全部随机」）')
        return
      }
    }
    if (showTriggerPositionPicker.value && !triggerPositionIdxs.value.length) {
      await warn('请至少选择一个投注位')
      return
    }
    const sample = triggerRows.value.find((r) => r.enabled && String(r.pos).trim())
    schemeGroups.value = [sample ? String(sample.pos).trim() : '0']
  } else if (rt === 'hot_cold_warm') {
    ensureHcwPools()
    // 按位玩法存成单组多行（万\n千\n百），避免被当成 3 个轮换组导致只取到一位
    schemeGroups.value = hcwSingleGroup.value
      ? [(hcwPools.value[0] ?? []).join(',')]
      : [
          Array.from({ length: positionCount.value }, (_, i) => (hcwPools.value[i] ?? []).join(',')).join('\n'),
        ]
  } else if (rt === 'random_draw') {
    ensureRdCounts()
    if (rdSingleCountMode.value) {
      if (!rdWholePreview.value.length) generateRdPreview()
      schemeGroups.value = [rdWholePreview.value.join(',') || '0']
    } else {
      if (!rdPreview.value.length || rdPreview.value.every((row) => !row.length)) {
        generateRdPreview()
      }
      // 按位：单组多行（万\n千\n…），禁止拆成多个轮换组
      const perPosMax = rdPerPosMax.value
      schemeGroups.value = [
        Array.from({ length: positionCount.value }, (_, i) => {
          const prev = rdPreview.value[i] ?? []
          if (prev.length) return prev.join(',')
          const count = Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1))
          return Array.from({ length: count }, (_, j) => String(j % 10)).join(',')
        }).join('\n'),
      ]
    }
  } else {
    if (groups.every((g) => !groupHasContent(g))) {
      await warn('方案内容不能为空')
      return
    }
    if (groups.some((g) => !groupHasContent(g))) {
      await warn('存在空的方案分组，请填写内容或删除该组')
      return
    }

    const baoziErrEarly = schemeSoloBaoziError(schemePlayConfig.value, groups)
    if (baoziErrEarly) {
      await warn(baoziErrEarly)
      return
    }

    const groupCheck = validateSchemeGroups(schemePlayConfig.value, groups)
    if (!groupCheck.ok) {
      const maxBetsMsg = isMaxBetUnitsExceededMessage(groupCheck.message)
      if (!maxBetsMsg) {
        for (const idx of groupCheck.invalidIndexes) {
          schemeGroups.value[idx] = ''
        }
      }
      await confirmDialog({
        title: maxBetsMsg ? '提示' : '输入不合法',
        message: maxBetsMsg
          ? groupCheck.message
          : `${groupCheck.message}。请按「${playModeSummary.value}」规则重新填写。`,
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return
    }
    schemeGroups.value = groupCheck.normalized
  }

  // 高级定码轮换 / 冷热 / 随机等未走 validateSchemeGroups 的入口：统一拦和值超 900 注
  {
    const contents =
      rt === 'adv_fixed_rotate'
        ? jushuList.value.map((r) => r.content)
        : rt === 'hot_cold_warm' || rt === 'random_draw'
          ? [...schemeGroups.value]
          : []
    for (const raw of contents) {
      if (!String(raw ?? '').trim()) continue
      const r = validateGroupContent(schemePlayConfig.value, String(raw ?? ''))
      if (!r.ok) {
        await warn(r.message)
        return
      }
    }
  }

  // 直选单式 / 直选复式 / 混合组选：不得「单独只有」111/222/333 等豹子号（含冷热/局数等入口）
  {
    const baoziContents =
      rt === 'adv_fixed_rotate'
        ? jushuList.value.map((r) => r.content)
        : rt === 'hot_cold_warm'
          ? [
              hcwSingleGroup.value
                ? (hcwPools.value[0] ?? []).join(',')
                : Array.from({ length: positionCount.value }, (_, i) => (hcwPools.value[i] ?? []).join(',')).join(
                    '\n',
                  ),
            ]
          : rt === 'random_draw' && rdWholeTicket.value
            ? [rdWholePreview.value.join(',') || '']
            : [...schemeGroups.value]
    const baoziErr = schemeSoloBaoziError(schemePlayConfig.value, baoziContents)
    if (baoziErr) {
      await warn(baoziErr)
      return
    }
  }

  cloudBusy.value = true
  flushPersistDraft()

  const cloudPayload = {
    kind: schemeKind.value,
    schemeName: name,
    lotteryCode: isBuiltinPlan.value ? '' : lottery,
    shareStatus: (isCustomKind.value ? shareStatus.value : 'private') as 'private' | 'public',
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    schemeCurrency: schemeCurrency.value,
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
      const check = await checkSchemeNameAvailable(name)
      if (!check.available) {
        if (check.existingDefinitionId && !check.existingHasInstance) {
          const resume = await confirmDialog({
            title: '名称已占用',
            message: `方案「${name}」已存在但未添加至云端。继续编辑该方案，或删除旧草稿后重新新建？`,
            tone: 'warning',
            confirmText: '继续编辑',
            cancelText: '删除重建',
          })
          if (resume) {
            await router.replace({
              name: 'advanced-scheme-edit',
              params: { schemeId: check.existingDefinitionId },
              query: { kind: 'custom' },
            })
            return
          }
          try {
            await deleteSchemeDefinition(check.existingDefinitionId)
          } catch (e) {
            ElMessage.error(e instanceof ApiError ? e.message : '删除旧方案失败')
            return
          }
        } else {
          ElMessage.error('方案名称已存在，请更换名称')
          return
        }
      }
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
            /* 回滚失败时保留定义，用户可删除后重建 */
          }
        }
        throw innerErr
      }
      return
    }

    // 已有云端实例：原地更新定义配置（勿 fork 新方案）
    if (hasCloudInstance.value) {
      await updateSchemeDefinition(schemeId.value, buildRemoteDraftPatch())
      ElMessage.success('已保存修改')
      navigateAfterCloudSave()
      return
    }

    await addSchemeToCloud(schemeId.value, cloudPayload)
    shareLocked.value = true
    remoteHasInstance.value = true
    ElMessage.success('已添加至云端（待开启）')
    navigateAfterCloudSave()
  } catch (err) {
    const message =
      err instanceof ApiError
        ? err.message
        : err instanceof Error
          ? err.message
          : hasCloudInstance.value
            ? '保存失败'
            : '添加失败'
    ElMessage.warning(message)
  } finally {
    setTimeout(() => {
      cloudBusy.value = false
    }, 1000)
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
        <span class="material-sym" aria-hidden="true">arrow_back_ios_new</span>
      </button>
      <h1 class="scf-title">{{ isDraftScheme ? '新增方案' : '方案配置' }}</h1>
      <div class="scf-header-right">
        <span v-if="instanceStatusText" class="scf-inst-badge">{{ instanceStatusText }}</span>
      </div>
    </header>

    <main class="scf-main">
      <section class="scf-section">
        <div class="scf-card scf-stack">
          <div class="scf-field">
            <label class="scf-lbl" for="scf-name">方案名称</label>
            <el-input
              id="scf-name"
              v-model="schemeName"
              size="large"
              class="scf-el-inp"
              placeholder="输入方案名称..."
              clearable
            />
          </div>
          <div v-if="!isBuiltinPlan" class="scf-field">
            <span class="scf-lbl" id="scf-lbl-lottery">彩种</span>
            <button
              v-if="identityEditable"
              type="button"
              class="scf-time-hit"
              aria-haspopup="dialog"
              :aria-expanded="identityPickerOpen && identityPickerKind === 'lottery'"
              aria-labelledby="scf-lbl-lottery scf-val-lottery"
              @click="openIdentityPicker('lottery')"
            >
              <span id="scf-val-lottery" class="scf-time-hit-val">{{
                labelOf(lotteryOptions, lotteryCode) || '请选择彩种'
              }}</span>
              <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">tune</span>
            </button>
            <div v-else class="scf-readonly">{{ gameNameDisplay }}</div>
          </div>
          <div class="scf-field">
            <span class="scf-lbl" id="scf-lbl-run">运行类型</span>
            <button
              v-if="identityEditable"
              type="button"
              class="scf-time-hit"
              aria-haspopup="dialog"
              :aria-expanded="identityPickerOpen && identityPickerKind === 'runType'"
              aria-labelledby="scf-lbl-run scf-val-run"
              @click="openIdentityPicker('runType')"
            >
              <span id="scf-val-run" class="scf-time-hit-val">{{
                labelOf(availableRunTypeOptions, runTypeId) || runTypeLabel
              }}</span>
              <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">tune</span>
            </button>
            <div v-else class="scf-readonly">{{ runTypeLabel }}</div>
          </div>
          <p v-if="isBuiltinPlan" class="scf-identity-hint">
            内置计划无需选择彩种与玩法，创建后在方案内容中选择已收藏的跟单大厅方案
          </p>
          <div v-if="!isBuiltinPlan" class="scf-field">
            <span class="scf-lbl" id="scf-lbl-play">玩法类型</span>
            <div class="scf-play-pair" role="group" aria-labelledby="scf-lbl-play">
              <button
                v-if="identityEditable"
                type="button"
                class="scf-time-hit"
                aria-haspopup="dialog"
                aria-label="玩法类型"
                :aria-expanded="identityPickerOpen && identityPickerKind === 'playType'"
                @click="openIdentityPicker('playType')"
              >
                <span id="scf-val-play" class="scf-time-hit-val">{{ playTypeLabelDisplay }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">tune</span>
              </button>
              <div v-else class="scf-readonly">{{ playTypeLabelDisplay }}</div>
              <button
                v-if="identityEditable"
                type="button"
                class="scf-time-hit"
                aria-haspopup="dialog"
                aria-label="子玩法"
                :aria-expanded="identityPickerOpen && identityPickerKind === 'subPlay'"
                @click="openIdentityPicker('subPlay')"
              >
                <span id="scf-val-sub" class="scf-time-hit-val">{{ subPlayLabelDisplay }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">tune</span>
              </button>
              <div v-else class="scf-readonly">{{ subPlayLabelDisplay }}</div>
            </div>
          </div>
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
          <div class="scf-field">
            <label class="scf-lbl" for="scf-funds">方案资金</label>
            <div class="scf-funds-row">
              <el-input
                id="scf-funds"
                v-model="schemeFunds"
                size="large"
                class="scf-el-inp scf-funds-amt"
                type="number"
                placeholder="金额"
              />
              <el-select
                v-model="schemeCurrency"
                class="scf-el-select scf-funds-cur"
                size="large"
                placeholder="币种"
                aria-label="方案币种"
              >
                <el-option
                  v-for="c in SCHEME_CURRENCY_OPTIONS"
                  :key="c"
                  :label="c"
                  :value="c"
                />
              </el-select>
            </div>
          </div>

          <div class="scf-field">
            <span class="scf-lbl scf-lbl--with-help" id="scf-lbl-runtime">
              <span>运行时间</span>
              <el-popover
                placement="top"
                :width="260"
                trigger="click"
                :content="TIME_RANGE_HINT"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="运行时间说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <div class="scf-play-pair" role="group" aria-labelledby="scf-lbl-runtime">
              <button
                type="button"
                class="scf-time-hit"
                aria-haspopup="dialog"
                aria-label="开始时间"
                @click="openTimePicker('start')"
              >
                <span class="scf-time-hit-val">{{ displayMainStart }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">schedule</span>
              </button>
              <button
                type="button"
                class="scf-time-hit"
                aria-haspopup="dialog"
                aria-label="结束时间"
                @click="openTimePicker('end')"
              >
                <span class="scf-time-hit-val">{{ displayMainEnd }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">schedule</span>
              </button>
            </div>
          </div>

          <div class="scf-field">
            <span class="scf-lbl" id="scf-lbl-sltp">止损-止盈</span>
            <div class="scf-play-pair" role="group" aria-labelledby="scf-lbl-sltp">
              <el-input
                id="scf-sl"
                v-model="stopLoss"
                size="large"
                class="scf-el-inp scf-el-inp--danger"
                placeholder="止损"
                type="number"
                aria-label="止损金额"
              />
              <el-input
                id="scf-tp"
                v-model="takeProfit"
                size="large"
                class="scf-el-inp scf-el-inp--profit"
                placeholder="止盈"
                type="number"
                aria-label="止盈金额"
              />
            </div>
          </div>

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
          <div class="scf-field">
            <span class="scf-lbl scf-lbl--with-help">
              <span>方案模式</span>
              <el-popover
                placement="top"
                :width="260"
                trigger="click"
                :content="BET_MODE_HINT"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="方案模式说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <button
              type="button"
              class="scf-time-hit"
              aria-haspopup="dialog"
              aria-label="方案模式设置"
              @click="goBetMultiplierSettings"
            >
              <span
                class="scf-time-hit-val"
                :class="{
                  'is-muted': betMultiplierFieldTone === 'muted',
                  'is-danger': betMultiplierFieldTone === 'danger',
                }"
              >{{ betMultiplierFieldText }}</span>
              <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">chevron_right</span>
            </button>
          </div>
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
        </div>

        <!-- 1/2. 定码轮换（多分组） / 固定取码（单组·每期复投） -->
        <div v-if="runTypeId === 'fixed_rotate' || runTypeId === 'fixed_number'" class="scf-groups-stack">
          <div v-for="idx in displayedGroupIndexes" :key="idx" class="scf-content-card">
            <div class="scf-group-bar">
              <h3 class="scf-group-title">
                {{ runTypeId === 'fixed_number' ? '固定号码' : `第 ${idx + 1} 组` }}
              </h3>
              <span class="scf-group-units">注数: {{ groupBetUnits(schemeGroups[idx] ?? '') }}</span>
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
              <SchemeRenxuanDanshiPanel
                v-if="schemeUsesRenxuanDanshi"
                v-model="schemeGroups[idx]"
                :config="schemePlayConfig"
              />
              <SchemeGroupInputPanel
                v-else-if="schemeUsesDigitInput"
                v-model="schemeGroups[idx]"
                :config="schemePlayConfig"
              />
              <SchemeGroupPickPanel
                v-else-if="schemeUsesPickPanel"
                v-model="schemeGroups[idx]"
                :config="schemePlayConfig"
              />
              <el-input
                v-else
                v-model="schemeGroups[idx]"
                type="textarea"
                :rows="8"
                resize="none"
                class="scf-area"
                :placeholder="groupInputPlaceholder"
              />
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
                <span class="scf-jushu-content">{{ formatJushuContentDisplay(row.content) }}</span>
              </div>
              <div class="scf-jushu-side">
                <span class="scf-jushu-jump">中后 → 第 {{ row.afterHit }} 局</span>
                <span class="scf-jushu-jump">挂后 → 第 {{ row.afterMiss }} 局</span>
                <div class="scf-jushu-actions" role="group" :aria-label="`第 ${row.ju} 局操作`">
                  <button
                    type="button"
                    class="scf-jushu-edit"
                    title="编辑局数"
                    :aria-label="`编辑第 ${row.ju} 局`"
                    @click="openJushuEditDialog(idx)"
                  >
                    <span class="scf-ms scf-ms--sm" aria-hidden="true">edit</span>
                  </button>
                  <button
                    type="button"
                    class="scf-jushu-del"
                    title="删除局数"
                    :aria-label="`删除第 ${row.ju} 局`"
                    @click="removeJushuRow(idx)"
                  >
                    <span class="scf-ms scf-ms--sm" aria-hidden="true">delete</span>
                  </button>
                </div>
              </div>
            </li>
          </ul>
        </div>

        <!-- 4. 高级开某投某：映射表 + 投向模式 -->
        <div v-else-if="runTypeId === 'adv_trigger_bet'" class="scf-content-card scf-panel">
          <div v-if="showTriggerPositionPicker" class="scf-field scf-panel-field">
            <span class="scf-lbl">投注位</span>
            <div
              class="scf-trig-pos-chips"
              role="group"
              aria-label="投注位多选"
              :style="{ '--scf-trig-pos-n': String(positionLabels.length || 5) }"
            >
              <button
                v-for="(label, idx) in positionLabels"
                :key="`trig-pos-${idx}`"
                type="button"
                class="scf-trig-pos-chip"
                :class="{ 'is-on': triggerPositionIdxs.includes(idx) }"
                :aria-pressed="triggerPositionIdxs.includes(idx)"
                @click="toggleTriggerPosition(idx)"
              >{{ label }}</button>
            </div>
          </div>
          <div class="scf-trig-toolbar">
            <div class="scf-trig-rand-ctrl">
              <span class="scf-trig-rand-lbl">随机出号</span>
              <div class="scf-stepper" role="group" aria-label="随机出号个数">
                <button
                  type="button"
                  class="scf-stepper-btn"
                  :disabled="triggerRandomCount <= 1"
                  aria-label="减少随机出号"
                  @click="triggerRandomCount = Math.max(1, triggerRandomCount - 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">remove</span>
                </button>
                <el-input
                  v-model.number="triggerRandomCount"
                  type="number"
                  inputmode="numeric"
                  maxlength="2"
                  class="scf-stepper-input scf-stepper-input--narrow"
                  :min="1"
                  :max="triggerRandomMax"
                  @change="triggerRandomCount = Math.min(triggerRandomMax, Math.max(1, Math.trunc(Number(triggerRandomCount) || 1)))"
                />
                <button
                  type="button"
                  class="scf-stepper-btn"
                  :disabled="triggerRandomCount >= triggerRandomMax"
                  aria-label="增加随机出号"
                  @click="triggerRandomCount = Math.min(triggerRandomMax, triggerRandomCount + 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">add</span>
                </button>
              </div>
            </div>
            <button type="button" class="scf-add-btn" @click="randomFillTrigger">
              <span class="scf-ms scf-ms--sm" aria-hidden="true">casino</span>
              <span>全部随机</span>
            </button>
          </div>
          <div
            class="scf-trig-grid scf-trig-grid--head"
            :class="{ 'scf-trig-grid--posrow': showTriggerPerPosColumns }"
            aria-hidden="true"
          >
            <span>启用</span>
            <span>开出</span>
            <template v-if="showTriggerPerPosColumns">
              <span>位置</span>
            </template>
            <span>正投</span>
            <span>反投</span>
          </div>
          <template v-if="showTriggerPerPosColumns">
            <div
              v-for="row in triggerRows"
              :key="row.open"
              class="scf-trig-block"
              :class="{ 'is-off': !row.enabled }"
            >
              <div
                v-for="(label, pIdx) in positionLabels"
                :key="`trig-c-${row.open}-${pIdx}`"
                class="scf-trig-grid scf-trig-grid--posrow"
              >
                <el-switch
                  v-if="pIdx === 0"
                  v-model="row.enabled"
                  size="small"
                  :aria-label="`启用开出 ${row.open} 的映射`"
                />
                <span v-else class="scf-trig-cell-placeholder" aria-hidden="true" />
                <span v-if="pIdx === 0" class="scf-trig-open">{{ row.open }}</span>
                <span v-else class="scf-trig-cell-placeholder" aria-hidden="true" />
                <span class="scf-trig-pos-name">{{ triggerPosName(label) }}</span>
                <el-input
                  :model-value="getTriggerFieldCell(row, 'pos', pIdx)"
                  size="small"
                  :placeholder="triggerInputPlaceholder"
                  inputmode="text"
                  :disabled="!row.enabled"
                  :aria-label="`${triggerPosName(label)}正投（开出 ${row.open}）`"
                  @update:model-value="(v: string | number) => writeTriggerFieldCell(row, 'pos', pIdx, String(v ?? ''))"
                  @change="commitTriggerFieldCell(row, 'pos', pIdx)"
                />
                <el-input
                  :model-value="getTriggerFieldCell(row, 'neg', pIdx)"
                  size="small"
                  :placeholder="triggerInputPlaceholder"
                  inputmode="text"
                  :disabled="!row.enabled"
                  :aria-label="`${triggerPosName(label)}反投（开出 ${row.open}）`"
                  @update:model-value="(v: string | number) => writeTriggerFieldCell(row, 'neg', pIdx, String(v ?? ''))"
                  @change="commitTriggerFieldCell(row, 'neg', pIdx)"
                />
              </div>
            </div>
          </template>
          <template v-else>
            <div
              v-for="row in triggerRows"
              :key="row.open"
              class="scf-trig-grid"
              :class="{ 'is-off': !row.enabled }"
            >
              <el-switch v-model="row.enabled" size="small" :aria-label="`启用开出 ${row.open} 的映射`" />
              <span class="scf-trig-open">{{ row.open }}</span>
              <template v-if="isTriggerTextPlay">
                <el-select
                  :model-value="triggerTextTokens(row.pos)"
                  size="small"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="正投（可多选）"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string[]) => setTriggerTextField(row, 'pos', v)"
                >
                  <el-option v-for="v in triggerBetOptions" :key="v" :label="v" :value="v" />
                </el-select>
                <el-select
                  :model-value="triggerTextTokens(row.neg)"
                  size="small"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="反投（可多选）"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string[]) => setTriggerTextField(row, 'neg', v)"
                >
                  <el-option v-for="v in triggerBetOptions" :key="`neg-${v}`" :label="v" :value="v" />
                </el-select>
              </template>
              <template v-else>
                <el-input
                  v-model="row.pos"
                  size="small"
                  :placeholder="triggerInputPlaceholder"
                  inputmode="text"
                  :disabled="!row.enabled"
                  @change="row.pos = sanitizeTriggerBetContent(row.pos)"
                />
                <el-input
                  v-model="row.neg"
                  size="small"
                  :placeholder="triggerInputPlaceholder"
                  inputmode="text"
                  :disabled="!row.enabled"
                  @change="row.neg = sanitizeTriggerBetContent(row.neg)"
                />
              </template>
            </div>
          </template>
          <div class="scf-field scf-panel-field">
            <span class="scf-lbl">投向模式</span>
            <el-radio-group v-model="triggerMode" class="scf-radio-wrap scf-radio-wrap--trigger-mode">
              <el-radio v-for="o in TRIGGER_MODE_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</el-radio>
            </el-radio-group>
          </div>
        </div>

        <!-- 5. 冷热出号（v6 仅冷/热） -->
        <div v-else-if="runTypeId === 'hot_cold_warm'" class="scf-content-card scf-panel">
          <div class="scf-hcw-bar scf-hcw-bar--top">
            <div class="scf-hcw-ctrl">
              <span class="scf-hcw-lbl">总期数</span>
              <div class="scf-stepper" role="group" aria-label="总期数">
                <button
                  type="button"
                  class="scf-stepper-btn"
                  :disabled="hcwTotalPeriods <= 20"
                  aria-label="减少总期数"
                  @click="hcwTotalPeriods = Math.max(20, hcwTotalPeriods - 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">remove</span>
                </button>
                <el-input
                  v-model.number="hcwTotalPeriods"
                  type="number"
                  inputmode="numeric"
                  maxlength="3"
                  class="scf-stepper-input"
                  :min="20"
                  :max="100"
                  @change="hcwTotalPeriods = Math.min(100, Math.max(20, Math.trunc(Number(hcwTotalPeriods) || 20)))"
                />
                <button
                  type="button"
                  class="scf-stepper-btn"
                  :disabled="hcwTotalPeriods >= 100"
                  aria-label="增加总期数"
                  @click="hcwTotalPeriods = Math.min(100, hcwTotalPeriods + 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">add</span>
                </button>
              </div>
              <button
                type="button"
                class="scf-hcw-refresh"
                :disabled="hcwLoading"
                aria-label="刷新统计"
                title="刷新统计"
                @click="loadHcwStats"
              >
                <span
                  class="scf-ms scf-ms--sm"
                  :class="{ 'scf-hcw-refresh-spin': hcwLoading }"
                  aria-hidden="true"
                >refresh</span>
              </button>
            </div>
            <div class="scf-hcw-ctrl">
              <span class="scf-hcw-lbl" title="在「最热→最冷」排序上跳过该端最极端的前 N 名（0=不跳过）">容错</span>
              <div class="scf-stepper" role="group" aria-label="容错">
                <button
                  type="button"
                  class="scf-stepper-btn"
                  :disabled="hcwFaultCount <= 0"
                  aria-label="减少容错"
                  @click="hcwFaultCount = Math.max(0, hcwFaultCount - 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">remove</span>
                </button>
                <el-input
                  v-model.number="hcwFaultCount"
                  type="number"
                  inputmode="numeric"
                  maxlength="1"
                  class="scf-stepper-input scf-stepper-input--narrow"
                  :min="0"
                  :max="9"
                  @change="hcwFaultCount = Math.min(9, Math.max(0, Math.trunc(Number(hcwFaultCount) || 0)))"
                />
                <button
                  type="button"
                  class="scf-stepper-btn"
                  :disabled="hcwFaultCount >= 9"
                  aria-label="增加容错"
                  @click="hcwFaultCount = Math.min(9, hcwFaultCount + 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">add</span>
                </button>
              </div>
            </div>
          </div>
          <div class="scf-hcw-bar scf-hcw-bar--strategy">
            <el-radio-group v-model="hcwStrategy" class="scf-hcw-strategy">
              <el-radio v-for="o in HCW_STRATEGY_OPTIONS" :key="o.value" :value="o.value">
                {{ o.label }}
              </el-radio>
            </el-radio-group>
            <span class="scf-hcw-units">总计：{{ hcwEstimatedUnits }} 注</span>
          </div>
          <div v-for="(label, pi) in hcwGroupLabels" :key="pi" class="scf-hcw-pos">
            <div class="scf-hcw-pos-head">
              <p class="scf-hcw-pos-name">{{ label }}</p>
              <div class="scf-hcw-quick" role="group" :aria-label="`${label}快捷选号`">
                <button
                  type="button"
                  class="scf-hcw-qbtn"
                  :class="{ 'is-on': hcwQuickActive(pi, 'cold') }"
                  :disabled="!hcwStatsReady || !(hcwTiers[pi]?.cold?.length)"
                  @click="applyHcwQuick(pi, 'cold')"
                >冷</button>
                <button
                  type="button"
                  class="scf-hcw-qbtn"
                  :class="{ 'is-on': hcwQuickActive(pi, 'hot') }"
                  :disabled="!hcwStatsReady || !(hcwTiers[pi]?.hot?.length)"
                  @click="applyHcwQuick(pi, 'hot')"
                >热</button>
                <button
                  type="button"
                  class="scf-hcw-qbtn"
                  :class="{ 'is-on': hcwQuickActive(pi, 'all') }"
                  :disabled="!hcwQuickTargets(pi, 'all').length"
                  @click="applyHcwQuick(pi, 'all')"
                >全</button>
                <button
                  type="button"
                  class="scf-hcw-qbtn"
                  @click="applyHcwQuick(pi, 'clear')"
                >清</button>
              </div>
            </div>
            <p v-if="!hcwStatsReady && !hcwLoading" class="scf-run-tip">
              {{ hcwAttribute ? '暂无选项频次，可点刷新重试' : '暂无开奖统计，可直接手动选号' }}
            </p>
            <div
              v-if="(hcwCellsByPos[pi] ?? []).length"
              class="scf-hcw-grid"
              :style="{
                '--hcw-cols': String(Math.min(10, (hcwCellsByPos[pi] ?? []).length) || 10),
              }"
            >
              <button
                v-for="cell in hcwCellsByPos[pi]"
                :key="cell.token"
                type="button"
                class="scf-hcw-cell"
                :class="{
                  'is-hot': cell.tier === 'hot',
                  'is-cold': cell.tier === 'cold',
                  'is-on': poolHasToken(hcwPools[pi], cell.token),
                }"
                @click="toggleHcwDigit(pi, cell.token)"
              >
                <span class="scf-hcw-cell-num">{{ cell.token }}</span>
                <span class="scf-hcw-cell-cnt">{{ cell.count == null ? '—' : cell.count }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 6. 随机出号 -->
        <div v-else-if="runTypeId === 'random_draw'" class="scf-content-card scf-panel">
          <!-- 单式整注随机 / 组选号码池随机：仅需单一数量 -->
          <template v-if="rdSingleCountMode">
            <div class="scf-rd-row">
              <span class="scf-rd-pos">{{ rdSingleCountLabel }}</span>
              <el-input-number v-model="rdCounts[0]" :min="rdSingleCountMin" :max="rdSingleCountMax" size="small" />
            </div>
          </template>
          <!-- 按位型（一星/前三/前二等）：每位只配数量，下方按位蓝色 tag 预览 -->
          <div v-else class="scf-rd-pos-grid">
            <div v-for="(label, pi) in positionLabels" :key="pi" class="scf-rd-row">
              <span class="scf-rd-pos">{{ label }}</span>
              <el-input-number
                v-model="rdCounts[pi]"
                :min="1"
                :max="rdPerPosMax"
                size="small"
              />
            </div>
          </div>
          <div class="scf-rd-toolbar">
            <el-button type="primary" plain size="small" @click="generateRdPreview">生成预览</el-button>
            <span class="scf-rd-units">预估 {{ rdEstimatedUnits }} 注</span>
            <el-radio-group v-model="rdStrategy" class="scf-rd-strategy" aria-label="换号策略">
              <el-radio v-for="o in RD_STRATEGY_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</el-radio>
            </el-radio-group>
          </div>
          <div class="scf-rd-preview-box" role="group" aria-label="预览号码">
            <template v-if="rdPreviewTags.length">
              <el-tag
                v-for="tag in rdPreviewTags"
                :key="tag.key"
                class="scf-rd-tag"
                type="primary"
                effect="dark"
                closable
                disable-transitions
                @close="removeRdPreviewTag(tag)"
              >{{ tag.label }}</el-tag>
            </template>
            <span v-else class="scf-rd-preview-empty">点击「生成预览」后在此显示</span>
          </div>
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
                <p class="scf-run-tip">内置计划配置只读，与收藏计划保持一致</p>
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

    <OptionPickerModal
      v-model="identityPickerOpen"
      :selected-value="identityPickerSelectedValue"
      :title="identityPickerTitle"
      :options="identityPickerOptions"
      selection-accent="primary"
      @confirm="onIdentityPickerConfirm"
      @cancel="onIdentityPickerCancel"
    />

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

    <el-dialog
      v-model="jushuDialogVisible"
      :title="jushuDialogTitle"
      width="min(24rem, calc(100vw - 2rem))"
      append-to-body
      align-center
      destroy-on-close
      class="scf-jushu-dialog"
      @closed="jushuEditIdx = null"
    >
      <div class="scf-jushu-form">
        <div class="scf-field">
          <span class="scf-lbl">局数</span>
          <el-input-number v-model="jushuForm.ju" :min="1" :step="1" step-strictly class="scf-jushu-num" />
        </div>
        <div class="scf-field scf-field--jushu-nums">
          <div class="scf-jushu-nums-lbl">
            <span class="scf-lbl">投注号码</span>
            <span class="scf-group-units">注数: {{ groupBetUnits(jushuForm.content) }}</span>
          </div>
          <SchemeRenxuanDanshiPanel
            v-if="schemeUsesRenxuanDanshi"
            v-model="jushuForm.content"
            :config="schemePlayConfig"
          />
          <SchemeGroupInputPanel
            v-else-if="schemeUsesDigitInput"
            v-model="jushuForm.content"
            :config="schemePlayConfig"
          />
          <SchemeGroupPickPanel
            v-else-if="schemeUsesPickPanel"
            v-model="jushuForm.content"
            :config="schemePlayConfig"
          />
          <el-input
            v-else
            v-model="jushuForm.content"
            type="textarea"
            :rows="8"
            resize="none"
            class="scf-area"
            :placeholder="groupInputPlaceholder"
          />
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
        <el-button @click="closeJushuDialog">取消</el-button>
        <el-button type="primary" @click="confirmJushuDialog">{{ jushuDialogConfirmLabel }}</el-button>
      </template>
    </el-dialog>

    <footer class="scf-footer">
      <el-button type="primary" class="scf-cloud-btn" size="large" :loading="cloudBusy" :disabled="cloudBusy"
        @click="onSaveCloud">
        <span class="scf-ms scf-ms--fill scf-cloud-ico" aria-hidden="true">save</span>
        保存修改
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
  --scf-profit: #0d9f6e;
  --scf-secondary-container: #9bb4fe;
  --scf-on-secondary-container: #f8f7ff;
  --scf-error-container: #ffdad6;
  min-height: 100dvh;
  background: var(--scf-surface);
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 400;
  padding-bottom: env(safe-area-inset-bottom);
}

/* 编辑页全文不加粗（含 Element Plus 控件） */
.scf :deep(.el-button),
.scf :deep(.el-input__inner),
.scf :deep(.el-input__wrapper),
.scf :deep(.el-select__wrapper),
.scf :deep(.el-select__placeholder),
.scf :deep(.el-select__selected-item),
.scf :deep(.el-textarea__inner),
.scf :deep(.el-radio__label),
.scf :deep(.el-checkbox__label),
.scf :deep(.el-form-item__label) {
  font-weight: 400;
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
  height: calc(var(--page-titlebar-height) + env(safe-area-inset-top));
  min-height: calc(var(--page-titlebar-height) + env(safe-area-inset-top));
  box-sizing: border-box;
  padding: env(safe-area-inset-top) var(--page-titlebar-pad-x) 0;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  box-shadow: 0 8px 32px rgba(25, 28, 30, 0.06);
}

.scf-back {
  justify-self: start;
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: #191c1e;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.scf-back .material-sym {
  font-size: var(--page-titlebar-back-icon-size);
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
  font-weight: 400;
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

.scf-main {
  padding: 1rem 1rem 0;
  max-width: 32rem;
  margin: 0 auto;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
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
  font-weight: 400;
  letter-spacing: 0;
  text-transform: none;
}

.scf-section-title {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 400;
  color: var(--scf-on-variant);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.scf-pill {
  font-size: 10px;
  font-weight: 400;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
  background: var(--scf-secondary-container);
  color: var(--scf-on-secondary-container);
}

.scf-card {
  background: #fff;
  border-radius: 0.875rem;
  padding: 0.85rem 1rem;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scf-stack {
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}

.scf-identity-hint {
  margin: 0;
  padding: 0 0.15rem;
  font-size: 0.75rem;
  line-height: 1.5;
  color: #64748b;
}

/* 双列改为单列：全页控件与「方案名称」等同宽 */
.scf-grid2 {
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}

.scf-field {
  --scf-lbl-col: 4.5rem;
  display: grid;
  grid-template-columns: var(--scf-lbl-col) minmax(0, 1fr);
  align-items: center;
  column-gap: 0.5rem;
  min-width: 0;
}

.scf-lbl {
  min-width: 0;
  width: 100%;
  font-size: 0.8125rem;
  font-weight: 400;
  color: var(--scf-on-variant);
  padding-left: 0;
  line-height: 1.3;
}

.scf-lbl--with-help {
  display: inline-flex;
  align-items: center;
  gap: 0;
  width: 100%;
  min-width: 0;
}

.scf-help-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 0.95rem;
  height: 0.95rem;
  margin: 0 0 0 -0.05rem;
  padding: 0;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.scf-help-btn:hover,
.scf-help-btn:focus-visible {
  color: var(--scf-primary);
}

.scf-help-btn:focus-visible {
  outline: 2px solid var(--scf-primary);
  outline-offset: 1px;
}

.scf-ms--help {
  font-size: 0.875rem !important;
  line-height: 1;
}

.scf-field > .scf-el-inp,
.scf-field > .scf-el-select,
.scf-field > .scf-seg,
.scf-field > .scf-readonly,
.scf-field > .scf-suffix-wrap,
.scf-field > .scf-funds-row,
.scf-field > .scf-time-hit,
.scf-field > .scf-radio-wrap,
.scf-field > .scf-play-pair {
  width: 100%;
  min-width: 0;
}

.scf-funds-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 6.25rem;
  gap: 0.5rem;
  align-items: stretch;
  width: 100%;
  min-width: 0;
}

.scf-funds-row > .scf-funds-amt,
.scf-funds-row > .scf-funds-cur {
  width: 100%;
  min-width: 0;
}

.scf-play-pair {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0.5rem;
  align-items: stretch;
}

.scf-play-pair > .scf-time-hit,
.scf-play-pair > .scf-readonly,
.scf-play-pair > .scf-el-inp {
  width: 100%;
  min-width: 0;
}

.scf-panel-field {
  align-items: start;
}

.scf-panel-field > .scf-lbl {
  padding-top: 0.35rem;
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
  font-weight: 400;
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

.scf-el-inp {
  width: 100%;
}

.scf-time-hit {
  width: 100%;
  box-sizing: border-box;
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
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: var(--scf-primary-strong);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scf-time-hit-val.is-muted {
  color: #94a3b8;
  font-weight: 400;
}

.scf-time-hit-val.is-danger {
  color: var(--scf-error);
}

.scf-time-hit-ico {
  flex-shrink: 0;
  opacity: 0.65;
  color: var(--scf-primary-strong);
}

.scf-el-inp--danger :deep(.el-input__inner) {
  color: var(--scf-error);
  font-weight: 400;
}

.scf-el-inp--profit :deep(.el-input__inner) {
  color: var(--scf-profit);
  font-weight: 400;
}

.scf-el-select {
  width: 100%;
}

.scf-el-select :deep(.el-select__wrapper) {
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  min-height: 2.5rem;
  width: 100%;
}

.scf-suffix-wrap {
  position: relative;
  width: 100%;
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
  font-weight: 400;
  color: #727687;
  pointer-events: none;
}

.scf-readonly {
  width: 100%;
  box-sizing: border-box;
  min-height: 2.5rem;
  padding: 0.55rem 0.9rem;
  border-radius: 0.5rem;
  background: rgba(230, 232, 234, 0.35);
  border: 1px solid rgba(194, 198, 216, 0.35);
  font-size: 0.9375rem;
  font-weight: 400;
  color: var(--scf-on-variant);
  display: flex;
  align-items: center;
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
  font-weight: 400;
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
  gap: 0.5rem 0.75rem;
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
  font-weight: 400;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.01em;
  color: var(--scf-primary-strong);
}

.scf-group-units {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 400;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  color: #64748b;
  letter-spacing: 0;
}

.scf-content-toolbar--group {
  flex: 0 0 auto;
  display: flex;
  justify-content: flex-end;
  align-items: stretch;
  align-self: stretch;
  margin-left: auto;
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
  font-weight: 400;
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
  padding: 1rem 1.1rem;
  min-height: 9.5rem;
  font-size: 0.9375rem;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  line-height: 1.65;
  box-shadow: none;
  white-space: pre-wrap;
}

.scf-area :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.scf-area :deep(.el-textarea__inner::placeholder) {
  color: #94a3b8;
  white-space: pre-wrap;
  word-break: break-word;
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
  font-weight: 400;
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
  font-weight: 400;
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
  font-weight: 400;
  font-size: 1.125rem;
}

.scf-tw-colon {
  align-self: center;
  font-weight: 400;
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
  font-weight: 400;
  color: #727687;
  margin-bottom: 0.25rem;
}

.scf-tw-sum-val {
  display: block;
  font-size: 1rem;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
}

.scf-tw-confirm {
  width: 100%;
  margin: 0;
  height: 3rem;
  border-radius: 0.75rem;
  font-weight: 400;
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
  font-weight: 400;
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

/* 投向模式：两行两列对齐（一直正投/反投；前正后反/前反后正） */
.scf-radio-wrap--trigger-mode {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 0.75rem;
  row-gap: 0.35rem;
  width: 100%;
}

.scf-radio-wrap--trigger-mode :deep(.el-radio) {
  margin-right: 0;
  height: auto;
  min-height: 2rem;
  align-items: center;
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
  font-weight: 400;
  color: var(--scf-primary-strong);
}

.scf-jushu-content {
  min-width: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  color: #191c1e;
  word-break: break-all;
  white-space: normal;
}

.scf-jushu-side {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.scf-jushu-jump {
  font-size: 11px;
  font-weight: 400;
  color: var(--scf-on-variant);
}

.scf-jushu-actions {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 0.1rem;
}

.scf-jushu-edit,
.scf-jushu-del {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0;
  border-radius: 0.4rem;
  line-height: 0;
}

.scf-jushu-edit {
  color: var(--scf-primary-strong, #0050cb);
}

.scf-jushu-edit:hover {
  background: rgba(0, 80, 203, 0.08);
}

.scf-jushu-del {
  color: var(--scf-error);
}

.scf-jushu-del:hover {
  background: rgba(186, 26, 26, 0.08);
}

.scf-jushu-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.scf-field--jushu-nums {
  align-items: start;
}

.scf-jushu-nums-lbl {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.25rem;
  padding-top: 0.35rem;
}

.scf-jushu-nums-lbl .scf-group-units {
  flex: none;
  font-size: 11px;
  font-weight: 400;
  color: var(--scf-on-variant);
}

.scf-jushu-num {
  width: 100%;
}

/* 开某投某 · 投注位（与 scf-field 同列，芯片均分一行） */
.scf-trig-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-bottom: 0.65rem;
}

.scf-trig-rand-ctrl {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.scf-trig-rand-lbl {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #4a5568;
  white-space: nowrap;
}

.scf-trig-pos-chips {
  --scf-trig-pos-n: 5;
  display: grid;
  grid-template-columns: repeat(var(--scf-trig-pos-n), minmax(0, 1fr));
  gap: 0.35rem;
  width: 100%;
  min-width: 0;
  padding: 0.25rem;
  border-radius: 0.65rem;
  background: rgba(242, 244, 246, 0.85);
}

.scf-trig-pos-chip {
  height: 2rem;
  margin: 0;
  padding: 0 0.2rem;
  border: none;
  border-radius: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 400;
  font-family: inherit;
  line-height: 1;
  cursor: pointer;
  background: transparent;
  color: var(--scf-on-variant);
  transition:
    background 0.15s,
    color 0.15s,
    box-shadow 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-trig-pos-chip:hover:not(.is-on) {
  background: rgba(255, 255, 255, 0.72);
  color: var(--scf-primary-strong, #0050cb);
}

.scf-trig-pos-chip.is-on {
  background: #fff;
  color: var(--el-color-primary, #0050cb);
  box-shadow: 0 2px 10px rgba(25, 28, 30, 0.08);
}

.scf-trig-pos-chip:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.35);
  outline-offset: 1px;
}

/* 开某投某映射表 */
.scf-trig-grid {
  display: grid;
  grid-template-columns: 3rem 3rem 1fr 1fr;
  align-items: center;
  gap: 0.6rem;
}

/* 前三复式等：启用|开出|位置|正投|反投，每位一行 */
.scf-trig-grid--posrow {
  grid-template-columns: 2.1rem 1.75rem 2.4rem 1fr 1fr;
  gap: 0.35rem 0.28rem;
}

.scf-trig-block {
  display: flex;
  flex-direction: column;
  gap: 0.28rem;
  padding: 0.35rem 0;
  border-radius: 0.55rem;
}

.scf-trig-block + .scf-trig-block {
  border-top: 1px solid rgba(25, 28, 30, 0.06);
}

.scf-trig-block.is-off {
  opacity: 0.55;
}

.scf-trig-block.is-off .scf-trig-open {
  opacity: 0.45;
}

.scf-trig-cell-placeholder {
  display: block;
  min-height: 1px;
}

.scf-trig-pos-name {
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--scf-on-variant);
  text-align: center;
  white-space: nowrap;
}

.scf-trig-grid--head span {
  font-size: 11px;
  font-weight: 400;
  color: var(--scf-on-variant);
  letter-spacing: 0.02em;
}

.scf-trig-grid.is-off .scf-trig-open {
  opacity: 0.35;
}

.scf-trig-open {
  font-size: 0.9375rem;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: var(--scf-primary-strong);
  text-align: center;
  padding: 0.25rem 0;
  border-radius: 0.45rem;
  background: rgba(0, 80, 203, 0.06);
}

.scf-trig-grid--posrow .scf-trig-open {
  font-size: 0.8125rem;
  padding: 0.2rem 0;
}

/* 冷热出号 */
.scf-hcw-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
}

.scf-hcw-bar--top {
  justify-content: space-between;
  flex-wrap: nowrap;
  gap: 0.65rem;
}

.scf-hcw-bar--strategy {
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: nowrap;
}

.scf-hcw-ctrl {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  flex: none;
}

.scf-hcw-refresh {
  display: grid;
  place-items: center;
  width: 1.85rem;
  height: 1.85rem;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 0.45rem;
  background: rgba(0, 80, 203, 0.08);
  color: var(--scf-primary-strong, #0050cb);
  cursor: pointer;
  transition: background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-hcw-refresh:hover:not(:disabled) {
  background: rgba(0, 80, 203, 0.14);
}

.scf-hcw-refresh:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.scf-hcw-refresh-spin {
  display: inline-block;
  animation: scf-hcw-spin 0.8s linear infinite;
}

@keyframes scf-hcw-spin {
  to {
    transform: rotate(360deg);
  }
}

.scf-hcw-strategy {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.1rem 0.45rem;
  min-width: 0;
  flex: 1 1 auto;
}

.scf-hcw-strategy :deep(.el-radio) {
  margin-right: 0;
  margin-left: 0;
  height: auto;
}

.scf-hcw-strategy :deep(.el-radio__label) {
  font-size: 0.75rem;
  font-weight: 400;
  padding-left: 0.25rem;
}

.scf-hcw-strategy :deep(.el-radio__inner) {
  width: 0.875rem;
  height: 0.875rem;
}

/* 勿用全局 .scf-lbl（width:100%），否则在 flex 行内会挤掉输入框 */
.scf-hcw-lbl {
  flex: none;
  font-size: 0.8125rem;
  font-weight: 400;
  color: var(--scf-on-variant);
  line-height: 1.3;
  white-space: nowrap;
}

/* 左减右加：紧凑步进器（最多 3 位数字） */
.scf-stepper {
  display: inline-flex;
  align-items: stretch;
  height: 1.85rem;
  border-radius: 0.45rem;
  background: #f2f4f6;
  overflow: hidden;
}

.scf-stepper-btn {
  display: grid;
  place-items: center;
  width: 1.45rem;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--scf-primary-strong);
  cursor: pointer;
  transition: background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-stepper-btn:hover:not(:disabled) {
  background: rgba(0, 80, 203, 0.08);
}

.scf-stepper-btn:disabled {
  color: rgba(66, 70, 86, 0.35);
  cursor: not-allowed;
}

.scf-stepper-btn:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.35);
  outline-offset: -2px;
}

.scf-stepper-input {
  width: 2.15rem;
}

/* 容错为单位数，缩短仅够展示 1 位 */
.scf-stepper-input--narrow {
  width: 1.35rem;
}

.scf-stepper-input :deep(.el-input__wrapper) {
  height: 1.85rem;
  padding: 0;
  box-shadow: none !important;
  background: transparent;
  border-radius: 0;
}

.scf-stepper-input :deep(.el-input__inner) {
  height: 1.85rem;
  line-height: 1.85rem;
  padding: 0;
  text-align: center;
  font-size: 0.8125rem;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
  /* 隐藏 number 原生上下箭头 */
  -moz-appearance: textfield;
}

.scf-stepper-input :deep(.el-input__inner::-webkit-outer-spin-button),
.scf-stepper-input :deep(.el-input__inner::-webkit-inner-spin-button) {
  -webkit-appearance: none;
  margin: 0;
}

.scf-hcw-units {
  flex: none;
  font-size: 0.75rem;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: var(--scf-primary-strong, #0050cb);
  white-space: nowrap;
}

.scf-hcw-pos {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem 0.85rem;
  border-radius: 0.75rem;
  background: rgba(247, 249, 251, 0.9);
}

.scf-hcw-pos-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  min-width: 0;
}

.scf-hcw-pos-name {
  margin: 0;
  flex: none;
  font-size: 0.8125rem;
  font-weight: 400;
  color: var(--scf-on-variant);
}

.scf-hcw-quick {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-left: auto;
}

.scf-hcw-qbtn {
  width: 1.85rem;
  height: 1.85rem;
  margin: 0;
  padding: 0;
  border: 1px solid rgba(194, 198, 216, 0.55);
  border-radius: 0.4rem;
  background: #fff;
  color: #424656;
  font-size: 0.75rem;
  font-weight: 400;
  font-family: inherit;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s,
    border-color 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-hcw-qbtn:hover:not(:disabled):not(.is-on) {
  border-color: rgba(0, 80, 203, 0.45);
  color: var(--scf-primary-strong);
}

.scf-hcw-qbtn.is-on {
  background: var(--el-color-primary, #0050cb);
  border-color: var(--el-color-primary, #0050cb);
  color: #fff;
}

.scf-hcw-qbtn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.scf-hcw-qbtn:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.35);
  outline-offset: 1px;
}

.scf-hcw-grid {
  display: grid;
  grid-template-columns: repeat(var(--hcw-cols, 10), minmax(0, 1fr));
  gap: 0.3rem;
  width: 100%;
  overflow: visible;
}

.scf-hcw-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.1rem;
  min-width: 0;
  margin: 0;
  padding: 0.35rem 0.1rem 0.3rem;
  border: 1px solid transparent;
  border-radius: 0.5rem;
  background: #fff;
  cursor: pointer;
  font-family: inherit;
  transition:
    box-shadow 0.15s,
    background 0.15s,
    border-color 0.15s,
    color 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-hcw-cell-num {
  font-size: 0.9375rem;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  line-height: 1.15;
  color: var(--scf-on-variant);
}

.scf-hcw-cell-cnt {
  font-size: 10px;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  color: var(--scf-on-variant);
  opacity: 0.85;
}

.scf-hcw-cell.is-hot .scf-hcw-cell-num,
.scf-hcw-cell.is-hot .scf-hcw-cell-cnt {
  color: #e53935;
}

.scf-hcw-cell.is-cold .scf-hcw-cell-num,
.scf-hcw-cell.is-cold .scf-hcw-cell-cnt {
  color: #b0b4be;
}

.scf-hcw-cell.is-on {
  border-color: rgba(0, 80, 203, 0.45);
  background: rgba(0, 80, 203, 0.08);
  box-shadow: 0 2px 10px rgba(0, 80, 203, 0.12);
}

.scf-hcw-cell.is-on.is-hot {
  border-color: rgba(229, 57, 53, 0.45);
  background: rgba(229, 57, 53, 0.1);
  box-shadow: 0 2px 10px rgba(229, 57, 53, 0.14);
}

.scf-hcw-cell.is-on.is-cold {
  border-color: rgba(176, 180, 190, 0.7);
  background: rgba(176, 180, 190, 0.16);
  box-shadow: none;
}

/* 随机出号 */
.scf-rd-pos-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.65rem 0.55rem;
  width: 100%;
}

.scf-rd-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  min-width: 0;
}

.scf-rd-pos {
  flex-shrink: 0;
  min-width: 2rem;
  font-size: 0.8125rem;
  font-weight: 400;
  color: var(--scf-on-variant);
}

.scf-rd-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem 0.85rem;
  flex-wrap: wrap;
  margin: 0.65rem 0 0.75rem;
  width: 100%;
}

.scf-rd-units {
  flex-shrink: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary, #64748b);
}

.scf-rd-strategy {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.1rem 0.35rem;
  flex: 1 1 12rem;
  min-width: 0;
}

.scf-rd-strategy :deep(.el-radio) {
  margin-right: 0;
  margin-left: 0;
  height: auto;
  flex: 1 1 0;
  justify-content: center;
}

.scf-rd-strategy :deep(.el-radio__label) {
  font-size: 0.75rem;
  padding-left: 0.3rem;
  white-space: nowrap;
}

.scf-rd-strategy :deep(.el-radio__inner) {
  width: 0.875rem;
  height: 0.875rem;
}

.scf-rd-preview-box {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
  min-height: 2.75rem;
  padding: 0.55rem 0.65rem;
  border-radius: 0.55rem;
  background: rgba(242, 244, 246, 0.55);
}

.scf-rd-preview-empty {
  font-size: 0.8125rem;
  font-weight: 500;
  color: #94a3b8;
}

.scf-rd-tag {
  --el-tag-bg-color: #40a9ff;
  --el-tag-border-color: #40a9ff;
  --el-tag-hover-color: #1890ff;
  --el-tag-text-color: #fff;
  border: none !important;
  border-radius: 0.25rem;
  font-size: 0.8125rem;
  font-weight: 600;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
  line-height: 1.4;
  background-color: #40a9ff !important;
  color: #fff !important;
}

.scf-rd-tag :deep(.el-tag__close) {
  color: #fff;
  margin-left: 0.2rem;
}

.scf-rd-tag :deep(.el-tag__close:hover) {
  background: rgba(255, 255, 255, 0.28);
  color: #fff;
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
  font-weight: 400;
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
  font-weight: 400;
  color: #191c1e;
  word-break: break-all;
}

.scf-bp-meta {
  font-size: 11px;
  font-weight: 400;
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
  font-weight: 400;
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

<style>
/* popover 挂到 body，需非 scoped；浮层不占文档流，不挤动页面布局 */
.scf-help-popper.el-popper {
  max-width: min(16.5rem, calc(100vw - 2rem));
  padding: 0.65rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 400;
  line-height: 1.55;
  color: #334155;
  border: none;
  border-radius: 0.65rem;
  box-shadow: 0 12px 36px rgba(25, 28, 30, 0.12);
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}
</style>
