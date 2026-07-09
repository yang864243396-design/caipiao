import { computed, ref, type Ref } from 'vue'
import { storeToRefs } from 'pinia'
import { fetchPlayTree } from '@/api/playCatalog'
import { useLotteryCatalogStore } from '@/stores/lotteryCatalog'
import {
  defaultPlaySelection,
  findSubPlay,
  resolvePlayConfigFromTree,
} from '@client/utils/playConfig'
import { resolvePlayConfig, type PlayConfig } from '@client/utils/betPayload'

export function useAdminPlayTreeConfig(
  lotteryCode: Ref<string>,
  typeId: Ref<string>,
  subId: Ref<string>,
) {
  const catalog = useLotteryCatalogStore()
  const { rows: lotteryRows } = storeToRefs(catalog)
  const playTemplate = ref('')
  const playTreeTypes = ref<Awaited<ReturnType<typeof fetchPlayTree>>['playTypes']>([])
  const loading = ref(false)

  async function load(): Promise<void> {
    const code = lotteryCode.value.trim()
    if (!code) {
      playTemplate.value = ''
      playTreeTypes.value = []
      return
    }
    const lot = lotteryRows.value.find((r) => r.code === code)
    const template = lot?.playTemplate?.trim()
    if (!template) {
      playTemplate.value = ''
      playTreeTypes.value = []
      return
    }
    loading.value = true
    try {
      const tree = await fetchPlayTree(template)
      playTemplate.value = template
      playTreeTypes.value = tree.playTypes ?? []
      if (!typeId.value || !subId.value) {
        const pseudoTree = { playTemplate: template, playTypes: tree.playTypes ?? [] }
        const def = defaultPlaySelection(pseudoTree)
        if (!typeId.value) typeId.value = def.typeId
        if (!subId.value) subId.value = def.subId
      }
    } catch {
      playTemplate.value = ''
      playTreeTypes.value = []
    } finally {
      loading.value = false
    }
  }

  const playConfig = computed((): PlayConfig => {
    if (playTemplate.value && typeId.value && subId.value && playTreeTypes.value.length) {
      const pseudoTree = { playTemplate: playTemplate.value, playTypes: playTreeTypes.value }
      const sel = findSubPlay(pseudoTree, typeId.value, subId.value)
      if (sel) {
        return resolvePlayConfigFromTree(playTemplate.value, sel.typeNode, sel.subNode)
      }
    }
    return resolvePlayConfig({
      playTypeId: typeId.value || undefined,
      subPlayId: subId.value || undefined,
    })
  })

  return { playTemplate, playTreeTypes, loading, load, playConfig }
}
