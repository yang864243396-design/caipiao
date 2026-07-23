<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  SSC_POSITION_LABELS,
  buildRenxuanPositionContent,
  defaultRenxuanPositions,
  parseRenxuanPositionContent,
  type PlayConfig,
} from '@/utils/betPayload'

const props = defineProps<{
  config: PlayConfig
  modelValue: string
  /** 详情只读：禁止编辑 */
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const pickCount = computed(() => {
  const k = props.config.renPositionCount ?? 0
  return k >= 2 && k <= 5 ? k : 2
})

const digitLen = computed(() =>
  props.config.segmentLen > 0 ? props.config.segmentLen : pickCount.value,
)

const positions = ref<string[]>([])
const picksText = ref('')
let syncing = false

function syncFromModel(raw: string) {
  const parsed = parseRenxuanPositionContent(raw, pickCount.value)
  positions.value = parsed.positions.length
    ? parsed.positions
    : defaultRenxuanPositions(pickCount.value)
  picksText.value = parsed.picks
}

function emitContent() {
  if (syncing || props.disabled) return
  const next = buildRenxuanPositionContent(positions.value, picksText.value)
  if (next !== props.modelValue) {
    emit('update:modelValue', next)
  }
}

watch(
  () =>
    [
      props.modelValue,
      props.config.renPositionCount,
      props.config.segmentLen,
      props.config.playTypeId,
      props.config.subPlayId,
    ] as const,
  () => {
    syncing = true
    syncFromModel(props.modelValue)
    syncing = false
  },
  { immediate: true },
)

watch([positions, picksText], emitContent, { deep: true })

function togglePosition(lab: string) {
  if (props.disabled) return
  const set = new Set(positions.value)
  if (set.has(lab)) {
    set.delete(lab)
  } else if (set.size < pickCount.value) {
    set.add(lab)
  } else {
    // 已满：替换最早选中的一位（保持恰好 k 个）
    const next = [...positions.value.slice(1), lab]
    positions.value = next
    return
  }
  // 按万千百十个顺序展示
  positions.value = SSC_POSITION_LABELS.filter((p) => set.has(p))
}

const placeholder = computed(() => {
  const example = `${'12'.slice(0, digitLen.value).padEnd(digitLen.value, '0')},34`
  return `从万、千、百、十、个中勾选 ${pickCount.value} 个位置，再输入 ${digitLen.value} 位号码组成一注；所选位置与号码顺序均须与开奖一致。示例：${example}`
})
</script>

<template>
  <div class="srd-panel" :class="{ 'is-disabled': disabled }">
    <div class="srd-pos-row">
      <span class="srd-pos-label">选位（{{ pickCount }}）</span>
      <div class="srd-chips" role="group" :aria-label="`从万千百十个中选 ${pickCount} 个位置`">
        <button
          v-for="lab in SSC_POSITION_LABELS"
          :key="lab"
          type="button"
          class="srd-chip"
          :class="{ 'is-active': positions.includes(lab) }"
          :disabled="disabled"
          @click="togglePosition(lab)"
        >
          {{ lab }}
        </button>
      </div>
    </div>
    <el-input
      :model-value="picksText"
      type="textarea"
      :rows="6"
      resize="none"
      class="srd-area"
      :placeholder="placeholder"
      :disabled="disabled"
      @update:model-value="(v: string) => { if (!disabled) picksText = v }"
    />
  </div>
</template>

<style scoped>
.srd-panel {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.srd-pos-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem 0.75rem;
}

.srd-pos-label {
  flex-shrink: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1a2332;
  font-family: 'Noto Sans SC', sans-serif;
}

.srd-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}

.srd-chip {
  min-width: 2.25rem;
  height: 2rem;
  padding: 0 0.625rem;
  border: none;
  border-radius: 0.5rem;
  background: #eef2f7;
  color: #3d4a5c;
  font-size: 0.8125rem;
  font-weight: 600;
  font-family: 'Inter', 'Noto Sans SC', sans-serif;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
}

.srd-chip:hover {
  background: #e2eaf4;
}

.srd-chip.is-active {
  background: #0066ff;
  color: #fff;
  box-shadow: 0 4px 12px rgba(0, 102, 255, 0.22);
}

.srd-chip:disabled {
  cursor: default;
}

.srd-chip:disabled:hover {
  background: #eef2f7;
}

.srd-chip.is-active:disabled:hover {
  background: #0066ff;
}

.srd-area {
  width: 100%;
}

.srd-area :deep(.el-textarea__inner) {
  min-height: 9.5rem;
  border: none;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  padding: var(--card-pad);
  font-family: 'Inter', 'Noto Sans SC', sans-serif;
  font-size: 0.9375rem;
  line-height: 1.65;
  box-shadow: none;
  white-space: pre-wrap;
}

.srd-area :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.srd-area :deep(.el-textarea__inner::placeholder) {
  color: #94a3b8;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
