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
    const tree = playTree.value
    const tid = typeId.value.trim()
    const sid = subId.value.trim()
    if (tree && tid && sid) {
      const sel = findSubPlay(tree, tid, sid)
      if (sel) {
        return resolvePlayConfigFromTree(tree.playTemplate, sel.typeNode, sel.subNode)
      }
      // 树已加载但未精确命中：勿把非 SSC 的 g001 误映射成时时彩「前三」
      // （六合彩 g001=特码、子玩法 272=特码A）
      if (tree.playTemplate !== 'ssc_std' && tree.playTemplate !== 'fast_ssc_std') {
        const typeNode = tree.playTypes.find((t) => t.typeId === tid)
        return {
          playTemplate: tree.playTemplate,
          typeId: tid,
          subId: sid,
          betMode: '',
          playTypeLabel: typeNode?.label?.trim() || tid,
          playMethodLabel: sid,
          playTypeId: tid,
          subPlayId: sid,
          catalogSubId: sid,
          segmentLen: 1,
          segmentLabels: ['选号'],
          inputMode: 'pool',
        } as PlayTreePlayConfig
      }
    }
    return resolvePlayConfig({
      playTypeId: tid || undefined,
      subPlayId: sid || undefined,
    })
  })

  return { playTree, loading, load, playConfig }
}
