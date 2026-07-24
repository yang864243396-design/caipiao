<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { fetchGameDraws } from '@/api/games/detail'
import { fetchHotColdWarmTiers } from '@/api/schemes/definitions'
import SchemeGroupInputPanel from '@/components/schemes/SchemeGroupInputPanel.vue'
import SchemeGroupPickPanel from '@/components/schemes/SchemeGroupPickPanel.vue'
import SchemeRenxuanDanshiPanel from '@/components/schemes/SchemeRenxuanDanshiPanel.vue'
import type { PlayConfig } from '@/utils/betPayload'
import {
  countBetUnits,
  groupContentPlaceholder,
  isRenxuanPositionDanshiConfig,
  playConfigSummary,
} from '@/utils/betPayload'
import {
  schemeGroupContentToInputBox,
  schemeGroupUsesDigitInput,
  schemeGroupUsesPickPanel,
  textPickOptionsForConfig,
} from '@/utils/pickPanelOptions'
import {
  isLonghuPlayConfigLike,
  supportsAdvTriggerPerPosColumns,
  supportsAdvTriggerPositionPicker,
} from '@/utils/runTypeMatrix'
import type {
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
  { label: '每期换', value: 'every' },
  { label: '不换号', value: 'keep' },
  { label: '中后换', value: 'after_hit' },
  { label: '挂后换', value: 'after_miss' },
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
  return supportsAdvTriggerPositionPicker(props.playConfig)
})

const showTriggerPerPosColumns = computed(() => {
  if (props.runTypeId !== 'adv_trigger_bet') return false
  if (isLonghuPlayConfigLike(props.playConfig)) return false
  if (textPickOptionsForConfig(props.playConfig).length > 0) return false
  return supportsAdvTriggerPerPosColumns(props.playConfig)
})

function triggerPosName(posLabel: string): string {
  const base = String(posLabel ?? '')
    .trim()
    .replace(/位$/, '')
  return `${base || '位'}位`
}

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

const triggerPositionIdxs = computed(() => {
  const tb = props.triggerBet
  const n = Math.max(1, positionCount.value)
  const all = Array.from({ length: n }, (_, i) => i)
  if (!tb) return all
  if (tb.positionIdxs?.length) {
    return [...tb.positionIdxs]
      .map((x) => Number(x))
      .filter((i) => Number.isInteger(i) && i >= 0 && i < n)
      .sort((a, b) => a - b)
  }
  if (tb.positionIdx != null && Number.isFinite(tb.positionIdx)) {
    const i = Number(tb.positionIdx)
    if (i >= 0 && i < n) return [i]
  }
  return all
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
  if (Number.isFinite(fc)) return Math.min(9, Math.max(0, fc))
  return 0
})

const hcwPickTypes = computed<Array<'hot' | 'cold'>>(() => {
  const arr = props.hotColdWarm?.pickTypes ?? []
  return arr
    .map((t) => String(t ?? '').toLowerCase())
    .filter((t): t is 'hot' | 'cold' => t === 'hot' || t === 'cold')
})

const hcwPickTypesLabel = computed(() =>
  hcwPickTypes.value.map((t) => (t === 'hot' ? '热号' : '冷号')).join(' + '),
)

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
  // 不传 segmentLen：单档 UI 常为 1，覆盖后跨度/和值计频会错位
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
  // 与编辑页一致：和值/跨度等走 countBetUnits（组合数×段倍乘），勿按选项个数计注
  if (hcwAttribute.value) {
    const line = (hcwPools.value[0] ?? []).filter((t) => t.trim() !== '').join(',')
    return line ? countBetUnits(props.playConfig, line) : 0
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

const schemeUsesPickPanel = computed(() => schemeGroupUsesPickPanel(props.playConfig))
const schemeUsesDigitInput = computed(() => schemeGroupUsesDigitInput(props.playConfig))
const schemeUsesRenxuanDanshi = computed(() => isRenxuanPositionDanshiConfig(props.playConfig))
const groupInputPlaceholder = computed(() => groupContentPlaceholder(props.playConfig))

const displayedGroupIndexes = computed(() => {
  if (props.runTypeId === 'fixed_number') return [0]
  return props.schemeGroups.map((_, i) => i)
})

const rdCounts = computed(() => {
  const raw = props.randomDraw?.counts ?? []
  if (raw.length) return raw.map((n) => Math.max(1, Math.trunc(Number(n) || 1)))
  return [1]
})

const rdStrategy = computed(() => {
  const st = props.randomDraw?.strategy
  if (st === 'every' || st === 'keep' || st === 'after_hit' || st === 'after_miss') return st
  return 'every'
})

/** 单式/组选单式：整注随机 */
const rdWholeTicket = computed(() => {
  const cfg = props.playConfig as { betMode?: string; subPlayId?: string; playMethodLabel?: string }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  const sub = String(cfg.subPlayId ?? '').toLowerCase()
  if (['danshi', 'zhixuan_ds', 'zuxuan_ds', 'hunhe'].includes(bm)) return true
  if (['zhixuan_ds', 'zuxuan_ds'].includes(sub)) return true
  const label = String(cfg.playMethodLabel ?? '')
  return label.includes('单式') || label.includes('混合')
})

/** 组选号池随机；包胆属属性单选（仅 1 码），勿因文案含「组选」误入。 */
const rdZuxuanPool = computed(() => {
  if (rdWholeTicket.value) return false
  const cfg = props.playConfig as {
    betMode?: string
    subPlayId?: string
    catalogSubId?: string
    playMethodLabel?: string
  }
  const bm = String(cfg.betMode ?? '').toLowerCase()
  const label = String(cfg.playMethodLabel ?? '')
  if (bm === 'baodan' || /包胆/.test(label)) return false
  if (['zu3', 'zu6', 'zu24', 'zu12', 'zu60', 'zu30', 'zu120'].includes(bm)) return true
  const cat = `${String(cfg.subPlayId ?? '')} ${String(cfg.catalogSubId ?? '')}`.toLowerCase()
  if (/baodan|_bd\b|包胆/.test(`${cat} ${label}`)) return false
  if (/zuxuan_fs|zuxuan|zu3|zu6|zu24|zu12|zu60|zu30|zu120/.test(cat)) return true
  return /组三|组六|组选/.test(label)
})

const rdAttribute = computed(() => {
  if (rdWholeTicket.value || rdZuxuanPool.value) return false
  const bm = String(props.playConfig.betMode ?? '').toLowerCase()
  return [
    'daxiao',
    'danshuang',
    'dxds',
    'zhuangxian',
    'longhu',
    'longhuhe',
    'longhubao',
    'teshu',
    'hezhi',
    'kuadu',
    'budingwei',
    'baodan',
  ].includes(bm)
})

const rdSingleCountMode = computed(
  () => rdWholeTicket.value || rdZuxuanPool.value || rdAttribute.value,
)
const rdSingleCountLabel = computed(() => {
  if (rdWholeTicket.value) return '注数'
  if (rdZuxuanPool.value) return '选码个数'
  return '选项个数'
})
const rdSingleCountMax = computed(() => {
  if (rdWholeTicket.value) return 200
  if (rdZuxuanPool.value) return Math.max(3, numberPoolTokens.value.length)
  if (rdAttribute.value) {
    const bm = String(props.playConfig.betMode ?? '').toLowerCase()
    if (bm === 'baodan') return 1
    return Math.max(1, numberPoolTokens.value.length || 1)
  }
  return 28
})
const rdSingleCountMin = computed(() => {
  if (rdWholeTicket.value) return 1
  if (rdZuxuanPool.value) return Math.max(2, positionCount.value)
  return 1
})

const rdEstimatedUnits = computed(() => {
  if (rdWholeTicket.value) return Math.min(200, Math.max(1, rdCounts.value[0] ?? 1))
  if (rdZuxuanPool.value) {
    const pool = [...numberPoolTokens.value]
    const k = Math.min(
      pool.length,
      Math.max(rdSingleCountMin.value, rdCounts.value[0] ?? rdSingleCountMin.value),
    )
    return countBetUnits(props.playConfig, pool.slice(0, k).join(','))
  }
  if (rdAttribute.value) return Math.max(1, rdCounts.value[0] ?? 1)
  const n = positionCount.value
  if (n <= 0) return 0
  const lines = Array.from({ length: n }, (_, i) => {
    const count = Math.min(10, Math.max(1, rdCounts.value[i] ?? 1))
    return Array.from({ length: count }, (_, j) => String(j % 10)).join(',')
  })
  return countBetUnits(props.playConfig, lines.join('\n'))
})

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

    <!-- 定码轮换 / 固定取码（与新建页同选号面板，只读） -->
    <div
      v-if="runTypeId === 'fixed_rotate' || runTypeId === 'fixed_number'"
      class="scr-groups-stack"
    >
      <el-empty
        v-if="!displayedGroupIndexes.length"
        description="暂无号码内容"
        :image-size="56"
      />
      <div
        v-for="idx in displayedGroupIndexes"
        :key="idx"
        class="scr-content-card"
      >
        <div class="scr-group-bar">
          <h3 class="scr-group-title">
            {{ runTypeId === 'fixed_number' ? '固定号码' : `第 ${idx + 1} 组` }}
          </h3>
          <span class="scr-group-units">注数: {{ groupBetUnits(schemeGroups[idx] ?? '') }}</span>
        </div>
        <div class="scr-textarea-wrap">
          <SchemeRenxuanDanshiPanel
            v-if="schemeUsesRenxuanDanshi"
            :model-value="schemeGroups[idx] ?? ''"
            :config="playConfig"
            disabled
          />
          <SchemeGroupInputPanel
            v-else-if="schemeUsesDigitInput"
            :model-value="schemeGroups[idx] ?? ''"
            :config="playConfig"
            disabled
          />
          <SchemeGroupPickPanel
            v-else-if="schemeUsesPickPanel"
            :model-value="schemeGroups[idx] ?? ''"
            :config="playConfig"
            disabled
          />
          <el-input
            v-else
            :model-value="schemeGroups[idx] ?? ''"
            type="textarea"
            :rows="8"
            resize="none"
            class="scr-area"
            :placeholder="groupInputPlaceholder"
            disabled
          />
        </div>
      </div>
    </div>

    <!-- 高级定码轮换 -->
    <div v-else-if="runTypeId === 'adv_fixed_rotate'" class="scr-content-card scr-panel">
      <p class="scr-run-tip">跳转到不存在的局数时，自动回到第 1 局</p>
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
    <div v-else-if="runTypeId === 'adv_trigger_bet'" class="scr-content-card scr-panel">
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
      <div
        class="scr-trig-grid scr-trig-grid--head"
        :class="{ 'scr-trig-grid--posrow': showTriggerPerPosColumns }"
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
          class="scr-trig-block"
          :class="{ 'is-off': !row.enabled }"
        >
          <div
            v-for="(label, pIdx) in positionLabels"
            :key="`scr-trig-c-${row.open}-${pIdx}`"
            class="scr-trig-grid scr-trig-grid--posrow"
          >
            <el-switch v-if="pIdx === 0" :model-value="row.enabled" size="small" disabled />
            <span v-else class="scr-trig-cell-placeholder" aria-hidden="true" />
            <span v-if="pIdx === 0" class="scr-trig-open">{{ row.open }}</span>
            <span v-else class="scr-trig-cell-placeholder" aria-hidden="true" />
            <span class="scr-trig-pos-name">{{ triggerPosName(label) }}</span>
            <el-input
              :model-value="triggerFieldParts(row.pos, positionCount)[pIdx] ?? ''"
              size="small"
              disabled
            />
            <el-input
              :model-value="triggerFieldParts(row.neg, positionCount)[pIdx] ?? ''"
              size="small"
              disabled
            />
          </div>
        </div>
      </template>
      <template v-else>
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
      </template>
      <div class="scr-field">
        <span class="scr-lbl">投向模式</span>
        <el-radio-group
          :model-value="triggerMode"
          class="scr-radio-wrap scr-radio-wrap--trigger-mode"
          disabled
        >
          <el-radio v-for="o in TRIGGER_MODE_OPTIONS" :key="o.value" :value="o.value">
            {{ o.label }}
          </el-radio>
        </el-radio-group>
      </div>
    </div>

    <!-- 冷热出号（与新建页同布局，只读） -->
    <div v-else-if="runTypeId === 'hot_cold_warm'" class="scr-content-card scr-panel">
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
          <div class="scr-stepper" role="group" aria-label="容错">
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
      <div v-if="hcwPickTypes.length" class="scr-hcw-bar scr-hcw-bar--types">
        <span class="scr-hcw-lbl">出号类型</span>
        <span class="scr-hcw-types-val">{{ hcwPickTypesLabel }}</span>
        <span class="scr-hcw-types-hint">按名次自动取号，某位手选可覆盖该位</span>
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
            <button
              type="button"
              class="scr-hcw-qbtn"
              :class="{ 'is-on': hcwQuickActive(pi, 'cold') }"
              disabled
            >冷</button>
            <button
              type="button"
              class="scr-hcw-qbtn"
              :class="{ 'is-on': hcwQuickActive(pi, 'hot') }"
              disabled
            >热</button>
            <button
              type="button"
              class="scr-hcw-qbtn"
              :class="{ 'is-on': hcwQuickActive(pi, 'all') }"
              disabled
            >全</button>
            <button type="button" class="scr-hcw-qbtn" disabled>清</button>
          </div>
        </div>
        <p v-if="!hcwStatsReady && !hcwLoading" class="scr-run-tip">
          {{ hcwAttribute ? '暂无选项频次，可点刷新重试' : '暂无开奖统计，已选号码见高亮' }}
        </p>
        <div
          v-if="(hcwCellsByPos[pi] ?? []).length"
          class="scr-hcw-grid"
          :style="{
            '--hcw-cols': String(Math.min(10, (hcwCellsByPos[pi] ?? []).length) || 10),
          }"
        >
          <button
            v-for="cell in hcwCellsByPos[pi]"
            :key="cell.token"
            type="button"
            class="scr-hcw-cell"
            :class="{
              'is-hot': cell.tier === 'hot',
              'is-cold': cell.tier === 'cold',
              'is-on': poolHasToken(hcwPools[pi], cell.token),
            }"
            disabled
          >
            <span class="scr-hcw-cell-num">{{ cell.token }}</span>
            <span class="scr-hcw-cell-cnt">{{ cell.count == null ? '—' : cell.count }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 随机出号（与新建页同布局，只读） -->
    <div v-else-if="runTypeId === 'random_draw'" class="scr-content-card scr-panel">
      <template v-if="rdSingleCountMode">
        <div class="scr-rd-row">
          <span class="scr-rd-pos">{{ rdSingleCountLabel }}</span>
          <el-input-number
            :model-value="rdCounts[0] ?? 1"
            :min="rdSingleCountMin"
            :max="rdSingleCountMax"
            size="small"
            disabled
          />
        </div>
      </template>
      <div v-else class="scr-rd-pos-grid">
        <div v-for="(label, pi) in positionLabels" :key="pi" class="scr-rd-row">
          <span class="scr-rd-pos">{{ label }}</span>
          <el-input-number
            :model-value="rdCounts[pi] ?? 1"
            :min="1"
            :max="10"
            size="small"
            disabled
          />
        </div>
      </div>
      <div class="scr-rd-toolbar">
        <el-button type="primary" plain size="small" disabled>生成预览</el-button>
        <span class="scr-rd-units">预估 {{ rdEstimatedUnits }} 注</span>
        <el-radio-group :model-value="rdStrategy" class="scr-rd-strategy" disabled aria-label="换号策略">
          <el-radio v-for="o in RD_STRATEGY_OPTIONS" :key="o.value" :value="o.value">
            {{ o.label }}
          </el-radio>
        </el-radio-group>
      </div>
      <div class="scr-rd-preview-box" role="group" aria-label="预览号码">
        <span class="scr-rd-preview-empty">实际号码由引擎按期生成，详见投注明细</span>
      </div>
    </div>

    <!-- 内置计画 -->
    <div v-else-if="runTypeId === 'builtin_plan'" class="scr-content-card scr-panel">
      <div class="scr-bp-summary">
        <div class="scr-bp-summary-main">
          <p class="scr-bp-summary-title">
            已跟随：{{ schemeName || '内置计划' }} · {{ playModeSummary }}
          </p>
          <p class="scr-run-tip">内置计划配置只读，与收藏计划保持一致</p>
        </div>
      </div>
    </div>

    <div v-else class="scr-content-card scr-panel">
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

.scr-content-card {
  background: #fff;
  border-radius: 0.875rem;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scr-panel {
  padding: var(--card-pad);
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.scr-groups-stack {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.scr-group-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem 0.75rem;
  flex-wrap: wrap;
  padding: 0.65rem 1rem;
  border-bottom: 1px solid rgba(194, 198, 216, 0.2);
  background: #fff;
  min-width: 0;
}

.scr-group-title {
  margin: 0;
  flex-shrink: 0;
  font-size: 0.875rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.01em;
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

.scr-textarea-wrap {
  padding: var(--card-pad);
}

.scr-area :deep(.el-textarea__inner) {
  border: none;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  padding: var(--card-pad);
  min-height: 9.5rem;
  font-size: 0.9375rem;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  line-height: 1.65;
  box-shadow: none;
  white-space: pre-wrap;
}

.scr-run-tip {
  margin: 0;
  font-size: 11px;
  font-weight: 500;
  line-height: 1.6;
  color: #727687;
}

.scr-run-tip--banner {
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

.scr-radio-wrap--trigger-mode {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 0.75rem;
  row-gap: 0.35rem;
  width: 100%;
}

.scr-radio-wrap--trigger-mode :deep(.el-radio) {
  margin-right: 0;
  height: auto;
  min-height: 2rem;
  align-items: center;
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

.scr-trig-grid--posrow {
  grid-template-columns: 2.1rem 1.75rem 2.4rem 1fr 1fr;
  gap: 0.35rem 0.28rem;
}

.scr-trig-block {
  display: flex;
  flex-direction: column;
  gap: 0.28rem;
  padding: 0.35rem 0;
}

.scr-trig-block + .scr-trig-block {
  border-top: 1px solid rgba(25, 28, 30, 0.06);
}

.scr-trig-block.is-off {
  opacity: 0.55;
}

.scr-trig-cell-placeholder {
  display: block;
  min-height: 1px;
}

.scr-trig-pos-name {
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--scr-on-variant);
  text-align: center;
  white-space: nowrap;
}

.scr-trig-grid--head span {
  font-size: 11px;
  font-weight: 700;
  color: var(--scr-on-variant);
  letter-spacing: 0.02em;
}

.scr-trig-grid--posrow .scr-trig-open {
  font-size: 0.8125rem;
  padding: 0.2rem 0;
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

.scr-hcw-bar--types {
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.scr-hcw-types-val {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--scr-primary-strong);
}

.scr-hcw-types-hint {
  font-size: 0.72rem;
  color: var(--scr-text-muted, #8a94a6);
  margin-left: auto;
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
  border: none;
  border-radius: 0.4rem;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--scr-on-variant);
  background: #f2f4f6;
  user-select: none;
  cursor: default;
}

.scr-hcw-qbtn.is-on {
  color: #fff;
  background: var(--scr-primary-strong);
}

.scr-hcw-qbtn:disabled {
  opacity: 1;
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
  cursor: default;
}

.scr-hcw-cell:disabled {
  opacity: 1;
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

.scr-rd-pos-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.65rem 0.55rem;
  width: 100%;
}

.scr-rd-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  min-width: 0;
}

.scr-rd-pos {
  flex-shrink: 0;
  min-width: 2rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scr-on-variant);
}

.scr-rd-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem 0.85rem;
  flex-wrap: wrap;
  margin: 0.5rem 0 0.75rem;
  width: 100%;
}

.scr-rd-units {
  flex-shrink: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary, #64748b);
}

.scr-rd-strategy {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.1rem 0.35rem;
  flex: 1 1 12rem;
  min-width: 0;
}

.scr-rd-strategy :deep(.el-radio) {
  margin-right: 0;
  height: auto;
  margin-left: 0;
  flex: 1 1 0;
  justify-content: center;
}

.scr-rd-strategy :deep(.el-radio__label) {
  font-size: 0.75rem;
  font-weight: 600;
  padding-left: 0.3rem;
  white-space: nowrap;
}

.scr-rd-strategy :deep(.el-radio__inner) {
  width: 0.875rem;
  height: 0.875rem;
}

.scr-rd-preview-box {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
  min-height: 2.75rem;
  padding: 0.55rem 0.65rem;
  border-radius: 0.55rem;
  background: rgba(242, 244, 246, 0.55);
}

.scr-rd-preview-empty {
  font-size: 0.8125rem;
  font-weight: 500;
  color: #94a3b8;
}

.scr-bp-summary {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.85rem 1rem;
  border-radius: 0.75rem;
  background: rgba(0, 80, 203, 0.06);
}

.scr-bp-summary-main {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}

.scr-bp-summary-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  line-height: 1.6;
  color: var(--scr-primary);
  word-break: break-all;
}
</style>
