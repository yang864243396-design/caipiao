import { requestApi } from '@/api/client'
import { ensureClientSession } from '@/api/auth'

/** 跟单大厅方案收藏（内置计画选择来源） */
export interface SchemeFavoriteRow {
  snapshotId: string
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  playMethod: string
  favoredAt: string
}

export async function fetchSchemeFavorites(): Promise<SchemeFavoriteRow[]> {
  await ensureClientSession()
  const res = await requestApi<{ items: SchemeFavoriteRow[] }>('/client/schemes/favorites')
  return res.items ?? []
}

export async function addSchemeFavorite(snapshotId: string): Promise<void> {
  await ensureClientSession()
  await requestApi<{ ok: boolean }>('/client/schemes/favorites', {
    method: 'POST',
    body: { snapshotId },
  })
}

export async function removeSchemeFavorite(snapshotId: string): Promise<void> {
  await ensureClientSession()
  await requestApi<{ ok: boolean }>(`/client/schemes/favorites/${encodeURIComponent(snapshotId)}`, {
    method: 'DELETE',
  })
}
