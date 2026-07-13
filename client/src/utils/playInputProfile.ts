/**
 * 各彩种方案内容输入画像：位标签、号池范围、与第三方 v6hs1 面板对齐。
 */

export const SSC_POSITION_LABELS = ['万', '千', '百', '十', '个'] as const

/** 十一选五：第一位…第五位 */
export const SYXW_POSITION_LABELS = ['一位', '二位', '三位', '四位', '五位'] as const

/** PK10：冠军…第十名 */
export const PK10_POSITION_LABELS = [
  '冠军',
  '亚军',
  '季军',
  '第四',
  '第五',
  '第六',
  '第七',
  '第八',
  '第九',
  '第十',
] as const

export function syxwSegmentLen(typeId: string, typeLabel: string, group: string): number {
  const text = `${group} ${typeLabel} ${typeId}`
  if (typeId === 'g001' || text.includes('前三')) return 3
  if (typeId === 'g002' || text.includes('前二')) return 2
  if (typeId === 'g003' || text.includes('一星') || text.includes('定位')) return 5
  return 1
}

export function pk10SegmentLen(typeId: string, typeLabel: string, subLabel: string, group: string): number {
  const text = `${group} ${typeLabel} ${subLabel} ${typeId}`
  if (text.includes('前五') || typeId === 'g007') return 5
  if (text.includes('前四') || typeId === 'g006') return 4
  if (text.includes('前三') || typeId === 'g005') return 3
  if (text.includes('前二') || text.includes('冠亚') || typeId === 'g004') return 2
  if (text.includes('前一') || typeId === 'g003') return 1
  if (typeId === 'g001' || text.includes('定位') || text.includes('一星')) return 10
  return 1
}

/** 从玩法文案推断和值位数（任二=2、任三=3…） */
export function hezhiDigitLenFromText(text: string, fallback = 3): number {
  const t = text.trim()
  if (!t) return fallback
  if (
    t.includes('任选四') ||
    t.includes('任四') ||
    t.includes('四星') ||
    t.includes('前后四') ||
    /\bren4\b/i.test(t)
  ) {
    return 4
  }
  if (
    t.includes('任选三') ||
    t.includes('任三') ||
    t.includes('前三') ||
    t.includes('中三') ||
    t.includes('后三') ||
    t.includes('前后三') ||
    t.includes('前中后三') ||
    t.includes('三星') ||
    /\bren3\b/i.test(t)
  ) {
    return 3
  }
  if (
    t.includes('任选二') ||
    t.includes('任二') ||
    t.includes('前二') ||
    t.includes('后二') ||
    t.includes('前后二') ||
    t.includes('二星') ||
    t.includes('冠亚') ||
    /\bren2\b/i.test(t)
  ) {
    return 2
  }
  if (t.includes('五星')) return 5
  return fallback
}

/** 和值号池：与第三方可选范围对齐 */
export function hezhiPoolRange(
  playTemplate: string,
  guajiGroup: string,
  subLabel: string,
  segmentLen: number,
  fullName = '',
): { min: number; max: number } {
  if (playTemplate === 'pc28_std') return { min: 0, max: 27 }
  if (playTemplate === 'k3_std') return { min: 3, max: 18 }
  if (playTemplate === 'pk10_std') {
    if (subLabel.includes('前三') || subLabel.includes('后三') || fullName.includes('前三') || fullName.includes('后三')) {
      return { min: 6, max: 27 }
    }
    if (subLabel.includes('冠亚') || subLabel.includes('首尾') || fullName.includes('冠亚')) {
      return { min: 3, max: 19 }
    }
    return { min: 3, max: 19 }
  }
  // SSC / fast_ssc
  const text = `${guajiGroup} ${subLabel} ${fullName}`
  const isZuxuan = text.includes('组选')
  const len =
    segmentLen > 1 ? segmentLen : hezhiDigitLenFromText(text, 3)
  if (isZuxuan) {
    // 任二/前二组选和值：第三方面板为 0–18（与直选和值同档）
    if (len === 2) return { min: 0, max: 18 }
    // 三星组选和值：1–26（不含豹子）
    if (len === 3) return { min: 1, max: 26 }
    if (len === 4) return { min: 1, max: 35 }
    return { min: 1, max: Math.min(44, len * 9 - 1) }
  }
  if (len === 2) return { min: 0, max: 18 }
  if (len === 4) return { min: 0, max: 36 }
  if (len === 5) return { min: 0, max: 45 }
  return { min: 0, max: 27 }
}

export function kuaduPoolRange(): { min: number; max: number } {
  return { min: 0, max: 9 }
}

export function weishuPoolRange(): { min: number; max: number } {
  return { min: 0, max: 9 }
}

/** 有序和值组合数（每位 0–9），对齐后端 countOrderedSumCombinations */
export function countOrderedSumCombinations(targetSum: number, positions: number): number {
  if (positions <= 0 || targetSum < 0) return 0
  const ways: number[][] = Array.from({ length: positions + 1 }, () =>
    Array.from({ length: targetSum + 1 }, () => 0),
  )
  ways[0][0] = 1
  for (let pos = 0; pos < positions; pos++) {
    for (let sum = 0; sum <= targetSum; sum++) {
      const n = ways[pos][sum]
      if (!n) continue
      for (let d = 0; d <= 9 && sum + d <= targetSum; d++) {
        ways[pos + 1][sum + d] += n
      }
    }
  }
  return ways[positions][targetSum]
}

/** 有序跨度组合数，对齐后端 countOrderedSpanCombinations */
export function countOrderedSpanCombinations(span: number, positions: number): number {
  if (positions <= 0 || span < 0) return 0
  let count = 0
  const dfs = (idx: number, min: number, max: number) => {
    if (idx === positions) {
      if (max - min === span) count++
      return
    }
    for (let d = 0; d <= 9; d++) {
      const nmin = idx === 0 ? d : Math.min(min, d)
      const nmax = idx === 0 ? d : Math.max(max, d)
      dfs(idx + 1, nmin, nmax)
    }
  }
  dfs(0, 0, 0)
  return count
}

/** 组选和值组合数（简化：无序，对齐后端常用三星） */
export function countZuxuanSumCombinations(targetSum: number, positions: number): number {
  if (positions === 2) {
    // 二星组选和值：两不同数字之和
    let n = 0
    for (let a = 0; a <= 9; a++) {
      for (let b = a + 1; b <= 9; b++) {
        if (a + b === targetSum) n++
      }
    }
    return n
  }
  if (positions === 3) {
    let n = 0
    for (let a = 0; a <= 9; a++) {
      for (let b = a; b <= 9; b++) {
        for (let c = b; c <= 9; c++) {
          if (a + b + c !== targetSum) continue
          if (a === b && b === c) continue // 豹子通常不计入组选和值
          n++
        }
      }
    }
    return n
  }
  return countOrderedSumCombinations(targetSum, positions)
}

/**
 * 三星混合组选注数（对齐第三方）：
 * - 每注须为 digitLen 位数字
 * - 排除豹子（三位相同）
 * - 按组选形态去重（123 与 321 同一注）
 *
 * 例：123,321,232,222,333,444,542 → 3 注（123、232、542）
 */
export function countHunheZuxuanUnits(content: string, digitLen: number): number {
  const len = digitLen > 0 ? digitLen : 3
  const parts = content
    .split(/[\n,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const seen = new Set<string>()
  let n = 0
  for (const p of parts) {
    if (!new RegExp(`^\\d{${len}}$`).test(p)) continue
    if ([...p].every((c) => c === p[0])) continue
    const key = [...p].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    n++
  }
  return n
}

/** 混合组选每注位数：优先 segmentLen，否则从玩法文案推断 */
export function hunheDigitLenFromConfig(config: {
  segmentLen?: number
  guajiGroup?: string
  playTypeLabel?: string
  playMethodLabel?: string
}): number {
  if ((config.segmentLen ?? 0) > 1) return config.segmentLen!
  const text = `${config.guajiGroup ?? ''} ${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''}`
  if (text.includes('任二') || text.includes('前二') || text.includes('后二') || text.includes('二星')) return 2
  if (text.includes('任四') || text.includes('四星')) return 4
  if (text.includes('五星')) return 5
  return 3
}

/** 六合彩 betMode → inputMode */
export function lhcInputModeFromBetMode(betMode: string, typeId: string, typeLabel: string): string {
  const bm = betMode.trim()
  const label = typeLabel.trim()
  if (bm === 'tema' || bm === 'zhengte' || typeId === 'g001' || typeId === 'g002' || label === '特码' || label === '正特码') {
    return 'lhc_num'
  }
  if (
    ['fushi', 'buzhong', 'xuanyi', 'tuotou', 'renzhong'].includes(bm) ||
    label === '连码' ||
    label === '全不中' ||
    label === '多选中一' ||
    typeId === 'g003' ||
    typeId === 'g013' ||
    typeId === 'g014'
  ) {
    return 'lhc_num'
  }
  if (
    ['texiao', 'xiao', 'xiao_z', 'xiao_bz', 'sx_dp', 'zongxiao'].includes(bm) ||
    label === '生肖' ||
    label === '生肖连' ||
    typeId === 'g005' ||
    typeId === 'g011'
  ) {
    return bm === 'zongxiao' ? 'lhc_attr' : 'lhc_zodiac'
  }
  if (['weishu', 'wei_bz', 'ws_dp'].includes(bm) || label === '尾数连' || label === '一肖尾数' || typeId === 'g012') {
    if (label === '一肖尾数' || typeId === 'g010') return 'lhc_attr'
    return 'lhc_tail'
  }
  if (
    [
      'tematouwei',
      'qima',
      'bose',
      'banbo',
      'banbanbo',
      'wuxing',
      'jiaye',
      'guoguan',
      'zongxiao',
      'sw_dp',
      'renyi_dp',
    ].includes(bm) ||
    ['特码头尾', '七码', '波色', '五行家野', '过关', '特平中'].includes(label)
  ) {
    return 'lhc_attr'
  }
  return 'lhc_num'
}
