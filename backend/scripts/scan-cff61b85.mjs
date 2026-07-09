import https from 'https'
import fs from 'fs'

const url = 'https://www.v6hs1.com/static/js/main~cff61b85.56e96683.chunk.js'
const text = await new Promise((resolve, reject) => {
  https.get(url, (r) => {
    let d = ''
    r.on('data', (c) => (d += c))
    r.on('end', () => resolve(d))
  }).on('error', reject)
})
fs.writeFileSync('data/guaji-js-snap/lott-cff61b85.chunk.js', text)
console.log('len', text.length)
const keys = [
  'num_code',
  'bet_content',
  'bets_nums',
  'solo',
  'full_name',
  'getBet',
  'format',
  'encode',
  'calc',
  'content',
  '任二直选单�?,
  '\\u4efb\\u4e8c\\u76f4\\u9009\\u5355\\u5f0f',
]
for (const k of keys) {
  let idx = 0
  let n = 0
  while (n < 5) {
    const i = text.indexOf(k, idx)
    if (i < 0) break
    console.log('\n---', k, n, '---')
    console.log(text.slice(Math.max(0, i - 100), i + 300).replace(/\s+/g, ' '))
    idx = i + k.length
    n++
  }
}
