import fs from 'fs'
import https from 'https'

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, (r) => {
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

function flatRules(g) {
  const out = []
  for (const grp of g.groups || []) {
    for (const team of grp.team || []) {
      for (const r of team.rule || []) {
        out.push({
          group: grp.name,
          team: team.name,
          id: String(r.id),
          full_name: r.full_name,
          active: r.active !== false,
        })
      }
    }
  }
  return out
}

function norm(s) {
  return String(s || '')
    .replace(/\s+/g, '')
    .replace(/[·•]/g, '')
}

const tplMap = {
  ssc_std: { tpId: '1', tpName: '时时�?, note: '普通SSC彩种' },
  syxw_std: { tpId: '2', tpName: '十一选五' },
  pk10_std: { tpId: '3', tpName: 'PK10' },
  k3_std: { tpId: '4', tpName: '快三' },
  pc28_std: { tpId: '5', tpName: 'PC28', note: '台湾28' },
  lhc_std: { tpId: '8', tpName: '六合�? },
}

const extraTp = {
  fast: { tpId: '7', tpName: '快速彩', note: '哈希/极速类SSC' },
  fc3d: { tpId: '6', tpName: '3D' },
  ca28: { tpId: '11', tpName: '加拿�?8' },
}

const rulesResp = await get('https://www.v6hs1.com/api/games/rules/v2')
const subPlays = parseCSV(fs.readFileSync('docs/seeds/sub_plays.csv', 'utf8'))
const playTypes = parseCSV(fs.readFileSync('docs/seeds/play_types.csv', 'utf8'))

console.log('rules/v2 顶层�?= 玩法类型模板 id（非 new_lott 彩种 id�?)
console.log('类型�?', Object.keys(rulesResp.data).length)
console.log('')

function compare(label, localRows, tpId) {
  const remote = flatRules(rulesResp.data[tpId]).filter((r) => r.active)
  const localNames = new Set(localRows.map((r) => norm(r.label)))
  const remoteByName = new Map(remote.map((r) => [norm(r.full_name), r]))
  let nameMatch = 0
  for (const n of localNames) if (remoteByName.has(n)) nameMatch++
  const localOnly = [...localNames].filter((n) => !remoteByName.has(n))
  const remoteOnly = [...remoteByName.keys()].filter((n) => !localNames.has(n))
  console.log(`--- ${label}  (local ${localRows.length} vs remote ${remote.length}, name match ${nameMatch}) ---`)
  if (localOnly.length) console.log(`  local only (${localOnly.length}):`, localOnly.slice(0, 12).join('; '))
  if (remoteOnly.length) console.log(`  remote only (${remoteOnly.length}):`, remoteOnly.slice(0, 12).join('; '))
  return { local: localRows.length, remote: remote.length, nameMatch, localOnly, remoteOnly }
}

const summary = {}
for (const [tpl, meta] of Object.entries(tplMap)) {
  const local = subPlays.filter((r) => r.template_code === tpl && r.enabled === 'true')
  summary[tpl] = compare(`${tpl} <-> type ${meta.tpId} ${meta.tpName}`, local, meta.tpId)
  console.log('')
}

// 快速彩 vs ssc_std
const sscLocal = subPlays.filter((r) => r.template_code === 'ssc_std' && r.enabled === 'true')
compare('ssc_std vs type 7 快速彩 (subset?)', sscLocal, extraTp.fast.tpId)
console.log('')

// 3D / 加拿�?8 - no local template
compare('(无本地模�? vs type 6 3D', [], extraTp.fc3d.tpId)
console.log('')
compare('(无本地模�? vs type 11 加拿�?8', [], extraTp.ca28.tpId)
console.log('')

// outbound_play_code stats from DB would be ideal; use CSV
const numericOutbound = subPlays.filter((r) => /^[0-9]+$/.test(r.outbound_play_code))
const compositeOutbound = subPlays.filter((r) => r.outbound_play_code.includes(':'))
console.log('=== 本地 outbound_play_code 形�?===')
console.log('numeric rule_id:', numericOutbound.length, numericOutbound.map((r) => `${r.template_code}/${r.sub_id}=${r.outbound_play_code}`).slice(0, 8).join(', '))
console.log('composite template:type:sub:', compositeOutbound.length)

// sample third-party rule ids for SSC dingwei
const sscRemote = flatRules(rulesResp.data['1']).filter((r) => r.active)
const dingwei = sscRemote.filter((r) => r.group === '一�?)
console.log('\n=== 第三�?type1 一星定�?rule_id ===')
console.log(dingwei.map((r) => `${r.id}:${r.full_name}`).join('\n'))

// group structure diff lhc
const lhcLocal = subPlays.filter((r) => r.template_code === 'lhc_std' && r.enabled === 'true')
const lhcRemote = flatRules(rulesResp.data['8']).filter((r) => r.active)
const lhcLocalGroups = [...new Set(lhcLocal.map((r) => playTypes.find((t) => t.template_code === 'lhc_std' && t.type_id === r.type_id)?.label || r.type_id))]
const lhcRemoteGroups = [...new Set(lhcRemote.map((r) => r.group))]
console.log('\n=== 六合彩大类对�?===')
console.log('local groups', lhcLocalGroups.length, ':', lhcLocalGroups.join(' / '))
console.log('remote groups', lhcRemoteGroups.length, ':', lhcRemoteGroups.join(' / '))
