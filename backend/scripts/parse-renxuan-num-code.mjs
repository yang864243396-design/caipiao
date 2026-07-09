import fs from 'fs'

const text = fs.readFileSync('data/guaji-js-snap/lott-bet-core.chunk.js', 'utf8')
const re = /"\\u[^"]+":\{"full_name":"\\u[^"]+","num_code":"[^"]*","sort":\d+\}/g
const decode = (s) => JSON.parse('"' + s + '"')

for (const m of text.matchAll(re)) {
  const raw = m[0]
  const nameMatch = raw.match(/full_name":"(\\u[^"]+)"/)
  const codeMatch = raw.match(/num_code":"([^"]*)"/)
  if (!nameMatch || !codeMatch) continue
  const full = decode(nameMatch[1])
  if (full.includes('任二') || full.includes('任三') || full.includes('任选四') || full.includes('大小单双') || full.includes('和值单') || full.includes('和值大')) {
    console.log(full, '=>', codeMatch[1])
  }
}
