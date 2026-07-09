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
const urls = [...new Set([...html.matchAll(/\/static\/js\/[^"']+\.js/g)].map((m) => m[0]))]
const outDir = path.join('data', 'guaji-js-snap')
fs.mkdirSync(outDir, { recursive: true })

for (const u of urls) {
  const text = await get('https://www.v6hs1.com' + u)
  if (!text.includes('num_code')) continue
  if (!text.includes('bet_content') && !text.includes('bets_nums') && !text.includes('solo')) continue
  const fname = u.replace(/\//g, '_').slice(1)
  fs.writeFileSync(path.join(outDir, fname), text)
  console.log('SAVE', fname, 'len', text.length)
  for (const k of ['bet_content', 'bets_nums', 'num_code', 'solo_bets']) {
    const i = text.indexOf(k)
    if (i >= 0) console.log(' ', k, text.slice(Math.max(0, i - 120), i + 200).replace(/\s+/g, ' '))
  }
}
