import { requestApi } from './client'

export interface AdminUser {
  id: number
  account: string
  displayName: string
  roleId: string
  roleName?: string
  status: 'active' | 'disabled'
  lastLoginAt?: string
  createdAt?: string
  updatedAt?: string
}

export interface AdminUserSaveInput {
  account?: string
  displayName: string
  roleId: string
  status: 'active' | 'disabled'
  password?: string
}

export async function fetchAdminUsers(): Promise<AdminUser[]> {
  const res = await requestApi<{ items: AdminUser[] }>('/admin/system/users')
  return res.items
}

export async function createAdminUser(body: AdminUserSaveInput): Promise<AdminUser> {
  if (false) { /* removed mock */
    return {
      id: Date.now(),
      account: body.account ?? '',
      displayName: body.displayName,
      roleId: body.roleId,
      status: body.status,
    }
  }
  return requestApi<AdminUser>('/admin/system/users', { method: 'POST', body })
}

export async function updateAdminUser(id: number, body: AdminUserSaveInput): Promise<AdminUser> {
  if (false) { /* removed mock */
    return {
      id,
      account: body.account ?? '',
      displayName: body.displayName,
      roleId: body.roleId,
      status: body.status,
    }
  }
  return requestApi<AdminUser>(`/admin/system/users/${id}`, { method: 'PUT', body })
}

export async function deleteAdminUser(id: number): Promise<void> {
  await requestApi(`/admin/system/users/${id}`, { method: 'DELETE' })
}
