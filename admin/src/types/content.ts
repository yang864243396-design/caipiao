export interface LobbySlot {
  id: string
  slotKey: string
  title: string
  sort: number
  enabled: boolean
  brief: string
}

export interface Announcement {
  id: string
  title: string
  status: '草稿' | '已发布'
  publishedAt: string | null
  bodyHtml: string
  pinned: boolean
}

export interface FaqArticle {
  id: string
  title: string
  sort: number
  bodyHtml: string
}
