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

/** 尾数对碰号码表（0 尾 4 个，1–9 尾各 5 个；与 n%10 一致） */
export const LHC_TAIL_NUMBERS: Record<string, string[]> = Object.fromEntries(
  LHC_TAIL_OPTIONS.map((t) => {
    const nums: string[] = []
    for (let n = 1; n <= 49; n++) {
      if (n % 10 === Number(t)) nums.push(String(n).padStart(2, '0'))
    }
    return [t, nums]
  }),
)

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

/**
 * 特码 wire 属性段顺序（非波色）：`号码|属性|波色` 的中间段。
 * 与第三方下单示例对齐。
 */
export const LHC_TEMA_ATTR_OPTIONS = [
  '尾双',
  '尾单',
  '尾小',
  '尾大',
  '总分大',
  '总分小',
  '合小',
  '合大',
  '大',
  '小',
  '单',
  '双',
  '合双',
  '合单',
  '总分单',
  '总分双',
] as const

/** 特码 wire 波色段：`号码|属性|波色` 的末段 */
export const LHC_TEMA_WAVE_OPTIONS = ['红波', '蓝波', '绿波'] as const

/**
 * 特码/正特码方案内容快捷属性（输入框上方点选）。
 * 与常见六合面板对齐：红波/蓝波/绿波（非「洪波」「绿播」）。
 */
export const LHC_TEMA_QUICK_OPTIONS = [
  '大',
  '小',
  '单',
  '双',
  '合大',
  '合小',
  '合单',
  '合双',
  '总分大',
  '总分小',
  '总分单',
  '总分双',
  '尾大',
  '尾小',
  '尾单',
  '尾双',
  '红波',
  '蓝波',
  '绿波',
] as const

export function isLhcTemaAttrOption(value: string): boolean {
  return (LHC_TEMA_ATTR_OPTIONS as readonly string[]).includes(value.trim())
}

export function isLhcTemaWaveOption(value: string): boolean {
  return (LHC_TEMA_WAVE_OPTIONS as readonly string[]).includes(value.trim())
}

export function isLhcTemaQuickOption(value: string): boolean {
  const t = value.trim()
  return isLhcTemaAttrOption(t) || isLhcTemaWaveOption(t)
}

/** 特码/正特冷热出号宇宙：01–49 + 16 属性 + 3 波色 = 68 */
export function lhcTemaHcwUniverse(): string[] {
  const nums = Array.from({ length: 49 }, (_, i) => String(i + 1).padStart(2, '0'))
  return [...nums, ...LHC_TEMA_ATTR_OPTIONS, ...LHC_TEMA_WAVE_OPTIONS]
}

/** 二全中复式/拖头号池选号：至少 2、至多 10（01–49，逗号分隔） */
export const LHC_ERQUANZHONG_NUM_MIN_PICKS = 2
export const LHC_ERQUANZHONG_NUM_MAX_PICKS = 10
/** @deprecated 使用 LHC_ERQUANZHONG_NUM_MIN_PICKS */
export const LHC_ERQUANZHONG_FUSHI_MIN_PICKS = LHC_ERQUANZHONG_NUM_MIN_PICKS
/** @deprecated 使用 LHC_ERQUANZHONG_NUM_MAX_PICKS */
export const LHC_ERQUANZHONG_FUSHI_MAX_PICKS = LHC_ERQUANZHONG_NUM_MAX_PICKS

type LhcErquanzhongConfigLike = {
  playTemplate?: string
  betMode?: string
  playTypeId?: string
  subPlayId?: string
  catalogSubId?: string
  playTypeLabel?: string
  playMethodLabel?: string
  guajiGroup?: string
}

function lhcErquanzhongContext(config: LhcErquanzhongConfigLike) {
  const tpl = String(config.playTemplate ?? '')
  const bm = String(config.betMode ?? '').trim().toLowerCase()
  const typeId = String(config.playTypeId ?? '').trim().toLowerCase()
  const typeLabel = String(config.playTypeLabel ?? '').trim()
  const method = String(config.playMethodLabel ?? '').trim()
  const group = String(config.guajiGroup ?? '').trim()
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  const text = `${typeLabel} ${method} ${group}`
  return { tpl, bm, typeId, typeLabel, method, group, sid, text }
}

/** 二全中·复式：号池 01–49，2–10 个，逗号分隔 */
export function isLhcErquanzhongFushiConfig(config: LhcErquanzhongConfigLike): boolean {
  const { tpl, bm, typeId, typeLabel, method, group, sid, text } = lhcErquanzhongContext(config)
  if (tpl && tpl !== 'lhc_std') return false
  if (bm === 'tuotou' || bm.endsWith('_dp')) return false
  if (/拖头|对碰/.test(text)) return false
  // 目录 rule：二全中复式 279（历史误写 277=正5特，兼容保留）
  if (sid === '279' || sid === '277') return true
  if (typeId === 'erquanzhong' || typeLabel === '二全中' || group === '二全中') {
    return bm === 'fushi' || bm === '' || /复式/.test(method)
  }
  return /二全中/.test(text) && /复式/.test(method)
}

/** 二全中·拖头：号池 01–49，2–10 个，逗号分隔（落库扁选；下单合成 胆|拖） */
export function isLhcErquanzhongTuotouConfig(config: LhcErquanzhongConfigLike): boolean {
  const { tpl, bm, typeId, typeLabel, method, group, sid, text } = lhcErquanzhongContext(config)
  if (tpl && tpl !== 'lhc_std') return false
  if (bm.endsWith('_dp') || /对碰/.test(text)) return false
  // 目录 rule：二全中拖头 280（历史误写 278=正6特，兼容保留）
  if (sid === '280' || sid === '278') return true
  if (bm === 'tuotou') {
    return (
      typeId === 'erquanzhong' ||
      typeLabel === '二全中' ||
      group === '二全中' ||
      /二全中/.test(text)
    )
  }
  return /二全中/.test(text) && /拖头/.test(method + text)
}

/** 二全中复式或拖头：共用逗号号池输入框 */
export function isLhcErquanzhongNumInputConfig(config: LhcErquanzhongConfigLike): boolean {
  return isLhcErquanzhongFushiConfig(config) || isLhcErquanzhongTuotouConfig(config)
}

/** 生肖对碰最多同时选 2 个生肖（落库/下单 肖A|肖B） */
export const LHC_SX_DUIPENG_MAX_PICKS = 2
export const LHC_SX_DUIPENG_MIN_PICKS = 2

/** 尾数对碰最多同时选 2 个尾数（落库/下单 尾A|尾B） */
export const LHC_WS_DUIPENG_MAX_PICKS = 2
export const LHC_WS_DUIPENG_MIN_PICKS = 2

/** 生尾对碰：恰好 1 生肖 + 1 尾（落库 肖|尾） */
export const LHC_SW_DUIPENG_MAX_PICKS = 2
export const LHC_SW_DUIPENG_MIN_PICKS = 2

/** 生尾对碰选项宇宙：十二生肖 + 0–9 尾 */
export const LHC_SW_DUIPENG_OPTIONS = [...LHC_ZODIACS, ...LHC_TAIL_OPTIONS] as const

/** 生尾对碰随机：固定各抽 1 肖 + 1 尾（勿从混合宇宙 slice k） */
export function pickRandomLhcSwDuipengPair(): [string, string] {
  const z = LHC_ZODIACS[Math.floor(Math.random() * LHC_ZODIACS.length)]!
  const t = LHC_TAIL_OPTIONS[Math.floor(Math.random() * LHC_TAIL_OPTIONS.length)]!
  return [z, t]
}

/** 二全中·生肖对碰（目录 281；betMode=sx_dp） */
export function isLhcSxDuipengConfig(config: LhcErquanzhongConfigLike): boolean {
  const { tpl, bm, typeId, typeLabel, method, group, sid, text } = lhcErquanzhongContext(config)
  if (tpl && tpl !== 'lhc_std') return false
  if (sid === '281') return true
  if (bm === 'sx_dp') {
    return (
      typeId === 'erquanzhong' ||
      typeId === 'g003' ||
      typeLabel === '二全中' ||
      typeLabel === '连码' ||
      group === '二全中' ||
      /二全中|连码/.test(text)
    )
  }
  return /二全中/.test(text) && /生肖对碰/.test(method + text)
}

/** 二全中·尾数对碰（目录 282；betMode=ws_dp） */
export function isLhcWsDuipengConfig(config: LhcErquanzhongConfigLike): boolean {
  const { tpl, bm, typeId, typeLabel, method, group, sid, text } = lhcErquanzhongContext(config)
  if (tpl && tpl !== 'lhc_std') return false
  if (sid === '282' || sid === '288' || sid === '294') return true
  if (bm === 'ws_dp') {
    return (
      typeId === 'erquanzhong' ||
      typeId === 'g003' ||
      typeLabel === '二全中' ||
      typeLabel === '连码' ||
      group === '二全中' ||
      /二全中|连码/.test(text)
    )
  }
  return /二全中/.test(text) && /尾数对碰/.test(method + text)
}

/** 二全中/二中特/特串·生尾对碰（目录 283/289/295；betMode=sw_dp） */
export function isLhcSwDuipengConfig(config: LhcErquanzhongConfigLike): boolean {
  const { tpl, bm, typeId, typeLabel, method, group, sid, text } = lhcErquanzhongContext(config)
  if (tpl && tpl !== 'lhc_std') return false
  if (sid === '283' || sid === '289' || sid === '295' || sid === 'sw_dp') return true
  // 与后端 isLHCSwDuipengPlayRule 对齐：betMode=sw_dp 即生尾对碰（勿再卡 typeId，否则二中特/特串会漏判）
  if (bm === 'sw_dp') return true
  return (
    /生尾对碰/.test(method + text) &&
    (typeId === 'erquanzhong' ||
      typeId === 'erzhongte' ||
      typeId === 'techuan' ||
      typeId === 'g003' ||
      typeId === 'g004' ||
      typeId === 'g005' ||
      typeLabel === '二全中' ||
      typeLabel === '二中特' ||
      typeLabel === '特串' ||
      typeLabel === '连码' ||
      group === '二全中' ||
      /二全中|二中特|特串|连码/.test(text))
  )
}

/** 特码 / 正特码（非特码头尾） */
export function isLhcTemaPlayConfig(config: {
  playTemplate?: string
  betMode?: string
  playTypeId?: string
  subPlayId?: string
  playTypeLabel?: string
  playMethodLabel?: string
}): boolean {
  const tpl = String(config.playTemplate ?? '')
  if (tpl && tpl !== 'lhc_std') return false
  const bm = String(config.betMode ?? '').trim()
  if (bm === 'tematouwei') return false
  if (bm === 'tema' || bm === 'zhengte') return true
  const tid = String(config.playTypeId ?? '').trim()
  if (tid === 'g001' || tid === 'g002') return true
  const sub = String(config.subPlayId ?? '').trim()
  if (sub === 'tema' || sub === 'zhengte') return true
  const typeLabel = String(config.playTypeLabel ?? '').trim()
  const method = String(config.playMethodLabel ?? '').trim()
  if (/特码头尾/.test(`${typeLabel}${method}`)) return false
  if (typeLabel === '特码' || typeLabel === '正特码') return true
  return /特码A|特码B/.test(method)
}

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
