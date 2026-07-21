<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import {
  getSchemeDefinition,
  saveBetMultiplier,
  type BetMultiplierPayload,
} from '@/api/schemes/betMultiplier'
import {
  SCHEME_DRAFT_ID,
  isDraftSchemeId,
  loadSchemeDraft,
  saveDraftBetMultiplier,
} from '@/utils/schemeDraftStorage'
import {
  loadPlayDetailShareDock,
  savePlayDetailShareDock,
} from '@/utils/playDetailShareDock'
import {
  startSchemeTemplatesSync,
  stopSchemeTemplatesSync,
  useSchemeTemplateLibrary,
} from '@/composables/useSchemeTemplateLibrary'
import {
  AGGRESSIVE_PRESET,
  DEFAULT_SIDES_PRESET,
  applyPresetTimes,
  canGenerateNewbiePlan,
  canGenerateOneclickPlan,
  generateNewbiePlan,
  generateOneclickPlan,
  type AdvanceMode,
  type CalcType,
  type PlanTableRow,
} from '@/utils/betMultiplierPlan'
import {
  normalizeBetMultiplierPersistKind,
  showAutoGenBetMultiplierTabs,
  type BetMultiplierPlayContext,
} from '@/utils/betMultiplierPlayCategory'
import { confirmDialog } from '@/utils/confirmDialog'

const route = useRoute()
const router = useRouter()

const ALL_SUB_TABS: readonly { id: '0' | '1' | '2' | '3'; label: string; autoGen?: boolean }[] = [
  { id: '0', label: '小白倍投', autoGen: true },
  { id: '1', label: '一键倍投', autoGen: true },
  { id: '2', label: '简单倍投' },
  { id: '3', label: '高级倍投' },
] as const

/** Tab 值与 el-radio-button :label 同型（字符串），避免与原生 radio 的值类型不一致导致选中态 class 不匹配 */
type SubTabId = `${0 | 1 | 2 | 3}`
/** 从玩法详情进入时默认「简单倍投」 */
const activeSubTab = ref<SubTabId>('2')

const playContext = computed<BetMultiplierPlayContext>(() => ({
  playTypeId: String(route.query.playType ?? route.query.playTypeId ?? route.query.typeId ?? '').trim(),
  subPlayId: String(route.query.subPlay ?? route.query.subPlayId ?? route.query.subId ?? '').trim(),
  betMode: String(route.query.betMode ?? '').trim(),
  playTypeLabel: String(route.query.playTypeLabel ?? '').trim(),
  subPlayLabel: String(
    route.query.subPlayLabel ?? route.query.playMethod ?? route.query.title ?? '',
  ).trim(),
  playMethod: String(route.query.playMethod ?? '').trim(),
  playTemplate: String(route.query.playTemplate ?? '').trim(),
  segmentLen: Number(route.query.segmentLen) || undefined,
}))

const showAutoGenTabs = computed(() => showAutoGenBetMultiplierTabs(playContext.value))
const visibleSubTabs = computed(() =>
  ALL_SUB_TABS.filter((t) => !t.autoGen || showAutoGenTabs.value),
)

const tabFromRoute = route.query.activeTab
const tabParsed = tabFromRoute == null || tabFromRoute === ''
  ? ''
  : String(Array.isArray(tabFromRoute) ? tabFromRoute[0] : tabFromRoute)
if (tabParsed === '2' || tabParsed === '3') {
  activeSubTab.value = tabParsed as SubTabId
} else if (tabParsed === '0' || tabParsed === '1') {
  // 小白 / 一键已屏蔽，旧链接落到简单倍投
  activeSubTab.value = '2'
}

watch(
  showAutoGenTabs,
  (ok) => {
    if (!ok && (activeSubTab.value === '0' || activeSubTab.value === '1')) {
      activeSubTab.value = '2'
    }
  },
  { immediate: true },
)

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push({ name: 'play-detail' })
}

function needsBetPlanTable(): boolean {
  // 小白/一键：无预览表且简单表也为空时才拦截（生成会写入简单表）
  if (activeSubTab.value === '0') {
    return newbiePlanRows.value.length === 0 && !simpleMultiples.value.trim()
  }
  if (activeSubTab.value === '1') {
    return oneclickPlanRows.value.length === 0 && !simpleMultiples.value.trim()
  }
  return false
}

function validateBetMultiplier(): string | null {
  const kind = normalizeBetMultiplierPersistKind(activeSubTab.value)
  if (kind === '2' && !simpleMultiples.value.trim()) {
    return '至少要输入一个直线倍投'
  }
  if (kind === '3' && !selectedAdvancedId.value) {
    return '无法保存，请先在高级倍投列表中选择一个方案'
  }
  return null
}

/** 将小白/一键预览表写入简单倍投倍数串 */
function applyPlanRowsToSimple(rows: PlanTableRow[]): boolean {
  const mults = rows
    .map((r) => String(r.mult ?? '').trim())
    .filter((m) => m !== '' && Number(m) > 0)
  if (!mults.length) return false
  simpleMultiples.value = mults.join(',')
  return true
}

function ensureSimpleFromActiveAutoGenTab(): void {
  if (activeSubTab.value === '0' && newbiePlanRows.value.length) {
    applyPlanRowsToSimple(newbiePlanRows.value)
  } else if (activeSubTab.value === '1' && oneclickPlanRows.value.length) {
    applyPlanRowsToSimple(oneclickPlanRows.value)
  }
}

/** 从方案配置进入倍投设定时，校验失败将报错文案带回方案页右侧展示 */
/** 进入页标识：scheme-detail / play-detail / advanced-scheme-edit（默认） */
const returnName = (() => {
  const r = route.query.returnName
  const v = String(Array.isArray(r) ? r[0] : (r ?? ''))
  return v === 'scheme-detail' || v === 'play-detail' ? v : 'advanced-scheme-edit'
})()

/** 返回进入页，带回 query（bmsKind 回显 / bmsError 报错） */
function returnToEntry(extra: Record<string, string>) {
  const schemeId = String(route.query.schemeId ?? '')
  const q: Record<string, string> = { ...extra }
  if (route.query.title != null && String(route.query.title) !== '') q.title = String(route.query.title)
  if (route.query.lottery != null && String(route.query.lottery) !== '') q.lottery = String(route.query.lottery)
  for (const key of ['draft', 'kind', 'runType', 'playType', 'subPlay'] as const) {
    const v = route.query[key]
    if (v != null && String(v) !== '') q[key] = String(Array.isArray(v) ? v[0] : v)
  }

  if (returnName === 'scheme-detail') {
    if (!schemeId) {
      router.back()
      return
    }
    void router.push({ name: 'scheme-detail', params: { definitionId: schemeId }, query: q })
    return
  }
  if (returnName === 'play-detail') {
    for (const key of [
      'scheme',
      'snapshotId',
      'lotteryCode',
      'playMethod',
      'board',
      'typeId',
      'playTypeId',
      'subId',
      'subPlayId',
      'tab',
    ] as const) {
      const v = route.query[key]
      if (v != null && String(v) !== '') q[key] = String(Array.isArray(v) ? v[0] : v)
    }
    void router.push({ name: 'play-detail', query: q })
    return
  }
  if (!schemeId) {
    router.back()
    return
  }
  // 从方案详情→配置页→倍投设定：回配置页时恢复 returnName=scheme-detail 与运行时 query
  const detailReturn = String(route.query.detailReturn ?? '')
  if (detailReturn === 'scheme-detail') {
    q.returnName = 'scheme-detail'
  }
  for (const key of ['turnover', 'sessionPnl', 'multiplier', 'status', 'detailReturn'] as const) {
    const v = route.query[key]
    if (v != null && String(v) !== '') q[key] = String(Array.isArray(v) ? v[0] : v)
  }
  // replace：避免在编辑页之上再叠一层同页，破坏详情↔云端的返回栈
  void router.replace({ name: 'advanced-scheme-edit', params: { schemeId }, query: q })
}

/** 是否从方案相关页面进入（需将校验错误带回上一页） */
function playDetailShareDockKey(): string {
  return String(route.query.snapshotId ?? '').trim() || '__no_snapshot__'
}

function shouldSavePlayDetailShareDock(): boolean {
  return returnName === 'play-detail'
}

function shouldReturnErrorToEntry(): boolean {
  if (returnName === 'play-detail') return true
  const schemeId = String(route.query.schemeId ?? '').trim()
  if (!schemeId) return false
  if (route.query.fromScheme === '1') return true
  return returnName === 'scheme-detail' || returnName === 'advanced-scheme-edit'
}

function showConfirmError(msg: string): void {
  if (shouldReturnErrorToEntry()) {
    returnToEntry({ bmsError: encodeURIComponent(msg) })
  } else {
    ElMessage.error(msg)
  }
}

function navigateBackToSchemeWithKind() {
  const schemeId = String(route.query.schemeId ?? '')
  if (!schemeId && returnName !== 'play-detail') return
  returnToEntry({ bmsKind: persistKindLabel() })
}

function newbiePlanInput() {
  return {
    odds: newbieOdds.value,
    firstBet: newbieFirstBet.value,
    targetProfit: newbieTargetProfit.value,
    cycle: newbieCycle.value,
    money: newbieMoney.value,
    number: newbieNumber.value,
  }
}

function oneclickPlanInput() {
  return {
    money: oneclickMoney.value,
    number: oneclickNumber.value,
    mode: oneclickMode.value,
    cycle: oneclickCycle.value,
    calcType: oneclickCalcType.value,
    targetRate: oneclickTargetRate.value,
    targetProfit: oneclickTargetProfit.value,
    sumBegin: oneclickSumBegin.value,
    sumStep: oneclickSumStep.value,
    freeList: oneclickFreeList.value,
  }
}

function buildBetMultiplierPayload(): BetMultiplierPayload {
  ensureSimpleFromActiveAutoGenTab()
  const kind = normalizeBetMultiplierPersistKind(activeSubTab.value)
  return {
    kind,
    newbie: {
      odds: newbieOdds.value,
      firstBet: newbieFirstBet.value,
      targetProfit: newbieTargetProfit.value,
      cycle: newbieCycle.value,
      money: newbieMoney.value,
      number: newbieNumber.value,
      profitTable: newbiePlanRows.value,
    },
    oneclick: {
      money: oneclickMoney.value,
      number: oneclickNumber.value,
      mode: oneclickMode.value,
      cycle: oneclickCycle.value,
      calcType: oneclickCalcType.value,
      targetRate: oneclickTargetRate.value,
      targetProfit: oneclickTargetProfit.value,
      sumBegin: oneclickSumBegin.value,
      sumStep: oneclickSumStep.value,
      freeList: oneclickFreeList.value,
      profitTable: oneclickPlanRows.value,
    },
    simple: {
      multiples: simpleMultiples.value,
      // 产品已屏蔽翻倍方式选择，固定挂翻倍
      advanceMode: 'on_lose' as AdvanceMode,
    },
    advanced: {
      selectedId: selectedAdvancedId.value,
    },
  }
}

function persistKindLabel(): '2' | '3' {
  return normalizeBetMultiplierPersistKind(activeSubTab.value)
}

function applyBetMultiplierPayload(raw: unknown) {
  if (!raw || typeof raw !== 'object') return
  const payload = raw as BetMultiplierPayload
  if (payload.kind === '0' || payload.kind === '1' || payload.kind === '2' || payload.kind === '3') {
    // 旧数据 kind=0/1：打开时落到对应产表 Tab（若玩法允许）；确认后会归一为 2
    if ((payload.kind === '0' || payload.kind === '1') && !showAutoGenTabs.value) {
      activeSubTab.value = '2'
    } else {
      activeSubTab.value = payload.kind
    }
  }
  const nb = payload.newbie as Record<string, unknown> | undefined
  if (nb) {
    if (nb.odds != null) newbieOdds.value = String(nb.odds)
    if (nb.firstBet != null) newbieFirstBet.value = String(nb.firstBet)
    if (nb.targetProfit != null) newbieTargetProfit.value = String(nb.targetProfit)
    if (nb.cycle != null) newbieCycle.value = String(nb.cycle)
    if (nb.money != null) newbieMoney.value = String(nb.money)
    if (nb.number != null) newbieNumber.value = String(nb.number)
    const nbTable = nb.profitTable
    if (Array.isArray(nbTable) && nbTable.length > 0) {
      newbiePlanRows.value = nbTable as PlanTableRow[]
    }
  }
  const oc = payload.oneclick as Record<string, unknown> | undefined
  if (oc) {
    if (oc.money != null) oneclickMoney.value = String(oc.money)
    if (oc.number != null) oneclickNumber.value = String(oc.number)
    if (oc.mode != null) oneclickMode.value = String(oc.mode)
    if (oc.cycle != null) oneclickCycle.value = String(oc.cycle)
    if (
      oc.calcType === 'rate' ||
      oc.calcType === 'fixed' ||
      oc.calcType === 'step' ||
      oc.calcType === 'free'
    ) {
      oneclickCalcType.value = oc.calcType
    }
    if (oc.targetRate != null) oneclickTargetRate.value = String(oc.targetRate)
    if (oc.targetProfit != null) oneclickTargetProfit.value = String(oc.targetProfit)
    if (oc.sumBegin != null) oneclickSumBegin.value = String(oc.sumBegin)
    if (oc.sumStep != null) oneclickSumStep.value = String(oc.sumStep)
    if (oc.freeList != null) oneclickFreeList.value = String(oc.freeList)
    const ocTable = oc.profitTable
    if (Array.isArray(ocTable) && ocTable.length > 0) {
      oneclickPlanRows.value = ocTable as PlanTableRow[]
    }
  }
  const sm = payload.simple as Record<string, string> | undefined
  if (sm?.multiples != null) simpleMultiples.value = String(sm.multiples)
  // 翻倍方式已屏蔽：忽略旧配置中的中翻倍，统一挂翻倍
  simpleAdvanceMode.value = 'on_lose'
  // 旧 kind=0/1 且简单表空：把产表结果灌进简单表，便于职责分离后继续运行
  if (!simpleMultiples.value.trim()) {
    if (payload.kind === '0' && newbiePlanRows.value.length) {
      applyPlanRowsToSimple(newbiePlanRows.value)
    } else if (payload.kind === '1' && oneclickPlanRows.value.length) {
      applyPlanRowsToSimple(oneclickPlanRows.value)
    }
  }
  const adv = payload.advanced as Record<string, string | null> | undefined
  if (adv?.selectedId) selectedAdvancedId.value = adv.selectedId
}

async function loadDefinitionBetMultiplier(definitionId: string) {
  if (isDraftSchemeId(definitionId)) {
    const draft = loadSchemeDraft()
    if (draft?.betMultiplier) applyBetMultiplierPayload(draft.betMultiplier)
    return
  }
  try {
    const def = await getSchemeDefinition(definitionId)
    applyBetMultiplierPayload(def.config?.betMultiplier)
  } catch {
    /* 加载失败保留默认表单 */
  }
}

/**
 * 倍投设定：从方案配置进入时持久化到 scheme_definitions.config.betMultiplier
 */
function shouldPersistBetMultiplier(): boolean {
  const schemeId = String(route.query.schemeId ?? '').trim()
  if (!schemeId) return false
  if (route.query.fromScheme === '1') return true
  return returnName === 'scheme-detail' || returnName === 'advanced-scheme-edit'
}

async function onConfirm() {
  if (needsBetPlanTable()) {
    await showParamRequiredDialog('请生成倍投计划')
    return
  }
  const err = validateBetMultiplier()
  if (err) {
    await showParamRequiredDialog(err)
    return
  }

  const schemeId = String(route.query.schemeId ?? '').trim()
  const persist = shouldPersistBetMultiplier()

  if (persist && isDraftSchemeId(schemeId)) {
    const kind = persistKindLabel()
    saveDraftBetMultiplier(route.query as Record<string, unknown>, kind, buildBetMultiplierPayload())
    ElMessage.success('已保存倍投方式')
    returnToEntry({ bmsKind: kind })
    return
  }

  // 方案配置/详情：仅暂存，由编辑页「完成」统一提交（不在此页落库）
  if (
    persist &&
    (returnName === 'advanced-scheme-edit' ||
      returnName === 'scheme-detail' ||
      String(route.query.detailReturn ?? '') === 'scheme-detail')
  ) {
    const kind = persistKindLabel()
    try {
      sessionStorage.setItem(
        `scheme-edit-bm-pending:${schemeId}`,
        JSON.stringify({ kind, payload: buildBetMultiplierPayload() }),
      )
    } catch {
      /* ignore */
    }
    ElMessage.success('已选择倍投方式，点击「完成」后生效')
    // 配置页已用 sessionStorage 暂存；back 避免再 push 一层编辑页导致详情/编辑历史环
    if (window.history.length > 1) router.back()
    else navigateBackToSchemeWithKind()
    return
  }

  if (persist) {
    try {
      await saveBetMultiplier(schemeId, buildBetMultiplierPayload())
    } catch (e) {
      const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '保存失败'
      showConfirmError(message)
      return
    }
  }

  if (persist) {
    ElMessage.success('已保存倍投方式')
    navigateBackToSchemeWithKind()
  } else if (shouldSavePlayDetailShareDock()) {
    const kind = persistKindLabel()
    savePlayDetailShareDock(playDetailShareDockKey(), {
      entryMode: 'cloud',
      betMultiplierKind: kind,
      betMultiplier: buildBetMultiplierPayload(),
    })
    ElMessage.success('已确认倍投设定')
    returnToEntry({ bmsKind: kind })
  } else {
    ElMessage.success('已确认倍投设定')
    router.back()
  }
}

function onCancel() {
  router.back()
}

// —— 小白倍投（简化递推产表）——
const newbieOdds = ref(String(route.query.odds ?? '1.9'))
const newbieFirstBet = ref('2')
const newbieTargetProfit = ref('1')
const newbieCycle = ref('10')
const newbieMoney = ref('1')
const newbieNumber = ref(String(route.query.number ?? route.query.betCount ?? '1'))

const newbiePlanRows = ref<PlanTableRow[]>([])
const oneclickPlanRows = ref<PlanTableRow[]>([])

// —— 一键倍投（完整计算器）——
const oneclickMoney = ref('1')
const oneclickNumber = ref(String(route.query.number ?? route.query.betCount ?? '1'))
const oneclickMode = ref(
  String(route.query.mode ?? route.query.prize ?? '').trim() ||
  String(Number(route.query.odds ?? 1.9) * Number(route.query.number ?? 1) || 1.9),
)
const oneclickCycle = ref('10')
const oneclickCalcType = ref<CalcType>('fixed')
const oneclickTargetRate = ref('10')
const oneclickTargetProfit = ref('1')
const oneclickSumBegin = ref('1')
const oneclickSumStep = ref('1')
const oneclickFreeList = ref('2,4,8')

async function showParamRequiredDialog(message: string): Promise<void> {
  await confirmDialog({
    title: '提示',
    message,
    tone: 'warning',
    confirmText: '我知道了',
    showCancel: false,
  })
}

function onGenerateNewbie() {
  const input = newbiePlanInput()
  const err = canGenerateNewbiePlan(input)
  if (err) {
    void showParamRequiredDialog(err)
    return
  }
  const rows = generateNewbiePlan(input)
  if (!rows?.length) {
    void showParamRequiredDialog('无法生成合适倍数，请检查参数后重试')
    newbiePlanRows.value = []
    return
  }
  newbiePlanRows.value = rows
  applyPlanRowsToSimple(rows)
  ElMessage.success('已生成并写入简单倍投')
}

function onGenerateOneclick() {
  const input = oneclickPlanInput()
  const err = canGenerateOneclickPlan(input)
  if (err) {
    void showParamRequiredDialog(err)
    return
  }
  const rows = generateOneclickPlan(input)
  if (!rows?.length) {
    void showParamRequiredDialog('该计划不适合投注，无法生成合适倍数')
    oneclickPlanRows.value = []
    return
  }
  oneclickPlanRows.value = rows
  applyPlanRowsToSimple(rows)
  ElMessage.success('已生成并写入简单倍投')
}

function onApplyOneclickPreset(kind: 'sides' | 'aggressive') {
  const money = Number(oneclickMoney.value) || 1
  const number = Number(oneclickNumber.value) || 1
  const mode = Number(oneclickMode.value)
  if (!(mode > 0)) {
    void showParamRequiredDialog('请先填写单倍奖金')
    return
  }
  const preset = kind === 'aggressive' ? AGGRESSIVE_PRESET : DEFAULT_SIDES_PRESET
  const rows = applyPresetTimes(preset, money, number, mode)
  oneclickCycle.value = String(preset.length)
  oneclickCalcType.value = 'free'
  oneclickFreeList.value = preset.join(',')
  // 参数变更的 watch 会清空预览表，下一 tick 再写入
  void nextTick(() => {
    oneclickPlanRows.value = rows
    applyPlanRowsToSimple(rows)
    ElMessage.success(kind === 'aggressive' ? '已套用激进预设' : '已套用默认两面预设')
  })
}

function onApplyNewbieToSimple() {
  if (!newbiePlanRows.value.length) {
    void showParamRequiredDialog('请先生成倍投计划')
    return
  }
  applyPlanRowsToSimple(newbiePlanRows.value)
  activeSubTab.value = '2'
  ElMessage.success('已应用到简单倍投')
}

function onApplyOneclickToSimple() {
  if (!oneclickPlanRows.value.length) {
    void showParamRequiredDialog('请先生成倍投计划')
    return
  }
  applyPlanRowsToSimple(oneclickPlanRows.value)
  activeSubTab.value = '2'
  ElMessage.success('已应用到简单倍投')
}

watch(oneclickCalcType, () => {
  oneclickPlanRows.value = []
})

watch(
  [
    newbieOdds,
    newbieFirstBet,
    newbieTargetProfit,
    newbieCycle,
    newbieMoney,
    newbieNumber,
    oneclickMoney,
    oneclickNumber,
    oneclickMode,
    oneclickCycle,
    oneclickTargetRate,
    oneclickTargetProfit,
    oneclickSumBegin,
    oneclickSumStep,
    oneclickFreeList,
  ],
  () => {
    newbiePlanRows.value = []
    oneclickPlanRows.value = []
  },
)

// —— 简单倍投 ——
const simpleMultiples = ref('1,2,4')
/** 固定挂翻倍（UI 已屏蔽翻倍方式） */
const simpleAdvanceMode = ref<AdvanceMode>('on_lose')

// —— 高级倍投（列表由管理后台方案模板库下发） ——
interface AdvancedScheme {
  id: string
  title: string
  lotteryCode?: string
  lotteryLabel?: string
}

const { advancedSchemes: templateSchemes } = useSchemeTemplateLibrary()
const advancedSchemes = computed(() => templateSchemes.value)
const selectedAdvancedId = ref<string | null>(null)

watch(
  advancedSchemes,
  (rows) => {
    if (selectedAdvancedId.value && !rows.some((r) => r.id === selectedAdvancedId.value)) {
      selectedAdvancedId.value = null
    }
  },
  { immediate: true },
)

function openAdvancedSchemeEditor(row: AdvancedScheme) {
  const q: Record<string, string> = { title: encodeURIComponent(row.title) }
  const ownerSchemeId = String(route.query.schemeId ?? '').trim()
  if (ownerSchemeId) q.schemeId = ownerSchemeId
  if (route.query.fromScheme === '1') q.fromScheme = '1'
  router.push({
    name: 'advanced-scheme-rounds',
    params: { schemeId: row.id },
    query: q,
  })
}

function onAddAdvancedScheme() {
  const ownerSchemeId = String(route.query.schemeId ?? '').trim()
  if (!ownerSchemeId) {
    ElMessage.warning('缺少方案 ID，无法新增高级倍投方案')
    return
  }
  const q: Record<string, string> = {
    title: encodeURIComponent('新方案'),
    newTemplate: '1',
    schemeId: ownerSchemeId,
  }
  if (route.query.fromScheme === '1') q.fromScheme = '1'
  router.push({
    name: 'advanced-scheme-rounds',
    params: { schemeId: 'new' },
    query: q,
  })
}

const showAdvancedAddBtn = computed(() => activeSubTab.value === '3')

/** 高级倍投模板库：有方案 ID 时用该方案；跟单大厅玩法详情无方案 ID 时用平台库 */
function templateSyncDefinitionId(): string {
  const schemeId = String(route.query.schemeId ?? '').trim()
  if (schemeId) return schemeId
  if (shouldSavePlayDetailShareDock()) return SCHEME_DRAFT_ID
  return ''
}

onMounted(() => {
  const schemeId = String(route.query.schemeId ?? '').trim()
  startSchemeTemplatesSync(templateSyncDefinitionId())
  if (schemeId) {
    void loadDefinitionBetMultiplier(schemeId)
  } else if (shouldSavePlayDetailShareDock()) {
    const dock = loadPlayDetailShareDock(playDetailShareDockKey())
    if (dock?.betMultiplier) applyBetMultiplierPayload(dock.betMultiplier)
  }
})
onUnmounted(stopSchemeTemplatesSync)

watch(
  () => templateSyncDefinitionId(),
  (definitionId) => {
    startSchemeTemplatesSync(definitionId)
  },
)

/** 倍投计划表列：仅用 `minWidth`，与全局 el-table 无横滚约定一致（见 DESIGN.md §8） */
interface PlanTableColumn {
  prop: string
  label: string
  minWidth: number
  /** 利润率列单行省略（其它列为多行换行） */
  overflowEllipsis?: boolean
}

const tableColumns: PlanTableColumn[] = [
  { prop: 'period', label: '期数', minWidth: 34 },
  { prop: 'mult', label: '倍数', minWidth: 40 },
  { prop: 'curBet', label: '本期投入', minWidth: 48 },
  { prop: 'totalBet', label: '总投入', minWidth: 42 },
  { prop: 'prize', label: '奖金', minWidth: 40 },
  { prop: 'profit', label: '利润', minWidth: 40 },
  { prop: 'margin', label: '利润率%', minWidth: 48, overflowEllipsis: true },
]

const planTableData = computed(() => {
  if (activeSubTab.value === '0') return newbiePlanRows.value
  if (activeSubTab.value === '1') return oneclickPlanRows.value
  return []
})

const showPlanTable = computed(() => activeSubTab.value === '0' || activeSubTab.value === '1')
</script>

<template>
  <div class="bms">
    <header class="bms-header">
      <div class="bms-header-top">
        <button type="button" class="bms-back" aria-label="返回" @click="goBack">
          <span class="material-sym" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="bms-title">倍投设定</h1>
        <div class="bms-header-right">
          <el-button v-if="showAdvancedAddBtn" type="primary" plain size="small" class="bms-add-scheme"
            @click="onAddAdvancedScheme">
            新增方案
          </el-button>
        </div>
      </div>
      <div class="bms-tabs-row">
        <el-radio-group v-model="activeSubTab" class="detail-tab-rg" size="small">
          <el-radio-button v-for="item in visibleSubTabs" :key="item.id" :value="item.id">{{ item.label
            }}</el-radio-button>
        </el-radio-group>
      </div>
    </header>

    <main class="bms-main">
      <!-- 小白倍投：简化递推产表 → 写入简单倍投 -->
      <template v-if="activeSubTab === '0'">
        <div class="bms-card">
          <div class="bms-field-row">
            <span class="bms-lbl">赔率</span>
            <el-input v-model="newbieOdds" size="small" class="bms-inp-short" />
            <span class="bms-lbl bms-lbl--push">首注倍数</span>
            <el-input v-model="newbieFirstBet" size="small" class="bms-inp-short" />
          </div>
          <div class="bms-field-row">
            <span class="bms-lbl">目标利润</span>
            <el-input v-model="newbieTargetProfit" size="small" class="bms-inp-short" />
            <span class="bms-unit">元</span>
            <span class="bms-lbl bms-lbl--push">档数</span>
            <el-input v-model="newbieCycle" size="small" class="bms-inp-short" />
          </div>
          <div class="bms-field-row">
            <span class="bms-lbl">单价</span>
            <el-input v-model="newbieMoney" size="small" class="bms-inp-short" />
            <span class="bms-lbl bms-lbl--push">注数</span>
            <el-input v-model="newbieNumber" size="small" class="bms-inp-short" />
          </div>
          <div class="bms-action-grid bms-action-grid--single">
            <el-button type="warning" class="bms-btn-generate bms-btn-generate--solid" @click="onGenerateNewbie">
              生成倍数表
            </el-button>
          </div>
        </div>
        <div class="bms-apply-row">
          <el-button type="primary" plain size="small" :disabled="!newbiePlanRows.length"
            @click="onApplyNewbieToSimple">
            应用到简单倍投
          </el-button>
        </div>
        <p class="bms-hint bms-hint--primary">* 按「连亏累加 + 保本利润」递推；默认两面：赔率 1.9 / 利润 1 / 首注 2 / 10 档</p>
        <p class="bms-hint bms-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>
      </template>

      <!-- 一键倍投：完整计算器 -->
      <template v-else-if="activeSubTab === '1'">
        <div class="bms-card">
          <div class="bms-field-row">
            <span class="bms-lbl">单价</span>
            <el-input v-model="oneclickMoney" size="small" class="bms-inp-short" />
            <span class="bms-lbl bms-lbl--push">注数</span>
            <el-input v-model="oneclickNumber" size="small" class="bms-inp-short" />
          </div>
          <div class="bms-field-row">
            <span class="bms-lbl">单倍奖金</span>
            <el-input v-model="oneclickMode" size="small" class="bms-inp-short" />
            <span class="bms-lbl bms-lbl--push">周期</span>
            <el-input v-model="oneclickCycle" size="small" class="bms-inp-short" />
          </div>
          <div class="bms-radio-block">
            <label class="bms-radio-row">
              <input v-model="oneclickCalcType" type="radio" value="rate" class="bms-native-radio" />
              <span class="bms-radio-lbl">收益率</span>
              <el-input v-model="oneclickTargetRate" size="small" class="bms-inp-grow"
                :disabled="oneclickCalcType !== 'rate'" />
              <span class="bms-suffix">%</span>
            </label>
            <label class="bms-radio-row">
              <input v-model="oneclickCalcType" type="radio" value="fixed" class="bms-native-radio" />
              <span class="bms-radio-lbl">固定利润</span>
              <el-input v-model="oneclickTargetProfit" size="small" class="bms-inp-grow"
                :disabled="oneclickCalcType !== 'fixed'" />
            </label>
            <div class="bms-radio-row bms-radio-row--accum">
              <label class="bms-accum-label">
                <input v-model="oneclickCalcType" type="radio" value="step" class="bms-native-radio" />
                <span class="bms-radio-lbl">累加利润：起步</span>
              </label>
              <el-input v-model="oneclickSumBegin" size="small" class="bms-inp-tiny"
                :disabled="oneclickCalcType !== 'step'" />
              <span class="bms-radio-lbl">累进</span>
              <el-input v-model="oneclickSumStep" size="small" class="bms-inp-tiny"
                :disabled="oneclickCalcType !== 'step'" />
            </div>
            <label class="bms-radio-row">
              <input v-model="oneclickCalcType" type="radio" value="free" class="bms-native-radio" />
              <span class="bms-radio-lbl">自由倍数</span>
              <el-input v-model="oneclickFreeList" size="small" class="bms-inp-grow"
                :disabled="oneclickCalcType !== 'free'" placeholder="如 2,4,8" />
            </label>
          </div>
          <div class="bms-action-grid">
            <el-button type="warning" class="bms-btn-generate bms-btn-generate--solid" @click="onGenerateOneclick">
              计算倍数表
            </el-button>
            <el-button type="warning" plain class="bms-btn-generate" @click="onApplyOneclickPreset('sides')">
              套用默认两面
            </el-button>
            <el-button type="warning" plain class="bms-btn-generate" @click="onApplyOneclickPreset('aggressive')">
              套用激进预设
            </el-button>
          </div>
        </div>
        <div class="bms-apply-row">
          <el-button type="primary" plain size="small" :disabled="!oneclickPlanRows.length"
            @click="onApplyOneclickToSimple">
            应用到简单倍投
          </el-button>
        </div>
        <p class="bms-hint bms-hint--primary">* 按连亏假设搜索最小倍数；结果写入简单倍投表</p>
        <p class="bms-hint bms-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>
      </template>

      <!-- 简单倍投（直线表；翻倍方式固定挂翻倍，UI 已屏蔽） -->
      <template v-else-if="activeSubTab === '2'">
        <div class="bms-card bms-card--simple">
          <el-input v-model="simpleMultiples" type="textarea" :rows="3" size="small" resize="none" class="bms-textarea"
            placeholder="直线倍数表，逗号分隔，如 2,4,8,17" />
        </div>
        <p class="bms-hint bms-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>
      </template>

      <!-- 高级倍投 -->
      <template v-else>
        <p v-if="advancedSchemes.length === 0" class="bms-hint">
          暂无可用高级倍投模板，请联系运营在管理后台「方案模板库」中创建并启用。
        </p>
        <div v-else class="bms-advanced-list">
          <div class="bms-advanced-head">
            <span>方案</span>
            <span>操作</span>
          </div>
          <div v-for="(row, idx) in advancedSchemes" :key="row.id" class="bms-advanced-row"
            :class="{ 'bms-advanced-row--alt': idx % 2 === 1 }">
            <label class="bms-advanced-left">
              <input v-model="selectedAdvancedId" type="radio" class="bms-native-radio" :value="row.id" @click.stop />
              <span class="bms-advanced-title bms-advanced-title--link" role="button" tabindex="0"
                @click.prevent.stop="openAdvancedSchemeEditor(row)"
                @keyup.enter.prevent="openAdvancedSchemeEditor(row)">{{ row.title }}</span>
            </label>
            <div class="bms-advanced-ops">
              <button type="button" class="bms-icon-btn bms-icon-btn--edit" aria-label="编辑"
                @click.stop="openAdvancedSchemeEditor(row)" />
            </div>
          </div>
        </div>
      </template>

      <div v-if="showPlanTable" class="table-card" aria-label="倍投计划表">
        <el-table :data="planTableData" class="detail-bet-table" size="small" stripe empty-text="暂无数据，请先填写参数并点击一键生成"
          :style="{ width: '100%' }">
          <el-table-column v-for="col in tableColumns" :key="col.prop" :prop="col.prop" :label="col.label"
            :min-width="col.minWidth" header-align="center"
            :class-name="col.overflowEllipsis ? 'bms-td-margin' : 'bms-td-wrap'"
            :show-overflow-tooltip="!!col.overflowEllipsis" />
        </el-table>
      </div>
    </main>

    <footer class="bms-footer">
      <el-button type="primary" class="bms-footer-btn bms-footer-btn--ok" @click="onConfirm">确认</el-button>
      <el-button class="bms-footer-btn bms-footer-btn--cancel" @click="onCancel">取消</el-button>
    </footer>
  </div>
</template>

<style scoped>
.bms {
  --bms-warn: #f39800;
  --bms-surface: #f7f9fb;
  --pri: #0066ff;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--bms-surface);
  color: #191c1e;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: env(safe-area-inset-bottom);
}

.bms-header {
  flex-shrink: 0;
  padding-top: env(safe-area-inset-top);
  padding-left: 0;
  padding-right: 0;
  padding-bottom: 0;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  color: #191c1e;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
  border-bottom: 1px solid #f1f5f9;
}

/* 左右等宽列 + 中间 auto，标题相对视口水平居中，右侧是否显示「新增方案」不挤偏 */
.bms-header-top {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  column-gap: 0.5rem;
  height: var(--page-titlebar-height);
  min-height: var(--page-titlebar-height);
  box-sizing: border-box;
  padding: 0 var(--page-titlebar-pad-x);
}

.bms-tabs-row {
  width: 100%;
}

.bms-back {
  justify-self: start;
  flex-shrink: 0;
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  padding: 0;
  border: none;
  background: transparent;
  color: #0f172a;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  -webkit-tap-highlight-color: transparent;
}

.bms-back:focus-visible {
  outline: 2px solid #0066ff;
  outline-offset: 2px;
}

.bms-back .material-sym {
  font-size: var(--page-titlebar-back-icon-size);
  color: #191c1e;
}

.bms-title {
  margin: 0;
  justify-self: center;
  text-align: center;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.bms-header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  justify-self: end;
  min-width: 0;
}

.bms-add-scheme {
  margin: 0;
  font-weight: 600;
  white-space: nowrap;
}

.bms-main {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0.75rem 0.75rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  -webkit-overflow-scrolling: touch;
}

.bms-card {
  background: #fff;
  border-radius: 0.75rem;
  padding: 1rem 0.875rem;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
}

.bms-card--simple {
  padding: 0.75rem;
}

.bms-field-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.35rem 0.5rem;
  margin-bottom: 0.875rem;
}

.bms-lbl {
  font-size: 0.8125rem;
  color: #334155;
  font-weight: 500;
  flex-shrink: 0;
}

.bms-lbl--push {
  margin-left: auto;
}

@media (max-width: 360px) {
  .bms-lbl--push {
    margin-left: 0;
    width: 100%;
  }
}

.bms-inp-short {
  width: 5rem;
}

.bms-unit {
  font-size: 0.8125rem;
  color: #64748b;
}

.bms-select-mode {
  width: 5rem;
}

.bms-radio-block {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  margin-bottom: 1rem;
}

.bms-radio-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  font-size: 0.8125rem;
  cursor: pointer;
}

.bms-radio-row--accum {
  flex-wrap: wrap;
}

.bms-accum-label {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  cursor: pointer;
}

.bms-native-radio {
  width: 1rem;
  height: 1rem;
  flex-shrink: 0;
  accent-color: #0066ff;
}

.bms-radio-lbl {
  color: #334155;
  flex-shrink: 0;
}

.bms-inp-grow {
  flex: 1 1 8rem;
  min-width: 6rem;
}

.bms-inp-tiny {
  width: 4rem;
}

.bms-action-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}

.bms-action-grid--single {
  grid-template-columns: 1fr;
}

.bms-radio-block--advance {
  margin-top: 0.75rem;
  padding-top: 0.65rem;
  border-top: 1px solid #f1f5f9;
}

.bms-hint-inline {
  margin-left: 0.35rem;
  font-size: 0.6875rem;
  color: #64748b;
  line-height: 1.4;
}

.bms-apply-row {
  display: flex;
  justify-content: flex-end;
  margin: 0.5rem 0 0;
}

.bms-btn-generate {
  margin: 0;
  height: auto;
  padding: 0.5rem 0.35rem;
  font-size: 0.6875rem;
  line-height: 1.35;
  white-space: normal;
  border-radius: 0.5rem;
}

.bms-btn-generate--solid {
  --el-button-bg-color: var(--bms-warn);
  --el-button-border-color: var(--bms-warn);
  --el-button-hover-bg-color: #e08900;
  --el-button-hover-border-color: #e08900;
}

.bms-btn-generate.is-plain {
  --el-button-text-color: var(--bms-warn);
  --el-button-border-color: var(--bms-warn);
  --el-button-bg-color: #fff;
}

.bms-field-row--oneclick {
  flex-wrap: nowrap;
  margin-bottom: 0.75rem;
}

.bms-field-row--oneclick .bms-lbl {
  flex: 0 0 5.25rem;
  text-align: left;
}

.bms-field-row--oneclick+.bms-field-row--oneclick {
  margin-bottom: 1rem;
}

.bms-oneclick-grow {
  flex: 1 1 0;
  min-width: 0;
}

.bms-oneclick-grow :deep(.el-input) {
  width: 100%;
}

.bms-inp-suffix-wrap {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.bms-inp-suffix-wrap :deep(.el-input) {
  flex: 1;
}

.bms-suffix {
  font-size: 0.875rem;
  color: #64748b;
  font-weight: 600;
}

.bms-textarea :deep(.el-textarea__inner) {
  font-size: 0.9375rem;
  line-height: 1.5;
  border-radius: 0.5rem;
}

.bms-hint {
  margin: 0;
  font-size: 0.6875rem;
  line-height: 1.45;
}

.bms-hint--danger {
  color: #ba1a1a;
}

.bms-hint--primary {
  color: #0066ff;
}

.bms-advanced-list {
  background: #fff;
  border-radius: 0.75rem;
  overflow: hidden;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
}

.bms-advanced-head {
  display: flex;
  justify-content: space-between;
  padding: 0.65rem 0.875rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: #0066ff;
  border-bottom: 1px solid #f1f5f9;
}

.bms-advanced-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.75rem 0.875rem;
  font-size: 0.8125rem;
}

.bms-advanced-row--alt {
  background: #f8fafc;
}

.bms-advanced-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.bms-advanced-title {
  color: #0f172a;
}

.bms-advanced-title--link {
  cursor: pointer;
  color: #0066ff;
  font-weight: 600;
}

.bms-advanced-title--link:hover {
  text-decoration: underline;
}

.bms-advanced-ops {
  display: flex;
  gap: 0.35rem;
  flex-shrink: 0;
}

.bms-icon-btn {
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  flex-shrink: 0;
  background-size: 55%;
  background-repeat: no-repeat;
  background-position: center;
  -webkit-tap-highlight-color: transparent;
}

.bms-icon-btn--edit {
  background-color: #0066ff;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='white'%3E%3Cpath d='M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z'/%3E%3C/svg%3E");
}

.bms-icon-btn--del {
  background-color: #e2e8f0;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%2364748b'%3E%3Cpath d='M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z'/%3E%3C/svg%3E");
}

.bms-icon-btn:focus-visible {
  outline: 2px solid #0066ff;
  outline-offset: 2px;
}

/* 底部计划表：与玩法详情「投注」Tab 的 .table-card + .detail-bet-table 一致 */
.table-card {
  margin-top: 0.25rem;
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

.detail-bet-table :deep(.el-table__header th .cell) {
  text-align: center;
}

.detail-bet-table :deep(.el-table__body .el-table__cell) {
  font-size: 11px;
  vertical-align: top;
}

/* 除利润率外：内容可换行，不撑出横向滚动 */
.detail-bet-table :deep(.bms-td-wrap .cell) {
  white-space: normal !important;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.45;
}

/* 利润率列：单行省略号，不超出单元格 */
.detail-bet-table :deep(td.bms-td-margin) {
  overflow: hidden;
}

.detail-bet-table :deep(.bms-td-margin .cell) {
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
}

.bms-footer {
  flex-shrink: 0;
  display: flex;
  gap: 0.65rem;
  padding: 0.75rem;
  padding-bottom: max(0.75rem, env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
  border-top: 1px solid #e2e8f0;
  backdrop-filter: blur(12px);
}

.bms-footer-btn {
  flex: 1;
  margin: 0;
  height: 2.75rem;
  font-weight: 700;
  border-radius: 0.625rem;
}

.bms-footer-btn--ok {
  background: #0066ff;
  border: none;
}

.bms-footer-btn--cancel {
  --el-button-bg-color: #fff;
  --el-button-text-color: #0066ff;
  --el-button-border-color: #0066ff;
}
</style>
