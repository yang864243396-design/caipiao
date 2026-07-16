<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { formatClientApiError } from '@/utils/guajiError'
import { redirectToGuajiAuthIfNeeded } from '@/composables/useGuajiAuthGuard'
import {
  contraryBet,
  shareAddToCloud,
  shareFollowBet,
} from '@/api/schemes/shareAddToCloud'
import type { BetMultiplierPayload } from '@/api/schemes/betMultiplier'
import {
  loadPlayDetailShareDock,
  savePlayDetailShareDock,
} from '@/utils/playDetailShareDock'
import {
  addSchemeFavorite,
  fetchSchemeFavorites,
  removeSchemeFavorite,
} from '@/api/schemes/favorites'
import {
  gameDetailCountdownDisplayFields,
  gameDetailCountdownText,
  mergeGameDetailCountdownOnPoll,
  SCHEME_COUNTDOWN_WAITING_LABEL,
  thirdPartyPeriodDisplay,
} from '@/api/cloud/center'
import {
  fetchGameDetail,
  fetchGameDraws,
  placeGameBet,
  type GameBetRecordDto,
  type GameBettingRow,
  type GamePlanTrendChartPoint,
  type GamePlanTrendRow,
} from '@/api/games/detail'
import { fetchPlayTree } from '@/api/games/lotteries'
import { fetchGuajiBalance } from '@/api/guaji/accounts'
import type { PlayTreeResponse } from '@/types/playCatalog'
import {
  buildGameBetPayload,
  buildGroupContent,
  buildRenxuanPositionContent,
  countBetUnits,
  defaultRenxuanPositions,
  isRenxuanPositionDanshiConfig,
  resolvePlayConfig,
  seedDigitsFromNumbers,
  type PlayConfig,
} from '@/utils/betPayload'
import {
  digitOptionsForConfig,
  poolMaxPicksForConfig,
  textPickOptionsForConfig,
  togglePoolPick,
} from '@/utils/pickPanelOptions'
import SchemeRenxuanDanshiPanel from '@/components/schemes/SchemeRenxuanDanshiPanel.vue'
import {
  defaultPlaySelection,
  findSubPlay,
  formatSubPlayLabel,
  resolvePlayConfigFromTree,
  type PlayTreePlayConfig,
} from '@/utils/playConfig'
import { supportsPlanContraryPlay } from '@/utils/planContrary'
import {
  LHC_NUMBERS,
  LHC_TAIL_OPTIONS,
  LHC_ZODIACS,
  lhcAttrOptions,
} from '@/constants/lhcPlay'
import { BET_MODE_OPTIONS } from '@/constants/betModeOptions'
import { demoAppBrand } from '@/demo/demoAccount'
import { startGameDrawSync } from '@/composables/useGameDrawSync'
import type { WsDrawResultPayload } from '@shared/types/ws'

const route = useRoute()
const router = useRouter()

const ICON_BACK = '/images/lobby/icon-back.png'
const ICON_TIMER = '/images/lobby/icon-timer.png'
const ICON_DRAG = '/images/lobby/icon-drag-handle.png'

function isLhcLotteryCode(code: string): boolean {
  return code.includes('lhc')
}

const pageTitle = computed(() => {
  const raw = route.query.scheme as string | undefined
  if (raw) return decodeURIComponent(raw)
  return isLhcLotteryCode(lotteryCode.value) ? '特码方案 - 特码A' : '禄螭万位 - 定位胆万位'
})

const snapshotId = computed(() => String(route.query.snapshotId ?? '').trim())
const lotteryCode = computed(() => String(route.query.lotteryCode ?? 'tron_ffc_1m').trim())
const playMethodQuery = computed(() => String(route.query.playMethod ?? '').trim())
const playTypeId = computed(() =>
  String(route.query.typeId ?? route.query.playTypeId ?? '').trim(),
)
const subPlayId = computed(() =>
  String(route.query.subId ?? route.query.subPlayId ?? '').trim(),
)

const playTree = ref<PlayTreeResponse | null>(null)
const playTreeLoading = ref(false)
const selectedTypeId = ref('')
const selectedSubId = ref('')

const activePlaySelection = computed(() => {
  if (!playTree.value) return null
  const typeId = selectedTypeId.value
  const subId = selectedSubId.value
  if (!typeId || !subId) return null
  return findSubPlay(playTree.value, typeId, subId)
})

const playMethod = computed(() => {
  if (activePlaySelection.value) return activePlaySelection.value.subNode.label
  if (playMethodQuery.value) return playMethodQuery.value
  return isLhcLotteryCode(lotteryCode.value) ? '特码A' : '一星定位胆 · 万位'
})

const playConfig = computed((): PlayConfig | PlayTreePlayConfig => {
  const sel = activePlaySelection.value
  if (sel && playTree.value) {
    return resolvePlayConfigFromTree(
      playTree.value.playTemplate,
      sel.typeNode,
      sel.subNode,
    )
  }
  return resolvePlayConfig({
    playMethod: playMethod.value,
    playTypeId: playTypeId.value || undefined,
    subPlayId: subPlayId.value || undefined,
  })
})

const activePlayTypes = computed(() => playTree.value?.playTypes ?? [])
const activeSubPlays = computed(() => {
  const typeId = selectedTypeId.value
  if (!typeId || !playTree.value) return []
  return playTree.value.playTypes.find((t) => t.typeId === typeId)?.subPlays ?? []
})

/** 跟单大厅等带 snapshot / 玩法 ID 进入时，投注 Tab 不展示完整玩法树 */
const isPlaySelectionLocked = computed(
  () =>
    Boolean(snapshotId.value) ||
    (Boolean(playTypeId.value) && Boolean(subPlayId.value)),
)
const showBetTabPlayPicker = computed(
  () => !isPlaySelectionLocked.value && activePlayTypes.value.length > 0,
)

const pickDigits = ref<string[]>(['1', '3', '7'])
const pickLines = ref<string[][]>([])
const danshiInput = ref('')
const renxuanDanshiContent = ref('')
const usesRenxuanDanshi = computed(() => isRenxuanPositionDanshiConfig(playConfig.value))

const isLhcTemplate = computed(() => playTree.value?.playTemplate === 'lhc_std')

const lhcDanshiPlaceholder = computed(() => {
  const mode = playConfig.value.betMode ?? ''
  if (mode === 'tuotou') return '拖头：胆码|拖码，如 01,02|03,04,05'
  if (mode.endsWith('_dp')) return '对碰：A组|B组，生肖如 马|龙 或号码如 01,02|03,04'
  if (mode === 'guoguan') return '过关：大,单,双 等，逗号分隔'
  if (isLhcTemplate.value) return '输入选号，逗号分隔'
  return '每行或逗号分隔，如 392,123'
})

const lhcPickOptions = computed((): readonly string[] => {
  const cfg = playConfig.value
  if (cfg.inputMode === 'lhc_zodiac') return LHC_ZODIACS
  if (cfg.inputMode === 'lhc_tail') return LHC_TAIL_OPTIONS
  if (cfg.inputMode === 'lhc_attr') {
    return lhcAttrOptions(cfg.betMode ?? '', 'lhc_attr')
  }
  return LHC_NUMBERS
})

function initManualPicks(cfg: PlayConfig = playConfig.value) {
  if (snapshotId.value) return
  if (cfg.inputMode === 'multiline') {
    pickLines.value = cfg.segmentLabels.map(() => ['0'])
    return
  }
  if (cfg.inputMode === 'danshi') {
    if (isRenxuanPositionDanshiConfig(cfg)) {
      const k = cfg.renPositionCount ?? 2
      const n = cfg.segmentLen > 0 ? cfg.segmentLen : k
      const sample = Array.from({ length: n }, (_, i) => String((i + 1) % 10)).join('')
      renxuanDanshiContent.value = buildRenxuanPositionContent(
        defaultRenxuanPositions(k),
        sample,
      )
      return
    }
    danshiInput.value = isLhcTemplate.value ? '大,单' : '0'.repeat(cfg.segmentLen)
    return
  }
  if (cfg.inputMode === 'lhc_num') {
    pickDigits.value = ['01', '13', '25']
    return
  }
  if (cfg.inputMode === 'lhc_zodiac') {
    pickDigits.value = ['马', '龙']
    return
  }
  if (cfg.inputMode === 'lhc_tail') {
    pickDigits.value = ['0', '5']
    return
  }
  if (cfg.inputMode === 'lhc_attr') {
    const opts = lhcAttrOptions(cfg.betMode ?? '', 'lhc_attr')
    pickDigits.value = opts.length ? [opts[0]!] : ['红']
    return
  }
  const opts = textPickOptions.value
  if (opts.length) {
    pickDigits.value = [opts[0]!]
    danshiInput.value = opts[0]!
    return
  }
  const maxPicks = poolMaxPicksForConfig(cfg)
  const seed = bettingRows.value[0]?.numbers
  if (seed) {
    const fromSeed = seedDigitsFromNumbers(seed)
    pickDigits.value =
      maxPicks != null && maxPicks > 0 ? fromSeed.slice(0, maxPicks) : fromSeed
    return
  }
  if (maxPicks === 1) {
    pickDigits.value = [digitOptions.value[1] ?? digitOptions.value[0] ?? '1']
    return
  }
  const defaults = digitOptions.value.slice(1, 4)
  pickDigits.value = defaults.length ? defaults : ['1', '3', '7']
}

function togglePickDigit(d: string) {
  pickDigits.value = togglePoolPick(
    pickDigits.value,
    d,
    poolMaxPicksForConfig(playConfig.value),
  )
}

function toggleLineDigit(lineIndex: number, d: string) {
  const lines = pickLines.value.map((line) => [...line])
  while (lines.length < playConfig.value.segmentLen) {
    lines.push([])
  }
  const line = new Set(lines[lineIndex] ?? [])
  if (line.has(d)) line.delete(d)
  else line.add(d)
  lines[lineIndex] = [...line].sort()
  pickLines.value = lines
}

function isLineDigitSelected(lineIndex: number, d: string) {
  return (pickLines.value[lineIndex] ?? []).includes(d)
}

const manualGroupContent = computed(() => {
  if (usesRenxuanDanshi.value) return renxuanDanshiContent.value
  return buildGroupContent(playConfig.value, {
    digits: pickDigits.value,
    lines: pickLines.value,
    danshi: danshiInput.value,
  })
})

const digitOptions = computed(() => digitOptionsForConfig(playConfig.value))
const textPickOptions = computed(() => textPickOptionsForConfig(playConfig.value))
const actionLoading = ref(false)
const detailLoading = ref(false)

const currentIssue = ref('')
const nextIssue = ref('')
const countdownSec = ref(0)
const countdownEndTime = ref('')
const countdownPeriod = ref('')
const countdownLabel = ref('')
const countdownDisplay = computed(() =>
  gameDetailCountdownText({
    countdownEndTime: countdownEndTime.value,
    countdownPeriod: countdownPeriod.value,
    lotteryCode: lotteryCode.value,
    countdownSec: countdownSec.value,
    countdownLabel: countdownLabel.value,
  }),
)

/** 顶部开奖区期号展示（去掉第三方前缀 101 等，保留完整期号供下注） */
const displayCurrentIssue = computed(() => thirdPartyPeriodDisplay(currentIssue.value))
const displayNextIssue = computed(() => thirdPartyPeriodDisplay(nextIssue.value))

function schemeNameFromRoute(): string {
  const raw = route.query.scheme as string | undefined
  if (!raw) return ''
  try {
    return decodeURIComponent(raw).split(' - ')[0]?.trim() ?? ''
  } catch {
    return String(raw).split(' - ')[0]?.trim() ?? ''
  }
}

watch(
  pageTitle,
  (t) => {
    document.title = `游戏详情 · ${t} · ${demoAppBrand}`
  },
  { immediate: true }
)

/** Tab 值与 el-radio-button :value 同型（字符串），避免 Android WebView 原生 radio 值类型不一致 */
type DetailTabId = '0' | '1' | '2' | '3' | '4'
const tab = ref<DetailTabId>('0')
const DETAIL_TABS: readonly { id: DetailTabId; label: string; contraryOnly?: boolean }[] = [
  { id: '0', label: '投注' },
  { id: '1', label: '计划反集', contraryOnly: true },
  { id: '2', label: '计划走势' },
  { id: '3', label: '历史开奖' },
  { id: '4', label: '投注记录' },
] as const

/** 详情接口：方案玩法是否支持反集；无号码时不展示 Tab */
const planContrarySupportedFromApi = ref(true)
const showPlanContraryTab = computed(() => {
  const fromPlay = supportsPlanContraryPlay(playConfig.value)
  if (isPlaySelectionLocked.value) {
    // 跟单快照：接口判定支持且已有反集号码才显示（算不出则隐藏，避免空态页）
    return (
      planContrarySupportedFromApi.value
      && fromPlay
      && planInverseDigits.value.trim() !== ''
    )
  }
  return fromPlay
})
const visibleDetailTabs = computed(() =>
  DETAIL_TABS.filter((t) => !t.contraryOnly || showPlanContraryTab.value),
)
const activeTabLabel = computed(
  () => DETAIL_TABS.find((t) => t.id === tab.value)?.label ?? '',
)

/** 计划反集：号码串与注数（由详情接口 planInverseDigits / planInverseBetCount 填充） */
const planInverseDigits = ref('')
const planInverseBetCount = ref(0)

/** 快照方案投注区推演（由详情接口 scheme* 字段填充） */
const schemeBetUnit = ref(0)
const schemeBetMultiplier = ref(1)
const schemeBetUnits = ref(0)
const schemeContraryBetUnits = ref(0)
const schemePickDigits = ref('')
const estimatedPrize = ref(0)
const contraryEstimatedPrize = ref(0)

/** 计划走势：走势注数、折线点、期号横轴、近期中挂（由详情接口填充） */
const PLAN_TREND_HISTORY_INITIAL = 20
const PLAN_TREND_HISTORY_MAX = 100
const planTrendGroupBets = ref(0)
const planTrendChartPoints = ref<GamePlanTrendChartPoint[]>([])
const planTrendHistoryRows = ref<GamePlanTrendRow[]>([])
const planTrendHistoryVisibleCount = ref(PLAN_TREND_HISTORY_INITIAL)

function buildPlanTrendChartView(points: readonly GamePlanTrendChartPoint[]) {
  if (!points.length) {
    return {
      lineD: '',
      areaD: '',
      dots: [] as { left: number; top: number; hit: boolean }[],
      xLabels: [] as { text: string; show: boolean }[],
      yTicks: [] as { label: string; top: number }[],
    }
  }
  const scores = points.map((p) => p.round)
  let min = Math.min(...scores)
  let max = Math.max(...scores)
  if (min === max) {
    min -= 1
    max += 1
  }
  const domainMin = Math.floor(min)
  const domainMax = Math.ceil(max)
  const domainRange = domainMax - domainMin || 1

  const valueToTop = (value: number) =>
    ((domainMax - value) / domainRange) * 100

  const step = Math.max(1, Math.ceil(domainRange / 5))
  const tickValues: number[] = []
  for (let v = domainMax; v >= domainMin; v -= step) {
    tickValues.push(v)
  }
  if (tickValues[tickValues.length - 1] !== domainMin) {
    tickValues.push(domainMin)
  }

  const yTicks = tickValues.map((value) => ({
    label: String(value),
    top: valueToTop(value),
  }))

  const dots = points.map((p, i) => {
    const left = points.length <= 1 ? 50 : (i / (points.length - 1)) * 100
    const top = valueToTop(p.round)
    return { left, top, hit: p.win }
  })
  const lineCoords = dots.map((d) => `${d.left} ${d.top}`)
  const lineD = `M ${lineCoords.join(' L ')}`
  const areaD = `${lineD} V 100 H 0 Z`
  const xLabels = points.map((p, i) => ({
    text: p.period,
    show: i % 2 === 0 || points.length <= 8,
  }))
  return { lineD, areaD, dots, xLabels, yTicks }
}

const planTrendChartView = computed(() => buildPlanTrendChartView(planTrendChartPoints.value))

/** 历史开奖：子 Tab 与列表 */
const historySubTabLabels = ['号码', '大小', '单双', '龙虎', '总和'] as const
type HistorySubTabId = '0' | '1' | '2' | '3' | '4'
const historySubTab = ref<HistorySubTabId>('0')

const historyGameTag = ref('')

interface HistoryDrawRecord {
  periodShort: string
  time: string
  balls: readonly string[]
  sum: number
}

const historyDrawRecords = ref<HistoryDrawRecord[]>([])
const historyDrawsLoading = ref(false)

const HISTORY_DT_LABELS = ['万千', '万百', '万十', '万个', '千百', '千十', '千个', '百十', '百个', '十个'] as const

function formatHistoryDate(time: string) {
  const t = time.trim()
  const sp = t.indexOf(' ')
  return sp > 0 ? t.slice(0, sp) : t.slice(0, 10)
}

function historyDigitsFromBalls(balls: readonly string[]) {
  return balls.map((b) => parseInt(b, 10))
}

function historyDragonTigerCells(digits: readonly number[]) {
  const d = digits
  const pairs: [number, number][] = [
    [d[0], d[1]],
    [d[0], d[2]],
    [d[0], d[3]],
    [d[0], d[4]],
    [d[1], d[2]],
    [d[1], d[3]],
    [d[1], d[4]],
    [d[2], d[3]],
    [d[2], d[4]],
    [d[3], d[4]],
  ]
  return pairs.map(([a, b], i) => {
    if (a > b) return { kind: 'dragon' as const, char: '龙' as const, label: HISTORY_DT_LABELS[i] }
    if (a < b) return { kind: 'tiger' as const, char: '虎' as const, label: HISTORY_DT_LABELS[i] }
    return { kind: 'tie' as const, char: '和' as const, label: HISTORY_DT_LABELS[i] }
  })
}

function historyBigSmallDigit(ball: string): '大' | '小' {
  const n = parseInt(ball, 10)
  return Number.isFinite(n) && n >= 5 ? '大' : '小'
}

function historyParityDigit(ball: string): '单' | '双' {
  const n = parseInt(ball, 10)
  return Number.isFinite(n) && n % 2 === 1 ? '单' : '双'
}

const betMultiplier = ref(1)
const betMode = ref('2')

/** 倍投设定 Tab 与中文名称（与 BetMultiplierSettingsView 一致） */
const BET_MULTIPLIER_KIND_LABELS: Record<string, string> = {
  '0': '小白倍投',
  '1': '一键倍投',
  '2': '简单倍投',
  '3': '高级倍投',
}
const betMultiplierKind = ref<'' | '0' | '1' | '2' | '3'>('')
const betMultiplierPayload = ref<BetMultiplierPayload | undefined>()
const betMultiplierError = ref('')
const betMultiplierSelectedLabel = computed(() =>
  betMultiplierKind.value ? (BET_MULTIPLIER_KIND_LABELS[betMultiplierKind.value] ?? '') : '',
)

/** 投注区面板展开（把手点击切换） */
const betDockOpen = ref(true)

/** 投注区入口：手动下注（手机） / 云端挂机 */
const betDockEntryMode = ref<'manual' | 'cloud'>('manual')

const dockConfirmLabel = computed(() => {
  if (tab.value === '1') return '确认投注'
  if (betDockEntryMode.value === 'cloud') return '添加至云端'
  return '确认投注'
})

function toggleBetDockEntryMode() {
  betDockEntryMode.value = betDockEntryMode.value === 'manual' ? 'cloud' : 'manual'
  persistShareDockState()
}

function shareDockStorageKey() {
  return snapshotId.value || '__no_snapshot__'
}

function persistShareDockState() {
  savePlayDetailShareDock(shareDockStorageKey(), {
    entryMode: betDockEntryMode.value,
    betMultiplierKind: betMultiplierKind.value,
    betMultiplier: betMultiplierPayload.value,
  })
}

function loadShareDockBetMultiplier() {
  const dock = loadPlayDetailShareDock(shareDockStorageKey())
  if (!dock) return
  if (dock.entryMode === 'manual' || dock.entryMode === 'cloud') {
    betDockEntryMode.value = dock.entryMode
  }
  if (dock.betMultiplierKind) betMultiplierKind.value = dock.betMultiplierKind
  if (dock.betMultiplier) betMultiplierPayload.value = dock.betMultiplier
}

function buildPlayDetailRouteQuery(): Record<string, string> {
  const q: Record<string, string> = {}
  for (const [key, val] of Object.entries(route.query)) {
    const v = Array.isArray(val) ? val[0] : val
    if (v != null && String(v) !== '') q[key] = String(v)
  }
  return q
}

function stripBetMultiplierRouteQuery() {
  const q = buildPlayDetailRouteQuery()
  delete q.bmsKind
  delete q.bmsError
  void router.replace({ query: q })
}

function applyBetMultiplierFromRoute() {
  const rawKind = route.query.bmsKind
  const kind = String(Array.isArray(rawKind) ? rawKind[0] : (rawKind ?? '')).trim()
  if (kind === '0' || kind === '1' || kind === '2' || kind === '3') {
    betMultiplierKind.value = kind
    betMultiplierError.value = ''
    loadShareDockBetMultiplier()
    betDockEntryMode.value = 'cloud'
    persistShareDockState()
    stripBetMultiplierRouteQuery()
    return
  }
  const rawErr = route.query.bmsError
  const errRaw = String(Array.isArray(rawErr) ? rawErr[0] : (rawErr ?? '')).trim()
  if (!errRaw) return
  try {
    betMultiplierError.value = decodeURIComponent(errRaw)
  } catch {
    betMultiplierError.value = errRaw
  }
  betMultiplierKind.value = ''
  stripBetMultiplierRouteQuery()
}

function goBetMultiplierSettings() {
  betMultiplierError.value = ''
  persistShareDockState()
  const cfg = playConfig.value
  const q: Record<string, string> = {
    ...buildPlayDetailRouteQuery(),
    returnName: 'play-detail',
    ...(betMultiplierKind.value ? { activeTab: betMultiplierKind.value } : {}),
    playType: selectedTypeId.value || playTypeId.value || ('typeId' in cfg ? cfg.typeId : '') || '',
    subPlay: selectedSubId.value || subPlayId.value || ('subId' in cfg ? cfg.subId : '') || '',
    betMode: cfg.betMode || '',
    playTypeLabel: cfg.playTypeLabel || '',
    subPlayLabel: playMethod.value || cfg.playMethodLabel || '',
    playTemplate: playTree.value?.playTemplate || cfg.playTemplate || '',
  }
  if (cfg.segmentLen) q.segmentLen = String(cfg.segmentLen)
  delete q.bmsKind
  delete q.bmsError
  void router.push({ name: 'bet-multiplier-settings', query: q })
}

async function onDockConfirm() {
  if (actionLoading.value) return

  if (tab.value === '1') {
    if (!planInverseDigits.value.trim()) {
      ElMessage.warning('计划反集号码为空')
      return
    }
    actionLoading.value = true
    try {
      const cfg = playConfig.value
      await contraryBet({
        lotteryCode: lotteryCode.value,
        planInverseNumbers: planInverseDigits.value.trim(),
        playMethod: playMethod.value || undefined,
        playTemplate: playTree.value?.playTemplate,
        typeId: 'typeId' in cfg ? cfg.typeId : selectedTypeId.value || undefined,
        subId: 'subId' in cfg ? cfg.subId : selectedSubId.value || undefined,
      })
      ElMessage.success('反买投注成功，方案已运行')
      void router.push({ name: 'cloud' })
    } catch (e) {
      await handleBetError(e)
    } finally {
      actionLoading.value = false
    }
    return
  }

  if (tab.value !== '0') {
    ElMessage.info('当前 Tab 不支持此操作')
    return
  }

  if (!snapshotId.value) {
    if (betDockEntryMode.value !== 'manual') {
      ElMessage.warning('缺少分享方案信息，请从跟单大厅进入')
      return
    }
    if (selectedBetCount.value <= 0) {
      ElMessage.warning('请先选择有效号码')
      return
    }
    actionLoading.value = true
    try {
      const pm = playMethod.value || '一星定位胆 · 万位'
      const cfg = playConfig.value
      await placeGameBet(lotteryCode.value, {
        issueNo: nextIssue.value,
        amount: estimatedBetAmount.value,
        multiplier: betMultiplier.value,
        betMode: betMode.value,
        playMethod: pm,
        runMode: 'real',
        betPayload: buildGameBetPayload(pm, manualGroupContent.value, {
          playTemplate: playTree.value?.playTemplate,
          typeId: 'typeId' in cfg ? cfg.typeId : playTypeId.value || undefined,
          subId: 'subId' in cfg ? cfg.subId : subPlayId.value || undefined,
        }),
      })
      ElMessage.success('投注成功')
      void loadGameDetail()
      void refreshDockBalance()
    } catch (e) {
      await handleBetError(e)
    } finally {
      actionLoading.value = false
    }
    return
  }

  actionLoading.value = true
  try {
    if (betDockEntryMode.value === 'cloud') {
      if (!betMultiplierKind.value || !betMultiplierPayload.value) {
        ElMessage.warning('请先设置倍投模式')
        return
      }
      await shareAddToCloud(snapshotId.value, {
        betMultiplier: betMultiplierPayload.value as unknown as Record<string, unknown>,
      })
      savePlayDetailShareDock(shareDockStorageKey(), {
        entryMode: 'cloud',
        betMultiplierKind: betMultiplierKind.value,
        betMultiplier: betMultiplierPayload.value,
      })
      ElMessage.success('已添加至云端，请手动开启')
    } else {
      const cfg = playConfig.value
      await shareFollowBet(snapshotId.value, {
        lotteryCode: lotteryCode.value,
        playMethod: playMethod.value || undefined,
        playTemplate: playTree.value?.playTemplate,
        typeId: 'typeId' in cfg ? cfg.typeId : selectedTypeId.value || undefined,
        subId: 'subId' in cfg ? cfg.subId : selectedSubId.value || undefined,
      })
      ElMessage.success('跟单投注成功，方案已运行')
    }
    void router.push({ name: 'cloud' })
  } catch (e) {
    await handleBetError(e, '操作失败')
  } finally {
    actionLoading.value = false
  }
}

/** 开奖展示：drawing=等待开奖；drawn=已开出，展示 {@link drawnNumbers} */
const drawPhase = ref<'drawing' | 'drawn'>('drawing')

/** 玩法收藏（仅跟单大厅来源，按 snapshotId 同步后端） */
const isFavorite = ref(false)
const favoritePending = ref(false)

async function loadFavoriteState() {
  if (!snapshotId.value) return
  try {
    const rows = await fetchSchemeFavorites()
    isFavorite.value = rows.some((r) => r.snapshotId === snapshotId.value)
  } catch {
    // 收藏状态拉取失败不影响页面其他功能
  }
}

async function toggleFavorite() {
  if (!snapshotId.value || favoritePending.value) return
  favoritePending.value = true
  try {
    if (isFavorite.value) {
      await removeSchemeFavorite(snapshotId.value)
      isFavorite.value = false
      ElMessage.success('已取消收藏')
    } else {
      await addSchemeFavorite(snapshotId.value)
      isFavorite.value = true
      ElMessage.success('已收藏，可在内置计画方案中跟投')
    }
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '操作失败')
  } finally {
    favoritePending.value = false
  }
}

function resetSchemeDock() {
  schemeBetUnit.value = 0
  schemeBetMultiplier.value = 1
  schemeBetUnits.value = 0
  schemeContraryBetUnits.value = 0
  schemePickDigits.value = ''
  estimatedPrize.value = 0
  contraryEstimatedPrize.value = 0
}

function applySchemeDockFromDetail(detail: Awaited<ReturnType<typeof fetchGameDetail>>) {
  if (!snapshotId.value) {
    resetSchemeDock()
    return
  }
  schemeBetUnit.value = detail.schemeBetUnit ?? 0
  schemeBetMultiplier.value = detail.schemeBetMultiplier ?? 1
  schemeBetUnits.value = detail.schemeBetUnits ?? 0
  schemeContraryBetUnits.value = detail.schemeContraryBetUnits ?? 0
  schemePickDigits.value = detail.schemePickDigits ?? ''
  estimatedPrize.value = detail.estimatedPrize ?? 0
  contraryEstimatedPrize.value = detail.contraryEstimatedPrize ?? 0
  if (schemeBetUnit.value > 0) {
    betMode.value = String(schemeBetUnit.value)
  }
  if (schemeBetMultiplier.value > 0) {
    betMultiplier.value = Math.max(1, Math.round(schemeBetMultiplier.value))
  }
}

/** 已开奖号码（由接口赋值；加载前为空） */
const drawnNumbers = ref<readonly string[]>([])

const bettingRows = ref<GameBettingRow[]>([])

const bettingTableList = computed(() => bettingRows.value)
const planTrendHistoryList = computed(() =>
  planTrendHistoryRows.value.slice(0, planTrendHistoryVisibleCount.value),
)
const planTrendHistoryCanLoadMore = computed(() => {
  const total = planTrendHistoryRows.value.length
  if (total <= 0) return false
  const cap = Math.min(total, PLAN_TREND_HISTORY_MAX)
  return planTrendHistoryVisibleCount.value < cap
})

function resetPlanTrendHistoryView() {
  planTrendHistoryVisibleCount.value = PLAN_TREND_HISTORY_INITIAL
}

function loadMorePlanTrendHistory() {
  const total = planTrendHistoryRows.value.length
  planTrendHistoryVisibleCount.value = Math.min(total, PLAN_TREND_HISTORY_MAX)
}

const betRecordRows = ref<GameBetRecordDto[]>([])

function isBarePlayToken(s: string): boolean {
  const t = s.trim()
  if (!t) return true
  if (/^\d+$/.test(t)) return true
  if (/^g\d+$/i.test(t)) return true
  return false
}

/** 投注记录玩法列：优先用已解析玩法名，避免 subPlayId（如 13）闪现 */
const betRecordPlayLabel = computed(() => {
  const pm = playMethod.value.trim()
  if (pm && !isBarePlayToken(pm)) return pm
  const sel = activePlaySelection.value
  if (sel) {
    const label = formatSubPlayLabel(sel.subNode.label).trim()
    if (label && !isBarePlayToken(label)) return label
  }
  return pm || '—'
})

function applyGameDetailData(detail: Awaited<ReturnType<typeof fetchGameDetail>>) {
  currentIssue.value = detail.currentIssue
  drawPhase.value = detail.drawPhase
  drawnNumbers.value = detail.drawnNumbers
  const mergedCountdown = mergeGameDetailCountdownOnPoll(
    {
      countdownEndTime: countdownEndTime.value,
      countdownPeriod: countdownPeriod.value,
      lotteryCode: lotteryCode.value,
      countdownSec: countdownSec.value,
      countdownLabel: countdownLabel.value,
    },
    {
      countdownEndTime: detail.countdownEndTime,
      countdownPeriod: detail.countdownPeriod ?? detail.nextIssue,
      lotteryCode: lotteryCode.value,
      countdownSec: detail.countdownSec,
      countdownLabel: detail.countdownLabel,
    },
  )
  countdownSec.value = mergedCountdown.countdownSec
  countdownEndTime.value = mergedCountdown.countdownEndTime ?? ''
  countdownPeriod.value = mergedCountdown.countdownPeriod ?? ''
  countdownLabel.value = mergedCountdown.countdownLabel ?? ''
  nextIssue.value = mergedCountdown.countdownPeriod || detail.nextIssue
  planInverseDigits.value = String(detail.planInverseDigits ?? '').trim()
  planInverseBetCount.value = Number(detail.planInverseBetCount) || 0
  // 无反集号码时视为不支持展示 Tab（与后端 planContrarySupported 语义对齐）
  planContrarySupportedFromApi.value =
    detail.planContrarySupported !== false && planInverseDigits.value !== ''
  historyGameTag.value = detail.lotteryLabel
  bettingRows.value = detail.bettingRows
  betRecordRows.value = detail.betRecords
  planTrendGroupBets.value = detail.planTrendGroupBets
  planTrendHistoryRows.value = detail.planTrendHistory
  resetPlanTrendHistoryView()
  planTrendChartPoints.value = detail.planTrendChart ?? []
  applySchemeDockFromDetail(detail)
  syncDrawingUrgentPoll()
  if (String(route.query.board ?? '') === 'contrary' && showPlanContraryTab.value) {
    tab.value = '1'
  }
}

function formatDockAmount(v: number) {
  if (!Number.isFinite(v) || v <= 0) return '—'
  return String(Math.round(v * 100) / 100)
}

/** 金额列展示：只显示整数部分（向下取整），接口可为带小数字符串 */
function formatBetRecordAmount(amount: string) {
  const n = Number(amount)
  if (!Number.isFinite(n))
    return amount
  return String(Math.trunc(n))
}

function formatBetRecordPl(n: number) {
  return String(Math.abs(Math.trunc(n)))
}

const betUnitAmount = computed(() => {
  if (snapshotId.value && schemeBetUnit.value > 0) return schemeBetUnit.value
  const n = Number(betMode.value)
  return Number.isFinite(n) && n > 0 ? n : 2
})

const effectiveMultiplier = computed(() => {
  if (snapshotId.value && schemeBetMultiplier.value > 0) return schemeBetMultiplier.value
  return betMultiplier.value
})

const selectedBetCount = computed(() => {
  if (tab.value === '1') {
    if (snapshotId.value && schemeContraryBetUnits.value > 0) return schemeContraryBetUnits.value
    return planInverseBetCount.value
  }
  if (snapshotId.value && schemeBetUnits.value > 0) return schemeBetUnits.value
  return countBetUnits(playConfig.value, manualGroupContent.value)
})

const estimatedBetAmount = computed(() =>
  Math.round(betUnitAmount.value * effectiveMultiplier.value * selectedBetCount.value * 100) / 100,
)

const dockEstimatedPrize = computed(() => {
  if (tab.value === '1') return contraryEstimatedPrize.value
  return estimatedPrize.value
})

// §23.2：real 手动下注 dock 展示第三方主币种实账（与会员中心顶栏同源）
const dockBalance = ref<{ currency: string; amount: number } | null>(null)
const dockBalanceText = computed(() => {
  const b = dockBalance.value
  if (!b) return ''
  const sym = b.currency === 'CNY' ? '¥' : b.currency + ' '
  return `${sym}${b.amount.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
})

async function refreshDockBalance() {
  try {
    const b = await fetchGuajiBalance()
    dockBalance.value = { currency: b.currency, amount: b.amount }
  } catch (e) {
    dockBalance.value = null
    if (await redirectToGuajiAuthIfNeeded(e, (path) => router.push(path))) return
  }
}

async function loadGameDetail(options?: { silent?: boolean }) {
  const silent = options?.silent === true
  if (!silent) detailLoading.value = true
  try {
    const detail = await fetchGameDetail(lotteryCode.value, {
      schemeName: schemeNameFromRoute() || undefined,
      playMethod: playMethod.value || undefined,
      snapshotId: snapshotId.value || undefined,
      board: String(route.query.board ?? '').trim() || undefined,
      playTypeId: playTypeId.value || undefined,
      subPlayId: subPlayId.value || undefined,
    })
    applyGameDetailData(detail)
    if (!silent) initManualPicks()
  } catch (e) {
    if (!silent) ElMessage.error(formatClientApiError(e, '加载游戏详情失败'))
  } finally {
    if (!silent) detailLoading.value = false
  }
}

async function loadGameDraws(options?: { silent?: boolean }) {
  const silent = options?.silent === true
  if (!silent) historyDrawsLoading.value = true
  try {
    const result = await fetchGameDraws(lotteryCode.value, undefined, 50)
    historyDrawRecords.value = result.items
  } catch (e) {
    historyDrawRecords.value = []
    if (!silent) ElMessage.error(formatClientApiError(e, '加载历史开奖失败'))
  } finally {
    if (!silent) historyDrawsLoading.value = false
  }
}

let countdownTimer: ReturnType<typeof setInterval> | undefined
let drawSync: ReturnType<typeof startGameDrawSync> | null = null

/** 等待开奖时的 REST 兜底：静默拉 detail，不触发整页 loading */
function pollDrawStateQuiet() {
  void loadGameDetail({ silent: true })
}

function applyDrawResultFromWs(payload: WsDrawResultPayload) {
  const issue = String(payload.issueNo ?? '').trim()
  const balls = Array.isArray(payload.balls) ? payload.balls.filter(Boolean) : []
  if (!balls.length) return
  const waitingIssue = String(currentIssue.value ?? '').trim()
  if (waitingIssue && issue && waitingIssue !== issue) return
  if (issue) currentIssue.value = issue
  drawnNumbers.value = balls
  drawPhase.value = 'drawn'
  syncDrawingUrgentPoll()
  void loadGameDraws({ silent: true })
  void loadGameDetail({ silent: true })
}

function refreshDrawStateAfterPeriodClose() {
  void loadGameDetail({ silent: true })
  void loadGameDraws({ silent: true })
}

function syncDrawingUrgentPoll() {
  const waiting =
    countdownSec.value <= 0
    && countdownLabel.value === SCHEME_COUNTDOWN_WAITING_LABEL
  drawSync?.setDrawingUrgent(drawPhase.value === 'drawing' || waiting)
}

function tickGameDetailCountdown() {
  // 与云端中心一致：仅认 countdownEndTime 本地重算；归零/请等待时主动刷新
  if (!countdownEndTime.value) {
    syncDrawingUrgentPoll()
    return
  }
  const prev = countdownSec.value
  const display = gameDetailCountdownDisplayFields({
    countdownEndTime: countdownEndTime.value,
    countdownPeriod: countdownPeriod.value,
    lotteryCode: lotteryCode.value,
    countdownSec: countdownSec.value,
    countdownLabel: countdownLabel.value,
  })
  countdownSec.value = display.countdownSec
  countdownLabel.value = display.countdownLabel

  const periodEnded = prev > 0 && countdownSec.value === 0
  if (periodEnded) {
    refreshDrawStateAfterPeriodClose()
    syncDrawingUrgentPoll()
    return
  }

  syncDrawingUrgentPoll()
}

onMounted(async () => {
  if (route.query.board === 'contrary' && showPlanContraryTab.value) tab.value = '1'
  loadShareDockBetMultiplier()
  applyBetMultiplierFromRoute()
  await loadPlayTree()
  void loadGameDetail()
  void loadGameDraws()
  void refreshDockBalance()
  void loadFavoriteState()
  drawSync = startGameDrawSync(lotteryCode.value, {
    onPoll: pollDrawStateQuiet,
    onDrawResult: applyDrawResultFromWs,
  })
  syncDrawingUrgentPoll()
  countdownTimer = setInterval(tickGameDetailCountdown, 1000)
})

watch(showPlanContraryTab, (ok) => {
  if (!ok && tab.value === '1') tab.value = '0'
})

async function loadPlayTree() {
  playTreeLoading.value = true
  try {
    const tree = await fetchPlayTree(lotteryCode.value)
    playTree.value = tree
    const routeType = playTypeId.value
    const routeSub = subPlayId.value
    if (routeType && routeSub && findSubPlay(tree, routeType, routeSub)) {
      selectedTypeId.value = routeType
      selectedSubId.value = routeSub
    } else {
      const def = defaultPlaySelection(tree)
      selectedTypeId.value = def.typeId
      selectedSubId.value = def.subId
    }
    initManualPicks()
  } catch (e) {
    playTree.value = null
    ElMessage.error(formatClientApiError(e, '加载玩法树失败'))
  } finally {
    playTreeLoading.value = false
  }
}

function selectPlayType(typeId: string) {
  if (selectedTypeId.value === typeId) return
  selectedTypeId.value = typeId
  const subs = playTree.value?.playTypes.find((t) => t.typeId === typeId)?.subPlays ?? []
  selectedSubId.value = subs[0]?.subId ?? ''
  initManualPicks()
}

function selectSubPlay(subId: string) {
  if (selectedSubId.value === subId) return
  selectedSubId.value = subId
  initManualPicks()
}

watch([playMethod, playTypeId, subPlayId, selectedTypeId, selectedSubId], () => {
  initManualPicks()
})

watch(lotteryCode, (code) => {
  drawnNumbers.value = []
  bettingRows.value = []
  betRecordRows.value = []
  planTrendChartPoints.value = []
  planTrendHistoryRows.value = []
  resetPlanTrendHistoryView()
  planTrendGroupBets.value = 0
  historyDrawRecords.value = []
  planInverseDigits.value = ''
  planInverseBetCount.value = 0
  planContrarySupportedFromApi.value = true
  resetSchemeDock()
  currentIssue.value = ''
  nextIssue.value = ''
  countdownSec.value = 0
  countdownEndTime.value = ''
  countdownPeriod.value = ''
  countdownLabel.value = ''
  drawSync?.stop()
  void loadPlayTree().then(() => {
    void loadGameDetail()
    void loadGameDraws()
  })
  drawSync = startGameDrawSync(code, {
    onPoll: pollDrawStateQuiet,
    onDrawResult: applyDrawResultFromWs,
  })
  syncDrawingUrgentPoll()
})

watch(snapshotId, () => {
  loadShareDockBetMultiplier()
  void loadPlayTree().then(() => loadGameDetail())
})

watch(
  () => [route.query.bmsKind, route.query.bmsError] as const,
  () => applyBetMultiplierFromRoute(),
)

watch(playMethod, (label, prev) => {
  if (!snapshotId.value) return
  if (!label || isBarePlayToken(label)) return
  if (prev && !isBarePlayToken(prev)) return
  void loadGameDetail()
})

onUnmounted(() => {
  drawSync?.stop()
  if (countdownTimer) clearInterval(countdownTimer)
})

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/copy-hall')
}

async function handleBetError(e: unknown, fallback = '投注失败'): Promise<void> {
  if (await redirectToGuajiAuthIfNeeded(e, (path) => router.push(path))) return
  ElMessage.error(formatClientApiError(e, fallback))
}

</script>

<template>
  <div class="detail">
    <header class="header-wrap">
      <div class="head-row">
        <div class="head-left">
          <button type="button" class="icon-link" aria-label="返回" @click="goBack">
            <img :src="ICON_BACK" alt="" width="30" height="30" class="primary-ico-img" decoding="async" />
          </button>
          <h1 class="head-title">{{ pageTitle }}</h1>
        </div>
        <button v-if="snapshotId" type="button" class="icon-link fav-btn" :class="{ 'fav-btn--on': isFavorite }"
          :aria-label="isFavorite ? '取消收藏' : '收藏'" :aria-pressed="isFavorite" :disabled="favoritePending"
          @click="toggleFavorite">
          <svg class="fav-star" viewBox="0 0 24 24" width="24" height="24" aria-hidden="true" focusable="false">
            <path v-if="!isFavorite" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"
              stroke-linejoin="round"
              d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
            <path v-else fill="currentColor"
              d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
          </svg>
        </button>
      </div>

      <div class="draw-block">
        <div class="draw-row" :class="{ 'draw-row-balls-inline': drawPhase === 'drawn' }">
          <h2 class="period-main">第 {{ displayCurrentIssue }} 期</h2>
          <span
            class="badge-wrap"
            :aria-label="drawPhase === 'drawing' ? '等待开奖' : `本期开奖号码 ${drawnNumbers.join(' ')}`"
          >
            <span v-if="drawPhase === 'drawing'" class="draw-badge" aria-live="polite">等待开奖</span>
            <span v-else class="draw-result" role="group" :aria-label="`本期开奖号码 ${drawnNumbers.join(' ')}`">
              <span v-for="(num, idx) in drawnNumbers" :key="idx" class="draw-ball">{{ num }}</span>
            </span>
          </span>
        </div>
        <div class="draw-row draw-row-2">
          <h2 class="period-sub">距离 {{ displayNextIssue }} 期</h2>
          <div class="timer-pill">
            <img :src="ICON_TIMER" alt="" width="18" height="18" class="timer-ico-img" decoding="async" />
            <span class="timer-txt">{{ countdownDisplay }}</span>
          </div>
        </div>
      </div>

      <el-radio-group v-model="tab" size="small" class="detail-tab-rg">
        <el-radio-button
          v-for="item in visibleDetailTabs"
          :key="item.id"
          :value="item.id"
        >{{ item.label }}</el-radio-button>
      </el-radio-group>
    </header>

    <main
      class="main"
      :class="{
        'main--with-dock': (tab === '0' || tab === '1') && betDockOpen,
        'main--dock-collapsed': (tab === '0' || tab === '1') && !betDockOpen,
        'main--no-dock': tab === '2' || tab === '3' || tab === '4',
      }"
    >
      <template v-if="tab === '0'">
        <section v-if="showBetTabPlayPicker" class="play-picker" aria-label="玩法选择">
          <div class="play-picker-types">
            <button
              v-for="pt in activePlayTypes"
              :key="pt.typeId"
              type="button"
              class="play-picker-chip"
              :class="{ 'is-active': selectedTypeId === pt.typeId }"
              @click="selectPlayType(pt.typeId)"
            >
              {{ pt.label }}
            </button>
          </div>
          <div v-if="activeSubPlays.length" class="play-picker-subs">
            <button
              v-for="sp in activeSubPlays"
              :key="sp.subId"
              type="button"
              class="play-picker-sub"
              :class="{ 'is-active': selectedSubId === sp.subId }"
              @click="selectSubPlay(sp.subId)"
            >
              {{ formatSubPlayLabel(sp.label) }}
            </button>
          </div>
        </section>
        <div v-loading="detailLoading" class="table-card">
          <el-table :data="bettingTableList" class="detail-bet-table" size="small" stripe empty-text="暂无数据"
            :style="{ width: '100%' }">
            <el-table-column prop="time" label="下注时间" :min-width="42" />
            <el-table-column prop="scheme" label="方案名" :min-width="42" />
            <el-table-column prop="numbers" label="下注号码" :min-width="44">
              <template #default="{ row }">
                <span class="detail-bet-table-nums">{{ row.numbers }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="period" label="下注期" :min-width="38" />
            <el-table-column prop="draw" label="开奖号码" :min-width="46" />
            <el-table-column label="中挂" :min-width="40" align="center">
              <template #default="{ row }">
                <el-tag :type="row.win ? 'success' : 'danger'" size="small" effect="light">{{ row.win ? '中' : '挂'
                }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
      <template v-else-if="tab === '1'">
        <div class="plan-inverse-page">
          <div class="plan-inverse-inner">
            <el-card class="plan-inverse-card" shadow="never">
              <p v-if="!planInverseDigits.trim()" class="plan-inverse-empty">暂无反集数据</p>
              <p v-else class="plan-inverse-digits">{{ planInverseDigits }}</p>
              <p class="plan-inverse-meta">共 {{ planInverseBetCount }} 注</p>
            </el-card>
          </div>
        </div>
      </template>
      <template v-else-if="tab === '2'">
        <div class="plan-trend-page">
          <section class="plan-trend-chart-card" aria-label="号码走势图表">
            <div class="plan-trend-chart-head">
              <p class="plan-trend-chart-title">本组号码走势分析（{{ planTrendGroupBets }}注）</p>
            </div>
            <div class="plan-trend-chart-body">
              <div class="plan-trend-y-axis">
                <span
                  v-for="tick in planTrendChartView.yTicks"
                  :key="tick.label"
                  class="plan-trend-y-label"
                  :style="{ top: `${tick.top}%` }"
                >{{ tick.label }}</span>
              </div>
              <div class="plan-trend-chart-plot">
                <div class="plan-trend-grid">
                  <div
                    v-for="tick in planTrendChartView.yTicks"
                    :key="`grid-${tick.label}`"
                    class="plan-trend-grid-line"
                    :style="{ top: `${tick.top}%` }"
                  />
                </div>
                <svg class="plan-trend-svg" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                  <defs>
                    <linearGradient id="planTrendChartGrad" x1="0" x2="0" y1="0" y2="1">
                      <stop offset="0%" stop-color="#0066ff" />
                      <stop offset="100%" stop-color="#ffffff" />
                    </linearGradient>
                  </defs>
                  <path
                    v-if="planTrendChartView.areaD"
                    :d="planTrendChartView.areaD"
                    fill="url(#planTrendChartGrad)"
                    opacity="0.1"
                  />
                  <path
                    v-if="planTrendChartView.lineD"
                    :d="planTrendChartView.lineD"
                    fill="none"
                    stroke="#0066ff"
                    stroke-width="1.5"
                    stroke-linejoin="round"
                  />
                </svg>
                <div v-if="!planTrendChartView.dots.length" class="plan-trend-chart-empty">暂无走势数据</div>
                <div v-else class="plan-trend-dots-layer">
                  <div
                    v-for="(d, idx) in planTrendChartView.dots"
                    :key="idx"
                    class="plan-trend-dot-anchor"
                    :style="{ left: `${d.left}%`, top: `${d.top}%` }"
                  >
                    <span class="plan-trend-dot" :class="d.hit ? 'plan-trend-dot--hit' : 'plan-trend-dot--miss'" />
                  </div>
                </div>
                <div class="plan-trend-x-axis">
                  <span
                    v-for="(x, xi) in planTrendChartView.xLabels"
                    :key="xi"
                    class="plan-trend-x-tick"
                    :class="{ 'plan-trend-x-tick--hide': !x.show }"
                  >{{ x.text }}</span>
                </div>
              </div>
            </div>
          </section>

          <section class="plan-trend-history-card" aria-label="近期中挂情况">
            <div class="plan-trend-history-head">
              <h3 class="plan-trend-history-title">近期中挂情况</h3>
            </div>
            <div class="plan-trend-history-scroll">
              <el-table :data="planTrendHistoryList" class="plan-trend-el-table" size="small" stripe empty-text="暂无数据"
                :style="{ width: '100%' }">
                <el-table-column prop="period" label="期数" :min-width="44" />
                <el-table-column label="状态" :min-width="40" align="center">
                  <template #default="{ row }">
                    <el-tag :type="row.win ? 'success' : 'danger'" size="small">{{ row.win ? '中' : '挂' }}</el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <div v-if="planTrendHistoryCanLoadMore" class="plan-trend-history-foot">
              <el-button type="primary" link @click="loadMorePlanTrendHistory">查看更多历史计划</el-button>
            </div>
          </section>
        </div>
      </template>
      <template v-else-if="tab === '3'">
        <div class="history-page">
          <div class="history-subtabs-wrap">
            <el-radio-group v-model="historySubTab" size="small" class="history-subtabs-ep">
              <el-radio-button v-for="(label, hi) in historySubTabLabels" :key="label" :value="String(hi)">{{ label
              }}</el-radio-button>
            </el-radio-group>
          </div>
          <section v-loading="historyDrawsLoading" class="history-list" aria-label="开奖记录">
            <p v-if="!historyDrawsLoading && !historyDrawRecords.length" class="history-empty">暂无开奖数据</p>
            <article v-for="(rec, ri) in historyDrawRecords" :key="`${rec.periodShort}-${ri}`" class="history-card">
              <div class="history-card-head">
                <span class="history-game-name">{{ historyGameTag }}</span>
                <span class="history-period-line">第 <strong class="history-period-num">{{ rec.periodShort }}</strong>
                  期</span>
              </div>
              <div class="history-card-divider" role="presentation" />

              <div class="history-card-content">
                <template v-if="historySubTab === '0'">
                  <div class="history-balls">
                    <div v-for="(b, bi) in rec.balls" :key="bi" class="history-ball history-ball--primary">{{ b }}
                    </div>
                  </div>
                </template>
                <template v-else-if="historySubTab === '1'">
                  <div class="history-sq-row">
                    <span v-for="(b, bi) in rec.balls" :key="bi" class="history-sq history-sq-dx"
                      :class="historyBigSmallDigit(b) === '大' ? 'history-sq-dx--big' : 'history-sq-dx--small'">{{
                        historyBigSmallDigit(b) }}</span>
                  </div>
                </template>
                <template v-else-if="historySubTab === '2'">
                  <div class="history-sq-row">
                    <span v-for="(b, bi) in rec.balls" :key="bi" class="history-sq history-sq-oe"
                      :class="historyParityDigit(b) === '单' ? 'history-sq-oe--odd' : 'history-sq-oe--even'">{{
                        historyParityDigit(b) }}</span>
                  </div>
                </template>
                <template v-else-if="historySubTab === '3'">
                  <div class="history-dt-grid">
                    <div v-for="(cell, ci) in historyDragonTigerCells(historyDigitsFromBalls(rec.balls))" :key="ci"
                      class="history-dt-cell">
                      <span class="history-dt-sq" :class="`history-dt-sq--${cell.kind}`">{{ cell.char }}</span>
                      <span class="history-dt-lbl">{{ cell.label }}</span>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="history-total-block">
                    <div class="history-total-group">
                      <span class="history-total-lbl">总和:</span>
                      <span class="history-total-circle" :class="rec.sum % 2 === 1
                        ? 'history-total-circle--warm'
                        : 'history-total-circle--cool'
                        ">{{ rec.sum }}</span>
                    </div>
                    <div class="history-total-pills">
                      <span class="history-sum-pill" :class="rec.sum >= 23 ? 'history-sum-pill--big' : 'history-sum-pill--small'
                        ">{{ rec.sum >= 23 ? '大' : '小' }}</span>
                      <span class="history-sum-pill" :class="rec.sum % 2 === 1 ? 'history-sum-pill--odd' : 'history-sum-pill--even'
                        ">{{ rec.sum % 2 === 1 ? '单' : '双' }}</span>
                    </div>
                  </div>
                </template>
              </div>

              <div class="history-card-date">{{ formatHistoryDate(rec.time) }}</div>
            </article>
          </section>
          <div class="history-foot-note" role="status">
            <span class="history-foot-dot" />
            {{ historyDrawRecords.length ? `已加载最近${historyDrawRecords.length}期数据` : '暂无历史开奖' }}
            <span class="history-foot-dot" />
          </div>
        </div>
      </template>
      <template v-else-if="tab === '4'">
        <div class="table-card">
          <el-table :data="betRecordRows" class="detail-bet-table" size="small" stripe empty-text="暂无数据"
            :style="{ width: '100%' }">
            <el-table-column prop="period" label="期数" :min-width="44" />
            <el-table-column label="玩法" :min-width="44">
              <template #default>{{ betRecordPlayLabel }}</template>
            </el-table-column>
            <el-table-column prop="multiplier" label="倍数" :min-width="24" align="center" />
            <el-table-column prop="round" label="轮次" :min-width="24" align="center" />
            <el-table-column prop="amount" label="金额" :min-width="36" align="right">
              <template #default="{ row }">
                <span class="detail-bet-table-nums bet-record-num">{{ formatBetRecordAmount(row.amount) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="盈亏" :min-width="40" align="right">
              <template #default="{ row }">
                <span class="bet-record-num" :class="row.profitLoss > 0
                  ? 'bet-record-pl--gain'
                  : row.profitLoss < 0
                    ? 'bet-record-pl--loss'
                    : 'bet-record-pl--neutral'
                  ">{{ formatBetRecordPl(row.profitLoss) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" :min-width="48" align="center">
              <template #default="{ row }">
                <el-tag type="primary" effect="light" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
      <div v-else class="tab-placeholder">
        <p>「{{ activeTabLabel }}」功能开发中</p>
      </div>
    </main>

    <div v-if="tab !== '2' && tab !== '3' && tab !== '4'" class="bet-dock" :class="{ 'is-collapsed': !betDockOpen }"
      aria-label="投注区">
      <button type="button" class="dock-handle" :aria-expanded="betDockOpen" aria-controls="bet-dock-panel"
        :aria-label="betDockOpen ? '收起投注区' : '展开投注区'" @click="betDockOpen = !betDockOpen">
        <img :src="ICON_DRAG" alt="" width="28" height="28" class="handle-ico-img"
          :class="{ 'handle-ico-collapsed': !betDockOpen }" decoding="async" />
      </button>
      <div id="bet-dock-panel" v-show="betDockOpen" class="dock-inner">
        <div
          v-if="tab === '0' && betDockEntryMode === 'manual' && !snapshotId"
          class="dock-picks"
        >
          <p class="dock-picks-label">选号 · {{ playMethod || '一星定位胆 · 万位' }}</p>
          <template v-if="playConfig.inputMode === 'danshi' && textPickOptions.length">
            <div class="dock-pick-chips">
              <button
                v-for="d in textPickOptions"
                :key="d"
                type="button"
                class="dock-pick-chip"
                :class="{ 'is-active': pickDigits.includes(d) }"
                @click="togglePickDigit(d)"
              >
                {{ d }}
              </button>
            </div>
          </template>
          <template v-else-if="usesRenxuanDanshi">
            <SchemeRenxuanDanshiPanel v-model="renxuanDanshiContent" :config="playConfig" />
          </template>
          <template v-else-if="playConfig.inputMode === 'danshi'">
            <el-input
              v-model="danshiInput"
              type="textarea"
              :rows="2"
              :placeholder="lhcDanshiPlaceholder"
              size="small"
            />
          </template>
          <template v-else-if="playConfig.inputMode === 'lhc_num'">
            <div class="dock-pick-chips dock-pick-chips--lhc">
              <button
                v-for="d in lhcPickOptions"
                :key="d"
                type="button"
                class="dock-pick-chip dock-pick-chip--lhc"
                :class="{ 'is-active': pickDigits.includes(d) }"
                @click="togglePickDigit(d)"
              >
                {{ d }}
              </button>
            </div>
          </template>
          <template v-else-if="playConfig.inputMode === 'lhc_zodiac' || playConfig.inputMode === 'lhc_tail' || playConfig.inputMode === 'lhc_attr'">
            <div class="dock-pick-chips dock-pick-chips--lhc">
              <button
                v-for="d in lhcPickOptions"
                :key="d"
                type="button"
                class="dock-pick-chip dock-pick-chip--lhc"
                :class="{ 'is-active': pickDigits.includes(d) }"
                @click="togglePickDigit(d)"
              >
                {{ d }}
              </button>
            </div>
          </template>
          <template v-else-if="playConfig.inputMode === 'multiline' && textPickOptions.length">
            <div v-for="(label, li) in playConfig.segmentLabels" :key="label" class="dock-pick-row">
              <span class="dock-pick-pos">{{ label }}</span>
              <div class="dock-pick-chips">
                <button
                  v-for="d in textPickOptions"
                  :key="`${label}-${d}`"
                  type="button"
                  class="dock-pick-chip"
                  :class="{ 'is-active': isLineDigitSelected(li, d) }"
                  @click="toggleLineDigit(li, d)"
                >
                  {{ d }}
                </button>
              </div>
            </div>
          </template>
          <template v-else-if="playConfig.inputMode === 'multiline'">
            <div v-for="(label, li) in playConfig.segmentLabels" :key="label" class="dock-pick-row">
              <span class="dock-pick-pos">{{ label }}</span>
              <div class="dock-pick-chips">
                <button
                  v-for="d in digitOptions"
                  :key="`${label}-${d}`"
                  type="button"
                  class="dock-pick-chip"
                  :class="{ 'is-active': isLineDigitSelected(li, d) }"
                  @click="toggleLineDigit(li, d)"
                >
                  {{ d }}
                </button>
              </div>
            </div>
          </template>
          <template v-else-if="textPickOptions.length">
            <div class="dock-pick-chips">
              <button
                v-for="d in textPickOptions"
                :key="d"
                type="button"
                class="dock-pick-chip"
                :class="{ 'is-active': pickDigits.includes(d) }"
                @click="togglePickDigit(d)"
              >
                {{ d }}
              </button>
            </div>
          </template>
          <template v-else>
            <div class="dock-pick-chips">
              <button
                v-for="d in digitOptions"
                :key="d"
                type="button"
                class="dock-pick-chip"
                :class="{ 'is-active': pickDigits.includes(d) }"
                @click="togglePickDigit(d)"
              >
                {{ d }}
              </button>
            </div>
          </template>
        </div>
        <el-form class="dock-form dock-form--row" label-width="auto">
          <el-form-item label="倍数" class="dock-form-item--mult">
            <span v-if="snapshotId && betDockEntryMode === 'manual'" class="dock-readonly-val">{{ effectiveMultiplier }} 倍</span>
            <el-button
              v-else-if="betDockEntryMode === 'cloud'"
              type="primary"
              plain
              size="small"
              class="dock-multiplier-settings-btn"
              @click="goBetMultiplierSettings"
            >
              <span v-if="betMultiplierError" class="dock-multiplier-err">{{ betMultiplierError }}</span>
              <span v-else-if="betMultiplierSelectedLabel">{{ betMultiplierSelectedLabel }}</span>
              <span v-else>请设置</span>
            </el-button>
            <template v-else>
              <div class="dock-mult-manual">
                <el-input-number v-model="betMultiplier" :min="1" :controls="true" controls-position="right"
                  size="small" class="dock-inp-num" />
                <span class="dock-unit">倍</span>
              </div>
            </template>
          </el-form-item>
          <el-form-item label="模式" class="dock-form-item--mode">
            <span v-if="snapshotId" class="dock-readonly-val">{{ betUnitAmount }} 元/注</span>
            <el-select v-else v-model="betMode" size="small" class="dock-select" placeholder="模式">
              <el-option
                v-for="opt in BET_MODE_OPTIONS"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="dock-bottom">
          <div class="stats">
            <div class="stat-line">
              <span class="stat-l">余额:</span>
              <span class="stat-v err">{{ dockBalanceText || '—' }}</span>
            </div>
            <div class="stat-line">
              <span class="stat-l">选中:</span>
              <span class="stat-v err">{{ selectedBetCount }}</span>
              <span class="stat-u">注</span>
            </div>
            <div class="stat-line">
              <span class="stat-l">总额:</span>
              <span class="stat-v err">{{ estimatedBetAmount }}</span>
              <span class="stat-u">元</span>
            </div>
            <div class="stat-line">
              <span class="stat-l">奖金:</span>
              <span class="stat-v err">{{ formatDockAmount(dockEstimatedPrize) }}</span>
              <span class="stat-u">元(预估)</span>
            </div>
          </div>
          <div class="dock-actions-col">
            <el-button
              type="primary"
              class="dock-confirm-btn dock-confirm-btn--stacked"
              :loading="actionLoading"
              @click="onDockConfirm"
            >
              {{ dockConfirmLabel }}
            </el-button>
            <el-button
              v-if="tab !== '1'"
              type="default"
              class="dock-switch-mode-btn"
              @click="toggleBetDockEntryMode"
            >
              {{ betDockEntryMode === 'manual' ? '切换至云端挂机' : '切换至手动下注' }}
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail {
  --pri: #0066ff;
  --err: #ba1a1a;
  --surface: #f7f9fb;
  display: flex;
  flex-direction: column;
  height: 100dvh;
  overflow: hidden;
  background: var(--surface);
  color: #191c1e;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.primary-ico-img {
  width: var(--page-titlebar-icon-size);
  height: var(--page-titlebar-icon-size);
  object-fit: contain;
  display: block;
  cursor: pointer;
  pointer-events: none;
}

.header-wrap {
  flex-shrink: 0;
  z-index: 50;
  width: 100%;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
}

.head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--page-titlebar-height);
  min-height: var(--page-titlebar-height);
  box-sizing: border-box;
  padding: 0 1.5rem;
  width: 100%;
}

.head-left {
  display: flex;
  align-items: center;
  gap: 1rem;
  min-width: 0;
}

.icon-link {
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
  line-height: 0;
}

.fav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
}

.fav-btn--on {
  color: #0066ff;
}

.fav-btn:focus-visible {
  outline: 2px solid var(--pri);
  outline-offset: 2px;
  border-radius: 4px;
}

.fav-star {
  display: block;
  flex-shrink: 0;
}

.head-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.04em;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.draw-block {
  background: #f7f9fb;
  padding: 1rem 1.5rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.draw-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.375rem 0.5rem;
  min-width: 0;
  border-bottom: 1px solid rgba(226, 232, 240, 0.9);
  padding-bottom: 0.75rem;
}

.draw-row-2 {
  border-bottom: none;
  padding-bottom: 0;
}

/* 已开奖：期号与 5 个球号同一行 */
.draw-row-balls-inline {
  flex-wrap: nowrap;
  overflow: visible;
}

.draw-row-balls-inline .period-main {
  min-width: 0;
  flex: 1 1 0%;
  white-space: nowrap;
  overflow: visible;
}

.draw-row-balls-inline .badge-wrap {
  flex-shrink: 0;
}

.period-main {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 0.9rem;
  font-weight: 600;
  line-height: 1.3;
  color: #0f172a;
  flex: 1 1 auto;
  min-width: min(11rem, 100%);
  max-width: 100%;
}

.period-sub {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 0.9rem;
  font-weight: 900;
  line-height: 1.3;
  color: #0f172a;
  flex: 1 1 auto;
  min-width: min(10rem, 100%);
  max-width: 100%;
}

.badge-wrap {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-height: 38px;
  flex-shrink: 0;
  margin-left: auto;
}

.badge-wrap-demo-toggle {
  margin: 0;
  border: none;
  padding: 0;
  background: none;
  font: inherit;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  border-radius: 0.5rem;
}

.badge-wrap-demo-toggle:focus-visible {
  outline: 2px solid var(--pri);
  outline-offset: 2px;
}

.draw-badge {
  padding: 0.375rem 1rem;
  background: rgba(186, 26, 26, 0.1);
  color: var(--err);
  font-size: 1.125rem;
  font-weight: 900;
  border-radius: 999px;
  animation: pulse 2s ease-in-out infinite;
}

.draw-result {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.375rem;
}

.draw-ball {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 2rem;
  height: 2rem;
  border-radius: 999px;
  background: linear-gradient(165deg, #ff7a5c, #dc2626);
  color: #fff;
  font-size: 0.8125rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
  box-shadow:
    0 2px 4px rgba(220, 38, 38, 0.25),
    inset 0 1px 0 rgba(255, 255, 255, 0.35);
}

@keyframes pulse {

  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.88;
  }
}

.timer-pill {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid #fff;
  border-radius: 999px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  flex-shrink: 0;
}

.timer-ico-img {
  width: 1.125rem;
  height: 1.125rem;
  object-fit: contain;
  display: block;
  flex-shrink: 0;
}

.timer-txt {
  font-size: 1.125rem;
  font-weight: 900;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  color: var(--pri);
  letter-spacing: -0.04em;
}

.main {
  flex: 1;
  min-height: 0;
  width: 100%;
  max-width: 32rem;
  margin: 0 auto;
  padding: 0.5rem 0.5rem 0;
  overflow-x: hidden;
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
  -webkit-overflow-scrolling: touch;
}

.main::-webkit-scrollbar {
  display: none;
}

.main--with-dock {
  padding-bottom: calc(18rem + env(safe-area-inset-bottom));
}

.main--dock-collapsed {
  padding-bottom: calc(2.5rem + env(safe-area-inset-bottom));
}

.main--no-dock {
  padding-bottom: calc(1.25rem + env(safe-area-inset-bottom));
}

.plan-inverse-page {
  width: 100%;
  display: flex;
  justify-content: center;
  padding: 2.5rem 1.5rem 2rem;
  box-sizing: border-box;
}

.plan-inverse-inner {
  width: 100%;
  max-width: 28rem;
  margin-left: auto;
  margin-right: auto;
}

.plan-inverse-card {
  border-radius: 0.75rem;
  border: 1px solid #f1f5f9;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.plan-inverse-card :deep(.el-card__body) {
  padding: 1.5rem;
}

.plan-inverse-empty {
  margin: 0;
  font-size: 0.875rem;
  color: #94a3b8;
}

.plan-inverse-digits {
  margin: 0;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  color: #0f172a;
  font-weight: 500;
  font-size: 1.125rem;
  line-height: 1.625;
  word-break: break-all;
  white-space: pre-line;
}

.plan-inverse-meta {
  margin: 0.5rem 0 0;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  color: #64748b;
  font-size: 0.875rem;
}

/* —— 计划走势（仅本 tab 使用）—— */
.plan-trend-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem 1rem 1.5rem;
  width: 100%;
  box-sizing: border-box;
}

.plan-trend-chart-card {
  background: #fff;
  border-radius: 0.75rem;
  padding: 1.5rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.plan-trend-chart-head {
  margin-bottom: 1.25rem;
}

.plan-trend-chart-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.125rem;
  font-weight: 700;
  color: #0f172a;
}

.plan-trend-chart-body {
  display: flex;
  align-items: flex-start;
  margin-top: 0.5rem;
  width: 100%;
}

.plan-trend-y-axis {
  position: relative;
  flex-shrink: 0;
  width: 2rem;
  padding-right: 0.75rem;
  height: calc(16rem - 1.5rem);
  font-size: 11px;
  font-weight: 500;
  color: #94a3b8;
}

.plan-trend-y-label {
  position: absolute;
  right: 0.75rem;
  transform: translateY(-50%);
  line-height: 1;
}

.plan-trend-chart-plot {
  position: relative;
  flex: 1;
  min-width: 0;
  height: 16rem;
}

.plan-trend-chart-empty,
.history-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #94a3b8;
  pointer-events: none;
}

.history-empty {
  position: static;
  padding: 2rem 1rem;
}

.plan-trend-grid {
  position: absolute;
  inset: 0;
  bottom: 1.5rem;
  pointer-events: none;
}

.plan-trend-grid-line {
  position: absolute;
  left: 0;
  right: 0;
  border-bottom: 1px solid #f1f5f9;
  height: 0;
}

.plan-trend-svg {
  position: absolute;
  inset: 0;
  bottom: 1.5rem;
  width: 100%;
  height: calc(100% - 1.5rem);
}

.plan-trend-dots-layer {
  position: absolute;
  inset: 0;
  bottom: 1.5rem;
  height: calc(100% - 1.5rem);
  pointer-events: none;
}

.plan-trend-dot-anchor {
  position: absolute;
  width: 0;
  height: 0;
}

.plan-trend-dot {
  position: absolute;
  left: 0;
  top: 0;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: #fff;
  transform: translate(-50%, -50%);
  box-sizing: border-box;
}

.plan-trend-dot--hit {
  border: 1px solid #00c853;
}

.plan-trend-dot--miss {
  border: 1px solid var(--err);
  box-shadow: 0 0 8px rgba(186, 26, 26, 0.3);
}

.plan-trend-x-axis {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  font-size: 8px;
  font-weight: 500;
  color: #94a3b8;
}

.plan-trend-x-tick {
  flex: 1;
  text-align: center;
  min-width: 0;
}

.plan-trend-x-tick--hide {
  opacity: 0;
  pointer-events: none;
}

.plan-trend-history-card {
  background: #fff;
  border-radius: 0.75rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.plan-trend-history-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #f8fafc;
}

.plan-trend-history-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  color: #0f172a;
}

.plan-trend-history-updated {
  font-size: 0.75rem;
  color: #94a3b8;
}

.plan-trend-history-scroll {
  overflow-x: hidden;
  overflow-y: auto;
  max-height: none;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.plan-trend-history-scroll::-webkit-scrollbar {
  display: none;
}

.plan-trend-el-table :deep(.el-table) {
  --el-table-border-color: transparent;
}

.plan-trend-el-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.plan-trend-history-foot {
  padding: 1rem;
  display: flex;
  justify-content: center;
  border-top: 1px solid #f8fafc;
}

/* —— 历史开奖（仅 tab 3）—— */
.history-page {
  padding: 0.5rem 1rem 2rem;
  width: 100%;
  box-sizing: border-box;
}

.history-subtabs-wrap {
  margin: 0 -0.5rem 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(248, 250, 252, 0.92);
  border-bottom: 1px solid #f1f5f9;
}

.history-subtabs-ep {
  width: 100%;
  display: flex;
  flex-wrap: nowrap;
}

.history-subtabs-ep :deep(.el-radio-button) {
  flex: 1 1 0;
  min-width: 0;
}

.history-subtabs-ep :deep(.el-radio-button__inner) {
  width: 100%;
  padding: 0.4rem 0.25rem;
  font-size: 0.7rem;
  border-radius: 999px;
}

.history-subtabs-ep :deep(.el-radio-button.is-active .el-radio-button__inner) {
  background: linear-gradient(180deg, #0066ff 0%, #0050cb 100%);
  border-color: #0050cb;
  color: #fff;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.history-card {
  background: #fff;
  padding: 0.875rem 1rem 0.75rem;
  border-radius: 0.75rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
  transition: box-shadow 0.3s ease;
}

.history-card:hover {
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.06);
}

.history-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.history-game-name {
  font-size: 0.875rem;
  font-weight: 700;
  color: #0f172a;
}

.history-period-line {
  font-size: 0.75rem;
  font-weight: 500;
  color: #94a3b8;
}

.history-period-num {
  margin: 0;
  font-weight: 700;
  color: #e53935;
  font-size: inherit;
}

.history-card-divider {
  height: 1px;
  background: #e2e8f0;
  margin: 0.625rem 0 0.75rem;
}

.history-card-content {
  margin-bottom: 0.625rem;
  min-height: 4rem;
  display: flex;
  align-items: center;
}

.history-card-date {
  text-align: right;
  font-size: 0.75rem;
  color: #94a3b8;
}

.history-balls {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.625rem;
  width: 100%;
}

.history-ball {
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 1.125rem;
  font-weight: 700;
  flex-shrink: 0;
  box-shadow:
    inset -4px -4px 8px rgba(0, 0, 0, 0.2),
    inset 4px 4px 8px rgba(255, 255, 255, 0.4);
}

.history-ball--primary {
  background: linear-gradient(145deg, #0050cb 0%, #0066ff 100%);
}

.history-sq-row {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.625rem;
  width: 100%;
}

.history-sq {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.75rem;
  height: 2.75rem;
  flex-shrink: 0;
  padding: 0;
  border-radius: 0.375rem;
  font-size: 1.0625rem;
  font-weight: 700;
  color: #fff;
}

.history-sq-dx--big {
  background: #ec407a;
}

.history-sq-dx--small {
  background: #43a047;
}

.history-sq-oe--odd {
  background: #f39800;
}

.history-sq-oe--even {
  background: #45a2cc;
}

.history-dt-grid {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.25rem;
  width: 100%;
}

.history-dt-cell {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
}

.history-dt-sq {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  max-width: 2.75rem;
  aspect-ratio: 1;
  max-height: 2.75rem;
  height: auto;
  margin: 0 auto;
  border-radius: 0.375rem;
  font-size: clamp(0.5rem, 3.2vw, 0.8125rem);
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
}

.history-dt-sq--dragon {
  background: #e53935;
}

.history-dt-sq--tiger {
  background: #5c6bc0;
}

.history-dt-sq--tie {
  background: #43a047;
}

.history-dt-lbl {
  font-size: 0.5625rem;
  font-weight: 500;
  color: #94a3b8;
  text-align: center;
  line-height: 1.15;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-total-block {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  width: 100%;
}

.history-total-group {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  flex-shrink: 0;
}

.history-total-lbl {
  font-size: 1.0625rem;
  color: #475569;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
}

.history-total-circle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 999px;
  border: 1px solid #e2e8f0;
  background: #fff;
  font-size: 1.0625rem;
  font-weight: 700;
  flex-shrink: 0;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.history-total-circle--cool {
  color: #0050cb;
}

.history-total-circle--warm {
  color: #e53935;
}

.history-total-pills {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.5rem;
  justify-content: flex-end;
  flex: 1 1 auto;
  min-width: 0;
}

.history-sum-pill {
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.35rem 0.65rem;
  border-radius: 0.35rem;
  color: #fff;
  flex-shrink: 0;
}

.history-sum-pill--big {
  background: #ec407a;
}

.history-sum-pill--small {
  background: #43a047;
}

.history-sum-pill--odd {
  background: #f39800;
}

.history-sum-pill--even {
  background: #2196f3;
}

.history-foot-note {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1.5rem 0 0;
  font-size: 0.75rem;
  font-weight: 500;
  color: rgba(66, 70, 86, 0.4);
}

.history-foot-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
}

/* —— 投注记录：与「投注」tab 共用 .table-card + .detail-bet-table，无额外包层 —— */
.bet-record-num {
  font-variant-numeric: tabular-nums;
}

.bet-record-pl--gain {
  color: #ba1a1a;
  font-weight: 600;
}

.bet-record-pl--loss {
  color: #0d7a4f;
  font-weight: 600;
}

.bet-record-pl--neutral {
  color: #64748b;
  font-weight: 500;
}

.play-picker {
  margin-bottom: 1rem;
  padding: 1rem;
  background: #fff;
  border-radius: 0.75rem;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.04);
}

.play-picker-types,
.play-picker-subs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.play-picker-subs {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid #f1f5f9;
}

.play-picker-chip,
.play-picker-sub {
  padding: 0.375rem 0.75rem;
  border: none;
  border-radius: 999px;
  background: #f1f5f9;
  color: #475569;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.play-picker-chip.is-active,
.play-picker-sub.is-active {
  background: rgba(0, 102, 255, 0.1);
  color: #0066ff;
}

.table-card {
  background: #fff;
  border-radius: 0.75rem;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  overflow: hidden;
  padding: 0;
}

.detail-bet-table :deep(.el-table) {
  --el-table-border-color: transparent;
  --el-table-bg-color: transparent;
  --el-table-header-bg-color: #f8fafc;
}

.detail-bet-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.detail-bet-table :deep(.el-table__header th) {
  font-size: 10px;
  font-weight: 700;
  color: #64748b !important;
  text-transform: uppercase;
}

.detail-bet-table :deep(.el-table__body .el-table__cell) {
  font-size: 11px;
}

.detail-bet-table-nums {
  font-weight: 700;
  color: var(--pri);
}

.tab-placeholder {
  padding: 2rem 1rem;
  text-align: center;
  color: #64748b;
  font-size: 0.9rem;
}

.bet-dock {
  position: fixed;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  max-width: 28rem;
  z-index: 50;
  background: #e5e7eb;
  box-shadow: 0 -4px 15px rgba(0, 0, 0, 0.05);
  border-top: 1px solid #cbd5e1;
  border-radius: 0;
  padding-top: 0.25rem;
  padding-bottom: env(safe-area-inset-bottom);
  transition: box-shadow 0.2s ease;
}

.bet-dock.is-collapsed {
  padding-top: 0.125rem;
}

.dock-handle {
  position: absolute;
  top: -1.5rem;
  left: 50%;
  transform: translateX(-50%);
  width: 5rem;
  height: 1.5rem;
  margin: 0;
  padding: 0;
  background: #e5e7eb;
  border-radius: 0.75rem 0.75rem 0 0;
  border: 1px solid #cbd5e1;
  border-bottom: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.dock-handle:hover {
  filter: brightness(0.98);
}

.dock-handle:focus-visible {
  outline: 2px solid var(--pri);
  outline-offset: 2px;
}

.handle-ico-img {
  width: 1.75rem;
  height: 1.75rem;
  object-fit: contain;
  display: block;
  pointer-events: none;
  transition: transform 0.25s ease;
}

.handle-ico-img.handle-ico-collapsed {
  transform: rotate(180deg);
}

.dock-inner {
  padding: 1.25rem 1rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.dock-readonly-val {
  font-size: 0.875rem;
  font-weight: 600;
  color: #0f172a;
}

.dock-picks {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.dock-picks-label {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.dock-pick-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.dock-pick-pos {
  flex: 0 0 1.25rem;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.dock-pick-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.dock-pick-chips--lhc {
  max-height: 10rem;
  overflow-y: auto;
}

.dock-pick-chip--lhc {
  min-width: 2.25rem;
  font-size: 12px;
}

.dock-pick-chip {
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.35rem;
  border: 1px solid rgb(148 163 184 / 35%);
  border-radius: 0.5rem;
  background: #fff;
  color: var(--el-text-color-primary);
  font-size: 13px;
  cursor: pointer;
}

.dock-pick-chip.is-active {
  border-color: var(--el-color-primary);
  background: rgb(0 102 255 / 8%);
  color: var(--el-color-primary);
  font-weight: 600;
}

.dock-form {
  width: 100%;
}

.dock-form--row {
  display: flex;
  flex-wrap: nowrap;
  align-items: flex-end;
  gap: 0.75rem 1rem;
}

.dock-form--row :deep(.el-form-item) {
  margin-bottom: 0;
  margin-right: 0;
}

.dock-form--row :deep(.el-form-item__content) {
  flex-wrap: nowrap;
}

.dock-form-item--mult {
  flex: 1 1 0;
  min-width: 0;
}

.dock-form-item--mode {
  flex: 0 0 auto;
}

.dock-mult-manual {
  display: inline-flex;
  align-items: center;
  width: 100%;
  min-width: 0;
}

.dock-form :deep(.el-form-item__label) {
  color: #334155;
  font-weight: 500;
}

.dock-unit {
  margin-left: 0.35rem;
  color: #334155;
  font-size: 0.875rem;
}

.dock-select {
  width: 6.75rem;
}

.dock-inp-num {
  width: 100%;
  max-width: 7.5rem;
}

.dock-multiplier-settings-btn {
  font-weight: 600;
  max-width: 100%;
  white-space: normal;
  height: auto;
  line-height: 1.35;
}

.dock-multiplier-err {
  color: #dc2626;
}

.dock-actions-col {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 8.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  box-sizing: border-box;
}

.dock-actions-col :deep(.el-button) {
  flex: 1 1 0;
  min-height: 0;
  margin: 0;
  width: 100%;
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.25;
  white-space: normal;
  padding: 0.5rem 0.55rem;
  height: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.dock-confirm-btn--stacked {
  width: 100%;
}

.dock-confirm-btn {
  font-weight: 700;
}

.dock-bottom {
  position: relative;
  padding-right: calc(8.25rem + 0.75rem);
  box-sizing: border-box;
}

.stats {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  font-size: 1rem;
  min-width: 0;
  line-height: 1.35;
}

.stat-line {
  display: flex;
  align-items: center;
}

.stat-l {
  color: #334155;
  width: 3.5rem;
  flex-shrink: 0;
}

.stat-v {
  font-weight: 600;
}

.stat-v.err {
  color: var(--err);
}

.stat-u {
  color: #334155;
  margin-left: 0.25rem;
}
</style>
