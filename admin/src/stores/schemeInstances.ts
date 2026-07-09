import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type {
  SchemeBetExecutionRow,
  SchemeBetRecordRow,
  SchemeChangeLogRow,
  SchemeCustomSettings,
  SchemeDrawHistoryRow,
  SchemeInstanceRow,
  SchemeInstanceStatus,
  SchemePlanTrendRow,
  SchemeShareSnapshotRow,
} from '@/types/schemes'
import { useAuditLogStore } from '@/stores/auditLog'
import {
  createShareSnapshot as apiCreateShareSnapshot,
  deleteShareSnapshot as apiDeleteShareSnapshot,
  fetchSchemeMonitorInstances,
  fetchSchemeShareSnapshots,
  forceStopSchemeInstance,
  patchShareSnapshot as apiPatchShareSnapshot,
  releaseStopSchemeInstance,
} from '@/api/schemes'
import type { CreateShareSnapshotInput, SchemeMonitorQuery, SchemeShareQuery } from '@/api/schemes'

export type {
  SchemeInstanceRow,
  SchemeChangeLogRow,
  SchemeShareSnapshotRow,
  SchemeBetExecutionRow,
  SchemeBetRecordRow,
  SchemeDrawHistoryRow,
  SchemePlanTrendRow,
  SchemeCustomSettings,
  SchemeInstanceStatus,
}

export interface SchemeBetStats {
  betCount: number
  winCount: number
  winRate: number
}

function inDateRange(iso: string, start?: string, end?: string): boolean {
  const t = new Date(iso).getTime()
  if (start && t < new Date(start).getTime()) return false
  if (end) {
    const endDate = new Date(end)
    endDate.setHours(23, 59, 59, 999)
    if (t > endDate.getTime()) return false
  }
  return true
}

export const useSchemeInstancesStore = defineStore('schemeInstances', () => {
  const list = ref<SchemeInstanceRow[]>([])
  const shareSnapshots = ref<SchemeShareSnapshotRow[]>([])
  const betExecutions = ref<SchemeBetExecutionRow[]>([])
  const planTrends = ref<SchemePlanTrendRow[]>([])
  const drawHistory = ref<SchemeDrawHistoryRow[]>([])
  const betRecords = ref<SchemeBetRecordRow[]>([])
  const changeLogs = ref<SchemeChangeLogRow[]>([])
  const hydrated = ref(false)
  const loading = ref(false)
  const shareLoading = ref(false)
  const lastUserQuery = ref<SchemeMonitorQuery>({})
  const lastShareQuery = ref<SchemeShareQuery>({})

  async function loadUserList(query: SchemeMonitorQuery = {}) {
    loading.value = true
    try {
      lastUserQuery.value = { ...query }
      list.value = await fetchSchemeMonitorInstances(query)
    } finally {
      loading.value = false
    }
  }

  async function loadShareList(query: SchemeShareQuery = {}) {
    shareLoading.value = true
    try {
      lastShareQuery.value = { ...query }
      shareSnapshots.value = await fetchSchemeShareSnapshots(query)
    } finally {
      shareLoading.value = false
    }
  }

  async function hydrate() {
    if (hydrated.value) return
    await loadShareList({})
    hydrated.value = true
  }

  async function reload() {
    await Promise.all([
      loadShareList(lastShareQuery.value),
      loadUserList(lastUserQuery.value),
    ])
  }

  function getById(id: string) {
    return list.value.find((s) => s.id === id)
  }

  function getShareById(id: string) {
    return shareSnapshots.value.find((s) => s.id === id)
  }

  function forMember(memberId: string) {
    return list.value.filter((s) => s.memberId === memberId)
  }

  function betExecutionsForScheme(schemeInstanceId: string) {
    return betExecutions.value.filter((r) => r.schemeInstanceId === schemeInstanceId)
  }

  function planTrendForScheme(schemeInstanceId: string) {
    return planTrends.value.filter((r) => r.schemeInstanceId === schemeInstanceId)
  }

  function drawHistoryForScheme(schemeInstanceId: string) {
    return drawHistory.value.filter((r) => r.schemeInstanceId === schemeInstanceId)
  }

  function betRecordsForScheme(schemeInstanceId: string) {
    return betRecords.value.filter((r) => r.schemeInstanceId === schemeInstanceId)
  }

  function changeLogsForScheme(schemeInstanceId: string) {
    return changeLogs.value
      .filter((r) => r.schemeInstanceId === schemeInstanceId)
      .sort((a, b) => b.changedAt.localeCompare(a.changedAt))
  }

  function betStatsInRange(schemeInstanceId: string, start?: string, end?: string): SchemeBetStats {
    const rows = betExecutionsForScheme(schemeInstanceId).filter((r) => inDateRange(r.betAt, start, end))
    const betCount = rows.length
    const winCount = rows.filter((r) => r.win).length
    const winRate = betCount > 0 ? Math.round((winCount / betCount) * 1000) / 10 : 0
    return { betCount, winCount, winRate }
  }

  const runningCount = computed(() => list.value.filter((s) => s.status === '运行中').length)
  const runningFormalCount = computed(
    () => list.value.filter((s) => s.status === '运行中' && !s.simBet).length,
  )
  const runningSimulatedCount = computed(
    () => list.value.filter((s) => s.status === '运行中' && s.simBet).length,
  )
  const platformPnlYuan = computed(() => {
    const memberPnl = betRecords.value
      .filter((r) => r.status === '已结算')
      .reduce((sum, r) => sum + r.profitLoss, 0)
    return Math.round(-memberPnl * 100) / 100
  })

  async function softStop(id: string): Promise<boolean> {
    const row = getById(id)
    if (!row || row.status !== '运行中') return false
    await forceStopSchemeInstance(id)
    row.status = '已封停'
    row.updatedAt = new Date().toISOString()
    useAuditLogStore().append(`强停方案实例 ${id}（${row.memberName} · ${row.kind} · ${row.refId}）`)
    return true
  }

  async function releaseStop(id: string): Promise<boolean> {
    const row = getById(id)
    if (!row || row.status !== '已封停') return false
    await releaseStopSchemeInstance(id)
    row.status = '已暂停'
    row.updatedAt = new Date().toISOString()
    useAuditLogStore().append(`解封方案实例 ${id} → 已暂停`)
    return true
  }

  async function createShareSnapshot(input: CreateShareSnapshotInput) {
    shareLoading.value = true
    try {
      const row = await apiCreateShareSnapshot(input)
      await loadShareList(lastShareQuery.value)
      useAuditLogStore().append(`新建分享池快照 ${row.id}（${row.schemeName}）`)
      return row
    } finally {
      shareLoading.value = false
    }
  }

  async function updateShareSnapshot(id: string, input: CreateShareSnapshotInput): Promise<boolean> {
    shareLoading.value = true
    try {
      const row = await apiPatchShareSnapshot(id, input)
      const idx = shareSnapshots.value.findIndex((s) => s.id === id)
      if (idx >= 0) shareSnapshots.value[idx] = row
      useAuditLogStore().append(`更新分享池快照 ${id}（${row.schemeName}）`)
      return true
    } finally {
      shareLoading.value = false
    }
  }

  async function deleteShareSnapshot(id: string): Promise<boolean> {
    await apiDeleteShareSnapshot(id)
    const before = shareSnapshots.value.length
    shareSnapshots.value = shareSnapshots.value.filter((s) => s.id !== id)
    if (shareSnapshots.value.length === before) return false
    useAuditLogStore().append(`删除分享池快照 ${id}`)
    return true
  }

  return {
    list,
    shareSnapshots,
    betExecutions,
    planTrends,
    drawHistory,
    betRecords,
    changeLogs,
    getById,
    getShareById,
    forMember,
    betExecutionsForScheme,
    planTrendForScheme,
    drawHistoryForScheme,
    betRecordsForScheme,
    changeLogsForScheme,
    betStatsInRange,
    runningCount,
    runningFormalCount,
    runningSimulatedCount,
    platformPnlYuan,
    hydrated,
    loading,
    shareLoading,
    lastUserQuery,
    lastShareQuery,
    hydrate,
    loadUserList,
    loadShareList,
    reload,
    createShareSnapshot,
    softStop,
    releaseStop,
    updateShareSnapshot,
    deleteShareSnapshot,
  }
})
