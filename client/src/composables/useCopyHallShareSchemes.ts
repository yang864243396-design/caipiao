import { computed, ref, watch } from 'vue'
import { fetchShareCatalog, type SchemeShareSnapshot } from '@/api/schemes/shareCatalog'
import { slotMatchesPlayType } from '@/composables/useCopyHallPlayFilter'
import type { PlayTreeResponse } from '@/types/playCatalog'
import type { CopyHallRankSlot } from '@shared/mock/copyHallRankings'

export function shareSnapshotToRankSlot(item: SchemeShareSnapshot): CopyHallRankSlot {
  const cfg = item.config ?? {}
  return {
    rank: 0,
    schemeId: item.id,
    schemeName: item.schemeName,
    playMethod: item.playMethod ?? '',
    playTypeId: String(cfg.playTypeId ?? cfg.typeId ?? '').trim(),
    subPlayId: String(cfg.subPlayId ?? cfg.subId ?? '').trim(),
    lotteryCode: item.lotteryCode,
    lotteryLabel: item.lotteryLabel,
  }
}

export function useCopyHallShareSchemes(
  lotteryCode: () => string,
  playTypeId: () => string,
  playTree: () => PlayTreeResponse | null,
) {
  const catalogItems = ref<SchemeShareSnapshot[]>([])
  const loading = ref(false)

  async function loadCatalog() {
    loading.value = true
    try {
      const all: SchemeShareSnapshot[] = []
      let cursor: string | undefined
      for (;;) {
        const res = await fetchShareCatalog({ limit: 100, cursor })
        all.push(...res.items)
        if (!res.page.hasMore || !res.page.nextCursor) break
        cursor = res.page.nextCursor
      }
      catalogItems.value = all
    } catch {
      catalogItems.value = []
    } finally {
      loading.value = false
    }
  }

  watch(lotteryCode, () => {
    void loadCatalog()
  }, { immediate: true })

  const lotterySchemes = computed(() => {
    const code = lotteryCode().trim()
    if (!code) return []
    return catalogItems.value.filter((item) => item.lotteryCode === code)
  })

  const filteredSchemes = computed(() => {
    const typeId = playTypeId().trim()
    const tree = playTree()
    return lotterySchemes.value.filter((item) =>
      slotMatchesPlayType(shareSnapshotToRankSlot(item), typeId, tree),
    )
  })

  return { filteredSchemes, loading, refresh: loadCatalog }
}
