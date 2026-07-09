import fs from 'fs'

const files = fs.readdirSync('data/guaji-js-snap').filter((f) => f.endsWith('.js'))
const needles = ['bets_nums', 'bet_content', 'solo_bets', 'num_code', 'getBet', 'encode', 'formatBet', 'lottBet']

for (const f of files) {
  const text = fs.readFileSync(`data/guaji-js-snap/${f}`, 'utf8')
  const hits = needles.filter((n) => text.includes(n))
  if (!hits.length) continue
  console.log('\nFILE', f, hits.join(','))
  for (const n of hits) {
    let idx = 0
    let c = 0
    while (c < 2) {
      const i = text.indexOf(n, idx)
      if (i < 0) break
      console.log(' ', n, '=>', text.slice(Math.max(0, i - 100), i + 220).replace(/\s+/g, ' '))
      idx = i + n.length
      c++
    }
  }
}
