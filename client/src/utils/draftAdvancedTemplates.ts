import type { BetMultiplierPayload, SchemeRoundRule } from '@/api/schemes/betMultiplier'
import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'
import {
  loadSchemeDraft,
  saveSchemeDraft,
  type SchemeDraftSnapshot,
} from '@/utils/schemeDraftStorage'

export const DRAFT_ADVANCED_TEMPLATE_PREFIX = 'draft_tpl_'

export interface DraftAdvancedTemplate {
  id: string
  name: string
  rounds: SchemeRoundRule[]
}

export function isDraftAdvancedTemplateId(id: string): boolean {
  return id.trim().startsWith(DRAFT_ADVANCED_TEMPLATE_PREFIX)
}

export function newDraftAdvancedTemplateId(): string {
  return `${DRAFT_ADVANCED_TEMPLATE_PREFIX}${Date.now()}`
}

function normalizeDraftAdvancedRounds(raw: unknown): SchemeRoundRule[] {
  if (!Array.isArray(raw)) return []
  const out: SchemeRoundRule[] = []
  for (const item of raw) {
    if (item == null || typeof item !== 'object') continue
    const row = item as Record<string, unknown>
    const mult = Number(row.mult)
    const afterHit = Number(row.afterHit)
    const afterMiss = Number(row.afterMiss)
    if (!Number.isFinite(mult) || !Number.isFinite(afterHit) || !Number.isFinite(afterMiss)) continue
    out.push({ mult, afterHit, afterMiss })
  }
  return out
}

function advancedSection(draft: SchemeDraftSnapshot): Record<string, unknown> {
  const adv = draft.betMultiplier?.advanced
  if (adv && typeof adv === 'object') return { ...adv }
  return {}
}

export function readDraftAdvancedTemplatesFromSnapshot(draft: SchemeDraftSnapshot): DraftAdvancedTemplate[] {
  if (!draft.betMultiplier?.advanced) return []
  const adv = draft.betMultiplier.advanced as Record<string, unknown>
  const raw = adv.customTemplates
  if (!Array.isArray(raw)) return []
  const out: DraftAdvancedTemplate[] = []
  for (const item of raw) {
    if (item == null || typeof item !== 'object') continue
    const row = item as Record<string, unknown>
    const id = String(row.id ?? '').trim()
    const name = String(row.name ?? '').trim()
    if (!id || !name || !isDraftAdvancedTemplateId(id)) continue
    const rounds = normalizeDraftAdvancedRounds(row.rounds)
    if (!rounds.length) continue
    out.push({ id, name, rounds })
  }
  return out
}

export function readDraftAdvancedTemplates(): DraftAdvancedTemplate[] {
  const draft = loadSchemeDraft()
  if (!draft) return []
  return readDraftAdvancedTemplatesFromSnapshot(draft)
}

export function getDraftAdvancedTemplate(id: string): DraftAdvancedTemplate | null {
  const key = id.trim()
  return readDraftAdvancedTemplates().find((t) => t.id === key) ?? null
}

export function upsertDraftAdvancedTemplate(tmpl: DraftAdvancedTemplate): boolean {
  const draft = loadSchemeDraft()
  if (!draft) return false
  const adv = advancedSection(draft)
  const list = readDraftAdvancedTemplatesFromSnapshot(draft).filter((t) => t.id !== tmpl.id)
  list.push(tmpl)
  adv.customTemplates = list
  const kind = draft.betMultiplier?.kind ?? draft.betMultiplierKind ?? '3'
  draft.betMultiplier = {
    ...draft.betMultiplier,
    kind: kind as BetMultiplierPayload['kind'],
    advanced: adv,
  }
  if (!draft.betMultiplierKind) draft.betMultiplierKind = '3'
  saveSchemeDraft(draft)
  return true
}

export function draftAdvancedTemplateToRow(t: DraftAdvancedTemplate): SchemeTemplateRow {
  const ts = new Date().toISOString()
  return {
    id: t.id,
    name: t.name,
    lotteryCode: '',
    lotteryLabel: '',
    sortOrder: 9100,
    enabled: true,
    memberOwned: true,
    config: { rounds: t.rounds },
    createdAt: ts,
    updatedAt: ts,
  }
}

/** 草稿上云后，将会员高级倍投模板写入服务端并 remap selectedId */
export async function syncDraftAdvancedTemplatesToServer(
  definitionId: string,
  draft: SchemeDraftSnapshot,
): Promise<BetMultiplierPayload | undefined> {
  const custom = readDraftAdvancedTemplatesFromSnapshot(draft)
  const base = draft.betMultiplier
  if (!base) return undefined

  const adv = (base.advanced ?? {}) as Record<string, unknown>
  if (custom.length === 0) {
    if (!adv.customTemplates) return base
    const { customTemplates: _removed, ...restAdv } = adv
    return { ...base, advanced: restAdv }
  }

  const { createClientSchemeTemplate } = await import('@/api/schemeTemplates')
  const idMap = new Map<string, string>()
  for (const t of custom) {
    const created = await createClientSchemeTemplate({
      name: t.name,
      definitionId,
      rounds: t.rounds,
    })
    idMap.set(t.id, created.id)
  }

  const selectedRaw = adv.selectedId
  let selectedId = selectedRaw
  if (typeof selectedRaw === 'string' && idMap.has(selectedRaw)) {
    selectedId = idMap.get(selectedRaw)
  }

  const { customTemplates: _removed, ...restAdv } = adv
  return {
    ...base,
    advanced: {
      ...restAdv,
      selectedId,
    },
  }
}
