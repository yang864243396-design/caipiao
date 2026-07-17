/**
 * 复制文本到剪贴板。
 *
 * navigator.clipboard 仅在安全上下文（HTTPS / localhost）可用；线上若走 HTTP
 * 或 WebView 内则不可用。此处提供 execCommand('copy') 降级方案，保证多数环境可用。
 */
export async function copyText(text: string): Promise<boolean> {
  const value = String(text ?? '')
  if (!value) return false

  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      // 继续降级
    }
  }

  return legacyCopy(value)
}

function legacyCopy(value: string): boolean {
  try {
    const textarea = document.createElement('textarea')
    textarea.value = value
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.top = '-9999px'
    textarea.style.left = '-9999px'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)

    const selection = document.getSelection()
    const savedRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null

    textarea.select()
    textarea.setSelectionRange(0, value.length)
    const ok = document.execCommand('copy')

    document.body.removeChild(textarea)
    if (savedRange && selection) {
      selection.removeAllRanges()
      selection.addRange(savedRange)
    }
    return ok
  } catch {
    return false
  }
}
