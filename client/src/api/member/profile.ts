import { requestApi } from '@/api/client'
import { ensureClientSession } from '@/api/auth'

/** 会员资料；余额一律来自第三方 fetchGuajiBalance，平台不维护自有金额 */
export interface MemberProfile {
  memberId: number
  account: string
  displayName: string
  currency: string
}

export async function fetchMemberProfile(): Promise<MemberProfile> {
  await ensureClientSession()
  const data = await requestApi<MemberProfile & { member_id?: number }>('/client/member/profile')
  return {
    ...data,
    memberId: Number(data.memberId ?? data.member_id ?? 0),
  }
}
