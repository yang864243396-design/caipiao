/** 玩法大类 typeId → 中文名（与 backend play_types 种子一致） */
export const PLAY_TYPE_LABELS: Record<string, string> = {
  // ssc_std
  dingwei: '定位胆',
  qian3: '前三',
  zhong3: '中三',
  hou3: '后三',
  qian2: '前二',
  hou2: '后二',
  longhu: '龙虎',
  renxuan: '任选',
  qianzhonghou3: '前中后三',
  qianhou3: '前后三',
  budingwei: '不定位',
  combo24: '前后二/前后四',
  sixing: '四星',
  wuxing: '五星',
  dxds: '大小单双',
  hezhi: '和值',
  kuadu: '跨度',
  // lhc_std
  tema: '特码',
  erquanzhong: '二全中',
  erzhongte: '二中特',
  techuan: '特串',
  sanzhonger: '三中二',
  sanquanzhong: '三全中',
  shengxiao: '生肖',
  weishu: '尾数',
  buzhong_xuanyi: '不中/选一',
  guoguan: '过关',
  tematouwei: '特码头尾',
  wuxingjiaye: '五行家野',
  bose: '波色',
  qima: '七码',
  renzhong: '任中',
  // syxw_std
  renxuan_fs: '复式任选',
  renxuan_ds: '单式任选',
  // pk10_std
  qian1: '前一',
  qian4: '前四',
  qian5: '前五',
  daxiao: '大小',
  danshuang: '单双',
  dxds_combo: '大小单双',
  // k3_std
  tonghao: '同号',
  butonghao: '不同号',
  lianhao_qita: '连号与其它',
  // pc28_std
  pc28_20: '2.0',
  pc28_28: '2.8',
  // 兼容旧字段
  hou4: '后四',
}

/** betMode 与 typeId 别名（少数场景 typeId 缺失时回退） */
const BET_MODE_TYPE_LABELS: Record<string, string> = {
  longhuhe: '龙虎',
  longhubao: '龙虎豹',
}

export function resolvePlayTypeLabel(options: {
  typeId?: string
  playTypeId?: string
  playTypeLabel?: string
  betMode?: string
}): string {
  const fromTree = options.playTypeLabel?.trim()
  if (fromTree) return fromTree

  const typeId = (options.typeId ?? options.playTypeId ?? '').trim()
  if (typeId && PLAY_TYPE_LABELS[typeId]) return PLAY_TYPE_LABELS[typeId]!

  const betMode = (options.betMode ?? '').trim()
  if (betMode && PLAY_TYPE_LABELS[betMode]) return PLAY_TYPE_LABELS[betMode]!
  if (betMode && BET_MODE_TYPE_LABELS[betMode]) return BET_MODE_TYPE_LABELS[betMode]!

  return typeId || betMode
}
