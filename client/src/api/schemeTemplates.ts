import { API_BASE } from './config'
import { requestApi } from './client'
import type { SchemeRoundRule } from '@/api/schemes/betMultiplier'
import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'

function templatesBase(definitionId: string) {
  return `/client/schemes/${encodeURIComponent(definitionId)}/bet-multiplier-templates`
}

export async function fetchClientSchemeTemplates(definitionId: string): Promise<SchemeTemplateRow[]> {
  const id = definitionId.trim()
  if (!id) return []
  const res = await requestApi<{ items: SchemeTemplateRow[] }>(templatesBase(id))
  return res.items
}

export async function getClientSchemeTemplate(
  definitionId: string,
  templateId: string,
): Promise<SchemeTemplateRow> {
  return requestApi<SchemeTemplateRow>(
    `${templatesBase(definitionId)}/${encodeURIComponent(templateId)}`,
  )
}

export interface SaveClientSchemeTemplateInput {
  name: string
  definitionId: string
  brief?: string
  rounds: SchemeRoundRule[]
}

export async function createClientSchemeTemplate(
  input: SaveClientSchemeTemplateInput,
): Promise<SchemeTemplateRow> {
  const { definitionId, name, brief, rounds } = input
  return requestApi<SchemeTemplateRow>(templatesBase(definitionId), {
    method: 'POST',
    body: { name, brief, rounds },
  })
}

export async function saveClientSchemeTemplate(
  definitionId: string,
  templateId: string,
  input: Omit<SaveClientSchemeTemplateInput, 'definitionId'>,
): Promise<SchemeTemplateRow> {
  return requestApi<SchemeTemplateRow>(
    `${templatesBase(definitionId)}/${encodeURIComponent(templateId)}`,
    {
      method: 'PUT',
      body: { name: input.name, brief: input.brief, rounds: input.rounds },
    },
  )
}

export function isSchemeTemplateId(id: string): boolean {
  const s = id.trim()
  return s.startsWith('tpl_') || s.startsWith('scheme_demo_')
}

export function isNewSchemeTemplateRoute(id: string): boolean {
  return id.trim() === 'new'
}

export function schemeTemplatesPollMs() {
  return 15000
}

export { API_BASE }
