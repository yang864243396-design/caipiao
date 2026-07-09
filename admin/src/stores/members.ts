import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  fetchMemberDetail,
  fetchMemberFundRecords,
  fetchMembers,
  type AdminMemberRow,
  type FetchMemberFundRecordsParams,
} from '@/api/members'
import type { MemberFundRecordRow } from '@/types/members'

export type MemberRow = AdminMemberRow
export type { MemberFundRecordRow }

export const useMembersStore = defineStore('members', () => {
  const list = ref<AdminMemberRow[]>([])
  const total = ref(0)
  const detailCache = ref<Record<string, AdminMemberRow>>({})
  const loading = ref(false)

  function getById(id: string) {
    return detailCache.value[id]
  }

  async function loadList(params: {
    keyword?: string
    searchField?: 'account' | 'guajiAccount' | 'id'
    page?: number
    pageSize?: number
  }) {
    loading.value = true
    try {
      const res = await fetchMembers(params)
      list.value = res.items
      total.value = res.total
    } finally {
      loading.value = false
    }
  }

  async function loadDetail(memberId: string) {
    const row = await fetchMemberDetail(memberId)
    detailCache.value[memberId] = row
    return row
  }

  async function loadFundRecords(memberId: string, params: FetchMemberFundRecordsParams) {
    return fetchMemberFundRecords(memberId, params)
  }

  return {
    list,
    total,
    loading,
    getById,
    loadList,
    loadDetail,
    loadFundRecords,
  }
})
