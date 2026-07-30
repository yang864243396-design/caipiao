import { computed, ref } from 'vue'
import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'
import {
  fetchClientSchemeTemplates,
  schemeTemplatesPollMs,
} from '@/api/schemeTemplates'
import {
  draftAdvancedTemplateToRow,
  getDraftAdvancedTemplate,
  isDraftAdvancedTemplateId,
  readDraftAdvancedTemplates,
} from '@/utils/draftAdvancedTemplates'

const templatesState = ref<SchemeTemplateRow[]>([])
let activeDefinitionId = ''
let stopSync: (() => void) | null = null

async function mergeDraftAdvancedTemplates(rows: SchemeTemplateRow[]): Promise<SchemeTemplateRow[]> {
  // 草稿与云端编辑都合并本地 draft_tpl_*，避免「新方案」只在新建草稿时可见
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

  /** 保存高级倍投时把选中模板的 rounds 一并写入定义，供 Worker 编译。 */
  function roundsForTemplate(id: string): Array<{ mult: number; afterHit: number; afterMiss: number }> | null {
    const key = id.trim()
    if (!key) return null
    const row = templatesState.value.find((t) => t.id === key)
    let raw = row?.config?.rounds
    // 云端编辑时列表可能不含 draft_tpl_*，回退读 localStorage 草稿模板
    if ((!Array.isArray(raw) || raw.length === 0) && isDraftAdvancedTemplateId(key)) {
      raw = getDraftAdvancedTemplate(key)?.rounds
    }
    if (!Array.isArray(raw) || raw.length === 0) return null
    const out: Array<{ mult: number; afterHit: number; afterMiss: number }> = []
    for (const item of raw) {
      if (item == null || typeof item !== 'object') continue
      const r = item as Record<string, unknown>
      const mult = Number(r.mult)
      const afterHit = Number(r.afterHit)
      const afterMiss = Number(r.afterMiss)
      if (!Number.isFinite(mult) || !Number.isFinite(afterHit) || !Number.isFinite(afterMiss)) continue
      out.push({ mult, afterHit, afterMiss })
    }
    return out.length > 0 ? out : null
  }

  return {
    advancedSchemes,
    roundsForTemplate,
    refresh: refreshSchemeTemplatesState,
    startSync: startSchemeTemplatesSync,
    stopSync: stopSchemeTemplatesSync,
  }
}
