<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import SchemeGroupInputPanel from '@/components/schemes/SchemeGroupInputPanel.vue'
import SchemeGroupPickPanel from '@/components/schemes/SchemeGroupPickPanel.vue'
import {
  SSC_POSITION_LABELS,
  bareConfigForRenxuanPicks,
  buildRenxuanPositionContent,
  defaultRenxuanPositions,
  isRenxuanPositionPoolConfig,
  parseRenxuanPositionContent,
  type PlayConfig,
} from '@/utils/betPayload'
import {
  schemeGroupUsesDigitInput,
  schemeGroupUsesPickPanel,
} from '@/utils/pickPanelOptions'

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

/** 选位上限：万千百十个共 5 位 */
const maxPositions = 5

const isPool = computed(() => isRenxuanPositionPoolConfig(props.config))

/** 剥位后的玩法配置，供号池/和值选号面板使用 */
const bareConfig = computed(() => bareConfigForRenxuanPicks(props.config))
const usesDigitInput = computed(
  () => isPool.value && schemeGroupUsesDigitInput(bareConfig.value),
)
const usesPickPanel = computed(
  () => isPool.value && schemeGroupUsesPickPanel(bareConfig.value),
)

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
      props.config.betMode,
      props.config.playMethodLabel,
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
  } else if (set.size >= maxPositions) {
    return
  } else {
    set.add(lab)
  }
  // 按万千百十个顺序展示
  positions.value = SSC_POSITION_LABELS.filter((p) => set.has(p))
}

const placeholder = computed(() => {
  const k = pickCount.value
  if (isPool.value) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再选择/输入号码；所选位置多于 ${k} 个时按组合计注（C(选位数,${k})×号码注数）。`
  }
  const example = `${'12'.slice(0, digitLen.value).padEnd(digitLen.value, '0')},34`
  return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入 ${digitLen.value} 位号码组成一注；所选位置多于 ${k} 个时按组合计注（C(选位数,${k})×号码注数）。所选位置与号码顺序均须与开奖一致。示例：${example}`
})
</script>

<template>
  <div class="srd-panel" :class="{ 'is-disabled': disabled }">
    <div class="srd-pos-row">
      <span class="srd-pos-label">选位（{{ positions.length }}/{{ maxPositions }}，至少{{ pickCount }}）</span>
      <div
        class="srd-chips"
        role="group"
        :aria-label="`从万千百十个中选至少 ${pickCount} 个、最多 ${maxPositions} 个位置`"
      >
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
    <SchemeGroupInputPanel
      v-if="usesDigitInput"
      v-model="picksText"
      :config="bareConfig"
      :disabled="disabled"
      :rows="6"
    />
    <SchemeGroupPickPanel
      v-else-if="usesPickPanel"
      v-model="picksText"
      :config="bareConfig"
      :disabled="disabled"
    />
    <el-input
      v-else
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
  background: #f7f9fb;
  box-shadow: none;
  font-family: 'Inter', 'Noto Sans SC', sans-serif;
  font-size: 0.875rem;
  line-height: 1.6;
}

.srd-panel.is-disabled {
  opacity: 0.85;
  pointer-events: none;
}
</style>
