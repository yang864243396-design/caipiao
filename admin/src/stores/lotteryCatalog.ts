import { defineStore } from 'pinia'

import { ref } from 'vue'

import type { LotteryCatalogRow } from '@/types/lottery'

import { fetchLotteryCatalog } from '@/api/lotteryCatalog'



export type { LotteryCatalogRow }



export const useLotteryCatalogStore = defineStore('lotteryCatalog', () => {

  const rows = ref<LotteryCatalogRow[]>([])

  const hydrated = ref(false)



  async function hydrate() {

    if (hydrated.value) return

    rows.value = await fetchLotteryCatalog()

    hydrated.value = true

  }



  return { rows, hydrated, hydrate }

})

