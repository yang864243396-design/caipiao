import { requestApi } from './client'

import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'

import type { SchemeRoundRule } from '@shared/schemeRoundRules'

export type { SchemeTemplateRow }

export type { SchemeRoundRule }

export interface AdminSchemeTemplateListResult {
  items: SchemeTemplateRow[]
  total: number
  page: number
  pageSize: number
}

export interface FetchSchemeTemplatesParams {
  page?: number
  pageSize?: number
  name?: string
}

export async function fetchSchemeTemplates(
  params: FetchSchemeTemplatesParams = {},
): Promise<AdminSchemeTemplateListResult> {
  const page = params.page ?? 1
  const pageSize = params.pageSize ?? 10
  const qs = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  })
  const name = params.name?.trim()
  if (name) qs.set('name', name)
  return requestApi<AdminSchemeTemplateListResult>(`/admin/games/scheme-templates?${qs}`)
}

export async function fetchSchemeTemplateById(id: string): Promise<SchemeTemplateRow> {
  return requestApi<SchemeTemplateRow>(`/admin/games/scheme-templates/${encodeURIComponent(id)}`)
}

export async function saveSchemeTemplate(body: {
  id?: string
  name: string
  brief?: string
  sortOrder?: number
  enabled?: boolean
  rounds: SchemeRoundRule[]
}): Promise<SchemeTemplateRow> {
  return requestApi<SchemeTemplateRow>('/admin/games/scheme-templates', { method: 'PUT', body })
}

export async function deleteSchemeTemplate(id: string): Promise<void> {
  await requestApi(`/admin/games/scheme-templates/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
