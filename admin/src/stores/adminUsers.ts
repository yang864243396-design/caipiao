import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  createAdminUser,
  deleteAdminUser,
  fetchAdminUsers,
  updateAdminUser,
  type AdminUser,
  type AdminUserSaveInput,
} from '@/api/adminUsers'

export type { AdminUser, AdminUserSaveInput }

export const useAdminUsersStore = defineStore('adminUsers', () => {
  const users = ref<AdminUser[]>([])
  const loading = ref(false)
  const hydrated = ref(false)

  async function hydrate() {
    if (hydrated.value) return
    loading.value = true
    try {
      users.value = await fetchAdminUsers()
      hydrated.value = true
    } finally {
      loading.value = false
    }
  }

  async function createUser(input: AdminUserSaveInput) {
    const saved = await createAdminUser(input)
    users.value.push(saved)
    return saved
  }

  async function saveUser(id: number, input: AdminUserSaveInput) {
    const saved = await updateAdminUser(id, input)
    const i = users.value.findIndex((u) => u.id === id)
    if (i >= 0) users.value[i] = saved
    return saved
  }

  async function removeUser(id: number): Promise<boolean> {
    const row = users.value.find((u) => u.id === id)
    if (row?.account === 'admin') return false
    await deleteAdminUser(id)
    users.value = users.value.filter((u) => u.id !== id)
    return true
  }

  return { users, loading, hydrated, hydrate, createUser, saveUser, removeUser }
})
