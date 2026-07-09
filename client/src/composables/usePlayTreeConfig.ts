import { computed, ref, type Ref } from 'vue'
import { fetchPlayTree } from '@/api/games/lotteries'
import type { PlayTreeResponse } from '@/types/playCatalog'
import type { PlayConfig } from '@/utils/betPayload'
import { resolvePlayConfig } from '@/utils/betPayload'
import {
  defaultPlaySelection,
  findSubPlay,
  resolvePlayConfigFromTree,
  type PlayTreePlayConfig,
} from '@/utils/playConfig'

export function usePlayTreeConfig(
  lotteryCode: Ref<string>,
  typeId: Ref<string>,
  subId: Ref<string>,
) {
  const playTree = ref<PlayTreeResponse | null>(null)
  const loading = ref(false)

  async function load(): Promise<void> {
    const code = lotteryCode.value.trim()
    if (!code) {
      playTree.value = null
      return
    }
    loading.value = true
    try {
      playTree.value = await fetchPlayTree(code)
      if (!typeId.value || !subId.value) {
        const def = defaultPlaySelection(playTree.value)
        typeId.value = def.typeId
        subId.value = def.subId
      }
    } catch {
      playTree.value = null
    } finally {
      loading.value = false
    }
  }

  const playConfig = computed((): PlayConfig | PlayTreePlayConfig => {
    if (playTree.value && typeId.value && subId.value) {
      const sel = findSubPlay(playTree.value, typeId.value, subId.value)
      if (sel) {
        return resolvePlayConfigFromTree(
          playTree.value.playTemplate,
          sel.typeNode,
          sel.subNode,
        )
      }
    }
    return resolvePlayConfig({
      playTypeId: typeId.value || undefined,
      subPlayId: subId.value || undefined,
    })
  })

  return { playTree, loading, load, playConfig }
}
