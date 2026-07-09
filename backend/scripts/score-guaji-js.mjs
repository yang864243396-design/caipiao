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
const needles = ['bet_content', 'bets_nums', 'num_code', 'solo_bets', 'getBetNums', 'getBetContent', 'formatContent', 'encodeContent']

for (const u of urls) {
  const t = await get('https://www.v6hs1.com' + u)
  const score = needles.filter((n) => t.includes(n)).length
  if (score >= 2) console.log(score, u, needles.filter((n) => t.includes(n)).join(','))
}
