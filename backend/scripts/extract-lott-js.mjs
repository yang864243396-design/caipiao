import fs from 'fs'

const text = fs.readFileSync('data/guaji-js-snap/lott-bet-core.chunk.js', 'utf8')
const keys = [
  'bet_content',
  'bets_nums',
  'solo_bets',
  'solo:',
  'max_bets',
  '任二',
  '任�?,
  '直选单�?,
  '组选复�?,
  '组选和�?,
  '直选和�?,
  '大小单双',
  'encodeBet',
  'formatBet',
  'getBetContent',
  'buildBet',
  'join(",")',
  'join(",")',
  'ren2',
  'ren3',
  'ren4',
  'zu3',
  'zu6',
  'dxds',
]

for (const k of keys) {
  let idx = 0
  let n = 0
  while (n < 5) {
    const i = text.indexOf(k, idx)
    if (i < 0) break
    console.log('\n===', k, '#', ++n, 'at', i, '===')
    console.log(text.slice(Math.max(0, i - 180), Math.min(text.length, i + 320)).replace(/\s+/g, ' '))
    idx = i + k.length
  }
}
