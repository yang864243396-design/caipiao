import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'

/** rules/v2 同步后 PC28 高级开某投某支持的子玩法 label */
export const ADV_TRIGGER_PC28_SUB_LABELS = new Set(['和值', '大小单双', '龙虎豹'])

/** 旧 sub_id 兼容 */
export const ADV_TRIGGER_PC28_SUBS = new Set(['hezhi', 'dxds', 'longhubao'])

/** rules/v2 同步后支持的玩法类型 label（groups[].name） */
export const ADV_TRIGGER_PLAY_TYPE_LABELS = new Set(['一星', '龙虎', '2.0模式', '2.8模式'])

/** 旧 type_id 兼容 */
export const ADV_TRIGGER_PLAY_TYPES = new Set(['dingwei', 'longhu', 'pc28_20', 'pc28_28'])

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
  // 属性/聚合家族：大小单双/龙虎/特殊号/庄闲/和值/跨度/不定位/包胆
  if (/大小单双|大小|单双|龙虎|庄闲|特殊号|豹子|对子|顺子|和值|跨度|不定位|包胆/.test(sub)) return true
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
  return /组三|组六|组选|不定位|包胆/.test(sub)
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

export function pc28HezhiNumberPool(): { min: number; max: number } {
  return { min: 0, max: 27 }
}
