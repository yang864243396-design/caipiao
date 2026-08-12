<script setup lang="ts">
import { computed } from 'vue'
import {
  LHC_GUOGUAN_OPTIONS,
  LHC_GUOGUAN_POSITION_LABELS,
  parseLhcGuoguanPositions,
} from '@/constants/lhcPlay'
import type { PlayConfig } from '@/utils/betPayload'

const props = defineProps<{
  config: PlayConfig
  modelValue: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const positions = computed(() => parseLhcGuoguanPositions(props.modelValue) ?? Array(6).fill(''))

function choose(positionIndex: number, pick: string): void {
  if (props.disabled) return
  const next = [...positions.value]
  next[positionIndex] = pick
  emit('update:modelValue', next.join(','))
}
</script>

<template>
  <div class="sgg-panel" :class="{ 'is-disabled': disabled }">
    <div class="sgg-positions">
      <section v-for="(label, positionIndex) in LHC_GUOGUAN_POSITION_LABELS" :key="label" class="sgg-position">
        <span class="sgg-label">{{ label }}</span>
        <select
          class="sgg-select"
          :value="positions[positionIndex]"
          :disabled="disabled"
          :aria-label="`${label}选项`"
          @change="choose(positionIndex, ($event.target as HTMLSelectElement).value)"
        >
          <option value="">请选择</option>
          <option v-for="pick in LHC_GUOGUAN_OPTIONS" :key="pick" :value="pick">{{ pick }}</option>
        </select>
      </section>
    </div>
  </div>
</template>

<style scoped>
.sgg-panel {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  width: 100%;
}

.sgg-positions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
}

.sgg-position {
  display: flex;
  flex-direction: row;
  gap: 0.5rem;
  align-items: center;
  min-width: 0;
}

.sgg-label {
  flex: 0 0 auto;
  color: #334155;
  font-family: 'Noto Sans SC', sans-serif;
  font-size: 0.8125rem;
  font-weight: 600;
}

.sgg-select {
  min-width: 0;
  flex: 1 1 auto;
  width: 100%;
  height: 2.25rem;
  padding: 0 0.625rem;
  border: 1px solid #dbe3ee;
  border-radius: 0.5rem;
  background: #fff;
  color: #334155;
  font-family: 'Noto Sans SC', sans-serif;
  font-size: 0.875rem;
}

.sgg-select:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 2px rgb(37 99 235 / 12%);
}

.sgg-select:disabled {
  cursor: default;
  color: #64748b;
  opacity: 1;
}

</style>
