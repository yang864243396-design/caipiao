import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import type { ClientSchemeKind } from '@/utils/schemeKind'



export type { ClientSchemeKind }



export interface SchemeDefinitionDto {

  id: string

  kind: ClientSchemeKind

  schemeName: string

  lotteryCode: string

  lotteryLabel?: string

  shareStatusLocked: 'private' | 'public'

  config?: Record<string, unknown>

  hasInstance: boolean

  createdAt: string

  updatedAt: string

}



export interface SchemeDefinitionListResult {

  items: SchemeDefinitionDto[]

}



export async function fetchSchemeDefinitions(

  kind?: ClientSchemeKind,

): Promise<SchemeDefinitionListResult> {

  await ensureClientSession()

  return requestApi<SchemeDefinitionListResult>('/client/schemes', {

    query: { kind },

  })

}



export interface CreateSchemeInput {

  kind: ClientSchemeKind

  schemeName: string

  lotteryCode: string

  runTypeId: string

  playTypeId: string

  subPlayId: string

}



export async function createScheme(input: CreateSchemeInput): Promise<SchemeDefinitionDto> {

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>('/client/schemes', {

    method: 'POST',

    body: input,

  })

}



export interface CheckSchemeNameResult {

  available: boolean

  /** 名称已被占用时的方案定义 ID */
  existingDefinitionId?: string

  /** 占用名称的方案是否已有云端实例 */
  existingHasInstance?: boolean

}



export async function checkSchemeNameAvailable(schemeName: string): Promise<CheckSchemeNameResult> {

  await ensureClientSession()

  return requestApi<CheckSchemeNameResult>('/client/schemes/check-name', {

    query: { name: schemeName.trim() },

  })

}



export interface SchemeJushuRow {
  /** 局数（1-based 唯一序号） */
  ju: number
  /** 投注号码 */
  content: string
  /** 中后跳转至第几局 */
  afterHit: number
  /** 挂后跳转至第几局 */
  afterMiss: number
}

export interface SchemeTriggerRow {
  enabled: boolean
  /** 开出号码（单值：0-9 或 龙/虎/和） */
  open: string
  /** 正投号码（单值） */
  pos: string
  /** 反投号码（单值） */
  neg: string
}

export interface SchemeTriggerBet {
  rows: SchemeTriggerRow[]
  /** always_pos 一直正投 / always_neg 一直反投 / alt_pos_first 前正后反 / alt_neg_first 前反后正 */
  mode: 'always_pos' | 'always_neg' | 'alt_pos_first' | 'alt_neg_first'
}

export interface SchemeHotColdWarm {
  /** 统计总期数 */
  totalPeriods: number
  /** 选号池（每位一行，逗号分隔） */
  pool: string[]
  /** 中奖轮换开关 */
  winRotate: boolean
}

export interface SchemeRandomDraw {
  /** 每个位置随机号码数量（1-10） */
  counts: number[]
  /** every 每期换 / keep 不换号 / after_hit 中后换 / after_miss 挂后换 */
  strategy: 'every' | 'keep' | 'after_hit' | 'after_miss'
}

export interface UpdateSchemeInput {

  /** false=正式盘，true=模拟盘 */
  simBet?: boolean

  schemeFunds?: string

  startTime?: string

  endTime?: string

  schemeGroups?: string[]

  stopLoss?: string

  takeProfit?: string

  betUnit?: string

  /** 玩法 betMode（单式/复式等），由 catalogFieldsFromPlayConfig 写入 */
  betMode?: string

  playTemplate?: string

  typeId?: string

  subId?: string

  betMultiplier?: Record<string, unknown>

  rounds?: unknown[]

  /** 高级定码轮换局数列表 */
  jushuList?: SchemeJushuRow[]

  /** 高级开某投某映射配置 */
  triggerBet?: SchemeTriggerBet

  /** 冷热温出号配置 */
  hotColdWarm?: SchemeHotColdWarm

  /** 随机出号配置 */
  randomDraw?: SchemeRandomDraw

  /** 内置计画：选择收藏快照（服务端物化复制配置） */
  builtinPlan?: { snapshotId: string }

}



export async function getSchemeDefinition(definitionId: string): Promise<SchemeDefinitionDto> {

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>(

    `/client/schemes/${encodeURIComponent(definitionId)}`,

  )

}



export async function updateSchemeDefinition(

  definitionId: string,

  input: UpdateSchemeInput,

): Promise<SchemeDefinitionDto> {

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>(

    `/client/schemes/${encodeURIComponent(definitionId)}`,

    { method: 'PATCH', body: input },

  )

}



export async function deleteSchemeDefinition(definitionId: string): Promise<void> {

  await ensureClientSession()

  await requestApi<Record<string, never>>(

    `/client/schemes/${encodeURIComponent(definitionId)}`,

    { method: 'DELETE' },

  )

}

