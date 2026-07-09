/**
 * 方案域（与 backend/docs/modules/schemes.md、openapi 同步）
 */

import type { Paginated, RunMode } from './common'

/** 方案类型（对外 code） */
export type SchemeKind = 'custom' | 'contrary' | 'follow'

/** 分享状态（仅自创、首次添加至云端前可选） */
export type SchemeShareStatus = 'private' | 'public'

/** 云端实例状态 */
export type SchemeInstanceStatus = 'pending' | 'running' | 'paused' | 'soft_stopped'

/** 分享池快照 type 恒为自创 */
export type SchemeShareSnapshotKind = 'custom'

/** GET /client/games/{code}/scheme-options */
export interface SchemeOptionItem {
  value: string
  label: string
}

export interface LotterySchemeOptionsData {
  lotteryCode: string
  runTypes: SchemeOptionItem[]
  playTypes: SchemeOptionItem[]
  subPlays: SchemeOptionItem[]
}

/** 方案配置 JSON（实现层 JSONB；此处为关键字段摘要） */
export interface SchemeConfigPayload {
  schemeName: string
  lotteryCode: string
  runTypeId: string
  playTypeId: string
  subPlayId: string
  schemeGroups?: string[]
  startTime?: string
  endTime?: string
  stopLoss?: number | null
  takeProfit?: number | null
  betMultiplier?: Record<string, unknown>
  rounds?: Array<{ mult: number; afterHit: number; afterMiss: number }>
  [key: string]: unknown
}

export interface SchemeDefinition {
  id: string
  kind: SchemeKind
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  shareStatusLocked: SchemeShareStatus
  config: SchemeConfigPayload
  hasInstance: boolean
  createdAt: string
  updatedAt: string
}

export interface SchemeInstance {
  id: string
  definitionId: string
  kind: SchemeKind
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  status: SchemeInstanceStatus
  /** 会员端展示文案 */
  statusLabel: string
  runMode: RunMode
  turnover: number
  pnl: number
  runTimeSec: number
  lookbackPnl: number
  multiplier: number
  countdownSec: number
  simBet: boolean
  createdAt: string
  updatedAt: string
}

export interface SchemeShareSnapshot {
  id: string
  kind: SchemeShareSnapshotKind
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  playMethod: string
  fundYuan: number
  config: SchemeConfigPayload
  createdAt: string
  updatedAt: string
}

/** POST /client/schemes — 创建 */
export interface CreateSchemeRequest {
  kind: SchemeKind
  schemeName: string
  lotteryCode: string
  runTypeId: string
  playTypeId: string
  subPlayId: string
}

/** PATCH /client/schemes/{id} — 不可改锁定五字段 */
export type UpdateSchemeRequest = Partial<
  Omit<
    SchemeConfigPayload,
    'schemeName' | 'lotteryCode' | 'runTypeId' | 'playTypeId' | 'subPlayId'
  >
>

/** POST add-to-cloud（自创首次） */
export interface AddSchemeToCloudRequest {
  /** 仅自创、且无实例时有效 */
  shareStatus?: SchemeShareStatus
}

export interface AddSchemeToCloudData {
  definition: SchemeDefinition
  instance: SchemeInstance
  shareSnapshotId?: string
}

export interface ForkSchemeToCloudData {
  sourceDefinitionId: string
  definition: SchemeDefinition
  instance: SchemeInstance
}

export interface ShareCatalogQuery {
  keyword?: string
  cursor?: string
  limit?: number
}

export interface ShareFollowBetRequest {
  lotteryCode: string
  playMethod?: string
}

export interface ContraryBetRequest {
  lotteryCode: string
  /** planInverse 号码等内容 */
  planInverseNumbers: string
  playMethod?: string
}

export interface ShareFollowActionData {
  definition: SchemeDefinition
  instance: SchemeInstance
}

/** 会员云端全局规则（CloudCenterView 顶栏 + breakPeriodStop） */
export interface CloudGlobalSettings {
  totalStopLoss: number
  totalTakeProfit: number
  planMultiplier: number
  /** 开=维护后手动续投；关=维护结束且未触止损止盈时自动续投 */
  breakPeriodStop: boolean
}

/** Admin 监控行 */
export interface AdminSchemeMonitorRow {
  instanceId: string
  definitionId: string
  memberId: string
  memberName: string
  kind: SchemeKind
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  status: SchemeInstanceStatus
  statusReason?: string
  statusLabel: string
  runMode: RunMode
  createdAt: string
  updatedAt: string
}

export interface AdminSchemeMonitorQuery {
  scope: 'user' | 'share'
  keyword?: string
  kind?: SchemeKind
  status?: SchemeInstanceStatus
  runMode?: RunMode
}

export interface PatchShareSnapshotRequest {
  config: SchemeConfigPayload
  schemeName?: string
  lotteryCode?: string
  lotteryLabel?: string
  playMethod?: string
  fundYuan?: number
}

/** 业务错误码（方案域） */
export const SchemeErrorCode = {
  NAME_DUPLICATE: 42201,
  ADD_CLOUD_TOO_FAST: 42901,
  INSTANCE_EXISTS: 40901,
  DELETE_WHILE_RUNNING: 40902,
  SHARE_NOT_ALLOWED: 42202,
  SNAPSHOT_KIND_IMMUTABLE: 42203,
} as const
