export type WsFrameType = 'system' | 'command' | 'event' | 'error'

export interface WsEnvelope<T = unknown> {
  type: WsFrameType
  name: string
  topic?: string
  eventId?: string
  ts: string
  payload?: T
}

export interface WsMaintenanceChangedPayload {
  enabled: boolean
  title?: string
  message?: string
  popupAnnouncementId?: string
  popupAnnouncement?: {
    id: string
    title: string
    bodyHtml: string
  } | null
}

export interface WsSchemeInstanceUpdatedPayload {
  instanceId: string
  runMode: 'real' | 'sim'
  status: 'running' | 'paused' | 'stopped' | 'pending'
  reason?: string
  hint?: 'refresh_running_list' | 'refresh_bet_records'
}

export interface WsCloudSchemeSnapshotItem {
  id: string
  definitionId?: string
  lotteryCode?: string
  lotteryName?: string
  lotteryLabel?: string
  schemeName: string
  status: 'pending' | 'running' | 'paused' | 'soft_stopped'
  statusReason?: string
  statusLabel: string
  turnover: number
  countdownSec: number
  countdownEndTime?: string
  countdownCloseAt?: string
  countdownPeriod?: string
  countdownWindowSec?: number
  countdownLabel?: string
  pnl: number
  runTimeSec: number
  lookbackPnl: number
  sessionPnl: number
  multiplier: number
  simBet: boolean
  schemeCurrency?: string
  runTypeId?: string
  runTypeLabel?: string
  updatedAt: string
}

export interface WsCloudCenterChannelStats {
  totalTurnover: number
  totalSessionPnl: number
  runningSessionPnl: number
}

export interface WsCloudCenterStats {
  formal: WsCloudCenterChannelStats
  sim: WsCloudCenterChannelStats
  simQuota: {
    todayStarts: number
    todayStartsLimit: number
    running: number
    runningLimit: number
  }
}

export interface WsSchemeInstancesSnapshotPayload {
  schemaVersion: 1
  generatedAt: string
  items: WsCloudSchemeSnapshotItem[]
  removedIds: string[]
}

export interface WsCloudStatsSnapshotPayload {
  schemaVersion: 1
  generatedAt: string
  stats: WsCloudCenterStats
}

export interface WsWalletUpdatedPayload {
  available: number
  frozen: number
  currency: string
  reason?: string
}

export interface WsWithdrawQueueChangedPayload {
  orderNo: string
  status: string
  action: string
  hint?: string
}

export interface WsAdminSchemeMonitorPayload {
  instanceId: string
  status: string
  action: string
  hint?: string
}

export interface WsDashboardKpiChangedPayload {
  metric: 'todayRecharge' | 'todayWithdraw' | 'pendingWithdrawCount' | string
  orderNo?: string
  amount?: number
  action: string
  hint?: string
}

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
  clientCloudStats: 'client.cloud.stats',
  clientWallet: 'client.wallet',
  publicDraw: (lotteryCode: string) => `public.draw:${lotteryCode}`,
  adminWithdrawQueue: 'admin.withdraw.queue',
  adminSchemeMonitor: 'admin.scheme.monitor',
  adminDashboardKpi: 'admin.dashboard.kpi',
} as const

export const WS_EVENTS = {
  maintenanceChanged: 'public.maintenance.changed',
  schemeInstanceUpdated: 'client.scheme.instance.updated',
  schemeInstancesSnapshot: 'client.scheme.instances.snapshot',
  cloudStatsSnapshot: 'client.cloud.stats.snapshot',
  walletUpdated: 'client.wallet.updated',
  withdrawQueueChanged: 'admin.withdraw.queue.changed',
  schemeMonitorChanged: 'admin.scheme.monitor.changed',
  dashboardKpiChanged: 'admin.dashboard.kpi.changed',
  drawResult: 'public.draw.result',
} as const
