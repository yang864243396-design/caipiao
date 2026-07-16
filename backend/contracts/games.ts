/**
 * 彩种游戏详情 / 开奖 / 手动下注（与 openapi GameDetail / BetPayload 同步）
 */

import type { PageMeta } from './common'

/** 与 schemes/play_api.go BetPayload 对齐 */
export interface BetPayload {
  playMethod?: string
  playTypeId?: 'dingwei' | 'hou4' | 'qian3' | 'zhong3'
  subPlayId?: 'zhixuan_fs' | 'zhixuan_ds' | 'zuxuan_fs'
  /** 选号串：定位/池逗号分隔；直选复式多行换行；直选单式逗号分隔 N 位 token */
  groupContent: string
}

export interface PlaceGameBetRequest {
  issueNo?: string
  amount: number
  multiplier: number
  betMode?: string
  playMethod?: string
  betPayload: BetPayload
}

export interface PlaceGameBetResult {
  orderNo: string
  issueNo: string
  amount: number
  status: 'pending'
  placedAt: string
}

export interface GameBettingRow {
  time: string
  scheme: string
  numbers: string
  period: string
  draw: string
  win: boolean
}

export interface GameBetRecordRow {
  period: string
  playMethod: string
  multiplier: string
  round: string
  amount: string
  profitLoss: number
  status: string
}

export interface GamePlanTrendRow {
  period: string
  win: boolean
}

export interface GamePlanTrendChartPoint {
  period: string
  round: number
  win: boolean
}

export interface GameDetail {
  lotteryCode: string
  lotteryLabel: string
  schemeTitle: string
  playMethod: string
  currentIssue: string
  nextIssue: string
  countdownSec: number
  countdownEndTime?: string
  countdownPeriod?: string
  countdownLabel?: string
  drawPhase: 'drawing' | 'drawn'
  drawnNumbers: string[]
  planInverseDigits: string
  planInverseBetCount: number
  /** 当前玩法是否适用计划反集；false 时前端隐藏「计划反集」Tab */
  planContrarySupported: boolean
  bettingRows: GameBettingRow[]
  betRecords: GameBetRecordRow[]
  planTrendGroupBets: number
  planTrendHistory: GamePlanTrendRow[]
  planTrendChart: GamePlanTrendChartPoint[]
  schemeBetUnit?: number
  schemeBetMultiplier?: number
  schemeBetUnits?: number
  schemeContraryBetUnits?: number
  schemePickDigits?: string
  estimatedPrize?: number
  contraryEstimatedPrize?: number
}

export interface HistoryDrawItem {
  periodShort: string
  time: string
  balls: string[]
  sum: number
}

export interface GameDrawsResult {
  items: HistoryDrawItem[]
  page: PageMeta
}
