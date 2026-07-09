/** 将后端/历史遗留的 guaji 原始错误转为用户可读文案 */
export function formatGuajiAccountError(raw?: string): string {
  if (!raw?.trim()) return ''
  const msg = raw.trim()
  if (/status 502|status 503|status 504|Bad gateway|Cloudflare/i.test(msg)) {
    return '第三方服务暂时不可用，请稍后重试'
  }
  if (/status 401|status 403|授权已失效|token expired/i.test(msg)) {
    return '授权已失效，请重新授权'
  }
  if (/timeout|deadline exceeded|connection refused|no such host/i.test(msg)) {
    return '第三方服务连接失败，请稍后重试'
  }
  if (/guaji http|body=\{/.test(msg)) {
    return '第三方服务异常，请稍后重试'
  }
  return msg
}

/** API 异常文案：优先按第三方文档规则归一化，再回退原始 message */
export function formatClientApiError(err: unknown, fallback = '操作失败'): string {
  if (err instanceof Error && err.message) {
    return formatGuajiAccountError(err.message) || err.message
  }
  return fallback
}
