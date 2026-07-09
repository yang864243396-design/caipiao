import { requestApi } from './client'

export interface AdminBetOrderRow {
  orderNo: string
  issueNo: string
  member: string
  lottery: string
  schemeName: string
  amount: number
  payoutAmount: number
  resultStatus: 'hit' | 'miss' | 'pending' | 'cancel' | string
  created: string
}

export interface AdminChaseOrderRow {
  chaseNo: string
  member: string
  lottery: string
  totalIssues: number
  doneIssues: number
  periodsLeft: number
  amount: number
  status: 'running' | 'completed' | 'cancelled' | string
  created: string
}

export interface AdminLedgerRow {
  id: string
  member: string
  schemeName: string
  currency: string
  amount: number
  flowType: string
  flowTypeCode: 'income' | 'expense'
  balanceAfter: number
  ledgerNo: string
  time: string
}

export interface AdminOrderListResult<T> {
  items: T[]
  total: number
}

export interface FetchBetOrdersParams {
  issueNo?: string
  memberAccount?: string
  schemeName?: string
  lotteryCode?: string
  page?: number
  pageSize?: number
}

export interface FetchLedgerParams {
  dateFrom: string
  dateTo: string
  flowType?: string
  currency?: string
  memberAccount?: string
  ledgerNo?: string
  page?: number
  pageSize?: number
}

function buildLedgerQuery(params: FetchLedgerParams): string {
  const query = new URLSearchParams()
  query.set('dateFrom', params.dateFrom)
  query.set('dateTo', params.dateTo)
  if (params.flowType && params.flowType !== 'all') query.set('flowType', params.flowType)
  if (params.currency && params.currency !== 'all') query.set('currency', params.currency)
  if (params.memberAccount) query.set('memberAccount', params.memberAccount)
  if (params.ledgerNo) query.set('ledgerNo', params.ledgerNo)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  return `?${query.toString()}`
}

export interface FetchChaseOrdersParams {
  chaseNo?: string
  memberAccount?: string
  status?: string
  lotteryCode?: string
  page?: number
  pageSize?: number
}

function buildChaseQuery(params: FetchChaseOrdersParams): string {
  const query = new URLSearchParams()
  if (params.chaseNo) query.set('chaseNo', params.chaseNo)
  if (params.memberAccount) query.set('memberAccount', params.memberAccount)
  if (params.status) query.set('status', params.status)
  if (params.lotteryCode) query.set('lotteryCode', params.lotteryCode)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  const qs = query.toString()
  return qs ? `?${qs}` : ''
}
function buildBetQuery(params: FetchBetOrdersParams): string {
  const query = new URLSearchParams()
  if (params.issueNo) query.set('issueNo', params.issueNo)
  if (params.memberAccount) query.set('memberAccount', params.memberAccount)
  if (params.schemeName) query.set('schemeName', params.schemeName)
  if (params.lotteryCode) query.set('lotteryCode', params.lotteryCode)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  const qs = query.toString()
  return qs ? `?${qs}` : ''
}

export async function fetchAdminBetOrders(
  params: FetchBetOrdersParams = {},
): Promise<AdminOrderListResult<AdminBetOrderRow>> {
  return requestApi<AdminOrderListResult<AdminBetOrderRow>>(`/admin/orders/bets${buildBetQuery(params)}`)
}

export async function fetchAdminChaseOrders(
  params: FetchChaseOrdersParams = {},
): Promise<AdminOrderListResult<AdminChaseOrderRow>> {
  return requestApi<AdminOrderListResult<AdminChaseOrderRow>>(`/admin/orders/chases${buildChaseQuery(params)}`)
}

export async function fetchAdminLedgerEntries(
  params: FetchLedgerParams,
): Promise<AdminOrderListResult<AdminLedgerRow>> {
  return requestApi<AdminOrderListResult<AdminLedgerRow>>(`/admin/orders/ledger${buildLedgerQuery(params)}`)
}
