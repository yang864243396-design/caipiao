export interface PublicLotteryRow {
  code: string
  displayName: string
  categoryCode: string
  playTemplate: string
  ballCount: number
  drawInterval?: string
  sortOrder: number
  outboundLotteryCode: string
}

export interface SubPlayNode {
  subId: string
  label: string
  sortOrder: number
  betMode?: string
  outboundPlayCode: string
  segmentRule?: Record<string, unknown>
}

export interface PlayTypeNode {
  typeId: string
  label: string
  sortOrder: number
  panelType?: string
  subPlays: SubPlayNode[]
}

export interface PlayTreeResponse {
  lotteryCode: string
  displayName: string
  playTemplate: string
  playTypes: PlayTypeNode[]
}
