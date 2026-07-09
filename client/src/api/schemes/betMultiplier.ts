import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import {

  getSchemeDefinition,

  type SchemeDefinitionDto,

} from '@/api/schemes/definitions'



export interface SchemeRoundRule {

  mult: number

  afterHit: number

  afterMiss: number

}



export interface BetMultiplierPayload {

  kind: '0' | '1' | '2' | '3'

  newbie?: Record<string, unknown>

  oneclick?: Record<string, unknown>

  simple?: Record<string, unknown>

  advanced?: Record<string, unknown>

}



export function isMemberDefinitionId(id: string): boolean {

  return id.startsWith('def-')

}



export { getSchemeDefinition }



export async function saveBetMultiplier(

  definitionId: string,

  payload: BetMultiplierPayload,

): Promise<SchemeDefinitionDto> {

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>(

    `/client/schemes/${encodeURIComponent(definitionId)}/bet-multiplier`,

    { method: 'PUT', body: payload },

  )

}



export async function saveSchemeRounds(

  definitionId: string,

  rounds: SchemeRoundRule[],

): Promise<SchemeDefinitionDto> {

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>(

    `/client/schemes/${encodeURIComponent(definitionId)}/rounds`,

    { method: 'PUT', body: { rounds } },

  )

}

