import https from 'https'
import fs from 'fs'
import path from 'path'

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, (r) => {
      let d = ''
      r.on('data', (c) => (d += c))
      r.on('end', () => resolve(d))
    }).on('error', reject)
  })
}

const html = await get('https://www.v6hs1.com/')
const urls = [...html.matchAll(/\/static\/js\/[^"']+\.js/g)].map((m) => m[0])
const uniq = [...new Set(urls)]
const outDir = path.join('data', 'guaji-js-snap')
fs.mkdirSync(outDir, { recursive: true })

const needles = [
  'web_bets/lott',
  'solo_bets',
  'max_bets',
  'bet_contents',
  'bets_nums',
  'rule_full_name',
  'encodeBet',
  'formatBet',
  '任二',
  '任�?,
  'ren2',
  'zu3',
  '大小单双',
  'lott_bet',
  'LottBet',
]

for (const u of uniq) {
  const text = await get('https://www.v6hs1.com' + u)
  const hits = needles.filter((p) => text.includes(p))
  if (!hits.length) continue
  const fname = u.replace(/\//g, '_').slice(1)
  fs.writeFileSync(path.join(outDir, fname), text)
  console.log('SAVE', fname, hits.join('|'))
  for (const p of hits) {
    const idx = text.indexOf(p)
    console.log(' ', p, '=>', text.slice(Math.max(0, idx - 80), idx + 160).replace(/\s+/g, ' '))
  }
}
