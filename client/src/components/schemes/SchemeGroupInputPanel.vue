<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PlayConfig } from '@/utils/betPayload'
import { groupContentPlaceholder, isHunhePlayConfig, isSscDanshiLikeConfig } from '@/utils/betPayload'
import {
  commitSchemeGroupContentOnBlur,
  groupDigitInputHint,
  schemeGroupContentHasDigits,
  schemeGroupContentToInputBox,
  schemeGroupInputBoxToContent,
  schemeGroupUsesDanshiTextInput,
} from '@/utils/pickPanelOptions'

/**
 * 方案内容文本输入面板：
 * - 复式/号池：逗号分隔各位，显示压缩、存储按位换行
 * - 直选/组选/混合单式：整注逗号分隔，失焦按位长过滤非法票
 */
const props = withDefaults(
  defineProps<{
    config: PlayConfig
    modelValue: string
    /** 详情只读：禁止编辑 */
    disabled?: boolean
    /** 文本行数（紧凑场景可缩小） */
    rows?: number
    /** 覆盖默认按玩法生成的 placeholder（如任选选位面板） */
    placeholder?: string
  }>(),
  { rows: 6 },
)

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const raw = ref('')
const rowCount = computed(() => Math.max(2, Math.trunc(props.rows || 6)))

/** 整注单式/混合：不做按位 box↔content */
const isTicketTextMode = computed(
  () =>
    schemeGroupUsesDanshiTextInput(props.config) ||
    props.config.inputMode === 'danshi' ||
    isSscDanshiLikeConfig(props.config) ||
    isHunhePlayConfig(props.config),
)

function boxToContent(box: string): string {
  const src = String(box ?? '').replace(/\r/g, '')
  if (isTicketTextMode.value) return src
  return schemeGroupInputBoxToContent(src, props.config)
}

function contentToBox(content: string): string {
  const src = String(content ?? '').replace(/\r/g, '')
  if (isTicketTextMode.value) return src
  return schemeGroupContentToInputBox(src, props.config)
}

function syncFromModel(content: string): void {
  const src = String(content ?? '')
  raw.value = contentToBox(src)
  // 仅逗号/空白的空槽（如历史误存的 ,,,,）归一为空，露出 placeholder
  if (src !== '' && !schemeGroupContentHasDigits(src)) {
    emit('update:modelValue', '')
  }
}

function onInput(value: string): void {
  if (props.disabled) return
  raw.value = value
  emit('update:modelValue', boxToContent(value))
}

function onBlur(): void {
  if (props.disabled) return
  const content = commitSchemeGroupContentOnBlur(raw.value, props.config)
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
      props.config.playMethodLabel,
    ] as const,
  () => syncFromModel(props.modelValue),
  { immediate: true },
)

watch(
  () => props.modelValue,
  (value) => {
    // 禁止 trim 比较：定位胆 "\n\n1,2\n\n" 与错误的 "1,2\n\n\n\n" trim 后相同，会跳过纠正
    const next = String(value ?? '').replace(/\r/g, '')
    if (boxToContent(raw.value) !== next) syncFromModel(next)
  },
)

const poolHint = computed(() => {
  const override = String(props.placeholder ?? '').trim()
  if (override) return override
  if (isTicketTextMode.value) return groupContentPlaceholder(props.config)
  return groupDigitInputHint(props.config)
})
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
  padding: var(--card-pad);
  font-size: 0.9375rem;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  line-height: 1.65;
  box-shadow: none;
  white-space: pre-wrap;
}

.sgi-panel.is-compact .sgi-input :deep(.el-textarea__inner) {
  min-height: 4.5rem;
  padding: var(--card-pad);
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
