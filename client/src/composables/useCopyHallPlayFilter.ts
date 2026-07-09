import { computed, ref, watch } from 'vue'
import { fetchPlayTree } from '@/api/games/lotteries'
import type { PlayTreeResponse } from '@/types/playCatalog'
import { resolvePlayTypeLabel } from '@/utils/playTypeLabels'
import type { CopyHallRankSlot } from '@shared/mock/copyHallRankings'

export function slotMatchesPlayType(
  slot: CopyHallRankSlot,
  typeId: string,
  tree: PlayTreeResponse | null,
): boolean {
  if (!typeId) return false

  if (slot.playTypeId === typeId) return true

  if (tree) {
    const typeNode = tree.playTypes.find((t) => t.typeId === typeId)
    if (typeNode) {
      if (slot.subPlayId && typeNode.subPlays.some((s) => s.subId === slot.subPlayId)) {
        return true
      }
      if (slot.playMethod && slot.playMethod.includes(typeNode.label)) {
        return true
      }
    }
    if (slot.subPlayId) {
      for (const t of tree.playTypes) {
        if (t.typeId === typeId && t.subPlays.some((s) => s.subId === slot.subPlayId)) {
          return true
        }
      }
    }
  }

  return false
}

export function useCopyHallPlayFilter(
  lotteryCode: () => string,
  slots: () => CopyHallRankSlot[],
) {
  const selectedPlayTypeId = ref('')
  const playTree = ref<PlayTreeResponse | null>(null)

  watch(
    lotteryCode,
    async (code) => {
      if (!code) {
        playTree.value = null
        return
      }
      try {
        playTree.value = await fetchPlayTree(code)
      } catch {
        playTree.value = null
      }
    },
    { immediate: true },
  )

  const playFilterOptions = computed(() => {
    const types = playTree.value?.playTypes ?? []
    if (types.length) {
      return types.map((t) => ({ label: t.label, value: t.typeId }))
    }
    const seen = new Map<string, string>()
    for (const slot of slots()) {
      const id = (slot.playTypeId ?? '').trim()
      if (!id || seen.has(id)) continue
      seen.set(id, resolvePlayTypeLabel({ playTypeId: id }))
    }
    return [...seen.entries()].map(([value, label]) => ({ value, label }))
  })

  function pickDefaultPlayType() {
    const first = playFilterOptions.value[0]
    selectedPlayTypeId.value = first ? String(first.value) : ''
  }

  watch(
    playFilterOptions,
    (opts) => {
      if (!opts.length) {
        selectedPlayTypeId.value = ''
        return
      }
      if (!opts.some((o) => String(o.value) === selectedPlayTypeId.value)) {
        selectedPlayTypeId.value = String(opts[0].value)
      }
    },
    { immediate: true },
  )

  const selectedPlayLabel = computed(() => {
    const hit = playFilterOptions.value.find((o) => String(o.value) === selectedPlayTypeId.value)
    return hit?.label ?? playFilterOptions.value[0]?.label ?? '—'
  })

  const filteredSlots = computed(() =>
    slots().filter((s) => slotMatchesPlayType(s, selectedPlayTypeId.value, playTree.value)),
  )

  function resetPlayFilter() {
    pickDefaultPlayType()
  }

  return {
    selectedPlayTypeId,
    playFilterOptions,
    selectedPlayLabel,
    filteredSlots,
    playTree,
    resetPlayFilter,
  }
}
