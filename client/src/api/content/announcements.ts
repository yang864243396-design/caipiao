import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface AnnouncementListItem {

  id: string

  title: string

  date: string

  read: boolean

}



export interface AnnouncementDetail {

  id: string

  title: string

  date: string

  bodyHtml: string

  read: boolean

}



export async function fetchAnnouncements(): Promise<AnnouncementListItem[]> {

  await ensureClientSession()

  const res = await requestApi<{ items: AnnouncementListItem[] }>('/client/content/announcements')

  return res.items

}



export async function fetchAnnouncementDetail(id: string): Promise<AnnouncementDetail> {

  await ensureClientSession()

  return requestApi<AnnouncementDetail>(`/client/content/announcements/${encodeURIComponent(id)}`)

}

