import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface ChaseOrderItem {

  time: string

  game: string

  chaseNo: string

  totalIssues: number

  doneIssues: number

  amount: number

  status: string

}



export interface ChaseOrdersResult {

  items: ChaseOrderItem[]

  page: {

    hasMore: boolean

    nextCursor?: string | null

  }

}



export interface ChaseOrdersQuery {

  dateFrom?: string

  dateTo?: string

  gameCode?: string

  cursor?: string

  limit?: number


}



export async function fetchChaseOrders(query: ChaseOrdersQuery = {}): Promise<ChaseOrdersResult> {

  await ensureClientSession()

  return requestApi<ChaseOrdersResult>('/client/orders/chases', {

    query: {

      dateFrom: query.dateFrom,

      dateTo: query.dateTo,

      gameCode: query.gameCode,

      cursor: query.cursor,

      limit: query.limit,


    },

  })

}



export function toChaseDisplayRow(item: ChaseOrderItem): {

  time: string

  game: string

  chaseNo: string

  totalIssues: string

  doneIssues: string

  amount: string

  status: string

} {

  return {

    time: item.time,

    game: item.game,

    chaseNo: item.chaseNo,

    totalIssues: String(item.totalIssues),

    doneIssues: String(item.doneIssues),

    amount: item.amount.toLocaleString('zh-CN', {

      minimumFractionDigits: 2,

      maximumFractionDigits: 2,

    }),

    status: item.status,

  }

}

