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
const urls = [...new Set([...html.matchAll(/\/static\/js\/[^"']+\.js/g)].map((m) => m[0]))]
const needles = [
  '\\u4efb\\u4e8c',
  '\\u4efb\\u9009',
  '\\u76f4\\u9009\\u5355\\u5f0f',
  '\\u7ec4\\u9009',
  '\\u5927\\u5c0f\\u5355\\u53cc',
  'new_lott',
  'LottPanel',
  'BetPanel',
  'spinach',
  'cold_hot',
  'rules/v2',
]

for (const u of urls) {
  const t = await get('https://www.v6hs1.com' + u)
  const h = needles.filter((p) => t.includes(p))
  if (h.length) console.log(u, h.join(','))
}
