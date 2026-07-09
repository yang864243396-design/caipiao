/** 方案类型与实例状态（Mock / API 共用，避免 api 层依赖 views/mock） */

export type ClientSchemeKind = 'custom' | 'contrary' | 'follow'

export type ClientSchemeInstanceStatus = 'pending' | 'running' | 'paused' | 'soft_stopped'

export function parseSchemeKind(raw: unknown): ClientSchemeKind {
  const s = String(raw ?? '').trim().toLowerCase()
  if (s === 'contrary' || s === '反买') return 'contrary'
  if (s === 'follow' || s === '跟单') return 'follow'
  return 'custom'
}

export function schemeKindLabel(kind: ClientSchemeKind): string {
  if (kind === 'contrary') return '反买'
  if (kind === 'follow') return '跟单'
  return '自创'
}

export function instanceStatusLabel(status: ClientSchemeInstanceStatus): string {
  const map: Record<ClientSchemeInstanceStatus, string> = {
    pending: '等待开启',
    running: '运行中',
    paused: '已暂停',
    soft_stopped: '已封停',
  }
  return map[status]
}
