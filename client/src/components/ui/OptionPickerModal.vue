<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'

/** 选项行数据；与表单绑定时 value 建议统一为 string，避免 === 误判 */
export interface OptionPickerItem {
  label: string
  value: string | number
}

const props = withDefaults(
  defineProps<{
    /** 是否显示弹层 */
    modelValue: boolean
    /** 当前已选值（打开时同步到内部草稿，仅在确定时回填） */
    selectedValue: string | number | null | undefined
    title: string
    options: OptionPickerItem[]
    /** 弹层卡片 `id`，供外部 `aria-controls` 引用 */
    panelId?: string
    /** primary：蓝色高亮；tertiary：玩法类型等场景用橙色高亮 */
    selectionAccent?: 'primary' | 'tertiary'
    showHeaderDivider?: boolean
    showFooterDivider?: boolean
    confirmText?: string
    /** 选项网格列数，默认可覆盖为 1 以适配纵向列表 */
    columns?: number
  }>(),
  {
    selectionAccent: 'primary',
    showHeaderDivider: true,
    showFooterDivider: true,
    confirmText: '确定',
    columns: 2,
  }
)

const emit = defineEmits<{
  'update:modelValue': [boolean]
  confirm: [value: string | number]
  cancel: []
}>()

const draft = ref<string | number>('')

function valuesEqual(a: string | number, b: string | number) {
  return String(a) === String(b)
}

function syncDraftFromProps() {
  const v = props.selectedValue
  if (v !== undefined && v !== null && props.options.some((o) => valuesEqual(o.value, v))) {
    draft.value = v
    return
  }
  if (props.options.length) draft.value = props.options[0]!.value
  else draft.value = ''
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) syncDraftFromProps()
  }
)

watch(
  () => [props.selectedValue, props.options] as const,
  () => {
    if (props.modelValue) syncDraftFromProps()
  },
  { deep: true }
)

function close() {
  emit('update:modelValue', false)
  emit('cancel')
}

function onBackdropPointerdown(ev: MouseEvent) {
  if (ev.target === ev.currentTarget) close()
}

function confirm() {
  emit('confirm', draft.value)
  emit('update:modelValue', false)
}

function onKeydown(ev: KeyboardEvent) {
  if (ev.key === 'Escape') {
    ev.stopPropagation()
    close()
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      document.addEventListener('keydown', onKeydown, true)
      const prev = document.body.style.overflow
      document.body.dataset.opmPrevOverflow = prev
      document.body.style.overflow = 'hidden'
    } else {
      document.removeEventListener('keydown', onKeydown, true)
      const prev = document.body.dataset.opmPrevOverflow ?? ''
      document.body.style.overflow = prev
      delete document.body.dataset.opmPrevOverflow
    }
  }
)

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown, true)
  if (document.body.dataset.opmPrevOverflow !== undefined) {
    document.body.style.overflow = document.body.dataset.opmPrevOverflow ?? ''
    delete document.body.dataset.opmPrevOverflow
  }
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="opm-backdrop"
      @pointerdown.self="onBackdropPointerdown"
    >
      <div
        :id="panelId"
        class="opm-panel"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        @pointerdown.stop
      >
        <div class="opm-header" :class="{ 'opm-header--plain': !showHeaderDivider }">
          <h2 class="opm-title">{{ title }}</h2>
          <button type="button" class="opm-close" aria-label="关闭" @click="close">
            <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true">
              <path
                fill="currentColor"
                d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"
              />
            </svg>
          </button>
        </div>

        <div class="opm-body">
          <div
            class="opm-grid"
            :style="{ '--opm-cols': columns }"
            role="listbox"
            :aria-label="title"
          >
            <button
              v-for="opt in options"
              :key="String(opt.value)"
              type="button"
              role="option"
              class="opm-option"
              :class="{
                'opm-option--sel-primary': selectionAccent === 'primary' && valuesEqual(draft, opt.value),
                'opm-option--sel-tertiary': selectionAccent === 'tertiary' && valuesEqual(draft, opt.value),
              }"
              :aria-selected="valuesEqual(draft, opt.value)"
              @click="draft = opt.value"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>

        <div class="opm-footer" :class="{ 'opm-footer--plain': !showFooterDivider }">
          <button type="button" class="opm-confirm" @click="confirm">
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.opm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 4000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--card-pad);
  background: rgba(224, 227, 229, 0.78);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-sizing: border-box;
}

.opm-panel {
  width: 100%;
  max-width: 22.5rem;
  max-height: min(90dvh, 640px);
  display: flex;
  flex-direction: column;
  background: #ffffff;
  border-radius: 1.5rem;
  box-shadow: 0 20px 60px rgba(25, 28, 30, 0.12);
  overflow: hidden;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
}

.opm-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 1.15rem 1.25rem 1rem;
  border-bottom: 1px solid rgba(241, 245, 249, 0.9);
  flex-shrink: 0;
}

.opm-header--plain {
  border-bottom: none;
}

.opm-title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 800;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #191c1e;
}

.opm-close {
  flex-shrink: 0;
  width: 2.5rem;
  height: 2.5rem;
  margin: -0.25rem -0.35rem -0.25rem 0;
  padding: 0;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: #727687;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition:
    background 0.15s,
    color 0.15s;
}

.opm-close:hover {
  background: #f2f4f6;
  color: #424656;
}

.opm-close:focus-visible {
  outline: 2px solid #0066ff;
  outline-offset: 2px;
}

.opm-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 1.25rem 1.25rem 0.75rem;
  -webkit-overflow-scrolling: touch;
}

.opm-grid {
  display: grid;
  grid-template-columns: repeat(var(--opm-cols, 2), minmax(0, 1fr));
  gap: 0.75rem;
}

.opm-option {
  min-height: 2.85rem;
  padding: 0.65rem 0.5rem;
  margin: 0;
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.35);
  background: #f2f4f6;
  color: #424656;
  font-size: 0.9375rem;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  text-align: center;
  line-height: 1.35;
  transition:
    transform 0.15s,
    box-shadow 0.15s,
    background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.opm-option:hover:not(.opm-option--sel-primary):not(.opm-option--sel-tertiary) {
  background: #eceef0;
}

.opm-option:active {
  transform: scale(0.98);
}

.opm-option--sel-primary {
  border-color: transparent;
  background: linear-gradient(145deg, #0050cb 0%, #0066ff 100%);
  color: #ffffff;
  box-shadow: 0 8px 22px rgba(0, 80, 203, 0.22);
}

.opm-option--sel-tertiary {
  border-color: transparent;
  background: linear-gradient(145deg, #a33200 0%, #cc4204 100%);
  color: #ffffff;
  box-shadow: 0 8px 22px rgba(163, 50, 0, 0.22);
}

.opm-footer {
  flex-shrink: 0;
  padding: 0.35rem 1.25rem 1.25rem;
  border-top: 1px solid rgba(241, 245, 249, 0.95);
}

.opm-footer--plain {
  border-top: none;
  padding-top: 0.15rem;
}

.opm-confirm {
  width: 100%;
  margin: 0;
  padding: 0.9rem 1rem;
  border: none;
  border-radius: 0.75rem;
  background: #0066ff;
  color: #ffffff;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: inherit;
  cursor: pointer;
  box-shadow: 0 8px 24px rgba(0, 102, 255, 0.28);
  transition:
    box-shadow 0.15s,
    transform 0.15s,
    background 0.15s;
}

.opm-confirm:hover {
  background: #0050cb;
  box-shadow: 0 10px 28px rgba(0, 102, 255, 0.34);
}

.opm-confirm:active {
  transform: scale(0.99);
}

.opm-confirm:focus-visible {
  outline: 2px solid #0050cb;
  outline-offset: 2px;
}
</style>
