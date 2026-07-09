import https from 'https'

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
const needles = ['bet_contents', 'amount_unit', 'max_bets', 'auto_type', 'web_bets/lott', '任二', '任�?, '组选和�?, '直选单�?, '大小单双', 'encodeSSC', 'formatSSC', 'lottContent', 'betContent']

for (const u of uniq) {
  const text = await get('https://www.v6hs1.com' + u)
  const hits = needles.filter((p) => text.includes(p))
  if (!hits.length) continue
  console.log('\nFILE', u, hits.join(','))
  for (const p of hits.slice(0, 3)) {
    const i = text.indexOf(p)
    console.log(' ', text.slice(Math.max(0, i - 100), i + 250).replace(/\s+/g, ' '))
  }
}
