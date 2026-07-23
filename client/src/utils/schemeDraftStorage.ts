import type { BetMultiplierPayload } from '@/api/schemes/betMultiplier'
import type {
  SchemeJushuRow,
  SchemeTriggerBet,
  SchemeHotColdWarm,
  SchemeRandomDraw,
  UpdateSchemeInput,
} from '@/api/schemes/definitions'
import type { ClientSchemeKind } from '@/utils/schemeKind'
import { simBetFromLegacyRunMode } from '@/utils/schemeSimBet'

/** 新建方案草稿路由 param，未上云前不落库 */
export const SCHEME_DRAFT_ID = 'new'

const STORAGE_KEY = 'client:scheme-draft:v1'

export function isDraftSchemeId(id: string): boolean {
  return id.trim() === SCHEME_DRAFT_ID
}

export interface SchemeDraftMeta {
  kind: ClientSchemeKind
  schemeName: string
  lotteryCode: string
  runTypeId: string
  playTypeId: string
  subPlayId: string
}

export interface SchemeDraftSnapshot {
  meta: SchemeDraftMeta
  /** false=正式，true=模拟 */
  simBet: boolean
  schemeFunds: string
  /** 方案币种；缺省按 USDT */
  schemeCurrency: string
  startTime: string
  endTime: string
  schemeGroups: string[]
  stopLoss: string
  takeProfit: string
  betUnit: string
  multCoeff: string
  shareStatus: 'private' | 'public'
  betMultiplierKind: '' | '0' | '1' | '2' | '3'
  betMultiplier?: BetMultiplierPayload
  builtinSnapshotId?: string
  jushuList?: SchemeJushuRow[]
  triggerBet?: SchemeTriggerBet
  hotColdWarm?: SchemeHotColdWarm
  randomDraw?: SchemeRandomDraw
}

type LegacyDraft = SchemeDraftSnapshot & { runMode?: 'prod' | 'sim' }

function normalizeDraft(parsed: LegacyDraft): SchemeDraftSnapshot {
  if (typeof parsed.simBet !== 'boolean') {
    const legacy = simBetFromLegacyRunMode(parsed.runMode)
    parsed.simBet = legacy ?? false
  }
  delete parsed.runMode
  const cur = String(parsed.schemeCurrency ?? '').trim().toUpperCase()
  parsed.schemeCurrency = cur === 'TRX' || cur === 'CNY' ? cur : 'USDT'
  return parsed
}

export function loadSchemeDraft(): SchemeDraftSnapshot | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = normalizeDraft(JSON.parse(raw) as LegacyDraft)
    if (!parsed.betUnit && (parsed as { betMode?: string }).betMode) {
      parsed.betUnit = (parsed as { betMode?: string }).betMode!
    }
    return parsed
  } catch {
    return null
  }
}

export function saveSchemeDraft(draft: SchemeDraftSnapshot): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(draft))
  } catch {
    /* ignore quota */
  }
}

export function clearSchemeDraft(): void {
  try {
    sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore */
  }
}

export function draftMetaFromQuery(query: Record<string, unknown>): SchemeDraftMeta {
  const titleRaw = query.title
  const title = decodeURIComponent(String(Array.isArray(titleRaw) ? titleRaw[0] ?? '' : titleRaw ?? ''))
  return {
    kind: 'custom',
    schemeName: title.trim() || '未命名方案',
    lotteryCode: String(Array.isArray(query.lottery) ? query.lottery[0] ?? '' : query.lottery ?? '').trim(),
    runTypeId: String(Array.isArray(query.runType) ? query.runType[0] ?? '' : query.runType ?? 'fixed_rotate').trim(),
    playTypeId: String(Array.isArray(query.playType) ? query.playType[0] ?? '' : query.playType ?? '').trim(),
    subPlayId: String(Array.isArray(query.subPlay) ? query.subPlay[0] ?? '' : query.subPlay ?? '').trim(),
  }
}

export function draftPatchFromSnapshot(draft: SchemeDraftSnapshot): UpdateSchemeInput {
  const patch: UpdateSchemeInput = {
    simBet: draft.simBet,
    schemeFunds: draft.schemeFunds,
    schemeCurrency: draft.schemeCurrency || 'USDT',
    multCoeff: draft.multCoeff,
    startTime: draft.startTime,
    endTime: draft.endTime,
    stopLoss: draft.stopLoss,
    takeProfit: draft.takeProfit,
    betUnit: draft.betUnit,
  }
  if (draft.meta.runTypeId !== 'builtin_plan') {
    patch.schemeGroups =
      draft.meta.runTypeId === 'fixed_number'
        ? [draft.schemeGroups[0] ?? '']
        : [...draft.schemeGroups]
  }
  if (draft.betMultiplier) {
    patch.betMultiplier = draft.betMultiplier as unknown as Record<string, unknown>
  }
  if (draft.jushuList?.length) patch.jushuList = draft.jushuList.map((r) => ({ ...r }))
  if (draft.triggerBet) patch.triggerBet = draft.triggerBet
  if (draft.hotColdWarm) patch.hotColdWarm = draft.hotColdWarm
  if (draft.randomDraw) patch.randomDraw = draft.randomDraw
  if (draft.builtinSnapshotId) {
    patch.builtinPlan = { snapshotId: draft.builtinSnapshotId }
  }
  return patch
}

const RESTORE_PREFIX = 'client:scheme-edit-restore:'

/** 进入子页（倍投设定等）前快照，返回时优先恢复，避免重挂载丢本地编辑 */
export function saveSchemeEditRestoreSnapshot(schemeId: string, draft: SchemeDraftSnapshot): void {
  try {
    sessionStorage.setItem(`${RESTORE_PREFIX}${schemeId.trim() || SCHEME_DRAFT_ID}`, JSON.stringify(draft))
  } catch {
    /* ignore quota */
  }
}

export function consumeSchemeEditRestoreSnapshot(schemeId: string): SchemeDraftSnapshot | null {
  const key = `${RESTORE_PREFIX}${schemeId.trim() || SCHEME_DRAFT_ID}`
  try {
    const raw = sessionStorage.getItem(key)
    sessionStorage.removeItem(key)
    if (!raw) return null
    return normalizeDraft(JSON.parse(raw) as LegacyDraft)
  } catch {
    return null
  }
}

export function saveDraftBetMultiplier(
  _query: Record<string, unknown>,
  kind: '0' | '1' | '2' | '3',
  payload: BetMultiplierPayload,
): void {
  const draft = loadSchemeDraft()
  if (!draft) return
  draft.betMultiplier = payload
  draft.betMultiplierKind = kind
  saveSchemeDraft(draft)
}
