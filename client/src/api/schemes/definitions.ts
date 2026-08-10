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
  /**
   * 正投号码。
   * - 定位胆/文字玩法：单值，多号用逗号（如 1,3,5）
   * - 前三直选复式等：换行分位（万\n千\n百），每位内可再逗号多号
   */
  pos: string
  /** 反投号码（格式同 pos） */
  neg: string
}

export interface SchemeTriggerBet {
  rows: SchemeTriggerRow[]
  /** always_pos 一直正投 / always_neg 一直反投 / alt_pos_first 前正后反 / alt_neg_first 前反后正 */
  mode: 'always_pos' | 'always_neg' | 'alt_pos_first' | 'alt_neg_first'
  /**
   * 投注选位（可多选，0=万/冠军 …）。
   * 任选非直选复式：≥k ≤5（任二至少 2）；下注内容带这些位名前缀。
   * 定位胆等：投注位（可多选）。
   */
  positionIdxs?: number[]
  /** @deprecated 兼容旧单值；新配置请用 positionIdxs */
  positionIdx?: number
  /**
   * 任选开奖选位（单选，0=万 … 4=个）：上期该位球号查开出映射。
   * 缺省时兼容旧配置：取 positionIdxs[0] 或万位。
   */
  openPositionIdx?: number
}

export type SchemeRotateStrategy = 'every' | 'keep' | 'after_hit' | 'after_miss'

export type SchemeHotColdPickType = 'hot' | 'cold'

export interface SchemeHotColdWarm {
  /** 统计总期数 */
  totalPeriods: number
  /**
   * 权威配置：每位一行名次（0=最热）。运行时按近 N 期重排后取「当前排在这些名次上的号码」。
   * 热/冷/全/清只是编辑态快捷勾选名次，不落库；点格子切换的也是名次。
   * 预览与编辑回显均以 ranks 为准。
   */
  ranks?: number[][]
  /**
   * @deprecated 旧配置兼容；新保存不再写入。无 ranks 时非空行表示该位启用，可反查为名次。
   */
  pool?: string[]
  /** every 每期换 / keep 不换号 / after_hit 中后换 / after_miss 挂后换 */
  strategy?: SchemeRotateStrategy
  /**
   * @deprecated 旧配置兼容（热/冷整区）；新保存不再写入。有 ranks 时忽略。
   */
  pickTypes?: SchemeHotColdPickType[]
  /** @deprecated 已废弃，忽略 */
  faultCount?: number
  /** @deprecated 已废弃，忽略 */
  pickCount?: number
  /**
   * @deprecated 兼容旧配置；优先读 strategy。true≈after_hit，false≈keep
   */
  winRotate?: boolean
  /**
   * 任选非直选复式：投注选位（万千百十个，0=万…4=个），至少 k、最多 5。
   * 出号内容带选位前缀，注数 × C(选位数,k)。
   */
  positionIdxs?: number[]
  /**
   * 任选·直选单式 / 任选·混合组选冷热：开奖选位恰好 k 个（0=万…4=个）。
   * 下方频次与 ranks 按这些绝对位统计；出票时与 positionIdxs 组合。
   * 任选·组选12 冷热按 positionIdxs 合并计频，不再使用本字段
   * （ranks[0]=二重号名次、ranks[1]=单号名次）。
   */
  openPositionIdxs?: number[]
}

export interface SchemeRandomDraw {
  /** 每个位置随机号码数量（1-10） */
  counts: number[]
  /** every 每期换 / keep 不换号 / after_hit 中后换 / after_miss 挂后换 */
  strategy: 'every' | 'keep' | 'after_hit' | 'after_miss'
  /**
   * 任选非直选复式：万千百十个选位下标（0=万…4=个），至少 k、最多 5。
   * 出号内容带选位前缀，注数 × C(选位数,k)。
   */
  positionIdxs?: number[]
}

export interface UpdateSchemeInput {

  /** 方案名称（同步定义与云端实例展示名） */
  schemeName?: string

  /** false=正式盘，true=模拟盘 */
  simBet?: boolean

  schemeFunds?: string

  /** 方案币种：USDT / TRX / CNY；未填历史方案默认 USDT */
  schemeCurrency?: string

  /** 方案倍数系数（正整数，最小 1） */
  multCoeff?: string

  startTime?: string

  endTime?: string

  /** 方案内容分组；固定取码仅用 [0]，每期复投 */
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

  /** 冷热出号配置 */
  hotColdWarm?: SchemeHotColdWarm

  /** 随机出号配置 */
  randomDraw?: SchemeRandomDraw

  /** 内置计画：选择收藏快照（服务端物化复制配置） */
  builtinPlan?: { snapshotId: string }

}



function requireDefinitionId(definitionId: string): string {
  const id = String(definitionId ?? '').trim()
  // 空 id 会拼成 /client/schemes/ → Go mux 404 page not found
  if (!id) throw new Error('缺少方案 ID')
  return id
}

export async function getSchemeDefinition(definitionId: string): Promise<SchemeDefinitionDto> {

  const id = requireDefinitionId(definitionId)

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>(

    `/client/schemes/${encodeURIComponent(id)}`,

  )

}



export async function updateSchemeDefinition(

  definitionId: string,

  input: UpdateSchemeInput,

  opts?: { throttle?: boolean },

): Promise<SchemeDefinitionDto> {

  const id = requireDefinitionId(definitionId)

  await ensureClientSession()

  return requestApi<SchemeDefinitionDto>(

    `/client/schemes/${encodeURIComponent(id)}`,

    { method: 'PATCH', body: input, throttle: opts?.throttle },

  )

}



export async function deleteSchemeDefinition(definitionId: string): Promise<void> {

  const id = requireDefinitionId(definitionId)

  await ensureClientSession()

  await requestApi<Record<string, never>>(

    `/client/schemes/${encodeURIComponent(id)}`,

    { method: 'DELETE' },

  )

}



export interface HotColdWarmTiersInput {
  lotteryCode: string
  playTypeId?: string
  subPlayId?: string
  playTemplate?: string
  betMode?: string
  catalogSubId?: string
  playMethodLabel?: string
  numberPoolMin?: number
  numberPoolMax?: number
  segmentLen?: number
  periods?: number
  /** 任选投注选位（0=万…4=个）；任选和值/尾数按此计频 */
  positionIdxs?: number[]
}

export interface HotColdWarmTiersResult {
  mode: string
  universe: string[] | null
  hot: string[] | null
  warm: string[] | null
  cold: string[] | null
  /** 各选项最近 N 期命中次数 */
  counts?: Record<string, number> | null
  counted: number
}

/** 冷热属性家族分档：按最近 N 期选项命中频次返回热/冷（服务端复用权威判定；warm 恒为空）。 */
export async function fetchHotColdWarmTiers(
  input: HotColdWarmTiersInput,
): Promise<HotColdWarmTiersResult> {

  await ensureClientSession()

  return requestApi<HotColdWarmTiersResult>('/client/schemes/hot-cold-warm/tiers', {

    method: 'POST',

    body: input,

  })

}

