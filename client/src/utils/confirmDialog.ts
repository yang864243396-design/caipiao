import { reactive } from 'vue'

export type ConfirmTone = 'primary' | 'warning' | 'danger'

export interface ConfirmOptions {
  title?: string
  message?: string
  icon?: string
  confirmText?: string
  cancelText?: string
  showCancel?: boolean
  tone?: ConfirmTone
}

interface ConfirmState extends Required<Omit<ConfirmOptions, 'icon'>> {
  visible: boolean
  icon: string
  resolve: ((v: boolean) => void) | null
}

const DEFAULTS: Omit<ConfirmState, 'visible' | 'resolve'> = {
  title: '提示',
  message: '',
  icon: '',
  confirmText: '确定',
  cancelText: '取消',
  showCancel: true,
  tone: 'primary',
}

export const confirmState = reactive<ConfirmState>({
  ...DEFAULTS,
  visible: false,
  resolve: null,
})

/**
 * 命令式弹出全局确认弹窗。
 * @returns Promise<boolean> 点击确定为 true，取消/遮罩关闭为 false。
 */
export function confirmDialog(options: ConfirmOptions = {}): Promise<boolean> {
  // 关闭上一次尚未结算的弹窗，避免状态串台
  if (confirmState.resolve) {
    confirmState.resolve(false)
    confirmState.resolve = null
  }
  Object.assign(confirmState, DEFAULTS, options)
  confirmState.visible = true
  return new Promise<boolean>((resolve) => {
    confirmState.resolve = resolve
  })
}

export function resolveConfirm(value: boolean): void {
  confirmState.visible = false
  if (confirmState.resolve) {
    confirmState.resolve(value)
    confirmState.resolve = null
  }
}
