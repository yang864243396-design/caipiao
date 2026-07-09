/** 六合彩 1–49 号码、生肖、波色等（2026 马年表，与后端 lhc_constants 对齐） */
export const LHC_NUMBERS = Array.from({ length: 49 }, (_, i) => String(i + 1).padStart(2, '0'))

export const LHC_ZODIACS = ['马', '蛇', '龙', '兔', '虎', '牛', '鼠', '猪', '狗', '鸡', '猴', '羊'] as const

export const LHC_ZODIAC_NUMBERS: Record<string, string[]> = {
  马: ['01', '13', '25', '37', '49'],
  蛇: ['02', '14', '26', '38'],
  龙: ['03', '15', '27', '39'],
  兔: ['04', '16', '28', '40'],
  虎: ['05', '17', '29', '41'],
  牛: ['06', '18', '30', '42'],
  鼠: ['07', '19', '31', '43'],
  猪: ['08', '20', '32', '44'],
  狗: ['09', '21', '33', '45'],
  鸡: ['10', '22', '34', '46'],
  猴: ['11', '23', '35', '47'],
  羊: ['12', '24', '36', '48'],
}

export const LHC_TAIL_OPTIONS = ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'] as const

export const LHC_WUXING_OPTIONS = ['金', '木', '水', '火', '土'] as const

export const LHC_JIAYE_OPTIONS = ['家', '野'] as const

export const LHC_BOSE_OPTIONS = ['红', '蓝', '绿'] as const

export const LHC_BANBO_OPTIONS = [
  '红大', '红小', '红单', '红双',
  '蓝大', '蓝小', '蓝单', '蓝双',
  '绿大', '绿小', '绿单', '绿双',
] as const

export const LHC_BANBANBO_OPTIONS = [
  '红大单', '红大双', '红小单', '红小双',
  '蓝大单', '蓝大双', '蓝小单', '蓝小双',
  '绿大单', '绿大双', '绿小单', '绿小双',
] as const

export const LHC_GUOGUAN_OPTIONS = ['大', '小', '单', '双'] as const

export const LHC_ZONGXIAO_OPTIONS = ['二肖', '三肖', '四肖', '五肖', '六肖', '七肖'] as const

/** 与 hash.iyes.dev 总肖面板一致（rule 301，仅 2–7 肖，无 0/1/8+）。 */
export const LHC_ZONGXIAO_ODDS: Record<(typeof LHC_ZONGXIAO_OPTIONS)[number], number> = {
  二肖: 14.841,
  三肖: 14.841,
  四肖: 14.841,
  五肖: 3.007,
  六肖: 1.92,
  七肖: 5.335,
}

export function isLhcZongxiaoOption(value: string): boolean {
  return (LHC_ZONGXIAO_OPTIONS as readonly string[]).includes(value.trim())
}

export const LHC_TEMATOUWEI_OPTIONS = [
  '头0', '头1', '头2', '头3', '头4',
  '尾0', '尾1', '尾2', '尾3', '尾4', '尾5', '尾6', '尾7', '尾8', '尾9',
] as const

/** 七码（rule 313）：第三方 wire 为「单0」–「小7」，共 32 项（非选 1–49 号码）。 */
export const LHC_QIMA_KINDS = ['单', '双', '大', '小'] as const
export const LHC_QIMA_COUNTS = [0, 1, 2, 3, 4, 5, 6, 7] as const

/** 与 hash.iyes.dev 七码面板一致：按种类分组（单0…单7、双0…双7…）。 */
export const LHC_QIMA_OPTIONS = LHC_QIMA_KINDS.flatMap((kind) =>
  LHC_QIMA_COUNTS.map((n) => `${kind}${n}`),
) as readonly string[]

export function isLhcQimaOption(value: string): boolean {
  return (LHC_QIMA_OPTIONS as readonly string[]).includes(value.trim())
}

export function lhcMinPickCount(betMode: string, subId: string): number {
  const s = subId.toLowerCase()
  if (betMode === 'fushi') {
    if (s.includes('san')) return 3
    return 2
  }
  if (betMode === 'buzhong') {
    const m = s.match(/^(\d+)bz$/)
    if (m) return Number(m[1])
    if (s === '15bz') return 15
    return 5
  }
  if (betMode === 'xuanyi') {
    const m = s.match(/^(\d+)x1$/)
    return m ? Number(m[1]) : 5
  }
  if (betMode === 'renzhong') {
    const m = s.match(/^(\d+)l_rz$/)
    return m ? Number(m[1]) : 1
  }
  if (betMode === 'xiao' || betMode === 'xiao_z' || betMode === 'xiao_bz') {
    const m = s.match(/^(\d+)xiao/)
    if (m) return Number(m[1])
    if (s === '1xiao' || s === '1xiao_bz') return 1
    return 2
  }
  if (betMode === 'wei_z' || betMode === 'wei_bz') {
    const m = s.match(/^(\d+)wei/)
    return m ? Number(m[1]) : 1
  }
  return 1
}

export function lhcAttrOptions(betMode: string, panelType: string): readonly string[] {
  if (betMode === 'tematouwei') return LHC_TEMATOUWEI_OPTIONS
  if (betMode === 'wuxing' || (panelType === 'lhc_attr' && betMode === 'wuxing')) {
    return LHC_WUXING_OPTIONS
  }
  if (betMode === 'jiaye') return LHC_JIAYE_OPTIONS
  if (betMode === 'bose') return LHC_BOSE_OPTIONS
  if (betMode === 'banbo') return LHC_BANBO_OPTIONS
  if (betMode === 'banbanbo') return LHC_BANBANBO_OPTIONS
  if (betMode === 'guoguan') return LHC_GUOGUAN_OPTIONS
  if (betMode === 'zongxiao') return LHC_ZONGXIAO_OPTIONS
  if (betMode === 'qima') return LHC_QIMA_OPTIONS
  return LHC_WUXING_OPTIONS
}
