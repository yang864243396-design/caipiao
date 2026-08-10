import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'

/** rules/v2 同步后 PC28 高级开某投某支持的子玩法 label */
export const ADV_TRIGGER_PC28_SUB_LABELS = new Set(['和值', '大小单双', '龙虎豹'])

/** 旧 sub_id 兼容 */
export const ADV_TRIGGER_PC28_SUBS = new Set(['hezhi', 'dxds', 'longhubao'])

/** rules/v2 同步后支持的玩法类型 label（groups[].name） */
export const ADV_TRIGGER_PLAY_TYPE_LABELS = new Set([
  '一星',
  '龙虎',
  '2.0模式',
  '2.8模式',
  '特码',
  '正特码',
])

/** 旧 type_id 兼容 */
export const ADV_TRIGGER_PLAY_TYPES = new Set([
  'dingwei',
  'longhu',
  'pc28_20',
  'pc28_28',
  'tema',
  'zhengte',
  'g001',
  'g002',
])

export const PC28_MODE_LABELS = new Set(['2.0模式', '2.8模式'])

export function guajiGroupFromSegment(rule: unknown): string {
  if (rule && typeof rule === 'object' && 'guajiGroup' in rule) {
    return String((rule as { guajiGroup?: string }).guajiGroup ?? '').trim()
  }
  return ''
}

export function guajiFullNameFromSegment(rule: unknown): string {
  if (rule && typeof rule === 'object' && 'guajiFullName' in rule) {
    return String((rule as { guajiFullName?: string }).guajiFullName ?? '').trim()
  }
  return ''
}

export function guajiTeamFromSegment(rule: unknown): string {
  if (rule && typeof rule === 'object' && 'guajiTeam' in rule) {
    return String((rule as { guajiTeam?: string }).guajiTeam ?? '').trim()
  }
  return ''
}

/** 同一组选号覆盖段数（前中后三=3，前后二/三/四=2） */
export function segmentBetMultiplier(guajiGroup: string): number {
  switch (guajiGroup.trim()) {
    case '前中后三':
      return 3
    case '前后三':
    case '前后二':
    case '前后四':
      return 2
    default:
      return 1
  }
}

export function findPlayTypeNode(
  playTreeTypes: PlayTypeNode[],
  typeId: string,
): PlayTypeNode | undefined {
  return playTreeTypes.find((t) => t.typeId === String(typeId ?? '').trim())
}

export function findSubPlayNode(
  typeNode: PlayTypeNode | undefined,
  subId: string,
): SubPlayNode | undefined {
  return typeNode?.subPlays.find((s) => s.subId === String(subId ?? '').trim())
}

export function isLonghuPlayType(typeLabel: string, typeId: string): boolean {
  return typeLabel.trim() === '龙虎' || typeId === 'longhu'
}

export function isPc28ModeType(typeLabel: string, typeId: string): boolean {
  const label = typeLabel.trim()
  return PC28_MODE_LABELS.has(label) || typeId === 'pc28_20' || typeId === 'pc28_28'
}

export function isDingweiStarType(typeLabel: string, typeId: string, subLabel = ''): boolean {
  const label = typeLabel.trim()
  const id = typeId.trim()
  // g006 = rules/v2 定位胆/一星；仅认 dingwei 会漏掉新建方案的 typeId
  return label === '一星' || id === 'dingwei' || id === 'g006' || subLabel.includes('定位胆')
}

/** rules/v2 同步后 bet_mode 可能为空，按 label / guajiGroup 推断（对齐后端 InferBetMode） */
export function inferBetModeFromCatalog(
  typeNode: Pick<PlayTypeNode, 'typeId' | 'label'>,
  subNode: Pick<SubPlayNode, 'subId' | 'label' | 'segmentRule'>,
  playTemplate = '',
): string {
  const typeLabel = typeNode.label.trim()
  const typeId = typeNode.typeId.trim()
  const subLabel = subNode.label.trim()
  const subId = subNode.subId.trim()
  const group = guajiGroupFromSegment(subNode.segmentRule)
  const fullName = guajiFullNameFromSegment(subNode.segmentRule)
  const text = `${group} ${typeLabel} ${fullName} ${subLabel} ${subId}`

  // 六合彩
  if (playTemplate === 'lhc_std') {
    if (typeId === 'g001' || typeLabel === '特码') return 'tema'
    if (typeId === 'g002' || typeLabel === '正特码') return 'zhengte'
    if (typeId === 'g013' && subLabel.includes('复式')) return 'buzhong'
    if (typeId === 'g014' && subLabel.includes('复式')) return 'xuanyi'
    if (subLabel.includes('拖头')) return 'tuotou'
    if (subLabel.includes('生肖对碰')) return 'sx_dp'
    if (subLabel.includes('尾数对碰')) return 'ws_dp'
    if (subLabel.includes('尾数对碰')) return 'ws_dp'
    if (subLabel.includes('生尾对碰')) return 'sw_dp'
    if (subLabel.includes('任意对碰')) return 'renyi_dp'
    if (subLabel.includes('特肖')) return 'texiao'
    if (subLabel.includes('总肖')) return 'zongxiao'
    if (subLabel.includes('特码头尾') || typeLabel === '特码头尾') return 'tematouwei'
    if (subLabel.includes('过关') || typeLabel === '过关') return 'guoguan'
    if (subLabel.includes('七码') || typeLabel === '七码') return 'qima'
    if (subLabel.includes('任中')) return 'renzhong'
    if (subLabel.includes('半半波')) return 'banbanbo'
    if (subLabel.includes('半波')) return 'banbo'
    if (subLabel.includes('波色') || typeLabel === '波色') return 'bose'
    if (subLabel.includes('五行')) return 'wuxing'
    if (subLabel.includes('家野')) return 'jiaye'
    if (subLabel.includes('复式') || subLabel === '复式') return 'fushi'
    if (subLabel.includes('尾数') && subLabel.includes('不中')) return 'wei_bz'
    if (subLabel.includes('尾数')) return 'weishu'
    if (subLabel.includes('肖') && subLabel.includes('不中')) return 'xiao_bz'
    if (subLabel.includes('肖')) return 'xiao'
    if (subLabel.includes('不中')) return 'buzhong'
    if (subLabel.includes('选中一')) return 'xuanyi'
    if (typeLabel === '连码' || typeId === 'g003') return 'fushi'
    if (typeLabel === '生肖' || typeId === 'g005') return 'xiao'
    if (typeLabel === '全不中') return 'buzhong'
    if (typeLabel === '多选中一') return 'xuanyi'
  }

  // PK10：g010=和值（勿与时时彩龙虎 g010 混淆）
  if (playTemplate === 'pk10_std') {
    if (typeId === 'g010' || typeLabel === '和值' || group === '和值') {
      if (subLabel.includes('大小') || subLabel.includes('单双')) return 'dxds'
      return 'hezhi'
    }
    if (typeId === 'g008' || typeLabel === '大小' || group === '大小') return 'daxiao'
    if (typeId === 'g009' || typeLabel === '单双' || group === '单双') return 'danshuang'
    if (typeId === 'g001' || group === '一星' || subLabel.includes('定位胆')) return 'dingwei'
    if (isLonghuPlayType(typeLabel, typeId) || group === '龙虎') return 'longhu'
    if (subLabel.includes('直选复式') || subLabel.includes('复式')) return 'fushi'
    if (subLabel.includes('直选单式') || subLabel.includes('单式')) return 'danshi'
  }

  if (isLonghuPlayType(typeLabel, typeId) || group === '龙虎') {
    if (subLabel.includes('和') || fullName.includes('龙虎和') || subId.includes('_he')) return 'longhuhe'
    return 'longhu'
  }
  if (isPc28ModeType(typeLabel, typeId) || PC28_MODE_LABELS.has(group)) {
    if (subLabel === '和值' || subId === 'hezhi') return 'hezhi'
    if (subLabel === '大小单双' || subId === 'dxds') return 'dxds'
    if (subLabel === '龙虎豹' || subId === 'longhubao') return 'longhubao'
    if (subLabel === '特殊号' || subId === 'teshu') return 'teshu'
  }
  if (isDingweiStarType(typeLabel, typeId, subLabel) || group === '一星' || subLabel.includes('定位胆')) {
    return 'dingwei'
  }
  if (subLabel.includes('组选复式')) return 'zuxuan_fs'
  if (subLabel.includes('组选单式')) return 'zuxuan_ds'
  if (subLabel.includes('直选复式') || (subLabel.includes('复式') && subLabel.includes('直选'))) {
    return 'fushi'
  }
  if (subLabel.includes('直选单式') || (subLabel.includes('单式') && subLabel.includes('直选'))) {
    return 'danshi'
  }
  if (subLabel.includes('组选和值')) return 'hezhi'
  if (subLabel.includes('直选和值') || (subLabel === '和值' && !subLabel.includes('尾数'))) return 'hezhi'
  if (subLabel.includes('和值') && !subLabel.includes('单双') && !subLabel.includes('大小') && !subLabel.includes('尾数')) {
    return 'hezhi'
  }
  if (subLabel.includes('跨度')) return 'kuadu'
  if (subLabel.includes('混合')) return 'hunhe'
  if (subLabel === '组合' || subLabel.includes('组合')) return 'zuhe'
  if (subLabel.includes('组三') && subLabel.includes('单式')) return 'zuxuan_ds'
  if (subLabel.includes('组六') && subLabel.includes('单式')) return 'zuxuan_ds'
  if (subLabel.includes('组三')) return 'zu3'
  if (subLabel.includes('组六') && !subLabel.includes('组选6') && !subLabel.includes('组选60')) return 'zu6'
  if (subLabel.includes('包胆')) return 'baodan'
  if (subLabel.includes('和值单双') || subLabel.includes('尾数单双')) return 'danshuang'
  if (subLabel.includes('和值大小') || subLabel.includes('尾数大小')) return 'daxiao'
  if (subLabel.includes('幸运庄闲') || subLabel.includes('庄闲')) return 'zhuangxian'
  if (subLabel.includes('和值尾数') || (subLabel.includes('尾数') && !subLabel.includes('单双') && !subLabel.includes('大小'))) {
    return 'weishu'
  }
  if (
    subLabel.includes('特殊号') ||
    subLabel.includes('一帆风顺') ||
    subLabel.includes('好事成双') ||
    subLabel.includes('三星报喜') ||
    subLabel.includes('四季发财')
  ) {
    return 'teshu'
  }
  if (subLabel.includes('不定位') || group === '不定位') return 'budingwei'
  if (subLabel.includes('组选120') || text.includes('zu120')) return 'zu120'
  if (subLabel.includes('组选60') || text.includes('zu60')) return 'zu60'
  if (subLabel.includes('组选30') || text.includes('zu30')) return 'zu30'
  if (subLabel.includes('组选24') || text.includes('zu24')) return 'zu24'
  if (subLabel.includes('组选20') || text.includes('zu20')) return 'zu20'
  if (subLabel.includes('组选12') || text.includes('zu12')) return 'zu12'
  if (subLabel.includes('组选10') || text.includes('zu10')) return 'zu10'
  if (subLabel.includes('组选5') || text.includes('zu5')) return 'zu5'
  if (subLabel.includes('组选4') || text.includes('zu4')) return 'zu4'
  if (subLabel.includes('组选6') || text.includes('zu6')) return 'zu6'
  if (subLabel.includes('大小') || subLabel.includes('单双') || group === '大小单双') return 'dxds'
  if (playTemplate === 'k3_std' && (typeLabel === '和值' || typeId === 'hezhi' || typeId === 'g001')) return 'hezhi'
  if (playTemplate === 'syxw_std') {
    if (typeId === 'g006' || typeId === 'renxuan_ds') return 'danshi'
    if (typeId === 'g005' || typeId === 'renxuan_fs') return 'fushi'
    if (typeId === 'g004' || group === '不定位') return 'budingwei'
    if (typeId === 'g003' || group === '一星') return 'dingwei'
  }
  if (playTemplate === 'k3_std') {
    if (subLabel.includes('复选') || subLabel.includes('标准选号')) return 'fushi'
    if (subLabel.includes('手动输入') || subLabel.includes('三连号')) return 'danshi'
    if (typeLabel === '单挑一骰' || typeId === 'g007') return 'fushi'
  }
  return ''
}

function isLhcTemaAdvTriggerType(typeId: string, typeLabel: string): boolean {
  const pt = String(typeId ?? '').trim()
  const label = String(typeLabel ?? '').trim()
  return (
    label === '特码' ||
    label === '正特码' ||
    pt === 'tema' ||
    pt === 'zhengte' ||
    pt === 'g001' ||
    pt === 'g002'
  )
}

/** 二全中复式（非拖头/对碰）开放高级开某投某 */
function isLhcErquanzhongFushiAdvTrigger(
  playTypeId: string,
  subPlayId: string,
  typeLabel: string,
  subLabel: string,
): boolean {
  const pt = String(playTypeId ?? '').trim()
  const sub = String(subPlayId ?? '').trim()
  const label = String(typeLabel ?? '').trim()
  const subLbl = String(subLabel ?? '').trim()
  if (/拖头|对碰/.test(subLbl)) return false
  // 目录：二全中复式 279（兼容旧误写 277）
  if (sub === '279' || sub === '277') return true
  if (pt === 'erquanzhong' || label === '二全中') {
    return sub === 'fushi' || sub === '' || /复式/.test(subLbl) || subLbl === ''
  }
  if (label === '连码' || pt === 'g003') {
    return /二全中/.test(subLbl) && /复式/.test(subLbl)
  }
  return /二全中/.test(`${label}${subLbl}`) && /复式/.test(subLbl)
}

/** 二全中生肖对碰：高级开某投某（开出=特码生肖，正/反投=两个生肖） */
export function isLhcSxDuipengAdvTrigger(
  playTypeId: string,
  subPlayId?: string,
  typeLabel?: string,
  subLabel?: string,
): boolean {
  const pt = String(playTypeId ?? '').trim()
  const sub = String(subPlayId ?? '').trim()
  const label = String(typeLabel ?? '').trim()
  const subLbl = String(subLabel ?? '').trim()
  if (sub === '281' || sub === 'sx_dp') return true
  if (/生肖对碰/.test(subLbl)) {
    return (
      pt === 'erquanzhong' ||
      pt === 'g003' ||
      label === '二全中' ||
      label === '连码' ||
      /二全中|连码/.test(`${label}${subLbl}`)
    )
  }
  return false
}

/** 二全中尾数对碰：高级开某投某（开出=特码尾数，正/反投=两个尾数） */
export function isLhcWsDuipengAdvTrigger(
  playTypeId: string,
  subPlayId?: string,
  typeLabel?: string,
  subLabel?: string,
): boolean {
  const pt = String(playTypeId ?? '').trim()
  const sub = String(subPlayId ?? '').trim()
  const label = String(typeLabel ?? '').trim()
  const subLbl = String(subLabel ?? '').trim()
  if (sub === '282' || sub === 'ws_dp' || sub === '288' || sub === '294') return true
  if (/尾数对碰/.test(subLbl)) {
    return (
      pt === 'erquanzhong' ||
      pt === 'g003' ||
      label === '二全中' ||
      label === '连码' ||
      /二全中|连码/.test(`${label}${subLbl}`)
    )
  }
  return false
}

/** 二全中生尾对碰：高级开某投某（开出=特码生肖或特码尾，正/反投=1肖+1尾） */
export function isLhcSwDuipengAdvTrigger(
  playTypeId: string,
  subPlayId?: string,
  typeLabel?: string,
  subLabel?: string,
): boolean {
  const pt = String(playTypeId ?? '').trim()
  const sub = String(subPlayId ?? '').trim()
  const label = String(typeLabel ?? '').trim()
  const subLbl = String(subLabel ?? '').trim()
  if (sub === '283' || sub === 'sw_dp' || sub === '289' || sub === '295') return true
  if (/生尾对碰/.test(subLbl)) {
    return (
      pt === 'erquanzhong' ||
      pt === 'g003' ||
      label === '二全中' ||
      label === '连码' ||
      /二全中|连码/.test(`${label}${subLbl}`)
    )
  }
  return false
}

export function supportsAdvTriggerBet(
  playTypeId: string,
  subPlayId?: string,
  typeLabel?: string,
  subLabel?: string,
): boolean {
  const pt = String(playTypeId ?? '').trim()
  const sub = String(subPlayId ?? '').trim()
  const label = String(typeLabel ?? '').trim()
  const subLbl = String(subLabel ?? '').trim()

  if (pt === 'dingwei' || pt === 'longhu' || label === '一星' || label === '龙虎') return true
  if (isLhcTemaAdvTriggerType(pt, label)) return true
  if (isLhcSxDuipengAdvTrigger(pt, sub, label, subLbl)) return true
  if (isLhcWsDuipengAdvTrigger(pt, sub, label, subLbl)) return true
  if (isLhcSwDuipengAdvTrigger(pt, sub, label, subLbl)) return true
  if (isLhcErquanzhongFushiAdvTrigger(pt, sub, label, subLbl)) return true
  if (pt === 'pc28_20' || pt === 'pc28_28' || label === '2.0模式' || label === '2.8模式') {
    return ADV_TRIGGER_PC28_SUB_LABELS.has(subLbl) || ADV_TRIGGER_PC28_SUBS.has(sub)
  }
  return false
}

export function lotteryHasAdvTriggerPlay(playTypes: PlayTypeNode[]): boolean {
  for (const t of playTypes) {
    const label = t.label.trim()
    if (label === '一星' || label === '龙虎' || t.typeId === 'dingwei' || t.typeId === 'longhu') {
      return true
    }
    if (isLhcTemaAdvTriggerType(t.typeId, label)) {
      return true
    }
    if (
      t.subPlays?.some(
        (s) =>
          isLhcErquanzhongFushiAdvTrigger(t.typeId, s.subId, label, s.label.trim()) ||
          isLhcSxDuipengAdvTrigger(t.typeId, s.subId, label, s.label.trim()) ||
          isLhcWsDuipengAdvTrigger(t.typeId, s.subId, label, s.label.trim()) ||
          isLhcSwDuipengAdvTrigger(t.typeId, s.subId, label, s.label.trim()),
      )
    ) {
      return true
    }
    if (isPc28ModeType(label, t.typeId)) {
      if (
        t.subPlays?.some(
          (s) => ADV_TRIGGER_PC28_SUB_LABELS.has(s.label.trim()) || ADV_TRIGGER_PC28_SUBS.has(s.subId),
        )
      ) {
        return true
      }
    }
  }
  return false
}

export function filterPlayTypesForRunType<T extends { value: string | number }>(
  runTypeId: string,
  all: T[],
  playTreeTypes: PlayTypeNode[],
): T[] {
  // 对齐 V8：运行类型与玩法正交、无门禁——任意运行类型可配任意玩法类型。
  void runTypeId
  void playTreeTypes
  return all
}

export function filterRandomDrawPlayTypes<T extends { value: string | number }>(
  all: T[],
  playTreeTypes: PlayTypeNode[],
): T[] {
  // 随机出号采用"选项宇宙+抽样"，覆盖按位/单式/组选/属性家族——仅保留至少有一个受支持子玩法的玩法类型。
  return all.filter((o) => {
    const id = String(o.value)
    const node = findPlayTypeNode(playTreeTypes, id)
    if (!node) return true
    const lab = node.label.trim()
    const subs = node.subPlays ?? []
    if (!subs.length) return supportsRandomDrawSubPlay(lab, lab)
    return subs.some((s) => supportsRandomDrawSubPlay(s.label, lab))
  })
}

export function filterSubPlaysForRunType<T extends { value: string | number; label?: string }>(
  runTypeId: string,
  all: T[],
  playTypeId: string,
  playTreeTypes: PlayTypeNode[],
): T[] {
  // 对齐 V8：运行类型与玩法正交、无门禁——任意运行类型可配任意子玩法。
  void runTypeId
  void playTypeId
  void playTreeTypes
  return all
}

export function filterAdvTriggerPlayTypes<T extends { value: string | number }>(
  all: T[],
  playTreeTypes: PlayTypeNode[],
): T[] {
  return all.filter((o) => {
    const id = String(o.value)
    const node = findPlayTypeNode(playTreeTypes, id)
    if (node) {
      const label = node.label.trim()
      if (ADV_TRIGGER_PLAY_TYPE_LABELS.has(label)) return true
    }
    return ADV_TRIGGER_PLAY_TYPES.has(id)
  })
}

export function filterAdvTriggerSubPlays<T extends { value: string | number }>(
  all: T[],
  playTypeId: string,
  playTreeTypes: PlayTypeNode[],
): T[] {
  const node = findPlayTypeNode(playTreeTypes, playTypeId)
  const label = node?.label.trim() ?? ''
  if (!isPc28ModeType(label, playTypeId)) return all
  return all.filter((o) => {
    const subId = String(o.value)
    const sub = findSubPlayNode(node, subId)
    if (sub) {
      return ADV_TRIGGER_PC28_SUB_LABELS.has(sub.label.trim()) || ADV_TRIGGER_PC28_SUBS.has(subId)
    }
    return ADV_TRIGGER_PC28_SUBS.has(subId)
  })
}

export function filterHotColdWarmPlayTypes<T extends { value: string | number }>(
  all: T[],
  playTreeTypes: PlayTypeNode[],
): T[] {
  return all.filter((o) => {
    const id = String(o.value)
    const node = findPlayTypeNode(playTreeTypes, id)
    if (!node) return id !== 'longhu'
    if (isLonghuPlayType(node.label, id)) return false
    const lab = node.label.trim()
    if (lab === '大小单双') return false
    const subs = node.subPlays ?? []
    if (!subs.length) return supportsHotColdWarmSubPlay(lab, lab)
    return subs.some((s) => supportsHotColdWarmSubPlay(s.label, lab))
  })
}

/**
 * 冷热出号 / 随机出号仅支持「按位产号」子玩法：
 * 直选复式、直选组合、定位胆、任选直选复式。
 * 单式/和值/组三组六/包胆/不定位/属性等须用定码轮换。
 */
export function supportsPositionSourceSubPlay(
  subLabel: string,
  playTypeLabel = '',
): boolean {
  const sub = (subLabel || '').trim()
  const play = (playTypeLabel || '').trim()
  if (!sub) return false
  if (play === '龙虎' || sub.includes('龙虎')) return false
  if (play === '大小单双' || /大小单双|和值单双|和值大小/.test(sub)) return false
  if (play === '不定位' || sub.includes('不定位')) return false
  if (/单式|混合组选/.test(sub) || (sub.includes('混合') && !sub.includes('组合'))) return false
  if (/和值|跨度|包胆|组三|组六|特殊号/.test(sub)) return false
  if (sub.includes('组选') && !sub.includes('组合')) return false
  if (play === '任选') {
    return sub.includes('直选复式') || (sub.includes('直选') && sub.includes('复式'))
  }
  if (sub.includes('组合') && !sub.includes('组选')) return true
  if (sub.includes('直选复式') || (sub.includes('复式') && sub.includes('直选'))) return true
  if (sub.includes('定位') || play === '一星') return true
  return false
}

export function filterPositionSourceSubPlays<T extends { value: string | number; label?: string }>(
  all: T[],
  playTypeId: string,
  playTreeTypes: PlayTypeNode[],
): T[] {
  const typeNode = findPlayTypeNode(playTreeTypes, playTypeId)
  const playLabel = typeNode?.label?.trim() ?? ''
  return all.filter((o) => {
    const subId = String(o.value)
    const sub = findSubPlayNode(typeNode, subId)
    const label = (sub?.label ?? o.label ?? '').trim()
    return supportsPositionSourceSubPlay(label, playLabel)
  })
}

/**
 * 随机出号支持的子玩法 = 按位型 + 单式（整注随机）。
 * 与后端 schemes.SupportsRandomDrawSubPlay 对齐。
 */
export function supportsRandomDrawSubPlay(subLabel: string, playTypeLabel = ''): boolean {
  if (supportsPositionSourceSubPlay(subLabel, playTypeLabel)) return true
  const sub = (subLabel || '').trim()
  // 直选/组选单式 / 混合组选单式（整注随机）
  if (sub.includes('单式') || sub.includes('混合')) return true
  // 组合家族：组三/组六/组选N/组选复式（号码池随机）
  if (/组三|组六|组选/.test(sub)) return true
  // 属性/聚合家族：大小单双/龙虎/特殊号/庄闲/和值/跨度/不定位/包胆；五星趣味数字池
  if (
    /大小单双|大小|单双|龙虎|庄闲|特殊号|豹子|对子|顺子|和值|跨度|不定位|包胆|一帆风顺|好事成双|三星报喜|四季发财/.test(
      sub,
    )
  ) {
    return true
  }
  return false
}

/**
 * 冷热出号支持的子玩法 = 按位型 + 号码池型 + 属性/聚合型（选项命中频次分档）。
 * 不含单式。与后端 schemes.SupportsHotColdWarmSubPlay 对齐。
 */
export function supportsHotColdWarmSubPlay(subLabel: string, playTypeLabel = ''): boolean {
  if (supportsPositionSourceSubPlay(subLabel, playTypeLabel)) return true
  const sub = (subLabel || '').trim()
  const play = (playTypeLabel || '').trim()
  if (sub.includes('单式')) return false
  // 属性/聚合家族：选项命中频次分档（特殊号→豹子/对子/顺子 等）
  if (
    /大小单双|特殊号|庄闲|龙虎豹|直选和值|组选和值|和值尾数|跨度|龙虎/.test(sub) ||
    sub === '和值' ||
    play === '龙虎' ||
    /和值|特殊号|大小单双/.test(play)
  ) {
    return true
  }
  // 号码池型：组选 + 不定位 + 包胆 + 五星趣味
  return /组三|组六|组选|不定位|包胆|一帆风顺|好事成双|三星报喜|四季发财/.test(sub)
}

export function filterHotColdWarmSubPlays<T extends { value: string | number; label?: string }>(
  all: T[],
  playTypeId: string,
  playTreeTypes: PlayTypeNode[],
): T[] {
  const typeNode = findPlayTypeNode(playTreeTypes, playTypeId)
  const playLabel = typeNode?.label?.trim() ?? ''
  return all.filter((o) => {
    const subId = String(o.value)
    const sub = findSubPlayNode(typeNode, subId)
    const label = (sub?.label ?? o.label ?? '').trim()
    return supportsHotColdWarmSubPlay(label, playLabel)
  })
}

export function filterRandomDrawSubPlays<T extends { value: string | number; label?: string }>(
  all: T[],
  playTypeId: string,
  playTreeTypes: PlayTypeNode[],
): T[] {
  const typeNode = findPlayTypeNode(playTreeTypes, playTypeId)
  const playLabel = typeNode?.label?.trim() ?? ''
  return all.filter((o) => {
    const subId = String(o.value)
    const sub = findSubPlayNode(typeNode, subId)
    const label = (sub?.label ?? o.label ?? '').trim()
    return supportsRandomDrawSubPlay(label, playLabel)
  })
}

/** 与后端 ValidateRunTypePlay 同源；返回错误文案或 null */
export function validateRunTypePlaySelection(
  runTypeId: string,
  playTypeId: string,
  subPlayId: string,
  playTreeTypes: PlayTypeNode[],
): string | null {
  // 对齐 V8：运行类型与玩法正交、无门禁——不再限制玩法。
  void runTypeId
  void playTypeId
  void subPlayId
  void playTreeTypes
  return null
}

/** 彩种 / 运行类型变更后，校正当前选中的玩法类型与子玩法 */
export function syncRunTypePlaySelection(input: {
  runTypeId: string
  playTypeId: string
  subPlayId: string
  playTreeTypes: PlayTypeNode[]
  playTypeOptions: Array<{ value: string | number }>
  subPlayOptions: Array<{ value: string | number }>
}): { playTypeId: string; subPlayId: string; runTypeId: string } {
  let { runTypeId, playTypeId, subPlayId } = input
  const { playTreeTypes } = input

  if (playTreeTypes.length > 0 && runTypeId === 'adv_trigger_bet' && !lotteryHasAdvTriggerPlay(playTreeTypes)) {
    runTypeId = 'fixed_rotate'
  }

  const filteredTypes = filterPlayTypesForRunType(
    runTypeId,
    input.playTypeOptions,
    playTreeTypes,
  )
  if (filteredTypes.length > 0 && !filteredTypes.some((o) => String(o.value) === playTypeId)) {
    playTypeId = String(filteredTypes[0]?.value ?? playTypeId)
  }

  const typeNode = findPlayTypeNode(playTreeTypes, playTypeId)
  const allSubs = (typeNode?.subPlays ?? []).map((s) => ({ label: s.label, value: s.subId }))
  const filteredSubs = filterSubPlaysForRunType(runTypeId, allSubs, playTypeId, playTreeTypes)
  if (filteredSubs.length > 0 && !filteredSubs.some((o) => String(o.value) === subPlayId)) {
    subPlayId = String(filteredSubs[0]?.value ?? subPlayId)
  }

  return { runTypeId, playTypeId, subPlayId }
}

/** 方案配置页：根据 PlayConfig 判断龙虎玩法 */
export function isLonghuPlayConfigLike(config: {
  betMode?: string
  playTypeId?: string
  playTypeLabel?: string
}): boolean {
  const bm = config.betMode ?? ''
  if (bm === 'longhubao') return false
  if (bm === 'longhu' || bm === 'longhuhe') return true
  if (isLonghuPlayType(config.playTypeLabel ?? '', config.playTypeId ?? '')) return true
  return false
}

/** 方案配置页：PC28 2.0 / 2.8 模式 */
export function isPc28ModeConfigLike(config: {
  playTypeId?: string
  playTypeLabel?: string
  playTemplate?: string
}): boolean {
  if (config.playTemplate === 'pc28_std') {
    const label = config.playTypeLabel?.trim() ?? ''
    if (!label || isPc28ModeType(label, config.playTypeId ?? '')) return true
  }
  return isPc28ModeType(config.playTypeLabel ?? '', config.playTypeId ?? '')
}

export function isPc28HezhiConfigLike(config: {
  betMode?: string
  playMethodLabel?: string
  catalogSubId?: string
  subPlayId?: string
  playTemplate?: string
  playTypeLabel?: string
}): boolean {
  const bm = config.betMode ?? ''
  if (bm === 'hezhi') return true
  const subLabel = config.playMethodLabel?.trim() ?? ''
  const subId = config.catalogSubId ?? config.subPlayId ?? ''
  if (subLabel === '和值' || subId === 'hezhi') return true
  if (config.playTemplate === 'pc28_std' && isPc28ModeType(config.playTypeLabel ?? '', '')) {
    return subLabel === '和值' || subId === '233' || subId === '237'
  }
  return false
}

type AdvTriggerPosConfig = {
  betMode?: string
  playTypeId?: string
  playTypeLabel?: string
  playMethodLabel?: string
  guajiGroup?: string
  subPlayId?: string
  catalogSubId?: string
  playTemplate?: string
  inputMode?: string
  segmentLen?: number
  segmentLabels?: string[]
}

function isAdvTriggerTextLikePlay(config: AdvTriggerPosConfig): boolean {
  if (isLonghuPlayConfigLike(config) || isPc28ModeConfigLike(config)) return true
  const bm = String(config.betMode ?? '')
  // 五星趣味为数字池，勿当文字特殊号
  const label = String(config.playMethodLabel ?? '')
  if (
    /一帆风顺|好事成双|三星报喜|四季发财/i.test(label) ||
    /yifan|haoshi|sanxing|siji/i.test(`${config.subPlayId ?? ''} ${label}`)
  ) {
    return false
  }
  return (
    bm === 'dxds' ||
    bm === 'daxiao' ||
    bm === 'danshuang' ||
    bm === 'teshu' ||
    bm === 'longhubao' ||
    bm === 'zhuangxian' ||
    bm === 'longhu' ||
    bm === 'longhuhe'
  )
}

function isAdvTriggerDingweiPlay(config: AdvTriggerPosConfig): boolean {
  const bm = String(config.betMode ?? '')
  const tid = String(config.playTypeId ?? '')
  const group = String(config.guajiGroup ?? '')
  const label = String(config.playMethodLabel ?? '')
  return (
    bm === 'dingwei' ||
    tid === 'dingwei' ||
    tid === 'g006' ||
    group === '一星' ||
    isDingweiStarType(config.playTypeLabel ?? '', tid, label)
  )
}

/**
 * 任选非直选复式：开某投某需「开奖选位」+「投注选位区」；
 * 一星等仍不单独勾选投注位。
 */
export function supportsAdvTriggerPositionPicker(config: AdvTriggerPosConfig): boolean {
  return isRenxuanNeedsPositionTriggerPlay(config)
}

function isAdvTriggerRenxuanPlay(config: AdvTriggerPosConfig): boolean {
  const tid = String(config.playTypeId ?? '').toLowerCase()
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  return (
    tid === 'g011' ||
    tid === 'renxuan' ||
    tid.includes('renxuan') ||
    label.includes('任选') ||
    String(config.guajiGroup ?? '') === '任选'
  )
}

function isAdvTriggerRenxuanZhixuanFushi(config: AdvTriggerPosConfig): boolean {
  if (!isAdvTriggerRenxuanPlay(config)) return false
  const text = `${config.betMode ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
  if (/单式|组选|和值|组三|组六|zu\d|hunhe|混合/i.test(text)) return false
  return (
    config.inputMode === 'multiline' ||
    /直选复式|zhixuan_fs/i.test(text) ||
    (String(config.betMode ?? '') === 'fushi' && !/组选/.test(text))
  )
}

/** 任选非直选复式（须选位）：对齐任二直选单式 */
export function isRenxuanNeedsPositionTriggerPlay(config: AdvTriggerPosConfig): boolean {
  if (!isAdvTriggerRenxuanPlay(config)) return false
  if (isAdvTriggerRenxuanZhixuanFushi(config)) return false
  const k = Math.max(0, Number(config.segmentLen) || 0)
  // 和值/号池 segmentLen 可能为 1，仍按任 k 选位（由文案/子玩法推断至少 2）
  if (k >= 2) return true
  const text = `${config.catalogSubId ?? ''} ${config.subPlayId ?? ''} ${config.playMethodLabel ?? ''}`
  return /任[二三四]|ren[234]|组选|和值|单式|组三|组六|zu\d/i.test(text)
}

/**
 * 任选·直选单式开某投某：启用区按「投注选位区」所选位分列正/反投。
 * 开奖选位单点查映射行，再取该行各位号码组合出票。
 * 组选/和值/号池仍整行正反投，不分列。
 */
export function isRenxuanPerPosTriggerPlay(config: AdvTriggerPosConfig): boolean {
  if (!isRenxuanNeedsPositionTriggerPlay(config)) return false
  const bm = String(config.betMode ?? '').toLowerCase()
  const sub = String(config.catalogSubId ?? config.subPlayId ?? '').toLowerCase()
  const label = `${config.playMethodLabel ?? ''}`
  if (
    bm === 'hezhi' ||
    bm === 'weishu' ||
    bm === 'zuxuan_fs' ||
    bm === 'zuxuan_ds' ||
    bm === 'zu3' ||
    bm === 'zu6' ||
    bm === 'zu24' ||
    bm === 'zu12' ||
    bm === 'zu4' ||
    bm === 'hunhe' ||
    /组和|直选和值|组选|混合|组三|组六/.test(label)
  ) {
    return false
  }
  return (
    bm === 'danshi' ||
    bm === 'zhixuan_ds' ||
    sub.includes('zhixuan_ds') ||
    /直选单式/.test(label) ||
    config.inputMode === 'danshi'
  )
}

/** @deprecated 使用 isRenxuanPerPosTriggerPlay；保留兼容旧测试名 */
export function isRenxuanZhixuanDanshiTriggerPlay(config: AdvTriggerPosConfig): boolean {
  return isRenxuanPerPosTriggerPlay(config)
}

/** 任选开某投某默认选位：任二万千；任三万千个；任四万千百十 */
export function defaultRenxuanTriggerPositionIdxs(k: number): number[] {
  const n = k >= 2 && k <= 5 ? k : 2
  if (n <= 2) return [0, 1]
  if (n === 3) return [0, 1, 4]
  if (n === 4) return [0, 1, 2, 3]
  return [0, 1, 2, 3, 4]
}

/**
 * 任选冷热开奖选位默认前 k 位（任二万千；任三万千百；任四万千百十）。
 * 用于直选单式 / 混合组选；与投注选位默认（任三含个位）可不同。
 */
export function defaultRenxuanHcwOpenPositionIdxs(k: number): number[] {
  const n = k >= 2 && k <= 5 ? k : 2
  return Array.from({ length: n }, (_, i) => i)
}

/**
 * 任选冷热「开奖选位」：须恰好 k 个，下方频次按这些绝对位分列。
 * - 任选直选单式（与开某投某按位分列同一集合）
 * - 任选混合组选（开某投某仍整注一行，但冷热与直选单式同开奖选位口径）
 */
export function isRenxuanHcwOpenPosPlay(config: AdvTriggerPosConfig): boolean {
  if (isRenxuanPerPosTriggerPlay(config)) return true
  if (!isRenxuanNeedsPositionTriggerPlay(config)) return false
  const bm = String(config.betMode ?? '').toLowerCase()
  if (bm === 'hunhe') return true
  const label = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /混合组选|混合/.test(label) && !/组合/.test(label)
}

/**
 * 组选12 冷热：二重号/单号双池（任四组选12、四星组选12、前后四组选12 等）。
 * 任选另需投注选位合并计频；定点星段（四星等）按玩法位合并计频。
 */
export function isHcwZu12DualPlay(config: AdvTriggerPosConfig): boolean {
  const bm = String(config.betMode ?? '').toLowerCase()
  if (bm === 'zu12') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选12|zu12/i.test(text) && !/组选120|zu120/i.test(text)
}

/**
 * 组选4 冷热：三重号/单号双池（四星组选4、前后四组选4 等）。
 */
export function isHcwZu4DualPlay(config: AdvTriggerPosConfig): boolean {
  const bm = String(config.betMode ?? '').toLowerCase()
  if (bm === 'zu4') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选4|zu4/i.test(text) && !/组选24|zu24|组选12|zu12/i.test(text)
}

/** 五星组选60/30/20/10/5 冷热双区 */
export function isHcwWuxingZuDualPlay(config: AdvTriggerPosConfig): boolean {
  const bm = String(config.betMode ?? '').toLowerCase()
  if (bm === 'zu60' || bm === 'zu30' || bm === 'zu20' || bm === 'zu10' || bm === 'zu5') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  if (/组选120|zu120/i.test(text)) return false
  return (
    /组选60|zu60/i.test(text) ||
    /组选30|zu30/i.test(text) ||
    /组选20|zu20/i.test(text) ||
    /组选10|zu10/i.test(text) ||
    (/组选5|zu5/i.test(text) && !/组选50|组选5\d/i.test(text))
  )
}

/** 组选12/4 或五星组选60·30·20·10·5 冷热双区 */
export function isHcwZuDualPlay(config: AdvTriggerPosConfig): boolean {
  return isHcwZu12DualPlay(config) || isHcwZu4DualPlay(config) || isHcwWuxingZuDualPlay(config)
}

/**
 * 任选·组选12 冷热：投注选位合并计频 + 二重号/单号双池（无独立开奖选位）。
 */
export function isRenxuanHcwZu12Play(config: AdvTriggerPosConfig): boolean {
  return isRenxuanNeedsPositionTriggerPlay(config) && isHcwZu12DualPlay(config)
}

/** 任选·组选4 冷热 */
export function isRenxuanHcwZu4Play(config: AdvTriggerPosConfig): boolean {
  return isRenxuanNeedsPositionTriggerPlay(config) && isHcwZu4DualPlay(config)
}

/** 任选·组选12/4 冷热（需投注选位） */
export function isRenxuanHcwZuDualPlay(config: AdvTriggerPosConfig): boolean {
  return isRenxuanHcwZu12Play(config) || isRenxuanHcwZu4Play(config)
}

/**
 * 单 token 大小/单双（第三方仅 1 选项）：
 * - 五星和值单双 / 和值大小
 * - 哈希玩法尾数单双 / 尾数大小（波场/哈希分分彩等）
 */
export function isWuxingSumDxdsPlayConfig(config: AdvTriggerPosConfig): boolean {
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''}`
  if (/和值单双|和值大小|尾数单双|尾数大小/.test(label)) return true
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  const typeId = String(config.playTypeId ?? '').trim()
  // 哈希 g017：267/387=尾数单双，270/390=尾数大小，389=和值单双（勿把 388 幸运庄闲算进来）
  if (typeId === 'g017' || label.includes('哈希')) {
    return ['267', '270', '387', '389', '390'].includes(sid)
  }
  // SSC 五星和值：263/268=单双，264/269=大小
  return ['263', '264', '268', '269'].includes(sid)
}

/**
 * 前二/后二/前三/后三大小单双：按位（十/个…）文字选项，非整期单档属性池。
 * 排除：PC28 整期大小单双、五星和值大小/单双。
 */
export function isPerPosDxdsPlayConfig(config: AdvTriggerPosConfig): boolean {
  const segLen = Math.max(0, Number(config.segmentLen) || 0)
  if (segLen < 2) return false
  if (isLonghuPlayConfigLike(config) || isPc28ModeConfigLike(config)) return false
  if (isWuxingSumDxdsPlayConfig(config)) return false
  const bm = String(config.betMode ?? '').toLowerCase()
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (label.includes('和值大小') || label.includes('和值单双') || label.includes('五星和值')) {
    return false
  }
  if (bm === 'dxds') return true
  if (label.includes('大小单双') && !label.includes('和值')) return true
  return false
}

/** 时时彩按位大小单双选项宇宙 */
export const PER_POS_DXDS_OPTIONS = ['大', '小', '单', '双'] as const

/** 球号 → 同时命中的大小/单双（每位一球对应两项） */
export function sscDigitDxdsAttrs(digit: number): string[] {
  if (!Number.isFinite(digit)) return []
  const n = Math.trunc(digit)
  return [n >= 5 ? '大' : '小', n % 2 === 1 ? '单' : '双']
}

/** 按位列位数：优先 segmentLen，其次 segmentLabels 长度（一星五位兜底） */
function advTriggerPosCount(config: AdvTriggerPosConfig): number {
  const segLen = Math.max(0, Number(config.segmentLen) || 0)
  const labelN = Array.isArray(config.segmentLabels) ? config.segmentLabels.length : 0
  return Math.max(segLen, labelN)
}

/**
 * 前三/中三/后三等直选单式：segmentLen>=2，开某投某 / 随机出号应按位分列；
 * 排除任选单式与组选单式。位标签缺失时仍按段长分列（展示用「第 N 位」兜底）。
 */
export function isZhixuanDanshiPerPosPlay(config: AdvTriggerPosConfig): boolean {
  const segLen = Math.max(0, Number(config.segmentLen) || 0)
  if (segLen < 2) return false
  const bm = String(config.betMode ?? '').toLowerCase()
  const sub = String(config.catalogSubId ?? config.subPlayId ?? '').toLowerCase()
  const tid = String(config.playTypeId ?? '').toLowerCase()
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (
    tid.includes('renxuan') ||
    tid === 'g011' ||
    label.includes('任选') ||
    label.includes('组选') ||
    bm === 'zuxuan_ds' ||
    sub === 'zuxuan_ds' ||
    sub.includes('zuxuan_ds') ||
    bm === 'hunhe' ||
    label.includes('混合')
  ) {
    return false
  }
  return (
    bm === 'danshi' ||
    bm === 'zhixuan_ds' ||
    sub === 'zhixuan_ds' ||
    sub.includes('zhixuan_ds') ||
    label.includes('直选单式') ||
    config.inputMode === 'danshi'
  )
}

/**
 * 高级开某投某：一星定位胆 / 前三直选复式 / 中三直选单式 / 后二大小单双等按位玩法，
 * 表格按「万位正投/反投、千位…」分列填写（不展示投注位芯片）。
 */
export function supportsAdvTriggerPerPosColumns(config: AdvTriggerPosConfig): boolean {
  // 任选直选单式：按投注选位区分列（配合开奖选位 + 投注选位芯片）
  if (isRenxuanPerPosTriggerPlay(config)) return true
  // 一星/定位胆五位（或 PK10 十名次）：与前三码同布局，每位旁填预备投注号
  if (isAdvTriggerDingweiPlay(config) && advTriggerPosCount(config) >= 2) return true
  const segLen = Math.max(0, Number(config.segmentLen) || 0)
  if (segLen < 2) return false
  const bm = String(config.betMode ?? '')
  const sub = String(config.catalogSubId ?? config.subPlayId ?? '')
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  // 前二/后二/前三/后三大小单双：按位（开出=球号，正反投=大/小/单/双）
  if (isPerPosDxdsPlayConfig(config)) return true
  if (isAdvTriggerTextLikePlay(config)) return false
  if (config.inputMode === 'multiline') return true
  if (bm === 'fushi' || bm === 'zhixuan_fs' || bm === 'zuhe') return true
  if (sub === 'zhixuan_fs' || sub.includes('zhixuan_fs') || label.includes('直选复式')) return true
  // 中三/前三混合组选：与直选复式同布局（千/百/十按位填正反投）
  // 任选混合组选：整注一行正投/反投（如 012,345），勿按位拆分
  if (bm === 'hunhe' || label.includes('混合组选') || (label.includes('混合') && !label.includes('组合'))) {
    if (isRenxuanNeedsPositionTriggerPlay(config)) return false
    return true
  }
  // 中三/前三直选单式：千百十（或万千百）三位分列
  if (isZhixuanDanshiPerPosPlay(config)) return true
  return false
}

export function pc28HezhiNumberPool(): { min: number; max: number } {
  return { min: 0, max: 27 }
}
