/**
 * 通用 API 契约（与 openapi/components 同步）
 * 实现后端或迁入 shared/ 时勿与 openapi 漂移
 */

/** 统一响应包 */
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

/** 分页（cursor 风格，明细列表用） */
export interface PageMeta {
  nextCursor?: string | null
  hasMore: boolean
}

export interface Paginated<T> {
  items: T[]
  page: PageMeta
}

/** 金额汇总（元） */
export interface MoneySummary {
  /** 总投注额 */
  totalBet: number
  /** 当日盈亏（可正可负） */
  dayPnl: number
  /** 胜率 0–100 */
  winRate: number
}

export type RunMode = 'real' | 'sim'

/** Client / Admin 登录响应 */
export interface AuthTokenPayload {
  accessToken: string
  expiresAt: string
  account: string
  displayName: string
}

/** 业务错误码（节选） */
export const ApiErrorCode = {
  OK: 0,
  UNAUTHORIZED: 40100,
  FORBIDDEN: 40300,
  NOT_FOUND: 40400,
  VALIDATION: 42200,
  INTERNAL: 50000,
  /** 方案域见 SchemeErrorCode（contracts/schemes.ts） */
} as const
