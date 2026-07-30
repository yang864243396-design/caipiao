/**
 * 用户端全局连点限制：同一可点击元素 1s 内只响应一次，避免重复触发接口。
 * 选号芯片 / 步进器 / 开关等纯 UI 控件排除。
 */

const DEFAULT_MS = 1000

/** 需要连点限制的操作控件 */
const ACTION_SELECTOR = [
  'button',
  'a.el-button',
  '.el-button',
  '[role="button"]',
].join(',')

/** 高频点选 / 表单控件：不做连点限制 */
const SKIP_CLOSEST_SELECTOR = [
  '[data-click-throttle="off"]',
  '.dock-pick-chip',
  '.play-picker-chip',
  '.play-picker-sub',
  '.sgp-chip',
  '.srd-chip',
  '.scf-seg-btn',
  '.scf-stepper-btn',
  '.scf-trig-pos-chip',
  '.scr-trig-pos-chip',
  '.scr-stepper-btn',
  '.scr-hcw-qbtn',
  '.pick-digit',
  '.digit-chip',
  '.el-switch',
  '.el-checkbox',
  '.el-radio',
  '.el-radio-button',
  '.el-input-number__decrease',
  '.el-input-number__increase',
  '.el-select-dropdown',
  '.el-picker-panel',
  '.el-collapse-item__header',
  '.el-tabs__item',
  // 会话过期等系统弹窗：确认按钮必须首击生效，不能被连点节流吞掉
  '.el-message-box',
  '.el-overlay.is-message-box',
  '.session-expired-msgbox',
  'input',
  'textarea',
  'select',
  'label',
].join(',')

let installed = false

export function installGlobalClickThrottle(ms: number = DEFAULT_MS): void {
  if (installed || typeof document === 'undefined') return
  installed = true

  const lastByEl = new WeakMap<EventTarget, number>()

  document.addEventListener(
    'click',
    (ev) => {
      const raw = ev.target
      if (!(raw instanceof Element)) return

      const el = raw.closest(ACTION_SELECTOR)
      if (!(el instanceof HTMLElement)) return
      if (el.closest(SKIP_CLOSEST_SELECTOR)) return

      if (
        (el instanceof HTMLButtonElement && el.disabled) ||
        el.getAttribute('aria-disabled') === 'true' ||
        el.classList.contains('is-disabled')
      ) {
        return
      }

      const now = Date.now()
      const prev = lastByEl.get(el) ?? 0
      if (now - prev < ms) {
        ev.preventDefault()
        ev.stopImmediatePropagation()
        return
      }
      lastByEl.set(el, now)
    },
    true,
  )
}

export const CLICK_THROTTLE_MS = DEFAULT_MS
