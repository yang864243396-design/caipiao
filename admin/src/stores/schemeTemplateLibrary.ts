import { defineStore } from 'pinia'

import { ref } from 'vue'

import {
  deleteSchemeTemplate,
  fetchSchemeTemplateById,
  fetchSchemeTemplates,
  saveSchemeTemplate,
  type SchemeRoundRule,
} from '@/api/schemeTemplates'

import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'

import { schemeRoundRulesFromConfig } from '@shared/schemeRoundRules'

export const useSchemeTemplateLibraryStore = defineStore('schemeTemplateLibrary', () => {
  const templates = ref<SchemeTemplateRow[]>([])

  const total = ref(0)

  const page = ref(1)

  const pageSize = ref(10)

  const loading = ref(false)

  const appliedName = ref('')

  async function loadList(opts?: { page?: number; pageSize?: number; name?: string }) {
    if (opts?.page != null) page.value = opts.page
    if (opts?.pageSize != null) pageSize.value = opts.pageSize
    if (opts?.name != null) appliedName.value = opts.name

    loading.value = true
    try {
      const res = await fetchSchemeTemplates({
        page: page.value,
        pageSize: pageSize.value,
        name: appliedName.value,
      })

      templates.value = res.items
      total.value = res.total
      page.value = res.page
      pageSize.value = res.pageSize
    } finally {
      loading.value = false
    }
  }

  async function fetchTemplate(id: string): Promise<SchemeTemplateRow> {
    return fetchSchemeTemplateById(id)
  }

  async function createTemplate(input: {
    name: string
    brief?: string
    sortOrder?: number
    enabled?: boolean
    rounds: SchemeRoundRule[]
  }): Promise<SchemeTemplateRow> {
    const row = await saveSchemeTemplate(input)
    await loadList()
    return row
  }

  async function updateTemplate(
    id: string,
    patch: Partial<SchemeTemplateRow> & { rounds?: SchemeRoundRule[] },
  ): Promise<SchemeTemplateRow> {
    const prev = templates.value.find((r) => r.id === id)
    if (!prev) {
      throw new Error('模板不存在')
    }

    const row = await saveSchemeTemplate({
      id,
      name: patch.name ?? prev.name,
      brief: patch.brief ?? prev.brief,
      sortOrder: patch.sortOrder ?? prev.sortOrder,
      enabled: patch.enabled ?? prev.enabled,
      rounds: patch.rounds ?? schemeRoundRulesFromConfig(prev.config),
    })

    const idx = templates.value.findIndex((r) => r.id === id)
    if (idx >= 0) {
      templates.value[idx] = row
    }
    await loadList()
    return row
  }

  async function removeTemplate(id: string): Promise<void> {
    await deleteSchemeTemplate(id)

    if (templates.value.length <= 1 && page.value > 1) {
      page.value -= 1
    }

    await loadList()
  }

  return {
    templates,
    total,
    page,
    pageSize,
    loading,
    appliedName,
    loadList,
    fetchTemplate,
    createTemplate,
    updateTemplate,
    removeTemplate,
  }
})
