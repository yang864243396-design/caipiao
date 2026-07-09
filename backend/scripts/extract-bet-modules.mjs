import fs from 'fs'

const t = fs.readFileSync('data/guaji-js-snap/static_js_main~06837ae4.367ef329.chunk.js', 'utf8')
const re = /,\d+:function\([^)]*\)\{/g
const mods = []
for (const m of t.matchAll(re)) {
  mods.push({ id: m[0].slice(1).split(':')[0], start: m.index })
}
for (let i = 0; i < mods.length; i++) {
  const start = mods[i].start
  const end = i + 1 < mods.length ? mods[i + 1].start : t.length
  const body = t.slice(start, end)
  if (!body.includes('num_code') && !body.includes('bet_content') && !body.includes('bets_nums')) continue
  console.log('\n=== module', mods[i].id, 'len', body.length, '===')
  const idx = body.indexOf('num_code')
  const j = body.indexOf('bets_nums')
  const k = body.indexOf('bet_content')
  for (const [label, pos] of [['num_code', idx], ['bets_nums', j], ['bet_content', k]]) {
    if (pos >= 0) console.log(label, '=>', body.slice(Math.max(0, pos - 150), pos + 350).replace(/\s+/g, ' '))
  }
}
