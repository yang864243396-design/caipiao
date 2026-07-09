import fs from 'fs'

const text = fs.readFileSync('data/guaji-js-snap/lott-bet-core.chunk.js', 'utf8')

// Extract JSON-like play config blocks around full_name entries
const re = /"\\u[^"]+":\{"full_name":"\\u[^"]+","num_code":"[^"]*","sort":\d+\}/g
const matches = [...text.matchAll(re)]
console.log('matches', matches.length)

const decode = (s) => JSON.parse('"' + s + '"')

const interesting = []
for (const m of matches) {
  const raw = m[0]
  const nameMatch = raw.match(/full_name":"(\\u[^"]+)"/)
  const codeMatch = raw.match(/num_code":"([^"]*)"/)
  if (!nameMatch || !codeMatch) continue
  const full = decode(nameMatch[1])
  if (
    full.includes('�?) ||
    full.includes('大小单双') ||
    full.includes('和�?) ||
    full.includes('组�?) ||
    full.includes('直�?)
  ) {
    interesting.push({ full, num_code: codeMatch[1] })
  }
}

for (const row of interesting) {
  console.log(JSON.stringify(row))
}
