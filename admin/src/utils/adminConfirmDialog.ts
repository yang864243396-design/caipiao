import { reactive } from 'vue'

export type AdminConfirmTone = 'primary' | 'warning' | 'danger'

export interface AdminConfirmOptions {
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  showCancel?: boolean
  tone?: AdminConfirmTone
}

export interface AdminPromptOptions extends AdminConfirmOptions {
  inputPlaceholder?: string
  inputValue?: string
}

interface AdminConfirmState extends Required<Omit<AdminConfirmOptions, never>> {
  visible: boolean
  mode: 'confirm' | 'prompt'
  promptPlaceholder: string
  promptValue: string
  resolveConfirm: ((v: boolean) => void) | null
  resolvePrompt: ((v: string | null) => void) | null
}

const DEFAULTS: Omit<AdminConfirmState, 'visible' | 'mode' | 'promptPlaceholder' | 'promptValue' | 'resolveConfirm' | 'resolvePrompt'> = {
  title: '提示',
  message: '',
  confirmText: '确定',
  cancelText: '取消',
  showCancel: true,
  tone: 'primary',
}

export const adminConfirmState = reactive<AdminConfirmState>({
  ...DEFAULTS,
  visible: false,
  mode: 'confirm',
  promptPlaceholder: '',
  promptValue: '',
  resolveConfirm: null,
  resolvePrompt: null,
})

function resetPending(): void {
  if (adminConfirmState.resolveConfirm) {
    adminConfirmState.resolveConfirm(false)
    adminConfirmState.resolveConfirm = null
  }
  if (adminConfirmState.resolvePrompt) {
    adminConfirmState.resolvePrompt(null)
    adminConfirmState.resolvePrompt = null
  }
}

/** 命令式确认弹窗，替代 ElMessageBox.confirm */
export function adminConfirmDialog(options: AdminConfirmOptions = {}): Promise<boolean> {
  resetPending()
  Object.assign(adminConfirmState, DEFAULTS, options, {
    visible: true,
    mode: 'confirm',
    promptPlaceholder: '',
    promptValue: '',
    resolvePrompt: null,
  })
  return new Promise<boolean>((resolve) => {
    adminConfirmState.resolveConfirm = resolve
  })
}

/** 命令式输入弹窗，替代 ElMessageBox.prompt */
export function adminPromptDialog(options: AdminPromptOptions = {}): Promise<string | null> {
  resetPending()
  Object.assign(adminConfirmState, DEFAULTS, options, {
    visible: true,
    mode: 'prompt',
    promptPlaceholder: options.inputPlaceholder ?? '',
    promptValue: options.inputValue ?? '',
    resolveConfirm: null,
  })
  return new Promise<string | null>((resolve) => {
    adminConfirmState.resolvePrompt = resolve
  })
}

export function resolveAdminConfirm(value: boolean): void {
  adminConfirmState.visible = false
  if (adminConfirmState.resolveConfirm) {
    adminConfirmState.resolveConfirm(value)
    adminConfirmState.resolveConfirm = null
  }
}

export function resolveAdminPrompt(value: string | null): void {
  adminConfirmState.visible = false
  if (adminConfirmState.resolvePrompt) {
    adminConfirmState.resolvePrompt(value)
    adminConfirmState.resolvePrompt = null
  }
}
