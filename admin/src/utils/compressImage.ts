/**
 * 管理端上传前压缩图片：限制最长边与体积，减小上传/加载耗时。
 * GIF 保留不动（避免破坏动图）；已很小的文件跳过。
 */
export interface CompressImageOptions {
  /** 最长边像素，默认 1920 */
  maxEdge?: number
  /** JPEG/WebP 质量 0~1，默认 0.82 */
  quality?: number
  /** 目标体积上限（字节），默认 800KB；超出则继续降质 */
  maxBytes?: number
  /** 小于该体积则不压缩，默认 200KB */
  skipBelowBytes?: number
}

function loadImage(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve(img)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片读取失败'))
    }
    img.src = url
  })
}

function canvasToBlob(
  canvas: HTMLCanvasElement,
  type: string,
  quality: number,
): Promise<Blob | null> {
  return new Promise((resolve) => {
    canvas.toBlob((blob) => resolve(blob), type, quality)
  })
}

function outputMime(file: File): { type: string; ext: string } {
  // PNG 透明图转 JPEG 会丢透明；优先 WebP，失败再 JPEG
  if (file.type === 'image/png' || file.type === 'image/webp') {
    return { type: 'image/webp', ext: 'webp' }
  }
  return { type: 'image/jpeg', ext: 'jpg' }
}

function renameExt(name: string, ext: string): string {
  const base = name.replace(/\.[^.]+$/, '') || 'image'
  return `${base}.${ext}`
}

export async function compressImageFile(
  file: File,
  opts: CompressImageOptions = {},
): Promise<File> {
  if (!file.type.startsWith('image/')) return file
  // 动图不压
  if (file.type === 'image/gif') return file

  const maxEdge = opts.maxEdge ?? 1920
  const skipBelow = opts.skipBelowBytes ?? 200 * 1024
  const maxBytes = opts.maxBytes ?? 800 * 1024
  let quality = opts.quality ?? 0.82

  if (file.size <= skipBelow) return file

  let img: HTMLImageElement
  try {
    img = await loadImage(file)
  } catch {
    return file
  }

  const scale = Math.min(1, maxEdge / Math.max(img.naturalWidth, img.naturalHeight))
  const w = Math.max(1, Math.round(img.naturalWidth * scale))
  const h = Math.max(1, Math.round(img.naturalHeight * scale))

  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')
  if (!ctx) return file

  // JPEG 无透明通道时铺白底，避免黑底
  const { type, ext } = outputMime(file)
  if (type === 'image/jpeg') {
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, w, h)
  }
  ctx.drawImage(img, 0, 0, w, h)

  let blob = await canvasToBlob(canvas, type, quality)
  // WebP 不支持时回退 JPEG
  if (!blob && type === 'image/webp') {
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, w, h)
    ctx.drawImage(img, 0, 0, w, h)
    blob = await canvasToBlob(canvas, 'image/jpeg', quality)
  }
  if (!blob) return file

  // 仍过大则逐步降质
  while (blob.size > maxBytes && quality > 0.5) {
    quality = Math.max(0.5, quality - 0.1)
    const next = await canvasToBlob(canvas, blob.type, quality)
    if (!next || next.size >= blob.size) break
    blob = next
  }

  // 压缩后反而更大则用原图
  if (blob.size >= file.size) return file

  return new File([blob], renameExt(file.name, blob.type.includes('webp') ? 'webp' : ext), {
    type: blob.type,
    lastModified: Date.now(),
  })
}
