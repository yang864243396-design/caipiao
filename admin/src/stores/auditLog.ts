import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { AuditLogRow } from '@/types/audit'
import { fetchAuditLogs } from '@/api/audit'

export type { AuditLogRow }

/** 操作审计：列表来自服务端；写操作由 API 留痕 */
export const useAuditLogStore = defineStore('auditLog', () => {
  const list = ref<AuditLogRow[]>([])
  const hydrated = ref(false)

  async function hydrate() {
    if (hydrated.value) return
    list.value = await fetchAuditLogs(100)
    hydrated.value = true
  }

  /** 本地乐观追加已废弃，审计由服务端写入 */
  function append(_action: string, _actor = 'admin', _ip = '127.0.0.1') {
    void _action
    void _actor
    void _ip
  }

  return { list, hydrated, hydrate, append }
})
