<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import {
  getSchemeDefinition,
  updateSchemeDefinition,
  type SchemeDefinitionDto,
} from '@/api/schemes/definitions'
import { fetchLotterySchemeOptions } from '@/api/schemes/schemeOptions'
import { countBetUnits } from '@/utils/betPayload'
import { usePlayTreeConfig } from '@/composables/usePlayTreeConfig'
import DateTimePickerModal from '@/components/ui/DateTimePickerModal.vue'
import { BET_MODE_OPTIONS, betModeLabelOf, betUnitFromSchemeConfig, normalizeBetUnitValue } from '@/constants/betModeOptions'
import { normalizeSchemeTimePairFromConfig, schemeTimeRangeError } from '@/utils/schemeDateTime'
import { simBetFromLegacyRunMode, simBetFromSchemeConfig, simBetLabel } from '@/utils/schemeSimBet'

/**
 * 方案详情：合并「新增方案 + 方案配置」全部字段。
 * 由云端中心运行方案点击进入；运行时数据(投注流水/本次盈亏/倍数系数)由列表页 query 带入，
 * 配置数据由 definitionId 加载。点击「编辑」整页进入编辑模式。
 */

const route = useRoute()
const router = useRouter()

const definitionId = String(route.params.definitionId ?? route.query.definitionId ?? '').trim()

const loading = ref(false)
const saving = ref(false)
const editing = ref(false)

// 运行时数据（来自云端中心列表 query）
const turnover = ref(String(route.query.turnover ?? '0'))
const sessionPnl = ref(String(route.query.sessionPnl ?? '0'))
/** 方案状态：运行中禁止编辑 */
const schemeStatus = ref(String(route.query.status ?? ''))
const isRunning = computed(() => schemeStatus.value === 'running')

// 只读信息
const schemeName = ref('')
const lotteryLabel = ref('')
const betMultiplierKind = ref('')
const multCoeff = ref(String(route.query.multiplier ?? ''))
const betUnit = ref('')

// 可编辑（运行设置）
const simBet = ref(false)
const startTime = ref('')
const endTime = ref('')
const stopLoss = ref('')
const takeProfit = ref('')

// 方案资金计算所需
const lotteryCode = ref('')
const playTypeId = ref('')
const subPlayId = ref('')
const schemeGroups = ref<string[]>([])

const { playConfig: schemePlayConfig, load: loadPlayTree } = usePlayTreeConfig(
  lotteryCode,
  playTypeId,
  subPlayId,
)


const simBetModeSelect = computed({
  get: () => (simBet.value ? 'sim' : 'prod'),
  set: (v: 'prod' | 'sim') => {
    simBet.value = v === 'sim'
  },
})
const BET_MULTIPLIER_LABELS: Record<string, string> = {
  '0': '小白倍投',
  '1': '一键倍投',
  '2': '简单倍投',
  '3': '高级倍投',
}

function asString(v: unknown): string {
  return v == null ? '' : String(v)
}

/** 编辑草稿：前往倍投设定页前暂存，返回后恢复（避免重新挂载丢失未保存编辑） */
const DRAFT_KEY = `scheme-detail-draft:${definitionId}`

function saveDraft(): void {
  try {
    sessionStorage.setItem(
      DRAFT_KEY,
      JSON.stringify({
        simBet: simBet.value,
        startTime: startTime.value,
        endTime: endTime.value,
        stopLoss: stopLoss.value,
        takeProfit: takeProfit.value,
        betUnit: betUnit.value,
      }),
    )
  } catch {
    /* ignore */
  }
}

function applyDraftIfAny(): boolean {
  try {
    const raw = sessionStorage.getItem(DRAFT_KEY)
    if (!raw) return false
    sessionStorage.removeItem(DRAFT_KEY)
    const d = JSON.parse(raw) as Record<string, unknown>
    if (typeof d.simBet === 'boolean') simBet.value = d.simBet
    else {
      const legacy = simBetFromLegacyRunMode(d.runMode)
      if (legacy !== undefined) simBet.value = legacy
    }
    const draftTimes = normalizeSchemeTimePairFromConfig(d.startTime, d.endTime)
    startTime.value = draftTimes.start
    endTime.value = draftTimes.end
    if (typeof d.stopLoss === 'string') stopLoss.value = d.stopLoss
    if (typeof d.takeProfit === 'string') takeProfit.value = d.takeProfit
    if (typeof d.betUnit === 'string') betUnit.value = normalizeBetUnitValue(d.betUnit)
    else if (typeof d.betMode === 'string') betUnit.value = normalizeBetUnitValue(d.betMode)
    return true
  } catch {
    return false
  }
}

const betMultiplierLabel = ref('未设置')
const betUnitLabel = computed(() => betModeLabelOf(betUnit.value))

/** 注数 = 各组内容按玩法自动计算之和 */
const betCount = computed(() => {
  const cfg = schemePlayConfig.value
  return schemeGroups.value.reduce((sum, g) => sum + countBetUnits(cfg, g), 0)
})

const betUnitAmount = computed(() => Number(betUnit.value) || 0)

/** 方案资金 = 注数 × 投注单位 */
const schemeFundsDisplay = computed(() => {
  const total = betCount.value * betUnitAmount.value
  if (!Number.isFinite(total) || total <= 0) return '—'
  const fixed = Number(total.toFixed(3))
  return `${fixed} 元`
})

async function applyOptionLabels(lotteryCode: string): Promise<void> {
  if (!lotteryCode) return
  try {
    await fetchLotterySchemeOptions(lotteryCode)
  } catch {
    /* ignore */
  }
}

async function load(): Promise<void> {
  if (!definitionId) {
    ElMessage.error('缺少方案标识')
    return
  }
  loading.value = true
  try {
    const d: SchemeDefinitionDto = await getSchemeDefinition(definitionId)
    schemeName.value = d.schemeName
    lotteryLabel.value = d.lotteryLabel || d.lotteryCode
    const cfg = (d.config ?? {}) as Record<string, unknown>
    simBet.value = simBetFromSchemeConfig(cfg)
    lotteryCode.value = d.lotteryCode
    playTypeId.value = asString(cfg.playTypeId ?? cfg.typeId)
    subPlayId.value = asString(cfg.subPlayId ?? cfg.subId)
    schemeGroups.value = Array.isArray(cfg.schemeGroups)
      ? cfg.schemeGroups.map((g) => asString(g))
      : []
    const times = normalizeSchemeTimePairFromConfig(cfg.startTime, cfg.endTime)
    startTime.value = times.start
    endTime.value = times.end
    stopLoss.value = cfg.stopLoss != null ? asString(cfg.stopLoss) : ''
    takeProfit.value = cfg.takeProfit != null ? asString(cfg.takeProfit) : ''
    if (cfg.multCoeff != null && asString(cfg.multCoeff) !== '') multCoeff.value = asString(cfg.multCoeff)
    betUnit.value = betUnitFromSchemeConfig(cfg)
    const bm = (cfg.betMultiplier ?? {}) as Record<string, unknown>
    betMultiplierKind.value = asString(bm.kind)
    // 从倍投设定页返回时，优先用 query.bmsKind 回显刚选择的 tab
    const bmsKind = String(route.query.bmsKind ?? '')
    if (bmsKind && BET_MULTIPLIER_LABELS[bmsKind]) {
      betMultiplierKind.value = bmsKind
      if (!isRunning.value) editing.value = true
    }
    betMultiplierLabel.value = BET_MULTIPLIER_LABELS[betMultiplierKind.value] ?? '未设置'
    // 恢复未保存编辑草稿（从倍投设定页返回时），保持编辑模式
    if (applyDraftIfAny() && !isRunning.value) editing.value = true
    await loadPlayTree()
    await applyOptionLabels(d.lotteryCode)
  } catch (e) {
    ElMessage.error(e instanceof ApiError ? e.message : '加载方案详情失败')
  } finally {
    loading.value = false
  }
}

// 日期时间选择弹窗
const timePickerVisible = ref(false)
const timePickerField = ref<'start' | 'end'>('start')
const timePickerValue = ref('')

function openTimePicker(field: 'start' | 'end'): void {
  timePickerField.value = field
  timePickerValue.value = field === 'start' ? startTime.value : endTime.value
  timePickerVisible.value = true
}

function onTimePicked(dt: string): void {
  if (timePickerField.value === 'start') startTime.value = dt
  else endTime.value = dt
}

function toggleEdit(): void {
  if (isRunning.value) {
    ElMessage.warning('运行中的方案不可编辑，请先暂停后再修改')
    return
  }
  if (editing.value) void onSave()
  else editing.value = true
}

async function onSave(): Promise<void> {
  if (!definitionId) return
  const timeErr = schemeTimeRangeError(startTime.value, endTime.value)
  if (timeErr) {
    ElMessage.warning(timeErr)
    return
  }
  saving.value = true
  try {
    await updateSchemeDefinition(definitionId, {
      simBet: simBet.value,
      startTime: startTime.value,
      endTime: endTime.value,
      stopLoss: stopLoss.value,
      takeProfit: takeProfit.value,
      betUnit: betUnit.value,
    })
    editing.value = false
    ElMessage.success('方案已保存')
  } catch (e) {
    ElMessage.error(e instanceof ApiError ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

/** 编辑模式下，方案模式跳转到倍投设定页（返回时回到本页并回显所选 tab） */
function gotoBetMultiplier(): void {
  saveDraft()
  const cfg = schemePlayConfig.value
  void router.push({
    name: 'bet-multiplier-settings',
    query: {
      schemeId: definitionId,
      fromScheme: '1',
      returnName: 'scheme-detail',
      activeTab: betMultiplierKind.value || '2',
      title: encodeURIComponent(schemeName.value),
      playType: playTypeId.value || cfg.playTypeId || '',
      subPlay: subPlayId.value || cfg.subPlayId || '',
      betMode: cfg.betMode || '',
      playTypeLabel: cfg.playTypeLabel || '',
      subPlayLabel: cfg.playMethodLabel || '',
      playTemplate: cfg.playTemplate || '',
      ...(cfg.segmentLen ? { segmentLen: String(cfg.segmentLen) } : {}),
      ...(lotteryCode.value ? { lottery: lotteryCode.value } : {}),
    },
  })
}

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'cloud' })
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="sd" data-page="scheme-detail">
    <header class="sd-head">
      <button type="button" class="sd-back" aria-label="返回" @click="goBack">
        <span class="material-sym">arrow_back_ios_new</span>
      </button>
      <h1 class="sd-title">方案详情</h1>
      <el-button
        type="primary"
        :plain="!editing"
        size="small"
        class="sd-edit-btn"
        :loading="saving"
        :disabled="isRunning"
        @click="toggleEdit"
      >
        {{ isRunning ? '运行中' : editing ? '完成' : '编辑' }}
      </el-button>
    </header>

    <main class="sd-main">
      <section class="sd-card">
        <!-- 模式 -->
        <div class="sd-field">
          <span class="sd-label">模式</span>
          <el-select v-if="editing" v-model="simBetModeSelect" size="small" class="sd-edit-ctrl">
            <el-option label="正式" value="prod" />
            <el-option label="模拟" value="sim" />
          </el-select>
          <span v-else class="sd-value">{{ simBetLabel(simBet) }}</span>
        </div>
        <!-- 方案id -->
        <div class="sd-field">
          <span class="sd-label">方案ID</span>
          <span class="sd-value sd-value--mono">{{ definitionId || '—' }}</span>
        </div>
        <!-- 方案名称 -->
        <div class="sd-field">
          <span class="sd-label">方案名称</span>
          <span class="sd-value sd-value--strong">{{ schemeName || '—' }}</span>
        </div>
        <!-- 方案资金 -->
        <div class="sd-field">
          <span class="sd-label">方案资金</span>
          <span class="sd-value">{{ schemeFundsDisplay }}</span>
        </div>
        <!-- 彩种 -->
        <div class="sd-field">
          <span class="sd-label">彩种</span>
          <span class="sd-value">{{ lotteryLabel || '—' }}</span>
        </div>
        <!-- 投注流水 -->
        <div class="sd-field">
          <span class="sd-label">投注流水</span>
          <span class="sd-value">{{ turnover }}</span>
        </div>
        <!-- 本次盈亏 -->
        <div class="sd-field">
          <span class="sd-label">本次盈亏</span>
          <span
            class="sd-value"
            :class="Number(sessionPnl) > 0 ? 'sd-up' : Number(sessionPnl) < 0 ? 'sd-down' : ''"
          >{{ sessionPnl }}</span>
        </div>
        <!-- 开始时间 -->
        <div class="sd-field">
          <span class="sd-label">开始时间</span>
          <button v-if="editing" type="button" class="sd-time-btn" @click="openTimePicker('start')">
            {{ startTime || '选择时间' }}
            <span class="material-sym sd-time-ico">event</span>
          </button>
          <span v-else class="sd-value">{{ startTime || '—' }}</span>
        </div>
        <!-- 结束时间 -->
        <div class="sd-field">
          <span class="sd-label">结束时间</span>
          <button v-if="editing" type="button" class="sd-time-btn" @click="openTimePicker('end')">
            {{ endTime || '选择时间' }}
            <span class="material-sym sd-time-ico">event</span>
          </button>
          <span v-else class="sd-value">{{ endTime || '—' }}</span>
        </div>
        <!-- 方案止损 -->
        <div class="sd-field">
          <span class="sd-label">方案止损</span>
          <el-input v-if="editing" v-model="stopLoss" size="small" class="sd-edit-ctrl" placeholder="不限" />
          <span v-else class="sd-value">{{ stopLoss || '不限' }}</span>
        </div>
        <!-- 方案止盈 -->
        <div class="sd-field">
          <span class="sd-label">方案止盈</span>
          <el-input v-if="editing" v-model="takeProfit" size="small" class="sd-edit-ctrl" placeholder="不限" />
          <span v-else class="sd-value">{{ takeProfit || '不限' }}</span>
        </div>
        <!-- 倍数系数 -->
        <div class="sd-field">
          <span class="sd-label">倍数系数</span>
          <span class="sd-value">{{ multCoeff || '—' }}</span>
        </div>
        <!-- 投注单位 -->
        <div class="sd-field">
          <span class="sd-label">投注单位</span>
          <el-select v-if="editing" v-model="betUnit" size="small" class="sd-edit-ctrl">
            <el-option v-for="o in BET_MODE_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
          <span v-else class="sd-value">{{ betUnitLabel }}</span>
        </div>
        <!-- 方案模式 -->
        <div class="sd-field">
          <span class="sd-label">方案模式</span>
          <button v-if="editing" type="button" class="sd-link-btn" @click="gotoBetMultiplier">
            {{ betMultiplierLabel }}
            <span class="material-sym sd-link-arrow">chevron_right</span>
          </button>
          <span v-else class="sd-value">{{ betMultiplierLabel }}</span>
        </div>
      </section>
    </main>

    <DateTimePickerModal
      v-model="timePickerVisible"
      :value="timePickerValue"
      :title="timePickerField === 'start' ? '开始时间' : '结束时间'"
      @confirm="onTimePicked"
    />
  </div>
</template>

<style scoped>
.sd {
  min-height: 100dvh;
  background: #f7f9fb;
  color: #191c1e;
  padding-bottom: calc(1.5rem + env(safe-area-inset-bottom));
}

.sd-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--page-titlebar-height);
  min-height: var(--page-titlebar-height);
  box-sizing: border-box;
  padding: 0 1.25rem;
  background: #fff;
  box-shadow: 0 8px 24px -16px rgba(15, 35, 95, 0.18);
}

.sd-back {
  display: grid;
  place-items: center;
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  border-radius: 0.65rem;
  color: #0f172a;
  background: #f1f5f9;
}

.sd-back .material-sym {
  font-size: var(--page-titlebar-icon-size);
}

.sd-title {
  margin: 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 1.0625rem;
}

.sd-edit-btn {
  font-weight: 600;
}

.sd-main {
  padding: 1.25rem;
  max-width: 32rem;
  margin: 0 auto;
}

.sd-card {
  background: #fff;
  border-radius: 1.15rem;
  padding: 0.4rem 1.15rem;
  box-shadow: 0 18px 44px -28px rgba(15, 35, 95, 0.16);
}

.sd-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid #f1f5f9;
}

.sd-field:last-child {
  border-bottom: none;
}

.sd-label {
  font-size: 0.875rem;
  color: #64748b;
  flex-shrink: 0;
}

.sd-value {
  font-size: 0.9375rem;
  color: #0f172a;
  text-align: right;
  word-break: break-all;
}

.sd-value--strong {
  font-weight: 700;
}

.sd-value--mono {
  font-family: 'Inter', monospace;
  font-size: 0.8125rem;
  color: #475569;
}

.sd-up {
  color: #dc2626;
  font-weight: 700;
}

.sd-down {
  color: #16a34a;
  font-weight: 700;
}

.sd-edit-ctrl {
  max-width: 11rem;
}

.sd-time-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.7rem;
  border-radius: 10px;
  background: var(--el-color-primary-light-9, #e6ebfa);
  color: var(--el-color-primary, #0050cb);
  font-size: 0.875rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.sd-time-ico {
  font-size: 1rem;
}

.sd-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.1rem;
  padding: 0.2rem 0.5rem;
  border-radius: 8px;
  background: var(--el-color-primary-light-9, #e6ebfa);
  color: var(--el-color-primary, #0050cb);
  font-size: 0.9375rem;
  font-weight: 600;
}

.sd-link-arrow {
  font-size: 1.05rem;
}
</style>
