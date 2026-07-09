/** 跟单大厅 Top10 演示玩法元数据（与 backend copyhall/defaults 对齐） */
export interface CopyHallPlayMeta {
  playMethod: string
  playTypeId: string
  subPlayId: string
}

export const COPY_HALL_PLAY_METHODS: readonly CopyHallPlayMeta[] = [
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
] as const

export function copyHallPlayMetaAt(index: number): CopyHallPlayMeta {
  return COPY_HALL_PLAY_METHODS[index] ?? COPY_HALL_PLAY_METHODS[0]
}
