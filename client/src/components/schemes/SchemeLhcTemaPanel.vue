<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  LHC_TEMA_QUICK_OPTIONS,
  isLhcTemaWaveOption,
} from '@/constants/lhcPlay'
import {
  normalizeLhcTemaContent,
  parseLhcTemaParts,
  type PlayConfig,
} from '@/utils/betPayload'

const props = defineProps<{
  config: PlayConfig
  modelValue: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

/** 输入框只展示号码，不展示属性/波色 */
const numberDraft = ref('')

const parts = computed(() => parseLhcTemaParts(props.modelValue))
const selected = computed(() => new Set([...parts.value.attrs, ...parts.value.waves]))

function numsKey(raw: string): string {
  return parseLhcTemaParts(raw).nums.join(',')
}

function compose(numsRaw: string, attrs: string[], waves: string[]): string {
  const n = String(numsRaw ?? '').trim()
  const a = attrs.join(',')
  const w = waves.join(',')
  if (!n && !a && !w) return ''
  return `${n}|${a}|${w}`
}

watch(
  () => props.modelValue,
  (v) => {
    const nextNums = parseLhcTemaParts(v).nums.join(',')
    // 用规范化后的号码比较，避免输入「1」时被写成「01」打断打字
    if (numsKey(numberDraft.value) !== nextNums) {
      numberDraft.value = nextNums
    }
  },
  { immediate: true },
)

function emitComposed(numsRaw: string, attrs: string[], waves: string[]) {
  const next = compose(numsRaw, attrs, waves)
  if (next !== props.modelValue) {
    emit('update:modelValue', next)
  }
}

function toggleOption(opt: string) {
  if (props.disabled) return
  const cur = parseLhcTemaParts(props.modelValue)
  if (isLhcTemaWaveOption(opt)) {
    const waves = cur.waves.includes(opt)
      ? cur.waves.filter((x) => x !== opt)
      : [...cur.waves, opt]
    emitComposed(numberDraft.value, cur.attrs, waves)
    return
  }
  const attrs = cur.attrs.includes(opt)
    ? cur.attrs.filter((x) => x !== opt)
    : [...cur.attrs, opt]
  emitComposed(numberDraft.value, attrs, cur.waves)
}

function onDraftInput(v: string) {
  numberDraft.value = v
  if (props.disabled) return
  const cur = parseLhcTemaParts(props.modelValue)
  emitComposed(v, cur.attrs, cur.waves)
}

function onBlur() {
  if (props.disabled) return
  const next = normalizeLhcTemaContent(
    compose(numberDraft.value, parts.value.attrs, parts.value.waves),
  )
  numberDraft.value = parseLhcTemaParts(next).nums.join(',')
  if (next !== props.modelValue) {
    emit('update:modelValue', next)
  }
}
</script>

<template>
  <div class="slt-panel" :class="{ 'is-disabled': disabled }">
    <div class="slt-chips" role="group" aria-label="特码快捷选项">
      <button
        v-for="opt in LHC_TEMA_QUICK_OPTIONS"
        :key="opt"
        type="button"
        class="slt-chip"
        :class="{ 'is-active': selected.has(opt) }"
        :disabled="disabled"
        @click="toggleOption(opt)"
      >
        {{ opt }}
      </button>
    </div>
    <el-input
      :model-value="numberDraft"
      class="slt-input"
      :disabled="disabled"
      clearable
      placeholder="输入 1–49 号码，多个用逗号分隔（属性请点上方选项，不会写入此框）"
      @update:model-value="onDraftInput(String($event ?? ''))"
      @blur="onBlur"
    />
  </div>
</template>

<style scoped>
.slt-panel {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  width: 100%;
}

.slt-panel.is-disabled {
  opacity: 0.72;
}

.slt-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.85rem 1rem;
  border-radius: 0.75rem;
  background: #f0f3f7;
}

.slt-chip {
  min-height: 2rem;
  padding: 0.25rem 0.75rem;
  border: none;
  border-radius: 0.5rem;
  background: #ffffff;
  color: #1a2332;
  font-family: 'Noto Sans SC', 'Inter', sans-serif;
  font-size: 0.8125rem;
  font-weight: 500;
  line-height: 1.4;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(15, 23, 42, 0.04);
  transition:
    background 0.15s ease,
    color 0.15s ease,
    box-shadow 0.15s ease;
}

.slt-chip:hover:not(:disabled) {
  box-shadow: 0 6px 20px rgba(0, 80, 203, 0.1);
}

.slt-chip.is-active {
  background: #0050cb;
  color: #ffffff;
  box-shadow: 0 6px 18px rgba(0, 80, 203, 0.22);
}

.slt-chip:disabled {
  cursor: not-allowed;
}

.slt-input {
  width: 100%;
}

.slt-input :deep(.el-input__wrapper) {
  min-height: 2.75rem;
  border-radius: 0.75rem;
  box-shadow: 0 4px 16px rgba(15, 23, 42, 0.04);
  background: #ffffff;
}
</style>
