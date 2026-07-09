import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface FaqListItem {

  id: string

  title: string

}



export interface FaqDetail {

  id: string

  title: string

  bodyHtml: string

}



export async function fetchFaqList(): Promise<FaqListItem[]> {

  await ensureClientSession()

  const res = await requestApi<{ items: FaqListItem[] }>('/client/content/faq')

  return res.items

}



export async function fetchFaqDetail(id: string): Promise<FaqDetail> {

  await ensureClientSession()

  return requestApi<FaqDetail>(`/client/content/faq/${encodeURIComponent(id)}`)

}

