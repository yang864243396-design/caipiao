<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { useSchemeTemplateLibraryStore } from '@/stores/schemeTemplateLibrary'
import {
  canGenerateNewbiePlan,
  canGenerateOneclickPlan,
  generateNewbiePlan,
  generateOneclickPlan,
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

const newbiePrincipal = ref('')
const newbieMode = ref('元')
const newbieProfitType = ref<'rate' | 'fixed' | 'accum'>('rate')
const newbieRateVal = ref('')
const newbieFixedVal = ref('')
const newbieAccumStart = ref('')
const newbieAccumStep = ref('')
const newbieGeneratePreset = ref<'line' | 'followStop' | 'suspend1' | 'suspend2'>('line')
const newbiePlanRows = ref<PlanTableRow[]>([])

const oneclickCycle = ref('')
const oneclickProfit = ref('')
const oneclickGeneratePreset = ref<'line' | 'wave'>('line')
const oneclickPlanRows = ref<PlanTableRow[]>([])

const simpleMultiples = ref('1,2,4,8')
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

function applyPayload(raw: BetMultiplierPayload | null | undefined) {
  if (!raw) {
    activeSubTab.value = '2'
    simpleMultiples.value = '1,2,4,8'
    selectedAdvancedId.value = null
    return
  }
  activeSubTab.value = raw.kind
  const nb = raw.newbie as Record<string, unknown> | undefined
  if (nb) {
    if (nb.principal != null) newbiePrincipal.value = String(nb.principal)
    if (typeof nb.mode === 'string') newbieMode.value = nb.mode
    if (nb.profitType === 'rate' || nb.profitType === 'fixed' || nb.profitType === 'accum') {
      newbieProfitType.value = nb.profitType
    }
    if (nb.rateVal != null) newbieRateVal.value = String(nb.rateVal)
    if (nb.fixedVal != null) newbieFixedVal.value = String(nb.fixedVal)
    if (nb.accumStart != null) newbieAccumStart.value = String(nb.accumStart)
    if (nb.accumStep != null) newbieAccumStep.value = String(nb.accumStep)
    const gp = nb.generatePreset
    if (gp === 'line' || gp === 'followStop' || gp === 'suspend1' || gp === 'suspend2') {
      newbieGeneratePreset.value = gp
    }
    if (Array.isArray(nb.profitTable)) newbiePlanRows.value = nb.profitTable as PlanTableRow[]
  }
  const oc = raw.oneclick as Record<string, unknown> | undefined
  if (oc) {
    if (oc.cycle != null) oneclickCycle.value = String(oc.cycle)
    if (oc.profit != null) oneclickProfit.value = String(oc.profit)
    const ogp = oc.generatePreset
    if (ogp === 'line' || ogp === 'wave') oneclickGeneratePreset.value = ogp
    if (Array.isArray(oc.profitTable)) oneclickPlanRows.value = oc.profitTable as PlanTableRow[]
  }
  const sm = raw.simple as Record<string, string> | undefined
  if (sm?.multiples != null) simpleMultiples.value = String(sm.multiples)
  const adv = raw.advanced as Record<string, string | null> | undefined
  if (adv?.selectedId) selectedAdvancedId.value = adv.selectedId
}

function buildPayload(): BetMultiplierPayload {
  return {
    kind: activeSubTab.value,
    newbie: {
      principal: newbiePrincipal.value,
      mode: newbieMode.value,
      profitType: newbieProfitType.value,
      rateVal: newbieRateVal.value,
      fixedVal: newbieFixedVal.value,
      accumStart: newbieAccumStart.value,
      accumStep: newbieAccumStep.value,
      generatePreset: newbieGeneratePreset.value,
      profitTable: newbiePlanRows.value,
    },
    oneclick: {
      cycle: oneclickCycle.value,
      profit: oneclickProfit.value,
      generatePreset: oneclickGeneratePreset.value,
      profitTable: oneclickPlanRows.value,
    },
    simple: { multiples: simpleMultiples.value },
    advanced: { selectedId: selectedAdvancedId.value },
  }
}

function needsBetPlanTable(): boolean {
  return (
    (activeSubTab.value === '0' && newbiePlanRows.value.length === 0) ||
    (activeSubTab.value === '1' && oneclickPlanRows.value.length === 0)
  )
}

function validate(): string | null {
  if (activeSubTab.value === '2' && !simpleMultiples.value.trim()) return '请填写倍数序列'
  if (activeSubTab.value === '3' && !selectedAdvancedId.value) {
    return '请选择一个高级倍投模板'
  }
  if (needsBetPlanTable()) return '请生成倍投计划'
  return null
}

function onGenerateNewbie(preset: typeof newbieGeneratePreset.value) {
  newbieGeneratePreset.value = preset
  const input = {
    principal: newbiePrincipal.value,
    mode: newbieMode.value,
    profitType: newbieProfitType.value,
    rateVal: newbieRateVal.value,
    fixedVal: newbieFixedVal.value,
    accumStart: newbieAccumStart.value,
    accumStep: newbieAccumStep.value,
    preset,
  }
  const err = canGenerateNewbiePlan(input)
  if (err) {
    ElMessage.warning(err)
    return
  }
  const rows = generateNewbiePlan(input)
  if (!rows?.length) {
    ElMessage.warning('无法生成利润表，请检查参数')
    newbiePlanRows.value = []
    return
  }
  newbiePlanRows.value = rows
  ElMessage.success('已生成利润表')
}

function onGenerateOneclick(preset: typeof oneclickGeneratePreset.value) {
  oneclickGeneratePreset.value = preset
  const input = {
    cycle: oneclickCycle.value,
    profit: oneclickProfit.value,
    preset,
  }
  const err = canGenerateOneclickPlan(input)
  if (err) {
    ElMessage.warning(err)
    return
  }
  const rows = generateOneclickPlan(input)
  if (!rows?.length) {
    ElMessage.warning('无法生成利润表，请填写计划周期与收益利润')
    oneclickPlanRows.value = []
    return
  }
  oneclickPlanRows.value = rows
  ElMessage.success('已生成利润表')
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
        <el-form-item label="总本金">
          <el-input v-model="newbiePrincipal" style="width: 120px" />
          <span style="margin: 0 0.5rem">元</span>
          <span style="margin-right: 0.5rem">投注模式</span>
          <el-select v-model="newbieMode" style="width: 88px">
            <el-option label="元" value="元" />
            <el-option label="角" value="角" />
          </el-select>
        </el-form-item>
        <el-form-item label="收益设置">
          <el-radio-group v-model="newbieProfitType">
            <el-radio value="rate">收益利率</el-radio>
            <el-input v-model="newbieRateVal" style="width: 88px; margin-right: 1rem" :disabled="newbieProfitType !== 'rate'" />
            <el-radio value="fixed">固定利润</el-radio>
            <el-input v-model="newbieFixedVal" style="width: 88px; margin-right: 1rem" :disabled="newbieProfitType !== 'fixed'" />
            <el-radio value="accum">累加利润</el-radio>
            <el-input v-model="newbieAccumStart" style="width: 64px" :disabled="newbieProfitType !== 'accum'" placeholder="起步" />
            <el-input v-model="newbieAccumStep" style="width: 64px; margin-left: 0.5rem" :disabled="newbieProfitType !== 'accum'" placeholder="累进" />
          </el-radio-group>
        </el-form-item>
        <el-form-item label="一键生成">
          <el-button @click="onGenerateNewbie('line')">直线倍投</el-button>
          <el-button @click="onGenerateNewbie('followStop')">中跟挂停</el-button>
          <el-button @click="onGenerateNewbie('suspend1')">挂停1期</el-button>
          <el-button @click="onGenerateNewbie('suspend2')">挂停2期</el-button>
        </el-form-item>
      </el-form>
    </template>

    <template v-else-if="activeSubTab === '1'">
      <el-form label-width="88px">
        <el-form-item label="计划周期">
          <el-input v-model="oneclickCycle" style="width: 160px" />
        </el-form-item>
        <el-form-item label="收益利润">
          <el-input v-model="oneclickProfit" style="width: 160px">
            <template #append>%</template>
          </el-input>
        </el-form-item>
        <el-form-item label="一键生成">
          <el-button @click="onGenerateOneclick('line')">直线倍投计划</el-button>
          <el-button @click="onGenerateOneclick('wave')">推波倍投计划</el-button>
        </el-form-item>
      </el-form>
    </template>

    <template v-else-if="activeSubTab === '2'">
      <p style="margin: 0 0 0.5rem; font-size: 13px; color: var(--el-text-color-secondary)">
        逗号分隔倍数序列，例如 1,2,4,8
      </p>
      <el-input v-model="simpleMultiples" type="textarea" :rows="3" resize="none" />
    </template>

    <template v-else>
      <p v-if="advancedSchemes.length === 0" style="color: var(--el-text-color-secondary)">
        暂无可用高级倍投模板，请先在「全局方案默认 / 方案模板库」中创建并启用。
      </p>
      <el-radio-group v-else v-model="selectedAdvancedId" style="display: flex; flex-direction: column; align-items: flex-start; gap: 0.5rem">
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
      <el-table-column v-for="col in tableColumns" :key="col.prop" :prop="col.prop" :label="col.label" :min-width="col.minWidth" />
    </el-table>

    <p style="margin: 1rem 0 0; font-size: 12px; color: var(--el-color-danger)">* 倍数计算上限为 200000 倍</p>
    <p v-if="kindLabel" style="margin: 0.25rem 0 0; font-size: 12px; color: var(--el-text-color-secondary)">
      当前选择：{{ kindLabel }}
    </p>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="onConfirm">确定</el-button>
    </template>
  </AdminDialog>
</template>
