/**
 * 投注内容展示行。
 * 位名由后端按玩法位段算好随详情下发（betContentLines）；这里只负责在后端未下发时
 * 原样分行——位名与玩法强相关，前端按行序硬编码会把中三码、后三码等标错位。
 */

export function betContentLines(lines: string[] | undefined, raw: string): string[] {
  if (lines?.length) return lines
  if (raw == null || raw === '') return ['—']
  const out = String(raw)
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) =>
      line
        .split(/[,，\s|]+/)
        .filter(Boolean)
        .join(' '),
    )
    .filter(Boolean)
  return out.length ? out : ['—']
}
