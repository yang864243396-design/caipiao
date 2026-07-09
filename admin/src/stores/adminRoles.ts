import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  deleteAdminRole,
  fetchAdminRoles,
  saveAdminRole,
  type AdminRole,
} from '@/api/adminRoles'

export type { AdminRole }

export const useAdminRolesStore = defineStore('adminRoles', () => {
  const roles = ref<AdminRole[]>([])
  const loading = ref(false)
  const hydrated = ref(false)

  async function hydrate() {
    if (hydrated.value) return
    loading.value = true
    try {
      roles.value = await fetchAdminRoles()
      hydrated.value = true
    } finally {
      loading.value = false
    }
  }

  async function upsertRole(row: AdminRole) {
    const saved = await saveAdminRole(row)
    const i = roles.value.findIndex((r) => r.id === saved.id)
    if (i >= 0) roles.value[i] = saved
    else roles.value.push(saved)
    return saved
  }

  async function removeRole(id: string): Promise<boolean> {
    if (id === 'r_super') return false
    await deleteAdminRole(id)
    roles.value = roles.value.filter((r) => r.id !== id)
    return true
  }

  return { roles, loading, hydrated, hydrate, upsertRole, removeRole }
})
