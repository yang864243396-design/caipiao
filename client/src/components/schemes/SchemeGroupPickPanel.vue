<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  LHC_NUMBERS,
  LHC_TAIL_OPTIONS,
  LHC_ZODIACS,
  lhcAttrOptions,
} from '@/constants/lhcPlay'
import {
  buildGroupContent,
  parseGroupPicks,
  type PlayConfig,
} from '@/utils/betPayload'
import {
  digitOptionsForConfig,
  poolMaxPicksForConfig,
  schemeGroupUsesPickPanel,
  textPickOptionsForConfig,
  togglePoolPick,
  useCompactPickChips,
} from '@/utils/pickPanelOptions'

const props = defineProps<{
  config: PlayConfig
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const digitOptions = computed(() => digitOptionsForConfig(props.config))
const textPickOptions = computed(() => textPickOptionsForConfig(props.config))
const compactChips = computed(() => useCompactPickChips(props.config))

/** 0–9 / 1–10 等小号池：号码按钮保持同一行 */
const singleRowChips = computed(() => {
  if (compactChips.value) return false
  if (textPickOptions.value.length > 0) return textPickOptions.value.length <= 6
  const n = digitOptions.value.length
  return n > 0 && n <= 10
})

const showPanel = computed(() => schemeGroupUsesPickPanel(props.config))

const pickDigits = ref<string[]>([])
const pickLines = ref<string[][]>([])

const lhcPickOptions = computed((): readonly string[] => {
  const cfg = props.config
  if (cfg.inputMode === 'lhc_zodiac') return LHC_ZODIACS
  if (cfg.inputMode === 'lhc_tail') return LHC_TAIL_OPTIONS
  if (cfg.inputMode === 'lhc_attr') {
    return lhcAttrOptions(cfg.betMode ?? '', 'lhc_attr')
  }
  if (cfg.inputMode === 'lhc_num') return LHC_NUMBERS
  return []
})

function syncFromModel(content: string) {
  // 单式等走 textarea 时不挂载选号态，避免把「123」解析成个位号池再回写清空
  if (!showPanel.value) {
    pickDigits.value = []
    pickLines.value = []
    return
  }
  const parsed = parseGroupPicks(props.config, content)
  const max = poolMaxPicksForConfig(props.config)
  pickDigits.value =
    max != null && max > 0 && parsed.digits.length > max
      ? parsed.digits.slice(0, max)
      : parsed.digits
  if (props.config.inputMode === 'multiline') {
    pickLines.value = parsed.lines
  }
}

function emitContent() {
  // 未展示选号面板时禁止回写，否则会覆盖同绑定的 textarea（直选单式等）
  if (!showPanel.value) return
  const next = buildGroupContent(props.config, {
    digits: pickDigits.value,
    lines: pickLines.value,
  })
  if (next !== props.modelValue) {
    emit('update:modelValue', next)
  }
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
      props.config.poolMaxPicks,
      showPanel.value,
    ] as const,
  () => syncFromModel(props.modelValue),
  { immediate: true },
)

watch(
  () => props.modelValue,
  (value) => {
    if (!showPanel.value) return
    const rebuilt = buildGroupContent(props.config, {
      digits: pickDigits.value,
      lines: pickLines.value,
    })
    if (value.trim() !== rebuilt.trim()) {
      syncFromModel(value)
    }
  },
)

watch([pickDigits, pickLines], emitContent, { deep: true })

function togglePickDigit(d: string) {
  pickDigits.value = togglePoolPick(pickDigits.value, d, poolMaxPicksForConfig(props.config))
}

function toggleLineDigit(lineIndex: number, d: string) {
  const lines = pickLines.value.map((line) => [...line])
  while (lines.length < props.config.segmentLen) {
    lines.push([])
  }
  const line = new Set(lines[lineIndex] ?? [])
  if (line.has(d)) line.delete(d)
  else line.add(d)
  lines[lineIndex] = [...line].sort()
  pickLines.value = lines
}

function isLineDigitSelected(lineIndex: number, d: string) {
  return (pickLines.value[lineIndex] ?? []).includes(d)
}
</script>

<template>
  <div v-if="showPanel" class="sgp-panel">
    <template v-if="textPickOptions.length && config.inputMode === 'multiline'">
      <div v-for="(label, li) in config.segmentLabels" :key="label" class="sgp-row">
        <span class="sgp-pos">{{ label }}</span>
        <div class="sgp-chips" :class="{ 'sgp-chips--single-row': singleRowChips }">
          <button
            v-for="d in textPickOptions"
            :key="`${label}-${d}`"
            type="button"
            class="sgp-chip"
            :class="{ 'is-active': isLineDigitSelected(li, d) }"
            @click="toggleLineDigit(li, d)"
          >
            {{ d }}
          </button>
        </div>
      </div>
    </template>
    <template v-else-if="textPickOptions.length">
      <div class="sgp-chips" :class="{ 'sgp-chips--single-row': singleRowChips }">
        <button
          v-for="d in textPickOptions"
          :key="d"
          type="button"
          class="sgp-chip"
          :class="{ 'is-active': pickDigits.includes(d) }"
          @click="togglePickDigit(d)"
        >
          {{ d }}
        </button>
      </div>
    </template>
    <template
      v-else-if="
        config.inputMode === 'lhc_num' ||
        config.inputMode === 'lhc_zodiac' ||
        config.inputMode === 'lhc_tail' ||
        config.inputMode === 'lhc_attr'
      "
    >
      <div class="sgp-chips sgp-chips--lhc">
        <button
          v-for="d in lhcPickOptions"
          :key="d"
          type="button"
          class="sgp-chip sgp-chip--lhc"
          :class="{ 'is-active': pickDigits.includes(d) }"
          @click="togglePickDigit(d)"
        >
          {{ d }}
        </button>
      </div>
    </template>
    <template v-else-if="config.inputMode === 'multiline'">
      <div v-for="(label, li) in config.segmentLabels" :key="label" class="sgp-row">
        <span class="sgp-pos">{{ label }}</span>
        <div
          class="sgp-chips"
          :class="{ 'sgp-chips--lhc': compactChips, 'sgp-chips--single-row': singleRowChips }"
        >
          <button
            v-for="d in digitOptions"
            :key="`${label}-${d}`"
            type="button"
            class="sgp-chip"
            :class="{ 'sgp-chip--lhc': compactChips, 'is-active': isLineDigitSelected(li, d) }"
            @click="toggleLineDigit(li, d)"
          >
            {{ d }}
          </button>
        </div>
      </div>
    </template>
    <template v-else>
      <div
        class="sgp-chips"
        :class="{ 'sgp-chips--lhc': compactChips, 'sgp-chips--single-row': singleRowChips }"
      >
        <button
          v-for="d in digitOptions"
          :key="d"
          type="button"
          class="sgp-chip"
          :class="{ 'sgp-chip--lhc': compactChips, 'is-active': pickDigits.includes(d) }"
          @click="togglePickDigit(d)"
        >
          {{ d }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.sgp-panel {
  margin-bottom: 0.75rem;
}

.sgp-row {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.sgp-row .sgp-chips {
  flex: 1;
  min-width: 0;
}

.sgp-pos {
  flex: 0 0 auto;
  min-width: 2rem;
  font-size: 12px;
  color: var(--el-text-color-regular);
  line-height: 2rem;
  text-align: center;
  white-space: nowrap;
}

.sgp-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.sgp-chips--lhc {
  max-height: 10rem;
  overflow-y: auto;
}

/* 0–9 / 1–10：单行等分，避免换行 */
.sgp-chips--single-row {
  flex-wrap: nowrap;
  gap: 0.25rem;
}

.sgp-chips--single-row .sgp-chip {
  flex: 1 1 0;
  min-width: 0;
  padding: 0;
}

.sgp-chip {
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.35rem;
  border: 1px solid rgb(148 163 184 / 35%);
  border-radius: 0.5rem;
  background: #fff;
  color: var(--el-text-color-primary);
  font-size: 13px;
  cursor: pointer;
}

.sgp-chip--lhc {
  min-width: 2.25rem;
  font-size: 12px;
}

.sgp-chip.is-active {
  border-color: var(--el-color-primary);
  background: rgb(0 102 255 / 8%);
  color: var(--el-color-primary);
  font-weight: 600;
}
</style>
