import { requestApi } from './client'

export interface AdminRole {
  id: string
  name: string
  menuPaths: string[]
}

export async function fetchAdminRoles(): Promise<AdminRole[]> {
  const res = await requestApi<{ items: AdminRole[] }>('/admin/system/roles')
  return res.items
}

export async function saveAdminRole(body: AdminRole): Promise<AdminRole> {
  return requestApi<AdminRole>('/admin/system/roles', { method: 'PUT', body })
}

export async function deleteAdminRole(id: string): Promise<void> {
  await requestApi(`/admin/system/roles/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
