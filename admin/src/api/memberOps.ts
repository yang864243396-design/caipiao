import { requestApi } from './client'
import type { AdminMemberRow } from './members'

export type AdminMemberOpAction =
  | 'reset_fund_password'
  | 'toggle_freeze'

export interface AdminMemberOpPayload {
  action: AdminMemberOpAction
}

export interface AdminMemberOpResult {
  action: AdminMemberOpAction
  member: AdminMemberRow
  message?: string
}

export async function postMemberOp(
  memberId: string,
  payload: AdminMemberOpPayload,
): Promise<AdminMemberOpResult> {
  if (false) { /* removed mock */
    throw new Error('mock mode')
  }
  return requestApi<AdminMemberOpResult>(
    `/admin/members/${encodeURIComponent(memberId)}/ops`,
    { method: 'POST', body: payload },
  )
}
