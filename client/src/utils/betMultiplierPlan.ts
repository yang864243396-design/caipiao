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

const MAX_MULT = 200_000
/** 时时彩常见赔率，用于计划表测算 */
const DEFAULT_ODDS = 9.8

function unitAmount(mode: string): number {
  return mode === '角' ? 0.1 : 1
}

function fmt(n: number, digits = 2): string {
  if (!Number.isFinite(n)) return '0'
  return n.toFixed(digits).replace(/\.?0+$/, '') || '0'
}

function marginPct(profit: number, totalBet: number): number {
  if (totalBet <= 0) return 0
  return (profit / totalBet) * 100
}

/** 最低利润率：前面都没中，最后一期中奖时 (利润不含本金) ÷ 追号总金额 */
function lastPeriodMargin(
  mults: number[],
  unit: number,
  odds: number,
): { rows: PlanTableRow[]; lastMargin: number; lastProfit: number } | null {
  if (!mults.length) return null
  const rows: PlanTableRow[] = []
  let totalBet = 0
  for (let i = 0; i < mults.length; i++) {
    const mult = Math.min(Math.max(1, Math.floor(mults[i]!)), MAX_MULT)
    const curBet = unit * mult
    totalBet += curBet
    const prize = curBet * odds
    const profit = prize - totalBet
    const margin = marginPct(profit, totalBet)
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
  const last = rows[rows.length - 1]!
  return {
    rows,
    lastMargin: Number(last.margin),
    lastProfit: Number(last.profit),
  }
}

function buildLineMults(periods: number): number[] {
  const out: number[] = []
  for (let i = 0; i < periods; i++) {
    out.push(Math.min(2 ** i, MAX_MULT))
  }
  return out
}

function buildWaveMults(periods: number): number[] {
  const out: number[] = []
  let m = 1
  for (let i = 0; i < periods; i++) {
    out.push(m)
    if (i % 2 === 0) m = Math.min(m * 2, MAX_MULT)
    else m = Math.max(1, Math.floor(m / 2))
  }
  return out
}

function buildFollowStopMults(periods: number): number[] {
  const out: number[] = []
  let m = 1
  for (let i = 0; i < periods; i++) {
    out.push(m)
    if (i > 0 && i % 3 === 0) m = 1
    else m = Math.min(m * 2, MAX_MULT)
  }
  return out
}

function buildSuspendMults(periods: number, suspend: number): number[] {
  const out: number[] = []
  let m = 1
  let skip = 0
  for (let i = 0; i < periods; i++) {
    if (skip > 0) {
      out.push(0)
      skip--
      continue
    }
    out.push(m)
    m = Math.min(m * 2, MAX_MULT)
    if (i > 0 && (i + 1) % (suspend + 2) === 0) skip = suspend
  }
  return out.filter((x, idx, arr) => {
    if (x > 0) return true
    return arr.slice(idx).some((v) => v > 0)
  })
}

function expandPeriodsUntilTarget(
  baseMults: (n: number) => number[],
  unit: number,
  targetMargin: number,
  maxPeriods = 20,
): PlanTableRow[] | null {
  for (let n = 2; n <= maxPeriods; n++) {
    const mults = baseMults(n).filter((m) => m > 0)
    if (!mults.length) continue
    const r = lastPeriodMargin(mults, unit, DEFAULT_ODDS)
    if (r && r.lastMargin >= targetMargin - 0.01) return r.rows
  }
  return null
}

export interface NewbiePlanInput {
  principal: string
  mode: string
  profitType: 'rate' | 'fixed' | 'accum'
  rateVal: string
  fixedVal: string
  accumStart: string
  accumStep: string
  preset: 'line' | 'followStop' | 'suspend1' | 'suspend2'
}

export function canGenerateNewbiePlan(input: NewbiePlanInput): string | null {
  const p = Number(input.principal)
  if (!input.principal.trim() || !Number.isFinite(p) || p <= 0) return '请填写总本金'
  if (input.profitType === 'rate') {
    const r = Number(input.rateVal)
    if (!input.rateVal.trim() || !Number.isFinite(r) || r <= 0) return '请填写收益利率'
  } else if (input.profitType === 'fixed') {
    const f = Number(input.fixedVal)
    if (!input.fixedVal.trim() || !Number.isFinite(f) || f <= 0) return '请填写固定利润'
  } else {
    const s = Number(input.accumStart)
    const step = Number(input.accumStep)
    if (!input.accumStart.trim() || !Number.isFinite(s) || s < 0) return '请填写累加起步'
    if (!input.accumStep.trim() || !Number.isFinite(step) || step <= 0) return '请填写累进步长'
  }
  return null
}

export function generateNewbiePlan(input: NewbiePlanInput): PlanTableRow[] | null {
  const err = canGenerateNewbiePlan(input)
  if (err) return null

  const unit = unitAmount(input.mode)
  const principal = Number(input.principal)
  let targetMargin = 10
  if (input.profitType === 'rate') {
    targetMargin = Number(input.rateVal)
  } else if (input.profitType === 'fixed') {
    const fixed = Number(input.fixedVal)
    targetMargin = (fixed / principal) * 100
  } else {
    const start = Number(input.accumStart)
    targetMargin = ((start + Number(input.accumStep)) / principal) * 100
  }
  if (!Number.isFinite(targetMargin) || targetMargin <= 0) return null

  const baseFn = (n: number): number[] => {
    switch (input.preset) {
      case 'followStop':
        return buildFollowStopMults(n)
      case 'suspend1':
        return buildSuspendMults(n, 1)
      case 'suspend2':
        return buildSuspendMults(n, 2)
      default:
        return buildLineMults(n)
    }
  }

  const rows = expandPeriodsUntilTarget(baseFn, unit, targetMargin)
  return rows
}

export interface OneclickPlanInput {
  cycle: string
  profit: string
  preset: 'line' | 'wave'
}

export function canGenerateOneclickPlan(input: OneclickPlanInput): string | null {
  const c = Number(input.cycle)
  if (!input.cycle.trim() || !Number.isInteger(c) || c <= 0) return '请填写计划周期'
  const p = Number(input.profit)
  if (!input.profit.trim() || !Number.isFinite(p) || p <= 0) return '请填写收益利润'
  return null
}

export function generateOneclickPlan(input: OneclickPlanInput): PlanTableRow[] | null {
  const err = canGenerateOneclickPlan(input)
  if (err) return null

  const periods = Math.min(Math.max(1, Math.floor(Number(input.cycle))), 30)
  const unit = 1
  const mults =
    input.preset === 'wave' ? buildWaveMults(periods) : buildLineMults(periods)
  const r = lastPeriodMargin(mults.filter((m) => m > 0), unit, DEFAULT_ODDS)
  return r?.rows ?? null
}
