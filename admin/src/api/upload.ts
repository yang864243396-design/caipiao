import { API_BASE } from './config'
import {
  handleSessionExpired,
  hangAfterSessionExpired,
  isSessionExpiredError,
  SessionExpiredError,
} from './authSession'
import { ApiError, getAccessToken } from './client'

export interface CmsUploadResult {
  url: string
}

export async function uploadCmsImage(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const headers: Record<string, string> = {}
  const token = getAccessToken()
  if (token) headers.Authorization = `Bearer ${token}`

  const url = `${API_BASE}/admin/content/uploads/image`

  try {
    const res = await fetch(url, { method: 'POST', headers, body: form })
    const text = await res.text()
    const parsed = text
      ? (JSON.parse(text) as { code: number; message: string; data?: CmsUploadResult })
      : null

    if (!res.ok || !parsed) {
      if (res.status === 401) {
        await handleSessionExpired()
        throw new SessionExpiredError()
      }
      throw new ApiError(parsed?.message || res.statusText || '上传失败', res.status, parsed?.code)
    }
    if (parsed.code !== 0) {
      throw new ApiError(parsed.message || '上传失败', res.status, parsed.code)
    }
    if (!parsed.data?.url) {
      throw new ApiError('上传响应无效', res.status, parsed.code)
    }
    return parsed.data.url
  } catch (err) {
    if (isSessionExpiredError(err)) {
      return hangAfterSessionExpired<string>()
    }
    throw err
  }
}
