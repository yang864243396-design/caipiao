import { isLonghuPlayConfigLike } from '@/utils/runTypeMatrix'

/** 与后端 schemes.SupportsPlanContrary 对齐：无反集玩法不展示「计划反集」Tab */
export function supportsPlanContraryPlay(config: {
  betMode?: string
  playTypeId?: string
  playTypeLabel?: string
}): boolean {
  if (isLonghuPlayConfigLike(config)) return true
  const bm = String(config.betMode ?? '')
    .trim()
    .toLowerCase()
  switch (bm) {
    case 'hezhi':
    case 'kuadu':
    case 'teshu':
    case 'zu3':
    case 'zu6':
    case 'zuhe':
    case 'baodan':
    case 'hunhe':
    case 'zu24':
    case 'zu12':
    case 'zu60':
    case 'zu30':
    case 'zu120':
    case 'longhubao':
    case 'daxiao':
    case 'danshuang':
    case 'dxds':
    case 'danshi':
    case 'zhixuan_ds':
    case 'zuxuan_ds':
      return false
  }
  const ptid = String(config.playTypeId ?? '')
    .trim()
    .toLowerCase()
  switch (ptid) {
    case 'hezhi':
    case 'kuadu':
    case 'dxds':
    case 'dxds_combo':
    case 'daxiao':
    case 'danshuang':
    case 'longhubao':
      return false
  }
  return true
}
