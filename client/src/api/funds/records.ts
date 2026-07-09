import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

export type FundFlowType = 'all' | 'income' | 'expense'

export type FundCurrency = 'all' | 'USDT' | 'TRX' | 'CNY'

export interface FundRecordItem {
  id: string
  schemeName: string
  currency: string
  amount: number
  time: string
  flowType: string
  flowTypeCode: 'income' | 'expense'
  balanceAfter: number
  ledgerNo: string
}

export interface FundRecordsResult {
  items: FundRecordItem[]
  page: {
    hasMore: boolean
    nextCursor?: string | null
  }
}

export interface FundRecordsQuery {
  dateFrom?: string
  dateTo?: string
  flowType?: FundFlowType
  currency?: FundCurrency
  cursor?: string
  limit?: number
}

export async function fetchFundRecords(query: FundRecordsQuery = {}): Promise<FundRecordsResult> {
  await ensureClientSession()
  return requestApi<FundRecordsResult>('/client/funds/records', {
    query: {
      dateFrom: query.dateFrom,
      dateTo: query.dateTo,
      flowType: query.flowType && query.flowType !== 'all' ? query.flowType : undefined,
      currency: query.currency && query.currency !== 'all' ? query.currency : undefined,
      cursor: query.cursor,
      limit: query.limit,
    },
  })
}

export function formatFundRecordMoney(n: number): string {
  return n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function formatFundRecordTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('zh-CN', { hour12: false })
}

export function formatFundRecordAmount(amount: number): string {
  const abs = formatFundRecordMoney(Math.abs(amount))
  if (!Number.isFinite(amount) || amount === 0) return abs
  return amount > 0 ? `+${abs}` : `-${abs}`
}

export interface FundRecordDisplayRow {
  id: string
  schemeName: string
  currency: string
  amount: string
  time: string
  flowType: string
  balanceAfter: string
  tone: 'income' | 'expense' | ''
}

export function toFundRecordDisplayRow(item: FundRecordItem): FundRecordDisplayRow {
  return {
    id: item.id,
    schemeName: item.schemeName,
    currency: item.currency,
    amount: formatFundRecordAmount(item.amount),
    time: formatFundRecordTime(item.time),
    flowType: item.flowType,
    balanceAfter: formatFundRecordMoney(item.balanceAfter),
    tone: item.flowTypeCode === 'income' ? 'income' : item.flowTypeCode === 'expense' ? 'expense' : '',
  }
}
