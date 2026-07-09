import { ref } from 'vue'
import { fetchPublicLotteries } from '@/api/games/lotteries'
import type { PublicLotteryRow } from '@/types/playCatalog'

const lotteries = ref<PublicLotteryRow[]>([])
const loading = ref(false)
const loaded = ref(false)

export function usePublicLotteries() {
  async function load(force = false) {
    if (loaded.value && !force) return lotteries.value
    loading.value = true
    try {
      lotteries.value = await fetchPublicLotteries()
      loaded.value = true
    } catch {
      if (!lotteries.value.length) lotteries.value = []
    } finally {
      loading.value = false
    }
    return lotteries.value
  }

  function labelToCode(label: string): string | undefined {
    return lotteries.value.find((l) => l.displayName === label)?.code
  }

  function codeToLabel(code: string): string | undefined {
    return lotteries.value.find((l) => l.code === code)?.displayName
  }

  return {
    lotteries,
    loading,
    loaded,
    load,
    labelToCode,
    codeToLabel,
  }
}
