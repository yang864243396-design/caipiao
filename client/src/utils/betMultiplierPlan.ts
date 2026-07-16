/** 倍投计划表行（与 BetMultiplierSettingsView 表格列一致） */
export interface PlanTableRow {
  period: string
  mult: string
  curBet: string
  totalBet: string
  prize: string
  profit: string
  margin: string
}

/** 倍数搜索 / 展示上限（与页面提示一致） */
export const MAX_MULT = 200_000

export type CalcType = 'rate' | 'fixed' | 'step' | 'free'
export type AdvanceMode = 'on_lose' | 'on_win'

export const DEFAULT_SIDES_PRESET: readonly number[] = [
  2, 4, 8, 17, 36, 76, 160, 338, 714, 1507,
]
export const AGGRESSIVE_PRESET: readonly number[] = [
  6, 12, 25, 52, 108, 224, 465, 965, 2003, 4156,
]

function fmt(n: number, digits = 2): string {
  if (!Number.isFinite(n)) return '0'
  return n.toFixed(digits).replace(/\.?0+$/, '') || '0'
}

function clampMult(n: number): number {
  if (!Number.isFinite(n) || n < 1) return 1
  return Math.min(Math.floor(n), MAX_MULT)
}

/** 避免 1.9-1=0.8999… 导致 ceil 多进 1 */
function ceilDiv(numer: number, denom: number): number {
  if (!(denom > 0) || !(numer >= 0)) return MAX_MULT
  return Math.ceil(numer / denom - 1e-10)
}

/** 由倍数表 + 账本参数生成预览行（连亏假设） */
export function buildPlanRowsFromTimes(
  mults: number[],
  money: number,
  number: number,
  mode: number,
): PlanTableRow[] {
  const rows: PlanTableRow[] = []
  let totalBet = 0
  const unit = money * number
  for (let i = 0; i < mults.length; i++) {
    const mult = clampMult(mults[i]!)
    const curBet = unit * mult
    totalBet += curBet
    const prize = mode * mult
    const profit = prize - totalBet
    const margin = totalBet > 0 ? (profit / totalBet) * 100 : 0
    rows.push({
      period: String(i + 1),
      mult: String(mult),
      curBet: fmt(curBet),
      totalBet: fmt(totalBet),
      prize: fmt(prize),
      profit: fmt(profit),
      margin: fmt(margin, 2),
    })
  }
  return rows
}

/**
 * 小白简化递推（与默认两面表对齐）：
 * lost = first * money * number
 * b[0] = first
 * b[n] = ceil((lost + profit) / (unit * (odds - 1)))
 * lost += b[n] * unit
 */
export function buildNewbieTimesList(input: {
  odds: number
  firstBet: number
  targetProfit: number
  cycle: number
  money?: number
  number?: number
}): number[] | null {
  const odds = Number(input.odds)
  const first = Math.floor(Number(input.firstBet))
  const profit = Number(input.targetProfit)
  const cycle = Math.floor(Number(input.cycle))
  const money = Number(input.money ?? 1)
  const number = Math.floor(Number(input.number ?? 1))
  if (!(odds > 1) || !(first >= 1) || !(profit > 0) || !(cycle >= 1) || cycle > 100) return null
  if (!(money > 0) || !(number >= 1)) return null

  const unit = money * number
  const denom = unit * (odds - 1)
  if (!(denom > 0)) return null

  const list: number[] = []
  let lost = first * unit
  list.push(clampMult(first))
  for (let i = 1; i < cycle; i++) {
    const raw = ceilDiv(lost + profit, denom)
    const times = clampMult(raw)
    if (times >= MAX_MULT && raw > MAX_MULT) {
      // 超上限：截断到已生成档，至少保留首档
      break
    }
    list.push(times)
    lost += times * unit
  }
  return list.length ? list : null
}

export interface NewbiePlanInput {
  odds: string
  firstBet: string
  targetProfit: string
  cycle: string
  money: string
  number: string
}

export function canGenerateNewbiePlan(input: NewbiePlanInput): string | null {
  const odds = Number(input.odds)
  if (!input.odds.trim() || !Number.isFinite(odds) || odds <= 1) return '请填写大于 1 的赔率'
  const first = Number(input.firstBet)
  if (!input.firstBet.trim() || !Number.isInteger(first) || first < 1) return '请填写首注倍数（正整数）'
  const profit = Number(input.targetProfit)
  if (!input.targetProfit.trim() || !Number.isFinite(profit) || profit <= 0) return '请填写目标利润'
  const cycle = Number(input.cycle)
  if (!input.cycle.trim() || !Number.isInteger(cycle) || cycle < 1 || cycle > 100) {
    return '请填写计划档数（1～100）'
  }
  const money = Number(input.money)
  if (!input.money.trim() || !Number.isFinite(money) || money <= 0) return '请填写单价'
  const number = Number(input.number)
  if (!input.number.trim() || !Number.isInteger(number) || number < 1) return '请填写注数（正整数）'
  return null
}

export function generateNewbiePlan(input: NewbiePlanInput): PlanTableRow[] | null {
  const err = canGenerateNewbiePlan(input)
  if (err) return null
  const odds = Number(input.odds)
  const money = Number(input.money)
  const number = Number(input.number)
  const list = buildNewbieTimesList({
    odds,
    firstBet: Number(input.firstBet),
    targetProfit: Number(input.targetProfit),
    cycle: Number(input.cycle),
    money,
    number,
  })
  if (!list?.length) return null
  const mode = money * number * odds
  return buildPlanRowsFromTimes(list, money, number, mode)
}

/** 单档账本（连亏假设） */
export function ledgerAtTimes(
  times: number,
  prevTotal: number,
  money: number,
  number: number,
  mode: number,
): { output: number; total: number; input: number; gain: number; gainRatio: number } {
  const t = clampMult(times)
  const output = money * number * t
  const total = prevTotal + output
  const input = mode * t
  const gain = input - total
  const gainRatio = total > 0 ? (gain / total) * 100 : Number.NaN
  return { output, total, input, gain, gainRatio }
}

function meetsTarget(
  calcType: CalcType,
  gain: number,
  gainRatio: number,
  opts: {
    targetRate?: number
    targetProfit?: number
    stepThreshold?: number
  },
): boolean {
  switch (calcType) {
    case 'rate':
      return Number.isFinite(gainRatio) && gainRatio >= (opts.targetRate ?? 0)
    case 'fixed':
      return gain >= (opts.targetProfit ?? 0)
    case 'step':
      return gain >= (opts.stepThreshold ?? 0)
    case 'free':
      return true
    default:
      return false
  }
}

/** 从 times=1 起找最小倍数满足目标；失败返回 null */
export function findTimes(
  prevTotal: number,
  money: number,
  number: number,
  mode: number,
  calcType: CalcType,
  opts: {
    targetRate?: number
    targetProfit?: number
    stepThreshold?: number
  },
): number | null {
  if (!(money > 0) || !(number >= 1) || !(mode > money * number)) return null
  for (let times = 1; times <= MAX_MULT; times++) {
    const { gain, gainRatio } = ledgerAtTimes(times, prevTotal, money, number, mode)
    if (meetsTarget(calcType, gain, gainRatio, opts)) return times
  }
  return null
}

export interface OneclickPlanInput {
  money: string
  number: string
  mode: string
  cycle: string
  calcType: CalcType
  targetRate: string
  targetProfit: string
  sumBegin: string
  sumStep: string
  freeList: string
}

export function canGenerateOneclickPlan(input: OneclickPlanInput): string | null {
  const money = Number(input.money)
  if (!input.money.trim() || !Number.isFinite(money) || money <= 0) return '请填写单价'
  const number = Number(input.number)
  if (!input.number.trim() || !Number.isInteger(number) || number < 1) return '请填写注数'
  const mode = Number(input.mode)
  if (!input.mode.trim() || !Number.isFinite(mode) || mode <= 0) return '请填写单倍奖金'
  if (!(mode > money * number)) return '单倍奖金须大于单价×注数'
  const cycle = Number(input.cycle)
  if (!input.cycle.trim() || !Number.isInteger(cycle) || cycle < 1 || cycle > 100) {
    return '请填写计划周期（1～100）'
  }
  if (input.calcType === 'rate') {
    const r = Number(input.targetRate)
    if (!input.targetRate.trim() || !Number.isFinite(r) || r <= 0) return '请填写收益率'
  } else if (input.calcType === 'fixed') {
    const p = Number(input.targetProfit)
    if (!input.targetProfit.trim() || !Number.isFinite(p) || p <= 0) return '请填写固定利润'
  } else if (input.calcType === 'step') {
    const b = Number(input.sumBegin)
    const s = Number(input.sumStep)
    if (!input.sumBegin.trim() || !Number.isFinite(b) || b < 0) return '请填写累加起步利润'
    if (!input.sumStep.trim() || !Number.isFinite(s) || s <= 0) return '请填写累进步长'
  } else if (input.calcType === 'free') {
    const list = parseFreeList(input.freeList)
    if (!list) return '请填写自由倍数表（逗号分隔正整数）'
    if (list.length !== cycle) return '倍数的个数和周期不一致'
  }
  return null
}

export function parseFreeList(raw: string): number[] | null {
  const parts = String(raw ?? '')
    .split(/[,，;；\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (!parts.length) return null
  const out: number[] = []
  for (const p of parts) {
    const n = Number(p)
    if (!Number.isInteger(n) || n < 1) return null
    out.push(clampMult(n))
  }
  return out
}

/** 完整计算器：连亏累加 + 最小倍数搜索 */
export function buildOneclickTimesList(input: {
  money: number
  number: number
  mode: number
  cycle: number
  calcType: CalcType
  targetRate?: number
  targetProfit?: number
  sumBegin?: number
  sumStep?: number
  freeList?: number[]
}): number[] | null {
  const { money, number, mode, cycle, calcType } = input
  if (!(money > 0) || !(number >= 1) || !(mode > money * number) || cycle < 1) return null

  if (calcType === 'free') {
    const free = input.freeList
    if (!free || free.length !== cycle) return null
    return free.map(clampMult)
  }

  const list: number[] = []
  let prevTotal = 0
  let sumBegin = Number(input.sumBegin ?? 0)

  for (let i = 0; i < cycle; i++) {
    const opts = {
      targetRate: input.targetRate,
      targetProfit: input.targetProfit,
      stepThreshold:
        calcType === 'step' ? sumBegin + Number(input.sumStep ?? 0) : undefined,
    }
    const times = findTimes(prevTotal, money, number, mode, calcType, opts)
    if (times == null) return null
    list.push(times)
    const led = ledgerAtTimes(times, prevTotal, money, number, mode)
    prevTotal = led.total
    if (calcType === 'step') {
      sumBegin = led.gain
    }
  }
  return list
}

export function generateOneclickPlan(input: OneclickPlanInput): PlanTableRow[] | null {
  const err = canGenerateOneclickPlan(input)
  if (err) return null
  const money = Number(input.money)
  const number = Number(input.number)
  const mode = Number(input.mode)
  const cycle = Number(input.cycle)
  const list = buildOneclickTimesList({
    money,
    number,
    mode,
    cycle,
    calcType: input.calcType,
    targetRate: Number(input.targetRate),
    targetProfit: Number(input.targetProfit),
    sumBegin: Number(input.sumBegin),
    sumStep: Number(input.sumStep),
    freeList: input.calcType === 'free' ? parseFreeList(input.freeList) ?? undefined : undefined,
  })
  if (!list?.length) return null
  return buildPlanRowsFromTimes(list, money, number, mode)
}

export function applyPresetTimes(
  preset: readonly number[],
  money: number,
  number: number,
  mode: number,
): PlanTableRow[] {
  return buildPlanRowsFromTimes([...preset], money, number, mode)
}
