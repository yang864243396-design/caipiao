import { computed, ref, watch } from 'vue'
import { fetchCopyHallRankings } from '@/api/copyHall/rankings'
import { usePublicLotteries } from '@/composables/usePublicLotteries'
import type { CopyHallBoardKind, CopyHallRankSlot } from '@shared/mock/copyHallRankings'

const apiSlots = ref<CopyHallRankSlot[]>([])
const apiLoading = ref(false)

export function startCopyHallRankingsSync() {
  /* 榜单由页面 watch 拉取 API */
}

export function stopCopyHallRankingsSync() {
  /* no-op */
}

async function loadApiRankings(
  lotteryCode: string,
  boardKind: CopyHallBoardKind,
) {
  apiLoading.value = true
  try {
    const result = await fetchCopyHallRankings(lotteryCode, boardKind)
    apiSlots.value = result.slots
  } catch {
    apiSlots.value = []
  } finally {
    apiLoading.value = false
  }
}

export function useCopyHallRankings(
  lotteryLabel: () => string,
  boardKind: () => CopyHallBoardKind,
) {
  const { lotteries, load, labelToCode } = usePublicLotteries()

  void load()

  const lotteryOptions = computed(() => lotteries.value.map((l) => l.displayName))

  const activeLotteryCode = computed(() => {
    const fromLabel = labelToCode(lotteryLabel())
    if (fromLabel) return fromLabel
    return lotteries.value[0]?.code ?? 'tron_ffc_1m'
  })

  watch(
    [lotteryLabel, boardKind, activeLotteryCode],
    () => {
      void loadApiRankings(activeLotteryCode.value, boardKind())
    },
    { immediate: true },
  )

  const activeSlots = computed(() => apiSlots.value)

  return {
    lotteryOptions,
    activeLotteryCode,
    activeSlots,
    loading: apiLoading,
    refresh: () => loadApiRankings(activeLotteryCode.value, boardKind()),
  }
}
