import { requestApi } from '@/api/client'
import type { PlayTreeResponse, PublicLotteryRow } from '@/types/playCatalog'

export type { PlayTreeResponse, PublicLotteryRow }

export async function fetchPublicLotteries(): Promise<PublicLotteryRow[]> {
  const res = await requestApi<{ items: PublicLotteryRow[] }>('/public/lotteries', { auth: false })
  return res.items ?? []
}

export async function fetchPlayTree(lotteryCode: string): Promise<PlayTreeResponse> {
  return requestApi<PlayTreeResponse>(
    `/public/lotteries/${encodeURIComponent(lotteryCode)}/play-tree`,
    { auth: false },
  )
}

export type LotteryRouteStatus = {
  code: string
  exists: boolean
  legacy: boolean
  saleStatus?: 'on_sale' | 'maintenance'
}

export async function fetchLotteryRouteStatus(code: string): Promise<LotteryRouteStatus> {
  return requestApi<LotteryRouteStatus>(
    `/public/lotteries/${encodeURIComponent(code)}/status`,
    { auth: false },
  )
}

export type LotteryFilterOption = {
  code: string
  displayName: string
  saleStatus: 'on_sale' | 'maintenance'
}

export async function fetchMemberLotteryFilterOptions(): Promise<LotteryFilterOption[]> {
  const res = await requestApi<{ items: LotteryFilterOption[] }>('/client/games/lottery-options')
  return (res.items ?? []).map((row) => ({
    code: row.code,
    displayName: row.displayName,
    saleStatus: row.saleStatus,
  }))
}
