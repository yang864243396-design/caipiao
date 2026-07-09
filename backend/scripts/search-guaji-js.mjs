import fs from 'fs'
import path from 'path'

const dir = 'data/guaji-js-snap'
const patterns = [
  '冠亚大小',
  '冠亚和值大�?,
  '前三大小',
  '后三大小',
  ':221',
  ',221,',
  '"221"',
  '大小单双',
]

for (const f of fs.readdirSync(dir)) {
  if (!f.endsWith('.js')) continue
  const s = fs.readFileSync(path.join(dir, f), 'utf8')
  for (const pat of patterns) {
    let idx = 0
    while (true) {
      const i = s.indexOf(pat, idx)
      if (i < 0) break
      console.log(`\n=== ${f} "${pat}" @ ${i} ===`)
      console.log(s.slice(Math.max(0, i - 100), i + 200))
      idx = i + pat.length
      if (idx > i + 5000) break
    }
  }
}
