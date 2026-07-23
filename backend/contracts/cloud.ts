/**
 * 云端中心 · 投注记录 / 运行方案（与 openapi paths 同步）
 */

import type { MoneySummary, Paginated, RunMode } from './common'

/** GET /client/cloud/bet-records — 方案分组 */
export interface BetRecordSchemeGroup {
  schemeId: string
  schemeName: string
  totalBet: number
  dayPnl: number
  winRate: number
}

export interface BetRecordGroupsData {
  mode: RunMode
  /** 统计窗口，默认 3 */
  days: number
  /** UTC+8 自然日起（含） */
  dateFrom?: string
  /** UTC+8 自然日止（含） */
  dateTo?: string
  summary: MoneySummary
  groups: Paginated<BetRecordSchemeGroup>
}

export interface BetRecordGroupsQuery {
  mode?: RunMode
  days?: number
  dateFrom?: string
  dateTo?: string
  lotteryCode?: string
  cursor?: string
  limit?: number
}

/** GET /client/cloud/bet-records/{schemeId} — 方案明细 */
export type BetRecordStatus = 'hit' | 'miss'

export interface BetRecordItem {
  id: string
  period: string
  /** 第三方接单返回的 periods；仅真实接单成功后有值 */
  periods?: string
  playType: string
  multiplier: string
  round: string
  /** 投注金额（元） */
  amount: number
  /** 盈亏（元，负数为亏） */
  pnl: number
  status: BetRecordStatus
}

export interface BetRecordDetailData {
  schemeId: string
  schemeName: string
  mode: RunMode
  days: number
  dateFrom?: string
  dateTo?: string
  summary: MoneySummary
  records: Paginated<BetRecordItem>
}

export interface BetRecordDetailQuery {
  mode?: RunMode
  days?: number
  cursor?: string
  limit?: number
}

/** 云端实例状态（与 schemes.md §6 一致） */
export type CloudSchemeStatus =
  | 'pending'
  | 'running'
  | 'paused'
  | 'soft_stopped'

export interface CloudRunningScheme {
  id: string
  lotteryCode: string
  lotteryName: string
  schemeName: string
  status: CloudSchemeStatus
  statusLabel: string
  statusReason?: string
  turnover: number
  countdownSec: number
  /** 第三方 periods 原始 end_time（UTC 墙钟） */
  countdownEndTime?: string
  /** RFC3339 封盘时刻（兼容旧客户端） */
  countdownCloseAt?: string
  pnl: number
  runTimeSec: number
  lookbackPnl: number
  /** 本次运行累计盈亏（从 pending 开启时归零） */
  sessionPnl: number
  multiplier: number
  simBet: boolean
  /** 方案币种：USDT / TRX / CNY；缺省 USDT */
  schemeCurrency?: string
}

export interface CloudLookbackSettings {
  runModes: RunMode[]
  judgment: 'individual' | 'overall' | ''
  singleProfitThreshold: number
  singleLossThreshold: number
  overallProfitThreshold: number
  overallLossThreshold: number
  schemeWinsMin: number
  schemeWinsMax: number
  periodProfit: number
  periodLoss: number
}

export interface CloudCenterChannelStats {
  totalTurnover: number
  /** 顶部「总盈亏」：该通道全部实例 session_pnl（本次盈亏）之和 */
  totalSessionPnl: number
  /** 运行中盈亏：running 实例 session_pnl 之和 */
  runningSessionPnl: number
}

export interface CloudCenterStats {
  formal: CloudCenterChannelStats
  sim: CloudCenterChannelStats
}
