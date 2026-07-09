import { requestApi } from '@/api/client'

export interface PublicLobbySlot {
  slotKey: string
  title: string
  brief?: string
  sort: number
}

export async function fetchPublicLobbySlots(): Promise<PublicLobbySlot[]> {
  const res = await requestApi<{ items: PublicLobbySlot[] }>('/public/lobby-slots', { auth: false })
  return res.items ?? []
}
