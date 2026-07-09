import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Announcement, FaqArticle, LobbySlot } from '@/types/content'
import {
  deleteAnnouncement as apiDeleteAnnouncement,
  deleteFaqArticle as apiDeleteFaqArticle,
  fetchContentBundle,
  saveAnnouncement as apiSaveAnnouncement,
  saveFaqArticle as apiSaveFaqArticle,
  saveLobbySlot as apiSaveLobbySlot,
  setAnnouncementPinned as apiSetAnnouncementPinned,
} from '@/api/content'

function nid(prefix: string) {
  return `${prefix}${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`
}

export const useContentStore = defineStore('content', () => {
  const lobbySlots = ref<LobbySlot[]>([])
  const announcements = ref<Announcement[]>([])
  const faqArticles = ref<FaqArticle[]>([])
  const hydrated = ref(false)
  const loading = ref(false)

  async function hydrate() {
    if (hydrated.value) return
    loading.value = true
    try {
      const bundle = await fetchContentBundle()
      announcements.value = bundle.announcements
      faqArticles.value = bundle.faqArticles
      lobbySlots.value = bundle.lobbySlots
      hydrated.value = true
    } finally {
      loading.value = false
    }
  }

  async function upsertLobby(row: LobbySlot) {
    const saved = await apiSaveLobbySlot(row)
    const i = lobbySlots.value.findIndex((x) => x.id === saved.id)
    if (i >= 0) lobbySlots.value[i] = { ...saved }
    else lobbySlots.value.push({ ...saved })
    lobbySlots.value.sort((a, b) => a.sort - b.sort)
  }

  async function upsertAnnouncement(row: Announcement) {
    const saved = await apiSaveAnnouncement(row)
    const i = announcements.value.findIndex((x) => x.id === saved.id)
    if (i >= 0) announcements.value[i] = saved
    else announcements.value.push(saved)
  }

  async function removeAnnouncement(id: string) {
    await apiDeleteAnnouncement(id)
    announcements.value = announcements.value.filter((x) => x.id !== id)
  }

  async function pinAnnouncement(id: string, pinned: boolean) {
    const saved = await apiSetAnnouncementPinned(id, pinned)
    announcements.value = announcements.value.map((row) => {
      if (row.id === saved.id) return saved
      if (pinned) return { ...row, pinned: false }
      return row
    })
  }

  async function upsertFaqArticle(row: FaqArticle) {
    const saved = await apiSaveFaqArticle(row)
    const i = faqArticles.value.findIndex((x) => x.id === saved.id)
    if (i >= 0) faqArticles.value[i] = saved
    else faqArticles.value.push(saved)
    faqArticles.value.sort((a, b) => a.sort - b.sort || a.title.localeCompare(b.title))
  }

  async function removeFaqArticle(id: string) {
    await apiDeleteFaqArticle(id)
    faqArticles.value = faqArticles.value.filter((x) => x.id !== id)
  }

  function newAnnouncement(): Announcement {
    return {
      id: nid('ANN_'),
      title: '未命名公告',
      status: '草稿',
      publishedAt: null,
      bodyHtml: '<p></p>',
      pinned: false,
    }
  }

  function newFaqArticle(): FaqArticle {
    const sort = (faqArticles.value.reduce((m, a) => Math.max(m, a.sort), 0) || 0) + 1
    return {
      id: nid('FAQ_'),
      title: '新问题',
      sort,
      bodyHtml: '<p></p>',
    }
  }

  return {
    lobbySlots,
    announcements,
    faqArticles,
    loading,
    hydrate,
    upsertLobby,
    upsertAnnouncement,
    removeAnnouncement,
    pinAnnouncement,
    upsertFaqArticle,
    removeFaqArticle,
    newAnnouncement,
    newFaqArticle,
  }
})
