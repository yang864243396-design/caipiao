import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import type { GameBetPayload } from '@/utils/betPayload'



export type { GameBetPayload }



export interface GameBettingRow {

  time: string

  scheme: string

  numbers: string

  period: string

  draw: string

  win: boolean

}



export interface GameBetRecordDto {

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



export interface GameDetailDto {

  lotteryCode: string

  lotteryLabel: string

  schemeTitle: string

  playMethod: string

  currentIssue: string

  nextIssue: string

  countdownSec: number
  countdownEndTime?: string
  countdownCloseAt?: string
  countdownPeriod?: string
  countdownWindowSec?: number
  countdownLabel?: string
  drawPhase: 'drawing' | 'drawn'

  drawnNumbers: string[]

  planInverseDigits: string

  planInverseBetCount: number

  /** 是否展示「计划反集」Tab：玩法支持且当前有可展示反集号码；false 时隐藏 */
  planContrarySupported?: boolean

  bettingRows: GameBettingRow[]

  betRecords: GameBetRecordDto[]

  planTrendGroupBets: number

  planTrendHistory: GamePlanTrendRow[]

  planTrendChart?: GamePlanTrendChartPoint[]

  schemeBetUnit?: number

  schemeBetMultiplier?: number

  schemeBetUnits?: number

  schemeContraryBetUnits?: number

  schemePickDigits?: string

  estimatedPrize?: number

  contraryEstimatedPrize?: number

}



export interface HistoryDrawDto {

  periodShort: string

  time: string

  balls: string[]

  sum: number

}



export interface GameDrawsResult {

  items: HistoryDrawDto[]

  page: { nextCursor?: string; hasMore: boolean }

}



export interface PlaceGameBetInput {

  issueNo?: string

  amount: number

  multiplier: number

  betMode?: string

  playMethod?: string

  runMode?: 'real' | 'sim'

  betPayload?: GameBetPayload

}



export interface PlaceGameBetResult {

  orderNo: string

  issueNo: string

  amount: number

  status: string

  placedAt: string

}



export async function fetchGameDetail(
  lotteryCode: string,
  query?: {
    schemeName?: string
    playMethod?: string
    snapshotId?: string
    board?: string
    playTypeId?: string
    subPlayId?: string
  },
): Promise<GameDetailDto> {
  await ensureClientSession()
  const q: Record<string, string> = {}
  if (query?.schemeName) q.schemeName = query.schemeName
  if (query?.playMethod) q.playMethod = query.playMethod
  if (query?.snapshotId) q.snapshotId = query.snapshotId
  if (query?.board) q.board = query.board
  if (query?.playTypeId) q.playTypeId = query.playTypeId
  if (query?.subPlayId) q.subPlayId = query.subPlayId
  return requestApi<GameDetailDto>(`/client/games/${encodeURIComponent(lotteryCode)}/detail`, { query: q })
}



export async function fetchGameDraws(

  lotteryCode: string,

  cursor?: string,

  limit = 20,

): Promise<GameDrawsResult> {

  await ensureClientSession()

  return requestApi<GameDrawsResult>(`/client/games/${encodeURIComponent(lotteryCode)}/draws`, {

    query: { limit, ...(cursor ? { cursor } : {}) },

  })

}



export async function placeGameBet(

  lotteryCode: string,

  input: PlaceGameBetInput,

): Promise<PlaceGameBetResult> {

  await ensureClientSession()

  return requestApi<PlaceGameBetResult>(`/client/games/${encodeURIComponent(lotteryCode)}/bets`, {

    method: 'POST',

    body: input,

  })

}

