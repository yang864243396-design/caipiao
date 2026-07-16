/**
 * 冒烟：倍投 Tab 玩法分类 + P2 小白/一键算法
 * 用法（在 client 目录）：node scripts/smoke-bet-multiplier-tabs.mjs
 */
import { execFileSync } from 'child_process'
import path from 'path'
import { fileURLToPath, pathToFileURL } from 'url'
import fs from 'fs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const clientRoot = path.resolve(__dirname, '..')
const esbuildBin = path.join(clientRoot, 'node_modules/esbuild/bin/esbuild')

function bundle(srcRel, outName) {
  const src = path.join(clientRoot, srcRel)
  const out = path.join(clientRoot, 'src/utils', outName)
  execFileSync(
    process.execPath,
    [esbuildBin, src, '--bundle', '--platform=node', '--format=esm', `--outfile=${out}`],
    { stdio: 'inherit' },
  )
  return out
}

const catOut = bundle('src/utils/betMultiplierPlayCategory.ts', '.smoke-bet-cat.mjs')
const planOut = bundle('src/utils/betMultiplierPlan.ts', '.smoke-bet-plan.mjs')

const mod = await import(pathToFileURL(catOut).href + `?t=${Date.now()}`)
const {
  resolveBetMultiplierPlayCategory,
  showAutoGenBetMultiplierTabs,
  normalizeBetMultiplierPersistKind,
} = mod

const plan = await import(pathToFileURL(planOut).href + `?t=${Date.now()}`)
const {
  buildNewbieTimesList,
  buildOneclickTimesList,
  DEFAULT_SIDES_PRESET,
  AGGRESSIVE_PRESET,
  canGenerateOneclickPlan,
} = plan

const cases = [
  { name: '定位胆', ctx: { betMode: 'dingwei', playTypeId: 'dingwei' }, four: true, cat: 'locate' },
  { name: '龙虎', ctx: { betMode: 'longhu', playTypeLabel: '龙虎' }, four: true, cat: 'sides' },
  { name: '定胆位标签', ctx: { subPlayLabel: '定位胆万位', playTypeLabel: '一星' }, four: true, cat: 'locate' },
  {
    name: '三星复式',
    ctx: { playTypeId: 'qian3', playTypeLabel: '三星', betMode: 'fushi', segmentLen: 3 },
    four: false,
    cat: 'multi_star',
  },
  {
    name: '二星',
    ctx: { playTypeLabel: '二星', playTypeId: 'qian2', betMode: 'fushi', segmentLen: 2 },
    four: false,
    cat: 'multi_star',
  },
  {
    name: '前二大小单双',
    ctx: { playTypeLabel: '前二', subPlayLabel: '大小单双', betMode: 'dxds' },
    four: true,
    cat: 'sides',
  },
  { name: '和值', ctx: { betMode: 'hezhi' }, four: false, cat: 'sum_span' },
  { name: '组三', ctx: { betMode: 'zu3' }, four: false, cat: 'combo_group' },
  {
    name: '任选一中一',
    ctx: { playTypeLabel: '任选', subPlayLabel: '任选一中一' },
    four: true,
    cat: 'locate_like',
  },
  {
    name: '猜前三',
    ctx: { playTemplate: 'pk10_std', playTypeId: 'qian3', subPlayLabel: '猜前三', segmentLen: 3 },
    four: false,
    cat: 'pk10_multi',
  },
]

let failed = 0
for (const c of cases) {
  const cat = resolveBetMultiplierPlayCategory(c.ctx)
  const four = showAutoGenBetMultiplierTabs(c.ctx)
  const ok = cat === c.cat && four === c.four
  console.log(
    `${ok ? 'PASS' : 'FAIL'} ${c.name}: cat=${cat} (want ${c.cat}), fourTab=${four} (want ${c.four})`,
  )
  if (!ok) failed++
}

for (const [inK, want] of [
  ['0', '2'],
  ['1', '2'],
  ['2', '2'],
  ['3', '3'],
]) {
  const got = normalizeBetMultiplierPersistKind(inK)
  const ok = got === want
  console.log(`${ok ? 'PASS' : 'FAIL'} persistKind ${inK} → ${got} (want ${want})`)
  if (!ok) failed++
}

// —— P2 算法验收 ——
function eqArr(a, b) {
  return Array.isArray(a) && Array.isArray(b) && a.length === b.length && a.every((v, i) => v === b[i])
}

{
  const got = buildNewbieTimesList({
    odds: 1.9,
    firstBet: 2,
    targetProfit: 1,
    cycle: 10,
    money: 1,
    number: 1,
  })
  const want = [...DEFAULT_SIDES_PRESET]
  const ok = eqArr(got, want)
  console.log(`${ok ? 'PASS' : 'FAIL'} newbie default sides: ${JSON.stringify(got)}`)
  if (!ok) failed++
}

{
  const got = buildNewbieTimesList({
    odds: 1.93,
    firstBet: 6,
    targetProfit: 5,
    cycle: 10,
    money: 1,
    number: 1,
  })
  const want = [...AGGRESSIVE_PRESET]
  const ok = eqArr(got, want)
  console.log(`${ok ? 'PASS' : 'FAIL'} newbie aggressive: ${JSON.stringify(got)}`)
  if (!ok) failed++
}

{
  const list = buildOneclickTimesList({
    money: 1,
    number: 1,
    mode: 9.8,
    cycle: 5,
    calcType: 'fixed',
    targetProfit: 2,
  })
  let ok = Array.isArray(list) && list.length === 5 && list.every((t, i) => t >= 1 && (i === 0 || t >= list[i - 1]))
  if (ok) {
    let prev = 0
    for (const t of list) {
      const output = 1 * 1 * t
      const total = prev + output
      const gain = 9.8 * t - total
      if (gain < 2) {
        ok = false
        break
      }
      prev = total
    }
  }
  console.log(`${ok ? 'PASS' : 'FAIL'} oneclick fixed profit search: ${JSON.stringify(list)}`)
  if (!ok) failed++
}

{
  const okList = buildOneclickTimesList({
    money: 1,
    number: 1,
    mode: 9.8,
    cycle: 3,
    calcType: 'free',
    freeList: [1, 2, 4],
  })
  const badErr = canGenerateOneclickPlan({
    money: '1',
    number: '1',
    mode: '9.8',
    cycle: '3',
    calcType: 'free',
    targetRate: '',
    targetProfit: '',
    sumBegin: '',
    sumStep: '',
    freeList: '1,2',
  })
  const ok = eqArr(okList, [1, 2, 4]) && badErr === '倍数的个数和周期不一致'
  console.log(`${ok ? 'PASS' : 'FAIL'} oneclick free list validation`)
  if (!ok) failed++
}

for (const f of [catOut, planOut]) {
  try {
    fs.unlinkSync(f)
  } catch {
    /* ignore */
  }
}

console.log(failed === 0 ? '\nSMOKE OK' : `\nSMOKE FAILED: ${failed}`)
process.exit(failed === 0 ? 0 : 1)
