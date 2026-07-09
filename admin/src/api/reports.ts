import { requestApi } from './client'

export interface LotteryStatSummary {
  effectiveBetYuan: number
  payoutYuan: number
  dateFrom: string
  dateTo: string
}

export interface LotteryStatRow {
  lottery: string
  betCount: number
  effectiveBetYuan: number
  memberPnlYuan: number
}

export interface LotteryStatResult {
  summary: LotteryStatSummary
  items: LotteryStatRow[]
}

export interface PnlSummary {
  platformPnlYuan: number
  validBetYuan: number
  dateFrom: string
  dateTo: string
}

export interface PnlDailyRow {
  period: string
  validBetYuan: number
  platformPnlYuan: number
}

export interface PnlReportResult {
  summary: PnlSummary
  items: PnlDailyRow[]
}

export interface DailyLotterySummary {
  betCount: number
  betAmountYuan: number
  platformPnlYuan: number
  dateFrom: string
  dateTo: string
}

export interface DailyLotteryRow {
  date: string
  lotteryCode: string
  lottery: string
  betCount: number
  betAmountYuan: number
  platformPnlYuan: number
}

export interface DailyLotteryReportResult {
  summary: DailyLotterySummary
  items: DailyLotteryRow[]
}

export async function fetchDailyLotteryReport(params: {
  dateFrom?: string
  dateTo?: string
  lotteryCode?: string
}): Promise<DailyLotteryReportResult> {
  const q = new URLSearchParams()
  if (params.dateFrom) q.set('dateFrom', params.dateFrom)
  if (params.dateTo) q.set('dateTo', params.dateTo)
  if (params.lotteryCode) q.set('lotteryCode', params.lotteryCode)
  const qs = q.toString()
  return requestApi<DailyLotteryReportResult>(`/admin/reports/daily-lottery${qs ? `?${qs}` : ''}`)
}

export async function fetchLotteryStatReport(params: {
  dateFrom?: string
  dateTo?: string
}): Promise<LotteryStatResult> {
  const q = new URLSearchParams()
  if (params.dateFrom) q.set('dateFrom', params.dateFrom)
  if (params.dateTo) q.set('dateTo', params.dateTo)
  const qs = q.toString()
  return requestApi<LotteryStatResult>(`/admin/reports/lottery-stat${qs ? `?${qs}` : ''}`)
}

export async function fetchPnlReport(params: {
  dateFrom?: string
  dateTo?: string
}): Promise<PnlReportResult> {
  const q = new URLSearchParams()
  if (params.dateFrom) q.set('dateFrom', params.dateFrom)
  if (params.dateTo) q.set('dateTo', params.dateTo)
  const qs = q.toString()
  return requestApi<PnlReportResult>(`/admin/reports/pnl${qs ? `?${qs}` : ''}`)
}
