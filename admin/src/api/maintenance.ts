import { requestApi } from './client'

export interface MaintenanceState {
  enabled: boolean
  popupAnnouncementId?: string
  title?: string
  message?: string
}

export async function fetchMaintenance(): Promise<MaintenanceState> {
  return requestApi<MaintenanceState>('/admin/operations/maintenance')
}

export async function saveMaintenance(state: MaintenanceState): Promise<MaintenanceState> {
  return requestApi<MaintenanceState>('/admin/operations/maintenance', { method: 'PUT', body: state })
}
