<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import {
  getSchemeDefinition,
  type SchemeDefinitionDto,
} from '@/api/schemes/definitions'
import { fetchLotterySchemeOptions } from '@/api/schemes/schemeOptions'
import { countBetUnits } from '@/utils/betPayload'
import { usePlayTreeConfig } from '@/composables/usePlayTreeConfig'
import SchemeContentReadonlyPanel from '@/components/schemes/SchemeContentReadonlyPanel.vue'
import { betModeLabelOf, betUnitFromSchemeConfig } from '@/constants/betModeOptions'
import { normalizeSchemeTimePairFromConfig } from '@/utils/schemeDateTime'
import { simBetFromSchemeConfig, simBetLabel } from '@/utils/schemeSimBet'
import { formatSubPlayLabel } from '@/utils/playConfig'
import type {
  SchemeFixedPick,
  SchemeHotColdPickType,
  SchemeHotColdWarm,
  SchemeJushuRow,
  SchemeRandomDraw,
  SchemeTriggerBet,
} from '@/api/schemes/definitions'

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
  builtin_plan: '内置计画',
  fixed_number: '固定取码',
}

function normalizeRunTypeId(raw: unknown): RunTypeId {
  const v = String(Array.isArray(raw) ? raw[0] ?? '' : raw ?? '').trim()
  if ((RUN_TYPE_IDS as readonly string[]).includes(v)) return v as RunTypeId
  return 'fixed_rotate'
}

/**
 * 方案详情：只读展示；运行时数据由列表 query 带入，配置由 definitionId 加载。
 * 点击「编辑」跳转完整方案配置页（AdvancedSchemeEditView）。
 */

const route = useRoute()
const router = useRouter()

const definitionId = String(route.params.definitionId ?? route.query.definitionId ?? '').trim()

const loading = ref(false)

// 运行时数据（来自云端中心列表 query）
const turnover = ref(String(route.query.turnover ?? '0'))
const sessionPnl = ref(String(route.query.sessionPnl ?? '0'))
/** 方案状态：运行中禁止编辑 */
const schemeStatus = ref(String(route.query.status ?? ''))
const isRunning = computed(() => schemeStatus.value === 'running')

const schemeName = ref('')
const lotteryLabel = ref('')
const betMultiplierKind = ref('')
const queryMultiplier = String(route.query.multiplier ?? '')
const multCoeff = ref(queryMultiplier)
const betUnit = ref('')
const runTypeId = ref<RunTypeId>('fixed_rotate')
const runTypeLabel = computed(() => RUN_TYPE_LABELS[runTypeId.value])

const lotteryRunTypeDisplay = computed(() => {
  const lottery = (lotteryLabel.value || '').trim()
  const runType = (runTypeLabel.value || '').trim()
  if (lottery && runType) return `${lottery} · ${runType}`
  return lottery || runType || '—'
})

const simBet = ref(false)
const startTime = ref('')
const endTime = ref('')
const stopLoss = ref('')
const takeProfit = ref('')

const schemeFunds = ref('')
const lotteryCode = ref('')
const playTypeId = ref('')
const subPlayId = ref('')
const schemeGroups = ref<string[]>([])
const jushuList = ref<SchemeJushuRow[]>([])
const triggerBet = ref<SchemeTriggerBet | null>(null)
const hotColdWarm = ref<SchemeHotColdWarm | null>(null)
const randomDraw = ref<SchemeRandomDraw | null>(null)
const fixedPick = ref<SchemeFixedPick | null>(null)
const builtinPlanSnapshotId = ref('')
const cachedPlayTypeLabel = ref('')
const cachedSubPlayLabel = ref('')

function asTriggerBet(raw: unknown): SchemeTriggerBet | null {
  if (!raw || typeof raw !== 'object') return null
  const tb = raw as Record<string, unknown>
  const rows: SchemeTriggerBet['rows'] = []
  if (Array.isArray(tb.rows)) {
    for (const item of tb.rows) {
      if (!item || typeof item !== 'object') continue
      const r = item as Record<string, unknown>
      rows.push({
        enabled: r.enabled !== false,
        open: asString(r.open),
        pos: asString(r.pos),
        neg: asString(r.neg),
      })
    }
  }
  const mode = asString(tb.mode) as SchemeTriggerBet['mode']
  const okMode =
    mode === 'always_pos' || mode === 'always_neg' || mode === 'alt_pos_first' || mode === 'alt_neg_first'
      ? mode
      : 'always_pos'
  const positionIdxs: number[] = []
  if (Array.isArray(tb.positionIdxs)) {
    for (const n of tb.positionIdxs) {
      const i = Number(n)
      if (Number.isInteger(i) && i >= 0 && !positionIdxs.includes(i)) positionIdxs.push(i)
    }
  } else if (tb.positionIdx != null) {
    const i = Number(tb.positionIdx)
    if (Number.isInteger(i) && i >= 0) positionIdxs.push(i)
  }
  positionIdxs.sort((a, b) => a - b)
  return {
    rows,
    mode: okMode,
    ...(positionIdxs.length ? { positionIdxs } : {}),
  }
}

function asJushuList(raw: unknown): SchemeJushuRow[] {
  if (!Array.isArray(raw)) return []
  const out: SchemeJushuRow[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const r = item as Record<string, unknown>
    const ju = Number(r.ju)
    if (!Number.isFinite(ju) || ju <= 0) continue
    out.push({
      ju: Math.trunc(ju),
      content: asString(r.content),
      afterHit: Math.max(1, Math.trunc(Number(r.afterHit) || 1)),
      afterMiss: Math.max(1, Math.trunc(Number(r.afterMiss) || 1)),
    })
  }
  return out.sort((a, b) => a.ju - b.ju)
}

const { playConfig: schemePlayConfig, load: loadPlayTree } = usePlayTreeConfig(
  lotteryCode,
  playTypeId,
  subPlayId,
)

const playDisplay = computed(() => {
  const cfg = schemePlayConfig.value as {
    playTypeLabel?: string
    playMethodLabel?: string
  }
  const typeLabel = (cfg.playTypeLabel || cachedPlayTypeLabel.value || '').trim()
  const subLabel = formatSubPlayLabel(cfg.playMethodLabel || cachedSubPlayLabel.value || '').trim()
  if (typeLabel && subLabel) return `${typeLabel} · ${subLabel}`
  if (typeLabel) return typeLabel
  if (subLabel) return subLabel
  return '—'
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

function limitDisplay(v: string): string {
  const s = (v || '').trim()
  if (!s) return '不限'
  const n = Number(s)
  if (Number.isFinite(n) && n <= 0) return '不限'
  return s
}

const betMultiplierLabel = ref('未设置')
const betUnitLabel = computed(() => betModeLabelOf(betUnit.value))

const runTimeDisplay = computed(() => {
  const s = startTime.value.trim()
  const e = endTime.value.trim()
  if (!s && !e) return '-'
  return `${s}-${e}`
})

const betCount = computed(() => {
  const cfg = schemePlayConfig.value
  return schemeGroups.value.reduce((sum, g) => sum + countBetUnits(cfg, g), 0)
})

const betUnitAmount = computed(() => Number(betUnit.value) || 0)

const schemeFundsDisplay = computed(() => {
  const entered = Number(String(schemeFunds.value).trim())
  if (Number.isFinite(entered) && entered > 0) {
    return `${Number(entered.toFixed(3))} 元`
  }
  const total = betCount.value * betUnitAmount.value
  if (!Number.isFinite(total) || total <= 0) return '—'
  const fixed = Number(total.toFixed(3))
  return `${fixed} 元`
})

/** 倍数系数：优先配置 multCoeff，否则回退列表实例倍数 */
const multCoeffDisplay = computed(() => {
  const cfg = (multCoeff.value || '').trim()
  if (cfg) return cfg
  return queryMultiplier || '—'
})

async function applyOptionLabels(code: string): Promise<void> {
  if (!code) return
  try {
    await fetchLotterySchemeOptions(code)
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
    schemeFunds.value = asString(cfg.schemeFunds)
    runTypeId.value = normalizeRunTypeId(cfg.runTypeId)
    playTypeId.value = asString(cfg.playTypeId ?? cfg.typeId)
    subPlayId.value = asString(cfg.subPlayId ?? cfg.subId)
    cachedPlayTypeLabel.value = asString(cfg.playTypeLabel)
    cachedSubPlayLabel.value = asString(cfg.playMethodLabel ?? cfg.subPlayLabel)
    schemeGroups.value = Array.isArray(cfg.schemeGroups)
      ? cfg.schemeGroups.map((g) => asString(g))
      : []
    jushuList.value = asJushuList(cfg.jushuList)
    triggerBet.value = asTriggerBet(cfg.triggerBet)
    {
      const hcw = cfg.hotColdWarm
      if (hcw && typeof hcw === 'object') {
        const h = hcw as Record<string, unknown>
        const st = asString(h.strategy)
        const strategy: SchemeHotColdWarm['strategy'] =
          st === 'every' || st === 'keep' || st === 'after_hit' || st === 'after_miss'
            ? st
            : h.winRotate === true
              ? 'after_hit'
              : 'keep'
        const pickTypes: SchemeHotColdPickType[] = []
        if (Array.isArray(h.pickTypes)) {
          for (const t of h.pickTypes) {
            const s = asString(t).toLowerCase()
            if ((s === 'hot' || s === 'cold') && !pickTypes.includes(s)) pickTypes.push(s)
          }
        }
        const fc = Math.trunc(Number(h.faultCount))
        const faultCount =
          Number.isFinite(fc) && fc >= 1 ? Math.min(10, fc) : undefined
        hotColdWarm.value = {
          totalPeriods: Math.max(1, Math.trunc(Number(h.totalPeriods) || 20)),
          pool: Array.isArray(h.pool) ? h.pool.map((p) => asString(p)) : [],
          strategy,
          pickTypes: pickTypes.length ? pickTypes : undefined,
          faultCount,
          winRotate: strategy === 'after_hit',
        }
      } else {
        hotColdWarm.value = null
      }
    }
    {
      const rd = cfg.randomDraw
      if (rd && typeof rd === 'object') {
        const r = rd as Record<string, unknown>
        const st = asString(r.strategy)
        const strategy: SchemeRandomDraw['strategy'] =
          st === 'keep' || st === 'after_hit' || st === 'after_miss' ? st : 'every'
        randomDraw.value = {
          counts: Array.isArray(r.counts)
            ? r.counts.map((n) => Math.max(1, Math.trunc(Number(n) || 1)))
            : [],
          strategy,
        }
      } else {
        randomDraw.value = null
      }
    }
    {
      const fp = cfg.fixedPick
      if (fp && typeof fp === 'object' && Array.isArray((fp as { rules?: unknown }).rules)) {
        const rules = ((fp as { rules: unknown[] }).rules ?? [])
          .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
          .map((r) => ({
            posStart: Math.max(0, Math.trunc(Number(r.posStart) || 0)),
            posEnd: Math.max(0, Math.trunc(Number(r.posEnd) || 0)),
            codeMin: Math.trunc(Number(r.codeMin) || 0),
            codeMax: Math.trunc(Number(r.codeMax) || 0),
            numbers: asString(r.numbers),
          }))
        fixedPick.value = { rules }
      } else {
        fixedPick.value = null
      }
    }
    {
      const bp = cfg.builtinPlan
      builtinPlanSnapshotId.value =
        bp && typeof bp === 'object' ? asString((bp as Record<string, unknown>).snapshotId) : ''
    }
    const times = normalizeSchemeTimePairFromConfig(cfg.startTime, cfg.endTime)
    startTime.value = times.start
    endTime.value = times.end
    stopLoss.value = cfg.stopLoss != null ? asString(cfg.stopLoss) : ''
    takeProfit.value = cfg.takeProfit != null ? asString(cfg.takeProfit) : ''
    if (cfg.multCoeff != null && asString(cfg.multCoeff) !== '') {
      multCoeff.value = asString(cfg.multCoeff)
    } else {
      multCoeff.value = ''
    }
    betUnit.value = betUnitFromSchemeConfig(cfg)
    const bm = (cfg.betMultiplier ?? {}) as Record<string, unknown>
    betMultiplierKind.value = asString(bm.kind)
    betMultiplierLabel.value = BET_MULTIPLIER_LABELS[betMultiplierKind.value] ?? '未设置'
    await loadPlayTree()
    await applyOptionLabels(d.lotteryCode)
  } catch (e) {
    ElMessage.error(e instanceof ApiError ? e.message : '加载方案详情失败')
  } finally {
    loading.value = false
  }
}

function runtimeQuery(): Record<string, string> {
  const q: Record<string, string> = {}
  if (turnover.value) q.turnover = turnover.value
  if (sessionPnl.value) q.sessionPnl = sessionPnl.value
  if (queryMultiplier) q.multiplier = queryMultiplier
  if (schemeStatus.value) q.status = schemeStatus.value
  return q
}

function gotoEdit(): void {
  if (isRunning.value) {
    ElMessage.warning('运行中的方案不可编辑，请先暂停后再修改')
    return
  }
  if (!definitionId) {
    ElMessage.error('缺少方案标识')
    return
  }
  // replace：避免 [详情→编辑→详情] 叠栈后与 history.back 互相顶，回不了云端中心
  void router.replace({
    name: 'advanced-scheme-edit',
    params: { schemeId: definitionId },
    query: {
      returnName: 'scheme-detail',
      ...runtimeQuery(),
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
  <div class="sd" data-page="scheme-detail" v-loading="loading">
    <header class="sd-head">
      <button type="button" class="sd-back" aria-label="返回" @click="goBack">
        <span class="material-sym">arrow_back_ios_new</span>
      </button>
      <h1 class="sd-title">方案详情</h1>
      <el-button
        type="primary"
        plain
        size="small"
        class="sd-edit-btn"
        :disabled="isRunning"
        @click="gotoEdit"
      >
        {{ isRunning ? '运行中' : '编辑' }}
      </el-button>
    </header>

    <main class="sd-main">
      <section class="sd-card">
        <div class="sd-field">
          <span class="sd-label">模式</span>
          <span class="sd-value">{{ simBetLabel(simBet) }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">方案ID</span>
          <span class="sd-value sd-value--mono">{{ definitionId || '—' }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">方案名称</span>
          <span class="sd-value sd-value--strong">{{ schemeName || '—' }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">方案资金</span>
          <span class="sd-value">{{ schemeFundsDisplay }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">彩种</span>
          <span class="sd-value">{{ lotteryRunTypeDisplay }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">玩法</span>
          <span class="sd-value">{{ playDisplay }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">投注流水</span>
          <span class="sd-value">{{ turnover }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">本次盈亏</span>
          <span
            class="sd-value"
            :class="Number(sessionPnl) > 0 ? 'sd-up' : Number(sessionPnl) < 0 ? 'sd-down' : ''"
          >{{ sessionPnl }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">运行时间</span>
          <span class="sd-value">{{ runTimeDisplay }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">方案止损</span>
          <span class="sd-value">{{ limitDisplay(stopLoss) }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">方案止盈</span>
          <span class="sd-value">{{ limitDisplay(takeProfit) }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">倍数系数</span>
          <span class="sd-value">{{ multCoeffDisplay }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">投注单位</span>
          <span class="sd-value">{{ betUnitLabel }}</span>
        </div>
        <div class="sd-field">
          <span class="sd-label">方案模式</span>
          <span class="sd-value">{{ betMultiplierLabel }}</span>
        </div>
      </section>

      <SchemeContentReadonlyPanel
        class="sd-content-panel"
        :run-type-id="runTypeId"
        :run-type-label="runTypeLabel"
        :play-config="schemePlayConfig"
        :scheme-groups="schemeGroups"
        :jushu-list="jushuList"
        :trigger-bet="triggerBet"
        :hot-cold-warm="hotColdWarm"
        :random-draw="randomDraw"
        :fixed-pick="fixedPick"
        :builtin-plan-snapshot-id="builtinPlanSnapshotId"
        :scheme-name="schemeName"
        :lottery-code="lotteryCode"
      />
    </main>
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
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
  position: sticky;
  top: 0;
  z-index: 20;
}

.sd-back {
  display: grid;
  place-items: center;
  width: 2.25rem;
  height: 2.25rem;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: #191c1e;
  cursor: pointer;
}

.sd-back .material-sym {
  font-size: var(--page-titlebar-back-icon-size);
}

.sd-title {
  margin: 0;
  flex: 1;
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.05rem;
  font-weight: 700;
  color: #191c1e;
}

.sd-edit-btn {
  min-width: 3.5rem;
}

.sd-main {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 1rem 1.25rem;
}

.sd-card {
  background: #fff;
  border-radius: 0.875rem;
  padding: 0.35rem 1rem;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.sd-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 2.75rem;
  padding: 0.35rem 0;
  border-bottom: 1px solid rgba(242, 244, 246, 0.95);
}

.sd-field:last-child {
  border-bottom: none;
}

.sd-label {
  flex: none;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #727687;
}

.sd-value {
  min-width: 0;
  text-align: right;
  font-size: 0.875rem;
  font-weight: 600;
  color: #191c1e;
  word-break: break-all;
}

.sd-value--mono {
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  font-size: 0.8125rem;
}

.sd-value--strong {
  font-weight: 700;
}

.sd-up {
  color: #e53935;
}

.sd-down {
  color: #2e7d32;
}

.sd-content-panel {
  margin-top: 0.15rem;
}
</style>
