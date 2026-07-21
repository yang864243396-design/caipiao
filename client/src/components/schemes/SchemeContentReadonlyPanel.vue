<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { fetchGameDraws } from '@/api/games/detail'
import { fetchHotColdWarmTiers } from '@/api/schemes/definitions'
import type { PlayConfig } from '@/utils/betPayload'
import { countBetUnits, playConfigSummary } from '@/utils/betPayload'
import {
  schemeGroupContentToInputBox,
  schemeGroupUsesDigitInput,
  textPickOptionsForConfig,
} from '@/utils/pickPanelOptions'
import { isLonghuPlayConfigLike } from '@/utils/runTypeMatrix'
import type {
  SchemeFixedPick,
  SchemeHotColdWarm,
  SchemeJushuRow,
  SchemeRandomDraw,
  SchemeTriggerBet,
  SchemeTriggerRow,
} from '@/api/schemes/definitions'

const props = defineProps<{
  runTypeId: string
  runTypeLabel: string
  playConfig: PlayConfig
  schemeGroups: string[]
  jushuList: SchemeJushuRow[]
  triggerBet: SchemeTriggerBet | null
  hotColdWarm: SchemeHotColdWarm | null
  randomDraw: SchemeRandomDraw | null
  fixedPick: SchemeFixedPick | null
  builtinPlanSnapshotId?: string
  schemeName?: string
  /** 冷热出号拉统计用 */
  lotteryCode?: string
}>()

const POSITION_FALLBACK = ['万位', '千位', '百位', '十位', '个位'] as const
const ALL_DIGITS = Array.from({ length: 10 }, (_, i) => String(i))

const TRIGGER_MODE_OPTIONS = [
  { label: '一直正投', value: 'always_pos' },
  { label: '一直反投', value: 'always_neg' },
  { label: '前正后反', value: 'alt_pos_first' },
  { label: '前反后正', value: 'alt_neg_first' },
] as const

const RD_STRATEGY_OPTIONS = [
  { label: '每期换号', value: 'every' },
  { label: '不换号', value: 'keep' },
  { label: '中后换号', value: 'after_hit' },
  { label: '挂后换号', value: 'after_miss' },
] as const

const HCW_STRATEGY_OPTIONS = [
  { label: '每期换', value: 'every' },
  { label: '不换号', value: 'keep' },
  { label: '中后换', value: 'after_hit' },
  { label: '挂后换', value: 'after_miss' },
] as const

interface HcwTier {
  hot: string[]
  warm: string[]
  cold: string[]
}
interface HcwCell {
  token: string
  count: number | null
  tier: 'hot' | 'cold' | 'none'
}

const playModeSummary = computed(() => playConfigSummary(props.playConfig) || '—')

const positionCount = computed(() => Math.max(1, props.playConfig.segmentLen || 1))
const positionLabels = computed(() =>
  Array.from(
    { length: positionCount.value },
    (_, i) => props.playConfig.segmentLabels?.[i] ?? POSITION_FALLBACK[i] ?? `第 ${i + 1} 位`,
  ),
)

const showTriggerPositionPicker = computed(() => {
  if (props.runTypeId !== 'adv_trigger_bet') return false
  if (isLonghuPlayConfigLike(props.playConfig)) return false
  if (textPickOptionsForConfig(props.playConfig).length > 0) return false
  const tid = String(props.playConfig.playTypeId ?? '')
  const bm = String(props.playConfig.betMode ?? '')
  const isDingwei =
    bm === 'dingwei' ||
    tid === 'dingwei' ||
    tid === 'g006' ||
    String(props.playConfig.guajiGroup ?? '') === '一星'
  return isDingwei && positionCount.value >= 2
})

const triggerPositionIdxs = computed(() => {
  const tb = props.triggerBet
  if (!tb) return [0]
  if (tb.positionIdxs?.length) return [...tb.positionIdxs].sort((a, b) => a - b)
  if (tb.positionIdx != null && Number.isFinite(tb.positionIdx)) return [Number(tb.positionIdx)]
  return [0]
})

const triggerRows = computed<SchemeTriggerRow[]>(() => props.triggerBet?.rows ?? [])
const triggerMode = computed(() => props.triggerBet?.mode ?? 'always_pos')

const jushuDisplayList = computed(() => {
  if (props.jushuList.length) return props.jushuList
  return props.schemeGroups
    .filter((g) => g.trim())
    .map((content, i) => ({
      ju: i + 1,
      content,
      afterHit: 1,
      afterMiss: 1,
    }))
})

const numberPoolTokens = computed<string[]>(() => {
  const min = props.playConfig.numberPoolMin
  const max = props.playConfig.numberPoolMax
  if (min != null && max != null && max >= min && (max > 9 || min > 0)) {
    const pad = max >= 11
    return Array.from({ length: max - min + 1 }, (_, i) => {
      const n = min + i
      return pad ? String(n).padStart(2, '0') : String(n)
    })
  }
  return [...ALL_DIGITS]
})

function normalizePoolToken(raw: string): string {
  const v = String(raw ?? '').trim()
  if (!/^\d{1,2}$/.test(v)) return ''
  const n = Number(v)
  for (const tk of numberPoolTokens.value) {
    if (Number(tk) === n) return tk
  }
  return ''
}

const hcwDigitOverall = computed(() => {
  const cfg = props.playConfig as { betMode?: string; subPlayId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  if (['zu3', 'zu6', 'zu24', 'zu12', 'zu60', 'zu30', 'zu120', 'budingwei', 'baodan'].includes(bm)) return true
  const sub = `${String(cfg.subPlayId ?? '')}`.toLowerCase()
  if (/zuxuan_fs|zu3|zu6|zu24|zu12|zu60|zu30|zu120|budingwei|baodan/.test(sub)) return true
  const label = String(cfg.playMethodLabel ?? '')
  if (label.includes('单式')) return false
  return /组三|组六|组选|不定位|包胆/.test(label)
})

const hcwAttribute = computed(() => {
  const bm = String(props.playConfig.betMode ?? '').toLowerCase()
  return ['daxiao', 'danshuang', 'dxds', 'zhuangxian', 'longhu', 'longhuhe', 'longhubao', 'teshu', 'hezhi', 'kuadu'].includes(bm)
})

const hcwSingleGroup = computed(() => hcwDigitOverall.value || hcwAttribute.value)

/** 按位展示档数：以玩法位数为准，并兼容历史配置中更长的选号池 */
const hcwPosCount = computed(() => {
  if (hcwSingleGroup.value) return 1
  const fromPool = (props.hotColdWarm?.pool ?? []).length
  return Math.max(1, positionCount.value, fromPool)
})

const hcwPools = computed(() => {
  const pool = props.hotColdWarm?.pool ?? []
  const mapped = pool.map((p) =>
    String(p ?? '')
      .split(/[,，\s]+/)
      .map((t) => t.trim())
      .filter(Boolean),
  )
  const n = hcwPosCount.value
  while (mapped.length < n) mapped.push([])
  return mapped.slice(0, n)
})

const hcwStrategy = computed(() => {
  const st = props.hotColdWarm?.strategy
  if (st === 'every' || st === 'keep' || st === 'after_hit' || st === 'after_miss') return st
  return props.hotColdWarm?.winRotate ? 'after_hit' : 'keep'
})

const hcwTotalPeriods = computed(() => {
  const tp = Math.trunc(Number(props.hotColdWarm?.totalPeriods))
  if (Number.isFinite(tp) && tp >= 20 && tp <= 100) return tp
  if (Number.isFinite(tp) && tp > 100) return 100
  return 20
})

const hcwFaultCount = computed(() => {
  const fc = Math.trunc(Number(props.hotColdWarm?.faultCount))
  if (Number.isFinite(fc) && fc >= 1 && fc <= 10) return fc
  return 1
})

const hcwAttrUniverse = ref<string[]>([])
const hcwLoading = ref(false)
const hcwStatsReady = ref(false)
const hcwTiers = ref<HcwTier[]>([])
const hcwFreq = ref<Array<Record<string, number>>>([])

const hcwFallbackOptions = computed(() =>
  hcwAttribute.value ? hcwAttrUniverse.value : numberPoolTokens.value,
)

const hcwGroupLabels = computed(() => {
  if (hcwAttribute.value) return ['选项池']
  if (hcwDigitOverall.value) return ['号码池']
  return Array.from(
    { length: hcwPosCount.value },
    (_, i) => positionLabels.value[i] ?? POSITION_FALLBACK[i] ?? `第 ${i + 1} 位`,
  )
})

function tokenEq(a: string, b: string): boolean {
  const na = Number(a)
  const nb = Number(b)
  if (a.trim() !== '' && b.trim() !== '' && Number.isFinite(na) && Number.isFinite(nb)) {
    return na === nb
  }
  return a === b
}

function poolHasToken(arr: string[] | undefined, token: string): boolean {
  if (!arr) return false
  return arr.some((t) => tokenEq(t, token))
}

function sortHcwTokens(tokens: string[]): string[] {
  return [...tokens].sort((a, b) => {
    const na = Number(a)
    const nb = Number(b)
    if (Number.isFinite(na) && Number.isFinite(nb)) return na - nb
    return 0
  })
}

function hcwPositionOffset(ballsLen: number): number {
  const segLen = positionCount.value
  if (ballsLen <= segLen) return 0
  if (segLen === 1) {
    const sub = (props.playConfig.catalogSubId ?? props.playConfig.subPlayId ?? '').toLowerCase()
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
  const typeId = props.playConfig.playTypeId
  if (typeId.startsWith('hou')) return ballsLen - segLen
  if (typeId.startsWith('zhong')) return Math.floor((ballsLen - segLen) / 2)
  return 0
}

async function loadHcwAttrStats(): Promise<void> {
  const cfg = props.playConfig
  const code = String(props.lotteryCode ?? '').trim()
  if (!code) {
    hcwStatsReady.value = false
    hcwFreq.value = []
    return
  }
  const res = await fetchHotColdWarmTiers({
    lotteryCode: code,
    playTypeId: cfg.playTypeId,
    subPlayId: cfg.subPlayId,
    playTemplate: cfg.playTemplate,
    betMode: cfg.betMode,
    catalogSubId: cfg.catalogSubId,
    playMethodLabel: cfg.playMethodLabel,
    numberPoolMin: cfg.numberPoolMin,
    numberPoolMax: cfg.numberPoolMax,
    segmentLen: cfg.segmentLen,
    periods: hcwTotalPeriods.value,
  })
  if (res.mode !== 'attribute' || !Array.isArray(res.universe) || res.universe.length === 0) {
    hcwStatsReady.value = false
    hcwFreq.value = []
    return
  }
  hcwAttrUniverse.value = res.universe
  hcwTiers.value = [{ hot: res.hot ?? [], warm: res.warm ?? [], cold: res.cold ?? [] }]
  hcwFreq.value = [res.counts && typeof res.counts === 'object' ? { ...res.counts } : {}]
  hcwStatsReady.value = true
}

async function loadHcwStats(): Promise<void> {
  if (hcwLoading.value) return
  if (props.runTypeId !== 'hot_cold_warm') return
  hcwLoading.value = true
  try {
    if (hcwAttribute.value) {
      await loadHcwAttrStats()
      return
    }
    const code = String(props.lotteryCode ?? '').trim()
    if (!code) {
      hcwStatsReady.value = false
      hcwFreq.value = []
      return
    }
    const res = await fetchGameDraws(code, undefined, hcwTotalPeriods.value)
    const items = Array.isArray(res?.items) ? res.items : []
    const segLen = positionCount.value
    const pool = numberPoolTokens.value
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
    hcwStatsReady.value = false
    hcwFreq.value = []
  } finally {
    hcwLoading.value = false
  }
}

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

const hcwCellsByPos = computed(() => hcwGroupLabels.value.map((_, pi) => hcwDisplayCells(pi)))

const hcwEstimatedUnits = computed(() => {
  if (hcwAttribute.value) {
    return (hcwPools.value[0] ?? []).filter((t) => t.trim() !== '').length
  }
  if (hcwDigitOverall.value) {
    const line = (hcwPools.value[0] ?? []).join(',')
    return line.trim() ? countBetUnits(props.playConfig, line) : 0
  }
  const n = hcwPosCount.value
  if (n <= 0) return 0
  const lines = Array.from({ length: n }, (_, i) => (hcwPools.value[i] ?? []).join(','))
  if (lines.every((x) => !x.trim())) return 0
  return countBetUnits(props.playConfig, lines.join('\n'))
})

watch(
  [
    () => props.runTypeId,
    () => props.lotteryCode,
    () => props.hotColdWarm?.totalPeriods,
    () => props.playConfig.betMode,
    () => props.playConfig.subPlayId,
    () => props.playConfig.catalogSubId,
    () => props.playConfig.playMethodLabel,
    () => props.playConfig.segmentLen,
  ],
  () => {
    if (props.runTypeId !== 'hot_cold_warm') return
    hcwStatsReady.value = false
    hcwAttrUniverse.value = []
    void loadHcwStats()
  },
  { immediate: true },
)

const rdCounts = computed(() => props.randomDraw?.counts ?? [])
const rdSingleCountMode = computed(() => rdCounts.value.length <= 1)

function groupBetUnits(content: string): number {
  return countBetUnits(props.playConfig, content)
}

function formatGroupContent(content: string): string {
  const raw = String(content ?? '').replace(/\r/g, '')
  if (!raw.trim()) return '—'
  if (schemeGroupUsesDigitInput(props.playConfig)) {
    const box = schemeGroupContentToInputBox(raw, props.playConfig)
    if (box) return box
  }
  if (raw.includes('\n')) {
    return raw
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean)
      .join(', ')
  }
  return raw.trim()
}

</script>

<template>
  <section class="scr" data-panel="scheme-content-readonly">
    <div class="scr-head">
      <div class="scr-head-left">
        <h2 class="scr-title">方案内容</h2>
        <p class="scr-hint">{{ runTypeLabel }} · {{ playModeSummary }}</p>
      </div>
    </div>

    <!-- 固定取码规则 -->
    <div v-if="runTypeId === 'fixed_number' && (fixedPick?.rules?.length ?? 0) > 0" class="scr-card scr-panel">
      <p class="scr-fp-title">取码规则</p>
      <div v-for="(rule, ri) in fixedPick?.rules ?? []" :key="ri" class="scr-fp-rule">
        <div class="scr-fp-line">
          <span class="scr-fp-lbl">位区间</span>
          <span class="scr-fp-val">{{ rule.posStart + 1 }} - {{ rule.posEnd + 1 }}</span>
          <span class="scr-fp-lbl">号码区间</span>
          <span class="scr-fp-val">{{ rule.codeMin }} - {{ rule.codeMax }}</span>
        </div>
        <div class="scr-fp-line">
          <span class="scr-fp-lbl">投注号码</span>
          <span class="scr-fp-nums">{{ formatGroupContent(rule.numbers) }}</span>
        </div>
      </div>
    </div>

    <!-- 定码轮换 / 固定取码号码 -->
    <div
      v-if="runTypeId === 'fixed_rotate' || runTypeId === 'fixed_number'"
      class="scr-groups"
    >
      <p v-if="runTypeId === 'fixed_number'" class="scr-tip scr-tip--banner">
        固定取码：未设置取码规则时，每期原样复投以下固定号码
      </p>
      <el-empty
        v-if="!schemeGroups.filter((g) => g.trim()).length"
        description="暂无号码内容"
        :image-size="56"
      />
      <div
        v-for="(g, idx) in schemeGroups"
        v-show="String(g).trim()"
        :key="idx"
        class="scr-card"
      >
        <div class="scr-group-bar">
          <h3 class="scr-group-title">
            {{ runTypeId === 'fixed_number' ? '固定取码' : `第 ${idx + 1} 组` }}
          </h3>
          <span class="scr-group-units">注数: {{ groupBetUnits(g) }}</span>
        </div>
        <div class="scr-group-body">
          <p class="scr-group-content">{{ formatGroupContent(g) }}</p>
        </div>
      </div>
    </div>

    <!-- 高级定码轮换 -->
    <div v-else-if="runTypeId === 'adv_fixed_rotate'" class="scr-card scr-panel">
      <p class="scr-tip">跳转到不存在的局数时，自动回到第 1 局</p>
      <el-empty v-if="!jushuDisplayList.length" description="暂无局数" :image-size="56" />
      <ul v-else class="scr-jushu-list">
        <li v-for="row in jushuDisplayList" :key="row.ju" class="scr-jushu-row">
          <div class="scr-jushu-main">
            <span class="scr-jushu-no">第 {{ row.ju }} 局</span>
            <span class="scr-jushu-content">{{ formatGroupContent(row.content) }}</span>
          </div>
          <div class="scr-jushu-side">
            <span class="scr-jushu-jump">中后 → 第 {{ row.afterHit }} 局</span>
            <span class="scr-jushu-jump">挂后 → 第 {{ row.afterMiss }} 局</span>
          </div>
        </li>
      </ul>
    </div>

    <!-- 高级开某投某 -->
    <div v-else-if="runTypeId === 'adv_trigger_bet'" class="scr-card scr-panel">
      <div v-if="showTriggerPositionPicker" class="scr-field">
        <span class="scr-lbl">投注位</span>
        <div
          class="scr-trig-pos-chips"
          role="group"
          aria-label="投注位"
          :style="{ '--scr-trig-pos-n': String(positionLabels.length || 5) }"
        >
          <span
            v-for="(label, idx) in positionLabels"
            :key="`pos-${idx}`"
            class="scr-trig-pos-chip"
            :class="{ 'is-on': triggerPositionIdxs.includes(idx) }"
          >{{ label }}</span>
        </div>
      </div>
      <div class="scr-trig-grid scr-trig-grid--head" aria-hidden="true">
        <span>启用</span>
        <span>开出</span>
        <span>正投</span>
        <span>反投</span>
      </div>
      <div
        v-for="row in triggerRows"
        :key="row.open"
        class="scr-trig-grid"
        :class="{ 'is-off': !row.enabled }"
      >
        <el-switch :model-value="row.enabled" size="small" disabled />
        <span class="scr-trig-open">{{ row.open }}</span>
        <el-input :model-value="row.pos" size="small" disabled />
        <el-input :model-value="row.neg" size="small" disabled />
      </div>
      <div class="scr-field">
        <span class="scr-lbl">投向模式</span>
        <el-radio-group :model-value="triggerMode" class="scr-radio-wrap" disabled>
          <el-radio v-for="o in TRIGGER_MODE_OPTIONS" :key="o.value" :value="o.value">
            {{ o.label }}
          </el-radio>
        </el-radio-group>
      </div>
      <p class="scr-tip">
        <template v-if="showTriggerPositionPicker">
          可多选投注位：每位按该位上期开奖各自查映射下注；某位无映射时用启用行第 1 行正投
        </template>
        <template v-else>
          上期开出号码无启用映射时，按启用行第 1 行的正投下注
        </template>
      </p>
    </div>

    <!-- 冷热出号（与新建页同布局，只读） -->
    <div v-else-if="runTypeId === 'hot_cold_warm'" class="scr-card scr-panel">
      <div class="scr-hcw-bar scr-hcw-bar--top">
        <div class="scr-hcw-ctrl">
          <span class="scr-hcw-lbl">总期数</span>
          <div class="scr-stepper" role="group" aria-label="总期数">
            <button type="button" class="scr-stepper-btn" disabled aria-label="减少总期数">
              <span class="material-sym scr-ms-sm" aria-hidden="true">remove</span>
            </button>
            <el-input
              :model-value="hcwTotalPeriods"
              type="number"
              class="scr-stepper-input"
              disabled
            />
            <button type="button" class="scr-stepper-btn" disabled aria-label="增加总期数">
              <span class="material-sym scr-ms-sm" aria-hidden="true">add</span>
            </button>
          </div>
          <button
            type="button"
            class="scr-hcw-refresh"
            :disabled="hcwLoading"
            aria-label="刷新统计"
            title="刷新统计"
            @click="loadHcwStats"
          >
            <span
              class="material-sym scr-ms-sm"
              :class="{ 'scr-hcw-refresh-spin': hcwLoading }"
              aria-hidden="true"
            >refresh</span>
          </button>
        </div>
        <div class="scr-hcw-ctrl">
          <span class="scr-hcw-lbl">容错</span>
          <div class="scr-stepper" role="group" aria-label="容错个数">
            <button type="button" class="scr-stepper-btn" disabled aria-label="减少容错">
              <span class="material-sym scr-ms-sm" aria-hidden="true">remove</span>
            </button>
            <el-input
              :model-value="hcwFaultCount"
              type="number"
              class="scr-stepper-input"
              disabled
            />
            <button type="button" class="scr-stepper-btn" disabled aria-label="增加容错">
              <span class="material-sym scr-ms-sm" aria-hidden="true">add</span>
            </button>
          </div>
        </div>
      </div>
      <div class="scr-hcw-bar scr-hcw-bar--strategy">
        <el-radio-group :model-value="hcwStrategy" class="scr-hcw-strategy" disabled>
          <el-radio v-for="o in HCW_STRATEGY_OPTIONS" :key="o.value" :value="o.value">
            {{ o.label }}
          </el-radio>
        </el-radio-group>
        <span class="scr-hcw-units">总计：{{ hcwEstimatedUnits }} 注</span>
      </div>
      <div v-for="(label, pi) in hcwGroupLabels" :key="pi" class="scr-hcw-pos">
        <div class="scr-hcw-pos-head">
          <p class="scr-hcw-pos-name">{{ label }}</p>
          <div class="scr-hcw-quick" role="group" :aria-label="`${label}快捷选号`">
            <span
              class="scr-hcw-qbtn"
              :class="{ 'is-on': hcwQuickActive(pi, 'cold') }"
            >冷</span>
            <span
              class="scr-hcw-qbtn"
              :class="{ 'is-on': hcwQuickActive(pi, 'hot') }"
            >热</span>
            <span
              class="scr-hcw-qbtn"
              :class="{ 'is-on': hcwQuickActive(pi, 'all') }"
            >全</span>
            <span class="scr-hcw-qbtn">清</span>
          </div>
        </div>
        <p v-if="!hcwStatsReady && !hcwLoading" class="scr-tip">
          {{ hcwAttribute ? '暂无选项频次，可点刷新重试' : '暂无开奖统计，已选号码见高亮' }}
        </p>
        <div
          v-if="(hcwCellsByPos[pi] ?? []).length"
          class="scr-hcw-grid"
          :style="{
            '--hcw-cols': String(Math.min(10, (hcwCellsByPos[pi] ?? []).length) || 10),
          }"
        >
          <div
            v-for="cell in hcwCellsByPos[pi]"
            :key="cell.token"
            class="scr-hcw-cell"
            :class="{
              'is-hot': cell.tier === 'hot',
              'is-cold': cell.tier === 'cold',
              'is-on': poolHasToken(hcwPools[pi], cell.token),
            }"
          >
            <span class="scr-hcw-cell-num">{{ cell.token }}</span>
            <span class="scr-hcw-cell-cnt">{{ cell.count == null ? '—' : cell.count }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 随机出号 -->
    <div v-else-if="runTypeId === 'random_draw'" class="scr-card scr-panel">
      <template v-if="rdSingleCountMode">
        <div class="scr-rd-row">
          <span class="scr-rd-pos">选号个数</span>
          <span class="scr-fp-val">{{ rdCounts[0] ?? '—' }}</span>
        </div>
      </template>
      <template v-else>
        <div v-for="(label, pi) in positionLabels" :key="pi" class="scr-rd-row">
          <span class="scr-rd-pos">{{ label }}</span>
          <span class="scr-fp-val">{{ rdCounts[pi] ?? '—' }} 个</span>
        </div>
      </template>
      <div class="scr-field">
        <span class="scr-lbl">换号策略</span>
        <el-radio-group :model-value="randomDraw?.strategy ?? 'every'" class="scr-radio-wrap" disabled>
          <el-radio v-for="o in RD_STRATEGY_OPTIONS" :key="o.value" :value="o.value">
            {{ o.label }}
          </el-radio>
        </el-radio-group>
      </div>
      <p class="scr-tip">云端运行时每期由引擎按数量自动随机，实际号码见投注明细</p>
    </div>

    <!-- 内置计画 -->
    <div v-else-if="runTypeId === 'builtin_plan'" class="scr-card scr-panel">
      <div class="scr-bp-summary">
        <p class="scr-bp-title">
          已跟随：{{ schemeName || '内置计画' }}
          <template v-if="builtinPlanSnapshotId"> · {{ builtinPlanSnapshotId }}</template>
        </p>
        <p class="scr-tip">内置计画配置只读，与收藏计划保持一致</p>
      </div>
      <div v-if="schemeGroups.some((g) => g.trim())" class="scr-groups">
        <div v-for="(g, idx) in schemeGroups" v-show="String(g).trim()" :key="idx" class="scr-card">
          <div class="scr-group-bar">
            <h3 class="scr-group-title">第 {{ idx + 1 }} 组</h3>
            <span class="scr-group-units">注数: {{ groupBetUnits(g) }}</span>
          </div>
          <div class="scr-group-body">
            <p class="scr-group-content">{{ formatGroupContent(g) }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="scr-card scr-panel">
      <el-empty description="暂无方案内容" :image-size="56" />
    </div>
  </section>
</template>

<style scoped>
.scr {
  --scr-primary: #0050cb;
  --scr-primary-strong: #0050cb;
  --scr-on-variant: #727687;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.scr-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0 0.1rem;
}

.scr-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.05rem;
  font-weight: 700;
  color: #191c1e;
}

.scr-hint {
  margin: 0.2rem 0 0;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--scr-on-variant);
  line-height: 1.45;
}

.scr-card {
  background: #fff;
  border-radius: 0.875rem;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scr-panel {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.scr-groups {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.scr-group-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem 0.75rem;
  flex-wrap: wrap;
  padding: 0.65rem 1rem;
  background: #fff;
}

.scr-group-title {
  margin: 0;
  flex-shrink: 0;
  font-size: 0.875rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  color: var(--scr-primary-strong);
}

.scr-group-units {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  color: #64748b;
}

.scr-group-body {
  padding: 0.85rem 1rem 1rem;
  background: rgba(242, 244, 246, 0.45);
}

.scr-group-content {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.65;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  color: #191c1e;
  word-break: break-all;
  white-space: pre-wrap;
}

.scr-tip {
  margin: 0;
  font-size: 11px;
  font-weight: 500;
  line-height: 1.6;
  color: #727687;
}

.scr-tip--banner {
  padding: 0.65rem 1rem;
  border-radius: 0.75rem;
  background: rgba(0, 80, 203, 0.06);
  color: var(--scr-primary);
}

.scr-field {
  --scr-lbl-col: 4.5rem;
  display: grid;
  grid-template-columns: var(--scr-lbl-col) minmax(0, 1fr);
  align-items: center;
  column-gap: 0.5rem;
  min-width: 0;
}

.scr-lbl {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--scr-on-variant);
  line-height: 1.3;
}

.scr-radio-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 0.15rem 1.1rem;
}

.scr-trig-pos-chips {
  --scr-trig-pos-n: 5;
  display: grid;
  grid-template-columns: repeat(var(--scr-trig-pos-n), minmax(0, 1fr));
  gap: 0.35rem;
  width: 100%;
  min-width: 0;
  padding: 0.25rem;
  border-radius: 0.65rem;
  background: rgba(242, 244, 246, 0.85);
}

.scr-trig-pos-chip {
  display: grid;
  place-items: center;
  height: 2rem;
  border-radius: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scr-on-variant);
  background: transparent;
}

.scr-trig-pos-chip.is-on {
  background: #fff;
  color: var(--el-color-primary, #0050cb);
  box-shadow: 0 2px 10px rgba(25, 28, 30, 0.08);
}

.scr-trig-grid {
  display: grid;
  grid-template-columns: 3rem 3rem 1fr 1fr;
  align-items: center;
  gap: 0.6rem;
}

.scr-trig-grid--head span {
  font-size: 11px;
  font-weight: 700;
  color: var(--scr-on-variant);
  letter-spacing: 0.02em;
}

.scr-trig-grid.is-off .scr-trig-open {
  opacity: 0.35;
}

.scr-trig-open {
  font-size: 0.9375rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--scr-primary-strong);
  text-align: center;
  padding: 0.25rem 0;
  border-radius: 0.45rem;
  background: rgba(0, 80, 203, 0.06);
}

.scr-jushu-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.scr-jushu-row {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  padding: 0.75rem 0.85rem;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
}

.scr-jushu-main {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
  min-width: 0;
}

.scr-jushu-no {
  flex-shrink: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scr-primary-strong);
}

.scr-jushu-content {
  min-width: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  color: #191c1e;
  word-break: break-all;
}

.scr-jushu-side {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.scr-jushu-jump {
  font-size: 11px;
  font-weight: 600;
  color: var(--scr-on-variant);
}

.scr-fp-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--scr-primary-strong);
}

.scr-fp-rule {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  padding: 0.7rem 0.8rem;
  border-radius: 0.7rem;
  background: rgba(242, 244, 246, 0.65);
}

.scr-fp-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.4rem 0.55rem;
}

.scr-fp-lbl {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--scr-on-variant);
}

.scr-fp-val {
  font-size: 0.875rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
}

.scr-fp-nums {
  font-size: 0.875rem;
  font-weight: 600;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  color: #191c1e;
  word-break: break-all;
}

.scr-ms-sm {
  font-size: 1.25rem;
  line-height: 1;
}

.scr-hcw-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
}

.scr-hcw-bar--top {
  justify-content: space-between;
  flex-wrap: nowrap;
}

.scr-hcw-bar--strategy {
  justify-content: space-between;
  gap: 0.35rem;
}

.scr-hcw-ctrl {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  flex: none;
}

.scr-hcw-lbl {
  flex: none;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--scr-on-variant);
  line-height: 1.3;
  white-space: nowrap;
}

.scr-stepper {
  display: inline-flex;
  align-items: stretch;
  height: 1.85rem;
  border-radius: 0.45rem;
  background: #f2f4f6;
  overflow: hidden;
}

.scr-stepper-btn {
  display: grid;
  place-items: center;
  width: 1.45rem;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  color: #727687;
  cursor: not-allowed;
  opacity: 0.55;
}

.scr-stepper-input {
  width: 2.6rem;
}

.scr-stepper-input :deep(.el-input__wrapper) {
  box-shadow: none !important;
  background: transparent;
  padding: 0 0.15rem;
  border-radius: 0;
}

.scr-stepper-input :deep(.el-input__inner) {
  text-align: center;
  font-size: 0.8125rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  height: 1.85rem;
  line-height: 1.85rem;
  padding: 0;
  color: #191c1e;
}

.scr-hcw-refresh {
  display: grid;
  place-items: center;
  width: 1.85rem;
  height: 1.85rem;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 0.45rem;
  background: rgba(0, 80, 203, 0.08);
  color: var(--scr-primary-strong);
  cursor: pointer;
  transition: background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scr-hcw-refresh:hover:not(:disabled) {
  background: rgba(0, 80, 203, 0.14);
}

.scr-hcw-refresh:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.scr-hcw-refresh-spin {
  display: inline-block;
  animation: scr-hcw-spin 0.8s linear infinite;
}

@keyframes scr-hcw-spin {
  to {
    transform: rotate(360deg);
  }
}

.scr-hcw-strategy {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.1rem 0.35rem;
  min-width: 0;
  flex: 1 1 auto;
}

.scr-hcw-strategy :deep(.el-radio) {
  margin-right: 0;
  height: auto;
  margin-left: 0;
}

.scr-hcw-strategy :deep(.el-radio__label) {
  font-size: 0.75rem;
  font-weight: 600;
  padding-left: 0.2rem;
}

.scr-hcw-strategy :deep(.el-radio__inner) {
  width: 0.875rem;
  height: 0.875rem;
}

.scr-hcw-units {
  flex: none;
  font-size: 0.75rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--scr-primary-strong);
  white-space: nowrap;
}

.scr-hcw-pos {
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
}

.scr-hcw-pos-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.scr-hcw-pos-name {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scr-primary-strong);
}

.scr-hcw-quick {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
}

.scr-hcw-qbtn {
  display: inline-grid;
  place-items: center;
  min-width: 1.7rem;
  height: 1.55rem;
  padding: 0 0.35rem;
  border-radius: 0.4rem;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--scr-on-variant);
  background: #f2f4f6;
  user-select: none;
}

.scr-hcw-qbtn.is-on {
  color: #fff;
  background: var(--scr-primary-strong);
}

.scr-hcw-grid {
  --hcw-cols: 10;
  display: grid;
  grid-template-columns: repeat(var(--hcw-cols), minmax(0, 1fr));
  gap: 0.35rem;
}

.scr-hcw-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.1rem;
  min-height: 2.65rem;
  padding: 0.3rem 0.15rem;
  border-radius: 0.5rem;
  border: 1px solid transparent;
  background: #f2f4f6;
  user-select: none;
}

.scr-hcw-cell-num {
  font-size: 0.875rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
  line-height: 1.2;
}

.scr-hcw-cell-cnt {
  font-size: 0.625rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: #727687;
  line-height: 1.2;
}

.scr-hcw-cell.is-hot .scr-hcw-cell-num,
.scr-hcw-cell.is-hot .scr-hcw-cell-cnt {
  color: #e53935;
}

.scr-hcw-cell.is-cold .scr-hcw-cell-num,
.scr-hcw-cell.is-cold .scr-hcw-cell-cnt {
  color: #b0b4be;
}

.scr-hcw-cell.is-on {
  border-color: rgba(0, 80, 203, 0.45);
  background: rgba(0, 80, 203, 0.08);
  box-shadow: 0 2px 10px rgba(0, 80, 203, 0.12);
}

.scr-hcw-cell.is-on.is-hot {
  border-color: rgba(229, 57, 53, 0.45);
  background: rgba(229, 57, 53, 0.1);
  box-shadow: 0 2px 10px rgba(229, 57, 53, 0.14);
}

.scr-hcw-cell.is-on.is-cold {
  border-color: rgba(176, 180, 190, 0.7);
  background: rgba(176, 180, 190, 0.16);
  box-shadow: none;
}

.scr-rd-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.scr-rd-pos {
  flex: none;
  min-width: 3rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scr-primary-strong);
}

.scr-bp-summary {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.scr-bp-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: #191c1e;
  line-height: 1.5;
}
</style>
