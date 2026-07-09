/**
 * WebSocket 帧与事件类型（二期）
 * 设计说明见 backend/docs/websocket.md
 */

export type WsFrameType = 'system' | 'command' | 'event' | 'error'

export interface WsEnvelope<T = unknown> {
  type: WsFrameType
  name: string
  topic?: string
  eventId?: string
  ts: string
  payload?: T
}

/** Client → Server */
export type WsClientCommand =
  | { name: 'auth'; payload: { accessToken: string } }
  | { name: 'subscribe'; payload: { topics: string[] } }
  | { name: 'unsubscribe'; payload: { topics: string[] } }
  | { name: 'ping'; payload?: Record<string, never> }

/** 维护开关变更 */
export interface WsMaintenanceChangedPayload {
  enabled: boolean
  title?: string
  message?: string
  popupAnnouncementId?: string
}

/** 云端方案实例 */
export interface WsSchemeInstanceUpdatedPayload {
  instanceId: string
  runMode: 'real' | 'sim'
  simBet?: boolean
  status: 'running' | 'paused' | 'stopped'
  reason?: 'insufficient_funds' | 'admin_force_stop' | 'lookback_reset' | 'user_action'
  hint?: 'refresh_running_list' | 'refresh_bet_records'
}

/** 钱包余额 */
export interface WsWalletUpdatedPayload {
  available: number
  frozen: number
  currency: string
  reason?: 'bet_settle' | 'withdraw' | 'recharge' | 'scheme_bet'
}

/** Admin 提现队列变更 */
export interface WsWithdrawQueueChangedPayload {
  orderNo: string
  status: string
  action: string
  hint?: string
}

/** Admin 方案监控变更 */
export interface WsAdminSchemeMonitorPayload {
  instanceId: string
  status: string
  action: string
  hint?: string
}

/** Admin 仪表盘 KPI 变更 */
export interface WsDashboardKpiChangedPayload {
  metric: 'todayRecharge' | 'todayWithdraw' | 'pendingWithdrawCount' | string
  orderNo?: string
  amount?: number
  action: string
  hint?: string
}

/** 彩种开奖结果（public） */
export interface WsDrawResultPayload {
  lotteryCode: string
  issueNo: string
  periodShort?: string
  balls: string[]
  sumValue: number
  drawnAt: string
  hint?: string
}

export const WS_TOPICS = {
  publicMaintenance: 'public.maintenance',
  clientSchemeInstance: 'client.scheme.instance',
  clientWallet: 'client.wallet',
  clientCloudBets: 'client.cloud.bets',
  publicDraw: (lotteryCode: string) => `public.draw:${lotteryCode}`,
  adminWithdrawQueue: 'admin.withdraw.queue',
  adminSchemeMonitor: 'admin.scheme.monitor',
  adminDashboardKpi: 'admin.dashboard.kpi',
} as const

export const WS_EVENTS = {
  maintenanceChanged: 'public.maintenance.changed',
  schemeInstanceUpdated: 'client.scheme.instance.updated',
  walletUpdated: 'client.wallet.updated',
  withdrawQueueChanged: 'admin.withdraw.queue.changed',
  schemeMonitorChanged: 'admin.scheme.monitor.changed',
  dashboardKpiChanged: 'admin.dashboard.kpi.changed',
  drawResult: 'public.draw.result',
} as const
