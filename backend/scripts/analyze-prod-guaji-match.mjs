import fs from 'fs'
import https from 'https'

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { timeout: 45000 }, (r) => {
      let d = ''
      r.on('data', (c) => (d += c))
      r.on('end', () => resolve(JSON.parse(d)))
    }).on('error', reject)
  })
}

function parseCSV(text) {
  const lines = text.trim().split(/\r?\n/)
  const hdr = lines[0].split(',')
  return lines.slice(1).map((line) => {
    const parts = []
    let cur = ''
    let q = false
    for (const c of line) {
      if (c === '"') {
        q = !q
        continue
      }
      if (c === ',' && !q) {
        parts.push(cur)
        cur = ''
        continue
      }
      cur += c
    }
    parts.push(cur)
    const o = {}
    hdr.forEach((h, i) => (o[h] = parts[i]))
    return o
  })
}

function norm(s) {
  return String(s || '').replace(/\s+/g, '').replace(/[·•]/g, '')
}

function flatRules(data) {
  const out = []
  for (const [tid, tpl] of Object.entries(data)) {
    for (const g of tpl.groups || []) {
      for (const t of g.team || []) {
        for (const r of t.rule || []) {
          if (r.active !== false) {
            out.push({
              tid,
              group: g.name,
              team: t.name,
              id: r.id,
              full_name: r.full_name,
              name: r.name,
            })
          }
        }
      }
    }
  }
  return out
}

const tplMap = {
  ssc_std: '1',
  syxw_std: '2',
  pk10_std: '3',
  k3_std: '4',
  pc28_std: '5',
  lhc_std: '8',
  fast_ssc_std: '7',
}

const alias = {
  哈希一分彩: '哈希1分彩',
  哈希三分�? '哈希3分彩',
  哈希五分�? '哈希5分彩',
  波场一分彩: '波场1分彩',
  波场三分�? '波场3分彩',
  波场五分�? '波场5分彩',
  币安一分彩: '币安1分彩',
  币安三分�? '币安3分彩',
  币安五分�? '币安5分彩',
}

const prodLott = (await get('https://www.v6hs1.com/api/games/new_lott?limit=299&page=1')).data
const rules = (await get('https://www.v6hs1.com/api/games/rules/v2')).data
const csv = parseCSV(fs.readFileSync('backend/docs/seeds/lottery_catalog.csv', 'utf8'))
const subPlays = parseCSV(fs.readFileSync('backend/docs/seeds/sub_plays.csv', 'utf8')).filter(
  (r) => r.enabled === 'true',
)

const prodByName = new Map(prodLott.map((x) => [norm(alias[x.name] || x.name), x]))
const matched = []
for (const row of csv) {
  const p = prodByName.get(norm(row.display_name))
  if (p) {
    matched.push({
      code: row.code,
      display: row.display_name,
      template: row.play_template,
      prodId: p.id,
      prodName: p.name,
    })
  }
}

console.log('=== 33 彩种 outbound 映射 ===')
for (const m of matched) {
  console.log(`${m.code}\t${m.prodId}\t${m.template}\t${m.display}`)
}

const offSale = csv.filter((r) => !matched.some((m) => m.code === r.code))
console.log('\n=== 需下架彩种 ===', offSale.length)
for (const r of offSale) console.log(`${r.code}\t${r.display_name}`)

const templatesUsed = [...new Set(matched.map((m) => m.template))]
console.log('\n=== 玩法模板匹配（按 full_name�?==')
for (const tpl of templatesUsed) {
  const tid = tplMap[tpl]
  if (!tid) {
    console.log(`\n${tpl}: �?rules type 映射`)
    continue
  }
  const remote = flatRules({ [tid]: rules[tid] })
  const local = subPlays.filter((r) => r.template_code === tpl)
  const rmap = new Map(remote.map((r) => [norm(r.full_name), r]))
  const lmap = new Map(local.map((l) => [norm(l.label), l]))
  let ok = 0
  const miss = []
  const remoteOnly = []
  for (const l of local) {
    const r = rmap.get(norm(l.label))
    if (r) ok++
    else miss.push(`${l.type_id}/${l.sub_id} ${l.label} outbound=${l.outbound_play_code}`)
  }
  for (const r of remote) {
    if (!lmap.has(norm(r.full_name))) remoteOnly.push(`${r.id} ${r.full_name}`)
  }
  console.log(
    `\n${tpl} (type ${tid}): local=${local.length} remote=${remote.length} match=${ok} miss=${miss.length} remoteOnly=${remoteOnly.length}`,
  )
  if (miss.length) {
    console.log('  本地有、正式无:')
    miss.slice(0, 15).forEach((x) => console.log('   ', x))
    if (miss.length > 15) console.log(`    ... +${miss.length - 15}`)
  }
  if (remoteOnly.length) {
    console.log('  正式有、本地无:')
    remoteOnly.slice(0, 10).forEach((x) => console.log('   ', x))
    if (remoteOnly.length > 10) console.log(`    ... +${remoteOnly.length - 10}`)
  }
}

// id drift for matched rules (same full_name different id)
console.log('\n=== 同名玩法 rule_id 漂移（需更新 outbound�?==')
for (const tpl of templatesUsed) {
  const tid = tplMap[tpl]
  if (!tid) continue
  const remote = flatRules({ [tid]: rules[tid] })
  const local = subPlays.filter((r) => r.template_code === tpl)
  const rmap = new Map(remote.map((r) => [norm(r.full_name), r.id]))
  let drift = 0
  for (const l of local) {
    const rid = rmap.get(norm(l.label))
    if (rid && rid !== l.outbound_play_code) drift++
  }
  console.log(`${tpl}: ${drift} �?rule_id 需变更`)
}
