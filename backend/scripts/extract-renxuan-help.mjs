import fs from 'fs'

const t = fs.readFileSync('data/guaji-js-snap/lott-cff61b85.chunk.js', 'utf8')
const decode = (s) => JSON.parse('"' + s + '"')

const re = /\\u6295\\u6ce8\\u65b9\\u6848[^"]{0,400}/g
const hits = [...t.matchAll(re)]
for (const m of hits) {
  const text = decode(m[0])
  if (text.includes('任二') || text.includes('任三') || text.includes('任选四') || text.includes('大小单双')) {
    console.log(text)
    console.log('---')
  }
}
