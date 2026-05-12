<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

/** 详情页图标占位 PNG，可按位置改为不同资源 */
const ICON_PLACEHOLDER = '/images/lobby/icon-placeholder.png'

const pageTitle = computed(() => {
  const raw = route.query.scheme as string | undefined
  if (raw) return decodeURIComponent(raw)
  return '禄螭万位 - 定位胆万位'
})

watch(
  pageTitle,
  (t) => {
    document.title = `游戏详情 · ${t} · 精密终端`
  },
  { immediate: true }
)

const tab = ref(0)
const tabLabels = ['投注', '计划反集', '计划走势', '历史开奖', '投注记录'] as const

/** 计划反集：号码串与注数（接口对接时替换） */
const planInverseDigits = ref('xxx,xxx,xxx,xxx,xxx')
const planInverseBetCount = ref(3)

/** 计划走势：走势注数、更新时间、折线点、期号横轴、近期中挂 */
const planTrendGroupBets = ref(7)
const planTrendChartLineD =
  'M 0 60 L 5.26 40 L 10.53 80 L 15.79 20 L 21.05 60 L 26.32 40 L 31.58 0 L 36.84 80 L 42.11 100 L 47.37 60 L 52.63 20 L 57.89 40 L 63.16 80 L 68.42 60 L 73.68 20 L 78.95 0 L 84.21 40 L 89.47 80 L 94.74 100 L 100 60'
const planTrendChartAreaD =
  'M 0 60 L 5.26 40 L 10.53 80 L 15.79 20 L 21.05 60 L 26.32 40 L 31.58 0 L 36.84 80 L 42.11 100 L 47.37 60 L 52.63 20 L 57.89 40 L 63.16 80 L 68.42 60 L 73.68 20 L 78.95 0 L 84.21 40 L 89.47 80 L 94.74 100 L 100 60 V 100 H 0 Z'

const planTrendDots = [
  { left: 0, top: 60, hit: true },
  { left: 5.26, top: 40, hit: false },
  { left: 10.53, top: 80, hit: true },
  { left: 15.79, top: 20, hit: false },
  { left: 21.05, top: 60, hit: true },
  { left: 26.32, top: 40, hit: false },
  { left: 31.58, top: 0, hit: false },
  { left: 36.84, top: 80, hit: true },
  { left: 42.11, top: 100, hit: true },
  { left: 47.37, top: 60, hit: false },
  { left: 52.63, top: 20, hit: false },
  { left: 57.89, top: 40, hit: true },
  { left: 63.16, top: 80, hit: true },
  { left: 68.42, top: 60, hit: false },
  { left: 73.68, top: 20, hit: false },
  { left: 78.95, top: 0, hit: false },
  { left: 84.21, top: 40, hit: true },
  { left: 89.47, top: 80, hit: true },
  { left: 94.74, top: 100, hit: true },
  { left: 100, top: 60, hit: false },
] as const

const planTrendXLabels = [
  { text: '001', show: true },
  { text: '002', show: false },
  { text: '003', show: true },
  { text: '004', show: false },
  { text: '005', show: true },
  { text: '006', show: false },
  { text: '007', show: true },
  { text: '008', show: false },
  { text: '009', show: true },
  { text: '010', show: false },
  { text: '011', show: true },
  { text: '012', show: false },
  { text: '013', show: true },
  { text: '014', show: false },
  { text: '015', show: true },
  { text: '016', show: false },
  { text: '017', show: true },
  { text: '018', show: false },
  { text: '019', show: true },
  { text: '020', show: false },
] as const

const planTrendHistoryRows = [
  { period: '032', win: false },
  { period: '029 - 031', win: true },
  { period: '028', win: false },
  { period: '025 - 027', win: true },
  { period: '024', win: false },
  { period: '021 - 023', win: true },
] as const

/** 历史开奖：子 Tab 与列表（接口对接时替换） */
const historySubTabLabels = ['号码', '大小', '单双', '龙虎', '总和'] as const
const historySubTab = ref(0)

const historyGameTag = '腾讯分分彩'

interface HistoryDrawRecord {
  periodShort: string
  time: string
  balls: readonly string[]
  sum: number
}

const historyDrawRecords: readonly HistoryDrawRecord[] = [
  {
    periodShort: '031',
    time: '2023-10-27 12:40:00',
    balls: ['3', '9', '2', '7', '5'],
    sum: 26,
  },
  {
    periodShort: '030',
    time: '2023-10-27 12:35:00',
    balls: ['8', '1', '0', '6', '4'],
    sum: 19,
  },
  {
    periodShort: '029',
    time: '2023-10-27 12:30:00',
    balls: ['4', '5', '5', '1', '8'],
    sum: 23,
  },
  {
    periodShort: '028',
    time: '2023-10-27 12:25:00',
    balls: ['2', '2', '9', '0', '3'],
    sum: 16,
  },
]

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
const betMode = ref('2元')

/** 投注区面板展开（把手点击切换） */
const betDockOpen = ref(true)

/** 投注区入口：手动下注（手机） / 云端挂机 */
const betDockEntryMode = ref<'manual' | 'cloud'>('manual')

function toggleBetDockEntryMode() {
  betDockEntryMode.value = betDockEntryMode.value === 'manual' ? 'cloud' : 'manual'
}

function goBetMultiplierSettings() {
  router.push({ name: 'bet-multiplier-settings' })
}

/** 开奖展示：drawing=开奖中；drawn=已开出，展示 {@link drawnNumbers} */
const drawPhase = ref<'drawing' | 'drawn'>('drawing')

/** 玩法收藏（后续可对接接口同步） */
const isFavorite = ref(false)
function toggleFavorite() {
  isFavorite.value = !isFavorite.value
}

/** 已开奖时的 5 个号码（两位字符串，由接口赋值） */
const drawnNumbers = ref<readonly string[]>(['0', '1', '9', '2', '3'])

const tableRows = [
  { time: '031-032', scheme: '禄螭万位', numbers: '1 3 7', period: '031', draw: '1 6 5 8 3', win: true },
  { time: '030-031', scheme: '禄螭万位', numbers: '4 5 9', period: '030', draw: '2 4 9 1 5', win: false },
  { time: '029-030', scheme: '禄螭万位', numbers: '2 8 9', period: '029', draw: '3 7 8 4 9', win: true },
  { time: '028-029', scheme: '禄螭万位', numbers: '1 2 5', period: '028', draw: '0 1 7 2 1', win: true },
  { time: '027-028', scheme: '禄螭万位', numbers: '3 6 0', period: '027', draw: '2 8 3 5 5', win: false },
  { time: '026-027', scheme: '禄螭万位', numbers: '1 9 1', period: '026', draw: '6 4 0 3 7', win: true },
  { time: '025-026', scheme: '禄螭万位', numbers: '4 7 2', period: '025', draw: '5 9 8 1 2', win: false },
] as const

const bettingTableList = computed(() => [...tableRows])
const planTrendHistoryList = computed(() => [...planTrendHistoryRows])

/** 投注记录：期数、玩法、倍数、轮次、金额、盈亏、状态（接口对齐字段） */
interface BetRecordRow {
  /** 期数 */
  period: string
  /** 玩法 */
  playMethod: string
  /** 倍数 */
  multiplier: string
  /** 轮次 */
  round: string
  /** 金额（元） */
  amount: string
  /** 盈亏：正数为盈，负数为亏，0 为走水/未结 */
  profitLoss: number
  /** 状态 */
  status: string
}

const betRecordRows = ref<BetRecordRow[]>([
  {
    period: '20231103032',
    playMethod: '禄螭万位',
    multiplier: '2',
    round: '1',
    amount: '12.00',
    profitLoss: 88.5,
    status: '已结算',
  },
  {
    period: '20231103031',
    playMethod: '禄螭万位',
    multiplier: '1',
    round: '2',
    amount: '6.00',
    profitLoss: -6.0,
    status: '已结算',
  },
  {
    period: '20231103033',
    playMethod: '禄螭万位',
    multiplier: '5',
    round: '1',
    amount: '30.00',
    profitLoss: 0,
    status: '待开奖',
  },
])

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

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/copy-hall')
}

/** 临时：点击开奖区域切换 drawing / drawn，联调完成后删除 */
function toggleDrawPhaseDemo() {
  drawPhase.value = drawPhase.value === 'drawing' ? 'drawn' : 'drawing'
}
</script>

<template>
  <div class="detail" :class="{
    'dock-collapsed': !betDockOpen && tab !== 2 && tab !== 3 && tab !== 4,
    'detail--plan-trend': tab === 2 || tab === 3 || tab === 4,
  }">
    <header class="header-wrap">
      <div class="head-row">
        <div class="head-left">
          <button type="button" class="icon-link" aria-label="返回" @click="goBack">
            <img :src="ICON_PLACEHOLDER" alt="" width="24" height="24" class="primary-ico-img" decoding="async" />
          </button>
          <h1 class="head-title">{{ pageTitle }}</h1>
        </div>
        <button type="button" class="icon-link fav-btn" :class="{ 'fav-btn--on': isFavorite }"
          :aria-label="isFavorite ? '取消收藏' : '收藏'" :aria-pressed="isFavorite" @click="toggleFavorite">
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
          <h2 class="period-main">第 20231103032 期</h2>
          <button type="button" class="badge-wrap badge-wrap-demo-toggle" :aria-label="drawPhase === 'drawing'
            ? '临时演示：切换到已开奖视图'
            : '临时演示：切换到开奖中视图'
            " title="临时：点击切换开奖展示状态" @click="toggleDrawPhaseDemo">
            <span v-if="drawPhase === 'drawing'" class="draw-badge" aria-live="polite">开奖中</span>
            <span v-else class="draw-result" role="group" :aria-label="`本期开奖号码 ${drawnNumbers.join(' ')}`">
              <span v-for="(num, idx) in drawnNumbers" :key="idx" class="draw-ball">{{ num }}</span>
            </span>
          </button>
        </div>
        <div class="draw-row draw-row-2">
          <h2 class="period-sub">距离 20231103033 期</h2>
          <div class="timer-pill">
            <img :src="ICON_PLACEHOLDER" alt="" width="18" height="18" class="timer-ico-img" decoding="async" />
            <span class="timer-txt">00:40</span>
          </div>
        </div>
      </div>

      <el-radio-group v-model="tab" size="small" class="detail-tab-rg">
        <el-radio-button v-for="(label, i) in tabLabels" :key="label" :value="i">{{ label }}</el-radio-button>
      </el-radio-group>
    </header>

    <main class="main">
      <template v-if="tab === 0">
        <div class="table-card">
          <el-table :data="bettingTableList" class="detail-bet-table" size="small" stripe empty-text="暂无数据"
            :style="{ width: '100%' }">
            <el-table-column prop="time" label="下注时间" :min-width="40" />
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
      <template v-else-if="tab === 1">
        <div class="plan-inverse-page">
          <div class="plan-inverse-inner">
            <el-card class="plan-inverse-card" shadow="never">
              <p class="plan-inverse-digits">{{ planInverseDigits }}</p>
              <p class="plan-inverse-meta">共 {{ planInverseBetCount }} 注</p>
            </el-card>
          </div>
        </div>
      </template>
      <template v-else-if="tab === 2">
        <div class="plan-trend-page">
          <section class="plan-trend-chart-card" aria-label="号码走势图表">
            <div class="plan-trend-chart-head">
              <p class="plan-trend-chart-title">本组号码走势分析（{{ planTrendGroupBets }}注）</p>
            </div>
            <div class="plan-trend-chart-body">
              <div class="plan-trend-y-axis">
                <span v-for="y in ['5', '4', '3', '2', '1', '0']" :key="y">{{ y }}</span>
              </div>
              <div class="plan-trend-chart-plot">
                <div class="plan-trend-grid">
                  <div v-for="i in 6" :key="i" class="plan-trend-grid-line" />
                </div>
                <svg class="plan-trend-svg" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                  <defs>
                    <linearGradient id="planTrendChartGrad" x1="0" x2="0" y1="0" y2="1">
                      <stop offset="0%" stop-color="#0066ff" />
                      <stop offset="100%" stop-color="#ffffff" />
                    </linearGradient>
                  </defs>
                  <path :d="planTrendChartAreaD" fill="url(#planTrendChartGrad)" opacity="0.1" />
                  <path :d="planTrendChartLineD" fill="none" stroke="#0066ff" stroke-width="1.5"
                    stroke-linejoin="round" />
                </svg>
                <div class="plan-trend-dots-layer">
                  <div v-for="(d, idx) in planTrendDots" :key="idx" class="plan-trend-dot-anchor"
                    :style="{ left: `${d.left}%`, top: `${d.top}%` }">
                    <span class="plan-trend-dot" :class="d.hit ? 'plan-trend-dot--hit' : 'plan-trend-dot--miss'" />
                  </div>
                </div>
                <div class="plan-trend-x-axis">
                  <span v-for="(x, xi) in planTrendXLabels" :key="xi" class="plan-trend-x-tick"
                    :class="{ 'plan-trend-x-tick--hide': !x.show }">{{ x.text }}</span>
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
            <div class="plan-trend-history-foot">
              <el-button type="primary" link @click.prevent>查看更多历史计划</el-button>
            </div>
          </section>
        </div>
      </template>
      <template v-else-if="tab === 3">
        <div class="history-page">
          <div class="history-subtabs-wrap">
            <el-radio-group v-model="historySubTab" size="small" class="history-subtabs-ep">
              <el-radio-button v-for="(label, hi) in historySubTabLabels" :key="label" :value="hi">{{ label
              }}</el-radio-button>
            </el-radio-group>
          </div>
          <section class="history-list" aria-label="开奖记录">
            <article v-for="(rec, ri) in historyDrawRecords" :key="ri" class="history-card">
              <div class="history-card-head">
                <span class="history-game-name">{{ historyGameTag }}</span>
                <span class="history-period-line">第 <strong class="history-period-num">{{ rec.periodShort }}</strong>
                  期</span>
              </div>
              <div class="history-card-divider" role="presentation" />

              <div class="history-card-content">
                <template v-if="historySubTab === 0">
                  <div class="history-balls">
                    <div v-for="(b, bi) in rec.balls" :key="bi" class="history-ball history-ball--primary">{{ b }}
                    </div>
                  </div>
                </template>
                <template v-else-if="historySubTab === 1">
                  <div class="history-sq-row">
                    <span v-for="(b, bi) in rec.balls" :key="bi" class="history-sq history-sq-dx"
                      :class="historyBigSmallDigit(b) === '大' ? 'history-sq-dx--big' : 'history-sq-dx--small'">{{
                        historyBigSmallDigit(b) }}</span>
                  </div>
                </template>
                <template v-else-if="historySubTab === 2">
                  <div class="history-sq-row">
                    <span v-for="(b, bi) in rec.balls" :key="bi" class="history-sq history-sq-oe"
                      :class="historyParityDigit(b) === '单' ? 'history-sq-oe--odd' : 'history-sq-oe--even'">{{
                        historyParityDigit(b) }}</span>
                  </div>
                </template>
                <template v-else-if="historySubTab === 3">
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
            已加载最近50期数据
            <span class="history-foot-dot" />
          </div>
        </div>
      </template>
      <template v-else-if="tab === 4">
        <div class="table-card">
          <el-table :data="betRecordRows" class="detail-bet-table" size="small" stripe empty-text="暂无数据"
            :style="{ width: '100%' }">
            <el-table-column prop="period" label="期数" :min-width="44" />
            <el-table-column prop="playMethod" label="玩法" :min-width="44" />
            <el-table-column prop="multiplier" label="倍数" :min-width="34" align="center" />
            <el-table-column prop="round" label="轮次" :min-width="34" align="center" />
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
            <el-table-column label="状态" :min-width="40" align="center">
              <template #default="{ row }">
                <el-tag type="primary" effect="light" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
      <div v-else class="tab-placeholder">
        <p>「{{ tabLabels[tab] }}」功能开发中</p>
      </div>
    </main>

    <div v-if="tab !== 2 && tab !== 3 && tab !== 4" class="bet-dock" :class="{ 'is-collapsed': !betDockOpen }"
      aria-label="投注区">
      <button type="button" class="dock-handle" :aria-expanded="betDockOpen" aria-controls="bet-dock-panel"
        :aria-label="betDockOpen ? '收起投注区' : '展开投注区'" @click="betDockOpen = !betDockOpen">
        <img :src="ICON_PLACEHOLDER" alt="" width="28" height="28" class="handle-ico-img"
          :class="{ 'handle-ico-collapsed': !betDockOpen }" decoding="async" />
      </button>
      <div id="bet-dock-panel" v-show="betDockOpen" class="dock-inner">
        <el-form class="dock-form dock-form--row" label-width="auto">
          <el-form-item label="倍数" class="dock-form-item--mult">
            <template v-if="betDockEntryMode === 'manual'">
              <div class="dock-mult-manual">
                <el-input-number v-model="betMultiplier" :min="1" :controls="true" controls-position="right"
                  size="small" class="dock-inp-num" />
                <span class="dock-unit">倍</span>
              </div>
            </template>
            <el-button v-else type="primary" plain size="small" class="dock-multiplier-settings-btn"
              @click="goBetMultiplierSettings">
              请设置
            </el-button>
          </el-form-item>
          <el-form-item label="模式" class="dock-form-item--mode">
            <el-select v-model="betMode" size="small" class="dock-select" placeholder="模式">
              <el-option label="2元" value="2元" />
              <el-option label="1元" value="1元" />
              <el-option label="0.2元" value="0.2元" />
            </el-select>
          </el-form-item>
        </el-form>
        <div class="dock-bottom">
          <div class="stats">
            <div class="stat-line">
              <span class="stat-l">余额:</span>
              <span class="stat-v err">0</span>
              <span class="stat-u">元</span>
            </div>
            <div class="stat-line">
              <span class="stat-l">选中:</span>
              <span class="stat-v err">3</span>
              <span class="stat-u">注</span>
            </div>
            <div class="stat-line">
              <span class="stat-l">总额:</span>
              <span class="stat-v err">6</span>
              <span class="stat-u">元</span>
            </div>
            <div class="stat-line">
              <span class="stat-l">奖金:</span>
              <span class="stat-v err">19.78</span>
              <span class="stat-u">元(预估)</span>
            </div>
          </div>
          <div class="dock-actions-col">
            <el-button type="primary" class="dock-confirm-btn dock-confirm-btn--stacked">
              {{ betDockEntryMode === 'manual' ? '确认投注' : '添加至云端' }}
            </el-button>
            <el-button type="default" class="dock-switch-mode-btn" @click="toggleBetDockEntryMode">
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
  min-height: 100dvh;
  background: var(--surface);
  color: #191c1e;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: calc(18rem + env(safe-area-inset-bottom));
  -webkit-font-smoothing: antialiased;
  transition: padding-bottom 0.25s ease;
}

.detail.dock-collapsed {
  padding-bottom: calc(2.5rem + env(safe-area-inset-bottom));
}

.detail--plan-trend {
  padding-bottom: calc(1.25rem + env(safe-area-inset-bottom));
}

.primary-ico-img {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
  display: block;
  cursor: pointer;
  pointer-events: none;
}

.header-wrap {
  position: sticky;
  top: 0;
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
  padding: 1rem 1.5rem;
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
  width: 100%;
  max-width: 32rem;
  margin: 0 auto;
  padding: 0.5rem 0.5rem 0;
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

.plan-inverse-digits {
  margin: 0;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  color: #0f172a;
  font-weight: 500;
  font-size: 1.125rem;
  line-height: 1.625;
  word-break: break-all;
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
  margin-top: 0.5rem;
  width: 100%;
}

.plan-trend-y-axis {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  flex-shrink: 0;
  padding-right: 0.75rem;
  padding-top: 0.25rem;
  height: calc(16rem - 1.5rem);
  font-size: 11px;
  font-weight: 500;
  color: #94a3b8;
}

.plan-trend-chart-plot {
  position: relative;
  flex: 1;
  min-width: 0;
  height: 16rem;
}

.plan-trend-grid {
  position: absolute;
  inset: 0;
  bottom: 1.5rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  pointer-events: none;
}

.plan-trend-grid-line {
  border-bottom: 1px solid #f1f5f9;
  height: 0;
  width: 100%;
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
  max-height: 16rem;
  -webkit-overflow-scrolling: touch;
}

.plan-trend-history-scroll::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.plan-trend-history-scroll::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 999px;
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
