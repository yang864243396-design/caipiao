import { computed, ref } from 'vue'
import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'
import {
  fetchClientSchemeTemplates,
  schemeTemplatesPollMs,
} from '@/api/schemeTemplates'
import { isDraftSchemeId } from '@/utils/schemeDraftStorage'

const templatesState = ref<SchemeTemplateRow[]>([])
let activeDefinitionId = ''
let stopSync: (() => void) | null = null

async function mergeDraftAdvancedTemplates(rows: SchemeTemplateRow[]): Promise<SchemeTemplateRow[]> {
  if (!isDraftSchemeId(activeDefinitionId)) return rows
  const { readDraftAdvancedTemplates, draftAdvancedTemplateToRow } = await import('@/utils/draftAdvancedTemplates')
  const draftRows = readDraftAdvancedTemplates().map(draftAdvancedTemplateToRow)
  if (draftRows.length === 0) return rows
  const seen = new Set(rows.map((r) => r.id))
  const merged = [...rows]
  for (const row of draftRows) {
    if (!seen.has(row.id)) merged.push(row)
  }
  return merged.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name))
}

async function refreshFromApi() {
  if (!activeDefinitionId) {
    templatesState.value = []
    return
  }
  try {
    templatesState.value = await mergeDraftAdvancedTemplates(
      await fetchClientSchemeTemplates(activeDefinitionId),
    )
  } catch {
    /* keep last good state */
  }
}

export function refreshSchemeTemplatesState(definitionId?: string) {
  if (definitionId != null && definitionId.trim() !== '') {
    activeDefinitionId = definitionId.trim()
  }
  void refreshFromApi()
}

export function startSchemeTemplatesSync(definitionId: string) {
  activeDefinitionId = definitionId.trim()
  stopSchemeTemplatesSync()
  void refreshFromApi()
  const timer = window.setInterval(refreshFromApi, schemeTemplatesPollMs())
  stopSync = () => window.clearInterval(timer)
}

export function stopSchemeTemplatesSync() {
  stopSync?.()
  stopSync = null
}

/** 客户端倍投设定 · 高级倍投：平台模板 + 当前方案下会员模板 */
export function useSchemeTemplateLibrary() {
  const advancedSchemes = computed(() =>
    templatesState.value.map((t) => ({
      id: t.id,
      title: t.name,
      lotteryCode: t.lotteryCode,
      lotteryLabel: t.lotteryLabel,
      brief: t.brief,
      memberOwned: Boolean(t.memberOwned),
      definitionId: t.definitionId,
    })),
  )

  return {
    advancedSchemes,
    refresh: refreshSchemeTemplatesState,
    startSync: startSchemeTemplatesSync,
    stopSync: stopSchemeTemplatesSync,
  }
}
