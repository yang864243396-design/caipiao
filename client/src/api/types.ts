/** 与 backend/contracts/common.ts 同步 */

export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

export type RunMode = 'real' | 'sim'

export interface MoneySummary {
  totalBet: number
  dayPnl: number
  winRate: number
  /** 中奖派奖总额（元） */
  totalPrize?: number
}

export interface PageMeta {
  nextCursor?: string | null
  hasMore: boolean
}

export interface Paginated<T> {
  items: T[]
  page: PageMeta
}

export interface AuthTokenPayload {
  accessToken: string
  expiresAt: string
  account: string
  displayName: string
}

export type CloudSchemeStatus = 'pending' | 'running' | 'paused' | 'soft_stopped'

export interface CloudRunningScheme {
  id: string
  definitionId?: string
  lotteryCode?: string
  lotteryName: string
  lotteryLabel?: string
  schemeName: string
  status: CloudSchemeStatus
  statusReason?: string
  statusLabel: string
  turnover: number
  countdownSec: number
  /** 第三方 periods 原始 end_time（UTC 墙钟） */
  countdownEndTime?: string
  /** RFC3339 封盘时刻（兼容旧客户端） */
  countdownCloseAt?: string
  /** 倒计时对应第三方期号，便于核对 */
  countdownPeriod?: string
  /** 单期投注窗口秒数（start→end），展示倒计时封顶 */
  countdownWindowSec?: number
/** running 且倒计时归零、下期尚未刷新时为「请等待」 */
  countdownLabel?: string
  pnl: number
  runTimeSec: number
  lookbackPnl: number
  /** 本次运行累计盈亏（从 pending 开启时归零） */
  sessionPnl: number
  multiplier: number
  simBet: boolean
  /** 运行类型（仅自创方案）：fixed_rotate / adv_fixed_rotate / adv_trigger_bet / hot_cold_warm / random_draw / builtin_plan / fixed_number */
  runTypeId?: string
  runTypeLabel?: string
}

export interface CloudLookbackSettings {
  applyFormal?: boolean
  applySim?: boolean
  /** @deprecated 使用 applyFormal/applySim */
  runModes: RunMode[]
  judgment: 'individual' | 'overall' | ''
  singleProfitThreshold: number
  singleLossThreshold: number
  overallProfitThreshold: number
  overallLossThreshold: number
  schemeWinsMin: number
  schemeWinsMax: number
  periodProfit: number
  periodLoss: number
}

export const ApiErrorCode = {
  OK: 0,
  UNAUTHORIZED: 40100,
  NOT_FOUND: 40400,
  VALIDATION: 42200,
} as const
