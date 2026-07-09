import { requestApi } from './client'
import type { Announcement, FaqArticle, LobbySlot } from '@/types/content'

export type { Announcement, FaqArticle, LobbySlot }

export interface ContentBundle {
  announcements: Announcement[]
  faqArticles: FaqArticle[]
  lobbySlots: LobbySlot[]
}

function mapAnnouncement(row: {
  id: string
  title: string
  status: string
  publishedAt?: string | null
  bodyHtml: string
  pinned?: boolean
}): Announcement {
  return {
    id: row.id,
    title: row.title,
    status: row.status as Announcement['status'],
    publishedAt: row.publishedAt ?? null,
    bodyHtml: row.bodyHtml,
    pinned: row.pinned ?? false,
  }
}

function mapFaqArticle(row: {
  id: string
  title: string
  sort: number
  bodyHtml: string
}): FaqArticle {
  return {
    id: row.id,
    title: row.title,
    sort: row.sort,
    bodyHtml: row.bodyHtml,
  }
}

function mapLobbySlot(row: {
  id: string
  slotKey: string
  title: string
  brief: string
  sort: number
  enabled: boolean
}): LobbySlot {
  return {
    id: row.id,
    slotKey: row.slotKey,
    title: row.title,
    brief: row.brief,
    sort: row.sort,
    enabled: row.enabled,
  }
}

export async function fetchContentBundle(): Promise<ContentBundle> {
  const data = await requestApi<{
    announcements: Announcement[]
    faqArticles: FaqArticle[]
    lobbySlots: LobbySlot[]
  }>('/admin/content/bundle')
  return {
    announcements: data.announcements.map(mapAnnouncement),
    faqArticles: data.faqArticles.map(mapFaqArticle),
    lobbySlots: (data.lobbySlots ?? []).map(mapLobbySlot),
  }
}

export async function saveAnnouncement(row: Announcement): Promise<Announcement> {
  const saved = await requestApi<{
    id: string
    title: string
    status: string
    publishedAt?: string | null
    bodyHtml: string
  }>('/admin/content/announcements', { method: 'PUT', body: row })
  return mapAnnouncement(saved)
}

export async function deleteAnnouncement(id: string): Promise<void> {
  await requestApi(`/admin/content/announcements/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function setAnnouncementPinned(id: string, pinned: boolean): Promise<Announcement> {
  const saved = await requestApi<{
    id: string
    title: string
    status: string
    publishedAt?: string | null
    bodyHtml: string
    pinned?: boolean
  }>(`/admin/content/announcements/${encodeURIComponent(id)}/pinned`, {
    method: 'PATCH',
    body: { pinned },
  })
  return mapAnnouncement(saved)
}

export async function saveFaqArticle(row: FaqArticle): Promise<FaqArticle> {
  const saved = await requestApi<FaqArticle>('/admin/content/faq/articles', { method: 'PUT', body: row })
  return mapFaqArticle(saved)
}

export async function deleteFaqArticle(id: string): Promise<void> {
  await requestApi(`/admin/content/faq/articles/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function saveLobbySlot(row: LobbySlot): Promise<LobbySlot> {
  const saved = await requestApi<LobbySlot>('/admin/content/lobby-slots', { method: 'PUT', body: row })
  return mapLobbySlot(saved)
}
