import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import type { SchemeDefinitionDto, SchemeInstanceDto } from '@/api/schemes/shareAddToCloud'

import type { ClientSchemeKind } from '@/utils/schemeKind'



export interface AddToCloudInput {

  shareStatus?: 'private' | 'public'

  /** false=正式盘，true=模拟盘 */
  simBet?: boolean

  schemeFunds?: string

  startTime?: string

  endTime?: string

  schemeGroups?: string[]

  stopLoss?: string

  takeProfit?: string

  betUnit?: string

  betMode?: string

  playTemplate?: string

  typeId?: string

  subId?: string

}



export interface AddToCloudResult {

  definition: SchemeDefinitionDto

  instance: SchemeInstanceDto

  shareSnapshotId?: string

}



export async function addSchemeToCloud(

  definitionId: string,

  input: AddToCloudInput & { kind: ClientSchemeKind; schemeName: string; lotteryCode: string },

): Promise<AddToCloudResult> {

  await ensureClientSession()

  return requestApi<AddToCloudResult>(

    `/client/schemes/${encodeURIComponent(definitionId)}/add-to-cloud`,

    { method: 'POST', body: input },

  )

}



export interface ForkToCloudResult {

  sourceDefinitionId: string

  definition: SchemeDefinitionDto

  instance: SchemeInstanceDto

}



export async function forkSchemeToCloud(

  definitionId: string,

  input: AddToCloudInput & { kind: ClientSchemeKind; schemeName: string; lotteryCode: string },

): Promise<ForkToCloudResult> {

  await ensureClientSession()

  return requestApi<ForkToCloudResult>(

    `/client/schemes/${encodeURIComponent(definitionId)}/fork-and-add-to-cloud`,

    { method: 'POST', body: input },

  )

}

