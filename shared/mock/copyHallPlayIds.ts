/** 跟单大厅榜单玩法 ID（与 backend copy_hall_rank_slots 对齐） */
export interface CopyHallPlayIds {
  playTypeId: string
  subPlayId: string
}

/** 演示榜单玩法展示名 → 结构化 ID（API 未返回时 fallback） */
export const COPY_HALL_PLAY_SLOT_META: readonly ({ playMethod: string } & CopyHallPlayIds)[] = [
  { playMethod: '定位胆万位', playTypeId: 'dingwei', subPlayId: 'dingwei_wan' },
  { playMethod: '定位胆后二', playTypeId: 'hou2', subPlayId: 'hou2_zhixuan_fs' },
  { playMethod: '定位胆十位', playTypeId: 'dingwei', subPlayId: 'dingwei_shi' },
  { playMethod: '定位胆个位', playTypeId: 'dingwei', subPlayId: 'dingwei_ge' },
  { playMethod: '组选六', playTypeId: 'zhong3', subPlayId: 'zhong3_zu6' },
  { playMethod: '定位胆前三', playTypeId: 'qian3', subPlayId: 'qian3_zhixuan_fs' },
  { playMethod: '任选四', playTypeId: 'renxuan', subPlayId: 'ren4_zu24' },
  { playMethod: '定位胆后一', playTypeId: 'dingwei', subPlayId: 'dingwei_ge' },
  { playMethod: '定位胆千位', playTypeId: 'dingwei', subPlayId: 'dingwei_qian' },
  { playMethod: '定位胆任二', playTypeId: 'renxuan', subPlayId: 'ren2_zhixuan_fs' },
]

export function playIdsForCopyHallMethod(playMethod: string): CopyHallPlayIds {
  const hit = COPY_HALL_PLAY_SLOT_META.find((m) => m.playMethod === playMethod)
  if (hit) return { playTypeId: hit.playTypeId, subPlayId: hit.subPlayId }
  return { playTypeId: 'dingwei', subPlayId: 'dingwei_wan' }
}
