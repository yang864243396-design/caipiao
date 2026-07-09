import { MOCK_APPENDIX } from './appendixMock'

/** 方案模板（管理后台维护 → 客户端倍投设定 · 高级倍投列表） */
export interface SchemeTemplateRow {
  id: string
  name: string
  lotteryCode: string
  lotteryLabel: string
  brief?: string
  sortOrder: number
  enabled: boolean
  memberOwned?: boolean
  definitionId?: string
  config?: { rounds?: unknown }
  createdAt: string
  updatedAt: string
}

export const SCHEME_TEMPLATE_LOTTERIES = [
  { code: 'tron_ffc_1m', label: '波场1分彩' },
  { code: 'hash_ffc_1m', label: '哈希1分彩' },
  { code: 'eth_ffc_1m', label: '以太坊1分彩' },
  { code: 'bnb_ffc_1m', label: '币安1分彩' },
  { code: 'tron_jisu', label: '波场极速彩' },
  { code: 'tron_syxw', label: '波场11选5' },
] as const

export const STORAGE_SCHEME_TEMPLATES = 'mock_scheme_template_library'
const LEGACY_SCHEME_TEMPLATES = 'admin_mock_scheme_template_library'

function nowIso() {
  return new Date().toISOString()
}

export function defaultSchemeTemplates(): SchemeTemplateRow[] {
  const ts = nowIso()
  return [
    {
      id: MOCK_APPENDIX.schemeAdvancedId,
      name: '两期中跟挂停（附录演示）',
      lotteryCode: 'tron_ffc_1m',
      lotteryLabel: '波场1分彩',
      brief: '平台预置演示模板',
      sortOrder: 10,
      enabled: true,
      createdAt: ts,
      updatedAt: ts,
    },
    {
      id: 'tpl_demo_wave_3',
      name: '三期推波方案',
      lotteryCode: 'hash_ffc_1m',
      lotteryLabel: '哈希1分彩',
      brief: '三期推波结构示例',
      sortOrder: 20,
      enabled: true,
      createdAt: ts,
      updatedAt: ts,
    },
    {
      id: 'tpl_demo_plan_4',
      name: '四期倍投计划',
      lotteryCode: 'bnb_ffc_1m',
      lotteryLabel: '币安1分彩',
      brief: '四期计划表示例',
      sortOrder: 30,
      enabled: true,
      createdAt: ts,
      updatedAt: ts,
    },
    {
      id: 'tpl_demo_plan_6',
      name: '六期倍投方案',
      lotteryCode: 'tron_ffc_1m',
      lotteryLabel: '波场1分彩',
      brief: '六期倍投表示例',
      sortOrder: 40,
      enabled: true,
      createdAt: ts,
      updatedAt: ts,
    },
  ]
}

function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null
  const m = document.cookie.match(
    new RegExp(`(?:^|; )${encodeURIComponent(name).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}=([^;]*)`),
  )
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
    /* ignore */
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

function normalizeTemplates(rows: SchemeTemplateRow[]): SchemeTemplateRow[] {
  return [...rows]
    .sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name))
    .map((r) => ({ ...r }))
}

function parseTemplates(raw: string): SchemeTemplateRow[] | null {
  try {
    const parsed = JSON.parse(raw) as SchemeTemplateRow[]
    if (!Array.isArray(parsed)) return null
    return normalizeTemplates(parsed.filter((r) => r?.id && r?.name))
  } catch {
    return null
  }
}

export function readSchemeTemplates(): SchemeTemplateRow[] {
  const fromCookie = readCookie(STORAGE_SCHEME_TEMPLATES)
  if (fromCookie) {
    const parsed = parseTemplates(fromCookie)
    if (parsed?.length) return parsed
  }
  const raw = readStorage(STORAGE_SCHEME_TEMPLATES, LEGACY_SCHEME_TEMPLATES)
  if (raw) {
    const parsed = parseTemplates(raw)
    if (parsed?.length) return parsed
  }
  return defaultSchemeTemplates()
}

export function writeSchemeTemplates(rows: SchemeTemplateRow[]) {
  const payload = JSON.stringify(normalizeTemplates(rows))
  writeStorage(STORAGE_SCHEME_TEMPLATES, payload)
  writeStorage(LEGACY_SCHEME_TEMPLATES, payload)
  writeCookie(STORAGE_SCHEME_TEMPLATES, payload)
}

export function enabledSchemeTemplates(rows?: SchemeTemplateRow[]): SchemeTemplateRow[] {
  return (rows ?? readSchemeTemplates()).filter((r) => r.enabled)
}

export function subscribeSchemeTemplatesSync(onChange: () => void): () => void {
  if (typeof window === 'undefined') return () => {}

  const onStorage = (e: StorageEvent) => {
    if (e.key === STORAGE_SCHEME_TEMPLATES || e.key === LEGACY_SCHEME_TEMPLATES) {
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

export function newTemplateId() {
  return `tpl_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 5)}`
}

export function lotteryLabelForCode(code: string): string {
  return SCHEME_TEMPLATE_LOTTERIES.find((l) => l.code === code)?.label ?? code
}
