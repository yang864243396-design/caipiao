import { requestApi } from './client'

import type { PlayTemplateRow, PlayTypeNode } from '@/types/lottery'



export async function fetchPlayTemplates(): Promise<PlayTemplateRow[]> {

  const res = await requestApi<{ items: PlayTemplateRow[] }>('/admin/games/play-templates')

  return res.items ?? []

}



export async function fetchPlayTree(templateCode: string): Promise<{

  templateCode: string

  playTypes: PlayTypeNode[]

}> {

  return requestApi(`/admin/games/play-templates/${encodeURIComponent(templateCode)}/play-tree`)

}

