import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import type { CloudLookbackSettings, CloudRunningScheme } from '@/api/types'



export type { CloudLookbackSettings, CloudRunningScheme }

export const CLOUD_SCHEME_PAGE_SIZE = 20

export interface CloudSchemeListPage {
  items: CloudRunningScheme[]
  total?: number
  page?: {
    nextCursor?: string
    hasMore: boolean
  }
}

/** @deprecated 使用 fetchRunningSchemesPage */
export async function fetchRunningSchemes(runMode?: 'real' | 'sim'): Promise<CloudRunningScheme[]> {
  const res = await fetchRunningSchemesPage({ runMode, limit: 0 })
  return res.items
}

export async function fetchRunningSchemesPage(opts?: {
  limit?: number
  cursor?: string
  runMode?: 'real' | 'sim'
  /** 服务端搜索：方案名称 / 彩种 / 方案定义 ID / 实例 ID */
  q?: string
}): Promise<CloudSchemeListPage> {
  await ensureClientSession()
  const params = new URLSearchParams()
  const limit = opts?.limit ?? CLOUD_SCHEME_PAGE_SIZE
  if (limit > 0) {
    params.set('limit', String(limit))
  }
  if (opts?.cursor) {
    params.set('cursor', opts.cursor)
  }
  if (opts?.runMode) {
    params.set('runMode', opts.runMode)
  }
  const q = opts?.q?.trim()
  if (q) {
    params.set('q', q)
  }
  const qs = params.toString()
  const data = await requestApi<CloudSchemeListPage>(
    `/client/cloud/schemes/running${qs ? `?${qs}` : ''}`,
  )
  return {
    ...data,
    items: data.items.map(mapInstanceToCard),
  }
}

export async function fetchRunningSchemesByIds(ids: string[]): Promise<CloudRunningScheme[]> {
  await ensureClientSession()
  if (ids.length === 0) return []
  const params = new URLSearchParams()
  params.set('ids', ids.join(','))
  const data = await requestApi<{ items: CloudRunningScheme[] }>(
    `/client/cloud/schemes/running?${params.toString()}`,
  )
  return data.items.map(mapInstanceToCard)
}

export interface CloudCenterChannelStatsDto {
  totalTurnover: number
  /** 顶部「总盈亏」：该通道全部实例 session_pnl（本次盈亏）之和 */
  totalSessionPnl: number
  /** 运行中盈亏：running 实例 session_pnl 之和 */
  runningSessionPnl: number
}

export interface CloudSimSchemeQuotaDto {
  todayStarts: number
  todayStartsLimit: number
  running: number
  runningLimit: number
}

export interface CloudCenterStatsDto {
  formal: CloudCenterChannelStatsDto
  sim: CloudCenterChannelStatsDto
  simQuota: CloudSimSchemeQuotaDto
}

const emptyCloudCenterChannelStats = (): CloudCenterChannelStatsDto => ({
  totalTurnover: 0,
  totalSessionPnl: 0,
  runningSessionPnl: 0,
})

const emptySimSchemeQuota = (): CloudSimSchemeQuotaDto => ({
  todayStarts: 0,
  todayStartsLimit: 5,
  running: 0,
  runningLimit: 5,
})

export function emptyCloudCenterStats(): CloudCenterStatsDto {
  return {
    formal: emptyCloudCenterChannelStats(),
    sim: emptyCloudCenterChannelStats(),
    simQuota: emptySimSchemeQuota(),
  }
}

export function formatCloudStatAmount(n: number): string {
  return Number(n ?? 0).toFixed(1)
}

export async function fetchCloudCenterStats(): Promise<CloudCenterStatsDto> {
  await ensureClientSession()
  const raw = await requestApi<CloudCenterStatsDto>('/client/cloud/schemes/stats')
  return {
    formal: raw?.formal ?? emptyCloudCenterChannelStats(),
    sim: raw?.sim ?? emptyCloudCenterChannelStats(),
    simQuota: {
      ...emptySimSchemeQuota(),
      ...(raw?.simQuota ?? {}),
    },
  }
}

export async function fetchLookbackSettings(): Promise<CloudLookbackSettings> {

  await ensureClientSession()

  return requestApi<CloudLookbackSettings>('/client/cloud/lookback')

}



export async function saveLookbackSettings(body: CloudLookbackSettings): Promise<CloudLookbackSettings> {

  await ensureClientSession()

  return requestApi<CloudLookbackSettings>('/client/cloud/lookback', { method: 'PUT', body })

}



export interface CloudGlobalSettingsDto {

  totalStopLoss: number

  totalTakeProfit: number

  planMultiplier: number

  breakPeriodStop: boolean

}



export async function fetchCloudGlobalSettings(): Promise<CloudGlobalSettingsDto> {

  await ensureClientSession()

  return requestApi<CloudGlobalSettingsDto>('/client/cloud/global-settings')

}



export async function saveCloudGlobalSettings(

  body: CloudGlobalSettingsDto,

): Promise<CloudGlobalSettingsDto> {

  await ensureClientSession()

  return requestApi<CloudGlobalSettingsDto>('/client/cloud/global-settings', { method: 'PUT', body })

}



export function globalSettingsToUi(api: CloudGlobalSettingsDto) {

  return {

    totalStopLoss: String(api.totalStopLoss ?? 0),

    totalTakeProfit: String(api.totalTakeProfit ?? 0),

    planMultiplier: String(api.planMultiplier ?? 1),

    breakPeriodStop: api.breakPeriodStop ?? false,

  }

}



export function globalSettingsFromUi(ui: {

  totalStopLoss: string

  totalTakeProfit: string

  planMultiplier: string

  breakPeriodStop: boolean

}): CloudGlobalSettingsDto {

  return {

    totalStopLoss: Number(ui.totalStopLoss) || 0,

    totalTakeProfit: Number(ui.totalTakeProfit) || 0,

    planMultiplier: Number(ui.planMultiplier) || 1,

    breakPeriodStop: ui.breakPeriodStop,

  }

}



export async function startCloudInstance(instanceId: string): Promise<CloudRunningScheme> {

  await ensureClientSession()

  const row = await requestApi<CloudRunningScheme>(`/client/cloud/instances/${encodeURIComponent(instanceId)}/start`, {

    method: 'POST',

  })

  return mapInstanceToCard(row)

}



export async function stopCloudInstance(instanceId: string): Promise<CloudRunningScheme> {

  await ensureClientSession()

  const row = await requestApi<CloudRunningScheme>(`/client/cloud/instances/${encodeURIComponent(instanceId)}/stop`, {

    method: 'POST',

  })

  return mapInstanceToCard(row)

}



/** @deprecated 使用 stopCloudInstance */
export async function pauseCloudInstance(instanceId: string): Promise<CloudRunningScheme> {
  return stopCloudInstance(instanceId)
}



export async function resumeCloudInstance(instanceId: string): Promise<CloudRunningScheme> {
  return startCloudInstance(instanceId)
}

export async function saveCloudInstanceMultiplier(
  instanceId: string,
  multiplier: number,
): Promise<CloudRunningScheme> {
  await ensureClientSession()
  const row = await requestApi<CloudRunningScheme>(
    `/client/cloud/instances/${encodeURIComponent(instanceId)}/multiplier`,
    { method: 'PUT', body: { multiplier } },
  )
  return mapInstanceToCard(row)
}

export async function saveCloudInstanceSimBet(
  instanceId: string,
  simBet: boolean,
): Promise<CloudRunningScheme> {
  await ensureClientSession()
  const row = await requestApi<CloudRunningScheme>(
    `/client/cloud/instances/${encodeURIComponent(instanceId)}/sim-bet`,
    { method: 'PUT', body: { simBet } },
  )
  return mapInstanceToCard(row)
}



/** API SchemeInstance → 卡片展示字段 */

function mapInstanceToCard(row: CloudRunningScheme): CloudRunningScheme {

  return {

    ...row,

    lotteryName: row.lotteryName || (row as CloudRunningScheme & { lotteryLabel?: string }).lotteryLabel || '',

  }

}



type SchemeCardStatusFields = Pick<CloudRunningScheme, 'status' | 'statusReason' | 'statusLabel' | 'countdownSec' | 'countdownLabel'>

export function schemeCardDisplayStatus(card: SchemeCardStatusFields): {
  label: string
  reason: string | undefined
} {
  return { label: card.statusLabel || '', reason: card.statusReason }
}

export const SCHEME_COUNTDOWN_WAITING_LABEL = '请等待'

export function schemeCountdownText(
  card: Pick<CloudSchemeCard, 'status' | 'countdownSec' | 'countdownLabel'>,
): string {
  if (card.countdownLabel) return card.countdownLabel
  if (card.countdownSec <= 0) return SCHEME_COUNTDOWN_WAITING_LABEL
  return formatCountdown(card.countdownSec)
}

export type GameDetailCountdownState = {
  countdownEndTime?: string | null
  countdownPeriod?: string | null
  lotteryCode?: string | null
  countdownSec: number
  countdownLabel?: string | null
}

/** 玩法详情倒计时展示：与云端中心 running 方案一致，仅认 countdownEndTime */
export function gameDetailCountdownDisplayFields(
  row: GameDetailCountdownState,
): SchemeCountdownDisplayResult {
  return schemeCountdownDisplayFields({
    countdownEndTime: row.countdownEndTime,
    lotteryCode: row.lotteryCode,
    status: 'running',
    countdownLabel: row.countdownLabel ?? undefined,
  })
}

export function gameDetailCountdownText(row: GameDetailCountdownState): string {
  const display = gameDetailCountdownDisplayFields(row)
  return schemeCountdownText({
    status: 'running',
    countdownSec: display.countdownSec,
    countdownLabel: display.countdownLabel,
  })
}

/** 第三方期号展示：取末 3 位（与投注 Tab 一致） */
export function thirdPartyPeriodShort(issue?: string | null): string {
  const s = String(issue ?? '').trim()
  if (!s) return '—'
  if (s.length <= 3) return s
  return s.slice(-3)
}

/** 第三方期号展示：去掉前三位前缀（顶部开奖区、历史开奖、投注记录） */
export function thirdPartyPeriodDisplay(issue?: string | null): string {
  const s = String(issue ?? '').trim()
  if (!s) return '—'
  if (s.length <= 3) return s
  return s.slice(3)
}

export const GAME_DETAIL_DRAWING_LABEL = '开奖中'

export function bumpIssuePeriod(issue?: string | null): string {
  const s = String(issue ?? '').trim()
  if (!s) return ''
  const n = Number(s)
  if (Number.isFinite(n)) return String(n + 1)
  return `${s}1`
}

/** 同一 periods 期号内合并 end_time：拒绝更大的倒计时（防止缓存抖动） */
function mergeCountdownEndTimeOnPoll(
  prevEndTime: string | undefined,
  nextEndTime: string | undefined,
  samePeriod: boolean,
  lotteryCode?: string,
): string | undefined {
  const next = nextEndTime?.trim()
  const prev = prevEndTime?.trim()
  if (!next) return prevEndTime
  if (!prev || !samePeriod) return nextEndTime
  const prevSec = countdownSecFromEndTime(prev, lotteryCode)
  const nextSec = countdownSecFromEndTime(next, lotteryCode)
  if (nextSec > prevSec + 1) return prevEndTime
  return nextEndTime
}

/** 轮询合并：同一 periods 期号内保留 countdownEndTime，秒数由 end_time 本地重算 */
export function mergeGameDetailCountdownOnPoll(
  prev: GameDetailCountdownState,
  next: GameDetailCountdownState,
): GameDetailCountdownState {
  const prevPeriod = prev.countdownPeriod?.trim()
  const nextPeriod = next.countdownPeriod?.trim()
  const samePeriod = Boolean(prevPeriod && nextPeriod && prevPeriod === nextPeriod)
  const lotteryCode = next.lotteryCode || prev.lotteryCode
  const endTime = mergeCountdownEndTimeOnPoll(
    prev.countdownEndTime ?? undefined,
    next.countdownEndTime ?? undefined,
    samePeriod,
    lotteryCode ?? undefined,
  )

  const merged: GameDetailCountdownState = {
    ...next,
    countdownEndTime: endTime,
    countdownPeriod: next.countdownPeriod ?? prev.countdownPeriod,
    lotteryCode: next.lotteryCode || prev.lotteryCode,
  }
  const computed = gameDetailCountdownDisplayFields(merged)
  return {
    ...merged,
    countdownSec: computed.countdownSec,
    countdownLabel: computed.countdownLabel || merged.countdownLabel || undefined,
  }
}

export function formatCountdown(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

/** 第三方 periods end_time 墙钟时区（与 backend guaji.PeriodWallClockLocation 一致） */
export function guajiPeriodWallClockOffset(lotteryCode?: string | null): string {
  const code = (lotteryCode ?? '').trim().toLowerCase()
  if (code.startsWith('hash_')) return 'Z'
  if (
    code.startsWith('tron_') ||
    code.startsWith('eth_') ||
    code.startsWith('bnb_') ||
    code.startsWith('taiwan_')
  ) {
    return '+08:00'
  }
  return 'Z'
}

/** 由 countdownEndTime（第三方 periods end_time 墙钟）计算距封盘剩余秒数 */
export function countdownSecFromEndTime(
  endTime?: string | null,
  lotteryCode?: string | null,
): number {
  if (!endTime) return 0
  const raw = endTime.trim()
  if (!raw) return 0
  if (raw.includes('T')) {
    const end = Date.parse(raw)
    if (Number.isFinite(end)) {
      return Math.max(0, Math.round((end - Date.now()) / 1000))
    }
  }
  const normalized = `${raw.replace(' ', 'T')}${guajiPeriodWallClockOffset(lotteryCode)}`
  const end = Date.parse(normalized)
  if (!Number.isFinite(end)) return 0
  return Math.max(0, Math.round((end - Date.now()) / 1000))
}

type SchemeCountdownRow = {
  countdownEndTime?: string | null
  lotteryCode?: string | null
  status?: string
  countdownLabel?: string
}

type SchemeCountdownDisplayResult = {
  countdownSec: number
  countdownLabel: string
}

/** 展示倒计时：仅认 countdownEndTime；running 且已归零、下一期 end_time 未到时显示「请等待」 */
export function schemeCountdownDisplayFields(row: SchemeCountdownRow): SchemeCountdownDisplayResult {
  const sec = countdownSecFromEndTime(row.countdownEndTime, row.lotteryCode)
  if (sec > 0) return { countdownSec: sec, countdownLabel: '' }
  if (row.countdownLabel) {
    return { countdownSec: 0, countdownLabel: row.countdownLabel }
  }
  if (row.status === 'running') {
    return { countdownSec: 0, countdownLabel: SCHEME_COUNTDOWN_WAITING_LABEL }
  }
  return { countdownSec: 0, countdownLabel: '' }
}

/** 同一投注期（periods 期号一致） */
export function isSameSchemeCountdownPeriod(prev: CloudSchemeCard, next: CloudSchemeCard): boolean {
  const p = prev.countdownPeriod?.trim()
  const n = next.countdownPeriod?.trim()
  return Boolean(p && n && p === n)
}

/** 轮询合并：保留 countdownEndTime，展示秒数始终由 end_time 本地重算 */
export function mergeSchemeCountdownOnPoll(
  prev: CloudSchemeCard,
  next: CloudSchemeCard,
): CloudSchemeCard {
  const samePeriod = isSameSchemeCountdownPeriod(prev, next)
  const endTime = mergeCountdownEndTimeOnPoll(
    prev.countdownEndTime,
    next.countdownEndTime,
    samePeriod,
    next.lotteryCode || prev.lotteryCode,
  )

  const mergedBase: CloudSchemeCard = {
    ...next,
    countdownEndTime: endTime,
    countdownPeriod: next.countdownPeriod ?? prev.countdownPeriod,
    lotteryCode: next.lotteryCode || prev.lotteryCode,
    schemeCurrency: normalizeSchemeCurrency(
      next.schemeCurrencyFromApi ? next.schemeCurrency : prev.schemeCurrency || next.schemeCurrency,
    ),
    schemeCurrencyFromApi: next.schemeCurrencyFromApi || prev.schemeCurrencyFromApi,
  }
  const computed = schemeCountdownDisplayFields(mergedBase)

  return {
    ...mergedBase,
    countdownSec: computed.countdownSec,
    countdownLabel: computed.countdownLabel || mergedBase.countdownLabel,
    runTimeSec: mergeSchemeRunTimeOnPoll(prev, mergedBase),
  }
}

/** running 状态本地 tick 自增；轮询时取较大值，避免回跳 */
export function mergeSchemeRunTimeOnPoll(prev: CloudSchemeCard, next: CloudSchemeCard): number {
  if (next.status !== 'running') return next.runTimeSec
  if (prev.status !== 'running') return next.runTimeSec
  return Math.max(prev.runTimeSec, next.runTimeSec)
}

/** running 方案运行时间 +1s（云端中心本地 tick） */
export function tickSchemeRunTimeSec(card: Pick<CloudSchemeCard, 'status' | 'runTimeSec'>): number {
  if (card.status !== 'running') return card.runTimeSec
  return Math.max(0, card.runTimeSec + 1)
}



export function formatRunTime(sec: number): string {

  const h = Math.floor(sec / 3600)

  const m = Math.floor((sec % 3600) / 60)

  const s = sec % 60

  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`

}

/** 方案倍数系数：正整数，最小 1，默认 1 */
export function normalizeSchemeMultiplier(v: string | number | null | undefined): string {
  const digits = String(v ?? '').replace(/[^\d]/g, '')
  if (!digits) return '1'
  const n = parseInt(digits, 10)
  return n >= 1 ? String(n) : '1'
}

/** 方案币种：仅 USDT/TRX/CNY，缺省 USDT */
export function normalizeSchemeCurrency(v: string | null | undefined): 'USDT' | 'TRX' | 'CNY' {
  const cur = String(v ?? '').trim().toUpperCase()
  if (cur === 'TRX' || cur === 'CNY') return cur
  return 'USDT'
}

export function instanceToDisplay(row: CloudRunningScheme) {
  const display = schemeCountdownDisplayFields(row)
  const currencyFromApi = row.schemeCurrency != null && String(row.schemeCurrency).trim() !== ''

  return {

    id: row.id,

    definitionId: row.definitionId || '',

    lotteryCode: row.lotteryCode || '',

    lotteryName: row.lotteryName,

    schemeName: row.schemeName,

    runTypeLabel: row.runTypeLabel ?? '',

    status: row.status,
    statusReason: row.statusReason,
    statusLabel: row.statusLabel,

    turnover: row.turnover.toFixed(1),

    countdownSec: display.countdownSec,

    countdownEndTime: row.countdownEndTime || undefined,

    countdownCloseAt: row.countdownCloseAt || undefined,

    countdownWindowSec: row.countdownWindowSec || undefined,

    countdownPeriod: row.countdownPeriod || undefined,

    countdownLabel: row.countdownLabel || display.countdownLabel,

    pnl: row.pnl.toFixed(1),

    runTimeSec: Math.max(0, Math.trunc(row.runTimeSec ?? 0)),

    lookbackPnl: row.lookbackPnl.toFixed(1),

    sessionPnl: (row.sessionPnl ?? 0).toFixed(1),

    multiplier: normalizeSchemeMultiplier(row.multiplier),

    simBet: row.simBet,

    schemeCurrency: normalizeSchemeCurrency(row.schemeCurrency),

    /** 接口是否显式返回了币种；缺省时合并列表勿用默认 USDT 覆盖本地已选 */
    schemeCurrencyFromApi: currencyFromApi,

  }

}



export type CloudSchemeCard = ReturnType<typeof instanceToDisplay>

/** 轮询/操作后合并列表，保持用户当前看到的顺序，不因 updated_at 重排 */
export function mergeCloudSchemesStable(
  prev: CloudSchemeCard[],
  incoming: CloudSchemeCard[],
): CloudSchemeCard[] {
  if (prev.length === 0) return incoming
  const byId = new Map(incoming.map((s) => [s.id, s]))
  const merged: CloudSchemeCard[] = []
  for (const item of prev) {
    const next = byId.get(item.id)
    if (next) {
      merged.push(mergeSchemeCountdownOnPoll(item, next))
      byId.delete(item.id)
    }
  }
  for (const item of incoming) {
    if (byId.has(item.id)) merged.push(item)
  }
  return merged
}

/** UI prod/sim ↔ API real/sim */

export type LookbackJudgment = 'individual' | 'overall' | ''

export function lookbackToUi(api: CloudLookbackSettings) {
  const applyFormal = api.applyFormal ?? (Array.isArray(api.runModes) && api.runModes.includes('real'))
  const applySim = api.applySim ?? (Array.isArray(api.runModes) && api.runModes.includes('sim'))
  const judgment: LookbackJudgment =
    api.judgment === 'individual' || api.judgment === 'overall' ? api.judgment : ''
  return {
    runModeSim: applySim,
    runModeProd: applyFormal,
    judgment,

    singleProfitThreshold: String(api.singleProfitThreshold ?? ''),

    singleLossThreshold: String(api.singleLossThreshold ?? ''),

    overallProfitThreshold: api.overallProfitThreshold ? String(api.overallProfitThreshold) : '',

    overallLossThreshold: api.overallLossThreshold ? String(api.overallLossThreshold) : '',

    schemeWinsMin: api.schemeWinsMin ? String(api.schemeWinsMin) : '',

    schemeWinsMax: api.schemeWinsMax ? String(api.schemeWinsMax) : '',

    periodProfit: api.periodProfit ? String(api.periodProfit) : '',

    periodLoss: api.periodLoss ? String(api.periodLoss) : '',

  }

}



export function lookbackFromUi(ui: {
  runModeSim: boolean
  runModeProd: boolean
  judgment: LookbackJudgment

  singleProfitThreshold: string

  singleLossThreshold: string

  overallProfitThreshold: string

  overallLossThreshold: string

  schemeWinsMin: string

  schemeWinsMax: string

  periodProfit: string

  periodLoss: string

}): CloudLookbackSettings {

  const num = (s: string) => {

    const n = parseFloat(s)

    return Number.isFinite(n) ? n : 0

  }

  const runModes: ('real' | 'sim')[] = []
  if (ui.runModeProd) runModes.push('real')
  if (ui.runModeSim) runModes.push('sim')
  return {
    applyFormal: ui.runModeProd,
    applySim: ui.runModeSim,
    runModes,
    judgment: ui.judgment,

    singleProfitThreshold: num(ui.singleProfitThreshold),

    singleLossThreshold: num(ui.singleLossThreshold),

    overallProfitThreshold: num(ui.overallProfitThreshold),

    overallLossThreshold: num(ui.overallLossThreshold),

    schemeWinsMin: num(ui.schemeWinsMin),

    schemeWinsMax: num(ui.schemeWinsMax),

    periodProfit: num(ui.periodProfit),

    periodLoss: num(ui.periodLoss),

  }

}



export function lookbackSummaryFromUi(ui: ReturnType<typeof lookbackToUi>): string {
  if (ui.judgment === 'individual') return '个别判断'
  if (ui.judgment === 'overall') return '整体判断'
  return '未选择判断方式'
}

