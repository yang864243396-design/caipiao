import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), '../src')
const bad = []
for (const f of walk(root)) {
  const t = fs.readFileSync(f, 'utf8')
  const lines = t.split(/\r?\n/)
  lines.forEach((line, i) => {
    if (/['"`][^'"`]*\?{2,}[^'"`]*['"`]/.test(line)) bad.push([f, i + 1, line.trim()])
    if (/>\?{2,}</.test(line)) bad.push([f, i + 1, line.trim()])
    if (/label="\?+/.test(line)) bad.push([f, i + 1, line.trim()])
    if (/placeholder="\?+/.test(line)) bad.push([f, i + 1, line.trim()])
    if (/title="\?+/.test(line)) bad.push([f, i + 1, line.trim()])
  })
}
console.log('bad count', bad.length)
for (const [f, l, c] of bad.slice(0, 50)) {
  console.log(`${path.relative(root, f)}:${l}: ${c.slice(0, 140)}`)
}
