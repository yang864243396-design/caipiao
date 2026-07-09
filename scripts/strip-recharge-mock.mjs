import fs from 'fs'

const p = new URL('../client/src/views/member/RechargeView.vue', import.meta.url)
let s = fs.readFileSync(p, 'utf8')

s = s.replace(/import \{ MOCK_APPENDIX \} from '@\/mock\/appendixMock'\r?\n\r?\n/, '')

const start = s.indexOf('const MOCK_CHANNELS: RechargeChannel[] = [')
const endMarker = 'const channels = ref<RechargeChannel[]>(MOCK_CHANNELS)'
const end = s.indexOf(endMarker)
if (start < 0 || end < 0) throw new Error('MOCK_CHANNELS markers not found')
s = `${s.slice(0, start)}const channels = ref<RechargeChannel[]>([])\n\n${s.slice(end + endMarker.length)}`

s = s.replace(
  /const selectedId = ref\(MOCK_CHANNELS\[0\]\.id\)/,
  'const selectedId = ref("")',
)

const lStart = s.indexOf('/** 充提记录演示数据')
const lEndMarker = 'const ledgerRows = ref<LedgerRecord[]>([...MOCK_LEDGER])'
const lEnd = s.indexOf(lEndMarker)
if (lStart >= 0 && lEnd >= 0) {
  s = `${s.slice(0, lStart)}const ledgerRows = ref<LedgerRecord[]>([])\n${s.slice(lEnd + lEndMarker.length)}`
}

s = s.replace(
  /async function loadFundRecords\(\) \{\r?\n  if \(false\) \{ \/\* removed mock \*\/\r?\n    ledgerRows\.value = \[\.\.\.MOCK_LEDGER\]\r?\n    return\r?\n  \}\r?\n/,
  'async function loadFundRecords() {\n',
)

s = s.replace(
  /\.catch\(\(\) => \{\r?\n        \/\* 保留 MOCK_CHANNELS \*\/\r?\n      \}\)/,
  '.catch(() => {\n        ElMessage.error("加载充值渠道失败")\n      })',
)

fs.writeFileSync(p, s)
console.log('RechargeView mock stripped')
