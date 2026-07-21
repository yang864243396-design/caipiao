<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PlayConfig } from '@client/utils/betPayload'
import {
  digitOptionsForConfig,
  groupDigitInputHint,
  poolMaxPicksForConfig,
  schemeGroupContentToInputBox,
} from '@client/utils/pickPanelOptions'

/**
 * 数字玩法方案内容输入面板（对齐第三方）：单个输入框，不点选、不分位框。
 *
 * 录入格式：逗号分隔各位（万,千,百,十,个…），每位号码连写；如「123,34,56,78,56」表示
 * 万位取 1/2/3、千位取 3/4、…。单位型玩法（定位胆/号池）直接连写号码「123」即可。
 *
 * 内部按号池 token 宽度拆分（0-9 按 1 位、11选5 等按 2 位补零），并与引擎所需的
 * 「每位一行、逗号分隔」内容格式双向转换（显示压缩、存储按位换行）。
 */
const props = defineProps<{
  config: PlayConfig
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const options = computed(() => digitOptionsForConfig(props.config))
const tokenWidth = computed(() => options.value[0]?.length || 1)
const segLen = computed(() => Math.max(1, props.config.segmentLen || 1))
const maxPicks = computed(() => poolMaxPicksForConfig(props.config))

const raw = ref('')

/** 解析单位内的连写号码为号池合法 token（按 token 宽度切块、去重、补零形态） */
function parseSegment(seg: string): string[] {
  const digits = String(seg ?? '').replace(/\D/g, '')
  const w = tokenWidth.value
  const seen = new Set<string>()
  const out: string[] = []
  for (let i = 0; i + w <= digits.length; i += w) {
    const chunk = digits.slice(i, i + w)
    const n = Number(chunk)
    const match = options.value.find((o) => Number(o) === n)
    if (!match || seen.has(match)) continue
    seen.add(match)
    out.push(match)
  }
  return out
}

/** 录入框（压缩格式）→ 引擎内容（单位型单行、多位型按位换行） */
function boxToContent(box: string): string {
  if (segLen.value <= 1) {
    let toks = parseSegment(box)
    const cap = maxPicks.value
    if (cap != null && cap > 0) toks = toks.slice(0, cap)
    return toks.join(',')
  }
  const segs = String(box ?? '').split(/[,，]/)
  const lines: string[] = []
  let any = false
  for (let i = 0; i < segLen.value; i++) {
    const toks = parseSegment(segs[i] ?? '')
    if (toks.length) any = true
    lines.push(toks.join(','))
  }
  return any ? lines.join('\n') : ''
}

function contentToBox(content: string): string {
  return schemeGroupContentToInputBox(content, props.config)
}

function syncFromModel(content: string): void {
  raw.value = contentToBox(content)
}

function onInput(value: string): void {
  raw.value = value
  emit('update:modelValue', boxToContent(value))
}

function onBlur(): void {
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
    // 禁止 trim 比较：定位胆前导空位 trim 后会与错位内容相等，跳过纠正
    const next = String(value ?? '').replace(/\r/g, '')
    if (boxToContent(raw.value) !== next) syncFromModel(next)
  },
)

const poolHint = computed(() => groupDigitInputHint(props.config))
</script>

<template>
  <div class="sgi-panel">
    <el-input
      :model-value="raw"
      type="textarea"
      :rows="6"
      resize="none"
      class="sgi-input"
      :placeholder="poolHint"
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

.sgi-input :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.sgi-input :deep(.el-textarea__inner::placeholder) {
  color: #94a3b8;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
