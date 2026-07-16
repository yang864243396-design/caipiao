<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { useSchemeTemplateLibraryStore } from '@/stores/schemeTemplateLibrary'
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
} from '@client/utils/betMultiplierPlan'

export interface BetMultiplierPayload {
  kind: '0' | '1' | '2' | '3'
  newbie?: Record<string, unknown>
  oneclick?: Record<string, unknown>
  simple?: Record<string, unknown>
  advanced?: Record<string, unknown>
}

const visible = defineModel<boolean>({ default: false })

const props = defineProps<{
  modelPayload?: BetMultiplierPayload | null
}>()

const emit = defineEmits<{
  confirm: [BetMultiplierPayload]
}>()

const templateStore = useSchemeTemplateLibraryStore()
const { templates } = storeToRefs(templateStore)

const advancedSchemes = computed(() =>
  templates.value
    .filter((t) => t.enabled)
    .map((t) => ({ id: t.id, title: t.name }))
    .sort((a, b) => a.title.localeCompare(b.title)),
)

const subTabLabels = ['小白倍投', '一键倍投', '简单倍投', '高级倍投'] as const
const activeSubTab = ref<'0' | '1' | '2' | '3'>('2')

const newbieOdds = ref('1.9')
const newbieFirstBet = ref('2')
const newbieTargetProfit = ref('1')
const newbieCycle = ref('10')
const newbieMoney = ref('1')
const newbieNumber = ref('1')
const newbiePlanRows = ref<PlanTableRow[]>([])

const oneclickMoney = ref('1')
const oneclickNumber = ref('1')
const oneclickMode = ref('1.9')
const oneclickCycle = ref('10')
const oneclickCalcType = ref<CalcType>('fixed')
const oneclickTargetRate = ref('10')
const oneclickTargetProfit = ref('1')
const oneclickSumBegin = ref('1')
const oneclickSumStep = ref('1')
const oneclickFreeList = ref('2,4,8')
const oneclickPlanRows = ref<PlanTableRow[]>([])

const simpleMultiples = ref('1,2,4,8')
const simpleAdvanceMode = ref<AdvanceMode>('on_lose')
const selectedAdvancedId = ref<string | null>(null)

const tableColumns = [
  { prop: 'period', label: '期数', minWidth: 40 },
  { prop: 'mult', label: '倍数', minWidth: 40 },
  { prop: 'curBet', label: '本期投入', minWidth: 72 },
  { prop: 'totalBet', label: '总投入', minWidth: 64 },
  { prop: 'profit', label: '利润', minWidth: 56 },
  { prop: 'margin', label: '利润率%', minWidth: 72 },
]

const planTableData = computed(() => {
  if (activeSubTab.value === '0') return newbiePlanRows.value
  if (activeSubTab.value === '1') return oneclickPlanRows.value
  return []
})

const showPlanTable = computed(() => activeSubTab.value === '0' || activeSubTab.value === '1')

const kindLabel = computed(() => subTabLabels[Number(activeSubTab.value)] ?? '')

function applyPlanRowsToSimple(rows: PlanTableRow[]): boolean {
  const mults = rows
    .map((r) => String(r.mult ?? '').trim())
    .filter((m) => m !== '' && Number(m) > 0)
  if (!mults.length) return false
  simpleMultiples.value = mults.join(',')
  return true
}

function applyPayload(raw: BetMultiplierPayload | null | undefined) {
  if (!raw) {
    activeSubTab.value = '2'
    simpleMultiples.value = '1,2,4,8'
    simpleAdvanceMode.value = 'on_lose'
    selectedAdvancedId.value = null
    return
  }
  activeSubTab.value = raw.kind === '0' || raw.kind === '1' || raw.kind === '2' || raw.kind === '3' ? raw.kind : '2'
  const nb = raw.newbie as Record<string, unknown> | undefined
  if (nb) {
    if (nb.odds != null) newbieOdds.value = String(nb.odds)
    if (nb.firstBet != null) newbieFirstBet.value = String(nb.firstBet)
    if (nb.targetProfit != null) newbieTargetProfit.value = String(nb.targetProfit)
    if (nb.cycle != null) newbieCycle.value = String(nb.cycle)
    if (nb.money != null) newbieMoney.value = String(nb.money)
    if (nb.number != null) newbieNumber.value = String(nb.number)
    if (Array.isArray(nb.profitTable)) newbiePlanRows.value = nb.profitTable as PlanTableRow[]
  }
  const oc = raw.oneclick as Record<string, unknown> | undefined
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
    if (Array.isArray(oc.profitTable)) oneclickPlanRows.value = oc.profitTable as PlanTableRow[]
  }
  const sm = raw.simple as Record<string, string> | undefined
  if (sm?.multiples != null) simpleMultiples.value = String(sm.multiples)
  if (sm?.advanceMode === 'on_win' || sm?.advanceMode === 'on_lose') {
    simpleAdvanceMode.value = sm.advanceMode
  }
  if (!simpleMultiples.value.trim()) {
    if (raw.kind === '0' && newbiePlanRows.value.length) applyPlanRowsToSimple(newbiePlanRows.value)
    else if (raw.kind === '1' && oneclickPlanRows.value.length) applyPlanRowsToSimple(oneclickPlanRows.value)
  }
  const adv = raw.advanced as Record<string, string | null> | undefined
  if (adv?.selectedId) selectedAdvancedId.value = adv.selectedId
}

function persistKind(): '2' | '3' {
  return activeSubTab.value === '3' ? '3' : '2'
}

function buildPayload(): BetMultiplierPayload {
  if (activeSubTab.value === '0' && newbiePlanRows.value.length) {
    applyPlanRowsToSimple(newbiePlanRows.value)
  } else if (activeSubTab.value === '1' && oneclickPlanRows.value.length) {
    applyPlanRowsToSimple(oneclickPlanRows.value)
  }
  return {
    kind: persistKind(),
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
      advanceMode: simpleAdvanceMode.value,
    },
    advanced: { selectedId: selectedAdvancedId.value },
  }
}

function needsBetPlanTable(): boolean {
  if (activeSubTab.value === '0') {
    return newbiePlanRows.value.length === 0 && !simpleMultiples.value.trim()
  }
  if (activeSubTab.value === '1') {
    return oneclickPlanRows.value.length === 0 && !simpleMultiples.value.trim()
  }
  return false
}

function validate(): string | null {
  if (persistKind() === '2' && !simpleMultiples.value.trim()) return '请填写倍数序列'
  if (persistKind() === '3' && !selectedAdvancedId.value) {
    return '请选择一个高级倍投模板'
  }
  if (needsBetPlanTable()) return '请生成倍投计划'
  return null
}

function onGenerateNewbie() {
  const input = {
    odds: newbieOdds.value,
    firstBet: newbieFirstBet.value,
    targetProfit: newbieTargetProfit.value,
    cycle: newbieCycle.value,
    money: newbieMoney.value,
    number: newbieNumber.value,
  }
  const err = canGenerateNewbiePlan(input)
  if (err) {
    ElMessage.warning(err)
    return
  }
  const rows = generateNewbiePlan(input)
  if (!rows?.length) {
    ElMessage.warning('无法生成合适倍数，请检查参数')
    newbiePlanRows.value = []
    return
  }
  newbiePlanRows.value = rows
  applyPlanRowsToSimple(rows)
  ElMessage.success('已生成并写入简单倍投')
}

function onGenerateOneclick() {
  const input = {
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
  const err = canGenerateOneclickPlan(input)
  if (err) {
    ElMessage.warning(err)
    return
  }
  const rows = generateOneclickPlan(input)
  if (!rows?.length) {
    ElMessage.warning('该计划不适合投注，无法生成合适倍数')
    oneclickPlanRows.value = []
    return
  }
  oneclickPlanRows.value = rows
  applyPlanRowsToSimple(rows)
  ElMessage.success('已生成并写入简单倍投')
}

function onApplyPreset(kind: 'sides' | 'aggressive') {
  const money = Number(oneclickMoney.value) || 1
  const number = Number(oneclickNumber.value) || 1
  const mode = Number(oneclickMode.value)
  if (!(mode > 0)) {
    ElMessage.warning('请先填写单倍奖金')
    return
  }
  const preset = kind === 'aggressive' ? AGGRESSIVE_PRESET : DEFAULT_SIDES_PRESET
  const rows = applyPresetTimes(preset, money, number, mode)
  oneclickPlanRows.value = rows
  oneclickCycle.value = String(preset.length)
  oneclickCalcType.value = 'free'
  oneclickFreeList.value = preset.join(',')
  applyPlanRowsToSimple(rows)
  ElMessage.success(kind === 'aggressive' ? '已套用激进预设' : '已套用默认两面预设')
}

function onConfirm() {
  const err = validate()
  if (err) {
    ElMessage.warning(err)
    return
  }
  emit('confirm', buildPayload())
  visible.value = false
}

watch(visible, (open) => {
  if (open) {
    void templateStore.loadList({ page: 1, pageSize: 200 })
    applyPayload(props.modelPayload)
  }
})
</script>

<template>
  <AdminDialog v-model="visible" title="方案模式（倍投设定）" width="min(100%, 760px)" destroy-on-close>
    <el-radio-group v-model="activeSubTab" style="margin-bottom: 1rem">
      <el-radio-button v-for="(lbl, i) in subTabLabels" :key="lbl" :value="String(i)">{{ lbl }}</el-radio-button>
    </el-radio-group>

    <template v-if="activeSubTab === '0'">
      <el-form label-width="88px" class="bms-form">
        <el-form-item label="赔率">
          <el-input v-model="newbieOdds" style="width: 100px" />
          <span style="margin: 0 0.5rem 0 1rem">首注</span>
          <el-input v-model="newbieFirstBet" style="width: 88px" />
        </el-form-item>
        <el-form-item label="目标利润">
          <el-input v-model="newbieTargetProfit" style="width: 100px" />
          <span style="margin: 0 0.5rem 0 1rem">档数</span>
          <el-input v-model="newbieCycle" style="width: 88px" />
        </el-form-item>
        <el-form-item label="单价">
          <el-input v-model="newbieMoney" style="width: 100px" />
          <span style="margin: 0 0.5rem 0 1rem">注数</span>
          <el-input v-model="newbieNumber" style="width: 88px" />
        </el-form-item>
        <el-form-item label="生成">
          <el-button type="warning" @click="onGenerateNewbie">生成倍数表</el-button>
        </el-form-item>
      </el-form>
    </template>

    <template v-else-if="activeSubTab === '1'">
      <el-form label-width="88px">
        <el-form-item label="单价 / 注数">
          <el-input v-model="oneclickMoney" style="width: 88px" />
          <el-input v-model="oneclickNumber" style="width: 88px; margin-left: 0.5rem" />
        </el-form-item>
        <el-form-item label="单倍奖金">
          <el-input v-model="oneclickMode" style="width: 100px" />
          <span style="margin: 0 0.5rem 0 1rem">周期</span>
          <el-input v-model="oneclickCycle" style="width: 88px" />
        </el-form-item>
        <el-form-item label="计算类型">
          <el-radio-group v-model="oneclickCalcType">
            <el-radio value="rate">收益率</el-radio>
            <el-radio value="fixed">固定利润</el-radio>
            <el-radio value="step">累加</el-radio>
            <el-radio value="free">自由</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="oneclickCalcType === 'rate'" label="收益率%">
          <el-input v-model="oneclickTargetRate" style="width: 120px" />
        </el-form-item>
        <el-form-item v-else-if="oneclickCalcType === 'fixed'" label="固定利润">
          <el-input v-model="oneclickTargetProfit" style="width: 120px" />
        </el-form-item>
        <el-form-item v-else-if="oneclickCalcType === 'step'" label="起步 / 累进">
          <el-input v-model="oneclickSumBegin" style="width: 88px" />
          <el-input v-model="oneclickSumStep" style="width: 88px; margin-left: 0.5rem" />
        </el-form-item>
        <el-form-item v-else label="自由倍数">
          <el-input v-model="oneclickFreeList" style="width: 240px" placeholder="如 2,4,8" />
        </el-form-item>
        <el-form-item label="操作">
          <el-button type="warning" @click="onGenerateOneclick">计算倍数表</el-button>
          <el-button @click="onApplyPreset('sides')">默认两面</el-button>
          <el-button @click="onApplyPreset('aggressive')">激进预设</el-button>
        </el-form-item>
      </el-form>
    </template>

    <template v-else-if="activeSubTab === '2'">
      <p style="margin: 0 0 0.5rem; font-size: 13px; color: var(--el-text-color-secondary)">
        逗号分隔倍数序列；小白/一键结果写入此处
      </p>
      <el-input v-model="simpleMultiples" type="textarea" :rows="3" resize="none" />
      <el-radio-group v-model="simpleAdvanceMode" style="margin-top: 0.75rem">
        <el-radio value="on_lose">挂翻倍</el-radio>
        <el-radio value="on_win">中翻倍</el-radio>
      </el-radio-group>
    </template>

    <template v-else>
      <p v-if="advancedSchemes.length === 0" style="color: var(--el-text-color-secondary)">
        暂无可用高级倍投模板，请先在「全局方案默认 / 方案模板库」中创建并启用。
      </p>
      <el-radio-group
        v-else
        v-model="selectedAdvancedId"
        style="display: flex; flex-direction: column; align-items: flex-start; gap: 0.5rem"
      >
        <el-radio v-for="row in advancedSchemes" :key="row.id" :value="row.id">{{ row.title }}</el-radio>
      </el-radio-group>
    </template>

    <el-table
      v-if="showPlanTable && planTableData.length"
      :data="planTableData"
      size="small"
      stripe
      style="margin-top: 1rem"
      empty-text="暂无计划表"
    >
      <el-table-column
        v-for="col in tableColumns"
        :key="col.prop"
        :prop="col.prop"
        :label="col.label"
        :min-width="col.minWidth"
      />
    </el-table>

    <p style="margin: 1rem 0 0; font-size: 12px; color: var(--el-color-danger)">* 倍数计算上限为 200000 倍</p>
    <p v-if="kindLabel" style="margin: 0.25rem 0 0; font-size: 12px; color: var(--el-text-color-secondary)">
      当前选择：{{ kindLabel }}（保存运行 kind={{ persistKind() }}）
    </p>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="onConfirm">确定</el-button>
    </template>
  </AdminDialog>
</template>
