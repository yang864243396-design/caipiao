import { requestApi } from './client'

import type { LotteryCatalogRow } from '@/types/lottery'

export async function fetchLotteryCatalog(): Promise<LotteryCatalogRow[]> {
  const res = await requestApi<{ items: LotteryCatalogRow[] }>('/admin/games/lottery-catalog')
  return res.items
}

export type PatchLotteryCatalogInput = {
  displayName?: string
  outboundLotteryCode?: string
  sortOrder?: number
  saleStatus?: 'on_sale' | 'maintenance'
  enterMaintenance?: boolean
}

export async function patchLotteryCatalog(
  code: string,
  body: PatchLotteryCatalogInput,
): Promise<LotteryCatalogRow> {
  return requestApi<LotteryCatalogRow>(`/admin/games/lottery-catalog/${encodeURIComponent(code)}`, {
    method: 'PATCH',
    body,
  })
}

