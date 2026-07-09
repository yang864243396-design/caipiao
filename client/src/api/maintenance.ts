import { requestApi } from '@/api/client'



export interface PublicMaintenanceAnnouncement {

  id: string

  title: string

  bodyHtml: string

}



export interface PublicMaintenanceState {

  enabled: boolean

  popupAnnouncementId?: string

  title?: string

  message?: string

  popupAnnouncement?: PublicMaintenanceAnnouncement | null

}



export async function fetchPublicMaintenance(): Promise<PublicMaintenanceState> {

  return requestApi<PublicMaintenanceState>('/public/maintenance', { auth: false })

}

