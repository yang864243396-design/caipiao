import { defineStore } from 'pinia'

import { computed, ref, watch } from 'vue'

import {
  moveRankSlot,
  type CopyHallBoardKind,
  type CopyHallRankSlot,
} from '@shared/mock/copyHallRankings'

import {
  candidateToRankSlot,
  filterCopyHallSchemes,
  findCopyHallSchemeIn,
  type CopyHallSchemeCandidate,
} from '@shared/mock/copyHallSchemeCatalog'

import { fetchCopyHallRankingsState, saveCopyHallBoard } from '@/api/copyHall'

import { fetchSchemeShareSnapshots } from '@/api/schemes'

import { shareSnapshotToCopyHallCandidate } from '@/utils/schemeInstanceToCopyHall'

export type CopyHallPickerSearchField = 'schemeName' | 'snapshotId'

export const useCopyHallOpsStore = defineStore('copyHallOps', () => {
  const hydrated = ref(false)

  const activeBoard = ref<CopyHallBoardKind>('master')

  const currentSlots = ref<CopyHallRankSlot[]>([])

  const poolKeyword = ref('')

  const poolStatus = ref<string>('运行中')

  const poolCreatedRange = ref<[string, string] | null>(null)

  /** 分享池方案（更换方案弹窗 · 全站） */
  const poolRows = ref<CopyHallSchemeCandidate[]>([])

  const poolLoading = ref(false)

  const pickerSearchField = ref<CopyHallPickerSearchField>('schemeName')

  const pickerKeyword = ref('')

  const appliedPickerQuery = ref<{ searchField: CopyHallPickerSearchField; keyword: string }>({
    searchField: 'schemeName',
    keyword: '',
  })

  const usedSchemeIds = computed(
    () => new Set(currentSlots.value.filter((s) => s.schemeId).map((s) => s.schemeId)),
  )

  const availableSchemes = computed(() =>
    filterCopyHallSchemes(poolRows.value, {
      keyword: poolKeyword.value,
      status: poolStatus.value || undefined,
      createdStart: poolCreatedRange.value?.[0],
      createdEnd: poolCreatedRange.value?.[1],
    }),
  )

  /** 更换方案弹窗：全站方案监控分享池 */
  const pickerSchemes = computed(() =>
    filterCopyHallSchemes(poolRows.value, {
      keyword: appliedPickerQuery.value.keyword,
      searchField: appliedPickerQuery.value.searchField,
    }),
  )

  async function saveCurrentBoardToApi(slots: CopyHallRankSlot[]) {
    await saveCopyHallBoard(activeBoard.value, slots)
  }

  function emptyRankSlot(rank: number): CopyHallRankSlot {
    return { rank, schemeId: '', schemeName: '', playMethod: '' }
  }

  function buildSlotsAfterAssign(rank: number, candidate: CopyHallSchemeCandidate): CopyHallRankSlot[] {
    const schemeId = candidate.schemeId.trim()
    return currentSlots.value.map((s) => {
      if (s.rank === rank) return candidateToRankSlot(candidate, rank)
      if (schemeId && s.schemeId === schemeId) return emptyRankSlot(s.rank)
      return s
    })
  }

  async function assignScheme(rank: number, candidate: CopyHallSchemeCandidate) {
    const schemeId = candidate.schemeId.trim()
    if (schemeId) {
      const occupied = currentSlots.value.find((s) => s.schemeId === schemeId)
      if (occupied && occupied.rank !== rank) {
        throw new Error(`该方案已在第 ${occupied.rank} 名，不可重复上榜`)
      }
    }

    const prev = currentSlots.value
    const next = buildSlotsAfterAssign(rank, candidate)
    try {
      await saveCurrentBoardToApi(next)
      currentSlots.value = next
      await reloadRankings()
    } catch (e) {
      currentSlots.value = prev
      throw e
    }
  }

  async function moveSlot(rank: number, direction: 'up' | 'down') {
    const prev = currentSlots.value
    const next = moveRankSlot(currentSlots.value, rank, direction)
    try {
      await saveCurrentBoardToApi(next)
      currentSlots.value = next
      await reloadRankings()
    } catch (e) {
      currentSlots.value = prev
      throw e
    }
  }

  async function reloadRankings() {
    const kind = activeBoard.value
    const result = await fetchCopyHallRankingsState(kind)
    currentSlots.value = result.slots
  }

  async function reloadPool() {
    poolLoading.value = true
    try {
      const rows = await fetchSchemeShareSnapshots()
      poolRows.value = rows.map(shareSnapshotToCopyHallCandidate)
    } catch {
      poolRows.value = []
    } finally {
      poolLoading.value = false
    }
  }

  async function searchPool() {
    await reloadPool()
  }

  function resetPickerSearch() {
    pickerSearchField.value = 'schemeName'
    pickerKeyword.value = ''
    appliedPickerQuery.value = { searchField: 'schemeName', keyword: '' }
  }

  function searchPicker() {
    appliedPickerQuery.value = {
      searchField: pickerSearchField.value,
      keyword: pickerKeyword.value.trim(),
    }
  }

  async function hydrate() {
    if (!hydrated.value) {
      hydrated.value = true
    }
    await reloadRankings()
    await reloadPool()
  }

  function syncFromStorage() {
    void hydrate()
  }

  function resolveScheme(schemeId: string): CopyHallSchemeCandidate | undefined {
    return findCopyHallSchemeIn(poolRows.value, schemeId)
  }

  watch(activeBoard, () => {
    void reloadRankings().catch(() => {
      currentSlots.value = []
    })
  })

  return {
    hydrated,
    activeBoard,
    poolKeyword,
    poolStatus,
    poolCreatedRange,
    poolLoading,
    currentSlots,
    usedSchemeIds,
    availableSchemes,
    pickerSchemes,
    pickerSearchField,
    pickerKeyword,
    hydrate,
    syncFromStorage,
    assignScheme,
    moveSlot,
    resolveScheme,
    searchPool,
    resetPickerSearch,
    searchPicker,
  }
})

export type { CopyHallRankSlot, CopyHallSchemeCandidate }
