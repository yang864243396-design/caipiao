<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PlayConfig } from '@/utils/betPayload'
import {
  groupDigitInputHint,
  schemeGroupContentHasDigits,
  schemeGroupContentToInputBox,
  schemeGroupInputBoxToContent,
} from '@/utils/pickPanelOptions'

/**
 * 数字玩法方案内容输入面板（对齐第三方）：单个输入框，不点选、不分位框。
 *
 * 录入格式：逗号分隔各位（万,千,百,十,个…），每位号码连写；如「123,34,56,78,56」表示
 * 万位取 1/2/3、千位取 3/4、…。单位型玩法（定位胆/号池）直接连写号码「123」即可。
 *
 * 内部按号池 token 宽度拆分（0-9 按 1 位、11选5 等按 2 位补零），并与引擎所需的
 * 「每位一行、逗号分隔」内容格式双向转换（显示压缩、存储按位换行）。
 */
const props = withDefaults(
  defineProps<{
    config: PlayConfig
    modelValue: string
    /** 详情只读：禁止编辑 */
    disabled?: boolean
    /** 文本行数（紧凑场景可缩小） */
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
    // 禁止 trim 比较：定位胆 "\n\n1,2\n\n" 与错误的 "1,2\n\n\n\n" trim 后相同，会跳过纠正
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
