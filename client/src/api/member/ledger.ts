import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface WalletLedgerItem {

  time: string

  type: string

  typeCode: string

  orderId: string

  delta: number

  balanceAfter: number

  ledgerNo: string

}



export interface WalletLedgerResult {

  items: WalletLedgerItem[]

  nextCursor?: string

}



export interface WalletLedgerQuery {

  dateFrom?: string

  dateTo?: string

  type?: string

  orderNo?: string

  cursor?: string

  limit?: number


}



export async function fetchWalletLedger(query: WalletLedgerQuery = {}): Promise<WalletLedgerResult> {

  await ensureClientSession()

  return requestApi<WalletLedgerResult>('/client/orders/ledger', {

    query: {

      dateFrom: query.dateFrom,

      dateTo: query.dateTo,

      type: query.type,

      orderNo: query.orderNo,

      cursor: query.cursor,

      limit: query.limit,


    },

  })

}



/** 表格展示行 */

export function toLedgerDisplayRow(item: WalletLedgerItem): {

  time: string

  type: string

  orderId: string

  delta: string

  balance: string

} {

  const d = new Date(item.time)

  const time = Number.isNaN(d.getTime())

    ? item.time

    : d.toLocaleString('zh-CN', { hour12: false })

  const abs = Math.abs(item.delta).toLocaleString('zh-CN', {

    minimumFractionDigits: 2,

    maximumFractionDigits: 2,

  })

  const delta = item.delta >= 0 ? `+${abs}` : `-${abs}`

  const balance = item.balanceAfter.toLocaleString('zh-CN', {

    minimumFractionDigits: 2,

    maximumFractionDigits: 2,

  })

  return { time, type: item.type, orderId: item.orderId, delta, balance }

}

