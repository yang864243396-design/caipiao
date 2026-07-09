import fs from 'fs'
import path from 'path'

function walk(dir, acc = []) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name)
    if (e.isDirectory()) walk(p, acc)
    else if (/\.(ts|vue)$/.test(e.name)) acc.push(p)
  }
  return acc
}

const roots = [
  'admin/src/api',
  'admin/src/stores',
  'admin/src/composables',
  'admin/src/views',
  'admin/src/utils',
  'client/src/composables',
  'client/src/views',
]

for (const root of roots) {
  const full = path.join(process.cwd(), root)
  if (!fs.existsSync(full)) continue
  for (const f of walk(full)) {
    let s = fs.readFileSync(f, 'utf8')
    const orig = s
    s = s.replace(/import \{ USE_MOCK \} from ['"]@\/api\/config['"]\r?\n/g, '')
    s = s.replace(/import \{ USE_MOCK \} from ['"]\.\/config['"]\r?\n/g, '')
    s = s.replace(/import \{ USE_MOCK, WS_ENABLED, WS_PUBLIC_BASE \} from ['"]@\/api\/config['"]/g,
      "import { WS_ENABLED, WS_PUBLIC_BASE } from '@/api/config'")
    s = s.replace(/import \{ USE_MOCK, WS_ADMIN_ENABLED \} from ['"]@\/api\/config['"]/g,
      "import { WS_ADMIN_ENABLED } from '@/api/config'")
    s = s.replace(/  if \(USE_MOCK\) return row\r?\n/g, '')
    s = s.replace(/  if \(USE_MOCK\) return\r?\n/g, '')
    s = s.replace(/  if \(USE_MOCK\) return \[\]\r?\n/g, '')
    s = s.replace(/  if \(USE_MOCK\) throw new Error\([^)]+\)\r?\n/g, '')
    s = s.replace(/  if \(USE_MOCK \|\| hydrated\.value\) return\r?\n    /g, '  if (hydrated.value) return\n    ')
    s = s.replace(/  if \(USE_MOCK \|\| hydrated\.value\) return\r?\n/g, '  if (hydrated.value) return\n')
    s = s.replace(/    if \(USE_MOCK\) return\r?\n/g, '')
    s = s.replace(/    if \(USE_MOCK \|\| hydrated\.value\) return\r?\n/g, '    if (hydrated.value) return\n')
    s = s.replace(/    if \(!USE_MOCK\) /g, '    ')
    s = s.replace(/  if \(!USE_MOCK\) \{/g, '  {')
    s = s.replace(/  if \(USE_MOCK\) \{/g, '  if (false) { /* removed mock */')
    s = s.replace(/    if \(USE_MOCK\) \{/g, '    if (false) { /* removed mock */')
    s = s.replace(/const saved = USE_MOCK \? [^:]+ : /g, 'const saved = ')
    s = s.replace(/v-if="USE_MOCK"/g, 'v-if="false"')
    s = s.replace(/@\/mock\/contentTypes/g, '@/types/content')
    s = s.replace(/@\/mock\/memberLedgerSeed/g, '@/types/members')
    s = s.replace(/@\/mock\/withdrawSeed/g, '@/types/funds')
    s = s.replace(/@\/mock\/auditLogSeed/g, '@/types/audit')
    s = s.replace(/@\/mock\/lotteryCatalogSeed/g, '@/types/lottery')
    s = s.replace(/@\/mock\/schemeCustomOptions/g, '@/types/schemes')
    s = s.replace(/@\/mock\/schemeInstancesSeed/g, '@/types/schemes')
    s = s.replace(/@\/mock\/schemeShareSnapshotsSeed/g, '@/types/schemes')
    s = s.replace(/@\/mock\/schemeHistorySeed/g, '@/types/schemes')
    if (s !== orig) {
      fs.writeFileSync(f, s)
      console.log('updated', f)
    }
  }
}
