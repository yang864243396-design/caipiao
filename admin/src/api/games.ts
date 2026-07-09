import { requestApi } from './client'

export interface LotteryDrawRow {
  periodShort: string
  time: string
  balls: string[]
  sum: number
}

export interface LotteryDrawsResult {
  items: LotteryDrawRow[]
  page: { nextCursor?: string; hasMore: boolean }
}

export async function fetchLotteryDraws(
  lotteryCode: string,
  cursor?: string,
  limit = 50,
): Promise<LotteryDrawsResult> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (cursor) params.set('cursor', cursor)
  return requestApi<LotteryDrawsResult>(
    `/admin/games/lottery-catalog/${encodeURIComponent(lotteryCode)}/draws?${params.toString()}`,
  )
}
