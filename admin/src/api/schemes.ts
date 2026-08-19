import { requestApi } from './client'
import type {
  SchemeBetHistoryItem,
  SchemeCustomSettings,
  SchemeInstanceRow,
  SchemeShareSnapshotRow,
} from '@/types/schemes'

export type { SchemeCustomSettings, SchemeInstanceRow, SchemeShareSnapshotRow }

type ApiKind = 'custom' | 'contrary' | 'follow'
type ApiStatus = 'pending' | 'running' | 'paused' | 'soft_stopped'

interface ApiMonitorRow {
  instanceId: string
  definitionId: string
  memberId: string
  memberName: string
  kind: ApiKind
  runTypeId?: string
  runTypeLabel?: string
  playTypeId?: string
  playTypeLabel?: string
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  status: ApiStatus
  statusLabel: string
  simBet: boolean
  totalPnl: number
  betWinRate: number
  currency: string
  createdAt: string
  updatedAt: string
}

interface ApiShareSnapshot {
  id: string
  kind: string
  schemeName: string
  lotteryCode: string
  lotteryLabel?: string
  playMethod?: string
  fundYuan?: number
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

const kindToUi: Record<ApiKind, SchemeInstanceRow['kind']> = {
  custom: '自创',
  contrary: '反买',
  follow: '跟单',
}

function mapKindToApi(kind: string): string | undefined {
  if (!kind) return undefined
  if (kind === '自创') return 'custom'
  if (kind === '反买') return 'contrary'
  if (kind === '跟单') return 'follow'
  return kind
}

function mapStatusToApi(status: string): string | undefined {
  if (!status) return undefined
  if (status === '待开启') return 'pending'
  if (status === '运行中') return 'running'
  if (status === '已暂停') return 'paused'
  if (status === '已封停') return 'soft_stopped'
  return status
}

function mapMonitorRow(row: ApiMonitorRow): SchemeInstanceRow {
  return {
    id: row.instanceId,
    memberId: row.memberId,
    memberName: row.memberName,
    kind: kindToUi[row.kind] ?? '自创',
    lotteryCode: row.lotteryCode,
    lotteryLabel: row.lotteryLabel,
    refId: row.definitionId,
    status: row.statusLabel as SchemeInstanceRow['status'],
    simBet: row.simBet,
    totalPnl: row.totalPnl ?? 0,
    betWinRate: row.betWinRate ?? 0,
    currency: row.currency ?? 'CNY',
    runTypeLabel: row.runTypeLabel ?? '',
    playTypeLabel: row.playTypeLabel ?? '',
    createdAt: row.createdAt,
    updatedAt: row.updatedAt,
    settings: {
      schemeName: row.schemeName,
      lotteryId: row.lotteryCode,
      runTypeId: String(row.runTypeId ?? ''),
      playTypeId: String(row.playTypeId ?? ''),
      subPlayId: '',
    },
  }
}

function mapShareSnapshot(row: ApiShareSnapshot): SchemeShareSnapshotRow {
  const cfg = row.config ?? {}
  return {
    id: row.id,
    kind: '自创',
    schemeName: row.schemeName,
    lotteryCode: row.lotteryCode,
    lotteryLabel: row.lotteryLabel ?? row.lotteryCode,
    playMethod: row.playMethod,
    fundYuan: row.fundYuan,
    config: cfg,
    settings: {
      schemeName: row.schemeName,
      lotteryId: row.lotteryCode,
      runTypeId: String(cfg.runTypeId ?? cfg.runType ?? ''),
      playTypeId: String(cfg.playTypeId ?? cfg.playType ?? cfg.typeId ?? ''),
      subPlayId: String(cfg.subPlayId ?? cfg.subPlay ?? cfg.subId ?? ''),
    },
    publishedAt: row.createdAt,
    updatedAt: row.updatedAt,
  }
}

export type SchemeMonitorSearchField = 'account' | 'schemeName'

export interface SchemeMonitorQuery {
  searchField?: SchemeMonitorSearchField
  keyword?: string
  kind?: string
  status?: string
  lotteryCode?: string
  /** undefined=全部，false=正式，true=模拟 */
  simBet?: boolean
}

export async function fetchSchemeMonitorInstances(query: SchemeMonitorQuery = {}): Promise<SchemeInstanceRow[]> {
  const params = new URLSearchParams({ scope: 'user' })
  if (query.searchField) params.set('searchField', query.searchField)
  if (query.keyword?.trim()) params.set('keyword', query.keyword.trim())
  const kind = mapKindToApi(query.kind ?? '')
  if (kind) params.set('kind', kind)
  const status = mapStatusToApi(query.status ?? '')
  if (status) params.set('status', status)
  if (query.lotteryCode?.trim()) params.set('lotteryCode', query.lotteryCode.trim())
  if (query.simBet === true) params.set('simBet', 'true')
  if (query.simBet === false) params.set('simBet', 'false')
  const res = await requestApi<{ scope: string; items: ApiMonitorRow[] }>(
    `/admin/schemes/instances?${params.toString()}`,
  )
  return res.items.map(mapMonitorRow)
}

export type SchemeShareSearchField = 'schemeName' | 'snapshotId'

export interface SchemeShareQuery {
  searchField?: SchemeShareSearchField
  keyword?: string
  lotteryCode?: string
}

export interface CreateShareSnapshotInput {
  schemeName: string
  lotteryCode: string
  runTypeId: string
  playTypeId: string
  subPlayId: string
  runMode?: string
  schemeFunds?: string
  startTime?: string
  endTime?: string
  schemeGroups?: string[]
  stopLoss?: string
  takeProfit?: string
  betUnit?: string
  multCoeff?: string
  betMultiplier?: Record<string, unknown>
}

export async function fetchSchemeShareSnapshots(query: SchemeShareQuery = {}): Promise<SchemeShareSnapshotRow[]> {
  const params = new URLSearchParams({ scope: 'share' })
  if (query.searchField) params.set('searchField', query.searchField)
  if (query.keyword?.trim()) params.set('keyword', query.keyword.trim())
  if (query.lotteryCode?.trim()) params.set('lotteryCode', query.lotteryCode.trim())
  const res = await requestApi<{ scope: string; items: ApiShareSnapshot[] }>(
    `/admin/schemes/instances?${params.toString()}`,
  )
  return res.items.map(mapShareSnapshot)
}

export async function createShareSnapshot(input: CreateShareSnapshotInput): Promise<SchemeShareSnapshotRow> {
  const row = await requestApi<ApiShareSnapshot>('/admin/schemes/share', {
    method: 'POST',
    body: {
      ...input,
      typeId: input.playTypeId,
      subId: input.subPlayId,
    },
  })
  return mapShareSnapshot(row)
}

export interface SchemeBetHistoryResult {
  instanceId: string
  schemeName: string
  simBet: boolean
  days: number
  dateFrom: string
  dateTo: string
  items: SchemeBetHistoryItem[]
}

export async function fetchSchemeBetHistory(
  instanceId: string,
  days = 30,
): Promise<SchemeBetHistoryResult> {
  const params = new URLSearchParams()
  if (days > 0) params.set('days', String(days))
  const qs = params.toString()
  return requestApi<SchemeBetHistoryResult>(
    `/admin/schemes/instances/${encodeURIComponent(instanceId)}/bet-history${qs ? `?${qs}` : ''}`,
  )
}

export async function forceStopSchemeInstance(instanceId: string): Promise<void> {
  await requestApi(`/admin/schemes/instances/${encodeURIComponent(instanceId)}/force-stop`, { method: 'POST' })
}

export async function releaseStopSchemeInstance(instanceId: string): Promise<void> {
  await requestApi(`/admin/schemes/instances/${encodeURIComponent(instanceId)}/release-stop`, { method: 'POST' })
}

export async function patchShareSnapshot(
  snapshotId: string,
  input: CreateShareSnapshotInput,
): Promise<SchemeShareSnapshotRow> {
  const row = await requestApi<ApiShareSnapshot>(`/admin/schemes/share/${encodeURIComponent(snapshotId)}`, {
    method: 'PATCH',
    body: {
      ...input,
      typeId: input.playTypeId,
      subId: input.subPlayId,
    },
  })
  return mapShareSnapshot(row)
}

export async function deleteShareSnapshot(snapshotId: string): Promise<void> {
  await requestApi(`/admin/schemes/share/${encodeURIComponent(snapshotId)}`, { method: 'DELETE' })
}

export interface SchemeBettingEventRow {
  decisionId: number
  outboxId?: number
  origin?: 'scheme' | 'api'
  schemeId: string
  schemeName: string
  lotteryCode: string
  sourcePeriod: string
  targetPeriod?: string
  decisionStatus: 'completed' | 'blocked' | 'duplicate' | 'chain_broken'
  mode?: 'shadow' | 'gray' | 'production'
  outboxState?: 'pending' | 'leased' | 'sent_unknown' | 'accepted' | 'rejected' | 'expired' | 'cancelled' | 'accepted_wrong_period' | 'external_acceptance_unknown'
  outcomeReason?: string
  requestId?: string
  stateVersionBefore: number
  stateVersionAfter: number
  localHit?: boolean
  safeDeadlineAt?: string
  closeAt?: string
  decidedAt: string
  chainState?: 'idle' | 'active' | 'blocked_requires_rearm'
  bettingOwner: 'legacy' | 'event'
  initialBet?: boolean
  requestedAmount?: number
  requestedCurrency?: string
  attemptCount?: number
  providerOrderNo?: string
  acceptedPeriodNo?: string
  providerAmount?: number
  providerCurrency?: string
  providerAccountId?: number
  queuePosition?: number
  readyCreatedAt?: string
  attemptStartedAt?: string
  attemptFinishedAt?: string
  lastError?: string
  drawSource?: string
  drawProviderAt?: string
  drawReceivedAt?: string
  drawConfirmedAt?: string
  strategyStartedAt?: string
  strategyCompletedAt?: string
  ruleVersion?: number
  ruleSnapshotHash?: string
  diagnostics: Record<string, unknown>
}

export async function fetchSchemeBettingEvents(query: {
  schemeId?: string
  limit?: number
} = {}): Promise<SchemeBettingEventRow[]> {
  const params = new URLSearchParams()
  if (query.schemeId?.trim()) params.set('schemeId', query.schemeId.trim())
  params.set('limit', String(query.limit ?? 100))
  const res = await requestApi<{ items: SchemeBettingEventRow[]; mode: string }>(
    `/admin/diagnostics/scheme-betting?${params.toString()}`,
  )
  return res.items
}
