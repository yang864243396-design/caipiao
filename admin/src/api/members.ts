import { requestApi } from './client'
import type { FundCurrency, FundFlowType, MemberFundRecordsResult } from '@/types/members'

export interface AdminGuajiBalances {
  usdt: number
  trx: number
  cny: number
}

export interface AdminMemberRow {
  id: string
  account: string
  displayName: string
  guajiBalances: AdminGuajiBalances
  balanceYuan?: number
  status: '正常' | '禁用'
  registeredAt: string
  lastLoginAt: string
}

export interface AdminMemberListResult {
  items: AdminMemberRow[]
  total: number
}

export type MemberSearchField = 'account' | 'guajiAccount' | 'id'

export interface FetchMembersParams {
  keyword?: string
  searchField?: MemberSearchField
  page?: number
  pageSize?: number
}

const emptyGuajiBalances = (): AdminGuajiBalances => ({ usdt: 0, trx: 0, cny: 0 })

function mapMember(row: AdminMemberRow): AdminMemberRow {
  return {
    ...row,
    guajiBalances: row.guajiBalances ?? emptyGuajiBalances(),
  }
}

export async function fetchMembers(params: FetchMembersParams = {}): Promise<AdminMemberListResult> {
  const query = new URLSearchParams()
  query.set('searchField', params.searchField ?? 'account')
  query.set('page', String(params.page ?? 1))
  query.set('pageSize', String(params.pageSize ?? 10))
  if (params.keyword) query.set('keyword', params.keyword)
  const qs = query.toString()
  const res = await requestApi<AdminMemberListResult>(`/admin/members?${qs}`)
  return {
    items: res.items.map(mapMember),
    total: res.total,
  }
}

export async function fetchMemberDetail(memberId: string): Promise<AdminMemberRow> {
  return mapMember(await requestApi<AdminMemberRow>(`/admin/members/${encodeURIComponent(memberId)}`))
}

export type MemberStatusCode = 'active' | 'frozen'

export interface CreateMemberPayload {
  account: string
  password: string
  status: MemberStatusCode
}

export interface UpdateMemberPayload {
  password?: string
  status: MemberStatusCode
}

export async function createMember(payload: CreateMemberPayload): Promise<AdminMemberRow> {
  return mapMember(await requestApi<AdminMemberRow>('/admin/members', { method: 'POST', body: payload }))
}

export async function updateMember(
  memberId: string,
  payload: UpdateMemberPayload,
): Promise<AdminMemberRow> {
  return mapMember(
    await requestApi<AdminMemberRow>(`/admin/members/${encodeURIComponent(memberId)}`, {
      method: 'PUT',
      body: payload,
    }),
  )
}

export interface ClearGuajiAuthResult {
  pausedSchemes: number
  clearedAuths: number
  message?: string
}

export async function clearMemberGuajiAuth(memberId: string): Promise<ClearGuajiAuthResult> {
  return requestApi<ClearGuajiAuthResult>(
    `/admin/members/${encodeURIComponent(memberId)}/clear-guaji-auth`,
    { method: 'POST' },
  )
}

export interface FetchMemberFundRecordsParams {
  dateFrom: string
  dateTo: string
  flowType?: FundFlowType
  currency?: FundCurrency
  page?: number
  pageSize?: number
}

export async function fetchMemberFundRecords(
  memberId: string,
  params: FetchMemberFundRecordsParams,
): Promise<MemberFundRecordsResult> {
  const query = new URLSearchParams()
  query.set('dateFrom', params.dateFrom)
  query.set('dateTo', params.dateTo)
  if (params.flowType && params.flowType !== 'all') query.set('flowType', params.flowType)
  if (params.currency && params.currency !== 'all') query.set('currency', params.currency)
  query.set('page', String(params.page ?? 1))
  query.set('pageSize', String(params.pageSize ?? 10))
  return requestApi<MemberFundRecordsResult>(
    `/admin/members/${encodeURIComponent(memberId)}/fund-records?${query.toString()}`,
  )
}
