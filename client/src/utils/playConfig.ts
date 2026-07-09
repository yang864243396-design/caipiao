import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'
import type { PlayConfig } from '@/utils/betPayload'
import {
  guajiGroupFromSegment,
  inferBetModeFromCatalog,
  isDingweiStarType,
  isLonghuPlayType,
  isPc28ModeType,
} from '@/utils/runTypeMatrix'

function segmentRulePool(subNode?: SubPlayNode): { min: number; max: number } | undefined {
  const rule = subNode?.segmentRule as { numberPoolMin?: number; numberPoolMax?: number } | undefined
  if (rule?.numberPoolMax != null) {
    return { min: rule.numberPoolMin ?? 0, max: rule.numberPoolMax }
  }
  return undefined
}

function defaultPoolByTemplate(template: string): { min: number; max: number } | undefined {
  switch (template) {
    case 'syxw_std':
      return { min: 1, max: 11 }
    case 'pk10_std':
      return { min: 1, max: 10 }
    case 'k3_std':
      return { min: 1, max: 6 }
    case 'pc28_std':
      return { min: 0, max: 9 }
    default:
      return undefined
  }
}

const POSITION_LABELS = ['万', '千', '百', '十', '个'] as const

/** 子玩法展示：去掉「一星定位胆 · 」等大类前缀，仅保留万位/千位等短名 */
export function formatSubPlayLabel(label: string): string {
  const t = label.trim()
  const sep = t.indexOf('·')
  if (sep >= 0) {
    const tail = t.slice(sep + 1).trim()
    if (tail) return tail
  }
  return t
}

function dingweiPositionIndex(subId: string): number {
  if (subId.endsWith('_wan')) return 0
  if (subId.endsWith('_qian')) return 1
  if (subId.endsWith('_bai')) return 2
  if (subId.endsWith('_shi')) return 3
  if (subId.endsWith('_ge')) return 4
  return 0
}

/** 子玩法 label 是否对应特定位（万/千/百/十/个） */
function dingweiPositionLabel(subLabel: string): string | null {
  const label = formatSubPlayLabel(subLabel)
  const pairs: Array<[string, string]> = [
    ['万位', '万'],
    ['千位', '千'],
    ['百位', '百'],
    ['十位', '十'],
    ['个位', '个'],
  ]
  for (const [long, short] of pairs) {
    if (label.includes(long) || label === short) return short
  }
  return null
}

function isSSCPlayTemplate(template: string): boolean {
  return template === 'ssc_std' || template === 'fast_ssc_std'
}

/** 定位胆五位面板：无特定位子玩法（如 rules/v2 同步后的「定位胆」） */
function isDingweiFivePositionScheme(
  playTemplate: string,
  typeLabel: string,
  typeId: string,
  subLabel: string,
  subId: string,
  guajiGroup: string,
): boolean {
  if (!isSSCPlayTemplate(playTemplate)) return false
  const isDingwei =
    typeId === 'dingwei' || isDingweiStarType(typeLabel, typeId, subLabel) || guajiGroup === '一星'
  if (!isDingwei) return false
  if (subId.startsWith('dingwei_')) return false
  if (dingweiPositionLabel(subLabel)) return false
  const raw = subLabel.trim()
  return raw.includes('定位胆') || raw.includes('定胆') || raw === '一星' || guajiGroup === '一星'
}

function renPickCount(subId: string): number {
  const s = subId.toLowerCase()
  if (s.startsWith('ren4')) return 4
  if (s.startsWith('ren3')) return 3
  if (s.startsWith('ren2')) return 2
  return 2
}

function budingweiOrDxdsSegment(typeId: string, subId: string): { start: number; len: number } {
  const s = subId.toLowerCase()
  if (typeId === 'budingwei') {
    if (s.startsWith('qian3')) return { start: 0, len: 3 }
    if (s.startsWith('zhong3')) return { start: 1, len: 3 }
    if (s.startsWith('hou3')) return { start: 2, len: 3 }
    if (s.startsWith('qian4')) return { start: 0, len: 4 }
    if (s.startsWith('hou4')) return { start: 1, len: 4 }
    if (s.startsWith('wuxing')) return { start: 0, len: 5 }
    return { start: 0, len: 3 }
  }
  if (s.startsWith('qian2')) return { start: 0, len: 2 }
  if (s.startsWith('hou2')) return { start: 3, len: 2 }
  if (s.startsWith('qian3')) return { start: 0, len: 3 }
  if (s.startsWith('hou3')) return { start: 2, len: 3 }
  if (s.startsWith('wuxing')) return { start: 0, len: 5 }
  return { start: 0, len: 2 }
}

function syxwSegmentRange(typeId: string): { start: number; len: number } {
  switch (typeId) {
    case 'qian3':
      return { start: 0, len: 3 }
    case 'qian2':
      return { start: 0, len: 2 }
    default:
      return { start: 0, len: 1 }
  }
}

function pk10SegmentRange(typeId: string): { start: number; len: number } {
  switch (typeId) {
    case 'qian1':
      return { start: 0, len: 1 }
    case 'qian2':
      return { start: 0, len: 2 }
    case 'qian3':
      return { start: 0, len: 3 }
    case 'qian4':
      return { start: 0, len: 4 }
    case 'qian5':
      return { start: 0, len: 5 }
    default:
      return { start: 0, len: 1 }
  }
}

function catalogSegmentRange(template: string, typeId: string): { start: number; len: number } {
  if (template === 'syxw_std') return syxwSegmentRange(typeId)
  if (template === 'pk10_std') return pk10SegmentRange(typeId)
  return sscSegmentRange(typeId)
}

function sscSegmentRange(typeId: string): { start: number; len: number } {
  switch (typeId) {
    case 'qian3':
    case 'qianzhonghou3':
    case 'qianhou3':
      return { start: 0, len: 3 }
    case 'zhong3':
      return { start: 1, len: 3 }
    case 'hou3':
      return { start: 2, len: 3 }
    case 'qian2':
      return { start: 0, len: 2 }
    case 'hou2':
      return { start: 3, len: 2 }
    case 'sixing':
      return { start: 1, len: 4 }
    case 'wuxing':
      return { start: 0, len: 5 }
    case 'combo24':
      return { start: 0, len: 2 }
    case 'dingwei':
      return { start: 0, len: 1 }
    default:
      return { start: 0, len: 1 }
  }
}

const CATALOG_PLAY_TYPE_IDS = new Set([
  'dingwei',
  'qian3',
  'zhong3',
  'hou3',
  'qian2',
  'hou2',
  'sixing',
  'wuxing',
  'qianzhonghou3',
  'qianhou3',
  'combo24',
  'longhu',
  'hezhi',
  'kuadu',
  'renxuan',
  'budingwei',
  'dxds',
])

export function isCatalogPlayTypeId(typeId: string): boolean {
  return CATALOG_PLAY_TYPE_IDS.has(typeId.trim())
}

function combo24SegmentPositions(subId: string): number[] {
  const s = subId.toLowerCase()
  if (s.startsWith('qh4')) return [0, 1, 3, 4]
  if (s.startsWith('qh2')) return [0, 4]
  return [0, 4]
}

function legacySubMode(subId: string, betMode: string): string {
  const s = subId.toLowerCase()
  if (s.includes('zhixuan_ds') || betMode === 'danshi') return 'zhixuan_ds'
  if (s.includes('zhixuan_fs') || betMode === 'fushi') return 'zhixuan_fs'
  if (['zu24', 'zu12', 'zu60', 'zu30', 'zu120'].includes(betMode)) return betMode
  if (
    s.includes('zu3') ||
    s.includes('zu6') ||
    s.includes('zuxuan') ||
    betMode === 'zu3' ||
    betMode === 'zu6'
  ) {
    return 'zuxuan_fs'
  }
  if (s.includes('_zu3') || s.includes('_zu6')) return 'zuxuan_fs'
  if (betMode === 'dingwei' || s.startsWith('dingwei_')) return 'dingwei'
  if (
    [
      'zuhe',
      'baodan',
      'hunhe',
      'weishu',
      'teshu',
      'zu24',
      'zu12',
      'zu60',
      'zu30',
      'zu120',
      'hezhi',
      'kuadu',
      'longhu',
      'budingwei',
      'dxds',
      'daxiao',
      'danshuang',
    ].includes(betMode)
  ) {
    return betMode
  }
  return ''
}

function inputModeFromPanelType(
  panelType: string,
  betMode: string,
  subPlayId: string,
  segmentLen: number,
): PlayConfig['inputMode'] | null {
  switch (panelType) {
    case 'dingwei':
      return segmentLen > 1 ? 'multiline' : 'dingwei'
    case 'longhu':
      return 'pool'
    case 'renxuan':
      return 'multiline'
    case 'textarea':
      return 'danshi'
    case 'segment':
      return inputModeFromBetMode(betMode, subPlayId, segmentLen)
    case 'lhc_num':
      return 'lhc_num'
    case 'lhc_zodiac':
      return 'lhc_zodiac'
    case 'lhc_tail':
      return 'lhc_tail'
    case 'lhc_attr':
      return 'lhc_attr'
    case 'k3_pool':
      return 'pool'
    default:
      return null
  }
}

function inputModeFromBetMode(
  betMode: string,
  subPlayId: string,
  segmentLen: number,
): PlayConfig['inputMode'] {
  if (betMode === 'danshi' || subPlayId === 'zhixuan_ds') return 'danshi'
  if (betMode === 'fushi' && segmentLen > 1) return 'multiline'
  if (betMode === 'fushi' || subPlayId === 'zuxuan_fs' || subPlayId === 'zhixuan_fs') {
    return segmentLen > 1 ? 'multiline' : 'pool'
  }
  if (betMode === 'dingwei' || subPlayId === 'dingwei') {
    return segmentLen > 1 ? 'multiline' : 'dingwei'
  }
  if (betMode === 'longhu' || betMode === 'longhuhe') return 'pool'
  // 和值/跨度/龙虎/组合/包胆/尾数/特殊号等：textarea 手输
  if (
    [
      'hezhi',
      'kuadu',
      'danshuang',
      'daxiao',
      'budingwei',
      'teshu',
      'hunhe',
      'zuhe',
      'baodan',
      'weishu',
      'zu24',
      'zu12',
      'zu60',
      'zu30',
      'zu120',
    ].includes(betMode)
  ) {
    return 'danshi'
  }
  if (betMode === 'zu3' || betMode === 'zu6') return 'pool'
  return segmentLen > 1 ? 'multiline' : 'dingwei'
}

/** 无 play-tree 时按 catalog typeId/subId 解析玩法配置 */
export function resolvePlayConfigFromCatalogIds(
  typeId: string,
  subId: string,
  betMode = '',
): PlayConfig {
  if (typeId === 'longhu') {
    const inferredBet = betMode || (subId.includes('_he') ? 'longhuhe' : 'longhu')
    return {
      playTypeId: typeId,
      subPlayId: subId,
      catalogSubId: subId,
      segmentLen: 1,
      segmentLabels: ['龙虎'],
      inputMode: 'pool',
      betMode: inferredBet,
    }
  }

  const subPlayId = legacySubMode(subId, betMode)
  let segmentStart = 0
  let segmentLen = 1
  let segmentLabels: string[]

  if (typeId === 'dingwei') {
    if (subId.startsWith('dingwei_')) {
      segmentStart = dingweiPositionIndex(subId)
      segmentLen = 1
      segmentLabels = [POSITION_LABELS[segmentStart]]
    } else {
      segmentLen = 5
      segmentLabels = [...POSITION_LABELS]
    }
  } else if (typeId === 'qianhou3') {
    segmentLen = 3
    segmentLabels = ['万', '百', '个']
  } else if (typeId === 'renxuan') {
    segmentLen = renPickCount(subId)
    segmentLabels = [...POSITION_LABELS]
  } else if (typeId === 'combo24') {
    const pos = combo24SegmentPositions(subId)
    segmentLen = pos.length
    segmentLabels = pos.map((i) => POSITION_LABELS[i])
  } else if (typeId === 'budingwei' || typeId === 'dxds') {
    const seg = budingweiOrDxdsSegment(typeId, subId)
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = POSITION_LABELS.slice(segmentStart, segmentStart + segmentLen)
  } else {
    const seg = sscSegmentRange(typeId)
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = POSITION_LABELS.slice(segmentStart, segmentStart + segmentLen)
  }

  const inputMode = inputModeFromBetMode(betMode, subPlayId, segmentLen)
  return {
    playTypeId: typeId,
    subPlayId,
    segmentLen,
    segmentLabels,
    inputMode,
    betMode: betMode || undefined,
  }
}

export interface PlayTreePlayConfig extends PlayConfig {
  playTemplate: string
  typeId: string
  subId: string
  betMode: string
  playTypeLabel: string
  playMethodLabel: string
}

export function resolvePlayConfigFromTree(
  playTemplate: string,
  typeNode: PlayTypeNode,
  subNode: SubPlayNode,
): PlayTreePlayConfig {
  const typeId = typeNode.typeId
  const subId = subNode.subId
  const typeLabel = typeNode.label.trim()
  const subLabel = subNode.label.trim()
  const betModeRaw = subNode.betMode ?? ''
  const betMode = betModeRaw || inferBetModeFromCatalog(typeNode, subNode, playTemplate)
  const subPlayId = legacySubMode(subId, betMode)
  const pool = segmentRulePool(subNode) ?? defaultPoolByTemplate(playTemplate)
  const guajiGroup = guajiGroupFromSegment(subNode.segmentRule)

  let segmentStart = 0
  let segmentLen = 1
  let segmentLabels: string[]

  if (typeId === 'dingwei' || isDingweiStarType(typeLabel, typeId, subLabel) || guajiGroup === '一星') {
    if (isDingweiFivePositionScheme(playTemplate, typeLabel, typeId, subLabel, subId, guajiGroup)) {
      segmentLen = 5
      segmentLabels = [...POSITION_LABELS]
    } else {
      segmentStart = dingweiPositionIndex(subId)
      segmentLen = 1
      segmentLabels = [
        dingweiPositionLabel(subLabel) ?? POSITION_LABELS[segmentStart] ?? formatSubPlayLabel(subLabel),
      ]
    }
  } else if (typeId === 'qianhou3') {
    segmentLen = 3
    segmentLabels = ['万', '百', '个']
  } else if (typeId === 'renxuan' || typeId === 'renxuan_fs' || typeId === 'renxuan_ds') {
    segmentLen = 5
    segmentLabels = [...POSITION_LABELS]
  } else if (typeId === 'combo24') {
    const pos = combo24SegmentPositions(subId)
    segmentLen = pos.length
    segmentLabels = pos.map((i) => POSITION_LABELS[i])
  } else if (typeId === 'budingwei' || typeId === 'dxds') {
    const seg = budingweiOrDxdsSegment(typeId, subId)
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = POSITION_LABELS.slice(segmentStart, segmentStart + segmentLen)
  } else if (typeId === 'pc28_20' || typeId === 'pc28_28' || isPc28ModeType(typeLabel, typeId)) {
    segmentLen = 1
    segmentLabels = [formatSubPlayLabel(subLabel) || '和值']
  } else if (typeId === 'longhu' || isLonghuPlayType(typeLabel, typeId) || guajiGroup === '龙虎') {
    segmentLen = 1
    segmentLabels = ['龙虎']
  } else {
    const seg = catalogSegmentRange(playTemplate, typeId)
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = POSITION_LABELS.slice(segmentStart, segmentStart + segmentLen)
  }
  const panelType = typeNode.panelType ?? ''
  let inputMode =
    inputModeFromPanelType(panelType, betMode, subPlayId, segmentLen) ??
    inputModeFromBetMode(betMode, subPlayId, segmentLen)
  if (playTemplate === 'syxw_std' && (typeId === 'renxuan_fs' || typeId === 'renxuan_ds')) {
    if (betMode === 'danshi' || subId.endsWith('_ds')) {
      inputMode = 'danshi'
    } else {
      inputMode = 'pool'
      segmentLen = 1
      segmentLabels = ['选号']
    }
  }
  if (betMode === 'tuotou' || betMode.endsWith('_dp')) {
    inputMode = 'danshi'
  }
  if (betMode === 'zongxiao') {
    inputMode = 'lhc_attr'
  }
  if (betMode === 'tematouwei') {
    inputMode = 'lhc_attr'
  }
  if (betMode === 'qima') {
    inputMode = 'lhc_attr'
  }
  if (typeId === 'longhu' || isLonghuPlayType(typeLabel, typeId) || guajiGroup === '龙虎' || betMode === 'longhu' || betMode === 'longhuhe') {
    segmentLen = 1
    segmentLabels = ['龙虎']
    inputMode = 'pool'
  }

  let numberPoolMin = pool?.min
  let numberPoolMax = pool?.max
  // PC28 和值：号池 0–27，用 chip 面板选号（rules/v2 同步后 bet_mode 常为空，靠 label 推断）
  if (playTemplate === 'pc28_std' && (betMode === 'hezhi' || subLabel === '和值' || subId === 'hezhi')) {
    inputMode = 'pool'
    numberPoolMin = 0
    numberPoolMax = 27
  }

  return {
    playTemplate,
    typeId,
    subId,
    betMode,
    playTypeLabel: typeLabel,
    playMethodLabel: formatSubPlayLabel(subLabel),
    playTypeId: typeId,
    subPlayId,
    catalogSubId: subId,
    segmentLen,
    segmentLabels,
    inputMode,
    numberPoolMin,
    numberPoolMax,
  }
}

export function findSubPlay(
  tree: { playTypes: PlayTypeNode[] },
  typeId: string,
  subId: string,
): { typeNode: PlayTypeNode; subNode: SubPlayNode } | null {
  const typeNode = tree.playTypes.find((t) => t.typeId === typeId)
  if (!typeNode) return null
  const subNode = typeNode.subPlays.find((s) => s.subId === subId)
  if (!subNode) return null
  return { typeNode, subNode }
}

export function defaultPlaySelection(tree: {
  playTemplate: string
  playTypes: PlayTypeNode[]
}): { typeId: string; subId: string } {
  if (tree.playTemplate === 'lhc_std') {
    const tema =
      tree.playTypes.find((t) => t.label.trim() === '特码') ??
      tree.playTypes.find((t) => t.typeId === 'tema')
    const sub =
      tema?.subPlays.find((s) => s.subId === 'tema_a') ??
      tema?.subPlays.find((s) => s.label.includes('特码')) ??
      tema?.subPlays[0]
    if (tema && sub) return { typeId: tema.typeId, subId: sub.subId }
  }
  if (tree.playTemplate === 'pc28_std') {
    const line =
      tree.playTypes.find((t) => t.label.trim() === '2.0模式') ??
      tree.playTypes.find((t) => t.typeId === 'pc28_20') ??
      tree.playTypes[0]
    const sub =
      line?.subPlays.find((s) => s.label.trim() === '和值' || s.subId === 'hezhi') ??
      line?.subPlays[0]
    if (line && sub) return { typeId: line.typeId, subId: sub.subId }
  }
  if (tree.playTemplate === 'k3_std') {
    const hezhi =
      tree.playTypes.find((t) => t.label.trim() === '和值') ??
      tree.playTypes.find((t) => t.typeId === 'hezhi')
    const sub = hezhi?.subPlays[0]
    if (hezhi && sub) return { typeId: hezhi.typeId, subId: sub.subId }
  }
  return defaultSSCSelection(tree)
}

export function defaultSSCSelection(tree: { playTypes: PlayTypeNode[] }): {
  typeId: string
  subId: string
} {
  const dingwei =
    tree.playTypes.find((t) => t.label.trim() === '一星') ??
    tree.playTypes.find((t) => t.typeId === 'dingwei')
  const sub =
    dingwei?.subPlays.find((s) => s.label.includes('定位胆')) ?? dingwei?.subPlays[0]
  if (dingwei && sub) return { typeId: dingwei.typeId, subId: sub.subId }
  const first = tree.playTypes[0]
  const firstSub = first?.subPlays[0]
  if (first && firstSub) return { typeId: first.typeId, subId: firstSub.subId }
  return { typeId: '', subId: '' }
}
