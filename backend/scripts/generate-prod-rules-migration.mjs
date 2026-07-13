/**
 * 从正式环�?rules/v2 生成 play_types / sub_plays 迁移 SQL�?
 * 用法: node backend/scripts/generate-prod-rules-migration.mjs
 */
import https from 'https'
import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const outPath = path.join(__dirname, '../migrations/00111_guaji_prod_rules_v2_sync.sql')

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { timeout: 45000 }, (r) => {
      let d = ''
      r.on('data', (c) => (d += c))
      r.on('end', () => resolve(JSON.parse(d)))
    }).on('error', reject)
  })
}

function esc(s) {
  return String(s ?? '').replace(/'/g, "''")
}

function jsonEsc(obj) {
  return esc(JSON.stringify(obj))
}

const bindings = [
  { template: 'ssc_std', typeId: '1' },
  { template: 'fast_ssc_std', typeId: '7' },
  { template: 'syxw_std', typeId: '2' },
  { template: 'pk10_std', typeId: '3' },
  { template: 'k3_std', typeId: '4' },
  { template: 'lhc_std', typeId: '8' },
]

const rules = (await get('https://www.v6hs1.com/api/games/rules/v2')).data

const lines = []
lines.push('-- +goose Up')
lines.push('-- 正式环境 www.v6hs1.com GET /api/games/rules/v2 全量重建玩法树（2026-06-30�?)
lines.push('-- +goose StatementBegin')
lines.push('')

for (const b of bindings) {
  const tpl = rules[b.typeId]
  if (!tpl) {
    console.error(`missing rules type ${b.typeId} for ${b.template}`)
    process.exit(1)
  }
  const name = esc(tpl.name.trim())
  lines.push(`INSERT INTO play_templates (code, label, version, guaji_rules_type_id)`)
  lines.push(`VALUES ('${b.template}', '${name}', 1, '${b.typeId}')`)
  lines.push(`ON CONFLICT (code) DO UPDATE SET label = EXCLUDED.label, guaji_rules_type_id = EXCLUDED.guaji_rules_type_id;`)
  lines.push(`DELETE FROM sub_plays WHERE template_code = '${b.template}';`)
  lines.push(`DELETE FROM play_types WHERE template_code = '${b.template}';`)

  let gi = 0
  for (const group of tpl.groups || []) {
    const groupName = (group.name || '').trim()
    if (!groupName) continue
    gi++
    const typeID = `g${String(gi).padStart(3, '0')}`
    lines.push(
      `INSERT INTO play_types (template_code, type_id, label, sort_order, enabled) VALUES ('${b.template}', '${typeID}', '${esc(groupName)}', ${gi}, true);`,
    )
    let subOrder = 0
    for (const team of group.team || []) {
      const teamName = (team.name || '').trim()
      for (const rule of team.rule || []) {
        if (rule.active === false) continue
        const ruleID = String(rule.id || '').trim()
        const ruleName = String(rule.name || '').trim()
        if (!ruleID || !ruleName) continue
        subOrder++
        const fullName = String(rule.full_name || '').trim()
        const label = fullName || ruleName
        const seg = {
          guajiGroup: groupName,
          guajiTeam: teamName,
          guajiFullName: fullName,
          guajiRuleId: ruleID,
        }
        lines.push(
          `INSERT INTO sub_plays (template_code, type_id, sub_id, label, sort_order, segment_rule, outbound_play_code, enabled) VALUES ('${b.template}', '${typeID}', '${esc(ruleID)}', '${esc(label)}', ${subOrder}, '${jsonEsc(seg)}'::jsonb, '${esc(ruleID)}', true);`,
        )
      }
    }
  }
  lines.push('')
}

lines.push('-- +goose StatementEnd')
lines.push('')
lines.push('-- +goose Down')
lines.push('-- +goose StatementBegin')
lines.push('-- 玩法树回退需从备份或测试环境 rules-sync 重新导入，此处不�?Down�?)
lines.push('-- +goose StatementEnd')

fs.writeFileSync(outPath, lines.join('\n') + '\n', 'utf8')
console.log('wrote', outPath, 'lines', lines.length)
