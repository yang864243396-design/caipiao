import { requestApi } from './client'

export interface SchemeBettingSummary {
  pending: number
  leased: number
  sentUnknown: number
  externalUnknown: number
  acceptedWrongPeriod: number
  expired: number
  deadlineRisk: number
  blockedRequiresRearm: number
  runningEventOwned: number
  apiDue: number
  acceptedUnfinalized: number
  activeStrategyLeases: number
  activeDispatcherLeases: number
  activeDrawLeases: number
  currentGlobalDispatches: number
  drawToStrategyP99Ms: number
  strategyToAcceptedP99Ms: number
  safeDeadlineCompletionRate: number
  providerPeriodConsistencyRate: number
  oldestPendingAgeSeconds: number
  modes: Array<'shadow' | 'gray' | 'production'>
  measuredAt: string
}

export interface CorePartitionStatus {
  phase: 'mirroring' | 'validated' | 'cutover' | 'rollback_ready'
  forwardSync: boolean
  reverseSync: boolean
  restartRequired: boolean
  lastValidation: {
    valid?: boolean
    betOrders?: Record<string, number>
    cloudBetRecords?: Record<string, number>
    walletLedger?: Record<string, number>
  }
  lastValidatedAt?: string
  cutoverAt?: string
  rollbackAt?: string
  activeTablesPartitioned: {
    betOrders: boolean
    cloudBetRecords: boolean
    walletLedger: boolean
  }
  partitionCounts: {
    betOrders: number
    cloudBetRecords: number
    walletLedger: number
  }
}

export interface ResolveSchemeBettingUnknownInput {
  reason: string
  outcome: 'accepted' | 'rejected'
  evidence: string
  providerOrderId?: string
  acceptedPeriod?: string
  providerAmount?: number
  providerAccountId?: number
  providerCurrency?: string
}

export async function fetchSchemeBettingSummary(): Promise<SchemeBettingSummary> {
  return requestApi<SchemeBettingSummary>('/admin/diagnostics/scheme-betting/summary')
}

export async function fetchCorePartitionStatus(): Promise<CorePartitionStatus> {
  return requestApi<CorePartitionStatus>('/admin/diagnostics/core-partition')
}

export async function enableEventSchemeBetting(schemeId: string, reason: string): Promise<void> {
  await requestApi(`/admin/diagnostics/scheme-betting/${encodeURIComponent(schemeId)}/enable`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export async function rearmSchemeBetting(schemeId: string, reason: string): Promise<void> {
  await requestApi(`/admin/diagnostics/scheme-betting/${encodeURIComponent(schemeId)}/rearm`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export async function cancelSchemeBettingOutbox(outboxId: number, reason: string): Promise<void> {
  await requestApi(`/admin/diagnostics/scheme-betting/outbox/${outboxId}/cancel`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export async function resolveSchemeBettingUnknown(
  outboxId: number,
  input: ResolveSchemeBettingUnknownInput,
): Promise<void> {
  await requestApi(`/admin/diagnostics/scheme-betting/outbox/${outboxId}/resolve`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
