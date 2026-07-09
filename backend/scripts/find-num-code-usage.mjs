import fs from 'fs'

const files = fs.readdirSync('data/guaji-js-snap').filter((f) => f.endsWith('.js'))
const patterns = ['num_code.split', '.num_code', 'bet_content', 'bets_nums', 'solo_bets', 'getBetNum', 'betContent', 'formatSSC', 'encodeSSC', 'lottContent']

for (const f of files) {
  const text = fs.readFileSync(`data/guaji-js-snap/${f}`, 'utf8')
  for (const p of patterns) {
    let idx = 0
    let c = 0
    while (c < 3) {
      const i = text.indexOf(p, idx)
      if (i < 0) break
      if (f.includes('lott') || f.includes('06837') || f.includes('cff61')) {
        console.log('\nFILE', f, 'PAT', p)
        console.log(text.slice(Math.max(0, i - 120), i + 280).replace(/\s+/g, ' '))
      }
      idx = i + p.length
      c++
    }
  }
}
