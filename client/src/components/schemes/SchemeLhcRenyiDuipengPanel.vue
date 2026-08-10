<script setup lang="ts">
import { ref, watch } from 'vue'
import type { PlayConfig } from '@/utils/betPayload'
import {
  formatLhcRenyiDuipengContent,
  parseLhcNumberTokens,
  parseLhcRenyiDuipengSides,
} from '@/utils/betPayload'

/**
 * 任意对碰方案内容：A区 / B区两个输入框。
 * 落库格式：A区号码|B区号码（区内逗号，两侧不可重复，01–49）。
 */
const props = defineProps<{
  config: PlayConfig
  modelValue: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
}>()

const sideA = ref('')
const sideB = ref('')

function zoneToBox(nums: string[]): string {
  return nums.join(',')
}

function boxToNums(raw: string): string[] {
  return [...new Set(parseLhcNumberTokens(raw))]
}

function syncFromModel(content: string): void {
  const sides = parseLhcRenyiDuipengSides(content)
  sideA.value = zoneToBox(sides?.a ?? [])
  sideB.value = zoneToBox(sides?.b ?? [])
}

function emitCombined(): void {
  if (props.disabled) return
  const a = boxToNums(sideA.value)
  const b = boxToNums(sideB.value)
  sideA.value = zoneToBox(a)
  sideB.value = zoneToBox(b)
  emit('update:modelValue', formatLhcRenyiDuipengContent(a, b))
}

function onZoneInput(which: 'a' | 'b', value: string): void {
  if (props.disabled) return
  // 编辑中允许临时非法字符；失焦再规范化
  if (which === 'a') sideA.value = value
  else sideB.value = value
  const a = boxToNums(sideA.value)
  const b = boxToNums(sideB.value)
  emit('update:modelValue', formatLhcRenyiDuipengContent(a, b))
}

watch(
  () =>
    [
      props.config.betMode,
      props.config.playTypeId,
      props.config.subPlayId,
      props.config.playMethodLabel,
    ] as const,
  () => syncFromModel(props.modelValue),
  { immediate: true },
)

watch(
  () => props.modelValue,
  (value) => {
    const next = String(value ?? '')
    const cur = formatLhcRenyiDuipengContent(boxToNums(sideA.value), boxToNums(sideB.value))
    if (cur !== next) syncFromModel(next)
  },
)
</script>

<template>
  <div class="srd-panel" :class="{ 'is-disabled': disabled }">
    <p class="srd-hint">A区、B区分别输入 1–49 号码（区内逗号分隔），两侧号码不可重复；保存格式为 A|B</p>
    <div class="srd-zones">
      <div class="srd-zone">
        <label class="srd-lbl" for="srd-a">A区</label>
        <el-input
          id="srd-a"
          :model-value="sideA"
          :disabled="disabled"
          clearable
          placeholder="如 1,13,25"
          class="srd-inp"
          @update:model-value="(v) => onZoneInput('a', String(v ?? ''))"
          @blur="emitCombined"
        />
      </div>
      <div class="srd-sep" aria-hidden="true">|</div>
      <div class="srd-zone">
        <label class="srd-lbl" for="srd-b">B区</label>
        <el-input
          id="srd-b"
          :model-value="sideB"
          :disabled="disabled"
          clearable
          placeholder="如 2,14,26"
          class="srd-inp"
          @update:model-value="(v) => onZoneInput('b', String(v ?? ''))"
          @blur="emitCombined"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.srd-panel {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.srd-hint {
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: #64748b;
  font-family: 'Noto Sans SC', sans-serif;
}

.srd-zones {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  gap: 0.5rem;
  align-items: end;
}

.srd-zone {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 0;
}

.srd-lbl {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #334155;
  font-family: 'Noto Sans SC', sans-serif;
}

.srd-sep {
  padding-bottom: 0.55rem;
  font-size: 1.125rem;
  font-weight: 700;
  color: #0050cb;
  font-family: 'Plus Jakarta Sans', 'Inter', sans-serif;
  line-height: 1;
}

.srd-inp :deep(.el-input__wrapper) {
  min-height: 2.5rem;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  box-shadow: none;
}

.srd-inp :deep(.el-input__wrapper:hover),
.srd-inp :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.srd-inp :deep(.el-input__inner) {
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 0.9375rem;
}

.srd-panel.is-disabled {
  opacity: 0.85;
}

@media (max-width: 480px) {
  .srd-zones {
    grid-template-columns: 1fr;
  }

  .srd-sep {
    display: none;
  }
}
</style>
