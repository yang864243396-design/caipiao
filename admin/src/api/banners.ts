import { requestApi } from './client'

export interface LobbyBanner {
  id: string
  remark: string
  imageUrl: string
  linkUrl: string
  sort: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface BannerListQuery {
  page?: number
  pageSize?: number
  enabled?: '' | 'true' | 'false'
  createdFrom?: string
  createdTo?: string
}

export interface BannerListResult {
  items: LobbyBanner[]
  total: number
}

export async function fetchBannerList(query: BannerListQuery): Promise<BannerListResult> {
  const params = new URLSearchParams()
  params.set('page', String(query.page ?? 1))
  params.set('pageSize', String(query.pageSize ?? 10))
  if (query.enabled) params.set('enabled', query.enabled)
  if (query.createdFrom) params.set('createdFrom', query.createdFrom)
  if (query.createdTo) params.set('createdTo', query.createdTo)
  return requestApi<BannerListResult>(`/admin/content/banners?${params.toString()}`)
}

export async function saveBanner(row: Partial<LobbyBanner> & { imageUrl: string }): Promise<LobbyBanner> {
  return requestApi<LobbyBanner>('/admin/content/banners', { method: 'PUT', body: row })
}

export async function setBannerEnabled(id: string, enabled: boolean): Promise<LobbyBanner> {
  return requestApi<LobbyBanner>(`/admin/content/banners/${encodeURIComponent(id)}/enabled`, {
    method: 'PATCH',
    body: { enabled },
  })
}

export async function deleteBanner(id: string): Promise<void> {
  await requestApi(`/admin/content/banners/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
