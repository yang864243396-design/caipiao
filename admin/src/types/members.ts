export type FundFlowType = 'all' | 'income' | 'expense'

export type FundCurrency = 'all' | 'USDT' | 'TRX' | 'CNY'

export interface MemberFundRecordRow {
  id: string
  schemeName: string
  currency: string
  amount: number
  time: string
  flowType: string
  flowTypeCode: 'income' | 'expense'
  balanceAfter: number
  ledgerNo: string
}

export interface MemberFundRecordsResult {
  items: MemberFundRecordRow[]
  total: number
  page: number
  pageSize: number
}
