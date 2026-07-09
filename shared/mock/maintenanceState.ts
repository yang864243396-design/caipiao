/** localStorage / Cookie 键名（双端 Mock 同源；Cookie 便于 localhost 跨端口 dev） */
export const STORAGE_MAINT = 'mock_maintenance_on'
export const STORAGE_POPUP = 'mock_maintenance_popup_announcement_id'

/** 兼容 admin 早期键名 */
const LEGACY_MAINT = 'admin_mock_maintenance_on'
const LEGACY_POPUP = 'admin_mock_maintenance_popup_announcement_id'

function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null
  const m = document.cookie.match(new RegExp(`(?:^|; )${encodeURIComponent(name).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}=([^;]*)`))
  return m ? decodeURIComponent(m[1]) : null
}

function writeCookie(name: string, value: string, maxAgeSec = 7 * 86400) {
  if (typeof document === 'undefined') return
  document.cookie = `${encodeURIComponent(name)}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSec}`
}

function readStorage(key: string, legacy?: string): string | null {
  try {
    const v = localStorage.getItem(key)
    if (v !== null) return v
    if (legacy) return localStorage.getItem(legacy)
  } catch {
    /* SSR / 隐私模式 */
  }
  return null
}

function writeStorage(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* ignore */
  }
}

export function readMaintenanceOn(): boolean {
  const fromCookie = readCookie(STORAGE_MAINT)
  if (fromCookie !== null) return fromCookie === '1'
  const raw = readStorage(STORAGE_MAINT, LEGACY_MAINT)
  return raw === '1'
}

export function writeMaintenanceOn(on: boolean) {
  const v = on ? '1' : '0'
  writeStorage(STORAGE_MAINT, v)
  writeStorage(LEGACY_MAINT, v)
  writeCookie(STORAGE_MAINT, v)
}

export function readPopupAnnouncementId(): string {
  const fromCookie = readCookie(STORAGE_POPUP)
  if (fromCookie !== null) return fromCookie
  return readStorage(STORAGE_POPUP, LEGACY_POPUP) ?? ''
}

export function writePopupAnnouncementId(id: string) {
  writeStorage(STORAGE_POPUP, id)
  writeStorage(LEGACY_POPUP, id)
  writeCookie(STORAGE_POPUP, id)
}

/** 轮询 + storage 事件：admin/client 跨 Tab、跨端口（Cookie）同步 */
export function subscribeMaintenanceSync(onChange: () => void): () => void {
  if (typeof window === 'undefined') return () => {}

  const onStorage = (e: StorageEvent) => {
    if (
      e.key === STORAGE_MAINT ||
      e.key === STORAGE_POPUP ||
      e.key === LEGACY_MAINT ||
      e.key === LEGACY_POPUP
    ) {
      onChange()
    }
  }
  window.addEventListener('storage', onStorage)
  const timer = window.setInterval(onChange, 2000)

  return () => {
    window.removeEventListener('storage', onStorage)
    window.clearInterval(timer)
  }
}
