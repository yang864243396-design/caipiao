/**
 * 冒烟：计划反集 — 前端 Tab 门禁 + 后端补集算法
 * 用法（在 client 目录）：node scripts/smoke-plan-contrary.mjs
 */
import { execFileSync, spawnSync } from 'child_process'
import path from 'path'
import { fileURLToPath, pathToFileURL } from 'url'
import fs from 'fs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const clientRoot = path.resolve(__dirname, '..')
const repoRoot = path.resolve(clientRoot, '..')
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

const out = bundle('src/utils/planContrary.ts', '.smoke-plan-contrary.mjs')
const mod = await import(pathToFileURL(out).href + `?t=${Date.now()}`)
const { supportsPlanContraryPlay } = mod

let failed = 0

const gateCases = [
  { name: '定位胆', ctx: { betMode: 'dingwei', playTypeId: 'dingwei' }, want: true },
  { name: '龙虎', ctx: { betMode: 'longhu', playTypeId: 'longhu', playTypeLabel: '龙虎' }, want: true },
  { name: '和值', ctx: { betMode: 'hezhi', playTypeId: 'hezhi' }, want: false },
  { name: '跨度', ctx: { betMode: 'kuadu' }, want: false },
  { name: '大小单双', ctx: { betMode: 'dxds' }, want: false },
  { name: '组三', ctx: { betMode: 'zu3' }, want: false },
  { name: '直选单式', ctx: { betMode: 'danshi' }, want: false },
  { name: '任选复式', ctx: { betMode: 'fushi', playTypeId: 'renxuan' }, want: true },
]

for (const c of gateCases) {
  const got = supportsPlanContraryPlay(c.ctx)
  const ok = got === c.want
  console.log(`${ok ? 'PASS' : 'FAIL'} gate ${c.name}: ${got} (want ${c.want})`)
  if (!ok) failed++
}

// 后端补集 / 门禁单测（分两次跑，避免 Windows 把 -run 里的 | 当管道）
const goRuns = ['ComplementPlanContent', 'SupportsPlanContrary']
for (const run of goRuns) {
  const go = spawnSync(
    'go',
    ['test', './internal/schemes/', '-count=1', '-timeout', '60s', '-run', run],
    {
      cwd: path.join(repoRoot, 'backend'),
      encoding: 'utf8',
      shell: false,
    },
  )
  const goOk = go.status === 0
  console.log(
    `${goOk ? 'PASS' : 'FAIL'} go -run ${run}` +
      (goOk ? '' : `\n${go.stdout || ''}\n${go.stderr || ''}`),
  )
  if (!goOk) failed++
}
try {
  fs.unlinkSync(out)
} catch {
  /* ignore */
}

console.log(failed === 0 ? '\nSMOKE OK' : `\nSMOKE FAILED: ${failed}`)
process.exit(failed === 0 ? 0 : 1)
