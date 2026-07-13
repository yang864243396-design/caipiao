import { onMounted, onUnmounted, readonly, ref, type Ref } from 'vue'

/** ≥1920×1080 屏幕启用桌面 Web 布局（按 screen 物理分辨率，非视口） */
export const LAYOUT_WEB_MIN_WIDTH = 1920
export const LAYOUT_WEB_MIN_HEIGHT = 1080

const LAYOUT_WEB_CLASS = 'layout-web'

let sharedIsWeb: Ref<boolean> | null = null
let sharedListener: (() => void) | null = null
let subscriberCount = 0

function readIsWebScreen(): boolean {
  if (typeof window === 'undefined') return false
  const s = window.screen
  if (!s) return false
  const w = Math.max(s.width || 0, s.availWidth || 0)
  const h = Math.max(s.height || 0, s.availHeight || 0)
  return w >= LAYOUT_WEB_MIN_WIDTH && h >= LAYOUT_WEB_MIN_HEIGHT
}

function applyHtmlClass(on: boolean) {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle(LAYOUT_WEB_CLASS, on)
}

function sync() {
  if (!sharedIsWeb) return
  const next = readIsWebScreen()
  if (sharedIsWeb.value !== next) {
    sharedIsWeb.value = next
  }
  applyHtmlClass(next)
}

function ensureShared() {
  if (sharedIsWeb) return
  sharedIsWeb = ref(false)
  if (typeof window === 'undefined') return
  sync()
  sharedListener = () => sync()
  window.addEventListener('resize', sharedListener)
}

/**
 * 桌面 Web 布局模式（屏幕分辨率 ≥1920×1080）。
 * 在 html 上同步 `layout-web` class，供全局 CSS 与组件使用。
 */
export function useLayoutMode() {
  ensureShared()
  const isWeb = sharedIsWeb!

  onMounted(() => {
    subscriberCount += 1
    ensureShared()
    sync()
  })

  onUnmounted(() => {
    subscriberCount = Math.max(0, subscriberCount - 1)
    if (subscriberCount === 0 && sharedListener) {
      window.removeEventListener('resize', sharedListener)
      sharedListener = null
      // 保留 sharedIsWeb 与 html class，避免路由切换闪烁
    }
  })

  return {
    isWeb: readonly(isWeb),
  }
}

/** 在应用启动时尽早同步 html class（无需等待组件挂载） */
export function initLayoutMode() {
  if (typeof window === 'undefined') return
  ensureShared()
  sync()
}
