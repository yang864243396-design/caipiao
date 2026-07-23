/** 币种展示符号：与会员中心帐户余额一致 */
export function currencySymbol(code: string): string {
  switch (String(code ?? '').trim().toUpperCase()) {
    case 'CNY':
      return '¥'
    case 'USDT':
      return '₮'
    case 'TRX':
      return 'Ŧ'
    default:
      return code
  }
}
