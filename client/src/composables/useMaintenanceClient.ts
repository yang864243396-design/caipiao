import { computed, ref } from 'vue'
import { WS_ENABLED, WS_PUBLIC_BASE } from '@/api/config'
import { fetchPublicMaintenance, type PublicMaintenanceAnnouncement } from '@/api/maintenance'
import { connectPublicWs, parseMaintenanceChanged } from '@/composables/ws/usePublicWs'
import type { WsMaintenanceChangedPayload } from '@shared/types/ws'

const maintenanceOn = ref(false)
const popupAnnouncementId = ref('')
const apiPopupAnnouncement = ref<PublicMaintenanceAnnouncement | null>(null)

let stopSync: (() => void) | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

function applyMaintenancePayload(payload: WsMaintenanceChangedPayload) {
  maintenanceOn.value = payload.enabled
  popupAnnouncementId.value = payload.popupAnnouncementId ?? ''
  apiPopupAnnouncement.value = payload.popupAnnouncement ?? null
}

async function refreshFromApi() {
  try {
    const state = await fetchPublicMaintenance()
    maintenanceOn.value = state.enabled
    popupAnnouncementId.value = state.popupAnnouncementId ?? ''
    apiPopupAnnouncement.value = state.popupAnnouncement ?? null
  } catch {
    /* 网络失败时保留上次状态 */
  }
}

export function refreshMaintenanceClientState() {
  void refreshFromApi()
}

export function startMaintenanceClientSync() {
  if (stopSync) return
  refreshMaintenanceClientState()

  const stopPoll = () => {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }
  const startPoll = () => {
    stopPoll()
    pollTimer = window.setInterval(() => {
      void refreshFromApi()
    }, 15_000)
  }

  if (WS_ENABLED && WS_PUBLIC_BASE) {
    const stopWs = connectPublicWs(WS_PUBLIC_BASE, (env) => {
      const payload = parseMaintenanceChanged(env)
      if (payload) applyMaintenancePayload(payload)
    })
    startPoll()
    stopSync = () => {
      stopWs()
      stopPoll()
    }
    return
  }

  startPoll()
  stopSync = stopPoll
}

export function stopMaintenanceClientSync() {
  stopSync?.()
  stopSync = null
}

export function useMaintenanceClient() {
  const popupAnnouncement = computed(() => {
    if (!apiPopupAnnouncement.value) return null
    return {
      id: apiPopupAnnouncement.value.id,
      title: apiPopupAnnouncement.value.title,
      status: '已发布' as const,
      publishedAt: null,
      bodyHtml: apiPopupAnnouncement.value.bodyHtml,
    }
  })

  const shouldBlockLobby = computed(() => maintenanceOn.value)

  const shouldShowMaintenancePopup = computed(
    () => maintenanceOn.value && popupAnnouncement.value !== null,
  )

  return {
    maintenanceOn,
    popupAnnouncementId,
    popupAnnouncement,
    shouldBlockLobby,
    shouldShowMaintenancePopup,
    refresh: refreshMaintenanceClientState,
    startSync: startMaintenanceClientSync,
    stopSync: stopMaintenanceClientSync,
  }
}
