<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog } from '@/utils/confirmDialog'
import { validateBuiltinPlanSave } from '@/utils/builtinPlanSave'
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
import {
  LHC_SX_DUIPENG_MAX_PICKS,
  LHC_GUOGUAN_OPTIONS,
  LHC_GUOGUAN_POSITION_LABELS,
  LHC_GUOGUAN_TRIGGER_OPEN_VALUES,
  LHC_SX_DUIPENG_MIN_PICKS,
  LHC_WS_DUIPENG_MAX_PICKS,
  LHC_WS_DUIPENG_MIN_PICKS,
  LHC_SW_DUIPENG_MAX_PICKS,
  LHC_SW_DUIPENG_MIN_PICKS,
  LHC_SW_DUIPENG_OPTIONS,
  LHC_TAIL_OPTIONS,
  LHC_TEMA_QUICK_OPTIONS,
  LHC_ZODIACS,
  isLhcSxDuipengConfig,
  isLhcWsDuipengConfig,
  isLhcSwDuipengConfig,
  isLhcRenyiDuipengConfig,
  lhcTemaHcwUniverse,
  lhcGuoguanAttrsForNumber,
  parseLhcGuoguanPositions,
  pickRandomLhcSwDuipengPair,
  randomLhcGuoguanContent,
} from '@/constants/lhcPlay'
import {
  isRandomDrawLhcRenyiDuipengConfig,
  normalizeLhcRenyiDuipengRandomCounts,
  normalizeLhcRenyiDuipengTriggerContent,
  randomLhcRenyiDuipengContent,
  randomLhcRenyiDuipengContentForCounts,
} from '@/utils/lhcRenyiDuipengRandom'
import {
  normalizeLhcRenyiDuipengHotColdRanks,
  replaceLhcRenyiDuipengHotColdRanks,
} from '@/utils/lhcRenyiDuipengHotCold'
import { toggleSingleHcwRank } from '@/utils/hcwRankSelection'
import SchemeGroupPickPanel from '@/components/schemes/SchemeGroupPickPanel.vue'
import SchemeGroupInputPanel from '@/components/schemes/SchemeGroupInputPanel.vue'
import SchemeLhcTemaPanel from '@/components/schemes/SchemeLhcTemaPanel.vue'
import SchemeLhcRenyiDuipengPanel from '@/components/schemes/SchemeLhcRenyiDuipengPanel.vue'
import SchemeLhcGuoguanPanel from '@/components/schemes/SchemeLhcGuoguanPanel.vue'
import SchemeRenxuanDanshiPanel from '@/components/schemes/SchemeRenxuanDanshiPanel.vue'
import {
  adaptSchemeGroupContentForPlay,
  catalogFieldsFromPlayConfig,
  countBetUnits,
  expandZhixuanPositionPoolToDanshi,
  expandZhixuanPoolToDanshiWithoutBaozi,
  formatLhcRenyiDuipengContent,
  groupContentPlaceholder,
  isBaoziDigitTicket,
  isHunhePlayConfig,
  isYixingDingweiPlayConfig,
  isBetLimitExceededMessage,
  betAmountExceedsMax,
  calcBetAmount,
  maxBetAmountExceededMessage,
  maxModeMultiplierFromPayload,
  isSchemeSoloBaoziContent,
  isSoloBaoziRestrictedPlay,
  isSscDanshiLikeConfig,
  isZu3DanshiConfig,
  isZu3DigitTicket,
  isZu6DanshiConfig,
  isZu6DigitTicket,
  isZuxuanDanshiConfig,
  isZhixuanFushiPlayConfig,
  randomZu3DanshiTickets,
  randomZu6DanshiTickets,
  ZU3_DANSHI_FORM_COUNT,
  ZU6_DANSHI_FORM_COUNT,
  isZhixuanPositionPoolContent,
  normalizeHunheGroupContent,
  normalizeZuxuanDanshiContent,
  WEISHU_MAX_BET_UNITS,
  YIXING_MAX_PICKS_MSG,
  YIXING_MAX_PICKS_PER_POS,
  isRenxuanNeedsPositionConfig,
  isRenxuanPositionDanshiConfig,
  buildRenxuanPositionContent,
  normalizeZu3DanshiContent,
  normalizeZu6DanshiContent,
  SSC_POSITION_LABELS,
  playConfigSummary,
  validateGroupContent,
  validateSchemeGroups,
  isLhcLianmaFushiConfig,
  isLhcGuoguanConfig,
  isLhcTemaPlayConfig,
  normalizeLhcTemaFlatContent,
  normalizeLhcTemaContent,
  lhcTemaInvalidTokens,
  parseLhcZodiacTokens,
  parseLhcTailTokens,
  parseLhcSwDuipengTokens,
  schemeSoloBaoziError,
  zhixuanFushiMaxBetUnits,
  greedyHezhiKuaduPicksUnderMax,
  maxHezhiKuaduRandomCount,
  zuxuanPoolMinPick,
  zuxuanPoolMinPickMessage,
  budingweiMinPicks,
  budingweiMinPicksMessage,
  isZuDualPlayConfig,
  randomZuDualContentForConfig,
  parseZuDualZones,
  zuDualMetaOf,
  uniqueDigitsFromRun,
} from '@/utils/betPayload'
import { defaultPlaySelection, formatSubPlayLabel } from '@/utils/playConfig'
import { normalizeSchemeTimePairFromConfig, schemeTimeRangeError } from '@/utils/schemeDateTime'
import { usePublicLotteries } from '@/composables/usePublicLotteries'
import { usePlayTreeConfig } from '@/composables/usePlayTreeConfig'
import { longhuPickOptionsForConfig } from '@/utils/longhuPickOptions'
import {
  commitSchemeGroupContentOnBlur,
  digitOptionsForConfig,
  lhcTailChipLabel,
  poolMaxPicksForConfig,
  schemeGroupContentToInputBox,
  schemeGroupUsesDigitInput,
  schemeGroupUsesLhcTemaPanel,
  schemeGroupUsesLhcRenyiDuipengPanel,
  schemeGroupUsesPickPanel,
  schemeGroupUsesTextInputPanel,
  textPickOptionsForConfig,
} from '@/utils/pickPanelOptions'
import {
  filterPlayTypesForRunType,
  filterSubPlaysForRunType,
  isLonghuPlayConfigLike,
  isPc28HezhiConfigLike,
  isPc28ModeConfigLike,
  isPerPosDxdsPlayConfig,
  isWuxingSumDxdsPlayConfig,
  PER_POS_DXDS_OPTIONS,
  sscDigitDxdsAttrs,
  lotteryHasAdvTriggerPlay,
  isZhixuanDanshiPerPosPlay,
  isRenxuanNeedsPositionTriggerPlay,
  isRenxuanPerPosTriggerPlay,
  isRenxuanHcwOpenPosPlay,
  isHcwZuDualPlay,
  isRenxuanHcwZuDualPlay,
  isTailParityPlayConfig,
  defaultRenxuanTriggerPositionIdxs,
  defaultRenxuanHcwOpenPositionIdxs,
  supportsAdvTriggerPerPosColumns,
  supportsAdvTriggerPositionPicker,
  syncRunTypePlaySelection,
  validateRunTypePlaySelection,
} from '@/utils/runTypeMatrix'
import type { PlayTypeNode } from '@/types/playCatalog'
import {
  clearSchemeDraft,
  consumeSchemeEditBmPending,
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
import {
  syncAdvancedTemplatesInPayload,
  syncDraftAdvancedTemplatesToServer,
} from '@/utils/draftAdvancedTemplates'
import type { BetMultiplierPayload } from '@/api/schemes/betMultiplier'
import { simBetFromSchemeConfig } from '@/utils/schemeSimBet'
import { PRIMARY_CURRENCIES, type PrimaryCurrency } from '@/api/guaji/accounts'
import { normalizeSchemeMultiplier } from '@/api/cloud/center'

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
  '方案保存后将自动同步至精算云中心。开始/结束时间按北京时间（UTC+8）判定；须同时填写，或同时留空表示无限期运行。'
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

/** 编辑中只保留数字；失焦/变更时规范为 ≥1 的正整数 */
function onMultCoeffInput(v: string | number) {
  multCoeff.value = String(v ?? '').replace(/[^\d]/g, '')
}
function normalizeMultCoeff() {
  multCoeff.value = normalizeSchemeMultiplier(multCoeff.value)
}
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

/** 当前玩法用 chip 选号时不再展示 textarea（避免双轨编辑） */
const schemeUsesPickPanel = computed(() => schemeGroupUsesPickPanel(schemePlayConfig.value))
/** 六合彩特码：19 属性快捷项 + 0–49 号码输入框 */
const schemeUsesLhcTemaPanel = computed(() => schemeGroupUsesLhcTemaPanel(schemePlayConfig.value))
/** 二全中任意对碰：A区 / B区双输入框（1–49，| 分隔） */
const schemeUsesLhcRenyiDuipengPanel = computed(() =>
  schemeGroupUsesLhcRenyiDuipengPanel(schemePlayConfig.value),
)
const schemeUsesLhcGuoguanPanel = computed(() => isLhcGuoguanConfig(schemePlayConfig.value))
/** 数字玩法方案内容改用输入框录入（对齐第三方，不点选） */
const schemeUsesDigitInput = computed(() => schemeGroupUsesDigitInput(schemePlayConfig.value))
/** 复式数字框或非任选单式整注框（带失焦校验） */
const schemeUsesTextInputPanel = computed(() => schemeGroupUsesTextInputPanel(schemePlayConfig.value))
/** 任选非直选复式：定码/固定/局数等走选位壳（单式票面或号池/和值） */
const schemeUsesRenxuanDanshi = computed(() => isRenxuanNeedsPositionConfig(schemePlayConfig.value))

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

/** 方案内容标题：优先身份区已选玩法文案，避免树未同步时 g001/272 被映射成「前三·272」 */
const playModeSummary = computed(() => {
  const pt = playTypeLabelDisplay.value
  const sp = subPlayLabelDisplay.value
  if (pt && pt !== '—' && sp && sp !== '—') return `${pt} · ${sp}`
  return playConfigSummary(schemePlayConfig.value)
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
    void loadPlayTree()
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

/** 玩法号码池：与 digitOptionsForConfig 一致（和值 0–27 不补零，避免开某投某 open「00」对不上「0」丢映射） */
const numberPoolTokens = computed<string[]>(() => {
  const opts = digitOptionsForConfig(schemePlayConfig.value as Parameters<typeof digitOptionsForConfig>[0])
  return opts.length ? opts : [...ALL_DIGITS]
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
  // 跳转目标允许指向尚未添加的局号（逐局录入时很常见）；引擎侧目标不存在时回第 1 局
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

/** 裸 textarea 失焦：按当前玩法规则规范化（与 SchemeGroupInputPanel 同源） */
function commitSchemeGroupAt(idx: number): void {
  const cur = schemeGroups.value[idx] ?? ''
  const next = commitSchemeGroupContentOnBlur(cur, schemePlayConfig.value)
  if (next !== cur) schemeGroups.value[idx] = next
}

function commitJushuFormContent(): void {
  const cur = jushuForm.value.content ?? ''
  const next = commitSchemeGroupContentOnBlur(cur, schemePlayConfig.value)
  if (next !== cur) jushuForm.value.content = next
}

/** 局数列表展示：按录入框原版（压缩）格式，不把引擎换行渲染成多行 */
function formatJushuContentDisplay(content: string): string {
  const raw = String(content ?? '').replace(/\r/g, '')
  if (!raw.trim()) return ''
  // 任选带位名前缀（万,个\n0）：勿走号池压缩，否则「万,个」被剥掉只剩号码
  if (schemeUsesRenxuanDanshi.value) {
    if (raw.includes('\n') || raw.includes('|')) {
      return raw
        .replace(/\|/g, '\n')
        .split('\n')
        .map((l) => l.trim())
        .filter(Boolean)
        .join(', ')
    }
    return raw.trim()
  }
  // 任意对碰：保持 A|B 展示，勿把 | 当成换行压缩
  if (schemeUsesLhcRenyiDuipengPanel.value) {
    return raw.trim()
  }
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
/** 投注选位（可多选）：0=万 … 4=个；任选 ≥k ≤5，定位胆等可多选 */
const triggerPositionIdxs = ref<number[]>([0])
/** 任选开奖选位（单选）：上期该位球号查开出映射 */
const triggerOpenPositionIdx = ref(0)
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
        triggerOpenPositionIdx.value = 0
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

/** 任选非直选复式：开某投某需开奖选位 + 投注选位区 */
const isRenxuanTriggerPlay = computed(() =>
  isRenxuanNeedsPositionTriggerPlay(schemePlayConfig.value),
)

/** 任选·直选单式：启用区按投注选位分列正/反投 */
const isRenxuanPerPosTrigger = computed(() =>
  isRenxuanPerPosTriggerPlay(schemePlayConfig.value),
)

/** 任选须勾选的最少投注位数（任二=2） */
const triggerRenPosNeed = computed(() => {
  const k =
    schemePlayConfig.value.renPositionCount ??
    schemePlayConfig.value.segmentLen ??
    2
  return k >= 2 && k <= 5 ? k : 2
})

/** 投注位芯片：任选显示开奖/投注双选位；一星等仍按位分列不单勾 */
const showTriggerPositionPicker = computed(() => {
  if (runTypeId.value !== 'adv_trigger_bet') return false
  return supportsAdvTriggerPositionPicker(schemePlayConfig.value)
})

/** 一星定位胆 / 前三复式 / 中三 / 任选直选单式 / 后二大小单双等：按位分列正投/反投 */
const showTriggerPerPosColumns = computed(() => {
  if (runTypeId.value !== 'adv_trigger_bet') return false
  if (isLonghuPlay.value) return false
  return supportsAdvTriggerPerPosColumns(schemePlayConfig.value)
})

/**
 * 启用区分列标签：
 * 任选直选单式固定「第1…k位」（与投注选位个数无关）；
 * 其它任选提示用投注绝对位名；其余玩法用段位名。
 */
const triggerColumnLabels = computed(() => {
  if (isRenxuanPerPosTrigger.value) {
    return Array.from({ length: triggerRenPosNeed.value }, (_, i) => `第${i + 1}位`)
  }
  if (isRenxuanTriggerPlay.value) {
    return triggerBetPositionLabels.value
  }
  return positionLabels.value
})

/** 任选投注选位区已选绝对位名（万千百十个），作出票前缀 */
const triggerBetPositionLabels = computed(() => {
  if (!isRenxuanTriggerPlay.value) return [] as string[]
  return triggerPositionIdxs.value.map(
    (i) => SSC_POSITION_LABELS[i] ?? `第${i + 1}位`,
  )
})

/** 选位芯片展示的全部位名（任选：万千百十个） */
const triggerPickerLabels = computed(() => {
  if (isRenxuanTriggerPlay.value) return [...SSC_POSITION_LABELS]
  return positionLabels.value
})

/** 任选开奖选位说明（问号气泡） */
const TRIGGER_OPEN_POS_HINT = '开奖选位：上期该位球号查开出映射行。'

/** 特码 / 二全中复式等：开某投某「开出」说明（问号气泡） */
const TRIGGER_TEMA_OPEN_HINT = '以上期开奖特码为开号标准'
const TRIGGER_SX_DP_OPEN_HINT = '以上期开奖特码对应生肖为开号标准'
const TRIGGER_WS_DP_OPEN_HINT = '以上期开奖特码对应尾数为开号标准'
const TRIGGER_SW_DP_OPEN_HINT = '以上期开奖特码对应生肖或尾数为开号标准'

/** 开出旁展示「特码/生肖/尾数开号标准」问号：特码/正特、二全中复式、生肖/尾数对碰 */
const showTriggerTemaOpenHint = computed(
  () =>
    isLhcTemaPlayConfig(schemePlayConfig.value) ||
    isLhcLianmaFushiConfig(schemePlayConfig.value) ||
    isLhcSxDuipengConfig(schemePlayConfig.value) ||
    isLhcWsDuipengConfig(schemePlayConfig.value) ||
    isLhcSwDuipengConfig(schemePlayConfig.value),
)

const triggerOpenHintText = computed(() => {
  if (isLhcSxDuipengConfig(schemePlayConfig.value)) return TRIGGER_SX_DP_OPEN_HINT
  if (isLhcWsDuipengConfig(schemePlayConfig.value)) return TRIGGER_WS_DP_OPEN_HINT
  if (isLhcSwDuipengConfig(schemePlayConfig.value)) return TRIGGER_SW_DP_OPEN_HINT
  return TRIGGER_TEMA_OPEN_HINT
})

/** 任选投注选位说明（问号气泡） */
const triggerBetPosHint = computed(() => {
  const need = triggerRenPosNeed.value
  const bet = triggerBetPositionLabels.value.join('/') || '所选投注位'
  return `投注选位至少选 ${need} 个（当前${bet}），取该行各位号码组合出票。`
})

/** 组选单式 / 组选号池等：开出说明（任选改用标题旁问号气泡，不再展示本段） */
const triggerSegmentOpenTip = computed(() => {
  if (runTypeId.value !== 'adv_trigger_bet') return ''
  const cfg = schemePlayConfig.value
  if (isRenxuanTriggerPlay.value) return ''
  const labels = (cfg.segmentLabels ?? []).filter(Boolean)
  const posHint =
    labels.length >= 2 ? labels.join('/') : cfg.segmentLen >= 2 ? `前 ${cfg.segmentLen} 位` : ''
  if (!posHint) return ''
  if (isZuxuanDanshiConfig(cfg)) {
    return `开出：${posHint}任一位开出该号码即命中（多号同时开出时按${labels[0] || '首位'}优先，只投一行）；正/反投填组选单式整注（如 12,13）。`
  }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (
    bm === 'zuxuan_fs' ||
    bm === 'zu3' ||
    bm === 'zu6' ||
    /组选复式|组三|组六/.test(cfg.playMethodLabel ?? '')
  ) {
    return `开出：${posHint}任一位开出该号码即命中（多号同时开出时按${labels[0] || '首位'}优先，只投一行）；正/反投填组选号码池（逗号多选）。`
  }
  return ''
})

/** 按位列里的正/反投是否用文字多选（大小单双），开出仍为球号 0-9 */
const triggerPerPosTextBet = computed(
  () => showTriggerPerPosColumns.value && triggerBetOptions.value.length > 0,
)

/** 后二大小单双等：每位正/反投仅单选一个文字选项 */
const triggerPerPosTextSingle = computed(
  () => triggerPerPosTextBet.value && poolMaxPicksForConfig(schemePlayConfig.value) === 1,
)

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

/** 按位分列列数：任选直选单式=玩法 k（任二=2），其余=玩法段长 */
function triggerPerPosColumnCount(): number {
  if (isRenxuanPerPosTrigger.value) {
    return triggerRenPosNeed.value
  }
  return Math.max(1, positionCount.value)
}

function getTriggerFieldCell(row: SchemeTriggerRow, field: 'pos' | 'neg', idx: number): string {
  return triggerFieldParts(row[field], triggerPerPosColumnCount())[idx] ?? ''
}

function writeTriggerFieldCell(
  row: SchemeTriggerRow,
  field: 'pos' | 'neg',
  idx: number,
  raw: string,
): void {
  const parts = triggerFieldParts(row[field], triggerPerPosColumnCount())
  parts[idx] = String(raw ?? '')
  row[field] = parts.join('\n')
}

function commitTriggerFieldCell(row: SchemeTriggerRow, field: 'pos' | 'neg', idx: number): void {
  const parts = triggerFieldParts(row[field], triggerPerPosColumnCount())
  parts[idx] = sanitizeTriggerBetContent(parts[idx] ?? '')
  row[field] = parts.join('\n')
}

/**
 * 任选直选单式开某投某：第1…k 位号池展开为单式票，前缀用投注选位区绝对位名。
 */
function renxuanTriggerPoolToGroupContent(raw: string): string {
  const k = triggerRenPosNeed.value
  const parts = triggerFieldParts(String(raw ?? ''), k)
  if (parts.some((c) => !String(c ?? '').trim())) return ''
  for (const cell of parts) {
    const digits = String(cell)
      .replace(/，/g, ',')
      .split(/[,，\s]+/)
      .map((t) => t.trim())
      .filter(Boolean)
    if (!digits.length || digits.some((t) => !/^\d+$/.test(t))) return ''
  }
  const expanded = expandZhixuanPositionPoolToDanshi(parts.join('\n'), k)
  if (!expanded.trim()) return ''
  const posLabels = triggerBetPositionLabels.value.length
    ? [...triggerBetPositionLabels.value]
    : defaultRenxuanTriggerPositionIdxs(k).map((i) => SSC_POSITION_LABELS[i] ?? String(i))
  return buildRenxuanPositionContent(posLabels, expanded)
}

function defaultTriggerPositionIdxs(): number[] {
  if (isRenxuanNeedsPositionTriggerPlay(schemePlayConfig.value)) {
    return defaultRenxuanTriggerPositionIdxs(triggerRenPosNeed.value)
  }
  const n = Math.max(1, positionCount.value)
  return Array.from({ length: n }, (_, i) => i)
}

/** 投注位下标空间：任选为五星绝对位 0–4；其余为玩法段长 */
function triggerPosSpaceSize(): number {
  return isRenxuanTriggerPlay.value ? 5 : Math.max(1, positionCount.value)
}

function ensureTriggerOpenPosition(): void {
  if (!isRenxuanTriggerPlay.value) return
  const idx = Math.trunc(Number(triggerOpenPositionIdx.value))
  triggerOpenPositionIdx.value =
    Number.isInteger(idx) && idx >= 0 && idx < 5 ? idx : 0
}

function ensureTriggerPositions(): void {
  if (!showTriggerPositionPicker.value) return
  const n = triggerPosSpaceSize()
  const need = isRenxuanTriggerPlay.value ? triggerRenPosNeed.value : 1
  const cur = triggerPositionIdxs.value
    .filter((i) => Number.isInteger(i) && i >= 0 && i < n)
    .filter((i, idx, arr) => arr.indexOf(i) === idx)
    .sort((a, b) => a - b)
  if (isRenxuanTriggerPlay.value) {
    triggerPositionIdxs.value =
      cur.length >= need && cur.length <= 5 ? cur : defaultTriggerPositionIdxs()
    ensureTriggerOpenPosition()
    return
  }
  triggerPositionIdxs.value = cur.length ? cur : defaultTriggerPositionIdxs()
}

function triggerOpenValues(): string[] {
  if (isLhcGuoguanConfig(schemePlayConfig.value)) return [...LHC_GUOGUAN_TRIGGER_OPEN_VALUES]
  if (isLonghuPlay.value) return longhuPickValues.value
  // 生肖对碰：开出=特码生肖（十二生肖）
  if (isLhcSxDuipengConfig(schemePlayConfig.value)) {
    return [...LHC_ZODIACS]
  }
  // 尾数对碰：开出=特码尾数（0–9）
  if (isLhcWsDuipengConfig(schemePlayConfig.value)) {
    return [...LHC_TAIL_OPTIONS]
  }
  // 生尾对碰：开出=特码生肖或特码尾
  if (isLhcSwDuipengConfig(schemePlayConfig.value)) {
    return [...LHC_SW_DUIPENG_OPTIONS]
  }
  if (isPc28HezhiConfigLike(schemePlayConfig.value) && isPc28PlayLine()) {
    return [...PC28_HEZHI_VALUES]
  }
  const bm = schemePlayConfig.value.betMode ?? ''
  if (bm === 'hezhi' && isPc28PlayLine()) {
    return [...PC28_HEZHI_VALUES]
  }
  // 按位玩法（直选复式 / 后二大小单双）：开出按该位球号 0-9
  if (supportsAdvTriggerPerPosColumns(schemePlayConfig.value)) {
    return [...numberPoolTokens.value]
  }
  const textOpts = textPickOptionsForConfig(schemePlayConfig.value)
  if (textOpts.length) return textOpts
  return [...numberPoolTokens.value]
}

/** 开出键：数字按数值归一（"00"/"0" 同源），文字原样 */
function triggerOpenKey(open: string): string {
  const t = String(open ?? '').trim()
  if (/^\d+$/.test(t)) return String(Number(t))
  return t
}

/** 尾数/生尾对碰展示：0 → 0尾（落库/映射键仍为 0） */
function formatWsDpTokenLabel(token: string): string {
  const sw = isLhcSwDuipengConfig(schemePlayConfig.value)
  const ws = isLhcWsDuipengConfig(schemePlayConfig.value)
  if (!ws && !sw) return token
  if ((LHC_TAIL_OPTIONS as readonly string[]).includes(String(token).replace(/尾$/, ''))) {
    return lhcTailChipLabel(token) || token
  }
  return token
}

function ensureTriggerRows(): void {
  const opens = triggerOpenValues()
  const cur = triggerRows.value
  if (cur.length === opens.length && cur.every((r, i) => r.open === opens[i])) return

  // 按归一化键保留正/反投，避免和值「00」↔「0」重建时把映射洗成空
  const byKey = new Map<string, SchemeTriggerRow>()
  for (const r of cur) {
    const k = triggerOpenKey(r.open)
    if (k && !byKey.has(k)) byKey.set(k, r)
  }

  if (triggerRowsLocked && cur.length) {
    const compatible =
      cur.length === opens.length && opens.every((o) => byKey.has(triggerOpenKey(o)))
    if (compatible) {
      triggerRows.value = opens.map((open) => {
        const prev = byKey.get(triggerOpenKey(open))!
        return {
          enabled: prev.enabled,
          open,
          pos: String(prev.pos ?? ''),
          neg: String(prev.neg ?? ''),
        }
      })
      return
    }
    triggerRowsLocked = false
  }
  triggerRows.value = opens.map((open) => {
    const prev = byKey.get(triggerOpenKey(open))
    return prev
      ? { enabled: prev.enabled, open, pos: String(prev.pos ?? ''), neg: String(prev.neg ?? '') }
      : { enabled: true, open, pos: '', neg: '' }
  })
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

function selectTriggerOpenPosition(idx: number, ev?: Event): void {
  if (!Number.isInteger(idx) || idx < 0 || idx >= 5) return
  triggerOpenPositionIdx.value = idx
  blurTrigPosChip(ev)
}

function toggleTriggerPosition(idx: number, ev?: Event): void {
  const max = triggerPosSpaceSize()
  if (!Number.isInteger(idx) || idx < 0 || idx >= max) return
  // 任选：投注选位至少 k、最多 5
  if (isRenxuanTriggerPlay.value) {
    const need = triggerRenPosNeed.value
    const cur = [...triggerPositionIdxs.value]
    const at = cur.indexOf(idx)
    if (at >= 0) {
      if (cur.length <= need) return
      cur.splice(at, 1)
    } else {
      if (cur.length >= 5) return
      cur.push(idx)
    }
    cur.sort((a, b) => a - b)
    triggerPositionIdxs.value = cur
    blurTrigPosChip(ev)
    return
  }
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
  blurTrigPosChip(ev)
}

function applyTriggerBetFromConfig(raw: unknown): void {
  if (!raw || typeof raw !== 'object') return
  const tb = raw as {
    rows?: unknown
    mode?: unknown
    positionIdx?: unknown
    positionIdxs?: unknown
    openPositionIdx?: unknown
  }
  if (Array.isArray(tb.rows) && tb.rows.length) {
    const rows: SchemeTriggerRow[] = []
    const tema = isLhcTemaPlayConfig(schemePlayConfig.value)
    for (const item of tb.rows) {
      if (!item || typeof item !== 'object') continue
      const r = item as Record<string, unknown>
      const pos = String(r.pos ?? '')
      const neg = String(r.neg ?? '')
      rows.push({
        enabled: r.enabled !== false,
        open: String(r.open ?? ''),
        // 编辑态用 flat「01,02,大,蓝波」；库内可能是 号码|属性|波色
        pos: tema ? normalizeLhcTemaFlatContent(pos) : pos,
        neg: tema ? normalizeLhcTemaFlatContent(neg) : neg,
      })
    }
    if (rows.length) {
      triggerRows.value = rows
      triggerRowsLocked = true
      // 与玩法 watch 对齐，避免远端灌入后被当成「换玩法」解锁并重建洗空
      lastTriggerPlayKey = `${playTypeId.value}:${subPlayId.value}`
    }
  }
  const mode = String(tb.mode ?? '')
  if (mode === 'always_pos' || mode === 'always_neg' || mode === 'alt_pos_first' || mode === 'alt_neg_first') {
    triggerMode.value = mode
  }
  if (tb.positionIdxs != null || tb.positionIdx != null) {
    const n = triggerPosSpaceSize()
    const idxs = Array.isArray(tb.positionIdxs)
      ? normalizeTriggerPositionIdxs(tb.positionIdxs, n)
      : normalizeTriggerPositionIdxs(tb.positionIdx, n)
    if (isRenxuanTriggerPlay.value) {
      const need = triggerRenPosNeed.value
      triggerPositionIdxs.value =
        idxs.length >= need && idxs.length <= 5 ? idxs : defaultTriggerPositionIdxs()
    } else {
      triggerPositionIdxs.value = idxs
    }
  } else if (isRenxuanTriggerPlay.value) {
    triggerPositionIdxs.value = defaultTriggerPositionIdxs()
  }
  if (isRenxuanTriggerPlay.value) {
    if (tb.openPositionIdx != null && Number.isFinite(Number(tb.openPositionIdx))) {
      const oi = Math.trunc(Number(tb.openPositionIdx))
      triggerOpenPositionIdx.value = oi >= 0 && oi < 5 ? oi : 0
    } else if (triggerPositionIdxs.value.length) {
      triggerOpenPositionIdx.value = triggerPositionIdxs.value[0] ?? 0
    } else {
      triggerOpenPositionIdx.value = 0
    }
  }
}

/**
 * 组选/组三/组六/任选混合组选等：正/反投为整注（前二每注 2 位，如 12,13；组三如 112；混合如 012,345），不是单位数号池。
 * 与按位分列的直选单式/中三混合组选互斥。
 */
const isTriggerWholeTicketBet = computed(() => {
  if (runTypeId.value !== 'adv_trigger_bet') return false
  if (showTriggerPerPosColumns.value) return false
  if (isTriggerTextPlay.value) return false
  const cfg = schemePlayConfig.value
  // 任三组三/组六单式 betMode 可能为 danshi，不落 isZuxuanDanshiConfig，须单独认
  // 任选混合组选：一行正/反投，每注 3 位逗号分隔
  return (
    isZuxuanDanshiConfig(cfg) ||
    isZu3DanshiConfig(cfg) ||
    isZu6DanshiConfig(cfg) ||
    isHunhePlayConfig(cfg)
  )
})

/** 整注位数：前二组选单式=2，前三/组三/组六/混合组选=3 */
const triggerWholeTicketLen = computed(() => {
  if (
    isZu3DanshiConfig(schemePlayConfig.value) ||
    isZu6DanshiConfig(schemePlayConfig.value) ||
    isHunhePlayConfig(schemePlayConfig.value)
  ) {
    return 3
  }
  const n = Math.trunc(Number(schemePlayConfig.value.segmentLen) || 0)
  return n >= 2 && n <= 5 ? n : 2
})

/** 包胆等：正/反投每格仅单号（与第三方单胆一致） */
const isTriggerSinglePickBet = computed(() => {
  if (isTriggerTextPlay.value) return false
  if (isTriggerWholeTicketBet.value) return false
  const cap = poolMaxPicksForConfig(schemePlayConfig.value)
  return cap != null && cap === 1
})

/** 生尾对碰（开某投某 / 随机出号共用） */
function isSchemeSwDuipeng(): boolean {
  const cfg = schemePlayConfig.value
  return (
    isLhcSwDuipengConfig(cfg) || String(cfg.betMode ?? '').toLowerCase() === 'sw_dp'
  )
}

/** 任意对碰（高级开某投某全部随机时生成 A区|B区） */
function isSchemeRenyiDuipeng(): boolean {
  const cfg = schemePlayConfig.value
  return isLhcRenyiDuipengConfig(cfg) || String(cfg.betMode ?? '').toLowerCase() === 'renyi_dp'
}

/** 随机出号个数下限（双区对碰至少两号） */
const triggerRandomMin = computed(() => {
  if (isLhcGuoguanConfig(schemePlayConfig.value)) return 1
  if (isSchemeRenyiDuipeng()) return 2
  if (isSchemeSwDuipeng()) return LHC_SW_DUIPENG_MIN_PICKS
  if (isLhcSxDuipengConfig(schemePlayConfig.value)) return LHC_SX_DUIPENG_MIN_PICKS
  if (isLhcWsDuipengConfig(schemePlayConfig.value)) return LHC_WS_DUIPENG_MIN_PICKS
  return 1
})

/** 随机出号个数上限 = 正/反投号池大小（至少 1） */
const triggerRandomMax = computed(() => {
  if (isLhcGuoguanConfig(schemePlayConfig.value)) return 6
  // 任意对碰总号数受 A区+B区最多 10 个号码的校验约束。
  if (isSchemeRenyiDuipeng()) return 10
  // 组选/组三/组六/混合整注：随机「注数」上限（前二≈45；组三 90；组六 120；混合 165）
  if (isTriggerWholeTicketBet.value) {
    if (isZu3DanshiConfig(schemePlayConfig.value)) return ZU3_DANSHI_FORM_COUNT
    if (isZu6DanshiConfig(schemePlayConfig.value)) return ZU6_DANSHI_FORM_COUNT
    // 混合组选形态：组六 120 + 组三 45 = 165（不含豹子）
    if (isHunhePlayConfig(schemePlayConfig.value)) return 165
    const len = triggerWholeTicketLen.value
    if (len === 2) return 45
    return 90
  }
  // 特码：01–49 + 16 属性 + 3 波色 = 68（勿只按开出号池 49）
  if (isLhcTemaPlayConfig(schemePlayConfig.value)) {
    return Math.max(1, temaTriggerRandomPool().length)
  }
  // 生尾对碰：恰好 1 肖 + 1 尾（勿按开出宇宙 22 抬上限）
  if (isSchemeSwDuipeng()) return LHC_SW_DUIPENG_MAX_PICKS
  // 按位大小单双：正反投池为大/小/单/双；其余用开出号池
  const poolMax = Math.max(
    1,
    triggerPerPosTextBet.value ? triggerBetOptions.value.length : triggerOpenValues().length,
  )
  const cfg = schemePlayConfig.value
  // 包胆：每格只能 1 个胆码
  const pickCap = poolMaxPicksForConfig(cfg)
  if (pickCap != null && pickCap > 0 && pickCap < poolMax) {
    return pickCap
  }
  // 直选复式：每格随机个数须使位积 ≤ 最大注数（前三 10³=1000>900 → 上限 9）
  if (isZhixuanFushiPlayConfig(cfg) && positionCount.value > 1) {
    const maxUnits = zhixuanFushiMaxBetUnits(cfg)
    if (maxUnits > 0) {
      let k = poolMax
      while (k > 1 && k ** positionCount.value > maxUnits) k -= 1
      return Math.max(1, k)
    }
  }
  return poolMax
})

watch([triggerRandomMin, triggerRandomMax], ([min, max]) => {
  const lo = Math.max(1, Math.trunc(Number(min) || 1))
  const hi = Math.max(lo, Math.trunc(Number(max) || lo))
  const cur = Math.trunc(Number(triggerRandomCount.value) || lo)
  if (cur < lo || cur > hi) triggerRandomCount.value = Math.min(hi, Math.max(lo, cur))
})

/** 取 count 个不重复随机项，逗号拼接（count 由「随机出号」步进器决定） */
function randomTriggerMultiValue(count: number, poolSrc?: string[]): string {
  const pool = [...(poolSrc?.length ? poolSrc : triggerOpenValues())]
  if (!pool.length) return '0'
  const n = Math.min(pool.length, Math.max(1, Math.trunc(count) || 1))
  for (let i = pool.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[pool[i], pool[j]] = [pool[j]!, pool[i]!]
  }
  return pool.slice(0, n).join(',')
}

/** 生尾对碰正/反投随机：固定 1 肖 + 1 尾 → 肖|尾 */
function randomSwDuipengTriggerContent(): string {
  return pickRandomLhcSwDuipengPair().join('|')
}

/** 组选/组三/组六单式：随机 count 注定长整注（形态去重） */
function randomTriggerWholeTickets(count: number): string {
  const want = Math.min(triggerRandomMax.value, Math.max(1, Math.trunc(count) || 1))
  if (isZu3DanshiConfig(schemePlayConfig.value)) {
    return randomZu3DanshiTickets(want, triggerRandomMax.value)
  }
  if (isZu6DanshiConfig(schemePlayConfig.value)) {
    return randomZu6DanshiTickets(want, triggerRandomMax.value)
  }
  const len = triggerWholeTicketLen.value
  const out: string[] = []
  const seen = new Set<string>()
  let guard = 0
  while (out.length < want && guard++ < 800) {
    let t = ''
    for (let i = 0; i < len; i++) t += String(Math.floor(Math.random() * 10))
    // 组选：各位不全相同（前二排除对子）
    if ([...t].every((c) => c === t[0])) continue
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out.join(',')
}

/** 特码开某投某随机池：1–49 + 属性/波色 */
function temaTriggerRandomPool(): string[] {
  const nums = Array.from({ length: 49 }, (_, i) => String(i + 1).padStart(2, '0'))
  return [...nums, ...LHC_TEMA_QUICK_OPTIONS]
}

function commitTriggerTemaField(row: SchemeTriggerRow, field: 'pos' | 'neg'): void {
  const raw = String(row[field] ?? '')
  const invalid = lhcTemaInvalidTokens(raw)
  row[field] = normalizeLhcTemaFlatContent(raw)
  if (invalid.length) {
    ElMessage.warning(`已忽略无效项：${invalid.join('、')}（仅支持 1–49 与大/小/单/双/红波等）`)
  }
}

/** 「全部随机」：纯前端一次性填表辅助，引擎下注不涉及随机；每格取「随机出号」个号 */
function randomFillTrigger(): void {
  const minN = triggerRandomMin.value
  const count = Math.min(
    triggerRandomMax.value,
    Math.max(minN, Math.trunc(triggerRandomCount.value) || minN),
  )
  triggerRandomCount.value = count
  const posN = triggerPerPosColumnCount()
  // 按位大小单双：正/反投从大/小/单/双抽，勿用开出球号池
  const betPool = triggerPerPosTextBet.value ? [...triggerBetOptions.value] : undefined
  const zuDual = isZuDualPlayConfig(schemePlayConfig.value)
  const temaPool = isLhcTemaPlayConfig(schemePlayConfig.value) ? temaTriggerRandomPool() : undefined
  const swDp = isSchemeSwDuipeng()
  const renyiDp = isSchemeRenyiDuipeng()
  const guoguan = isLhcGuoguanConfig(schemePlayConfig.value)
  for (const row of triggerRows.value) {
    if (showTriggerPerPosColumns.value) {
      row.pos = Array.from({ length: posN }, () => randomTriggerMultiValue(count, betPool)).join('\n')
      row.neg = Array.from({ length: posN }, () => randomTriggerMultiValue(count, betPool)).join('\n')
    } else if (zuDual) {
      row.pos = randomZuDualContent(count)
      row.neg = randomZuDualContent(count)
    } else if (isTriggerWholeTicketBet.value) {
      row.pos = randomTriggerWholeTickets(count)
      row.neg = randomTriggerWholeTickets(count)
    } else if (swDp) {
      // 生尾对碰：勿从 22 项混合池 slice → 两肖/两尾；固定各抽 1 肖 + 1 尾
      row.pos = randomSwDuipengTriggerContent()
      row.neg = randomSwDuipengTriggerContent()
    } else if (renyiDp) {
      row.pos = randomLhcRenyiDuipengContent(count)
      row.neg = randomLhcRenyiDuipengContent(count)
    } else if (guoguan) {
      row.pos = randomLhcGuoguanContent(count)
      row.neg = randomLhcGuoguanContent(count)
    } else {
      const pool = temaPool ?? (isTriggerTextPlay.value ? [...triggerBetOptions.value] : undefined)
      row.pos = randomTriggerMultiValue(count, pool)
      row.neg = randomTriggerMultiValue(count, pool)
    }
  }
  ElMessage.success(
    renyiDp
      ? `已随机填充正投 / 反投（每格 ${count} 个号，A区/B区各至少 1 个）`
      : swDp
      ? '已随机填充正投 / 反投（每格 1 生肖 + 1 尾数）'
      : zuDual
        ? `已随机填充正投 / 反投（${zuDualZoneHeadLabel()} + 单号，每格至少 1 注）`
        : isTriggerWholeTicketBet.value
          ? `已随机填充正投 / 反投（每格 ${count} 注，每注 ${triggerWholeTicketLen.value} 位）`
          : guoguan
            ? `已随机填充正投 / 反投（每格 ${count} 个正码位置）`
            : `已随机填充正投 / 反投号码（每格 ${count} 个号）`,
  )
}

function normalizeLhcGuoguanTriggerContent(raw: string): string {
  const normalized = String(raw ?? '').replace(/，/g, ',')
  const positions = parseLhcGuoguanPositions(normalized)
  return positions ? positions.join(',') : normalized
}

/** 规范化按位正投/反投（换行分位，每位内可逗号多号） */
function sanitizeTriggerPerPosField(raw: string): string {
  const n = triggerPerPosColumnCount()
  return triggerFieldParts(raw, n)
    .map((cell) => sanitizeTriggerBetContent(cell))
    .join('\n')
}

/** 规范化单个投注 token（数字池 / PC28 和值） */
function sanitizeOneTriggerToken(raw: string): string {
  const p = String(raw ?? '').trim()
  if (!p) return ''
  const cfg = schemePlayConfig.value
  if (isLhcTemaPlayConfig(cfg)) {
    // 特码由 sanitizeTriggerBetContent 整段处理（可含 大/蓝波）
    return p
  }
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

/** 组选/组三/组六单式整注：须恰好 segmentLen 位数字；组三两同+一异；组六三位互异 */
function sanitizeOneTriggerWholeTicket(raw: string): string {
  const digits = String(raw ?? '').replace(/\D/g, '')
  const len = triggerWholeTicketLen.value
  if (digits.length !== len) return ''
  if (![...digits].every((ch) => ch >= '0' && ch <= '9')) return ''
  if (isZu3DanshiConfig(schemePlayConfig.value)) {
    return isZu3DigitTicket(digits) ? digits : ''
  }
  if (isZu6DanshiConfig(schemePlayConfig.value)) {
    return isZu6DigitTicket(digits) ? digits : ''
  }
  // 各位不全相同（前二 11 无效；前三 111 无效）
  if ([...digits].every((c) => c === digits[0])) return ''
  return digits
}

/**
 * 正投/反投内容：支持逗号分隔多个号码（如 1,3,5）。
 * 组选单式：每注定长整注（前二如 12,13），勿压成单位数。
 * 中文逗号会归一为英文逗号；去重保序。
 * 包胆等单选玩法只保留第一个合法号码。
 */
function sanitizeTriggerBetContent(v: string): string {
  const raw = String(v ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!raw) return ''
  // 任意对碰的 A|B 是一个完整投注内容，不能按通用逗号号码 token 清洗。
  if (isLhcRenyiDuipengConfig(schemePlayConfig.value)) {
    return normalizeLhcRenyiDuipengTriggerContent(raw)
  }
  // 特码·特码A：正/反投支持「01,02,大,蓝波」混选；下单再合成 号码|属性|波色
  if (isLhcTemaPlayConfig(schemePlayConfig.value)) {
    return normalizeLhcTemaFlatContent(raw)
  }
  // 含换行时按位规范化（勿用逗号切碎把「4\\n5」粘成 45）
  if (raw.includes('\n') || raw.includes('\r')) {
    return sanitizeTriggerPerPosField(raw)
  }
  // 双区组选：保留「头区,尾区」形态，勿把各区当单位数 token 清掉
  if (isZuDualPlayConfig(schemePlayConfig.value)) {
    const meta = zuDualMetaOf(schemePlayConfig.value)
    if (meta) {
      const zones = parseZuDualZones(raw, meta.minHead, meta.minTail, meta.equalCounts)
      if (zones) return zones.normalized
    }
    const parts = raw.split(',')
    if (parts.length === 2) {
      const a = uniqueDigitsFromRun(parts[0] ?? '').join('')
      const b = uniqueDigitsFromRun(parts[1] ?? '').join('')
      if (a || b) return `${a},${b}`
    }
    return uniqueDigitsFromRun(raw).join('')
  }
  if (isTriggerWholeTicketBet.value) {
    const parts = raw.split(',').map((s) => s.trim()).filter(Boolean)
    const out: string[] = []
    const seen = new Set<string>()
    for (const p of parts) {
      const tok = sanitizeOneTriggerWholeTicket(p)
      if (!tok) continue
      // 组选形态去重：12 与 21 同注，保序留首次
      const formKey = [...tok].sort().join('')
      if (seen.has(formKey)) continue
      seen.add(formKey)
      out.push(tok)
    }
    return out.join(',')
  }
  const parts = raw.split(',').map((s) => s.trim()).filter(Boolean)
  const out: string[] = []
  const seen = new Set<string>()
  for (const p of parts) {
    const tok = sanitizeOneTriggerToken(p)
    if (!tok || seen.has(tok)) continue
    seen.add(tok)
    out.push(tok)
  }
  const cap = poolMaxPicksForConfig(schemePlayConfig.value)
  if (cap != null && cap > 0 && out.length > cap) {
    return out.slice(0, cap).join(',')
  }
  return out.join(',')
}

/** 文字玩法：字符串 ↔ 多选数组（生肖对碰兼容 马|蛇；尾数对碰兼容 0|1） */
function triggerTextTokens(v: string): string[] {
  if (isLhcSxDuipengConfig(schemePlayConfig.value)) {
    return parseLhcZodiacTokens(v)
  }
  if (isLhcWsDuipengConfig(schemePlayConfig.value)) {
    return parseLhcTailTokens(v)
  }
  if (isLhcSwDuipengConfig(schemePlayConfig.value)) {
    // 编辑中可只选了一侧；落库时再要求肖+尾齐全
    const zs = parseLhcZodiacTokens(v).slice(0, 1)
    const ts = parseLhcTailTokens(v).slice(0, 1)
    return [...zs, ...ts]
  }
  return String(v ?? '')
    .replace(/，/g, ',')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 生肖对碰正/反投落库：恰好 2 肖 → 肖A|肖B */
function normalizeSxDuipengTriggerContent(raw: string): string {
  const zs = parseLhcZodiacTokens(raw).slice(0, LHC_SX_DUIPENG_MAX_PICKS)
  if (zs.length < LHC_SX_DUIPENG_MIN_PICKS) return zs.join('|')
  return `${zs[0]}|${zs[1]}`
}

/** 尾数对碰正/反投落库：恰好 2 尾 → 尾A|尾B */
function normalizeWsDuipengTriggerContent(raw: string): string {
  const ts = parseLhcTailTokens(raw).slice(0, LHC_WS_DUIPENG_MAX_PICKS)
  if (ts.length < LHC_WS_DUIPENG_MIN_PICKS) return ts.join('|')
  return `${ts[0]}|${ts[1]}`
}

/** 生尾对碰正/反投落库：恰好 1 肖 + 1 尾 → 肖|尾 */
function normalizeSwDuipengTriggerContent(raw: string): string {
  const parts = parseLhcSwDuipengTokens(raw)
  if (parts.length < LHC_SW_DUIPENG_MIN_PICKS) return String(raw ?? '').trim()
  return `${parts[0]}|${parts[1]}`
}

/** 双区组选开某投某：是否用「头区/尾区」双行录入 */
const isTriggerZuDualInput = computed(() => isZuDualPlayConfig(schemePlayConfig.value))

const triggerZuDualMeta = computed(() => zuDualMetaOf(schemePlayConfig.value))

function zuDualZoneHeadLabel(): string {
  return triggerZuDualMeta.value?.headLabel ?? '二重号'
}

const triggerZuDualZoneLabels = computed(() => {
  const m = triggerZuDualMeta.value
  return m ? ([m.headLabel, m.tailLabel] as const) : (['二重号', '单号'] as const)
})

const triggerZuDualZonePlaceholders = computed(() => {
  const m = triggerZuDualMeta.value
  if (!m) return (['如 12', '如 3234'] as const)
  const [a = '', b = ''] = m.example.split(',')
  return ([`如 ${a || '1'}`, `如 ${b || '2'}`] as const)
})

function zuDualMinSinglesCount(): number {
  return triggerZuDualMeta.value?.minTail ?? 2
}

function zuDualMinHeadCount(): number {
  return triggerZuDualMeta.value?.minHead ?? 1
}

function randomZuDualContent(headCount: number, tailCount?: number): string {
  return randomZuDualContentForConfig(schemePlayConfig.value, headCount, tailCount)
}

/** 读取双区某一段（0=头区，1=单号）；无逗号时整段视为头区 */
function getZuDualTriggerZone(raw: string, zone: 0 | 1): string {
  const text = String(raw ?? '').replace(/，/g, ',')
  if (!text) return ''
  const idx = text.indexOf(',')
  if (idx < 0) return zone === 0 ? text : ''
  return zone === 0 ? text.slice(0, idx) : text.slice(idx + 1)
}

/** 写入双区某一段，始终拼成「二重,单号」供落库/出站 */
function setZu12TriggerZone(
  row: SchemeTriggerRow,
  field: 'pos' | 'neg',
  zone: 0 | 1,
  value: string,
): void {
  const digits = String(value ?? '').replace(/\D/g, '')
  const a = zone === 0 ? digits : getZuDualTriggerZone(row[field], 0).replace(/\D/g, '')
  const b = zone === 1 ? digits : getZuDualTriggerZone(row[field], 1).replace(/\D/g, '')
  if (!a && !b) {
    row[field] = ''
    return
  }
  row[field] = `${a},${b}`
}

function commitZu12TriggerField(row: SchemeTriggerRow, field: 'pos' | 'neg'): void {
  row[field] = sanitizeTriggerBetContent(row[field])
}

function setTriggerTextField(row: SchemeTriggerRow, field: 'pos' | 'neg', vals: string[]): void {
  // 生尾对碰：多选时同侧替换，落库 肖|尾（勿允许两肖/两尾）
  if (isSchemeSwDuipeng()) {
    let z = ''
    let t = ''
    for (const v of vals ?? []) {
      const s = String(v ?? '').trim()
      if ((LHC_ZODIACS as readonly string[]).includes(s)) z = s
      else if ((LHC_TAIL_OPTIONS as readonly string[]).includes(s)) t = s
    }
    row[field] = [z, t].filter(Boolean).join('|')
    return
  }
  const allow = new Set(triggerBetOptions.value)
  const max = poolMaxPicksForConfig(schemePlayConfig.value)
  const out: string[] = []
  const seen = new Set<string>()
  for (const v of vals ?? []) {
    const t = String(v ?? '').trim()
    if (!t || !allow.has(t) || seen.has(t)) continue
    seen.add(t)
    out.push(t)
    if (max != null && max > 0 && out.length >= max) break
  }
  row[field] =
    isLhcSxDuipengConfig(schemePlayConfig.value) || isLhcWsDuipengConfig(schemePlayConfig.value)
      ? out.join('|')
      : out.join(',')
}

/** 单档文字正/反投（五星和值单双等）仅单选 */
const triggerTextSingle = computed(
  () => isTriggerTextPlay.value && !triggerPerPosTextBet.value && poolMaxPicksForConfig(schemePlayConfig.value) === 1,
)

/** 按位文字正/反投（后二大小单双：每位一格大/小/单/双） */
function setTriggerTextFieldCell(
  row: SchemeTriggerRow,
  field: 'pos' | 'neg',
  idx: number,
  vals: string[],
): void {
  const allow = new Set(triggerBetOptions.value)
  const max = poolMaxPicksForConfig(schemePlayConfig.value)
  const out: string[] = []
  const seen = new Set<string>()
  for (const v of vals ?? []) {
    const t = String(v ?? '').trim()
    if (!t || !allow.has(t) || seen.has(t)) continue
    seen.add(t)
    out.push(t)
    if (max != null && max > 0 && out.length >= max) break
  }
  writeTriggerFieldCell(row, field, idx, out.join(','))
}

function sanitizeTriggerPerPosTextField(raw: string): string {
  const n = triggerPerPosColumnCount()
  const allow = new Set(triggerBetOptions.value)
  const max = poolMaxPicksForConfig(schemePlayConfig.value)
  return triggerFieldParts(raw, n)
    .map((cell) => {
      let toks = triggerTextTokens(cell).filter((t) => allow.has(t))
      if (max != null && max > 0 && toks.length > max) toks = toks.slice(0, max)
      return toks.join(',')
    })
    .join('\n')
}

const triggerInputPlaceholder = computed(() => {
  if (isLhcTemaPlayConfig(schemePlayConfig.value)) {
    return '如 01,02,大,蓝波（号码与属性逗号分隔）'
  }
  if (isPc28HezhiConfigLike(schemePlayConfig.value) && isPc28PlayLine()) {
    return '如 1,2,15'
  }
  const bm = schemePlayConfig.value.betMode ?? ''
  if (bm === 'hezhi' && isPc28PlayLine()) {
    return '如 1,2,15'
  }
  // 组选12/4：双行录入时用分区 placeholder
  if (isTriggerZuDualInput.value) {
    return triggerZuDualZoneLabels.value.join(' / ')
  }
  // 组选/组三/混合：每注 N 位，逗号分隔多注
  if (isTriggerWholeTicketBet.value) {
    if (isZu3DanshiConfig(schemePlayConfig.value)) return '如 112,223（两同+一异）'
    if (isZu6DanshiConfig(schemePlayConfig.value)) return '如 012,345（三位互异）'
    if (isHunhePlayConfig(schemePlayConfig.value)) return '如 012,345（每注3位，不含豹子）'
    const n = triggerWholeTicketLen.value
    if (n === 2) return '如 12,13（每注2位）'
    if (n === 3) return '如 123,135（每注3位）'
    return `如 ${'12'.padEnd(n, '0')},${'13'.padEnd(n, '0')}（每注${n}位）`
  }
  // 包胆：每格仅一个 0–9
  if (isTriggerSinglePickBet.value) {
    const pool = numberPoolTokens.value
    return pool.length ? `如 ${pool[Math.min(5, pool.length - 1)]}` : '如 5'
  }
  const pool = numberPoolTokens.value
  if (!pool.length) return '如 1,3,5'
  if (pool.length <= 10) return `如 ${pool[0]},${pool[1] ?? pool[0]},${pool[2] ?? pool[0]}`
  return `如 ${pool[0]},${pool[1]},${pool[2]}`
})

// --- 任选冷热/随机：选位（对齐定码任二直选单式，≥k ≤5） ---
const renxuanRunPosIdxs = ref<number[]>([])
/** 任选·直选单式冷热：开奖选位（恰好 k 个），频次按这些绝对位统计 */
const hcwOpenPosIdxs = ref<number[]>([])

const schemeUsesRenxuanRunPos = computed(() => isRenxuanNeedsPositionConfig(schemePlayConfig.value))

/** 任选直选单式 / 任选混合组选冷热：双选位区（开奖恰好 k + 投注 ≥k） */
const isRenxuanHcwDualPosPlay = computed(() =>
  isRenxuanHcwOpenPosPlay(schemePlayConfig.value),
)

/** 任选·组选12/4：投注选位说明（四星等定点星段无选位栏） */
const isRenxuanHcwZuDual = computed(() => isRenxuanHcwZuDualPlay(schemePlayConfig.value))

/** 组选12/4 冷热双池（任四 / 四星等）：二重或三重 + 单号 */
const isHcwZuDual = computed(() => isHcwZuDualPlay(schemePlayConfig.value))

/** 直选单式/混合组选冷热需要独立开奖选位；组选双区按投注选位/玩法位计频 */
const showHcwOpenPosition = computed(() => isRenxuanHcwDualPosPlay.value)

const renxuanRunPosNeed = computed(() => {
  const k =
    schemePlayConfig.value.renPositionCount ??
    schemePlayConfig.value.segmentLen ??
    2
  return k >= 2 && k <= 5 ? k : 2
})

function sameIntIdxs(a: number[], b: number[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

function ensureRenxuanRunPositions(): void {
  if (!schemeUsesRenxuanRunPos.value) return
  const need = renxuanRunPosNeed.value
  const cur = renxuanRunPosIdxs.value
    .filter((i) => Number.isInteger(i) && i >= 0 && i < 5)
    .filter((i, idx, arr) => arr.indexOf(i) === idx)
    .sort((a, b) => a - b)
  // 已有合法选位（2…5）一律保留：玩法元数据异步回填时 need 可能瞬时变成 5，
  // 若此时用「cur.length >= need」会把万千误扩成万千百十个。
  let next: number[]
  if (cur.length === 0) {
    next = defaultRenxuanTriggerPositionIdxs(need)
  } else if (cur.length > 5) {
    next = cur.slice(0, 5)
  } else {
    next = cur
  }
  if (!sameIntIdxs(renxuanRunPosIdxs.value, next)) {
    renxuanRunPosIdxs.value = next
  }
}

/** 清洗开奖选位下标（不去默认补齐，避免取消后被写回万位） */
function sanitizeHcwOpenPosIdxs(raw: number[]): number[] {
  return raw
    .filter((i) => Number.isInteger(i) && i >= 0 && i < 5)
    .filter((i, idx, arr) => arr.indexOf(i) === idx)
    .sort((a, b) => a - b)
}

/**
 * 规范化开奖选位（仅直选单式冷热）：
 * 空→默认前 k；多于 k→截断；1…k-1 原样保留。
 */
function ensureHcwOpenPositions(): void {
  if (!isRenxuanHcwDualPosPlay.value) return
  const need = renxuanRunPosNeed.value
  const cur = sanitizeHcwOpenPosIdxs(hcwOpenPosIdxs.value)
  let next = cur
  if (cur.length === 0) {
    next = defaultRenxuanHcwOpenPositionIdxs(need)
  } else if (cur.length > need) {
    next = cur.slice(0, need)
  }
  if (!sameIntIdxs(hcwOpenPosIdxs.value, next)) {
    hcwOpenPosIdxs.value = next
  }
}

function blurTrigPosChip(ev?: Event): void {
  const el = ev?.currentTarget
  if (el instanceof HTMLElement) el.blur()
}

function toggleHcwOpenPosition(idx: number, ev?: Event): void {
  if (!showHcwOpenPosition.value) return
  if (!Number.isInteger(idx) || idx < 0 || idx >= 5) return
  const need = renxuanRunPosNeed.value
  const cur = sanitizeHcwOpenPosIdxs(hcwOpenPosIdxs.value)
  const at = cur.indexOf(idx)
  if (at >= 0) {
    // 允许取消任意已选（含万位）；可暂时少于 k，再点其他位补齐
    if (cur.length <= 1) return
    cur.splice(at, 1)
  } else {
    if (cur.length >= need) {
      ElMessage.warning(`开奖选位须选 ${need} 个，请先取消一个再选`)
      return
    }
    cur.push(idx)
  }
  const next = sanitizeHcwOpenPosIdxs(cur)
  if (sameIntIdxs(hcwOpenPosIdxs.value, next)) return
  hcwOpenPosIdxs.value = next
  // 勿 ensureHcwOpenPositions：会在缺位时补默认万位
  ensureHcwPools()
  scheduleHcwStats(true)
  blurTrigPosChip(ev)
}

function toggleHcwBetPosition(idx: number, ev?: Event): void {
  if (!schemeUsesRenxuanRunPos.value) return
  if (!Number.isInteger(idx) || idx < 0 || idx >= 5) return
  const need = renxuanRunPosNeed.value
  const cur = [...renxuanRunPosIdxs.value]
  const at = cur.indexOf(idx)
  if (at >= 0) {
    if (cur.length <= need) return
    cur.splice(at, 1)
  } else {
    if (cur.length >= 5) return
    cur.push(idx)
  }
  renxuanRunPosIdxs.value = cur.sort((a, b) => a - b)
  blurTrigPosChip(ev)
  // 任选和值/尾数/组选复式/组选12：频次随投注选位重算
  if (
    runTypeId.value === 'hot_cold_warm' &&
    (hcwAttribute.value || hcwDigitOverall.value || isHcwZuDual.value)
  ) {
    scheduleHcwStats(true)
  }
}

/** 只读标签（勿在 computed 内 ensure，避免写回自身依赖触发递归更新） */
const renxuanRunPosLabelsComputed = computed(() =>
  renxuanRunPosIdxs.value.map((i) => SSC_POSITION_LABELS[i] ?? String(i)),
)

function renxuanRunPosLabels(): string[] {
  return renxuanRunPosLabelsComputed.value
}

/**
 * 任选冷热/随机：按位号池（如 1,2\\n3,4\\n5,6）先展成整注再加选位前缀，
 * 与后端 applyRenxuanRunPositionWrap 一致；混合/组三/组六再按形态过滤。
 */
function wrapRenxuanRunContent(picks: string): string {
  if (!schemeUsesRenxuanRunPos.value) return picks
  let body = String(picks ?? '').replace(/\r/g, '')
  const cfg = schemePlayConfig.value
  if (isRenxuanPositionDanshiConfig(cfg)) {
    const digitLen =
      cfg.segmentLen > 0 ? cfg.segmentLen : renxuanRunPosNeed.value
    if (digitLen > 1 && isZhixuanPositionPoolContent(body, digitLen)) {
      const expanded = expandZhixuanPositionPoolToDanshi(body, digitLen)
      if (expanded) {
        if (isHunhePlayConfig(cfg)) {
          body = normalizeHunheGroupContent(expanded, digitLen) || expanded
        } else if (isZu3DanshiConfig(cfg)) {
          body = normalizeZu3DanshiContent(expanded, digitLen) || expanded
        } else if (isZu6DanshiConfig(cfg)) {
          body = normalizeZu6DanshiContent(expanded, digitLen) || expanded
        } else {
          body = expanded
        }
      }
    }
  }
  return buildRenxuanPositionContent(renxuanRunPosLabels(), body)
}

const hcwOpenPosLabels = computed(() =>
  hcwOpenPosIdxs.value
    .filter((i) => Number.isInteger(i) && i >= 0 && i < 5)
    .map((i) => SSC_POSITION_LABELS[i] ?? String(i)),
)

const hcwOpenPosHint = computed(() => {
  const need = renxuanRunPosNeed.value
  const open = hcwOpenPosLabels.value.join('/') || '所选开奖位'
  return `开奖选位须选 ${need} 个（当前${open}），下方显示这些位的历史开奖频次。`
})

const hcwBetPosHint = computed(() => {
  const need = renxuanRunPosNeed.value
  const bet = renxuanRunPosLabelsComputed.value.join('/') || '所选投注位'
  if (isRenxuanHcwZuDual.value) {
    const head = zuDualZoneHeadLabel()
    const minS = zuDualMinSinglesCount()
    const tail = triggerZuDualMeta.value?.tailLabel ?? '单号'
    const minH = zuDualMinHeadCount()
    return `投注选位至少选 ${need} 个（当前${bet}），下方${head}/${tail}池按所选位合并历史开奖频次更新；${head}≥${minH}、${tail}≥${minS}，凑不足 1 注则跳过当期。`
  }
  return `投注选位至少选 ${need} 个（当前${bet}），取开奖选位频次号码组合出票。`
})

// --- hot_cold_warm 冷热出号（v6 仅冷/热） ---
const hcwTotalPeriods = ref(20)
/** 旧配置进页合成 ranks 用；不落库、不驱动按钮高亮。 */
const hcwLegacyPickTypes = ref<SchemeHotColdPickType[]>([])
const hcwStrategy = ref<SchemeRotateStrategy>('keep')
const HCW_STRATEGY_OPTIONS: Array<{ label: string; value: SchemeRotateStrategy }> = [
  { label: '每期换', value: 'every' },
  { label: '不换号', value: 'keep' },
  { label: '中后换', value: 'after_hit' },
  { label: '挂后换', value: 'after_miss' },
]
/** 权威：每位勾选的名次（0=最热）。热/冷/全/清与点格都改这里；预览/回显亦以此为准。 */
const hcwRanks = ref<number[][]>([])
/** 预览：当前排名下名次映射出的号码（仅展示，不落库） */
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
/** 统计成功代数：开奖选位切换后强制双池格子重挂，避免第二池 DOM 残留旧频次 */
const hcwStatsGen = ref(0)

/**
 * 号码整体频次模式（组三/组六/组选复式/不定位/包胆）：单档选号池（跨位合并频次），
 * 区别于按位型（每位一档）。混合组选与直选复式同按位（千/百/十）。
 */
const hcwDigitOverall = computed(() => {
  // 组选12/4：双区池，勿走单档「号码池」
  if (isHcwZuDual.value) return false
  const cfg = schemePlayConfig.value as { betMode?: string; subPlayId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (bm === 'hunhe') return false
  const label = String(cfg.playMethodLabel ?? '')
  if (/一帆风顺|好事成双|三星报喜|四季发财/i.test(label)) return true
  if (
    ['zu3', 'zu6', 'zu24', 'zu60', 'zu30', 'zu120', 'budingwei', 'baodan', 'zuxuan_fs', 'zuxuan_ds'].includes(
      bm,
    )
  ) {
    return true
  }
  const sub = `${String(cfg.subPlayId ?? '')}`.toLowerCase()
  if (/zuxuan_fs|zu3|zu6|zu24|zu12|zu4|zu60|zu30|zu120|budingwei|baodan/.test(sub)) {
    if (/zu12/i.test(sub) && !/zu120/i.test(sub)) return false
    if (/zu4/i.test(sub) && !/zu24|zu12/i.test(sub)) return false
    return true
  }
  if (label.includes('单式') || label.includes('混合')) return false
  if (/组选12/.test(label) && !/组选120/.test(label)) return false
  if (/组选4/.test(label) && !/组选24|组选12/.test(label)) return false
  return /组三|组六|组选|不定位|包胆/.test(label)
})

/** 任意对碰冷热：同一份 01–49 排名拆为 A、B 两个名次池。 */
const isHcwRenyiDuipeng = computed(() => isLhcRenyiDuipengConfig(schemePlayConfig.value))

/**
 * 属性/聚合家族（大小单双/龙虎/庄闲/特殊号/和值/跨度）：单档「选项池」，
 * 分档频次由服务端复用权威判定计算（避免前端重复实现各彩种大小阈值/和值/跨度/龙虎口径）。
 */
/** 前二/后二/前三/后三大小单双：冷热按位（十/个…），每位大/小/单/双单选 */
const isHcwLhcGuoguan = computed(() => isLhcGuoguanConfig(schemePlayConfig.value))
const hcwPerPosDxds = computed(() => isPerPosDxdsPlayConfig(schemePlayConfig.value))

const hcwAttribute = computed(() => {
  const cfg = schemePlayConfig.value
  if (isHcwLhcGuoguan.value) return false
  const label = String(cfg.playMethodLabel ?? '')
  // 五星趣味：走号码池整体频次，勿当豹子/对子/顺子属性选项
  if (/一帆风顺|好事成双|三星报喜|四季发财/i.test(label)) return false
  // 按位大小单双：十/个分列，勿进单档「选项池」
  if (isPerPosDxdsPlayConfig(cfg)) return false
  // 特码/正特：号码+属性+波色 单档选项池（68 项）
  if (isLhcTemaPlayConfig(cfg)) return true
  // 生肖/尾数/生尾对碰：属性选项池
  if (isLhcSxDuipengConfig(cfg) || isLhcWsDuipengConfig(cfg) || isLhcSwDuipengConfig(cfg)) return true
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (['daxiao', 'danshuang', 'dxds', 'zhuangxian', 'longhu', 'longhuhe', 'longhubao', 'teshu', 'hezhi', 'kuadu', 'weishu', 'sx_dp', 'ws_dp', 'sw_dp'].includes(bm)) {
    return true
  }
  return /特殊号|大小单双|庄闲|龙虎豹|直选和值|组选和值|和值尾数|跨度/.test(label)
    || (label.includes('龙虎') && !label.includes('龙虎豹'))
    || (label === '和值' || (label.includes('和值') && !/单双|大小|尾数/.test(label)))
})

/** 单档布局（整体频次 或 属性选项池），区别于按位型每位一档 */
const hcwSingleGroup = computed(() => hcwDigitOverall.value || hcwAttribute.value)

/** 属性选项宇宙（服务端回填；无统计时用本地宇宙兜底，确保特殊号始终显示豹子/对子/顺子） */
const hcwAttrUniverse = ref<string[]>([])

/** 本地属性选项宇宙：特殊号→豹子/对子/顺子；和值等→数字池；特码→68 项；生肖对碰→12 肖；尾数对碰→0–9 */
function hcwLocalAttrUniverse(): string[] {
  const cfg = schemePlayConfig.value
  if (isLhcTemaPlayConfig(cfg)) return lhcTemaHcwUniverse()
  if (isLhcSxDuipengConfig(cfg)) return [...LHC_ZODIACS]
  if (isLhcWsDuipengConfig(cfg)) return [...LHC_TAIL_OPTIONS]
  if (isLhcSwDuipengConfig(cfg)) return [...LHC_SW_DUIPENG_OPTIONS]
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

/** 一码不定位等：名次条数不得超过上限（旧方案 / 热·冷·全快捷遗留） */
function clampHcwRanksToCap(): void {
  const cap = hcwPosPickCap()
  if (cap == null || cap <= 0) return
  for (let i = 0; i < hcwRanks.value.length; i++) {
    const row = hcwRanks.value[i]
    if (row && row.length > cap) hcwRanks.value[i] = row.slice(0, cap)
  }
}

function ensureHcwPools(): void {
  const n = hcwDimCount()
  let pools = hcwPools.value
  let ranks = hcwRanks.value
  let changed = false
  if (pools.length < n) {
    pools = [...pools]
    while (pools.length < n) pools.push([])
    changed = true
  }
  if (ranks.length < n) {
    ranks = [...ranks]
    while (ranks.length < n) ranks.push([])
    changed = true
  }
  if (pools.length > n) {
    pools = pools.slice(0, n)
    changed = true
  }
  if (ranks.length > n) {
    ranks = ranks.slice(0, n)
    changed = true
  }
  if (changed) {
    hcwPools.value = pools
    hcwRanks.value = ranks
  }
  clampHcwRanksToCap()
}

/** 冷热分档行数：组选12=二重+单号；直选单式=开奖选位数；属性/整体=1；其它=玩法位数 */
function hcwDimCount(): number {
  if (isHcwLhcGuoguan.value) return 6
  if (isHcwRenyiDuipeng.value) return 2
  if (isHcwZuDual.value) return 2
  if (hcwSingleGroup.value) return 1
  if (isRenxuanHcwDualPosPlay.value) {
    const n = hcwOpenPosIdxs.value.filter((i) => Number.isInteger(i) && i >= 0 && i < 5).length
    return Math.max(1, n)
  }
  return positionCount.value
}

/** 冷热分档分组：属性=选项池；整体=号码池；组选12=二重/单号；直选单式=开奖选位名；按位=每位一档 */
const hcwGroupLabels = computed(() => {
  if (isHcwLhcGuoguan.value) return [...LHC_GUOGUAN_POSITION_LABELS]
  if (isHcwRenyiDuipeng.value) return ['A区', 'B区']
  if (hcwAttribute.value) return ['选项池']
  if (isHcwZuDual.value) return [...triggerZuDualZoneLabels.value]
  if (hcwDigitOverall.value) return ['号码池']
  if (isRenxuanHcwDualPosPlay.value) return hcwOpenPosLabels.value
  return positionLabels.value
})

/** 组选12/4 冷热：拼双区供计注/占位 */
function hcwZuDualPicks(): string {
  const head = uniqueDigitsFromRun((hcwPools.value[0] ?? []).join('')).join('')
  const singles = uniqueDigitsFromRun((hcwPools.value[1] ?? []).join('')).join('')
  if (!head && !singles) return ''
  return `${head},${singles}`
}

/** 无统计时的兜底可选项：属性优先本地宇宙（豹子/对子/顺子），再回退服务端回填 */
const hcwFallbackOptions = computed(() => {
  if (isHcwLhcGuoguan.value) return [...LHC_GUOGUAN_OPTIONS]
  if (hcwPerPosDxds.value) return [...PER_POS_DXDS_OPTIONS]
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
  // pickTypes/pool 仅旧配置合成 ranks；有 ranks 时忽略
  if (Array.isArray(c.pickTypes)) {
    hcwLegacyPickTypes.value = c.pickTypes
      .map((t) => String(t ?? '').toLowerCase())
      .filter((t): t is SchemeHotColdPickType => t === 'hot' || t === 'cold')
  } else {
    hcwLegacyPickTypes.value = []
  }
  const st = String(c.strategy ?? '')
  if (st === 'every' || st === 'keep' || st === 'after_hit' || st === 'after_miss') {
    hcwStrategy.value = st
  } else if (typeof c.winRotate === 'boolean') {
    hcwStrategy.value = c.winRotate ? 'after_hit' : 'keep'
  }
  if (Array.isArray(c.positionIdxs)) {
    renxuanRunPosIdxs.value = c.positionIdxs
      .map((x) => Math.trunc(Number(x)))
      .filter((n) => Number.isInteger(n) && n >= 0 && n < 5)
    ensureRenxuanRunPositions()
  }
  if (Array.isArray(c.openPositionIdxs)) {
    hcwOpenPosIdxs.value = c.openPositionIdxs
      .map((x) => Math.trunc(Number(x)))
      .filter((n) => Number.isInteger(n) && n >= 0 && n < 5)
  }
  ensureHcwOpenPositions()
  if (Array.isArray(c.ranks)) {
    hcwRanks.value = c.ranks.map((row) => {
      if (!Array.isArray(row)) return []
      const seen = new Set<number>()
      const out: number[] = []
      for (const item of row) {
        const n = Math.trunc(Number(item))
        if (!Number.isFinite(n) || n < 0 || seen.has(n)) continue
        seen.add(n)
        out.push(n)
      }
      return out
    })
  } else {
    hcwRanks.value = []
  }
  if (Array.isArray(c.pool) && !hcwRanks.value.some((r) => (r?.length ?? 0) > 0)) {
    // 无 ranks 的旧配置：用 pool 反查名次
    hcwPools.value = c.pool.map((line) =>
      String(line ?? '')
        .split(/[,，\s]+/)
        .map((s) => s.trim())
        .filter((s) => s !== ''),
    )
  } else {
    hcwPools.value = []
  }
  ensureHcwPools()
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
  const typeId = String(schemePlayConfig.value.playTypeId ?? '').toLowerCase()
  // 四星/后四：千百十个（偏移 1）；勿落万位起算
  if (typeId === 'sixing' || typeId === 'g013' || typeId === 'hou4') {
    return ballsLen >= 5 ? 1 : 0
  }
  if (typeId.startsWith('hou')) return ballsLen - segLen
  if (typeId.startsWith('zhong')) return Math.floor((ballsLen - segLen) / 2)
  return 0
}

/** 定点星段冷热合并位（四星=千百十个）；优先 segmentLabels 映射到万千百十个下标 */
function hcwFixedSegmentBallIdxs(): number[] {
  const labels = schemePlayConfig.value.segmentLabels ?? []
  const idxs = labels
    .map((l) => SSC_POSITION_LABELS.indexOf(l as (typeof SSC_POSITION_LABELS)[number]))
    .filter((i) => i >= 0)
  if (idxs.length > 0) return idxs
  const n = Math.max(1, positionCount.value)
  const offset = hcwPositionOffset(5)
  return Array.from({ length: n }, (_, i) => offset + i)
}

/** 冷热统计请求代数：玩法异步回填时 watch 会连触发，只采纳最后一次结果，避免中途 return 卡住「—」 */
let hcwLoadSeq = 0
/** 上次成功发起统计的玩法键；用于方案模式返回时去重，避免清空后竞态失败留下空频次 */
let lastHcwStatsKey = ''

function hcwStatsKey(): string {
  const cfg = schemePlayConfig.value
  const family = isHcwLhcGuoguan.value
    ? 'lhc_guoguan'
    : hcwPerPosDxds.value
      ? 'dxds_pos'
      : hcwAttribute.value
      ? 'attr'
      : isHcwZuDual.value
        ? 'zu12'
        : hcwDigitOverall.value
          ? 'overall'
          : 'pos'
  const openKey = showHcwOpenPosition.value ? hcwOpenPosIdxs.value.join(',') : ''
  // 任选和值/尾数/组选复式/组选12：频次随投注选位变化；四星组选12 用玩法位
  const betPosKey =
    schemeUsesRenxuanRunPos.value &&
    (hcwAttribute.value || hcwDigitOverall.value || isHcwZuDual.value)
      ? renxuanRunPosIdxs.value.join(',')
      : isHcwZuDual.value
        ? hcwFixedSegmentBallIdxs().join(',')
        : ''
  return [
    lotteryCode.value.trim(),
    cfg.playTypeId ?? '',
    cfg.catalogSubId ?? '',
    cfg.subPlayId ?? '',
    cfg.betMode ?? '',
    cfg.playMethodLabel ?? '',
    String(hcwTotalPeriods.value),
    family,
    openKey,
    betPosKey,
  ].join('|')
}

function hcwPlayReadyForStats(): boolean {
  if (runTypeId.value !== 'hot_cold_warm') return false
  if (!lotteryCode.value.trim()) return false
  const cfg = schemePlayConfig.value
  return Boolean(String(cfg.playTypeId ?? '').trim())
}

/** 拉取/刷新冷热频次。force=true 忽略同玩法去重（刷新按钮）。 */
function scheduleHcwStats(force = false): void {
  if (!hcwPlayReadyForStats()) return
  ensureHcwPools()
  const key = hcwStatsKey()
  if (!force && key === lastHcwStatsKey && (hcwStatsReady.value || hcwLoading.value)) return
  const familyChanged =
    Boolean(lastHcwStatsKey) && lastHcwStatsKey.split('|').pop() !== key.split('|').pop()
  lastHcwStatsKey = key
  // 仅玩法家族切换时清空；同玩法重拉保留旧频次，避免从方案模式返回时先闪成「—」
  if (familyChanged || force) {
    if (familyChanged) {
      hcwStatsReady.value = false
      hcwAttrUniverse.value = []
      hcwFreq.value = []
      hcwTiers.value = []
    }
  }
  void loadHcwStats()
}

/** 属性家族分档：调用服务端接口（复用权威 evaluatePlayHit 计频），单档选项池 */
async function loadHcwAttrStats(seq: number): Promise<void> {
  const cfg = schemePlayConfig.value
  const localUni = hcwLocalAttrUniverse()
  // 先铺本地宇宙，避免接口失败/延迟时选项池空白（前三特殊号须始终可见豹子/对子/顺子）
  if (localUni.length && !hcwAttrUniverse.value.length) hcwAttrUniverse.value = localUni
  // 特殊号/和值/跨度/尾数：playConfig.segmentLen=1 仅表示单档选项池，不是开奖截取长度。
  // 勿传 segmentLen，否则后端若覆盖 resolve 的 3 位，跨度恒为 0、次数全堆在「0」。
  const bm = String(cfg.betMode || (localUni.includes('豹子') ? 'teshu' : '')).toLowerCase()
  if (schemeUsesRenxuanRunPos.value) ensureRenxuanRunPositions()
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
    ...(schemeUsesRenxuanRunPos.value
      ? { positionIdxs: [...renxuanRunPosIdxs.value] }
      : {}),
  })
  if (seq !== hcwLoadSeq) return
  if (res.mode !== 'attribute' || !Array.isArray(res.universe) || res.universe.length === 0) {
    // 接口未识别：若已有频次则保留（方案模式往返时的瞬时失败勿清掉）
    if (hcwStatsReady.value && hcwFreq.value.length) return
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
  hcwStatsGen.value += 1
  refreshHcwEstimatePools()
}

/** 按位大小单双：每位按该位球号计大/小/单/双频次 */
async function loadHcwPerPosDxdsStats(seq: number): Promise<void> {
  const guoguan = isHcwLhcGuoguan.value
  const uni = guoguan ? [...LHC_GUOGUAN_OPTIONS] : [...PER_POS_DXDS_OPTIONS]
  const ballIdxs = guoguan ? [0, 1, 2, 3, 4, 5] : hcwFixedSegmentBallIdxs()
  const dims = guoguan ? 6 : Math.max(1, ballIdxs.length || positionCount.value)
  const res = await fetchGameDraws(lotteryCode.value, undefined, hcwTotalPeriods.value)
  if (seq !== hcwLoadSeq) return
  const items = Array.isArray(res?.items) ? res.items : []
  const freq: Array<Record<string, number>> = Array.from({ length: dims }, () => {
    const row: Record<string, number> = {}
    for (const opt of uni) row[opt] = 0
    return row
  })
  let counted = 0
  for (const it of items) {
    const balls = Array.isArray(it?.balls) ? it.balls : []
    if (!balls.length) continue
    for (let p = 0; p < dims; p++) {
      const ballIdx = ballIdxs[p] ?? p
      const n = Number(balls[ballIdx])
      if (!Number.isFinite(n)) continue
      for (const opt of guoguan ? lhcGuoguanAttrsForNumber(n) : sscDigitDxdsAttrs(n)) {
        freq[p]![opt] = (freq[p]![opt] ?? 0) + 1
        counted += 1
      }
    }
  }
  if (!counted) {
    if (hcwStatsReady.value && hcwFreq.value.length) return
    hcwStatsReady.value = false
    hcwTiers.value = Array.from({ length: dims }, () => ({
      hot: [] as string[],
      warm: [] as string[],
      cold: [...uni],
    }))
    hcwFreq.value = []
    return
  }
  const half = Math.ceil(uni.length / 2)
  hcwTiers.value = freq.map((counts) => {
    const sorted = [...uni].sort((a, b) => {
      const diff = (counts[b] ?? 0) - (counts[a] ?? 0)
      return diff !== 0 ? diff : uni.indexOf(a) - uni.indexOf(b)
    })
    return { hot: sorted.slice(0, half), warm: [], cold: sorted.slice(half) }
  })
  hcwFreq.value = freq.map((counts) => ({ ...counts }))
  hcwStatsReady.value = true
  hcwStatsGen.value += 1
  refreshHcwEstimatePools()
}

async function loadHcwStats(): Promise<void> {
  const seq = ++hcwLoadSeq
  hcwLoading.value = true
  try {
    if (hcwAttribute.value) {
      await loadHcwAttrStats(seq)
      return
    }
    if (hcwPerPosDxds.value || isHcwLhcGuoguan.value) {
      await loadHcwPerPosDxdsStats(seq)
      return
    }
    const res = await fetchGameDraws(lotteryCode.value, undefined, hcwTotalPeriods.value)
    if (seq !== hcwLoadSeq) return
    const items = Array.isArray(res?.items) ? res.items : []
    const pool = numberPoolTokens.value
    // 任意对碰：七个开奖球合并计频，A/B 两区共享完全相同的冷热排序。
    if (isHcwRenyiDuipeng.value) {
      const counts: Record<string, number> = {}
      let counted = 0
      for (const it of items) {
        const balls = Array.isArray(it?.balls) ? it.balls : []
        for (const ball of balls) {
          const token = normalizePoolToken(String(ball ?? ''))
          if (!token) continue
          counts[token] = (counts[token] ?? 0) + 1
          counted += 1
        }
      }
      if (!counted) {
        if (hcwStatsReady.value && hcwFreq.value.length) return
        hcwStatsReady.value = false
        hcwFreq.value = []
        return
      }
      const sorted = [...pool].sort((a, b) => {
        const diff = (counts[b] ?? 0) - (counts[a] ?? 0)
        return diff !== 0 ? diff : Number(a) - Number(b)
      })
      const half = Math.ceil(pool.length / 2)
      const mkTier = (): HcwTier => ({ hot: sorted.slice(0, half), warm: [], cold: sorted.slice(half) })
      hcwTiers.value = [mkTier(), mkTier()]
      hcwFreq.value = [{ ...counts }, { ...counts }]
      hcwStatsReady.value = true
      hcwStatsGen.value += 1
      refreshHcwEstimatePools()
      return
    }
    // 组选12：任选按投注选位、四星等按玩法位合并计频；二重/单号两池共用同一排序
    if (isHcwZuDual.value) {
      const betIdxs = schemeUsesRenxuanRunPos.value
        ? renxuanRunPosIdxs.value.filter((i) => Number.isInteger(i) && i >= 0 && i < 5)
        : hcwFixedSegmentBallIdxs()
      const counts: Record<string, number> = {}
      let counted = 0
      for (const it of items) {
        const balls = Array.isArray(it?.balls) ? it.balls : []
        if (!balls.length) continue
        for (const ballIdx of betIdxs) {
          const tk = normalizePoolToken(String(balls[ballIdx] ?? ''))
          if (tk) {
            counts[tk] = (counts[tk] ?? 0) + 1
            counted += 1
          }
        }
      }
      if (!counted) {
        if (hcwStatsReady.value && hcwFreq.value.length) return
        hcwStatsReady.value = false
        hcwFreq.value = []
        return
      }
      const half = Math.ceil(pool.length / 2)
      const sorted = [...pool].sort((a, b) => {
        const diff = (counts[b] ?? 0) - (counts[a] ?? 0)
        return diff !== 0 ? diff : Number(a) - Number(b)
      })
      // 二重/单号两池必须各持独立分档与频次对象，避免共享引用导致只刷新第一池
      const mkTier = (): HcwTier => ({
        hot: sorted.slice(0, half),
        warm: [],
        cold: sorted.slice(half),
      })
      hcwTiers.value = [mkTier(), mkTier()]
      hcwFreq.value = [{ ...counts }, { ...counts }]
      hcwStatsReady.value = true
      hcwStatsGen.value += 1
      refreshHcwEstimatePools()
      return
    }
    // 任选直选单式：按当前开奖选位绝对下标计频（勿 ensure 补默认万位）
    const openIdxs = isRenxuanHcwDualPosPlay.value
      ? hcwOpenPosIdxs.value.filter((i) => Number.isInteger(i) && i >= 0 && i < 5)
      : []
    // 任选组选复式等整体号池：按投注选位绝对下标合并计频
    const betIdxs =
      hcwDigitOverall.value && schemeUsesRenxuanRunPos.value
        ? renxuanRunPosIdxs.value.filter((i) => Number.isInteger(i) && i >= 0 && i < 5)
        : []
    const segLen = openIdxs.length > 0 ? openIdxs.length : positionCount.value
    const dims = hcwDigitOverall.value ? 1 : segLen
    const freq: Array<Record<string, number>> = Array.from({ length: dims }, () => ({}))
    let counted = 0
    for (const it of items) {
      const balls = Array.isArray(it?.balls) ? it.balls : []
      if (!balls.length) continue
      if (openIdxs.length > 0) {
        for (let p = 0; p < openIdxs.length; p++) {
          const ballIdx = openIdxs[p]!
          const tk = normalizePoolToken(String(balls[ballIdx] ?? ''))
          if (tk) {
            freq[p]![tk] = (freq[p]![tk] ?? 0) + 1
            counted += 1
          }
        }
        continue
      }
      if (betIdxs.length > 0) {
        for (const ballIdx of betIdxs) {
          const tk = normalizePoolToken(String(balls[ballIdx] ?? ''))
          if (tk) {
            freq[0]![tk] = (freq[0]![tk] ?? 0) + 1
            counted += 1
          }
        }
        continue
      }
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
      if (hcwStatsReady.value && hcwFreq.value.length) return
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
    hcwStatsGen.value += 1
    refreshHcwEstimatePools()
  } catch {
    if (seq !== hcwLoadSeq) return
    // 刷新失败保留旧频次，避免方案模式返回后选项池次数被清空
    if (hcwStatsReady.value && hcwFreq.value.length) return
    hcwStatsReady.value = false
    hcwFreq.value = []
  } finally {
    if (seq === hcwLoadSeq) hcwLoading.value = false
  }
}

/** 当前位「最热→最冷」全序（与后端 hotColdWarmTiers 一致） */
function hcwOrderedTokens(pos: number): string[] {
  if (hcwStatsReady.value) {
    const t = hcwTiers.value[pos]
    const seen: string[] = []
    for (const d of [...(t?.hot ?? []), ...(t?.cold ?? [])]) {
      if (!seen.some((x) => tokenEq(x, d))) seen.push(d)
    }
    if (seen.length) return seen
  }
  return [...hcwFallbackOptions.value]
}

/** 无 ranks 的旧配置：用 pickTypes/pool 合成名次（仅进页一次；合成后以 ranks 为准） */
function synthesizeHcwRanksFromLegacy(): void {
  ensureHcwPools()
  if (hcwRanks.value.some((r) => (r?.length ?? 0) > 0)) return
  const n = Math.max(hcwGroupLabels.value.length, hcwPools.value.length, 1)
  const kinds = hcwLegacyPickTypes.value
  const wantHot = !kinds.length || kinds.includes('hot')
  const wantCold = kinds.includes('cold')
  const anyPool = hcwPools.value.some((p) => (p?.length ?? 0) > 0)
  for (let pi = 0; pi < n; pi++) {
    const enabled = !anyPool || (hcwPools.value[pi]?.length ?? 0) > 0
    if (!enabled) {
      hcwRanks.value[pi] = []
      continue
    }
    const ordered = hcwOrderedTokens(pi)
    if (!ordered.length) continue
    // 旧 pool 有具体号：尽量反查为名次
    const pool = hcwPools.value[pi] ?? []
    if (pool.length && hcwStatsReady.value) {
      const ranks: number[] = []
      for (const tok of pool) {
        const idx = ordered.findIndex((x) => tokenEq(x, tok))
        if (idx >= 0 && !ranks.includes(idx)) ranks.push(idx)
      }
      if (ranks.length) {
        hcwRanks.value[pi] = ranks
        continue
      }
    }
    const half = Math.ceil(ordered.length / 2)
    const ranks: number[] = []
    if (wantHot) for (let i = 0; i < half; i++) ranks.push(i)
    if (wantCold) for (let i = half; i < ordered.length; i++) ranks.push(i)
    hcwRanks.value[pi] = ranks
  }
  clampHcwRanksToCap()
}

/** 统计就绪后：名次 → 当前排名号码（预览高亮） */
function refreshHcwEstimatePools(): void {
  if (!hcwStatsReady.value) return
  ensureHcwPools()
  synthesizeHcwRanksFromLegacy()
  clampHcwRanksToCap()
  const n = Math.max(hcwGroupLabels.value.length, hcwRanks.value.length, 1)
  // 整表替换：下标赋值时第二池预览偶发不触发视图更新（组选12 二重/单号）
  const nextPools: string[][] = Array.from({ length: n }, () => [])
  for (let pi = 0; pi < n; pi++) {
    const ranks = hcwRanks.value[pi] ?? []
    if (!ranks.length) continue
    const ordered = hcwOrderedTokens(pi)
    const picked: string[] = []
    for (const r of ranks) {
      if (r >= 0 && r < ordered.length) picked.push(ordered[r]!)
    }
    const cap = hcwPosPickCap()
    nextPools[pi] = sortHcwTokens(
      cap != null && picked.length > cap ? picked.slice(0, cap) : picked,
    )
  }
  hcwPools.value = nextPools
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
  const cfg = schemePlayConfig.value
  if (isHcwLhcGuoguan.value) return 1
  // 后二大小单双等：每位仅 1 个大/小/单/双
  if (hcwPerPosDxds.value) return 1
  // 五星和值单双/大小、包胆等：走 poolMaxPicks（优先于属性池「不设上限」）
  const pickCap = poolMaxPicksForConfig(cfg)
  if (pickCap != null && pickCap > 0) return pickCap
  // 属性选项池（龙虎/特殊号等）不设号池勾选上限
  if (hcwAttribute.value) return null
  if (isYixingDingweiPlayConfig(cfg)) return YIXING_MAX_PICKS_PER_POS
  return null
}

function hcwPosPickCapMsg(): string {
  const cfg = schemePlayConfig.value
  if (isHcwLhcGuoguan.value) return '过关：每个正码位置只能选择一个选项'
  if (hcwPerPosDxds.value) return '仅能选择一个选项（大/小/单/双）'
  if (isWuxingSumDxdsPlayConfig(cfg)) {
    return /和值大小|尾数大小/.test(cfg.playMethodLabel ?? '') || cfg.betMode === 'daxiao'
      ? '仅能选择一个选项（大/小）'
      : '仅能选择一个选项（单/双）'
  }
  if (cfg.betMode === 'baodan' || /包胆/.test(cfg.playMethodLabel ?? '')) {
    return '包胆：只能选择一个 0–9 的号码'
  }
  if (isLhcSxDuipengConfig(cfg)) {
    return `生肖对碰：请选择 ${LHC_SX_DUIPENG_MIN_PICKS} 个生肖`
  }
  if (isLhcWsDuipengConfig(cfg)) {
    return `尾数对碰：请选择 ${LHC_WS_DUIPENG_MIN_PICKS} 个尾数`
  }
  if (isLhcSwDuipengConfig(cfg)) {
    return '生尾对碰：请各选择 1 个生肖和 1 个尾数'
  }
  const pickCap = poolMaxPicksForConfig(cfg)
  if (pickCap === 2 && (cfg.betMode === 'budingwei' || /不定位/.test(cfg.playMethodLabel ?? ''))) {
    return '一码不定位：最多选择 2 个号码'
  }
  return YIXING_MAX_PICKS_MSG
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
    hcwRanks.value[pos] = []
    hcwPools.value[pos] = []
    return
  }
  const ordered = hcwOrderedTokens(pos)
  if (!ordered.length) return
  const half = Math.ceil(ordered.length / 2)
  let ranks: number[] = []
  if (kind === 'hot') {
    ranks = Array.from({ length: half }, (_, i) => i)
  } else if (kind === 'cold') {
    ranks = Array.from({ length: Math.max(0, ordered.length - half) }, (_, i) => half + i)
  } else {
    ranks = Array.from({ length: ordered.length }, (_, i) => i)
  }
  const cap = hcwPosPickCap()
  // 一码不定位等有固定上限：热/冷/全快捷直接截到上限，不弹「提交类」提示
  if (cap != null && ranks.length > cap) {
    ranks = ranks.slice(0, cap)
  }
  if (isHcwRenyiDuipeng.value) {
    hcwRanks.value = replaceLhcRenyiDuipengHotColdRanks(
      hcwRanks.value,
      pos as 0 | 1,
      ranks,
      ordered.length,
    )
  } else {
    hcwRanks.value[pos] = ranks
  }
  // 仅改当前位；其它位的冷/热/全保持各自选择
  refreshHcwEstimatePools()
}

/** 点击号码 = 切换该号在当前排名中的名次 */
function toggleHcwDigit(pos: number, digit: string): void {
  ensureHcwPools()
  const ordered = hcwOrderedTokens(pos)
  const rank = ordered.findIndex((x) => tokenEq(x, digit))
  if (rank < 0) return
  const ranks = [...(hcwRanks.value[pos] ?? [])]
  const at = ranks.indexOf(rank)
  if (at >= 0) {
    ranks.splice(at, 1)
  } else {
    if (isHcwRenyiDuipeng.value) {
      const other = hcwRanks.value[pos === 0 ? 1 : 0] ?? []
      if (other.includes(rank)) {
        ElMessage.warning('任意对碰：A区与B区号码不可重复')
        return
      }
      if (ranks.length + other.length >= 10) {
        ElMessage.warning('任意对碰：A区和B区合计最多选择 10 个号码')
        return
      }
    }
    const cap = hcwPosPickCap()
    if (
      cap === 1 &&
      (isHcwLhcGuoguan.value || isTailParityPlayConfig(schemePlayConfig.value))
    ) {
      ranks.splice(0, ranks.length, ...toggleSingleHcwRank(ranks, rank))
    } else if (cap != null && ranks.length >= cap) {
      ElMessage.warning(hcwPosPickCapMsg())
      return
    } else {
      ranks.push(rank)
    }
  }
  if (isHcwRenyiDuipeng.value) {
    hcwRanks.value = replaceLhcRenyiDuipengHotColdRanks(
      hcwRanks.value,
      pos as 0 | 1,
      ranks,
      ordered.length,
    )
  } else {
    hcwRanks.value[pos] = ranks
  }
  // 预览映射（统计未就绪时用兜底序）
  const picked = ranks
    .filter((r) => r >= 0 && r < ordered.length)
    .map((r) => ordered[r]!)
  if (isHcwRenyiDuipeng.value) {
    refreshHcwEstimatePools()
  } else {
    hcwPools.value[pos] = sortHcwTokens(picked)
  }
}

/** 快捷钮高亮：当前名次集与该快捷目标完全一致 */
function hcwQuickActive(pos: number, kind: 'cold' | 'hot' | 'all'): boolean {
  const ordered = hcwOrderedTokens(pos)
  if (!ordered.length) return false
  const half = Math.ceil(ordered.length / 2)
  let want: number[] = []
  if (kind === 'hot') want = Array.from({ length: half }, (_, i) => i)
  else if (kind === 'cold') want = Array.from({ length: Math.max(0, ordered.length - half) }, (_, i) => half + i)
  else want = Array.from({ length: ordered.length }, (_, i) => i)
  const cap = hcwPosPickCap()
  if (cap != null && want.length > cap) want = want.slice(0, cap)
  const got = [...(hcwRanks.value[pos] ?? [])].sort((a, b) => a - b)
  const w = [...want].sort((a, b) => a - b)
  if (got.length !== w.length) return false
  return w.every((r, i) => r === got[i])
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

/** 每位展示：0–9（或选项宇宙）升序；下方带频次；热/冷着色；高亮为预估下注号 */
function hcwDisplayCells(pos: number): Array<{ token: string; count: number | null; tier: 'hot' | 'cold' | 'none' }> {
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

const hcwCellsByPos = computed(() => {
  // 显式依赖：选位/频次/分档，保证二重与单号两池同时重算
  void hcwStatsGen.value
  void hcwOpenPosIdxs.value
  void renxuanRunPosIdxs.value
  void hcwFreq.value
  void hcwTiers.value
  void hcwStatsReady.value
  return hcwGroupLabels.value.map((_, pi) => hcwDisplayCells(pi))
})

/** 预估注数：与随机出号一致，走 countBetUnits（位积/组合×段长/任选 C(n,k)） */
const hcwEstimatedUnits = computed(() => {
  // 属性/聚合家族（和值/跨度/大小单双/龙虎等）：统一走 countBetUnits
  // 和值/跨度按组合数×段倍乘（前中后三组选和值 2,6,13,17,24 → 38×3=114），勿按「选几个算几注」
  let picks = ''
  if (isHcwLhcGuoguan.value) {
    const content = Array.from({ length: 6 }, (_, i) => (hcwPools.value[i] ?? [])[0] ?? '').join(',')
    return countBetUnits(schemePlayConfig.value, content)
  }
  if (hcwAttribute.value) {
    picks = (hcwPools.value[0] ?? []).filter((t) => t.trim() !== '').join(',')
  } else if (isHcwZuDual.value) {
    picks = hcwZuDualPicks()
  } else if (hcwDigitOverall.value) {
    picks = (hcwPools.value[0] ?? []).join(',')
  } else {
    const n = hcwDimCount()
    if (n <= 0) return 0
    const lines = Array.from({ length: n }, (_, i) => (hcwPools.value[i] ?? []).join(','))
    if (lines.every((x) => !x.trim())) return 0
    picks = lines.join('\n')
  }
  if (!picks.trim()) return 0
  return countBetUnits(schemePlayConfig.value, wrapRenxuanRunContent(picks))
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

/** 组选单式/任选混合组选：整注随机（仅需注数 rdCounts[0]）。直选单式/中三混合组选：千/百/十按位产号。 */
const rdWholeTicket = computed(() => {
  // 直选单式（段长≥2）：千/百/十…各位配数量，与直选复式同布局
  if (isZhixuanDanshiPerPosPlay(schemePlayConfig.value)) return false
  const cfg = schemePlayConfig.value as { betMode?: string; subPlayId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  const sub = String(cfg.subPlayId ?? '').toLowerCase()
  const label = String(cfg.playMethodLabel ?? '')
  // 中三/前三混合组选按位；任选混合组选整注（如 012,345）
  if (bm === 'hunhe' || label.includes('混合')) {
    return isRenxuanNeedsPositionConfig(schemePlayConfig.value)
  }
  if (['danshi', 'zhixuan_ds', 'zuxuan_ds'].includes(bm)) return true
  if (['zhixuan_ds', 'zuxuan_ds'].includes(sub)) return true
  return label.includes('单式')
})

/** 单式整注随机的本地预览注单 */
const rdWholePreview = ref<string[]>([])

/** 组选12/4 随机：头区/单号两个选码个数（勿走扁选「选码个数」） */
const isRdZuDual = computed(() => isZuDualPlayConfig(schemePlayConfig.value) && !rdWholeTicket.value)
/** 任选二全中任意对碰随机：A/B 两区各自配置随机个数。 */
const isRdLhcRenyiDuipeng = computed(
  () =>
    isLhcRenyiDuipengConfig(schemePlayConfig.value) ||
    isRandomDrawLhcRenyiDuipengConfig(schemePlayConfig.value),
)
const rdLhcRenyiDuipengCounts = computed(() => normalizeLhcRenyiDuipengRandomCounts(rdCounts.value))
const rdLhcRenyiDuipengAMax = computed(() => 10 - rdLhcRenyiDuipengCounts.value[1])
const rdLhcRenyiDuipengBMax = computed(() => 10 - rdLhcRenyiDuipengCounts.value[0])

function ensureRdCounts(): void {
  if (isRdLhcRenyiDuipeng.value) {
    const counts = normalizeLhcRenyiDuipengRandomCounts(rdCounts.value)
    if (rdCounts.value.length !== 2 || rdCounts.value[0] !== counts[0] || rdCounts.value[1] !== counts[1]) {
      rdCounts.value = counts
    }
    return
  }
  // 双区组选：counts=[头区个数, 尾区个数]
  if (isRdZuDual.value) {
    const minH = zuDualMinHeadCount()
    const minS = zuDualMinSinglesCount()
    let d = Math.min(10, Math.max(minH, Math.trunc(Number(rdCounts.value[0]) || minH)))
    let s = Math.min(10, Math.max(minS, Math.trunc(Number(rdCounts.value[1]) || minS)))
    const meta = triggerZuDualMeta.value
    if (meta?.equalCounts) {
      const n = Math.max(d, s, minH, minS)
      d = n
      s = n
    }
    if (rdCounts.value.length !== 2 || rdCounts.value[0] !== d || rdCounts.value[1] !== s) {
      rdCounts.value = [d, s]
    }
    return
  }
  const n = positionCount.value
  while (rdCounts.value.length < n) rdCounts.value.push(1)
  if (rdCounts.value.length === 0) rdCounts.value.push(1)
}

/** 组三/组六/组选N/组选复式：号码池随机（选 K 个号），非按位、非整注。包胆/和值属属性单选，勿因文案含「组选」误入。 */
const rdZuxuanPool = computed(() => {
  if (rdWholeTicket.value || isRdZuDual.value) return false
  const cfg = schemePlayConfig.value as { betMode?: string; subPlayId?: string; catalogSubId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  const label = String(cfg.playMethodLabel ?? '')
  // 混合组选已按位分列，勿因文案含「组选」进单档号码池
  if (bm === 'hunhe' || label.includes('混合')) return false
  if (bm === 'baodan' || /包胆/.test(label)) return false
  // 组选和值/直选和值/跨度：走属性「选项个数」，下限 1（勿被「组选」文案抬成 ≥2/≥3）
  if (['hezhi', 'kuadu', 'weishu', 'budingwei'].includes(bm) || /和值|跨度|尾数/.test(label)) return false
  if (['zu3', 'zu6', 'zu24', 'zu12', 'zu60', 'zu30', 'zu120'].includes(bm)) return true
  const cat = `${String(cfg.subPlayId ?? '')} ${String(cfg.catalogSubId ?? '')}`.toLowerCase()
  if (/baodan|_bd\b|包胆|hezhi|_hz\b|kuadu|和值|跨度/.test(`${cat} ${label}`)) return false
  if (/zuxuan_fs|zuxuan|zu3|zu6|zu24|zu12|zu60|zu30|zu120/.test(cat)) return true
  return /组三|组六|组选/.test(label)
})

/** 属性/聚合家族（大小单双/龙虎/特殊号/庄闲/和值/跨度/不定位/包胆/特码）：从选项宇宙随机抽 K 个 */
const rdAttribute = computed(() => {
  if (rdWholeTicket.value || rdZuxuanPool.value || isRdZuDual.value) return false
  if (isLhcGuoguanConfig(schemePlayConfig.value)) return true
  // 前二/后二/前三/后三大小单双：按位（十/个…），不走单档「选项个数」
  if (isPerPosDxdsPlayConfig(schemePlayConfig.value)) return false
  // 特码/正特：01–49 + 属性 + 波色，单档「选项个数」上限 68
  if (isLhcTemaPlayConfig(schemePlayConfig.value)) return true
  // 生肖/尾数/生尾对碰：恰好 2 个属性
  if (
    isLhcSxDuipengConfig(schemePlayConfig.value) ||
    isLhcWsDuipengConfig(schemePlayConfig.value) ||
    isLhcSwDuipengConfig(schemePlayConfig.value)
  ) {
    return true
  }
  const bm = String(schemePlayConfig.value.betMode ?? '').toLowerCase()
  return ['daxiao', 'danshuang', 'dxds', 'zhuangxian', 'longhu', 'longhuhe', 'longhubao', 'teshu', 'hezhi', 'kuadu', 'weishu', 'budingwei', 'baodan', 'tema', 'zhengte', 'sx_dp', 'ws_dp', 'sw_dp'].includes(bm)
})

/** 随机出号是否用"单一数量输入"（单式注数 / 组选选码个数 / 属性选项个数） */
const rdSingleCountMode = computed(
  () => !isRdZuDual.value && (rdWholeTicket.value || rdZuxuanPool.value || rdAttribute.value),
)
const rdSingleCountLabel = computed(() => {
  if (rdWholeTicket.value) return '注数'
  if (rdZuxuanPool.value) return '选码个数'
  if (isLhcGuoguanConfig(schemePlayConfig.value)) return '随机正码位置数'
  return '选项个数'
})

/** 属性/聚合玩法选项宇宙（特殊号=豹子/对子/顺子，大小单双=大/小/单/双，和值=号池等） */
function rdAttributeUniverse(): string[] {
  const cfg = schemePlayConfig.value
  if (isLhcGuoguanConfig(cfg)) return [...LHC_GUOGUAN_OPTIONS]
  if (isLhcTemaPlayConfig(cfg)) return lhcTemaHcwUniverse()
  if (isLhcSxDuipengConfig(cfg)) return [...LHC_ZODIACS]
  if (isLhcWsDuipengConfig(cfg)) return [...LHC_TAIL_OPTIONS]
  if (isLhcSwDuipengConfig(cfg)) return [...LHC_SW_DUIPENG_OPTIONS]
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
  const cfg = schemePlayConfig.value
  if (isLhcGuoguanConfig(cfg)) return 6
  // 特码/正特：01–49+属性+波色=68（勿被号池 49 / 按位默认 10 夹住）
  if (isLhcTemaPlayConfig(cfg)) return lhcTemaHcwUniverse().length
  // 生肖/尾数对碰：恰好 2 个属性
  if (isLhcSxDuipengConfig(cfg)) return LHC_SX_DUIPENG_MAX_PICKS
  if (isLhcWsDuipengConfig(cfg)) return LHC_WS_DUIPENG_MAX_PICKS
  if (isLhcSwDuipengConfig(cfg)) return LHC_SW_DUIPENG_MAX_PICKS
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (bm === 'baodan') return 1
  // 一码不定位：随机「选项个数」步进器上限固定 2（与第三方一致，勿等到提交）
  const pickCap = poolMaxPicksForConfig(cfg)
  if (bm === 'budingwei' && pickCap != null && pickCap > 0) {
    return Math.min(pickCap, numberPoolTokens.value.length || pickCap)
  }
  if (bm === 'weishu') return Math.min(WEISHU_MAX_BET_UNITS, numberPoolTokens.value.length || 10)
  const uni = rdAttributeUniverse()
  // 和值/跨度：选项个数还受组合注数上限约束（前二满选 19→18，避免保存偶发超 90）
  if (bm === 'hezhi' || bm === 'kuadu' || /和值|跨度/.test(cfg.playMethodLabel ?? '')) {
    return maxHezhiKuaduRandomCount(cfg, uni)
  }
  if (pickCap != null && pickCap > 0) {
    return Math.min(pickCap, Math.max(1, uni.length))
  }
  return Math.max(1, uni.length)
}

const rdSingleCountMax = computed(() => {
  if (rdWholeTicket.value) return 200
  if (rdZuxuanPool.value) return Math.max(3, numberPoolTokens.value.length)
  if (rdAttribute.value) return rdAttributeUniverseMax()
  return 10
})
const rdSingleCountMin = computed(() => {
  if (rdWholeTicket.value) return 1
  if (rdZuxuanPool.value) {
    // 组三≥2、组六≥3；其它组选号池默认 ≥2（勿用 segmentLen/positionCount，任选回退五位会把下限抬成 5）
    return zuxuanPoolMinPick(schemePlayConfig.value) ?? 2
  }
  if (isLhcGuoguanConfig(schemePlayConfig.value)) return 2
  // 生肖/尾数对碰：恰好 2 个属性
  if (rdAttribute.value && isLhcSxDuipengConfig(schemePlayConfig.value)) {
    return LHC_SX_DUIPENG_MIN_PICKS
  }
  if (rdAttribute.value && isLhcWsDuipengConfig(schemePlayConfig.value)) {
    return LHC_WS_DUIPENG_MIN_PICKS
  }
  if (rdAttribute.value && isLhcSwDuipengConfig(schemePlayConfig.value)) {
    return LHC_SW_DUIPENG_MIN_PICKS
  }
  // 二码不定位≥2、三码≥3、五星二/三码≥4（第三方「投注数字不能低于两个」）
  if (rdAttribute.value) {
    const min = budingweiMinPicks(schemePlayConfig.value)
    if (min != null && min > 1) return min
  }
  return 1
})

/** 玩法切换后把选项个数钳到当前宇宙上下限（避免特殊号仍显示 >3） */
watch(
  [rdSingleCountMax, rdSingleCountMin, rdSingleCountMode, isRdZuDual, isRdLhcRenyiDuipeng],
  ([max, min, single, zu12, renyiDuipeng]) => {
    if (zu12 || renyiDuipeng) {
      ensureRdCounts()
      return
    }
    if (!single) return
    const cur = Math.trunc(Number(rdCounts.value[0]) || 0)
    if (cur <= 0) {
      rdCounts.value = [min, ...rdCounts.value.slice(1)]
      return
    }
    // 低于 min 才抬；高于 max 才压。勿在 cur=0 时用 min 覆盖已灌入前的空档误伤后续赋值。
    if (cur < min) rdCounts.value = [min, ...rdCounts.value.slice(1)]
    else if (cur > max) rdCounts.value = [max, ...rdCounts.value.slice(1)]
  },
)

/** 各面板状态随玩法位数 / 运行类型就绪（须在 ensureHcwPools / ensureRdCounts 声明之后） */
watch(
  [
    positionCount,
    runTypeId,
    isLonghuPlay,
    lotteryCode,
    hcwTotalPeriods,
    () => schemePlayConfig.value.playTypeId,
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
      ensureRenxuanRunPositions()
      // 直选单式冷热：开奖选位为空时填默认前 k 位
      if (
        isRenxuanHcwDualPosPlay.value &&
        sanitizeHcwOpenPosIdxs(hcwOpenPosIdxs.value).length === 0
      ) {
        ensureHcwOpenPositions()
      }
      ensureHcwPools()
      // 勿在此无条件清空频次：从方案模式返回时玩法字段会连触发，
      // 清空 + 竞态失败会把选项池次数留成「—」。
      scheduleHcwStats()
    }
    if (runTypeId.value === 'random_draw') {
      ensureRenxuanRunPositions()
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
    if (isRdLhcRenyiDuipeng.value) {
      // 兼容旧版 counts=[总数]，按 A=floor(总数/2)、B=其余回填。
      rdCounts.value = normalizeLhcRenyiDuipengRandomCounts(c.counts)
    } else {
      // 灌入时只保底 ≥1，勿用尚未稳定的 max 把已存选码个数夹小（详情 5 / 编辑变 2）
      rdCounts.value = c.counts.map((n) => Math.max(1, Math.trunc(Number(n)) || 1))
      // 组选12/4：旧配置仅 [K] 时补默认单号个数
      if (isZuDualPlayConfig(schemePlayConfig.value) && rdCounts.value.length < 2) {
        rdCounts.value = [
          Math.max(zuDualMinHeadCount(), rdCounts.value[0] ?? zuDualMinHeadCount()),
          zuDualMinSinglesCount(),
        ]
      }
    }
  }
  if (isLhcGuoguanConfig(schemePlayConfig.value)) {
    const count = Math.max(2, Math.min(6, rdCounts.value[0] ?? 2))
    rdCounts.value = [count]
  }
  const s = String(c.strategy ?? '')
  if (s === 'every' || s === 'keep' || s === 'after_hit' || s === 'after_miss') rdStrategy.value = s
  if (Array.isArray(c.positionIdxs)) {
    renxuanRunPosIdxs.value = c.positionIdxs
      .map((x) => Math.trunc(Number(x)))
      .filter((n) => Number.isInteger(n) && n >= 0 && n < 5)
    ensureRenxuanRunPositions()
  }
}

function shuffleInPlace<T>(arr: T[]): T[] {
  for (let j = arr.length - 1; j > 0; j--) {
    const k = Math.floor(Math.random() * (j + 1))
    ;[arr[j], arr[k]] = [arr[k]!, arr[j]!]
  }
  return arr
}

/** 一星每位最多 9 个号；其它按位玩法最多 10 */
const rdPerPosMax = computed(() => {
  if (isYixingDingweiPlayConfig(schemePlayConfig.value)) return YIXING_MAX_PICKS_PER_POS
  // 后二大小单双等：每位仅 1 个（大/小/单/双）
  if (isPerPosDxdsPlayConfig(schemePlayConfig.value)) return 1
  return 10
})

/** 本地生成预览号码（含属性家族选项抽样） */
function generateRdPreview(): void {
  ensureRdCounts()
  if (isRdLhcRenyiDuipeng.value) {
    const [aCount, bCount] = rdLhcRenyiDuipengCounts.value
    rdWholePreview.value = [randomLhcRenyiDuipengContentForCounts(aCount, bCount)]
    rdPreview.value = []
    return
  }
  if (rdWholeTicket.value) {
    const n = Math.min(200, Math.max(1, rdCounts.value[0] ?? 1))
    // 组三单式：须两同+一异（勿生成 012/111）
    if (isZu3DanshiConfig(schemePlayConfig.value)) {
      const raw = randomZu3DanshiTickets(n, Math.min(n, ZU3_DANSHI_FORM_COUNT))
      rdWholePreview.value = raw ? raw.split(',') : []
      rdPreview.value = []
      return
    }
    // 组六单式：须三位互异（勿生成 112/111）
    if (isZu6DanshiConfig(schemePlayConfig.value)) {
      const raw = randomZu6DanshiTickets(n, Math.min(n, ZU6_DANSHI_FORM_COUNT))
      rdWholePreview.value = raw ? raw.split(',') : []
      rdPreview.value = []
      return
    }
    // 混合组选：3 位整注，排除豹子，按组选形态去重
    if (isHunhePlayConfig(schemePlayConfig.value)) {
      const want = Math.min(n, 165)
      const out: string[] = []
      const seen = new Set<string>()
      let guard = 0
      while (out.length < want && guard++ < 3000) {
        let t = ''
        for (let i = 0; i < 3; i++) t += String(Math.floor(Math.random() * 10))
        if ([...t].every((c) => c === t[0])) continue
        const key = [...t].sort().join('')
        if (seen.has(key)) continue
        seen.add(key)
        out.push(t)
      }
      rdWholePreview.value = out
      rdPreview.value = []
      return
    }
    const pool = [...numberPoolTokens.value]
    const positions = positionCount.value
    const skipBaozi = isSoloBaoziRestrictedPlay(schemePlayConfig.value)
    const seen = new Set<string>()
    const out: string[] = []
    for (let a = 0; out.length < n && a < n * 100 + 100; a++) {
      const digits = Array.from({ length: positions }, () => pool[Math.floor(Math.random() * pool.length)] ?? '0')
      const key = digits.join('')
      if (skipBaozi && isBaoziDigitTicket(key)) continue
      if (seen.has(key)) continue
      seen.add(key)
      out.push(key)
    }
    rdWholePreview.value = out
    rdPreview.value = []
    return
  }
  if (isRdZuDual.value) {
    ensureRdCounts()
    const d = rdCounts.value[0] ?? zuDualMinHeadCount()
    const minS = zuDualMinSinglesCount()
    const s = rdCounts.value[1] ?? minS
    const raw = randomZuDualContent(d, s)
    const meta = zuDualMetaOf(schemePlayConfig.value)
    if (meta) {
      const zones = parseZuDualZones(raw, meta.minHead, meta.minTail, meta.equalCounts)
      rdWholePreview.value = zones ? [zones.head.join(''), zones.tail.join('')] : []
    } else {
      rdWholePreview.value = []
    }
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
    const cfg = schemePlayConfig.value
    const bm = String(cfg.betMode ?? '').toLowerCase()
    const hezhiKuadu =
      bm === 'hezhi' || bm === 'kuadu' || /和值|跨度/.test(cfg.playMethodLabel ?? '')
    const universe = shuffleInPlace(rdAttributeUniverse())
    const k = Math.min(
      universe.length,
      rdSingleCountMax.value,
      Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value),
    )
    let picks = universe.slice(0, k)
    if (isLhcGuoguanConfig(cfg)) {
      const positions = Array(6).fill('') as string[]
      const positionIndexes = shuffleInPlace([0, 1, 2, 3, 4, 5]).slice(0, k)
      for (const positionIndex of positionIndexes) {
        positions[positionIndex] = universe[Math.floor(Math.random() * universe.length)] ?? ''
      }
      rdWholePreview.value = [positions.join(',')]
      rdPreview.value = []
      return
    }
    // 生尾对碰：必须各 1 肖 + 1 尾（勿从混合宇宙 slice k → 两肖/两尾）
    if (isLhcSwDuipengConfig(cfg) || String(cfg.betMode ?? '').toLowerCase() === 'sw_dp') {
      picks = [...pickRandomLhcSwDuipengPair()]
    }
    // 和值/跨度：随机样例可能组合注数超限，回退为组合数较小的可行集（对齐引擎重抽）
    if (hezhiKuadu && !validateGroupContent(cfg, picks.join(',')).ok) {
      picks = greedyHezhiKuaduPicksUnderMax(cfg, k, rdAttributeUniverse())
    }
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
  // 按位玩法（一星/前三/前二/五星复式 / 后二大小单双等）：按位预览
  // 直选单式/复式：避免抽成仅豹子（各位同一单码），必要时重抽
  const perPosMax = rdPerPosMax.value
  const perPosDxds = isPerPosDxdsPlayConfig(schemePlayConfig.value)
  const textOpts = perPosDxds ? textPickOptionsForConfig(schemePlayConfig.value) : []
  const attrUniverse = perPosDxds ? rdAttributeUniverse() : []
  const cfg = schemePlayConfig.value
  const seg = cfg.segmentLen > 0 ? cfg.segmentLen : positionCount.value
  const guardBaozi =
    !perPosDxds &&
    isSoloBaoziRestrictedPlay(cfg) &&
    (isZhixuanDanshiPerPosPlay(cfg) || isZhixuanFushiPlayConfig(cfg))
  for (let attempt = 0; attempt < 48; attempt++) {
    const pools = Array.from({ length: positionCount.value }, (_, i) => {
      const source = perPosDxds
        ? [...(textOpts.length ? textOpts : attrUniverse)]
        : [...numberPoolTokens.value]
      const pool = shuffleInPlace(source)
      const count = Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1), pool.length)
      const picks = pool.slice(0, count)
      if (perPosDxds && textOpts.length) {
        const order = new Map(textOpts.map((t, oi) => [t, oi]))
        picks.sort((a, b) => (order.get(a) ?? 0) - (order.get(b) ?? 0))
      } else {
        picks.sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
      }
      return picks
    })
    const poolContent = pools.map((row) => row.join(',')).join('\n')
    if (guardBaozi && isSchemeSoloBaoziContent(cfg, poolContent)) {
      continue
    }
    if (guardBaozi && isZhixuanDanshiPerPosPlay(cfg) && seg > 1) {
      const ok = expandZhixuanPoolToDanshiWithoutBaozi(poolContent, seg)
      if (!ok) continue
    }
    rdPreview.value = pools
    rdWholePreview.value = []
    return
  }
  // 兜底：仍写入最后一轮（保存时还会再滤豹子）
  const fallback = Array.from({ length: positionCount.value }, (_, i) => {
    const source = perPosDxds
      ? [...(textOpts.length ? textOpts : attrUniverse)]
      : [...numberPoolTokens.value]
    const pool = shuffleInPlace(source)
    const count = Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1), pool.length)
    return pool.slice(0, count).sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
  })
  rdPreview.value = fallback
  rdWholePreview.value = []
}

interface RdPreviewTag {
  key: string
  /** 展示文案 */
  label: string
  kind: 'whole' | 'pos'
  /** 按位：位下标；整注/号池：条目下标；组选12：0=二重号 1=单号 */
  index: number
  /** 按位：该 tag 对应的单个号码（关闭时只删这一号） */
  digit?: string
}

/** 组选12/4 预览两段拼回「头区,单号」；缺段则空串 */
function rdZuDualPreviewWire(): string {
  const head = String(rdWholePreview.value[0] ?? '').replace(/[,，\s]/g, '')
  const singles = String(rdWholePreview.value[1] ?? '').replace(/[,，\s]/g, '')
  if (!head && !singles) return ''
  return `${head},${singles}`
}

/**
 * 预览 tag：
 * - 按位玩法（中三/前三直选复式、一星、大小单双等）：每位一枚 tag，位内号码合并
 *   （中三 → 3 枚；勿把「百位 1,2」拆成两枚数字 tag）
 * - 组选12：二重号 / 单号各一枚 tag
 * - 单式整注 / 组选号池：一注或号池条目一枚
 */
const rdPreviewTags = computed<RdPreviewTag[]>(() => {
  if (isRdLhcRenyiDuipeng.value) {
    const ticket = String(rdWholePreview.value[0] ?? '')
    const [a = '', b = ''] = ticket.split('|')
    if (!a || !b) return []
    return [{
      key: `renyi-duipeng-${ticket}`,
      label: `A区 ${a.split(',').join('\u2009')}  |  B区 ${b.split(',').join('\u2009')}`,
      kind: 'whole',
      index: 0,
    }]
  }
  if (isRdZuDual.value) {
    const labels = [...triggerZuDualZoneLabels.value]
    const out: RdPreviewTag[] = []
    for (let i = 0; i < 2; i++) {
      const run = String(rdWholePreview.value[i] ?? '').replace(/[,，\s]/g, '')
      if (!run) continue
      out.push({
        key: `zu-dual-${i}-${run}`,
        label: `${labels[i]} ${[...run].join('\u2009')}`,
        kind: 'whole',
        index: i,
      })
    }
    return out
  }
  if (isLhcGuoguanConfig(schemePlayConfig.value)) {
    const ticket = String(rdWholePreview.value[0] ?? '')
    const parts = ticket.replace(/，/g, ',').split(',')
    const label = parts
      .slice(0, 6)
      .map((pick, index) => (pick ? `${LHC_GUOGUAN_POSITION_LABELS[index]} ${pick}` : ''))
      .filter(Boolean)
      .join(' · ')
    return label ? [{ key: `guoguan-${ticket}`, label, kind: 'whole' as const, index: 0 }] : []
  }
  // 单式整注 / 组选号池：整注预览
  if (rdSingleCountMode.value) {
    if (!rdWholePreview.value.length) return []
    return rdWholePreview.value.map((ticket, index) => ({
      key: `w-${index}-${ticket}`,
      label: ticket.includes(',')
        ? ticket.split(/[,，]/).filter(Boolean).join('\u2009')
        : ticket,
      kind: 'whole' as const,
      index,
    }))
  }
  // 按位模式：只用 rdPreview，禁止整注笛卡尔残留盖住按位 tag
  const rows = rdPreview.value
  if (!rows.length) return []
  const out: RdPreviewTag[] = []
  rows.forEach((row, index) => {
    if (!row?.length) return
    const posName = positionLabels.value[index] ?? ''
    const body = row.join('\u2009')
    out.push({
      key: `p-${index}-${row.join(',')}`,
      label: posName ? `${posName} ${body}` : body,
      kind: 'pos' as const,
      index,
      // 不设 digit：关闭时清空该整位
    })
  })
  return out
})

function removeRdPreviewTag(tag: RdPreviewTag): void {
  if (tag.kind === 'whole') {
    if (isRdZuDual.value) {
      // 清空对应区，保留另一区（长度固定 2，避免下标错位）
      const next = [rdWholePreview.value[0] ?? '', rdWholePreview.value[1] ?? '']
      next[tag.index] = ''
      rdWholePreview.value = next[0] || next[1] ? next : []
      return
    }
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
  if (isRdLhcRenyiDuipeng.value) {
    const [aCount, bCount] = rdLhcRenyiDuipengCounts.value
    const sample = rdWholePreview.value[0] || randomLhcRenyiDuipengContentForCounts(aCount, bCount)
    return countBetUnits(schemePlayConfig.value, sample)
  }
  // 单式整注随机：有预览则按选位×注数计；否则先按注数（选位倍率在保存/出号时生效）
  if (rdWholeTicket.value) {
    const n = Math.min(200, Math.max(1, rdCounts.value[0] ?? 1))
    if (schemeUsesRenxuanRunPos.value && rdWholePreview.value.length) {
      return countBetUnits(
        schemePlayConfig.value,
        wrapRenxuanRunContent(rdWholePreview.value.join(',')),
      )
    }
    if (schemeUsesRenxuanRunPos.value) {
      const seg = Math.max(1, schemePlayConfig.value.segmentLen || renxuanRunPosNeed.value)
      const sample = Array.from({ length: n }, (_, i) =>
        String(i).padStart(seg, '0').slice(-seg),
      ).join(',')
      return countBetUnits(schemePlayConfig.value, wrapRenxuanRunContent(sample))
    }
    return n
  }
  // 组选12/4：头区/单号双区预览计注
  if (isRdZuDual.value) {
    const sample =
      rdZuDualPreviewWire() ||
      randomZuDualContent(rdCounts.value[0] ?? 1, rdCounts.value[1] ?? zuDualMinSinglesCount())
    return countBetUnits(schemePlayConfig.value, wrapRenxuanRunContent(sample))
  }
  // 组选号码池随机：按选中号码池走 countBetUnits（组选口径）
  if (rdZuxuanPool.value) {
    const pool = [...numberPoolTokens.value]
    const k = Math.min(pool.length, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value))
    return countBetUnits(schemePlayConfig.value, wrapRenxuanRunContent(pool.slice(0, k).join(',')))
  }
  // 属性家族：和值等走组合计注；任选再乘选位组合
  if (rdAttribute.value) {
    const uni = rdAttributeUniverse()
    const k = Math.min(rdSingleCountMax.value, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? 1))
    const swDp =
      isLhcSwDuipengConfig(schemePlayConfig.value) ||
      String(schemePlayConfig.value.betMode ?? '').toLowerCase() === 'sw_dp'
    const barDp =
      isLhcSxDuipengConfig(schemePlayConfig.value) ||
      isLhcWsDuipengConfig(schemePlayConfig.value) ||
      swDp
    const sep = barDp ? '|' : ','
    // 生尾对碰：宇宙前两项是两肖，不能 slice 估注；用预览或固定 肖|尾 样例
    const picks = swDp
      ? rdWholePreview.value.length >= 2
        ? rdWholePreview.value.slice(0, 2).join('|')
        : pickRandomLhcSwDuipengPair().join('|')
      : uni.slice(0, k).join(sep)
    if (barDp && picks) {
      return countBetUnits(schemePlayConfig.value, picks)
    }
    if (schemeUsesRenxuanRunPos.value && picks) {
      return countBetUnits(schemePlayConfig.value, wrapRenxuanRunContent(picks))
    }
    return k
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
  return countBetUnits(schemePlayConfig.value, wrapRenxuanRunContent(lines.join('\n')))
})

// --- builtin_plan 内置计画 ---
const favorites = ref<SchemeFavoriteRow[]>([])
const favoritesLoading = ref(false)
const favoritesLoaded = ref(false)
const favoritesLoadError = ref('')
const favSelectedSnapshotId = ref('')
const builtinSnapshotId = ref('')
const builtinApplying = ref(false)

/** 内置计划仅展示当前关联彩种下的收藏 */
const favoritesForLottery = computed(() => {
  const lot = lotteryCode.value.trim()
  if (!lot) return []
  return favorites.value.filter((f) => f.lotteryCode === lot)
})

async function loadFavorites(force = false): Promise<void> {
  if (favoritesLoading.value) return
  if (favoritesLoaded.value && !force) return
  favoritesLoading.value = true
  favoritesLoadError.value = ''
  try {
    favorites.value = await fetchSchemeFavorites()
    favoritesLoaded.value = true
  } catch (err) {
    favorites.value = []
    favoritesLoaded.value = false
    favoritesLoadError.value =
      err instanceof ApiError ? err.message : err instanceof Error ? err.message : '收藏列表加载失败'
    ElMessage.warning(favoritesLoadError.value)
  } finally {
    favoritesLoading.value = false
  }
}

function formatFavoredAt(raw: string): string {
  const t = new Date(raw)
  if (Number.isNaN(t.getTime())) return raw
  return t.toLocaleString('zh-CN', { hour12: false })
}

/** 点击收藏方案即应用，不再额外要求确认。 */
async function selectBuiltinPlan(snapshotId: string): Promise<void> {
  if (builtinApplying.value) return
  const previousSnapshotId = builtinSnapshotId.value
  favSelectedSnapshotId.value = snapshotId
  if (!lotteryCode.value.trim()) {
    ElMessage.warning('请先选择彩种')
    return
  }
  if (!favSelectedSnapshotId.value) {
    ElMessage.warning('请先选择一个收藏方案')
    return
  }
  const picked = favorites.value.find((f) => f.snapshotId === favSelectedSnapshotId.value)
  if (picked && picked.lotteryCode !== lotteryCode.value.trim()) {
    ElMessage.warning('所选收藏方案与当前彩种不一致')
    return
  }
  if (isDraftScheme.value) {
    builtinSnapshotId.value = favSelectedSnapshotId.value
    ElMessage.success('已选择收藏方案')
    persistDraft()
    return
  }
  builtinApplying.value = true
  try {
    await updateSchemeDefinition(schemeId.value, {
      builtinPlan: { snapshotId: favSelectedSnapshotId.value },
    })
    builtinSnapshotId.value = favSelectedSnapshotId.value
    ElMessage.success('已复制该方案配置')
    await loadRemoteDefinition()
  } catch (err) {
    favSelectedSnapshotId.value = previousSnapshotId
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
  multCoeff.value = normalizeSchemeMultiplier(draft.multCoeff || '1')
  shareStatus.value = draft.shareStatus
  runTypeId.value = normalizeRunTypeId(draft.meta.runTypeId)
  lotteryCode.value = draft.meta.lotteryCode || lotteryCode.value
  playTypeId.value = draft.meta.playTypeId || playTypeId.value
  subPlayId.value = draft.meta.subPlayId || subPlayId.value
  if (draft.betMultiplierKind) betMultiplierKind.value = draft.betMultiplierKind
  if (draft.betMultiplier) applyBetMultiplierFromConfig(draft.betMultiplier)
  if (draft.builtinSnapshotId) {
    builtinSnapshotId.value = draft.builtinSnapshotId
    favSelectedSnapshotId.value = draft.builtinSnapshotId
  }
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
    lotteryCode: lotteryCode.value.trim(),
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
    multCoeff: normalizeSchemeMultiplier(multCoeff.value),
    shareStatus: shareStatus.value,
    betMultiplierKind: betMultiplierKind.value,
    betMultiplier: betMultiplierPayload.value ?? existing?.betMultiplier,
    builtinSnapshotId: builtinSnapshotId.value || undefined,
    jushuList: rtFields.jushuList,
    triggerBet: rtFields.triggerBet,
    hotColdWarm: rtFields.hotColdWarm,
    randomDraw: rtFields.randomDraw,
  }
}

function syncRunTypePanelsAfterSnapshot(): void {
  if (runTypeId.value === 'adv_trigger_bet') {
    ensureTriggerRows()
    ensureTriggerPositions()
  }
  if (runTypeId.value === 'hot_cold_warm') {
    // 从方案模式等子页返回：同玩法去重，已有频次则不重拉清空
    scheduleHcwStats()
  }
  if (runTypeId.value === 'random_draw') ensureRdCounts()
  if (runTypeId.value === 'builtin_plan') void loadFavorites()
  if (runTypeId.value === 'adv_fixed_rotate' && !jushuList.value.length) seedJushuFromGroups()
}

watch(isBuiltinPlan, (on) => {
  if (on) void loadFavorites(true)
})

watch(lotteryCode, () => {
  if (!isBuiltinPlan.value) return
  const lot = lotteryCode.value.trim()
  if (
    favSelectedSnapshotId.value &&
    !favorites.value.some(
      (f) => f.snapshotId === favSelectedSnapshotId.value && f.lotteryCode === lot,
    )
  ) {
    favSelectedSnapshotId.value = ''
  }
})

/** 从倍投设定等子页返回时，用离开前快照覆盖远端/草稿加载结果 */
function applyPendingRestoreSnapshot(): void {
  const restored = consumeSchemeEditRestoreSnapshot(schemeId.value)
  if (restored) {
    applyDraftSnapshot(restored)
    // 草稿页可能已在倍投确认时写过 localStorage；先灌回，再让 pending 盖最终选择
    const draft = loadSchemeDraft()
    if (draft?.betMultiplier) {
      applyBetMultiplierFromConfig(draft.betMultiplier)
      if (draft.betMultiplierKind) betMultiplierKind.value = draft.betMultiplierKind
    }
  }
  // 必须在快照/远端旧配置之后消费：pending 代表刚确认的方案模式（尚未点「完成」落库）
  consumePendingBetMultiplierIfAny()
  const qk = route.query.bmsKind
  const kindFromQuery = String(Array.isArray(qk) ? qk[0] : qk ?? '')
  if (kindFromQuery === '0' || kindFromQuery === '1' || kindFromQuery === '2' || kindFromQuery === '3') {
    betMultiplierKind.value = kindFromQuery
  }
  if (restored) syncRunTypePanelsAfterSnapshot()
}

async function loadRemoteDefinition() {
  suppressPersistHydration = true
  if (remotePersistTimer) {
    clearTimeout(remotePersistTimer)
    remotePersistTimer = null
  }
  try {
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
      if (!lotteryCode.value && lotteries.value.length) {
        lotteryCode.value = lotteries.value[0].code
      }
      if (lotteryCode.value) {
        await loadRunTypeOptions(lotteryCode.value)
        await loadIdentityPlayTree(lotteryCode.value)
      }
      await loadPlayTree()
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
        multCoeff.value = normalizeSchemeMultiplier(String(cfg.multCoeff).trim())
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
      // 先就绪玩法树再灌映射表，避免号池未解析时用 0–9 重建把正/反投洗空
      await loadPlayTree()
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
        favSelectedSnapshotId.value = builtinSnapshotId.value
      }
      if (runTypeId.value === 'adv_fixed_rotate' && !jushuList.value.length) {
        seedJushuFromGroups()
      }
      if (lotteryCode.value) {
        void loadRunTypeOptions(lotteryCode.value)
        void loadIdentityPlayTree(lotteryCode.value)
      }
      syncRunTypePanelsAfterSnapshot()
      applyPendingRestoreSnapshot()
    } catch {
      /* 列表加载失败时保留 query 默认值 */
    }
  } finally {
    await nextTick()
    remoteReady.value = true
    suppressPersistHydration = false
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
      const perPosText = triggerPerPosTextBet.value
      const tema = isLhcTemaPlayConfig(schemePlayConfig.value)
      const sxDp = isLhcSxDuipengConfig(schemePlayConfig.value)
      const wsDp = isLhcWsDuipengConfig(schemePlayConfig.value)
      const swDp = isLhcSwDuipengConfig(schemePlayConfig.value)
      const renyiDp = isLhcRenyiDuipengConfig(schemePlayConfig.value)
      const guoguan = isLhcGuoguanConfig(schemePlayConfig.value)
      const normalizeDp = (raw: string) =>
        sxDp
          ? normalizeSxDuipengTriggerContent(raw)
          : wsDp
            ? normalizeWsDuipengTriggerContent(raw)
            : swDp
              ? normalizeSwDuipengTriggerContent(raw)
              : renyiDp
                ? normalizeLhcRenyiDuipengTriggerContent(raw)
                : ''
      const triggerBet: SchemeTriggerBet = {
        rows: triggerRows.value.map((r) => ({
          ...r,
          pos: perPos
            ? perPosText
              ? sanitizeTriggerPerPosTextField(r.pos)
              : sanitizeTriggerPerPosField(r.pos)
            : guoguan
              ? normalizeLhcGuoguanTriggerContent(String(r.pos ?? ''))
              : sxDp || wsDp || swDp || renyiDp
              ? normalizeDp(r.pos)
              : textPlay
                ? triggerTextTokens(r.pos)
                    .filter((t) => triggerBetOptions.value.includes(t))
                    .join(',')
                : tema
                  ? // 落库/下单用 号码|属性|波色；编辑框仍是 flat 混选
                    normalizeLhcTemaContent(sanitizeTriggerBetContent(r.pos))
                  : sanitizeTriggerBetContent(r.pos),
          neg: perPos
            ? perPosText
              ? sanitizeTriggerPerPosTextField(r.neg)
              : sanitizeTriggerPerPosField(r.neg)
            : guoguan
              ? normalizeLhcGuoguanTriggerContent(String(r.neg ?? ''))
              : sxDp || wsDp || swDp || renyiDp
              ? normalizeDp(r.neg)
              : textPlay
                ? triggerTextTokens(r.neg)
                    .filter((t) => triggerBetOptions.value.includes(t))
                    .join(',')
                : tema
                  ? normalizeLhcTemaContent(sanitizeTriggerBetContent(r.neg))
                  : sanitizeTriggerBetContent(r.neg),
        })),
        mode: triggerMode.value,
      }
      if (showTriggerPositionPicker.value || isRenxuanTriggerPlay.value) {
        ensureTriggerPositions()
        triggerBet.positionIdxs = normalizeTriggerPositionIdxs(
          triggerPositionIdxs.value,
          triggerPosSpaceSize(),
        )
        if (isRenxuanTriggerPlay.value) {
          ensureTriggerOpenPosition()
          triggerBet.openPositionIdx = triggerOpenPositionIdx.value
        }
      }
      return { triggerBet }
    }
    case 'hot_cold_warm': {
      ensureHcwPools()
      ensureRenxuanRunPositions()
      // 草稿/自动保存：只快照当前开奖选位，禁止 ensure 补万位写回编辑态
      const n = hcwDimCount()
      const hotColdWarm: SchemeHotColdWarm = {
        totalPeriods: Math.min(100, Math.max(20, Math.trunc(Number(hcwTotalPeriods.value) || 20))),
        // 权威：名次（0=最热）；热/冷/全/清只改 ranks，不落库 pool/pickTypes
        ranks: Array.from({ length: n }, (_, i) => [...(hcwRanks.value[i] ?? [])]),
        strategy: hcwStrategy.value,
        winRotate: hcwStrategy.value === 'after_hit',
      }
      if (schemeUsesRenxuanRunPos.value) {
        hotColdWarm.positionIdxs = [...renxuanRunPosIdxs.value]
      }
      if (showHcwOpenPosition.value) {
        hotColdWarm.openPositionIdxs = sanitizeHcwOpenPosIdxs(hcwOpenPosIdxs.value)
      }
      return { hotColdWarm }
    }
    case 'random_draw': {
      // 单式=注数 / 组选=选码个数 → counts=[K]；组选12 → [二重,单号]；按位型 → 每位号码/选项数量
      ensureRenxuanRunPositions()
      const perPosMax = rdPerPosMax.value
      const counts = isRdLhcRenyiDuipeng.value
        ? [...rdLhcRenyiDuipengCounts.value]
        : isRdZuDual.value
        ? [
            Math.min(10, Math.max(1, Math.trunc(Number(rdCounts.value[0]) || 1))),
            Math.min(10, Math.max(2, Math.trunc(Number(rdCounts.value[1]) || 2))),
          ]
        : rdSingleCountMode.value
          ? [Math.min(rdSingleCountMax.value, Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value))]
          : Array.from({ length: positionCount.value }, (_, i) =>
              Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1)),
            )
      const randomDraw: SchemeRandomDraw = { counts, strategy: rdStrategy.value }
      if (schemeUsesRenxuanRunPos.value) {
        randomDraw.positionIdxs = [...renxuanRunPosIdxs.value]
      }
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
  const name = schemeName.value.trim()
  return {
    // 空名不写回（防抖输入中途清空时）；显式保存前已校验非空
    ...(name ? { schemeName: name } : {}),
    simBet: simBet.value,
    schemeFunds: schemeFunds.value,
    schemeCurrency: schemeCurrency.value,
    multCoeff: normalizeSchemeMultiplier(multCoeff.value),
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
    ...(betMultiplierPayload.value
      ? { betMultiplier: betMultiplierPayload.value as unknown as Record<string, unknown> }
      : {}),
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
  if (suppressPersistHydration || !remoteReady.value) return
  if (isDraftScheme.value) {
    saveSchemeDraft(buildDraftSnapshot())
    return
  }
  // 无合法 id 时勿 PATCH /client/schemes/（会 404 page not found）
  if (!schemeId.value.trim()) return
  void updateSchemeDefinition(schemeId.value, buildRemoteDraftPatch()).catch(() => { })
}

/** 远端灌入 / 玩法树就绪前禁止防抖写回，避免开某投某映射被空表覆盖入库 */
let suppressPersistHydration = false

function persistDraft() {
  if (suppressPersistHydration || !remoteReady.value) return
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
    hcwRanks,
    hcwStrategy,
    hcwOpenPosIdxs,
    renxuanRunPosIdxs,
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

/** 完整倍投载荷（含高级 rounds）；编辑云端方案时「完成」必须写回定义 */
const betMultiplierPayload = ref<BetMultiplierPayload | null>(null)

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
  const payload = raw as BetMultiplierPayload
  const kind = payload.kind
  if (kind === '0' || kind === '1' || kind === '2' || kind === '3') {
    betMultiplierKind.value = kind
    betMultiplierPayload.value = payload
  }
}

function consumePendingBetMultiplierIfAny(): void {
  const pending = consumeSchemeEditBmPending(schemeId.value)
  if (!pending) return
  applyBetMultiplierFromConfig(pending)
  betMultiplierError.value = ''
  if (isDraftScheme.value) {
    const draft = loadSchemeDraft()
    if (draft) {
      draft.betMultiplier = pending
      draft.betMultiplierKind = pending.kind
      saveSchemeDraft(draft)
    }
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
    // 未水合时勿消费 pending：setup 的 immediate 会抢在 loadRemoteDefinition 前清空，
    // 随后离开前快照/远端旧 betMultiplier 会把右侧方案模式盖回旧值。
    // 已水合（同页 replace、不重挂载）时在此消费。
    if (remoteReady.value) {
      consumePendingBetMultiplierIfAny()
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
  ElMessage.success(`已新增第 ${schemeGroups.value.length} 组`)
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
  if (!lottery) {
    await warn('请选择彩种')
    return
  }
  if (!isBuiltinPlan.value && (!playTypeId.value || !subPlayId.value)) {
    await warn('请选择玩法')
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
  normalizeMultCoeff()
  const multCoeffRaw = multCoeff.value.trim()
  const multCoeffNum = Number(multCoeffRaw)
  if (!multCoeffRaw || !Number.isInteger(multCoeffNum) || multCoeffNum < 1) {
    await warn('倍数系数须为不小于 1 的正整数')
    return
  }

  const rt = runTypeId.value
  const builtinPlanSave = validateBuiltinPlanSave(rt, builtinSnapshotId.value)
  if (!builtinPlanSave.ok) {
    await warn(builtinPlanSave.message ?? '内置计划配置不完整')
    return
  }
  if (builtinPlanSave.skipManualContentValidation) {
    // 内置计划的方案内容由收藏快照在服务端物化，勿按空 schemeGroups 拦截。
  } else if (rt === 'adv_fixed_rotate') {
    if (!jushuList.value.length) {
      await warn('请至少添加一局投注号码')
      return
    }
    // 与局数内容对齐，供仍读取 schemeGroups 的下游兜底
    schemeGroups.value = jushuList.value.map((r) => r.content)
  } else if (rt === 'adv_trigger_bet') {
    if (showTriggerPerPosColumns.value) {
      const n = Math.max(1, triggerColumnLabels.value.length || positionCount.value)
      const incomplete = triggerRows.value.find((r) => {
        if (!r.enabled) return false
        const posParts = triggerFieldParts(r.pos, n)
        const negParts = triggerFieldParts(r.neg, n)
        return posParts.some((c) => !c) || negParts.some((c) => !c)
      })
      if (incomplete) {
        const posNames = triggerColumnLabels.value.map((l) => triggerPosName(l)).join('、')
        await warn(`启用行须填齐各位正投与反投（${posNames}）`)
        return
      }
      const anyEnabled = triggerRows.value.some((r) => r.enabled)
      if (!anyEnabled) {
        await warn('请至少启用一行开某投某映射')
        return
      }
    } else {
      // 和值/组选和值/包胆等单档：每个启用号码的正投、反投都必须填写
      const anyEnabledFlat = triggerRows.value.some((r) => r.enabled)
      if (!anyEnabledFlat) {
        await warn('请至少启用一行开某投某映射')
        return
      }
      const incompleteFlat = triggerRows.value.find(
        (r) => r.enabled && (String(r.pos).trim() === '' || String(r.neg).trim() === ''),
      )
      if (incompleteFlat) {
        await warn(`启用号码「${incompleteFlat.open}」须同时填写正投与反投（可用「全部随机」）`)
        return
      }
    }
    if (showTriggerPositionPicker.value) {
      ensureTriggerPositions()
      if (isRenxuanTriggerPlay.value) {
        const need = triggerRenPosNeed.value
        const n = triggerPositionIdxs.value.length
        if (n < need || n > 5) {
          await warn(`请从万千百十个中勾选至少 ${need} 个、最多 5 个投注位置`)
          return
        }
        ensureTriggerOpenPosition()
      } else if (!triggerPositionIdxs.value.length) {
        await warn('请至少选择一个投注位')
        return
      }
    }
    if (isLhcGuoguanConfig(schemePlayConfig.value)) {
      for (const row of triggerRows.value) {
        if (!row.enabled) continue
        for (const [name, raw] of [['正投', row.pos], ['反投', row.neg]] as const) {
          const check = validateGroupContent(schemePlayConfig.value, String(raw ?? ''))
          if (!check.ok) {
            await warn(`开出 ${row.open} 的${name}：${check.message}`)
            return
          }
        }
      }
    }
    // 组三/组六号池下限；组选/组三单式整注合法性（每注 N 位、组三须两同+一异）
    {
      const minPick = zuxuanPoolMinPick(schemePlayConfig.value)
      const needValidate =
        minPick != null ||
        isZuxuanDanshiConfig(schemePlayConfig.value) ||
        isZu3DanshiConfig(schemePlayConfig.value) ||
        isZu6DanshiConfig(schemePlayConfig.value) ||
        isHunhePlayConfig(schemePlayConfig.value)
      if (needValidate) {
        for (const r of triggerRows.value) {
          if (!r.enabled) continue
          for (const [name, raw] of [
            ['正投', r.pos],
            ['反投', r.neg],
          ] as const) {
            const cell = String(raw ?? '').trim()
            if (!cell) continue
            // 任选开某投某：正/反投格子无位名前缀，校验前补投注选位（与下方统一校验一致）
            let toCheck = cell
            if (isRenxuanTriggerPlay.value) {
              toCheck = showTriggerPerPosColumns.value
                ? renxuanTriggerPoolToGroupContent(cell)
                : buildRenxuanPositionContent(triggerBetPositionLabels.value, cell)
            }
            if (!toCheck.trim()) {
              await warn(`开出 ${r.open} 的${name}：选号无效`)
              return
            }
            const check = validateGroupContent(schemePlayConfig.value, toCheck)
            if (!check.ok) {
              await warn(`开出 ${r.open} 的${name}：${check.message}`)
              return
            }
          }
        }
      }
    }
    // schemeGroups 仅作占位样例；按位号池需展开成单式整注，避免保存校验误报「单式组合不合法」
    const sample = triggerRows.value.find((r) => r.enabled && String(r.pos).trim())
    let sampleContent = sample ? String(sample.pos).trim() : '0'
    const seg = schemePlayConfig.value.segmentLen
    const cfg = schemePlayConfig.value
    if (sampleContent && isRenxuanTriggerPlay.value) {
      // 任选单式：各位一码（如 4\n5）→ 万,千\n45；号池/和值：所选位 + 内容
      sampleContent = showTriggerPerPosColumns.value
        ? renxuanTriggerPoolToGroupContent(sampleContent) || sampleContent
        : buildRenxuanPositionContent(triggerBetPositionLabels.value, sampleContent)
    } else if (
      sampleContent &&
      showTriggerPerPosColumns.value &&
      (isSscDanshiLikeConfig(cfg) || isHunhePlayConfig(cfg)) &&
      seg > 1 &&
      isZhixuanPositionPoolContent(sampleContent, seg)
    ) {
      sampleContent = expandZhixuanPositionPoolToDanshi(sampleContent, seg) || sampleContent
      if (isHunhePlayConfig(cfg)) {
        // 排除豹子后可能为空：占位给一注合法样例，真实出号仍看 triggerBet
        sampleContent = normalizeHunheGroupContent(sampleContent, seg) || '123'
      }
    }
    schemeGroups.value = [sampleContent]
  } else if (rt === 'hot_cold_warm') {
    ensureHcwPools()
    ensureRenxuanRunPositions()
    if (isHcwLhcGuoguan.value) {
      const selected = hcwRanks.value.filter((r) => r.length > 0).length
      if (selected < 2) {
        await warn('过关冷热：请至少选择两个正码位置')
        return
      }
      const content = Array.from({ length: 6 }, (_, i) => (hcwPools.value[i] ?? [])[0] ?? '').join(',')
      const check = validateGroupContent(schemePlayConfig.value, content)
      if (!check.ok) {
        await warn(check.message)
        return
      }
      schemeGroups.value = [content]
    } else if (isHcwRenyiDuipeng.value) {
      const ranks = normalizeLhcRenyiDuipengHotColdRanks(hcwRanks.value, numberPoolTokens.value.length)
      if (!ranks.valid) {
        await warn('冷热任意对碰：A区、B区均须至少选择 1 个名次，且不可重复、合计最多 10 个')
        return
      }
      const content = formatLhcRenyiDuipengContent(hcwPools.value[0] ?? [], hcwPools.value[1] ?? [])
      if (!content) {
        await warn('冷热任意对碰：请等待冷热统计就绪后再提交')
        return
      }
      schemeGroups.value = [content]
    } else if (isHcwZuDual.value) {
      const headN = (hcwRanks.value[0] ?? []).length
      const singlesN = (hcwRanks.value[1] ?? []).length
      const headLabel = zuDualZoneHeadLabel()
      const tailLabel = triggerZuDualMeta.value?.tailLabel ?? '单号'
      const minHead = zuDualMinHeadCount()
      const minSingles = zuDualMinSinglesCount()
      if (headN < minHead) {
        await warn(`${headLabel}至少选择 ${minHead} 个`)
        return
      }
      if (singlesN < minSingles) {
        await warn(`${tailLabel}至少选择 ${minSingles} 个`)
        return
      }
      if (triggerZuDualMeta.value?.equalCounts && headN !== singlesN) {
        await warn(`${headLabel}与${tailLabel}个数须相同`)
        return
      }
      const dual = hcwZuDualPicks()
      const check = validateGroupContent(schemePlayConfig.value, wrapRenxuanRunContent(dual))
      if (!check.ok) {
        await warn(check.message)
        return
      }
      schemeGroups.value = [wrapRenxuanRunContent(dual)]
    } else if (isRenxuanHcwDualPosPlay.value) {
      const need = renxuanRunPosNeed.value
      const openN = sanitizeHcwOpenPosIdxs(hcwOpenPosIdxs.value).length
      if (openN !== need) {
        await warn(`开奖选位须选 ${need} 个（当前 ${openN} 个）`)
        return
      }
    }
    // 组三/组六冷热：号码池最少选号（组三≥2、组六≥3），与定码校验一致
    // 二码不定位≥2（第三方「投注数字不能低于两个」）
    // 组选单式：至少 segLen 个单码，保存时两两组合为整注
    if (!isHcwZuDual.value && hcwDigitOverall.value) {
      const pickN = (hcwRanks.value[0] ?? []).length
      const zuxuanMin = zuxuanPoolMinPick(schemePlayConfig.value)
      if (zuxuanMin != null && pickN < zuxuanMin) {
        await warn(zuxuanPoolMinPickMessage(schemePlayConfig.value))
        return
      }
      const bdwMin = budingweiMinPicks(schemePlayConfig.value)
      if (bdwMin != null && pickN < bdwMin) {
        await warn(budingweiMinPicksMessage(schemePlayConfig.value))
        return
      }
      if (isZuxuanDanshiConfig(schemePlayConfig.value)) {
        const need =
          schemePlayConfig.value.segmentLen >= 2
            ? schemePlayConfig.value.segmentLen
            : (schemePlayConfig.value.renPositionCount ?? 2)
        if (pickN < need) {
          await warn(`组选单式冷热：请至少勾选 ${need} 个名次（号码将自动组合为投注整注）`)
          return
        }
      }
    }
    // schemeGroups 仅占位：直选单式勿塞按位号池（会被校验成「N 个单式组合不合法」）。
    // 真正出号看 hotColdWarm；这里写一注合法样例即可。
    if (!isHcwLhcGuoguan.value && !isHcwZuDual.value && !isHcwRenyiDuipeng.value) {
      const seg = schemePlayConfig.value.segmentLen
      if (
        !hcwSingleGroup.value &&
        isSscDanshiLikeConfig(schemePlayConfig.value) &&
        seg > 1
      ) {
        const sample = Array.from({ length: positionCount.value }, (_, i) => {
          const d = (hcwPools.value[i] ?? [])[0]
          return d != null && d !== '' ? String(d) : '0'
        }).join('')
        schemeGroups.value = [sample || '0'.repeat(Math.max(1, seg))]
      } else if (hcwDigitOverall.value && isZuxuanDanshiConfig(schemePlayConfig.value)) {
        // 组选单式冷热：预览号池（如 1,2,3）→ 两两组合整注（12,13,23），供保存校验与样例
        const pool = (hcwPools.value[0] ?? []).join(',')
        const digitLen =
          seg >= 2 ? seg : (schemePlayConfig.value.renPositionCount ?? 2)
        const expanded = normalizeZuxuanDanshiContent(pool, digitLen)
        if (!expanded) {
          await warn(`组选单式冷热：请至少勾选 ${digitLen} 个名次（号码将自动组合为投注整注）`)
          return
        }
        schemeGroups.value = [expanded]
      } else {
        // 复式/定位胆等：按位玩法存成单组多行（万\n千\n百）
        schemeGroups.value = hcwSingleGroup.value
          ? [(hcwPools.value[0] ?? []).join(',')]
          : [
              Array.from({ length: positionCount.value }, (_, i) => (hcwPools.value[i] ?? []).join(',')).join(
                '\n',
              ),
            ]
      }
      if (schemeUsesRenxuanRunPos.value && schemeGroups.value[0]) {
        schemeGroups.value = [wrapRenxuanRunContent(schemeGroups.value[0])]
      }
    }
  } else if (rt === 'random_draw') {
    ensureRdCounts()
    ensureRenxuanRunPositions()
    if (isRdLhcRenyiDuipeng.value) {
      const [aCount, bCount] = rdLhcRenyiDuipengCounts.value
      if (!rdWholePreview.value.length) generateRdPreview()
      const sample =
        rdWholePreview.value[0] || randomLhcRenyiDuipengContentForCounts(aCount, bCount)
      schemeGroups.value = [sample]
    } else if (isRdZuDual.value) {
      const d = Math.trunc(Number(rdCounts.value[0]) || 0)
      const minH = zuDualMinHeadCount()
      const minS = zuDualMinSinglesCount()
      const s = Math.trunc(Number(rdCounts.value[1]) || 0)
      const headLabel = zuDualZoneHeadLabel()
      const tailLabel = triggerZuDualMeta.value?.tailLabel ?? '单号'
      if (d < minH || d > 10) {
        await warn(`${headLabel}选码个数须为 ${minH}–10`)
        return
      }
      if (s < minS || s > 10) {
        await warn(`${tailLabel}选码个数须为 ${minS}–10`)
        return
      }
      if (triggerZuDualMeta.value?.equalCounts && d !== s) {
        await warn(`${headLabel}与${tailLabel}个数须相同`)
        return
      }
      if (!rdZuDualPreviewWire()) generateRdPreview()
      const sample = rdZuDualPreviewWire() || randomZuDualContent(d, s)
      schemeGroups.value = [schemeUsesRenxuanRunPos.value ? wrapRenxuanRunContent(sample) : sample]
    } else if (rdSingleCountMode.value) {
      if (!rdWholePreview.value.length) generateRdPreview()
      // 生肖/尾数/生尾对碰占位：A|B（勿逗号）
      const barDpRd =
        isLhcSxDuipengConfig(schemePlayConfig.value) ||
        isLhcWsDuipengConfig(schemePlayConfig.value) ||
        isLhcSwDuipengConfig(schemePlayConfig.value)
      let sample = (barDpRd ? rdWholePreview.value.join('|') : rdWholePreview.value.join(',')) || '0'
      // 和值/跨度随机样例偶发超限：占位改为组合更小的可行集，真实下注仍由引擎按 counts 重抽
      const cfgRd = schemePlayConfig.value
      const bmRd = String(cfgRd.betMode ?? '').toLowerCase()
      // 任选和值校验需带选位前缀
      const sampleForCap = schemeUsesRenxuanRunPos.value ? wrapRenxuanRunContent(sample) : sample
      if (
        (bmRd === 'hezhi' || bmRd === 'kuadu' || /和值|跨度/.test(cfgRd.playMethodLabel ?? '')) &&
        !validateGroupContent(cfgRd, sampleForCap).ok
      ) {
        const k = Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value)
        sample = greedyHezhiKuaduPicksUnderMax(cfgRd, k, rdAttributeUniverse()).join(',') || sample
        rdWholePreview.value = sample.split(',').filter(Boolean)
      }
      schemeGroups.value = [schemeUsesRenxuanRunPos.value ? wrapRenxuanRunContent(sample) : sample]
    } else {
      if (!rdPreview.value.length || rdPreview.value.every((row) => !row.length)) {
        generateRdPreview()
      }
      // 按位：单组多行（万\n千\n… / 十\n个），禁止拆成多个轮换组
      const perPosMax = rdPerPosMax.value
      const perPosDxds = isPerPosDxdsPlayConfig(schemePlayConfig.value)
      const fallbackUniverse = perPosDxds ? rdAttributeUniverse() : []
      const poolContent = Array.from({ length: positionCount.value }, (_, i) => {
        const prev = rdPreview.value[i] ?? []
        if (prev.length) return prev.join(',')
        const count = Math.min(perPosMax, Math.max(1, rdCounts.value[i] ?? 1))
        if (perPosDxds) {
          return fallbackUniverse.slice(0, count).join(',') || '大'
        }
        return Array.from({ length: count }, (_, j) => String(j % 10)).join(',')
      }).join('\n')
      // 直选单式：占位写成展开后的整注样例，避免保存校验把按位号池当非法单式；剔除豹子
      const seg = schemePlayConfig.value.segmentLen
      if (
        isZhixuanDanshiPerPosPlay(schemePlayConfig.value) &&
        seg > 1 &&
        isZhixuanPositionPoolContent(poolContent, seg)
      ) {
        const expanded =
          expandZhixuanPoolToDanshiWithoutBaozi(poolContent, seg) ||
          expandZhixuanPositionPoolToDanshi(poolContent, seg) ||
          poolContent
        schemeGroups.value = [expanded]
      } else {
        schemeGroups.value = [poolContent]
      }
      if (schemeUsesRenxuanRunPos.value && schemeGroups.value[0]) {
        schemeGroups.value = [wrapRenxuanRunContent(schemeGroups.value[0])]
      }
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
      const limitMsg = isBetLimitExceededMessage(groupCheck.message)
      if (!limitMsg) {
        for (const idx of groupCheck.invalidIndexes) {
          schemeGroups.value[idx] = ''
        }
      }
      await confirmDialog({
        title: limitMsg ? '提示' : '输入不合法',
        message: limitMsg
          ? groupCheck.message
          : `${groupCheck.message}。请按「${playModeSummary.value}」规则重新填写。`,
        tone: 'warning',
        confirmText: '我知道了',
        showCancel: false,
      })
      return
    }
    schemeGroups.value = groupCheck.normalized

    // 单次金额上限（对齐第三方 10 万）：注数×单位×倍数系数×模式最高倍率
    {
      const unit = Number(betUnit.value)
      const coef = Number(multCoeff.value.trim() || '1')
      const modeMax = maxModeMultiplierFromPayload(betMultiplierPayload.value)
      const mult = (Number.isFinite(coef) && coef > 0 ? coef : 1) * modeMax
      for (const raw of groupCheck.normalized) {
        const r = validateGroupContent(schemePlayConfig.value, raw)
        if (!r.ok || r.betUnits <= 0) continue
        const amount = calcBetAmount(r.betUnits, mult, Number.isFinite(unit) && unit > 0 ? unit : 2)
        if (betAmountExceedsMax(amount)) {
          await confirmDialog({
            title: '提示',
            message: maxBetAmountExceededMessage(schemeCurrency.value),
            tone: 'warning',
            confirmText: '我知道了',
            showCancel: false,
          })
          return
        }
      }
    }
  }

  // 高级定码轮换 / 开某投某 / 冷热 / 随机等未走 validateSchemeGroups 的入口：统一拦超注上限
  {
    const contents: string[] =
      rt === 'adv_fixed_rotate'
        ? jushuList.value.map((r) => r.content)
        : rt === 'adv_trigger_bet'
          ? triggerRows.value.flatMap((r) => {
              if (!r.enabled) return []
              const out: string[] = []
              if (String(r.pos ?? '').trim()) out.push(String(r.pos))
              if (String(r.neg ?? '').trim()) out.push(String(r.neg))
              return out
            })
          : rt === 'random_draw'
            ? [...schemeGroups.value]
            : rt === 'hot_cold_warm'
              ? // 冷热：单式占位已是整注样例；复式等仍验按位号池
                isSscDanshiLikeConfig(schemePlayConfig.value) && !hcwSingleGroup.value
                  ? []
                  : [...schemeGroups.value]
              : []
    for (const raw of contents) {
      if (!String(raw ?? '').trim()) continue
      // 任选开某投某：单式按位号池展开；号池/和值补选位前缀后再校验
      let toCheck = String(raw ?? '')
      if (rt === 'adv_trigger_bet' && isRenxuanTriggerPlay.value) {
        toCheck = showTriggerPerPosColumns.value
          ? renxuanTriggerPoolToGroupContent(toCheck)
          : buildRenxuanPositionContent(triggerBetPositionLabels.value, toCheck)
      }
      if (!toCheck.trim()) {
        await warn('选号无效')
        return
      }
      const r = validateGroupContent(schemePlayConfig.value, toCheck)
      if (!r.ok) {
        await warn(r.message)
        return
      }
      if (r.betUnits <= 0) {
        await warn(
          zuxuanPoolMinPick(schemePlayConfig.value) != null
            ? zuxuanPoolMinPickMessage(schemePlayConfig.value)
            : '选号无效',
        )
        return
      }
    }
  }

  // 直选单式 / 直选复式 / 混合组选：不得「单独只有」111/222/333 等豹子号（含冷热/局数等入口）
  // 高级开某投某 + 中三/前三混合组选：按位保存，下注时再排除豹子；任选混合组选已是整注，须拦豹子。
  if (!(rt === 'adv_trigger_bet' && isHunhePlayConfig(schemePlayConfig.value) && showTriggerPerPosColumns.value)) {
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
  // 取消自动草稿定时器，勿在此处再发一遍 PATCH：否则会与下方显式保存撞上
  // 写接口 1s 节流，提示「操作过于频繁」而非「已保存修改」。
  if (remotePersistTimer) {
    clearTimeout(remotePersistTimer)
    remotePersistTimer = null
  }
  if (isDraftScheme.value) {
    saveSchemeDraft(buildDraftSnapshot())
  }

  const cloudPayload = {
    kind: schemeKind.value,
    schemeName: name,
    lotteryCode: lottery,
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
      // 关闭节流：编辑时自动草稿可能刚写过同一载荷，显式「保存修改」必须放行
      let patch = buildRemoteDraftPatch()
      if (betMultiplierPayload.value) {
        const synced = await syncAdvancedTemplatesInPayload(schemeId.value, betMultiplierPayload.value)
        betMultiplierPayload.value = synced
        patch = {
          ...patch,
          betMultiplier: synced as unknown as Record<string, unknown>,
        }
      }
      await updateSchemeDefinition(schemeId.value, patch, { throttle: false })
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
          <div class="scf-field">
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
            内置计划需关联彩种，无需选择玩法；请在下方方案内容中选择同彩种的已收藏跟单方案
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
              <el-input
                id="scf-mult"
                :model-value="multCoeff"
                size="large"
                class="scf-el-inp"
                inputmode="numeric"
                maxlength="6"
                placeholder="正整数，最小 1"
                @update:model-value="onMultCoeffInput"
                @change="normalizeMultCoeff"
                @blur="normalizeMultCoeff"
              />
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
              <SchemeLhcTemaPanel
                v-else-if="schemeUsesLhcTemaPanel"
                v-model="schemeGroups[idx]"
                :config="schemePlayConfig"
              />
              <SchemeLhcRenyiDuipengPanel
                v-else-if="schemeUsesLhcRenyiDuipengPanel"
                v-model="schemeGroups[idx]"
                :config="schemePlayConfig"
              />
              <SchemeLhcGuoguanPanel
                v-else-if="schemeUsesLhcGuoguanPanel"
                v-model="schemeGroups[idx]"
                :config="schemePlayConfig"
              />
              <SchemeGroupInputPanel
                v-else-if="schemeUsesTextInputPanel"
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
                @blur="commitSchemeGroupAt(idx)"
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
          <p v-if="triggerSegmentOpenTip" class="scf-run-tip">{{ triggerSegmentOpenTip }}</p>
          <div
            v-if="showTriggerPositionPicker && isRenxuanTriggerPlay"
            class="scf-field scf-panel-field scf-trig-pos-field"
          >
            <span class="scf-lbl scf-lbl--with-help">
              <span>开奖选位</span>
              <el-popover
                placement="bottom"
                :width="280"
                trigger="click"
                :content="TRIGGER_OPEN_POS_HINT"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="开奖选位说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <div
              class="scf-trig-pos-chips"
              role="radiogroup"
              aria-label="开奖选位（只能选一个）"
              :style="{ '--scf-trig-pos-n': String(triggerPickerLabels.length || 5) }"
            >
              <button
                v-for="(label, idx) in triggerPickerLabels"
                :key="`trig-open-${idx}`"
                type="button"
                class="scf-trig-pos-chip"
                role="radio"
                :class="{ 'is-on': triggerOpenPositionIdx === idx }"
                :aria-checked="triggerOpenPositionIdx === idx"
                @click="selectTriggerOpenPosition(idx, $event)"
              >{{ label }}</button>
            </div>
          </div>
          <div
            v-if="showTriggerPositionPicker"
            class="scf-field scf-panel-field scf-trig-pos-field"
          >
            <span class="scf-lbl" :class="{ 'scf-lbl--with-help': isRenxuanTriggerPlay }">
              <span>{{ isRenxuanTriggerPlay ? '投注选位' : '选位' }}</span>
              <el-popover
                v-if="isRenxuanTriggerPlay"
                placement="bottom"
                :width="280"
                trigger="click"
                :content="triggerBetPosHint"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="投注选位说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <div
              class="scf-trig-pos-chips"
              role="group"
              :aria-label="
                isRenxuanTriggerPlay
                  ? `从万千百十个中勾选至少 ${triggerRenPosNeed} 个、最多 5 个投注位置`
                  : '投注位'
              "
              :style="{ '--scf-trig-pos-n': String(triggerPickerLabels.length || 5) }"
            >
              <button
                v-for="(label, idx) in triggerPickerLabels"
                :key="`trig-pos-${idx}`"
                type="button"
                class="scf-trig-pos-chip"
                :class="{ 'is-on': triggerPositionIdxs.includes(idx) }"
                :aria-pressed="triggerPositionIdxs.includes(idx)"
                @click="toggleTriggerPosition(idx, $event)"
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
                  :disabled="triggerRandomCount <= triggerRandomMin"
                  aria-label="减少随机出号"
                  @click="triggerRandomCount = Math.max(triggerRandomMin, triggerRandomCount - 1)"
                >
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">remove</span>
                </button>
                <el-input
                  v-model.number="triggerRandomCount"
                  type="number"
                  inputmode="numeric"
                  maxlength="2"
                  class="scf-stepper-input scf-stepper-input--narrow"
                  :min="triggerRandomMin"
                  :max="triggerRandomMax"
                  @change="triggerRandomCount = Math.min(triggerRandomMax, Math.max(triggerRandomMin, Math.trunc(Number(triggerRandomCount) || triggerRandomMin)))"
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
            :class="{
              'scf-trig-grid--posrow': showTriggerPerPosColumns || isTriggerZuDualInput,
              'scf-trig-grid--zu12-head': isTriggerZuDualInput,
            }"
            :aria-hidden="showTriggerTemaOpenHint ? undefined : true"
          >
            <span>启用</span>
            <span
              class="scf-trig-open-lbl"
              :class="{ 'scf-lbl--with-help': showTriggerTemaOpenHint }"
            >
              <span>开出</span>
              <el-popover
                v-if="showTriggerTemaOpenHint"
                placement="bottom"
                :width="260"
                trigger="click"
                :content="triggerOpenHintText"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="开出说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <template v-if="showTriggerPerPosColumns || isTriggerZuDualInput">
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
                v-for="(label, pIdx) in triggerColumnLabels"
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
                <span v-if="pIdx === 0" class="scf-trig-open">{{ formatWsDpTokenLabel(row.open) }}</span>
                <span v-else class="scf-trig-cell-placeholder" aria-hidden="true" />
                <span class="scf-trig-pos-name">{{ triggerPosName(label) }}</span>
                <template v-if="triggerPerPosTextBet">
                  <el-select
                    v-if="triggerPerPosTextSingle"
                    :model-value="triggerTextTokens(getTriggerFieldCell(row, 'pos', pIdx))[0] ?? ''"
                    size="small"
                    clearable
                    placeholder="正投"
                    :disabled="!row.enabled"
                    :aria-label="`${triggerPosName(label)}正投（开出 ${row.open}）`"
                    @update:model-value="(v: string) => setTriggerTextFieldCell(row, 'pos', pIdx, v ? [v] : [])"
                  >
                    <el-option v-for="v in triggerBetOptions" :key="`pp-${row.open}-${pIdx}-${v}`" :label="formatWsDpTokenLabel(v)" :value="v" />
                  </el-select>
                  <el-select
                    v-else
                    :model-value="triggerTextTokens(getTriggerFieldCell(row, 'pos', pIdx))"
                    size="small"
                    multiple
                    collapse-tags
                    collapse-tags-tooltip
                    placeholder="正投（可多选）"
                    :disabled="!row.enabled"
                    :aria-label="`${triggerPosName(label)}正投（开出 ${row.open}）`"
                    @update:model-value="(v: string[]) => setTriggerTextFieldCell(row, 'pos', pIdx, v)"
                  >
                    <el-option v-for="v in triggerBetOptions" :key="`pp-${row.open}-${pIdx}-${v}`" :label="formatWsDpTokenLabel(v)" :value="v" />
                  </el-select>
                  <el-select
                    v-if="triggerPerPosTextSingle"
                    :model-value="triggerTextTokens(getTriggerFieldCell(row, 'neg', pIdx))[0] ?? ''"
                    size="small"
                    clearable
                    placeholder="反投"
                    :disabled="!row.enabled"
                    :aria-label="`${triggerPosName(label)}反投（开出 ${row.open}）`"
                    @update:model-value="(v: string) => setTriggerTextFieldCell(row, 'neg', pIdx, v ? [v] : [])"
                  >
                    <el-option v-for="v in triggerBetOptions" :key="`pn-${row.open}-${pIdx}-${v}`" :label="formatWsDpTokenLabel(v)" :value="v" />
                  </el-select>
                  <el-select
                    v-else
                    :model-value="triggerTextTokens(getTriggerFieldCell(row, 'neg', pIdx))"
                    size="small"
                    multiple
                    collapse-tags
                    collapse-tags-tooltip
                    placeholder="反投（可多选）"
                    :disabled="!row.enabled"
                    :aria-label="`${triggerPosName(label)}反投（开出 ${row.open}）`"
                    @update:model-value="(v: string[]) => setTriggerTextFieldCell(row, 'neg', pIdx, v)"
                  >
                    <el-option v-for="v in triggerBetOptions" :key="`pn-${row.open}-${pIdx}-${v}`" :label="formatWsDpTokenLabel(v)" :value="v" />
                  </el-select>
                </template>
                <template v-else>
                  <el-input
                    :model-value="getTriggerFieldCell(row, 'pos', pIdx)"
                    size="small"
                    :placeholder="triggerInputPlaceholder"
                    :inputmode="isTriggerSinglePickBet ? 'numeric' : 'text'"
                    :maxlength="isTriggerSinglePickBet ? 1 : undefined"
                    :disabled="!row.enabled"
                    :aria-label="`${triggerPosName(label)}正投（开出 ${row.open}）`"
                    @update:model-value="(v: string | number) => writeTriggerFieldCell(row, 'pos', pIdx, isTriggerSinglePickBet ? sanitizeTriggerBetContent(String(v ?? '')) : String(v ?? ''))"
                    @change="commitTriggerFieldCell(row, 'pos', pIdx)"
                  />
                  <el-input
                    :model-value="getTriggerFieldCell(row, 'neg', pIdx)"
                    size="small"
                    :placeholder="triggerInputPlaceholder"
                    :inputmode="isTriggerSinglePickBet ? 'numeric' : 'text'"
                    :maxlength="isTriggerSinglePickBet ? 1 : undefined"
                    :disabled="!row.enabled"
                    :aria-label="`${triggerPosName(label)}反投（开出 ${row.open}）`"
                    @update:model-value="(v: string | number) => writeTriggerFieldCell(row, 'neg', pIdx, isTriggerSinglePickBet ? sanitizeTriggerBetContent(String(v ?? '')) : String(v ?? ''))"
                    @change="commitTriggerFieldCell(row, 'neg', pIdx)"
                  />
                </template>
              </div>
            </div>
          </template>
          <template v-else-if="isTriggerZuDualInput">
            <div
              v-for="row in triggerRows"
              :key="row.open"
              class="scf-trig-block scf-trig-block--zu12"
              :class="{ 'is-off': !row.enabled }"
            >
              <div
                v-for="(zLabel, zIdx) in triggerZuDualZoneLabels"
                :key="`trig-zu-dual-${row.open}-${zIdx}`"
                class="scf-trig-grid scf-trig-grid--posrow"
              >
                <el-switch
                  v-if="zIdx === 0"
                  v-model="row.enabled"
                  size="small"
                  :aria-label="`启用开出 ${row.open} 的映射`"
                />
                <span v-else class="scf-trig-cell-placeholder" aria-hidden="true" />
                <span v-if="zIdx === 0" class="scf-trig-open">{{ formatWsDpTokenLabel(row.open) }}</span>
                <span v-else class="scf-trig-cell-placeholder" aria-hidden="true" />
                <span class="scf-trig-pos-name">{{ zLabel }}</span>
                <el-input
                  :model-value="getZuDualTriggerZone(row.pos, zIdx as 0 | 1)"
                  size="small"
                  :placeholder="triggerZuDualZonePlaceholders[zIdx]"
                  inputmode="numeric"
                  :disabled="!row.enabled"
                  :aria-label="`正投${zLabel}（开出 ${row.open}）`"
                  @update:model-value="(v: string | number) => setZu12TriggerZone(row, 'pos', zIdx as 0 | 1, String(v ?? ''))"
                  @change="commitZu12TriggerField(row, 'pos')"
                />
                <el-input
                  :model-value="getZuDualTriggerZone(row.neg, zIdx as 0 | 1)"
                  size="small"
                  :placeholder="triggerZuDualZonePlaceholders[zIdx]"
                  inputmode="numeric"
                  :disabled="!row.enabled"
                  :aria-label="`反投${zLabel}（开出 ${row.open}）`"
                  @update:model-value="(v: string | number) => setZu12TriggerZone(row, 'neg', zIdx as 0 | 1, String(v ?? ''))"
                  @change="commitZu12TriggerField(row, 'neg')"
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
              <span class="scf-trig-open">{{ formatWsDpTokenLabel(row.open) }}</span>
              <template v-if="isTriggerTextPlay">
                <el-select
                  v-if="triggerTextSingle"
                  :model-value="triggerTextTokens(row.pos)[0] ?? ''"
                  size="small"
                  clearable
                  placeholder="正投"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string) => setTriggerTextField(row, 'pos', v ? [v] : [])"
                >
                  <el-option v-for="v in triggerBetOptions" :key="v" :label="formatWsDpTokenLabel(v)" :value="v" />
                </el-select>
                <el-select
                  v-else
                  :model-value="triggerTextTokens(row.pos)"
                  size="small"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="正投（可多选）"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string[]) => setTriggerTextField(row, 'pos', v)"
                >
                  <el-option v-for="v in triggerBetOptions" :key="v" :label="formatWsDpTokenLabel(v)" :value="v" />
                </el-select>
                <el-select
                  v-if="triggerTextSingle"
                  :model-value="triggerTextTokens(row.neg)[0] ?? ''"
                  size="small"
                  clearable
                  placeholder="反投"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string) => setTriggerTextField(row, 'neg', v ? [v] : [])"
                >
                  <el-option v-for="v in triggerBetOptions" :key="`neg-${v}`" :label="formatWsDpTokenLabel(v)" :value="v" />
                </el-select>
                <el-select
                  v-else
                  :model-value="triggerTextTokens(row.neg)"
                  size="small"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="反投（可多选）"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string[]) => setTriggerTextField(row, 'neg', v)"
                >
                  <el-option v-for="v in triggerBetOptions" :key="`neg-${v}`" :label="formatWsDpTokenLabel(v)" :value="v" />
                </el-select>
              </template>
              <template v-else>
                <el-input
                  :model-value="row.pos"
                  size="small"
                  :placeholder="isLhcGuoguanConfig(schemePlayConfig) ? '如 大,大,,大,,大' : triggerInputPlaceholder"
                  :inputmode="isTriggerSinglePickBet ? 'numeric' : 'text'"
                  :maxlength="isTriggerSinglePickBet ? 1 : undefined"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string | number) => { row.pos = isTriggerSinglePickBet ? sanitizeTriggerBetContent(String(v ?? '')) : String(v ?? '') }"
                  @change="isLhcGuoguanConfig(schemePlayConfig) ? (row.pos = normalizeLhcGuoguanTriggerContent(row.pos)) : isLhcTemaPlayConfig(schemePlayConfig) ? commitTriggerTemaField(row, 'pos') : (row.pos = sanitizeTriggerBetContent(row.pos))"
                />
                <el-input
                  :model-value="row.neg"
                  size="small"
                  :placeholder="isLhcGuoguanConfig(schemePlayConfig) ? '如 大,大,,大,,大' : triggerInputPlaceholder"
                  :inputmode="isTriggerSinglePickBet ? 'numeric' : 'text'"
                  :maxlength="isTriggerSinglePickBet ? 1 : undefined"
                  :disabled="!row.enabled"
                  @update:model-value="(v: string | number) => { row.neg = isTriggerSinglePickBet ? sanitizeTriggerBetContent(String(v ?? '')) : String(v ?? '') }"
                  @change="isLhcGuoguanConfig(schemePlayConfig) ? (row.neg = normalizeLhcGuoguanTriggerContent(row.neg)) : isLhcTemaPlayConfig(schemePlayConfig) ? commitTriggerTemaField(row, 'neg') : (row.neg = sanitizeTriggerBetContent(row.neg))"
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
          <div
            v-if="showHcwOpenPosition"
            class="scf-field scf-panel-field scf-trig-pos-field"
          >
            <span class="scf-lbl scf-lbl--with-help">
              <span>开奖选位</span>
              <el-popover
                placement="bottom"
                :width="280"
                trigger="click"
                :content="hcwOpenPosHint"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="开奖选位说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <div
              class="scf-trig-pos-chips"
              role="group"
              :aria-label="`开奖选位须选 ${renxuanRunPosNeed} 个`"
              style="--scf-trig-pos-n: 5"
            >
              <button
                v-for="(label, idx) in SSC_POSITION_LABELS"
                :key="`hcw-open-${idx}`"
                type="button"
                class="scf-trig-pos-chip"
                :class="{ 'is-on': hcwOpenPosIdxs.includes(idx) }"
                :aria-pressed="hcwOpenPosIdxs.includes(idx)"
                @click="toggleHcwOpenPosition(idx, $event)"
              >{{ label }}</button>
            </div>
          </div>
          <div
            v-if="showHcwOpenPosition || schemeUsesRenxuanRunPos"
            class="scf-field scf-panel-field scf-trig-pos-field"
          >
            <span
              class="scf-lbl"
              :class="{ 'scf-lbl--with-help': showHcwOpenPosition || isRenxuanHcwZuDual }"
            >
              <span>投注选位</span>
              <el-popover
                v-if="showHcwOpenPosition || isRenxuanHcwZuDual"
                placement="bottom"
                :width="280"
                trigger="click"
                :content="hcwBetPosHint"
                popper-class="scf-help-popper"
              >
                <template #reference>
                  <button type="button" class="scf-help-btn" aria-label="投注选位说明" @click.stop>
                    <span class="scf-ms scf-ms--help" aria-hidden="true">help</span>
                  </button>
                </template>
              </el-popover>
            </span>
            <div
              class="scf-trig-pos-chips"
              role="group"
              :aria-label="`投注选位至少选 ${renxuanRunPosNeed} 个、最多 5 个`"
              style="--scf-trig-pos-n: 5"
            >
              <button
                v-for="(label, idx) in SSC_POSITION_LABELS"
                :key="`hcw-bet-${idx}`"
                type="button"
                class="scf-trig-pos-chip"
                :class="{ 'is-on': renxuanRunPosIdxs.includes(idx) }"
                :aria-pressed="renxuanRunPosIdxs.includes(idx)"
                @click="toggleHcwBetPosition(idx, $event)"
              >{{ label }}</button>
            </div>
          </div>
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
                @click="scheduleHcwStats(true)"
              >
                <span
                  class="scf-ms scf-ms--sm"
                  :class="{ 'scf-hcw-refresh-spin': hcwLoading }"
                  aria-hidden="true"
                >refresh</span>
              </button>
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
          <div
            v-for="(label, pi) in hcwGroupLabels"
            :key="`hcw-${pi}-${isHcwZuDual ? (schemeUsesRenxuanRunPos ? renxuanRunPosIdxs.join('_') : hcwFixedSegmentBallIdxs().join('_')) : hcwOpenPosIdxs.join('_')}-${hcwStatsGen}`"
            class="scf-hcw-pos"
          >
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
              {{ hcwAttribute ? '暂无选项频次，可点刷新重试' : '暂无开奖统计；热/冷/全勾选名次，就绪后按当前排名预览号码' }}
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
                :key="`${cell.token}-${cell.count}-${cell.tier}-${hcwStatsGen}`"
                type="button"
                class="scf-hcw-cell"
                :class="{
                  'is-hot': cell.tier === 'hot',
                  'is-cold': cell.tier === 'cold',
                  'is-on': poolHasToken(hcwPools[pi], cell.token),
                }"
                @click="toggleHcwDigit(pi, cell.token)"
              >
                <span class="scf-hcw-cell-num">{{ formatWsDpTokenLabel(cell.token) }}</span>
                <span class="scf-hcw-cell-cnt">{{ cell.count == null ? '—' : cell.count }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 6. 随机出号 -->
        <div v-else-if="runTypeId === 'random_draw'" class="scf-content-card scf-panel">
          <div
            v-if="schemeUsesRenxuanRunPos"
            class="scf-field scf-panel-field scf-trig-pos-field"
          >
            <span class="scf-lbl">投注选位</span>
            <div
              class="scf-trig-pos-chips"
              role="group"
              :aria-label="`投注选位至少选 ${renxuanRunPosNeed} 个、最多 5 个`"
              style="--scf-trig-pos-n: 5"
            >
              <button
                v-for="(label, idx) in SSC_POSITION_LABELS"
                :key="`rd-bet-${idx}`"
                type="button"
                class="scf-trig-pos-chip"
                :class="{ 'is-on': renxuanRunPosIdxs.includes(idx) }"
                :aria-pressed="renxuanRunPosIdxs.includes(idx)"
                @click="toggleHcwBetPosition(idx, $event)"
              >{{ label }}</button>
            </div>
          </div>
          <!-- 任选二全中任意对碰：A/B 两区独立配置，合计最多 10 个号 -->
          <div v-if="isRdLhcRenyiDuipeng" class="scf-rd-pos-grid">
            <div class="scf-rd-row">
              <span class="scf-rd-pos">A区随机</span>
              <el-input-number
                v-model="rdCounts[0]"
                :min="1"
                :max="rdLhcRenyiDuipengAMax"
                size="small"
                @change="ensureRdCounts"
              />
            </div>
            <div class="scf-rd-row">
              <span class="scf-rd-pos">B区随机</span>
              <el-input-number
                v-model="rdCounts[1]"
                :min="1"
                :max="rdLhcRenyiDuipengBMax"
                size="small"
                @change="ensureRdCounts"
              />
            </div>
          </div>
          <!-- 双区组选：头区 + 尾区两个选码个数 -->
          <div v-else-if="isRdZuDual" class="scf-rd-pos-grid">
            <div class="scf-rd-row">
              <span class="scf-rd-pos">{{ zuDualZoneHeadLabel() }}</span>
              <el-input-number v-model="rdCounts[0]" :min="zuDualMinHeadCount()" :max="10" size="small" />
            </div>
            <div class="scf-rd-row">
              <span class="scf-rd-pos">{{ triggerZuDualMeta?.tailLabel ?? '单号' }}</span>
              <el-input-number
                v-model="rdCounts[1]"
                :min="zuDualMinSinglesCount()"
                :max="10"
                size="small"
              />
            </div>
          </div>
          <!-- 单式整注随机 / 组选号码池随机：仅需单一数量 -->
          <template v-else-if="rdSingleCountMode">
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
            <div class="scf-rd-toolbar-summary">
              <el-button type="primary" plain size="small" @click="generateRdPreview">生成预览</el-button>
              <span class="scf-rd-units">预估 {{ rdEstimatedUnits }} 注</span>
            </div>
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
          <div v-if="favoritesLoading" class="scf-run-tip" style="padding: 1rem 0">正在加载收藏方案…</div>
          <el-empty
            v-else-if="favoritesLoadError"
            :description="favoritesLoadError"
            :image-size="64"
          >
            <el-button type="primary" @click="loadFavorites(true)">重新加载</el-button>
          </el-empty>
          <el-empty
            v-else-if="!lotteryCode.trim()"
            description="请先选择彩种，再选择同彩种的收藏方案"
            :image-size="64"
          />
          <el-empty
            v-else-if="favoritesLoaded && !favorites.length"
            description="暂无收藏方案，先去跟单大厅收藏方案"
            :image-size="64"
          />
          <el-empty
            v-else-if="favoritesLoaded && !favoritesForLottery.length"
            description="当前彩种下暂无收藏方案，请切换彩种或去跟单大厅收藏"
            :image-size="64"
          />
          <template v-else>
            <div class="scf-bp-list">
              <button
                v-for="f in favoritesForLottery"
                :key="f.snapshotId"
                type="button"
                class="scf-bp-item"
                :class="{ 'is-sel': favSelectedSnapshotId === f.snapshotId }"
                :disabled="builtinApplying"
                @click="selectBuiltinPlan(f.snapshotId)"
              >
                <span
                  class="scf-bp-radio"
                  :class="{ 'is-on': favSelectedSnapshotId === f.snapshotId }"
                  aria-hidden="true"
                />
                <span class="scf-bp-info">
                  <span class="scf-bp-name">{{ f.schemeName }}</span>
                  <span class="scf-bp-meta">
                    {{ f.lotteryLabel }} · {{ f.playMethod }} · 收藏于 {{ formatFavoredAt(f.favoredAt) }}
                  </span>
                </span>
              </button>
            </div>
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
          <SchemeLhcTemaPanel
            v-else-if="schemeUsesLhcTemaPanel"
            v-model="jushuForm.content"
            :config="schemePlayConfig"
          />
          <SchemeLhcRenyiDuipengPanel
            v-else-if="schemeUsesLhcRenyiDuipengPanel"
            v-model="jushuForm.content"
            :config="schemePlayConfig"
          />
          <SchemeLhcGuoguanPanel
            v-else-if="schemeUsesLhcGuoguanPanel"
            v-model="jushuForm.content"
            :config="schemePlayConfig"
          />
          <SchemeGroupInputPanel
            v-else-if="schemeUsesTextInputPanel"
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
            @blur="commitJushuFormContent"
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

.scf-trig-pos-field + .scf-trig-pos-field {
  margin-top: -0.4rem;
}

.scf-trig-pos-chips {
  --scf-trig-pos-n: 5;
  display: grid;
  grid-template-columns: repeat(var(--scf-trig-pos-n), minmax(0, 1fr));
  gap: 0.25rem;
  width: 100%;
  min-width: 0;
  padding: 0.15rem;
  border-radius: 0.5rem;
  background: rgba(242, 244, 246, 0.85);
}

.scf-trig-pos-chip {
  height: 1.55rem;
  margin: 0;
  padding: 0 0.15rem;
  border: none;
  border-radius: 0.4rem;
  font-size: 0.75rem;
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

/* 悬停勿用主色字+白底，否则取消后鼠标仍在按钮上会像「浅选中」 */
@media (hover: hover) {
  .scf-trig-pos-chip:hover:not(.is-on) {
    background: rgba(25, 28, 30, 0.04);
    color: var(--scf-on-variant);
  }
}

.scf-trig-pos-chip.is-on {
  background: #fff;
  color: var(--el-color-primary, #0050cb);
  box-shadow: 0 2px 10px rgba(25, 28, 30, 0.08);
}

.scf-trig-pos-chip:focus:not(:focus-visible) {
  outline: none;
}

.scf-trig-pos-chip:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.35);
  outline-offset: 1px;
}

.scf-trig-pos-chip:active:not(.is-on) {
  background: transparent;
  color: var(--scf-on-variant);
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

/* 组选12：位置列放「二重号/单号」，略加宽 */
.scf-trig-block--zu12 .scf-trig-grid--posrow,
.scf-trig-grid--zu12-head {
  grid-template-columns: 2.1rem 1.75rem 3rem 1fr 1fr;
}

.scf-trig-block {
  display: flex;
  flex-direction: column;
  gap: 0.28rem;
  padding: 0.28rem 0 0.08rem;
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

.scf-trig-open-lbl {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.1rem;
  min-width: 0;
}

.scf-trig-grid--head .scf-trig-open-lbl > span {
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
  flex-direction: column;
  align-items: stretch;
  gap: 0.55rem;
  margin: 0.65rem 0 0.75rem;
  width: 100%;
}

.scf-rd-toolbar-summary {
  display: flex;
  align-items: center;
  gap: 0.85rem;
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
  width: 100%;
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
