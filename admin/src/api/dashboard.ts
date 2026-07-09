import { requestApi } from './client'

export interface DashboardKpi {
  todayRecharge: number
  todayWithdraw: number
  pendingWithdrawCount: number
  todayBetVolume: number
  memberTotalPnl: number
  runningSchemesReal: number
  runningSchemesSim: number
  registrationsLast7Days: number
}

export async function fetchDashboardKpi(): Promise<DashboardKpi> {
  return requestApi<DashboardKpi>('/admin/dashboard/kpi')
}
