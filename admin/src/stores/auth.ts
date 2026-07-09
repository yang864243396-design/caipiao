import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getAccessToken, getExpiresAt } from '@/api/client'
import { fetchAdminSession, loginAdmin, logoutAdmin } from '@/api/auth'

const ADMIN_ROLE_KEY = 'admin_active_role_id'

export const useAuthStore = defineStore('auth', () => {
  const adminRoleId = ref(localStorage.getItem(ADMIN_ROLE_KEY) || 'r_super')

  // 响应式持有 token / 过期时间：localStorage 本身非响应式，必须经 ref 才能
  // 让 isAuthenticated 在登录/登出后即时重算（否则 computed 会缓存初始的 false）。
  const token = ref<string | null>(getAccessToken())
  const expiresAt = ref<number>(getExpiresAt())

  function syncToken() {
    token.value = getAccessToken()
    expiresAt.value = getExpiresAt()
  }

  function setAdminRole(roleId: string) {
    if (!roleId) return
    adminRoleId.value = roleId
    localStorage.setItem(ADMIN_ROLE_KEY, roleId)
  }

  const isAuthenticated = computed(() => {
    if (!token.value) return false
    return Date.now() < expiresAt.value
  })

  async function login(username: string, password: string): Promise<boolean> {
    try {
      const result = await loginAdmin(username.trim(), password)
      syncToken()
      setAdminRole(result.roleId || 'r_super')
      return true
    } catch {
      return false
    }
  }

  async function syncSessionRole() {
    if (!isAuthenticated.value) return
    try {
      const session = await fetchAdminSession()
      if (session.roleId) setAdminRole(session.roleId)
    } catch (err) {
      if (err instanceof Error && (err as { sessionExpired?: boolean }).sessionExpired) return
      /* 其它错误忽略，沿用本地角色 */
    }
  }

  function logout() {
    logoutAdmin()
    syncToken()
    adminRoleId.value = 'r_super'
    localStorage.removeItem(ADMIN_ROLE_KEY)
  }

  return { isAuthenticated, adminRoleId, setAdminRole, login, syncSessionRole, logout }
})
