/**
 * 倍投设定 Tab 玩法分类（P0 门禁）
 * 产品已屏蔽小白 / 一键，界面仅简单 + 高级两 Tab。
 * 分类仍用于其它倍投策略推断（若有）。
 */

export type BetMultiplierPlayCategory =
  | 'locate'
  | 'sides'
  | 'locate_like'
  | 'multi_star'
  | 'combo_group'
  | 'sum_span'
  | 'budingwei_multi'
  | 'renxuan_multi'
  | 'pk10_multi'
  | 'fun'
  | 'unknown'

export interface BetMultiplierPlayContext {
  playTypeId?: string
  subPlayId?: string
  betMode?: string
  playTypeLabel?: string
  subPlayLabel?: string
  playMethod?: string
  playTemplate?: string
  segmentLen?: number
}

function norm(s: string | undefined): string {
  return String(s ?? '')
    .trim()
    .toLowerCase()
}

function textBlob(ctx: BetMultiplierPlayContext): string {
  return [
    ctx.playTypeLabel,
    ctx.subPlayLabel,
    ctx.playMethod,
    ctx.playTypeId,
    ctx.subPlayId,
    ctx.betMode,
  ]
    .map((x) => String(x ?? '').trim())
    .filter(Boolean)
    .join(' ')
}

/**
 * 由玩法上下文推导倍投 Tab 分类。
 * 优先 betMode / typeId；再回退中文标签。
 */
export function resolveBetMultiplierPlayCategory(
  ctx: BetMultiplierPlayContext,
): BetMultiplierPlayCategory {
  const bm = norm(ctx.betMode)
  const typeId = norm(ctx.playTypeId)
  const subId = norm(ctx.subPlayId)
  const typeLabel = String(ctx.playTypeLabel ?? '').trim()
  const subLabel = String(ctx.subPlayLabel ?? ctx.playMethod ?? '').trim()
  const blob = textBlob(ctx)
  const template = norm(ctx.playTemplate)
  const segLen = ctx.segmentLen ?? 0

  // —— 两面 / 龙虎（含前二/前三大小单双，属 sides 非 multi_star）——
  if (
    bm === 'longhu' ||
    bm === 'longhuhe' ||
    bm === 'daxiao' ||
    bm === 'danshuang' ||
    bm === 'dxds' ||
    bm === 'zhuangxian'
  ) {
    return 'sides'
  }
  if (
    typeId === 'longhu' ||
    typeLabel === '龙虎' ||
    typeLabel.includes('大小单双') ||
    typeLabel === '大小' ||
    typeLabel === '单双' ||
    subLabel.includes('大小单双') ||
    (subLabel.includes('龙虎') && !subLabel.includes('龙虎豹'))
  ) {
    return 'sides'
  }
  if (blob.includes('质合') || bm === 'zhihe') return 'sides'

  // —— 定位 / 定胆 / 猜冠军 ——
  if (bm === 'dingwei' || typeId === 'dingwei' || typeId === 'qian1') {
    return 'locate'
  }
  if (
    typeLabel === '一星' ||
    typeLabel.includes('定位胆') ||
    subLabel.includes('定位胆') ||
    subLabel.includes('定胆') ||
    typeLabel === '猜冠军' ||
    subLabel.includes('猜冠军')
  ) {
    return 'locate'
  }
  if (template === 'pk10_std' && (typeId === 'qian1' || segLen === 1) && bm !== 'hezhi') {
    if (bm === 'fushi' || bm === 'danshi') {
      // 单位置复式仍算 locate
      return 'locate'
    }
  }

  // —— locate_like：一码不定位、任选一、前一 ——
  if (bm === 'budingwei' && (segLen <= 1 || subLabel.includes('一码') || blob.includes('一码不定位'))) {
    return 'locate_like'
  }
  if (
    blob.includes('一码不定位') ||
    blob.includes('任选一中一') ||
    blob.includes('任选一') ||
    typeId === 'renxuan_yi' ||
    (typeId.includes('renxuan') && (subLabel.includes('一中一') || subId.includes('rx1') || subId.includes('1z1')))
  ) {
    return 'locate_like'
  }
  if (blob.includes('精确前一') || (blob.includes('前一') && !blob.includes('前二') && !blob.includes('前三'))) {
    return 'locate_like'
  }

  // —— 趣味 ——
  if (
    bm === 'teshu' ||
    blob.includes('报喜') ||
    blob.includes('豹子') ||
    blob.includes('对子') ||
    blob.includes('顺子') ||
    blob.includes('半顺') ||
    blob.includes('杂六') ||
    typeLabel.includes('趣味')
  ) {
    return 'fun'
  }

  // —— 和值 / 跨度 ——
  if (bm === 'hezhi' || bm === 'kuadu' || typeId === 'hezhi' || typeId === 'kuadu') {
    return 'sum_span'
  }
  if (subLabel.includes('和值') || subLabel.includes('跨度') || typeLabel.includes('和值')) {
    return 'sum_span'
  }

  // —— 组选形态 / 包胆 ——
  if (
    bm === 'zu3' ||
    bm === 'zu6' ||
    bm === 'baodan' ||
    bm === 'zuxuan_fs' ||
    bm === 'zuxuan_ds' ||
    bm === 'hunhe' ||
    ['zu24', 'zu12', 'zu60', 'zu30', 'zu120', 'zu20', 'zu10', 'zu5', 'zu4'].includes(bm)
  ) {
    return 'combo_group'
  }
  if (
    subLabel.includes('组三') ||
    subLabel.includes('组六') ||
    subLabel.includes('包胆') ||
    subLabel.includes('组选') ||
    typeLabel.includes('组选')
  ) {
    return 'combo_group'
  }

  // —— 多码不定位 ——
  if (bm === 'budingwei' || typeId === 'budingwei' || typeLabel.includes('不定位')) {
    return 'budingwei_multi'
  }
  if (subLabel.includes('二码不定位') || subLabel.includes('三码不定位') || subLabel.includes('不定位')) {
    return 'budingwei_multi'
  }

  // —— 任选多码 ——
  if (
    typeId === 'renxuan' ||
    typeId.startsWith('renxuan') ||
    typeLabel.includes('任选') ||
    typeLabel.startsWith('任二') ||
    typeLabel.startsWith('任三') ||
    typeLabel.startsWith('任四')
  ) {
    if (blob.includes('一中一') || blob.includes('任选一')) return 'locate_like'
    return 'renxuan_multi'
  }

  // —— PK10 多位（仅 pk10 模板；SSC 的 qian2/qian3 走下方 multi_star）——
  if (template === 'pk10_std') {
    if (typeId === 'qian1' || segLen === 1) {
      if (bm === 'hezhi' || bm === 'dxds' || bm === 'daxiao' || bm === 'danshuang') {
        // 已在前面处理；兜底
        if (bm === 'hezhi') return 'sum_span'
        return 'sides'
      }
      return 'locate'
    }
    if (
      typeId === 'qian2' ||
      typeId === 'qian3' ||
      typeId === 'qian4' ||
      typeId === 'qian5' ||
      blob.includes('猜前二') ||
      blob.includes('猜前三') ||
      blob.includes('猜前四') ||
      blob.includes('猜前五') ||
      blob.includes('冠亚军')
    ) {
      return 'pk10_multi'
    }
    if (segLen > 1) return 'pk10_multi'
  }
  if (
    blob.includes('猜前二') ||
    blob.includes('猜前三') ||
    blob.includes('猜前四') ||
    blob.includes('猜前五') ||
    blob.includes('冠亚军复式')
  ) {
    return 'pk10_multi'
  }

  // —— 多星直选（二～五星、前后中）——
  if (
    typeId === 'qian2' ||
    typeId === 'hou2' ||
    typeId === 'qian3' ||
    typeId === 'zhong3' ||
    typeId === 'hou3' ||
    typeId === 'qian4' ||
    typeId === 'hou4' ||
    typeId === 'wuxing' ||
    typeId === 'sixing' ||
    typeLabel.includes('二星') ||
    typeLabel.includes('三星') ||
    typeLabel.includes('四星') ||
    typeLabel.includes('五星') ||
    typeLabel.includes('前二') ||
    typeLabel.includes('后二') ||
    typeLabel.includes('前三') ||
    typeLabel.includes('中三') ||
    typeLabel.includes('后三') ||
    typeLabel.includes('前四') ||
    typeLabel.includes('后四')
  ) {
    // 前二/前三大小单双已在 sides 处理；此处是直选/组选大类
    if (subLabel.includes('大小') || subLabel.includes('单双')) return 'sides'
    return 'multi_star'
  }

  if (bm === 'fushi' || bm === 'danshi' || bm === 'zhixuan_fs' || bm === 'zhixuan_ds' || bm === 'zuhe') {
    if (segLen <= 1) return 'locate'
    return 'multi_star'
  }

  // 缺省：无明确自动算表依据 → 两 Tab（稳妥）
  return 'unknown'
}

/** 是否展示小白 + 一键（自动算表）。产品已屏蔽，仅保留简单 / 高级。 */
export function showAutoGenBetMultiplierTabs(_ctx: BetMultiplierPlayContext): boolean {
  void _ctx
  return false
}

/** 运行时持久化 kind：仅简单(2) / 高级(3) */
export function normalizeBetMultiplierPersistKind(
  tab: string,
): '2' | '3' {
  return tab === '3' ? '3' : '2'
}
