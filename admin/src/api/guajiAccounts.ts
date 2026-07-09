import { requestApi } from './client'

/** Admin 会员详情 · 第三方授权只读 Tab（guaji 方案 §19.5 / §11.1） */
export interface AdminGuajiAccountRow {
  id: number
  guajiUsername: string
  isActive: boolean
  boundAt: string
  lastSyncAt?: string
  lastTokenError?: string
  lastBetAt?: string
}

export async function fetchMemberGuajiAccounts(memberId: string): Promise<AdminGuajiAccountRow[]> {
  const res = await requestApi<{ items: AdminGuajiAccountRow[] }>(
    `/admin/members/${encodeURIComponent(memberId)}/guaji-accounts`,
  )
  return res.items ?? []
}
