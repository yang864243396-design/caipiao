import { requestApi } from '@/api/client'

export interface PublicBanner {
  id: string
  imageUrl: string
  linkUrl: string
  sort: number
}

export async function fetchPublicBanners(): Promise<PublicBanner[]> {
  const res = await requestApi<{ items: PublicBanner[] }>('/public/banners', { auth: false })
  return res.items ?? []
}
