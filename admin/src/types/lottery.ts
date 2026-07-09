export interface LotteryCatalogRow {

  code: string

  displayName: string

  categoryCode?: string

  playTemplate?: string

  ballCount?: number

  drawInterval?: string

  sortOrder: number

  onSale: boolean

  saleStatus: 'on_sale' | 'maintenance'

  outboundLotteryCode?: string

}



export interface PlayTemplateRow {

  code: string

  label: string

  version: number

}



export interface PlayTypeNode {

  typeId: string

  label: string

  sortOrder: number

  panelType?: string

  subPlays: SubPlayNode[]

}



export interface SubPlayNode {

  subId: string

  label: string

  sortOrder: number

  betMode?: string

  outboundPlayCode: string

}


