import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import type { MoneySummary, Paginated, RunMode } from '@/api/types'



export type BetRecordMode = RunMode

export type BetRecordStatus = 'pending' | 'won' | 'lost' | 'cancelled' | 'hit' | 'miss'



export interface BetRecordSchemeGroup {

  schemeId: string

  schemeName: string

  totalBet: number

  totalPrize?: number

  dayPnl: number

  winRate: number

}



export interface BetRecordGroupsData {

  mode: RunMode

  days: number

  /** 统计区间起（UTC+8 自然日，含） */

  dateFrom?: string

  /** 统计区间止（UTC+8 自然日，含） */

  dateTo?: string

  summary: MoneySummary

  groups: Paginated<BetRecordSchemeGroup>

}

export const BET_RECORD_GROUP_PAGE_SIZE = 10

export const BET_RECORD_DETAIL_PAGE_SIZE = 20



export interface BetRecordItem {

  id: string

  period: string

  /** 第三方接单返回的 periods；仅真实接单成功后有值 */
  periods?: string

  playType: string

  multiplier: string

  round: string

  amount: number

  pnl: number

  status: BetRecordStatus

  /** 实际下注号码（可能多行，含 \n） */
  betContent?: string

}



export interface BetRecordDetailData {

  schemeId: string

  schemeName: string

  mode: RunMode

  days: number

  dateFrom?: string

  dateTo?: string

  summary: MoneySummary

  records: Paginated<BetRecordItem>

}



/** 方案投注明细表格行 */
export interface BetRecordDisplayRow {
  period: string
  multiplier: string
  round: string
  amount: string
  pnl: string
  pnlPositive: boolean
  status: BetRecordStatus
  statusLabel: string
  statusHit: boolean | null
}

/** 投注记录展示期号：优先第三方 periods，与第三方 UI 一致 */
export function betRecordDisplayPeriod(item: BetRecordItem): string {
  const p = (item.periods || item.period || '').trim()
  return p || '—'
}

export function formatBetMultiplierDisplay(raw: string): string {
  const n = Number(raw)
  if (Number.isFinite(n)) return String(Math.round(n))
  const dot = raw.indexOf('.')
  if (dot > 0) {
    const head = raw.slice(0, dot).trim()
    if (head) return head
  }
  return raw.trim() || '1'
}

export function formatBetRoundDisplay(raw: string): string {
  const slash = raw.indexOf('/')
  if (slash > 0) {
    const head = raw.slice(0, slash).trim()
    if (head) return head
  }
  return raw.trim() || '1'
}

export function formatBetRecordStatus(status: BetRecordStatus): {
  label: string
  hit: boolean | null
} {
  if (status === 'hit' || status === 'won') return { label: '中', hit: true }
  if (status === 'miss' || status === 'lost') return { label: '挂', hit: false }
  if (status === 'pending') return { label: '待开奖', hit: null }
  if (status === 'cancelled') return { label: '已撤单', hit: null }
  return { label: String(status), hit: null }
}

export function toDisplayRow(item: BetRecordItem): BetRecordDisplayRow {
  const abs = Math.abs(item.pnl).toFixed(2)
  const statusFmt = formatBetRecordStatus(item.status)
  return {
    period: betRecordDisplayPeriod(item),
    multiplier: formatBetMultiplierDisplay(item.multiplier),
    round: formatBetRoundDisplay(item.round),
    amount: item.amount.toFixed(2),
    pnl: item.pnl >= 0 ? `+${abs}` : `-${abs}`,
    pnlPositive: item.pnl >= 0,
    status: item.status,
    statusLabel: statusFmt.label,
    statusHit: statusFmt.hit,
  }
}



export interface BetRecordGroupsQuery {
  mode?: BetRecordMode
  days?: number
  dateFrom?: string
  dateTo?: string
  lotteryCode?: string
  cursor?: string
  limit?: number
}

export async function fetchBetRecordGroups(
  query: BetRecordGroupsQuery = {},
): Promise<BetRecordGroupsData> {
  await ensureClientSession()
  return requestApi<BetRecordGroupsData>('/client/cloud/bet-records', {
    query: {
      mode: query.mode,
      days: query.days,
      dateFrom: query.dateFrom,
      dateTo: query.dateTo,
      lotteryCode: query.lotteryCode,
      cursor: query.cursor,
      limit: query.limit,
    },
  })
}

export interface BetRecordDetailQuery {
  mode?: BetRecordMode
  days?: number
  cursor?: string
  limit?: number
}

export async function fetchBetRecordDetail(
  schemeId: string,
  query: BetRecordDetailQuery = {},
): Promise<BetRecordDetailData> {
  await ensureClientSession()
  const mode = query.mode ?? 'real'
  const days = query.days ?? 3
  return requestApi<BetRecordDetailData>(`/client/cloud/bet-records/${encodeURIComponent(schemeId)}`, {
    query: {
      mode,
      days,
      cursor: query.cursor,
      limit: query.limit,
    },
  })
}


