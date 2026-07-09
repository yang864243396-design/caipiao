import { computed, ref } from 'vue'
import { fetchAnnouncements, type AnnouncementListItem } from '@/api/content/announcements'
import { fetchPublicBanners, type PublicBanner } from '@/api/content/banners'

const NEWS_TONES = ['blue', 'amber', 'green'] as const
const NEWS_PREVIEW_LIMIT = 3

export type LobbyNewsRow = {
  id: string
  iconImg: string
  tone: (typeof NEWS_TONES)[number]
  title: string
  time: string
}

export function useLobbyPageContent(iconPlaceholder: string) {
  const banners = ref<PublicBanner[]>([])
  const announcements = ref<AnnouncementListItem[]>([])
  const loaded = ref(false)

  const latestAnnouncement = computed(() => announcements.value[0] ?? null)

  const newsRows = computed<LobbyNewsRow[]>(() =>
    announcements.value.slice(0, NEWS_PREVIEW_LIMIT).map((row, index) => ({
      id: row.id,
      iconImg: iconPlaceholder,
      tone: NEWS_TONES[index % NEWS_TONES.length],
      title: row.title,
      time: row.date,
    })),
  )

  async function load() {
    const [bannerItems, items] = await Promise.all([
      fetchPublicBanners().catch(() => [] as PublicBanner[]),
      fetchAnnouncements().catch(() => [] as AnnouncementListItem[]),
    ])
    banners.value = bannerItems
    announcements.value = items
    loaded.value = true
  }

  return {
    loaded,
    banners,
    latestAnnouncement,
    newsRows,
    load,
  }
}
