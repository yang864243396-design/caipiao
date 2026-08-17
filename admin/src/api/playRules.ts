import { requestApi } from './client'

export type PublishedPlayRule = {
  id: number
  templateCode: string
  typeId: string
  subId: string
  lotteryCode: string
  ruleVersion: number
  evaluatorKey: string
  evaluatorVersion: number
  strategyEnabled: boolean
  publishedAt: string
  updatedAt: string
}

export async function fetchPublishedPlayRules(): Promise<PublishedPlayRule[]> {
  const result = await requestApi<{ items: PublishedPlayRule[] }>('/admin/play-rules')
  return result.items
}
