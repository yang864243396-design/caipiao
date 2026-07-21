<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import ShareBetMultiplierDialog, { type BetMultiplierPayload } from '@/components/schemes/ShareBetMultiplierDialog.vue'
import SchemeGroupPickPanel from '@/components/schemes/SchemeGroupPickPanel.vue'
import SchemeGroupInputPanel from '@/components/schemes/SchemeGroupInputPanel.vue'
import type { CreateShareSnapshotInput } from '@/api/schemes'
import { BET_MODE_OPTIONS, betUnitFromSchemeConfig } from '@/constants/betModeOptions'
import { useAdminPlayTreeConfig } from '@/composables/useAdminPlayTreeConfig'
import { useLotteryCatalogStore } from '@/stores/lotteryCatalog'
import { useSchemeInstancesStore } from '@/stores/schemeInstances'
import {
  countBetUnits,
  groupContentPlaceholder,
  validateSchemeGroups,
} from '@client/utils/betPayload'
import {
  groupDigitInputHint,
  schemeGroupUsesDigitInput,
  schemeGroupUsesPickPanel,
} from '@client/utils/pickPanelOptions'
import {
  fromDatePickerValue,
  normalizeSchemeTimePairFromConfig,
  schemeTimeRangeError,
  toDatePickerValue,
} from '@/utils/schemeDateTime'
import type { SchemeShareSnapshotRow } from '@/types/schemes'

const visible = defineModel<boolean>({ default: false })

const props = defineProps<{
  editSnapshot?: SchemeShareSnapshotRow | null
}>()

const catalog = useLotteryCatalogStore()
const store = useSchemeInstancesStore()
const { rows: lotteryRows } = storeToRefs(catalog)

const saving = ref(false)
const schemeName = ref('')
const lotteryCode = ref('')
const runTypeId = ref('fixed_rotate')
const playTypeId = ref('')
const subPlayId = ref('')
const runMode = ref<'formal' | 'sim'>('formal')
const schemeFunds = ref('10000')
const betUnit = ref('2')
const multCoeff = ref('1')
const stopLoss = ref('')
const takeProfit = ref('')
const startTime = ref('')
const endTime = ref('')
const schemeGroups = ref<string[]>([''])
const betMultiplier = ref<BetMultiplierPayload | null>(null)
const betMultiplierVisible = ref(false)

const { load: loadPlayConfig, playConfig } = useAdminPlayTreeConfig(lotteryCode, playTypeId, subPlayId)

const formRef = ref<FormInstance>()

/** el-form 校验模型：与各字段 ref 同步，用于必填项内联提示 */
const validateModel = reactive({
  schemeName: '',
  lotteryCode: '',
  runTypeId: 'fixed_rotate',
  playTypeId: '',
  subPlayId: '',
  multCoeff: '1',
  betMultiplierKind: '',
})

watch(
  [schemeName, lotteryCode, runTypeId, playTypeId, subPlayId, multCoeff, () => betMultiplier.value?.kind ?? ''],
  ([name, lottery, runType, playType, subPlay, mult, bmKind]) => {
    validateModel.schemeName = name.trim()
    validateModel.lotteryCode = lottery
    validateModel.runTypeId = runType
    validateModel.playTypeId = playType
    validateModel.subPlayId = subPlay
    validateModel.multCoeff = mult.trim()
    validateModel.betMultiplierKind = bmKind
  },
)

const formRules: FormRules<typeof validateModel> = {
  schemeName: [{ required: true, message: '请填写方案名称', trigger: 'blur' }],
  lotteryCode: [{ required: true, message: '请选择彩种', trigger: 'change' }],
  runTypeId: [{ required: true, message: '请选择运行类型', trigger: 'change' }],
  playTypeId: [
    {
      validator: (_rule, _value, callback) => {
        if (isBuiltinPlan.value || playTypeId.value) callback()
        else callback(new Error('请选择玩法类型'))
      },
      trigger: 'change',
    },
  ],
  subPlayId: [
    {
      validator: (_rule, _value, callback) => {
        if (isBuiltinPlan.value || subPlayId.value) callback()
        else callback(new Error('请选择子玩法'))
      },
      trigger: 'change',
    },
  ],
  multCoeff: [
    {
      validator: (_rule, _value, callback) => {
        const raw = multCoeff.value.trim()
        if (!raw) {
          callback(new Error('倍数系数不能为空'))
          return
        }
        const num = Number(raw)
        if (!Number.isFinite(num) || num < 0 || !Number.isInteger(num)) {
          callback(new Error('倍数系数须为非负整数'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
  betMultiplierKind: [{ required: true, message: '请设置方案模式（倍投设定）', trigger: 'change' }],
}

const RUN_TYPE_OPTIONS = [
  { label: '定码轮换', value: 'fixed_rotate' },
  { label: '高级定码轮换', value: 'adv_fixed_rotate' },
  { label: '高级开某投某', value: 'adv_trigger_bet' },
  { label: '冷热出号', value: 'hot_cold_warm' },
  { label: '随机出号', value: 'random_draw' },
  { label: '固定取码', value: 'fixed_number' },
] as const

const BET_MULTIPLIER_KIND_LABELS: Record<string, string> = {
  '0': '小白倍投',
  '1': '一键倍投',
  '2': '简单倍投',
  '3': '高级倍投',
}

const isEdit = computed(() => !!props.editSnapshot?.id)
const dialogTitle = computed(() => (isEdit.value ? '编辑分享池方案' : '新建分享池方案'))
const submitLabel = computed(() => (isEdit.value ? '保存' : '创建'))

const lotteryOptions = computed(() =>
  [...lotteryRows.value].sort((a, b) => a.sortOrder - b.sortOrder),
)

/** 维护彩种仅展示不可选；编辑时保留当前已选维护彩种以免下拉为空 */
function isLotterySelectable(lot: { code: string; saleStatus: string }) {
  if (lot.saleStatus === 'on_sale') return true
  return lot.code === lotteryCode.value
}

const isBuiltinPlan = computed(() => runTypeId.value === 'builtin_plan')
const usesGroupContent = computed(
  () => runTypeId.value === 'fixed_rotate' || runTypeId.value === 'fixed_number',
)
const schemeUsesPickPanel = computed(() => schemeGroupUsesPickPanel(playConfig.value))
/** 数字玩法方案内容改用输入框录入（对齐第三方，不点选） */
const schemeUsesDigitInput = computed(() => schemeGroupUsesDigitInput(playConfig.value))
/** 数字录入提示（按玩法动态：位名 + 示例） */
const digitInputHint = computed(() => groupDigitInputHint(playConfig.value))
const groupInputPlaceholder = computed(() => groupContentPlaceholder(playConfig.value))

const betMultiplierLabel = computed(() => {
  const kind = betMultiplier.value?.kind
  return kind ? (BET_MULTIPLIER_KIND_LABELS[kind] ?? kind) : ''
})

function resetForm() {
  schemeName.value = ''
  lotteryCode.value = ''
  runTypeId.value = 'fixed_rotate'
  playTypeId.value = ''
  subPlayId.value = ''
  runMode.value = 'formal'
  schemeFunds.value = '10000'
  betUnit.value = '2'
  multCoeff.value = '1'
  stopLoss.value = ''
  takeProfit.value = ''
  startTime.value = ''
  endTime.value = ''
  schemeGroups.value = ['']
  betMultiplier.value = { kind: '2', simple: { multiples: '1,2,4,8' } }
}

function configValue(cfg: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const val = cfg[key]
    if (val !== undefined && val !== null && String(val).trim() !== '') {
      return String(val).trim()
    }
  }
  return ''
}

function configSimBet(cfg: Record<string, unknown>): boolean {
  if (cfg.simBet === true) return true
  if (typeof cfg.simBet === 'string' && cfg.simBet.toLowerCase() === 'true') return true
  if (cfg.runMode === 'sim') return true
  return false
}

function applyBetMultiplierFromConfig(raw: unknown) {
  if (!raw || typeof raw !== 'object') {
    betMultiplier.value = { kind: '2', simple: { multiples: '1,2,4,8' } }
    return
  }
  const payload = raw as BetMultiplierPayload
  if (payload.kind === '0' || payload.kind === '1' || payload.kind === '2' || payload.kind === '3') {
    betMultiplier.value = payload
  } else {
    betMultiplier.value = { kind: '2', simple: { multiples: '1,2,4,8' } }
  }
}

function readConfigTime(cfg: Record<string, unknown>, key: string): string {
  const val = cfg[key]
  if (val === undefined || val === null) return ''
  return String(val).trim()
}

async function populateFromSnapshot(row: SchemeShareSnapshotRow) {
  const cfg = row.config ?? {}
  schemeName.value = row.schemeName
  lotteryCode.value = row.lotteryCode
  runTypeId.value = configValue(cfg, 'runTypeId', 'runType') || row.settings.runTypeId || 'fixed_rotate'
  playTypeId.value = configValue(cfg, 'playTypeId', 'typeId', 'playType') || row.settings.playTypeId
  subPlayId.value = configValue(cfg, 'subPlayId', 'subId', 'subPlay') || row.settings.subPlayId
  runMode.value = configSimBet(cfg) ? 'sim' : 'formal'
  schemeFunds.value = configValue(cfg, 'schemeFunds') || (row.fundYuan != null ? String(row.fundYuan) : '')
  betUnit.value = betUnitFromSchemeConfig(cfg)
  multCoeff.value = configValue(cfg, 'multCoeff') || '1'
  stopLoss.value = configValue(cfg, 'stopLoss')
  takeProfit.value = configValue(cfg, 'takeProfit')
  const times = normalizeSchemeTimePairFromConfig(readConfigTime(cfg, 'startTime'), readConfigTime(cfg, 'endTime'))
  startTime.value = toDatePickerValue(times.start)
  endTime.value = toDatePickerValue(times.end)
  applyBetMultiplierFromConfig(cfg.betMultiplier)

  const groups = cfg.schemeGroups
  if (Array.isArray(groups) && groups.length > 0) {
    schemeGroups.value = groups.map((g) => String(g))
  } else {
    schemeGroups.value = runTypeId.value === 'fixed_number' ? [''] : ['']
  }

  await loadPlayConfig()
}

watch(
  () => [visible.value, props.editSnapshot?.id] as const,
  ([open, snapId]) => {
    if (!open) return
    saving.value = false
    void catalog.hydrate().then(() => {
      if (snapId && props.editSnapshot) {
        void populateFromSnapshot(props.editSnapshot)
      } else {
        resetForm()
      }
      void nextTick(() => formRef.value?.clearValidate())
    })
  },
)

watch(lotteryCode, () => {
  if (!lotteryCode.value) return
  void loadPlayConfig()
})

function addGroup() {
  if (runTypeId.value === 'fixed_number') return
  schemeGroups.value.push('')
}

function removeGroup(index: number) {
  if (runTypeId.value === 'fixed_number') {
    schemeGroups.value[0] = ''
    return
  }
  if (schemeGroups.value.length <= 1) {
    schemeGroups.value[0] = ''
    return
  }
  schemeGroups.value.splice(index, 1)
}

function groupBetUnits(raw: string): number {
  return countBetUnits(playConfig.value, raw ?? '')
}

function onBetMultiplierConfirm(payload: BetMultiplierPayload) {
  betMultiplier.value = payload
  if (payload?.kind) {
    void nextTick(() => formRef.value?.clearValidate('betMultiplierKind'))
  }
}

function buildInput(): CreateShareSnapshotInput {
  // 定位胆多位内容含前导空行，禁止 trim（否则 ,,12,, 会压成万位）
  const keepGroup = (g: string) => String(g ?? '').replace(/\r/g, '')
  const groups = schemeGroups.value.map(keepGroup).filter((g) => g.trim())
  return {
    schemeName: schemeName.value.trim(),
    lotteryCode: lotteryCode.value,
    runTypeId: runTypeId.value,
    playTypeId: playTypeId.value,
    subPlayId: subPlayId.value,
    runMode: runMode.value === 'sim' ? 'sim' : 'real',
    schemeFunds: schemeFunds.value.trim(),
    betUnit: betUnit.value,
    multCoeff: multCoeff.value.trim(),
    betMultiplier: betMultiplier.value as unknown as Record<string, unknown>,
    stopLoss: stopLoss.value.trim(),
    takeProfit: takeProfit.value.trim(),
    startTime: fromDatePickerValue(startTime.value),
    endTime: fromDatePickerValue(endTime.value),
    schemeGroups:
      runTypeId.value === 'fixed_number' ? [keepGroup(schemeGroups.value[0] ?? '')] : groups,
  }
}

/** 业务校验（必填项由 el-form 内联规则负责）：返回错误文案（null 表示通过）。 */
function validateBeforeSubmit(): string | null {
  const selectedLottery = lotteryRows.value.find((r) => r.code === lotteryCode.value)
  if (selectedLottery && selectedLottery.saleStatus !== 'on_sale') {
    const keepMaintOnEdit = isEdit.value && props.editSnapshot?.lotteryCode === lotteryCode.value
    if (!keepMaintOnEdit) return '维护中的彩种不可选择，请更换为上架彩种'
  }

  const timeErr = schemeTimeRangeError(fromDatePickerValue(startTime.value), fromDatePickerValue(endTime.value))
  if (timeErr) return timeErr

  if (usesGroupContent.value) {
    const groups = runTypeId.value === 'fixed_number' ? [schemeGroups.value[0] ?? ''] : schemeGroups.value
    const hasContent = groups.some((g) => (g ?? '').trim() !== '')
    if (!hasContent) return '请在「方案内容」中选号后再提交'
    const groupCheck = validateSchemeGroups(playConfig.value, groups)
    if (!groupCheck.ok) {
      for (const idx of groupCheck.invalidIndexes) {
        schemeGroups.value[idx] = ''
      }
      return `${groupCheck.message}，请按当前玩法规则填写`
    }
    schemeGroups.value = groupCheck.normalized
  } else if (!isBuiltinPlan.value && runTypeId.value !== 'adv_trigger_bet') {
    const groups = schemeGroups.value
      .map((g) => String(g ?? '').replace(/\r/g, ''))
      .filter((g) => g.trim())
    if (groups.length === 0) return '请至少填写一组方案内容'
  }

  return null
}

/** 是否需要填写「方案内容」选号（内置计画与高级开某投某除外） */
const requiresSchemeGroups = computed(
  () => !isBuiltinPlan.value && runTypeId.value !== 'adv_trigger_bet',
)

/** 同步收集未填写的必填项（直接读 ref，避免依赖表单校验 Promise） */
function collectMissingRequired(): string[] {
  const missing: string[] = []
  if (!schemeName.value.trim()) missing.push('方案名称')
  if (!lotteryCode.value) missing.push('彩种')
  if (!runTypeId.value) missing.push('运行类型')
  if (!isBuiltinPlan.value && !playTypeId.value) missing.push('玩法类型')
  if (!isBuiltinPlan.value && !subPlayId.value) missing.push('子玩法')
  if (
    !isBuiltinPlan.value &&
    subPlayId.value &&
    resolvedSubPlayOptions.value.length > 0 &&
    !resolvedSubPlayOptions.value.some((o) => o.value === subPlayId.value)
  ) {
    // 子玩法与当前玩法类型不匹配（残留旧值），要求重选
    missing.push('子玩法（请重新选择）')
  }
  if (!schemeFunds.value.trim()) missing.push('方案资金')
  if (!multCoeff.value.trim()) missing.push('倍数系数')
  if (!betMultiplier.value?.kind) missing.push('方案模式（倍投设定）')
  if (requiresSchemeGroups.value) {
    const filled = schemeGroups.value.filter((g) => String(g ?? '').trim())
    if (filled.length === 0) missing.push('方案内容（请在下方选号）')
  }
  return missing
}

async function onSubmit() {
  if (saving.value) {
    ElMessage.info('正在提交中，请稍候…')
    return
  }

  const missing = collectMissingRequired()
  if (missing.length > 0) {
    ElMessage.warning(`请填写：${missing.join('、')}`)
    return
  }

  const fundsNum = Number(schemeFunds.value.trim())
  if (!Number.isFinite(fundsNum) || fundsNum <= 0) {
    ElMessage.warning('方案资金必须大于 0')
    return
  }

  const multCoeffNum = Number(multCoeff.value.trim())
  if (!Number.isFinite(multCoeffNum) || multCoeffNum < 0 || !Number.isInteger(multCoeffNum)) {
    ElMessage.warning('倍数系数须为非负整数')
    return
  }

  let validationError: string | null
  try {
    validationError = validateBeforeSubmit()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '表单校验异常，请检查填写内容')
    return
  }
  if (validationError) {
    ElMessage.warning(validationError)
    return
  }

  saving.value = true
  try {
    const input = buildInput()
    if (isEdit.value && props.editSnapshot) {
      await store.updateShareSnapshot(props.editSnapshot.id, input)
      ElMessage.success('分享池方案已更新')
    } else {
      await store.createShareSnapshot(input)
      ElMessage.success('分享池方案已创建')
    }
    visible.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : isEdit.value ? '保存失败' : '创建失败')
  } finally {
    saving.value = false
  }
}

// 玩法选项来自 play tree
const playTreeTypes = ref<{ typeId: string; label: string; subPlays: { subId: string; label: string }[] }[]>([])

watch(lotteryCode, async (code) => {
  if (!code) {
    playTreeTypes.value = []
    return
  }
  await loadPlayConfig()
  const lot = lotteryRows.value.find((r) => r.code === code)
  const template = lot?.playTemplate?.trim()
  if (!template) {
    playTreeTypes.value = []
    return
  }
  try {
    const { fetchPlayTree } = await import('@/api/playCatalog')
    const tree = await fetchPlayTree(template)
    playTreeTypes.value = tree.playTypes ?? []
    if (resolvedPlayTypeOptions.value.length && !playTypeId.value) {
      playTypeId.value = playTreeTypes.value[0]?.typeId ?? ''
    }
  } catch {
    playTreeTypes.value = []
  }
})

const resolvedPlayTypeOptions = computed(() =>
  playTreeTypes.value.map((t) => ({ label: t.label, value: t.typeId })),
)

const resolvedSubPlayOptions = computed(() => {
  const type = playTreeTypes.value.find((t) => t.typeId === playTypeId.value)
  return (type?.subPlays ?? []).map((s) => ({ label: s.label, value: s.subId }))
})

// 玩法类型可选项变化（含玩法树异步加载完成）时，纠正无效/残留的玩法类型
watch(resolvedPlayTypeOptions, (opts) => {
  if (opts.length && !opts.some((o) => o.value === playTypeId.value)) {
    playTypeId.value = opts[0]?.value ?? ''
  }
})

// 子玩法可选项变化时，纠正不属于当前玩法类型的残留子玩法 ID（修复「玩法不存在」）
watch(resolvedSubPlayOptions, (opts) => {
  if (opts.length && !opts.some((o) => o.value === subPlayId.value)) {
    subPlayId.value = opts[0]?.value ?? ''
  }
})
</script>

<template>
  <AdminDialog v-model="visible" :title="dialogTitle" width="min(100%, 760px)" destroy-on-close>
    <p style="margin: 0 0 1rem; font-size: 13px; color: var(--el-text-color-secondary)">
      方案结构与会员端「自创方案 → 公开分享」一致，写入分享池供下载与跟单大厅选用。
    </p>

    <el-form ref="formRef" :model="validateModel" :rules="formRules" label-width="96px" @submit.prevent="onSubmit">
      <el-form-item v-if="isEdit" label="快照 ID">
        <el-input :model-value="editSnapshot?.id" disabled />
      </el-form-item>

      <el-form-item label="方案名称" prop="schemeName">
        <el-input v-model="schemeName" maxlength="128" show-word-limit placeholder="与会员端命名规则一致" />
      </el-form-item>

      <el-form-item label="彩种" prop="lotteryCode">
        <el-select v-model="lotteryCode" filterable placeholder="选择彩种" style="width: 100%">
          <el-option
            v-for="lot in lotteryOptions"
            :key="lot.code"
            :label="lot.displayName"
            :value="lot.code"
            :disabled="!isLotterySelectable(lot)"
          >
            <span :class="{ 'lottery-maint': lot.saleStatus !== 'on_sale' }">{{ lot.displayName }}</span>
          </el-option>
        </el-select>
      </el-form-item>

      <el-form-item label="运行类型" prop="runTypeId">
        <el-select v-model="runTypeId" style="width: 100%">
          <el-option v-for="opt in RUN_TYPE_OPTIONS" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
      </el-form-item>

      <template v-if="!isBuiltinPlan">
        <el-form-item label="玩法类型" prop="playTypeId">
          <el-select v-model="playTypeId" filterable placeholder="选择玩法类型" style="width: 100%">
            <el-option v-for="opt in resolvedPlayTypeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>

        <el-form-item label="子玩法" prop="subPlayId">
          <el-select v-model="subPlayId" filterable placeholder="选择子玩法" style="width: 100%">
            <el-option v-for="opt in resolvedSubPlayOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
      </template>

      <el-form-item label="投注通道">
        <el-radio-group v-model="runMode">
          <el-radio-button value="formal">正式</el-radio-button>
          <el-radio-button value="sim">模拟</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="方案资金" required>
        <el-input v-model="schemeFunds" type="number" :min="1" placeholder="例如 10000（须大于 0）" />
      </el-form-item>

      <el-form-item label="运行时段">
        <div style="display: flex; gap: 0.75rem; width: 100%; flex-wrap: wrap; align-items: center">
          <el-date-picker
            v-model="startTime"
            type="datetime"
            placeholder="开始时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            style="flex: 1; min-width: 200px"
          />
          <span style="color: var(--el-text-color-secondary)">至</span>
          <el-date-picker
            v-model="endTime"
            type="datetime"
            placeholder="结束时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            style="flex: 1; min-width: 200px"
          />
        </div>
        <p style="margin: 0.35rem 0 0; font-size: 12px; color: var(--el-text-color-secondary)">
          格式：年-月-日 时:分:秒；须同时填写或同时留空（无限期运行）
        </p>
      </el-form-item>

      <el-form-item label="止损 / 止盈">
        <div style="display: flex; gap: 0.75rem; width: 100%">
          <el-input v-model="stopLoss" placeholder="止损" />
          <el-input v-model="takeProfit" placeholder="止盈" />
        </div>
      </el-form-item>

      <el-divider content-position="left">投注参数</el-divider>

      <el-form-item label="倍数系数" prop="multCoeff">
        <el-input v-model="multCoeff" type="number" :min="0" :step="1" placeholder="例如 1" style="max-width: 160px" />
      </el-form-item>

      <el-form-item label="投注单位" required>
        <el-select v-model="betUnit" style="width: 160px">
          <el-option v-for="o in BET_MODE_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
      </el-form-item>

      <el-form-item label="方案模式" prop="betMultiplierKind">
        <div style="display: flex; flex-direction: column; gap: 0.35rem; align-items: flex-start">
          <el-button type="primary" plain @click="betMultiplierVisible = true">倍投设定</el-button>
          <span v-if="betMultiplierLabel" style="font-size: 13px; color: var(--el-color-primary)">
            已选：{{ betMultiplierLabel }}
          </span>
          <span v-else style="font-size: 13px; color: var(--el-color-warning)">未设置，请点击进入倍投设定</span>
        </div>
      </el-form-item>

      <el-form-item v-if="usesGroupContent && !isBuiltinPlan" label="方案内容" required>
        <div style="display: flex; flex-direction: column; gap: 1rem; width: 100%">
          <div
            v-for="(idx) in (runTypeId === 'fixed_number' ? [0] : schemeGroups.map((_, i) => i))"
            :key="idx"
            class="scheme-group-card"
          >
            <div class="scheme-group-head">
              <strong>{{ runTypeId === 'fixed_number' ? '固定取码' : `第 ${idx + 1} 组` }}</strong>
              <el-button
                v-if="runTypeId === 'fixed_rotate' && schemeGroups.length > 1"
                link
                type="danger"
                @click="removeGroup(idx)"
              >
                删除组
              </el-button>
            </div>
            <SchemeGroupInputPanel
              v-if="schemeUsesDigitInput"
              v-model="schemeGroups[idx]"
              :config="playConfig"
            />
            <SchemeGroupPickPanel
              v-else-if="schemeUsesPickPanel"
              v-model="schemeGroups[idx]"
              :config="playConfig"
            />
            <el-input
              v-else
              v-model="schemeGroups[idx]"
              type="textarea"
              :rows="5"
              resize="none"
              :placeholder="groupInputPlaceholder"
              style="margin-top: 0.5rem"
            />
            <p class="scheme-group-meta">
              <span v-if="schemeUsesDigitInput">{{ digitInputHint }}</span>
              <span v-else-if="schemeUsesPickPanel">
                {{
                  playConfig.inputMode === 'multiline'
                    ? '按位选号，每位多选以逗号保存'
                    : '点击号码选择，多选以逗号保存'
                }}
              </span>
              <span v-else>每注以逗号分隔</span>
              <span>注数: {{ groupBetUnits(schemeGroups[idx] ?? '') }}</span>
            </p>
          </div>
          <el-button v-if="runTypeId === 'fixed_rotate'" link type="primary" @click="addGroup">添加一组</el-button>
        </div>
      </el-form-item>

      <el-form-item v-else-if="!isBuiltinPlan && runTypeId !== 'adv_trigger_bet'" label="方案内容" required>
        <div style="display: flex; flex-direction: column; gap: 0.75rem; width: 100%">
          <p style="margin: 0; font-size: 12px; color: var(--el-text-color-secondary)">
            当前运行类型（{{ RUN_TYPE_OPTIONS.find((o) => o.value === runTypeId)?.label }}）的专用编辑器请在会员端配置后同步至分享池，此处可填写 JSON 兼容的文本内容。
          </p>
          <div v-for="(_, idx) in schemeGroups" :key="idx" style="display: flex; gap: 0.5rem; align-items: flex-start">
            <el-input
              v-model="schemeGroups[idx]"
              type="textarea"
              :rows="3"
              resize="none"
              :placeholder="`第 ${idx + 1} 组投注内容`"
            />
            <el-button v-if="schemeGroups.length > 1" link type="danger" @click="removeGroup(idx)">删除</el-button>
          </div>
          <el-button link type="primary" @click="addGroup">添加一组</el-button>
        </div>
      </el-form-item>
    </el-form>

    <ShareBetMultiplierDialog
      v-model="betMultiplierVisible"
      :model-payload="betMultiplier"
      @confirm="onBetMultiplierConfirm"
    />

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="onSubmit">{{ submitLabel }}</el-button>
    </template>
  </AdminDialog>
</template>

<style scoped>
.lottery-maint {
  color: var(--el-color-danger);
}

.scheme-group-card {
  padding: 0.75rem;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.scheme-group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.scheme-group-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin: 0.5rem 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
