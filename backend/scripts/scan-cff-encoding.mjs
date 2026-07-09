import fs from 'fs'

const t = fs.readFileSync('data/guaji-js-snap/lott-cff61b85.chunk.js', 'utf8')
const keys = [
  'bet_content',
  'betContent',
  'bets_nums',
  'betsNums',
  'num_code',
  'numCode',
  'selected',
  'position',
  'checked',
  'submitBet',
  'web_bets',
  'formatContent',
  'getContent',
  'encode',
  'solo',
  'join(',
  'concat',
  '任二',
]
for (const k of keys) {
  let idx = 0
  let n = 0
  while (n < 3) {
    const i = t.indexOf(k, idx)
    if (i < 0) break
    console.log('\n---', k, n, '@', i, '---')
    console.log(t.slice(Math.max(0, i - 80), i + 220).replace(/\s+/g, ' '))
    idx = i + k.length
    n++
  }
}
