import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

export interface BetOrderItem {
  time: string
  game: string
  orderId: string
  amount: number
  returnAmount: number
  status: string
}

export interface BetOrdersResult {
  items: BetOrderItem[]
  page: {
    hasMore: boolean
    nextCursor?: string | null
  }
}

export interface BetOrdersQuery {
  dateFrom?: string
  dateTo?: string
  gameCode?: string
  schemeDefinitionId?: string
  orderNo?: string
  cursor?: string
  limit?: number
}

export async function fetchBetOrders(query: BetOrdersQuery = {}): Promise<BetOrdersResult> {
  await ensureClientSession()
  return requestApi<BetOrdersResult>('/client/orders/bets', {
    query: {
      dateFrom: query.dateFrom,
      dateTo: query.dateTo,
      gameCode: query.gameCode,
      schemeDefinitionId: query.schemeDefinitionId,
      orderNo: query.orderNo,
      cursor: query.cursor,
      limit: query.limit,
    },
  })
}

function formatMoney(n: number): string {
  return n.toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

export function toBetDisplayRow(item: BetOrderItem): {
  time: string
  game: string
  orderId: string
  amount: string
  returnAmount: string
  status: string
} {
  return {
    time: item.time,
    game: item.game,
    orderId: item.orderId,
    amount: formatMoney(item.amount),
    returnAmount: formatMoney(item.returnAmount),
    status: item.status,
  }
}
