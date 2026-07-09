import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface MemberWallet {

  balance: number

  frozenBalance: number

  availableBalance: number

  currency: string

}



export async function fetchMemberWallet(): Promise<MemberWallet> {

  await ensureClientSession()

  return requestApi<MemberWallet>('/client/member/wallet')

}

