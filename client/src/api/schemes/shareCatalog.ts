import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface SchemeDownloadRow {

  schemeId: string

  schemeName: string

  lotteryLabel: string

  playMethod: string

  fundYuan: number

}



export interface SchemeShareSnapshot {

  id: string

  kind: 'custom'

  schemeName: string

  lotteryCode: string

  lotteryLabel?: string

  playMethod?: string

  fundYuan?: number

  config: Record<string, unknown>

  createdAt: string

  updatedAt: string

}



export interface ShareCatalogResult {

  items: SchemeShareSnapshot[]

  page: {

    nextCursor?: string

    hasMore: boolean

  }

}



export interface ShareCatalogQuery {

  keyword?: string

  cursor?: string

  limit?: number

}



export function toDownloadRow(item: SchemeShareSnapshot): SchemeDownloadRow {

  return {

    schemeId: item.id,

    schemeName: item.schemeName,

    lotteryLabel: item.lotteryLabel ?? item.lotteryCode,

    playMethod: item.playMethod ?? '—',

    fundYuan: item.fundYuan ?? 0,

  }

}



export async function fetchShareCatalog(query: ShareCatalogQuery = {}): Promise<ShareCatalogResult> {

  await ensureClientSession()

  return requestApi<ShareCatalogResult>('/client/schemes/share-catalog', {

    query: {

      keyword: query.keyword,

      cursor: query.cursor,

      limit: query.limit,

    },

  })

}



export async function fetchShareCatalogRows(keyword = ''): Promise<SchemeDownloadRow[]> {

  const result = await fetchShareCatalog({ keyword: keyword || undefined })

  return result.items.map(toDownloadRow)

}

