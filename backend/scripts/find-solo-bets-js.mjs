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

for (const u of uniq) {
  const text = await get('https://www.v6hs1.com' + u)
  if (!text.includes('solo_bets') && !text.includes('bets_nums')) continue
  console.log('\n===', u, '===')
  let idx = 0
  while (true) {
    const i = text.indexOf('solo_bets', idx)
    if (i < 0) break
    console.log(text.slice(Math.max(0, i - 200), i + 400).replace(/\s+/g, ' '))
    console.log('---')
    idx = i + 9
    if (idx > text.length) break
  }
  idx = 0
  while (true) {
    const i = text.indexOf('bets_nums', idx)
    if (i < 0) break
    console.log(text.slice(Math.max(0, i - 200), i + 400).replace(/\s+/g, ' '))
    console.log('---')
    idx = i + 9
    if (idx > text.length) break
  }
}
