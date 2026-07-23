<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PlayConfig } from '@client/utils/betPayload'
import {
  groupDigitInputHint,
  schemeGroupContentToInputBox,
  schemeGroupInputBoxToContent,
} from '@client/utils/pickPanelOptions'

/**
 * 数字玩法方案内容输入面板（对齐第三方）：单个输入框，不点选、不分位框。
 * 与 client SchemeGroupInputPanel 同源转换逻辑。
 */
const props = withDefaults(
  defineProps<{
    config: PlayConfig
    modelValue: string
    disabled?: boolean
    rows?: number
  }>(),
  { rows: 6 },
)

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const raw = ref('')
const rowCount = computed(() => Math.max(2, Math.trunc(props.rows || 6)))

function boxToContent(box: string): string {
  return schemeGroupInputBoxToContent(box, props.config)
}

function contentToBox(content: string): string {
  return schemeGroupContentToInputBox(content, props.config)
}

function syncFromModel(content: string): void {
  raw.value = contentToBox(content)
}

function onInput(value: string): void {
  if (props.disabled) return
  raw.value = value
  emit('update:modelValue', boxToContent(value))
}

function onBlur(): void {
  if (props.disabled) return
  const content = boxToContent(raw.value)
  raw.value = contentToBox(content)
  emit('update:modelValue', content)
}

watch(
  () =>
    [
      props.config.inputMode,
      props.config.betMode,
      props.config.playTypeId,
      props.config.subPlayId,
      props.config.numberPoolMin,
      props.config.numberPoolMax,
      props.config.segmentLen,
    ] as const,
  () => syncFromModel(props.modelValue),
  { immediate: true },
)

watch(
  () => props.modelValue,
  (value) => {
    const next = String(value ?? '').replace(/\r/g, '')
    if (boxToContent(raw.value) !== next) syncFromModel(next)
  },
)

const poolHint = computed(() => groupDigitInputHint(props.config))
</script>

<template>
  <div class="sgi-panel" :class="{ 'is-disabled': disabled, 'is-compact': rowCount <= 3 }">
    <el-input
      :model-value="raw"
      type="textarea"
      :rows="rowCount"
      resize="none"
      class="sgi-input"
      :placeholder="poolHint"
      :disabled="disabled"
      @update:model-value="onInput"
      @blur="onBlur"
    />
  </div>
</template>

<style scoped>
.sgi-panel {
  width: 100%;
}

.sgi-input {
  width: 100%;
}

.sgi-input :deep(.el-textarea__inner) {
  min-height: 9.5rem;
  border: none;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  padding: 1rem 1.1rem;
  font-size: 0.9375rem;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  line-height: 1.65;
  box-shadow: none;
  white-space: pre-wrap;
}

.sgi-panel.is-compact .sgi-input :deep(.el-textarea__inner) {
  min-height: 4.5rem;
  padding: 0.65rem 0.85rem;
  font-size: 0.875rem;
  line-height: 1.5;
}

.sgi-input :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.sgi-input :deep(.el-textarea__inner::placeholder) {
  color: #94a3b8;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
