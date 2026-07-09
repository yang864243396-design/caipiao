import { requestApi } from './client'
import type { AuditLogRow } from '@/types/audit'

interface ApiAuditEntry {
  id: string
  time: string
  actor: string
  action: string
  ip: string
}

export async function fetchAuditLogs(limit = 100): Promise<AuditLogRow[]> {
  const res = await requestApi<{ items: ApiAuditEntry[] }>(
    `/admin/system/audit-logs?limit=${encodeURIComponent(String(limit))}`,
  )
  return res.items.map((row) => ({
    id: row.id,
    time: row.time,
    actor: row.actor,
    action: row.action,
    ip: row.ip,
  }))
}
