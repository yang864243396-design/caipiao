import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchMaintenance, saveMaintenance } from '@/api/maintenance'

/** 维护开关 + 维护时置顶公告（数据来自后端） */
export const useMaintenanceOpsStore = defineStore('maintenanceOps', () => {
  const maintenanceOn = ref(false)
  const popupAnnouncementId = ref('')
  const hydrated = ref(false)
  const loading = ref(false)
  const saving = ref(false)

  async function hydrate() {
    if (hydrated.value) return
    loading.value = true
    try {
      const state = await fetchMaintenance()
      maintenanceOn.value = state.enabled
      popupAnnouncementId.value = state.popupAnnouncementId ?? ''
      hydrated.value = true
    } finally {
      loading.value = false
    }
  }

  async function persist() {
    saving.value = true
    try {
      const saved = await saveMaintenance({
        enabled: maintenanceOn.value,
        popupAnnouncementId: popupAnnouncementId.value || undefined,
      })
      maintenanceOn.value = saved.enabled
      popupAnnouncementId.value = saved.popupAnnouncementId ?? ''
    } finally {
      saving.value = false
    }
  }

  function syncFromStorage() {
    void hydrate()
  }

  return {
    maintenanceOn,
    popupAnnouncementId,
    loading,
    saving,
    hydrated,
    hydrate,
    persist,
    syncFromStorage,
  }
})
