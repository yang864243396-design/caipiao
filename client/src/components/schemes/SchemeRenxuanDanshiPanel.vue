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
  isHunhePlayConfig,
  isSixingZu6PlayConfig,
  isZu3DanshiConfig,
  isZu6DanshiConfig,
  isZuxuanDanshiConfig,
  parseRenxuanPositionContent,
  type PlayConfig,
} from '@/utils/betPayload'
import {
  commitSchemeGroupContentOnBlur,
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
  // 选位不足 k 时不落库，避免「千\n12」被 parse 当成整段号码
  if (positions.value.length < pickCount.value) return
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

function onPicksBlur(): void {
  if (props.disabled) return
  // 用完整玩法配置做组三/组六形态过滤（bareConfig 会清 playType，易误判号池）
  picksText.value = commitSchemeGroupContentOnBlur(picksText.value, props.config)
}

function togglePosition(lab: string) {
  if (props.disabled) return
  const set = new Set(positions.value)
  if (set.has(lab)) {
    // 至少保留 k 个；再点已选位不得取消，否则会发出非法内容被 parse 整段塞进号码框
    if (set.size <= pickCount.value) return
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
  const bm = (props.config.betMode ?? '').trim()
  const method = `${props.config.playMethodLabel ?? ''} ${props.config.subPlayId ?? ''}`
  // 任四组选24：选位 + 至少 4 个单码号池
  if (isPool.value && (bm === 'zu24' || /组选24|zu24/i.test(method))) {
    return `从万、千、百、十、个中勾选至少 ${k} 个，再输入4个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7`
  }
  // 任四组选12：选位 + 二重号/单号双区
  if (isPool.value && (bm === 'zu12' || (/组选12|zu12/i.test(method) && !/组选120|zu120/i.test(method)))) {
    return `从万、千、百、十、个中勾选至少 ${k} 个，再从0-9中，输入1个及以上的二重号码，2个及以上的单号，两个位置由逗号分隔，如：12,3234`
  }
  // 任四组选6：选位 + 至少 2 个 0-9 号码（C(n,2)）
  if (isPool.value && isSixingZu6PlayConfig(props.config)) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入两个及以上的0-9的号码，多选用逗号分隔，如1,2`
  }
  if (isPool.value) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再选择/输入号码；所选位置多于 ${k} 个时按组合计注（C(选位数,${k})×号码注数）。`
  }
  // 任三组三单式：须两同号 + 一异号（勿用 012 这类组六形态示例）
  if (isZu3DanshiConfig(props.config)) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入两个号相同号码和一个不同号码组成一注。所选位置与号码须与开奖一致，顺序不限。示例：112,223`
  }
  // 任三组六单式：须三位互不相同
  if (isZu6DanshiConfig(props.config)) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入三个各不相同的3个号码组成一注。所选位置与号码须与开奖一致，顺序不限。示例：012,345`
  }
  // 任三混合组选：三位一注，选位开奖与输入号码一致、顺序不限（组选）
  if (isHunhePlayConfig(props.config)) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入三个号码组成一注，所选${k}个位置的开奖号码与输入号码一致，顺序不限。示例：012,345`
  }
  const example = Array.from({ length: 2 }, (_, ti) =>
    Array.from({ length: digitLen.value }, (_, i) => String((i + ti * 3) % 10)).join(''),
  ).join(',')
  if (isZuxuanDanshiConfig(props.config)) {
    return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入 ${digitLen.value} 位号码组成一注。所选位置与号码须与开奖一致，顺序不限。示例：${example}`
  }
  return `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置，再输入 ${digitLen.value} 位号码组成一注。所选位置与号码顺序均须与开奖一致。示例：${example}`
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
      :placeholder="placeholder"
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
      @blur="onPicksBlur"
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
