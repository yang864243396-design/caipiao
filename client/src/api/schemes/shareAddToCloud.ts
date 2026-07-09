import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface SchemeDefinitionDto {

  id: string

  kind: 'custom' | 'contrary' | 'follow'

  schemeName: string

  lotteryCode: string

  lotteryLabel?: string

  shareStatusLocked: 'private' | 'public'

  config?: Record<string, unknown>

  hasInstance: boolean

  createdAt: string

  updatedAt: string

}



export interface SchemeInstanceDto {

  id: string

  definitionId: string

  kind: 'custom' | 'contrary' | 'follow'

  schemeName: string

  lotteryCode: string

  lotteryLabel?: string

  status: 'pending' | 'running' | 'paused' | 'soft_stopped'

  statusLabel: string

  runMode: 'real' | 'sim'

  turnover: number

  pnl: number

  runTimeSec: number

  lookbackPnl: number

  sessionPnl: number

  multiplier: number

  countdownSec: number

  simBet: boolean

  createdAt: string

  updatedAt: string

}



export interface ShareAddToCloudResult {

  definition: SchemeDefinitionDto

  instance: SchemeInstanceDto

}



export interface ShareAddToCloudInput {
  betMultiplier?: Record<string, unknown>
}

export async function shareAddToCloud(
  snapshotId: string,
  input: ShareAddToCloudInput = {},
): Promise<ShareAddToCloudResult> {
  await ensureClientSession()
  return requestApi<ShareAddToCloudResult>(
    `/client/schemes/share/${encodeURIComponent(snapshotId)}/add-to-cloud`,
    { method: 'POST', body: input },
  )
}



export interface ShareFollowBetInput {
  lotteryCode?: string
  playMethod?: string
  playTemplate?: string
  typeId?: string
  subId?: string
}



export async function shareFollowBet(

  snapshotId: string,

  input: ShareFollowBetInput = {},

): Promise<ShareAddToCloudResult> {

  await ensureClientSession()

  return requestApi<ShareAddToCloudResult>(

    `/client/schemes/share/${encodeURIComponent(snapshotId)}/follow-bet`,

    { method: 'POST', body: input },

  )

}



export interface ContraryBetInput {
  lotteryCode: string
  planInverseNumbers: string
  playMethod?: string
  playTemplate?: string
  typeId?: string
  subId?: string
}



export async function contraryBet(input: ContraryBetInput): Promise<ShareAddToCloudResult> {

  await ensureClientSession()

  return requestApi<ShareAddToCloudResult>('/client/schemes/contrary/bet', {

    method: 'POST',

    body: input,

  })

}

