import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'
import type { PlayConfig } from '@/utils/betPayload'
import {
  hezhiPoolRange,
  hezhiDigitLenFromText,
  kuaduPoolRange,
  lhcInputModeFromBetMode,
  pk10SegmentLen,
  PK10_POSITION_LABELS,
  syxwSegmentLen,
  SYXW_POSITION_LABELS,
  weishuPoolRange,
} from '@/utils/playInputProfile'
import {
  guajiFullNameFromSegment,
  guajiGroupFromSegment,
  guajiTeamFromSegment,
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
  const tid = typeId.trim()
  const isDingwei =
    tid === 'dingwei' ||
    tid === 'g006' ||
    isDingweiStarType(typeLabel, typeId, subLabel) ||
    guajiGroup === '一星'
  if (!isDingwei) return false
  // 已指定万/千/百/十/个 → 单位面板；其余一星/定位胆（含 rules 数字 subId）→ 五位
  if (subId.startsWith('dingwei_')) return false
  if (dingweiPositionLabel(subLabel)) return false
  if (tid === 'g006' || tid === 'dingwei' || guajiGroup === '一星' || typeLabel.trim() === '一星') {
    return true
  }
  const raw = subLabel.trim()
  return raw.includes('定位胆') || raw.includes('定胆') || raw === '一星'
}

function renPickCount(subId: string): number {
  const s = subId.toLowerCase()
  if (s.startsWith('ren4')) return 4
  if (s.startsWith('ren3')) return 3
  if (s.startsWith('ren2')) return 2
  return 2
}

function budingweiOrDxdsSegment(
  typeId: string,
  subId: string,
  subLabel = '',
  fullName = '',
): { start: number; len: number } {
  const text = `${subId} ${subLabel} ${fullName}`.toLowerCase()
  const raw = `${subId} ${subLabel} ${fullName}`
  if (typeId === 'budingwei' || typeId === 'g009' || raw.includes('不定位')) {
    if (raw.includes('前三') || text.startsWith('qian3')) return { start: 0, len: 3 }
    if (raw.includes('中三') || text.startsWith('zhong3')) return { start: 1, len: 3 }
    if (raw.includes('后三') || text.startsWith('hou3')) return { start: 2, len: 3 }
    if (raw.includes('前四') || text.startsWith('qian4')) return { start: 0, len: 4 }
    if (raw.includes('后四') || text.startsWith('hou4')) return { start: 1, len: 4 }
    if (raw.includes('五星') || text.startsWith('wuxing')) return { start: 0, len: 5 }
    return { start: 0, len: 3 }
  }
  if (raw.includes('和值单双') || raw.includes('和值大小')) return { start: 0, len: 1 }
  if (raw.includes('前二') || text.startsWith('qian2')) return { start: 0, len: 2 }
  if (raw.includes('后二') || text.startsWith('hou2')) return { start: 3, len: 2 }
  if (raw.includes('前三') || text.startsWith('qian3')) return { start: 0, len: 3 }
  if (raw.includes('后三') || text.startsWith('hou3')) return { start: 2, len: 3 }
  if (raw.includes('五星') || text.startsWith('wuxing')) return { start: 0, len: 5 }
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
    case 'g001':
    case 'g007':
    case 'g012':
      return { start: 0, len: 3 }
    case 'zhong3':
    case 'g002':
      return { start: 1, len: 3 }
    case 'hou3':
    case 'g003':
      return { start: 2, len: 3 }
    case 'qian2':
    case 'g004':
      return { start: 0, len: 2 }
    case 'hou2':
    case 'g005':
      return { start: 3, len: 2 }
    case 'sixing':
    case 'g013':
      return { start: 1, len: 4 }
    case 'g014':
      return { start: 0, len: 4 }
    case 'wuxing':
    case 'g015':
      return { start: 0, len: 5 }
    case 'combo24':
    case 'g008':
      return { start: 0, len: 2 }
    case 'dingwei':
    case 'g006':
      return { start: 0, len: 1 }
    default:
      return { start: 0, len: 1 }
  }
}

/** 对齐后端 guajibet.segmentRange：优先 guajiGroup / label，再回退 typeId */
function resolveSscSegmentMeta(input: {
  group: string
  typeLabel: string
  typeId: string
  subLabel: string
  fullName: string
  team: string
}): { start: number; len: number } {
  const { group, typeLabel, typeId, subLabel, fullName, team } = input
  const text = `${group} ${typeLabel} ${fullName} ${subLabel} ${team}`

  switch (group) {
    case '前中后三':
    case '前后三':
      return { start: 0, len: 3 }
    case '前后二':
      return { start: 0, len: 2 }
    case '前后四':
      return { start: 0, len: 4 }
    case '四星':
      return { start: 1, len: 4 }
    case '五星':
      return { start: 0, len: 5 }
    case '任选':
      return { start: 0, len: renxuanSegmentLenFromText(`${team} ${fullName} ${subLabel}`) }
    case '不定位':
    case '大小单双':
      return budingweiOrDxdsSegment(typeId, '', subLabel, fullName)
  }

  if (text.includes('前中后三')) return { start: 0, len: 3 }
  if (text.includes('前后四')) return { start: 0, len: 4 }
  if (text.includes('前后三')) return { start: 0, len: 3 }
  if (text.includes('前后二')) return { start: 0, len: 2 }
  if (text.includes('五星')) return { start: 0, len: 5 }
  if (text.includes('四星')) return { start: 1, len: 4 }
  if (text.includes('前三')) return { start: 0, len: 3 }
  if (text.includes('中三')) return { start: 1, len: 3 }
  if (text.includes('后三')) return { start: 2, len: 3 }
  if (text.includes('前二')) return { start: 0, len: 2 }
  if (text.includes('后二')) return { start: 3, len: 2 }
  if (text.includes('后四')) return { start: 1, len: 4 }
  if (text.includes('前四')) return { start: 0, len: 4 }
  return sscSegmentRange(typeId)
}

function renxuanSegmentLenFromText(text: string): number {
  if (text.includes('任选四') || text.includes('任四') || text.toLowerCase().includes('ren4')) return 4
  if (text.includes('任选三') || text.includes('任三') || text.toLowerCase().includes('ren3')) return 3
  return 2
}

/** 第三方位标签：前后三=万百个，四星=千百十个，等 */
function sscSegmentLabelsForMeta(
  group: string,
  typeLabel: string,
  typeId: string,
  start: number,
  len: number,
  subLabel = '',
): string[] {
  const g = group || typeLabel
  switch (g) {
    case '前后三':
    case 'qianhou3':
      return ['万', '百', '个']
    case '前后二':
      return ['万', '个']
    case '前后四':
      return ['万', '千', '十', '个']
    case '四星':
    case 'sixing':
    case 'g013':
      return ['千', '百', '十', '个']
    case '前中后三':
      return ['万', '千', '百']
    case '大小单双':
    case 'dxds':
    case 'g016': {
      if (subLabel.includes('和值')) return ['选号']
      const seg = budingweiOrDxdsSegment(typeId, '', subLabel)
      return POSITION_LABELS.slice(seg.start, seg.start + seg.len)
    }
  }
  if (typeId === 'qianhou3') return ['万', '百', '个']
  if (typeId === 'combo24') return POSITION_LABELS.slice(start, start + len)
  return POSITION_LABELS.slice(start, start + len)
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

/** rules/v2 typeId（g00x）→ 语义 catalog id，避免无 play-tree 时单式/段长识别失败 */
const GUAJI_TYPE_ID_TO_CATALOG: Record<string, string> = {
  g001: 'qian3',
  g002: 'zhong3',
  g003: 'hou3',
  g004: 'qian2',
  g005: 'hou2',
  g006: 'dingwei',
  g013: 'sixing',
  g015: 'wuxing',
}

export function mapGuajiTypeIdToCatalog(typeId: string): string {
  const id = typeId.trim()
  return GUAJI_TYPE_ID_TO_CATALOG[id] ?? id
}

export function isCatalogPlayTypeId(typeId: string): boolean {
  const id = typeId.trim()
  return CATALOG_PLAY_TYPE_IDS.has(id) || CATALOG_PLAY_TYPE_IDS.has(mapGuajiTypeIdToCatalog(id))
}

function combo24SegmentPositions(subId: string): number[] {
  const s = subId.toLowerCase()
  if (s.startsWith('qh4')) return [0, 1, 3, 4]
  if (s.startsWith('qh2')) return [0, 4]
  return [0, 4]
}

function legacySubMode(subId: string, betMode: string): string {
  const s = subId.toLowerCase()
  if (s.includes('zhixuan_ds') || betMode === 'danshi' || betMode === 'zuxuan_ds') {
    return betMode === 'zuxuan_ds' ? 'zuxuan_ds' : 'zhixuan_ds'
  }
  if (s.includes('zhixuan_fs') || betMode === 'fushi') return 'zhixuan_fs'
  if (betMode === 'zuxuan_fs') return 'zuxuan_fs'
  if (['zu24', 'zu12', 'zu60', 'zu30', 'zu120', 'zu20', 'zu10', 'zu5', 'zu4'].includes(betMode)) {
    return betMode
  }
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
      'zu20',
      'zu10',
      'zu5',
      'zu4',
      'hezhi',
      'kuadu',
      'longhu',
      'longhuhe',
      'budingwei',
      'dxds',
      'daxiao',
      'danshuang',
      'zuxuan_fs',
      'zuxuan_ds',
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
  if (betMode === 'danshi' || subPlayId === 'zhixuan_ds' || betMode === 'zuxuan_ds' || subPlayId === 'zuxuan_ds') {
    return 'danshi'
  }
  // 组选复式：单行号池（非按位）
  if (betMode === 'zuxuan_fs' || subPlayId === 'zuxuan_fs') return 'pool'
  if (betMode === 'fushi' || subPlayId === 'zhixuan_fs') {
    return segmentLen > 1 ? 'multiline' : 'pool'
  }
  if (betMode === 'dingwei' || subPlayId === 'dingwei') {
    return segmentLen > 1 ? 'multiline' : 'dingwei'
  }
  if (betMode === 'longhu' || betMode === 'longhuhe') return 'pool'
  // 大小单双：按位选 大/小/单/双（后二=2 行）
  if (betMode === 'dxds') return segmentLen > 1 ? 'multiline' : 'pool'
  if (betMode === 'daxiao' || betMode === 'danshuang' || betMode === 'zhuangxian') return 'pool'
  // 和值/跨度/尾数/包胆/特殊号/不定位/组选类：号池 chip（对齐第三方）
  if (
    betMode === 'hezhi' ||
    betMode === 'kuadu' ||
    betMode === 'weishu' ||
    betMode === 'baodan' ||
    betMode === 'teshu' ||
    betMode === 'budingwei' ||
    betMode === 'zu3' ||
    betMode === 'zu6' ||
    ['zu24', 'zu12', 'zu60', 'zu30', 'zu120', 'zu20', 'zu10', 'zu5', 'zu4'].includes(betMode)
  ) {
    return 'pool'
  }
  // 混合组选/组合：手输或号池；组合多为按位，混合为单式文本
  if (betMode === 'hunhe') return 'danshi'
  if (betMode === 'zuhe') return segmentLen > 1 ? 'multiline' : 'pool'
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
      poolMaxPicks: 1,
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
    const k = renPickCount(subId)
    if (
      betMode === 'danshi' ||
      betMode === 'zuxuan_ds' ||
      subId.includes('_ds') ||
      /单式/.test(subId)
    ) {
      return {
        playTypeId: typeId,
        subPlayId,
        catalogSubId: subId,
        segmentLen: k,
        segmentLabels: ['选号'],
        inputMode: 'danshi',
        betMode: betMode || 'danshi',
        renPositionCount: k,
        guajiGroup: '任选',
      }
    }
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
  const guajiFullName = guajiFullNameFromSegment(subNode.segmentRule)
  const guajiTeam = guajiTeamFromSegment(subNode.segmentRule)

  let segmentStart = 0
  let segmentLen = 1
  let segmentLabels: string[]
  let numberPoolMin = pool?.min
  let numberPoolMax = pool?.max
  let poolMaxPicks: number | undefined
  let renPositionCount: number | undefined

  if (playTemplate === 'lhc_std') {
    const lhcMode = lhcInputModeFromBetMode(betMode, typeId, typeLabel) as PlayConfig['inputMode']
    return finishConfig({
      playTemplate, typeId, subId, betMode, typeLabel, subLabel,
      subPlayId: betMode || subPlayId, segmentLen: 1,
      segmentLabels: [formatSubPlayLabel(subLabel) || '选号'],
      inputMode: lhcMode, numberPoolMin: 1, numberPoolMax: 49, guajiGroup,
    })
  }

  if (playTemplate === 'pk10_std') {
    let poolMaxPicksPk10: number | undefined
    if (betMode === 'longhu' || guajiGroup === '龙虎' || typeLabel === '龙虎') {
      segmentLen = 1
      segmentLabels = ['龙虎']
      poolMaxPicksPk10 = 1
    } else if (betMode === 'daxiao' || betMode === 'danshuang' || typeLabel === '大小' || typeLabel === '单双') {
      segmentLen = 1
      segmentLabels = [formatSubPlayLabel(subLabel) || '选号']
    } else if (betMode === 'hezhi' || (betMode === 'dxds' && guajiGroup === '和值')) {
      segmentLen = 1
      segmentLabels = ['选号']
      const hz = hezhiPoolRange(playTemplate, guajiGroup, subLabel, 2)
      numberPoolMin = hz.min
      numberPoolMax = hz.max
    } else if (betMode === 'dingwei' || typeId === 'g001' || guajiGroup === '一星') {
      segmentLen = 10
      segmentLabels = [...PK10_POSITION_LABELS]
    } else {
      segmentLen = pk10SegmentLen(typeId, typeLabel, subLabel, guajiGroup)
      segmentLabels = PK10_POSITION_LABELS.slice(0, segmentLen)
    }
    let inputMode = inputModeFromBetMode(betMode, subPlayId, segmentLen)
    if (betMode === 'daxiao' || betMode === 'danshuang' || betMode === 'hezhi' || betMode === 'longhu') inputMode = 'pool'
    if (betMode === 'dxds' && guajiGroup === '和值') inputMode = 'pool'
    return finishConfig({
      playTemplate, typeId, subId, betMode, typeLabel, subLabel, subPlayId,
      segmentLen, segmentLabels, inputMode,
      numberPoolMin: numberPoolMin ?? 1, numberPoolMax: numberPoolMax ?? 10,
      poolMaxPicks: poolMaxPicksPk10, guajiGroup,
    })
  }

  if (playTemplate === 'syxw_std') {
    if (typeId === 'g005' || typeId === 'renxuan_fs' || guajiGroup === '任选复式') {
      return finishConfig({
        playTemplate, typeId, subId, betMode: betMode || 'fushi', typeLabel, subLabel, subPlayId,
        segmentLen: 1, segmentLabels: ['选号'], inputMode: 'pool',
        numberPoolMin: 1, numberPoolMax: 11, guajiGroup,
      })
    }
    if (typeId === 'g006' || typeId === 'renxuan_ds' || guajiGroup === '任选单式') {
      return finishConfig({
        playTemplate, typeId, subId, betMode: betMode || 'danshi', typeLabel, subLabel, subPlayId,
        segmentLen: 1, segmentLabels: ['选号'], inputMode: 'danshi',
        numberPoolMin: 1, numberPoolMax: 11, guajiGroup,
      })
    }
    if (betMode === 'dingwei' || typeId === 'g003' || guajiGroup === '一星') {
      segmentLen = 5
      segmentLabels = [...SYXW_POSITION_LABELS]
    } else if (betMode === 'budingwei' || typeId === 'g004' || guajiGroup === '不定位') {
      segmentLen = 1
      segmentLabels = ['选号']
    } else if (betMode === 'zuxuan_fs' || betMode === 'zuxuan_ds') {
      segmentLen = 1
      segmentLabels = ['选号']
    } else {
      segmentLen = syxwSegmentLen(typeId, typeLabel, guajiGroup)
      segmentLabels = SYXW_POSITION_LABELS.slice(0, segmentLen)
    }
    let inputMode = inputModeFromBetMode(betMode, subPlayId, segmentLen)
    if (betMode === 'budingwei' || betMode === 'zuxuan_fs') inputMode = 'pool'
    if (betMode === 'zuxuan_ds' || betMode === 'danshi') inputMode = 'danshi'
    return finishConfig({
      playTemplate, typeId, subId, betMode, typeLabel, subLabel, subPlayId,
      segmentLen, segmentLabels, inputMode, numberPoolMin: 1, numberPoolMax: 11, guajiGroup,
    })
  }

  if (playTemplate === 'k3_std') {
    segmentLen = 1
    segmentLabels = [formatSubPlayLabel(subLabel) || '选号']
    let inputMode: PlayConfig['inputMode'] = 'pool'
    if (betMode === 'danshi' || subLabel.includes('手动') || subLabel.includes('三连号')) inputMode = 'danshi'
    if (betMode === 'hezhi' || typeId === 'g001' || typeLabel === '和值') {
      const hz = hezhiPoolRange(playTemplate, guajiGroup, subLabel, 3)
      numberPoolMin = hz.min
      numberPoolMax = hz.max
      inputMode = 'pool'
    } else {
      numberPoolMin = 1
      numberPoolMax = 6
    }
    return finishConfig({
      playTemplate, typeId, subId, betMode: betMode || 'fushi', typeLabel, subLabel, subPlayId,
      segmentLen, segmentLabels, inputMode, numberPoolMin, numberPoolMax, guajiGroup,
    })
  }

  // SSC / fast_ssc / 默认
  if (typeId === 'dingwei' || (isSSCPlayTemplate(playTemplate) && (isDingweiStarType(typeLabel, typeId, subLabel) || guajiGroup === '一星'))) {
    if (isDingweiFivePositionScheme(playTemplate, typeLabel, typeId, subLabel, subId, guajiGroup)) {
      segmentLen = 5
      segmentLabels = [...POSITION_LABELS]
    } else {
      segmentStart = dingweiPositionIndex(subId)
      segmentLen = 1
      segmentLabels = [dingweiPositionLabel(subLabel) ?? POSITION_LABELS[segmentStart] ?? formatSubPlayLabel(subLabel)]
    }
  } else if (typeId === 'renxuan' || typeId === 'g011' || guajiGroup === '任选' || typeLabel === '任选') {
    if (betMode === 'fushi' || subPlayId === 'zhixuan_fs') {
      segmentLen = 5
      segmentLabels = [...POSITION_LABELS]
    } else {
      segmentLen = 1
      segmentLabels = ['选号']
    }
  } else if (typeId === 'combo24') {
    const pos = combo24SegmentPositions(subId)
    segmentLen = pos.length
    segmentLabels = pos.map((i) => POSITION_LABELS[i])
  } else if (
    typeId === 'budingwei' || typeId === 'dxds' || typeId === 'g009' || typeId === 'g016' ||
    guajiGroup === '不定位' || guajiGroup === '大小单双' || typeLabel === '不定位' || typeLabel === '大小单双'
  ) {
    const seg = budingweiOrDxdsSegment(typeId, subId, subLabel, guajiFullName)
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = sscSegmentLabelsForMeta(guajiGroup || typeLabel, typeLabel, typeId, segmentStart, segmentLen, subLabel)
  } else if (typeId === 'pc28_20' || typeId === 'pc28_28' || isPc28ModeType(typeLabel, typeId)) {
    segmentLen = 1
    segmentLabels = [formatSubPlayLabel(subLabel) || '和值']
  } else if (typeId === 'longhu' || isLonghuPlayType(typeLabel, typeId) || guajiGroup === '龙虎') {
    segmentLen = 1
    segmentLabels = ['龙虎']
  } else if (typeLabel === '哈希玩法' || typeId === 'g017') {
    segmentLen = 1
    segmentLabels = ['选号']
  } else if (isSSCPlayTemplate(playTemplate)) {
    const seg = resolveSscSegmentMeta({ group: guajiGroup, typeLabel, typeId, subLabel, fullName: guajiFullName, team: guajiTeam })
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = sscSegmentLabelsForMeta(guajiGroup || typeLabel, typeLabel, typeId, segmentStart, segmentLen, subLabel)
  } else {
    const seg = catalogSegmentRange(playTemplate, typeId)
    segmentStart = seg.start
    segmentLen = seg.len
    segmentLabels = POSITION_LABELS.slice(segmentStart, segmentStart + segmentLen)
  }

  if ((betMode === 'danshuang' || betMode === 'daxiao' || betMode === 'zhuangxian') &&
      (subLabel.includes('和值') || subLabel.includes('尾数') || subLabel.includes('庄闲') || guajiFullName.includes('和值'))) {
    segmentLen = 1
    segmentLabels = ['选号']
  }

  const panelType = typeNode.panelType ?? ''
  let inputMode = inputModeFromPanelType(panelType, betMode, subPlayId, segmentLen) ?? inputModeFromBetMode(betMode, subPlayId, segmentLen)

  if ((guajiGroup === '任选' || typeLabel === '任选' || typeId === 'g011') &&
      (betMode === 'zuxuan_fs' || betMode === 'zu3' || betMode === 'zu6' || betMode === 'hezhi')) {
    inputMode = 'pool'
    segmentLen = 1
    segmentLabels = ['选号']
  }
  if ((guajiGroup === '任选' || typeLabel === '任选' || typeId === 'g011' || typeId === 'renxuan') &&
      (betMode === 'danshi' || betMode === 'zuxuan_ds' || betMode === 'hunhe')) {
    const k =
      renPickCount(subId) ||
      renxuanSegmentLenFromText(`${guajiTeam} ${guajiFullName} ${subLabel}`) ||
      2
    if (betMode === 'hunhe') {
      inputMode = 'danshi'
      segmentLen = 1
      segmentLabels = ['选号']
    } else {
      inputMode = 'danshi'
      segmentLen = k
      segmentLabels = ['选号']
      renPositionCount = k
    }
  }
  if (betMode === 'tuotou' || betMode.endsWith('_dp')) inputMode = 'danshi'
  if (typeId === 'longhu' || isLonghuPlayType(typeLabel, typeId) || guajiGroup === '龙虎' || betMode === 'longhu' || betMode === 'longhuhe') {
    segmentLen = 1
    segmentLabels = ['龙虎']
    inputMode = 'pool'
    poolMaxPicks = 1
  }

  if (betMode === 'hezhi' || (playTemplate === 'pc28_std' && (subLabel === '和值' || subId === 'hezhi'))) {
    inputMode = 'pool'
    const hzText = `${guajiGroup} ${guajiTeam} ${guajiFullName} ${subLabel}`
    const hzLen =
      segmentLen > 1
        ? segmentLen
        : hezhiDigitLenFromText(hzText, renxuanSegmentLenFromText(hzText))
    const hz = hezhiPoolRange(playTemplate, guajiGroup, subLabel, hzLen, guajiFullName || guajiTeam)
    numberPoolMin = hz.min
    numberPoolMax = hz.max
    segmentLen = 1
    segmentLabels = ['和值']
  }
  if (betMode === 'kuadu') {
    inputMode = 'pool'
    const kd = kuaduPoolRange()
    numberPoolMin = kd.min
    numberPoolMax = kd.max
    segmentLen = 1
    segmentLabels = ['跨度']
  }
  if (betMode === 'weishu') {
    inputMode = 'pool'
    const ws = weishuPoolRange()
    numberPoolMin = ws.min
    numberPoolMax = ws.max
    segmentLen = 1
    segmentLabels = ['尾数']
  }
  if (betMode === 'baodan') {
    inputMode = 'pool'
    numberPoolMin = 0
    numberPoolMax = 9
    segmentLen = 1
    segmentLabels = ['包胆']
    poolMaxPicks = 1
  }
  // 组三/组六/组选N/组选复式/不定位：单行号池（0–9 逗号多选）。
  // 前中后三等区位 resolve 会得到 segmentLen=3，若保留则录入框被当成「三位按位」只能输 3 段。
  if (
    betMode === 'zu3' ||
    betMode === 'zu6' ||
    betMode === 'zuxuan_fs' ||
    betMode === 'budingwei' ||
    ['zu24', 'zu12', 'zu60', 'zu30', 'zu120', 'zu20', 'zu10', 'zu5', 'zu4'].includes(betMode)
  ) {
    inputMode = 'pool'
    if (numberPoolMin == null) numberPoolMin = 0
    if (numberPoolMax == null) numberPoolMax = 9
    segmentLen = 1
    segmentLabels = ['选号']
  }
  if (betMode === 'teshu') {
    inputMode = 'pool'
    segmentLen = 1
    segmentLabels = ['特殊号']
  }
  if (playTemplate === 'pc28_std' && (betMode === 'hezhi' || subLabel === '和值' || subId === 'hezhi')) {
    inputMode = 'pool'
    numberPoolMin = 0
    numberPoolMax = 27
  }

  return finishConfig({
    playTemplate, typeId, subId, betMode, typeLabel, subLabel, subPlayId,
    segmentLen, segmentLabels, inputMode, numberPoolMin, numberPoolMax, poolMaxPicks, renPositionCount, guajiGroup,
  })
}

function finishConfig(input: {
  playTemplate: string
  typeId: string
  subId: string
  betMode: string
  typeLabel: string
  subLabel: string
  subPlayId: string
  segmentLen: number
  segmentLabels: string[]
  inputMode: PlayConfig['inputMode']
  numberPoolMin?: number
  numberPoolMax?: number
  poolMaxPicks?: number
  renPositionCount?: number
  guajiGroup: string
}): PlayTreePlayConfig {
  return {
    playTemplate: input.playTemplate,
    typeId: input.typeId,
    subId: input.subId,
    betMode: input.betMode,
    playTypeLabel: input.typeLabel,
    playMethodLabel: formatSubPlayLabel(input.subLabel),
    playTypeId: input.typeId,
    subPlayId: input.subPlayId,
    catalogSubId: input.subId,
    segmentLen: input.segmentLen,
    segmentLabels: input.segmentLabels,
    inputMode: input.inputMode,
    numberPoolMin: input.numberPoolMin,
    numberPoolMax: input.numberPoolMax,
    poolMaxPicks: input.poolMaxPicks,
    renPositionCount: input.renPositionCount,
    guajiGroup: input.guajiGroup || undefined,
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
