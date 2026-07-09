/**
 * 内容契约（与 openapi/components 同步）
 */

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

export interface FaqListItem {
  id: string
  title: string
}

export interface FaqDetail {
  id: string
  title: string
  bodyHtml: string
}

export interface HelpArticle {
  id: string
  title: string
  sort: number
  bodyHtml: string
}

export interface FeedbackSubmitInput {
  subject: string
  content: string
}

export interface FeedbackResult {
  id: number
  subject: string
  createdAt: string
}

export interface AdminAnnouncement {
  id?: string
  title: string
  status: 'draft' | 'published' | 'archived' | string
  publishedAt?: string | null
  bodyHtml?: string
  pinned?: boolean
}

export interface AdminFaqArticle {
  id?: string
  title: string
  sort?: number
  bodyHtml?: string
}

export interface AdminHelpArticle {
  id?: string
  title: string
  sort?: number
  bodyHtml?: string
}

export interface AdminLobbySlot {
  id?: string
  slotKey: string
  title: string
  brief?: string
  sort?: number
  enabled?: boolean
}

export interface PublicLobbySlot {
  slotKey: string
  title: string
  brief?: string
  sort: number
}

export interface AdminBanner {
  id?: string
  remark?: string
  imageUrl: string
  linkUrl?: string
  sort?: number
  enabled?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface AdminBannerListResult {
  items: AdminBanner[]
  total: number
}

export interface PublicBanner {
  id: string
  imageUrl: string
  linkUrl?: string
  sort: number
}

export interface SiteBrand {
  siteName: string
  logoUrl?: string
  tagline?: string
}

export interface AdminMaintenanceState {
  enabled: boolean
  popupAnnouncementId?: string
  title?: string
  message?: string
}

export interface AdminContentBundle {
  announcements: AdminAnnouncement[]
  faqArticles: AdminFaqArticle[]
  helpArticles: AdminHelpArticle[]
  lobbySlots: AdminLobbySlot[]
}
