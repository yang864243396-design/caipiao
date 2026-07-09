import { ref } from 'vue'
import { fetchPlayTree } from '@/api/playCatalog'
import type { PlayTypeNode } from '@/types/lottery'
import { resolvePlayTypeLabel } from '../../../client/src/utils/playTypeLabels'

function isBarePlayToken(value?: string): boolean {
  const t = String(value ?? '').trim()
  if (!t) return true
  if (/^\d+$/.test(t)) return true
  if (/^g\d+$/i.test(t)) return true
  return false
}

const treeByTemplate = ref<Record<string, PlayTypeNode[]>>({})
const pending = new Set<string>()

export function useAdminPlayTypeLabelCache() {
  async function ensureTemplate(templateCode: string): Promise<void> {
    const code = templateCode.trim()
    if (!code || treeByTemplate.value[code] || pending.has(code)) return
    pending.add(code)
    try {
      const tree = await fetchPlayTree(code)
      treeByTemplate.value = { ...treeByTemplate.value, [code]: tree.playTypes ?? [] }
    } catch {
      treeByTemplate.value = { ...treeByTemplate.value, [code]: [] }
    } finally {
      pending.delete(code)
    }
  }

  async function preloadTemplates(codes: string[]): Promise<void> {
    const unique = [...new Set(codes.map((c) => c.trim()).filter(Boolean))]
    await Promise.all(unique.map(ensureTemplate))
  }

  function resolvePlayTypeLabelForRow(templateCode: string, playTypeId?: string): string {
    const id = String(playTypeId ?? '').trim()
    if (!id) return '—'
    const fromTree = treeByTemplate.value[templateCode.trim()]?.find((t) => t.typeId === id)?.label?.trim()
    if (fromTree) return fromTree
    const fallback = resolvePlayTypeLabel({ playTypeId: id, typeId: id }).trim()
    if (fallback && fallback !== id) return fallback
    return isBarePlayToken(id) ? '—' : fallback || '—'
  }

  return { treeByTemplate, ensureTemplate, preloadTemplates, resolvePlayTypeLabelForRow }
}
