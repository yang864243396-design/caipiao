/**
 * 全站演示账号 / 钱包 / 品牌文案（与会员中心首页同源）。
 * 附录 A 关键单号与方案 id 见 `@shared/mock/appendixMock`。
 */

/** 产品对外品牌（页签标题、大厅顶栏等） */
export const demoAppBrand = '精密终端'

/** 当前演示用户身份（会员中心身份卡） */
export const demoUser = {
  name: 'vs8888',
  account: 'vs8888',
  platform: 'V6哈希',
  logoText: '富联',
} as const

/** 钱包余额（数值），首页与充值/提现/资料等页共用 */
export const demoBalanceNumber = 12888.66

/** 首页仪表：当前下注、盈亏、余额展示（千分位字符串与货币单位） */
export const demoStats = {
  betting: '2,860.00',
  pnl: '198.32',
  balance: demoBalanceNumber.toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }),
  currency: '¥',
} as const

/** 资料页等：彩票返点（演示） */
export const demoProfile = {
  rebatePercent: 9,
} as const

/** 与 demoStats.balance 一致，供帐变等仅需要「当前余额字符串」的场景 */
export const demoBalanceFormatted = demoStats.balance
