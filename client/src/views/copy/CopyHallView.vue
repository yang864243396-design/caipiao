<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import LobbyTabBar from '@/components/lobby/LobbyTabBar.vue'
import OptionPickerModal from '@/components/ui/OptionPickerModal.vue'
import {
  startCopyHallRankingsSync,
  stopCopyHallRankingsSync,
  useCopyHallRankings,
} from '@/composables/useCopyHallRankings'
import { useCopyHallPlayFilter } from '@/composables/useCopyHallPlayFilter'
import { shareSnapshotToRankSlot, useCopyHallShareSchemes } from '@/composables/useCopyHallShareSchemes'
import type { CopyHallBoardKind, CopyHallRankSlot } from '@shared/mock/copyHallRankings'

/** 工具栏 / 列表图标 */
const ICON_BACK = '/images/lobby/icon-back.png'
const ICON_SEARCH = '/images/lobby/icon-search.png'
const ICON_SCHEME = '/images/lobby/icon-scheme.png'
const ICON_FILTER = '/images/lobby/icon-filter.png'
const ICON_CHEVRON = '/images/lobby/icon-chevron-down.png'

const tab = ref<CopyHallBoardKind>('master')

const selectedLottery = ref('')
const lotteryDialogVisible = ref(false)

const { lotteryOptions, activeSlots, activeLotteryCode } = useCopyHallRankings(
  () => selectedLottery.value,
  () => tab.value,
)

const {
  selectedPlayTypeId,
  playFilterOptions,
  selectedPlayLabel,
  playTree,
  resetPlayFilter,
} = useCopyHallPlayFilter(
  () => activeLotteryCode.value,
  () => [],
)

const { filteredSchemes: filteredShareSchemes } = useCopyHallShareSchemes(
  () => activeLotteryCode.value,
  () => selectedPlayTypeId.value,
  () => playTree.value,
)

watch(tab, () => {
  resetPlayFilter()
})

const playDialogVisible = ref(false)

const playPickerOptions = computed(() => playFilterOptions.value)

function openPlayDialog() {
  playDialogVisible.value = true
}

function onPlayPickerConfirm(val: string | number) {
  selectedPlayTypeId.value = String(val)
}

watch(lotteryOptions, (opts) => {
  if (!selectedLottery.value && opts.length) {
    selectedLottery.value = opts[0]
  }
}, { immediate: true })

const lotteryPickerOptions = computed(() =>
  lotteryOptions.value.map((name) => ({ label: name, value: name })),
)

function openLotteryDialog() {
  lotteryDialogVisible.value = true
}

function onLotteryPickerConfirm(val: string | number) {
  selectedLottery.value = String(val)
}

const MEDAL_COLORS = [
  'yellow',
  'slate',
  'orange',
  'blue',
  'emerald',
  'blue',
  'emerald',
  'teal',
  'cyan',
  'emerald',
] as const

const GRAD_CLASSES = [
  'g1',
  'g2',
  'g3',
  'g4',
  'g5',
  'g6',
  'g7',
  'g8',
  'g9',
  'g10',
] as const

const topRanks = computed(() =>
  activeSlots.value.map((slot) => ({
    slot,
    rank: slot.rank,
    medal: MEDAL_COLORS[slot.rank - 1] ?? 'blue',
    name: slot.schemeName,
    iconImg: ICON_SCHEME,
  })),
)

const schemeCards = computed(() =>
  filteredShareSchemes.value.map((item, i) => ({
    slot: shareSnapshotToRankSlot(item),
    iconImg: ICON_SCHEME,
    grad: GRAD_CLASSES[i % GRAD_CLASSES.length] ?? 'g1',
    name: item.schemeName,
  })),
)

function gameDetailQuery(slot: CopyHallRankSlot) {
  const q: Record<string, string> = {
    scheme: `${slot.schemeName} - ${slot.playMethod}`,
    snapshotId: slot.schemeId,
    lotteryCode: activeLotteryCode.value,
    playMethod: slot.playMethod,
    board: tab.value,
  }
  if (slot.playTypeId) {
    q.typeId = slot.playTypeId
    q.playTypeId = slot.playTypeId
  }
  if (slot.subPlayId) {
    q.subId = slot.subPlayId
    q.subPlayId = slot.subPlayId
  }
  return q
}

onMounted(startCopyHallRankingsSync)
onUnmounted(stopCopyHallRankingsSync)
</script>

<template>
  <div class="copy-hall">
    <header class="topbar">
      <div class="topbar-inner">
        <RouterLink to="/" class="icon-btn topbar-side" aria-label="返回">
          <img
            :src="ICON_BACK"
            alt=""
            width="24"
            height="24"
            class="toolbar-ico"
            decoding="async"
          />
        </RouterLink>
        <div class="topbar-center">
          <button
            type="button"
            class="lottery-trigger"
            aria-haspopup="dialog"
            :aria-expanded="lotteryDialogVisible"
            aria-controls="copy-hall-lottery-dialog"
            :aria-label="`选择彩种，当前为 ${selectedLottery}`"
            @click="openLotteryDialog"
          >
            <span class="lottery-trigger-name">{{ selectedLottery }}</span>
            <svg class="lottery-trigger-chev" viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
              <path fill="currentColor" d="M7 10l5 5 5-5z" />
            </svg>
          </button>
        </div>
        <button type="button" class="icon-btn topbar-side" aria-label="搜索">
          <img
            :src="ICON_SEARCH"
            alt=""
            width="24"
            height="24"
            class="toolbar-ico"
            decoding="async"
          />
        </button>
      </div>
    </header>

    <OptionPickerModal
      v-model="lotteryDialogVisible"
      panel-id="copy-hall-lottery-dialog"
      :selected-value="selectedLottery"
      title="选择彩种"
      :options="lotteryPickerOptions"
      selection-accent="primary"
      :show-header-divider="true"
      :show-footer-divider="true"
      :columns="2"
      @confirm="onLotteryPickerConfirm"
    />

    <OptionPickerModal
      v-model="playDialogVisible"
      panel-id="copy-hall-play-dialog"
      :selected-value="selectedPlayTypeId"
      title="选择玩法"
      :options="playPickerOptions"
      selection-accent="primary"
      :show-header-divider="true"
      :show-footer-divider="true"
      :columns="2"
      @confirm="onPlayPickerConfirm"
    />

    <main class="main">
      <!-- Segmented -->
      <el-radio-group v-model="tab" size="default" class="seg-ep">
        <el-radio-button value="master">大神榜</el-radio-button>
        <el-radio-button value="contrary">反买榜</el-radio-button>
      </el-radio-group>

      <!-- 空态占位 -->
      <section v-if="!topRanks.length" class="rank-card hall-empty-card">
        <el-empty
          class="hall-empty"
          :image-size="120"
          :description="tab === 'master' ? '大神榜暂无上榜方案' : '反买榜暂无上榜方案'"
        >
          <template #description>
            <p class="hall-empty-title">{{ tab === 'master' ? '大神榜暂无上榜方案' : '反买榜暂无上榜方案' }}</p>
            <p class="hall-empty-hint">榜单按方案战绩实时生成，稍后再来看看吧</p>
          </template>
        </el-empty>
      </section>

      <!-- Top 10 grid -->
      <section v-else class="rank-card">
        <div class="rank-grid">
          <RouterLink
            v-for="r in topRanks"
            :key="r.rank"
            class="rank-cell"
            :to="{
              path: '/play/detail',
              query: gameDetailQuery(r.slot),
            }"
          >
            <div class="medal-wrap">
              <img
                :src="r.iconImg"
                alt=""
                width="36"
                height="36"
                class="medal-img"
                decoding="async"
              />
              <span class="rank-num" :class="{ small: r.rank >= 10 }">{{ r.rank }}</span>
            </div>
            <span class="rank-name">{{ r.name }}</span>
          </RouterLink>
        </div>
      </section>

      <!-- Filter -->
      <div v-if="playFilterOptions.length" class="filter-bar">
        <div class="filter-left">
          <div class="filter-ico">
            <img
              :src="ICON_FILTER"
              alt=""
              width="18"
              height="18"
              class="filter-ico-img"
              decoding="async"
            />
          </div>
          <span class="filter-lbl">玩法筛选</span>
        </div>
        <button
          type="button"
          class="filter-chip"
          aria-haspopup="dialog"
          :aria-expanded="playDialogVisible"
          aria-controls="copy-hall-play-dialog"
          :aria-label="`选择玩法，当前为 ${selectedPlayLabel}`"
          @click="openPlayDialog"
        >
          <span>{{ selectedPlayLabel }}</span>
          <img
            :src="ICON_CHEVRON"
            alt=""
            width="14"
            height="14"
            class="chip-chev-img"
            decoding="async"
          />
        </button>
      </div>

      <!-- Scheme grid -->
      <div v-if="schemeCards.length" class="scheme-grid">
        <RouterLink
          v-for="c in schemeCards"
          :key="c.slot.schemeId"
          class="scheme-item"
          :to="{
            path: '/play/detail',
            query: gameDetailQuery(c.slot),
          }"
        >
          <div class="scheme-grad" :class="c.grad">
            <img
              :src="c.iconImg"
              alt=""
              width="16"
              height="16"
              class="scheme-ico-img"
              decoding="async"
            />
          </div>
          <h3 class="scheme-name">{{ c.name }}</h3>
        </RouterLink>
      </div>
      <p
        v-else-if="playFilterOptions.length && !schemeCards.length"
        class="scheme-filter-empty"
      >
        当前玩法暂无方案
      </p>
    </main>

    <LobbyTabBar />
  </div>
</template>

<style scoped>
.copy-hall {
  --surface: #f7f9fb;
  --on-surface: #191c1e;
  --on-surface-variant: #424656;
  --primary: #0050cb;
  --primary-container: #0066ff;
  --secondary: #425ca0;
  --secondary-container: #9bb4fe;
  --tertiary: #a33200;
  --tertiary-container: #cc4204;
  --surface-low: #f2f4f6;
  --surface-lowest: #ffffff;
  --outline-variant: #c2c6d8;
  min-height: 100dvh;
  background: var(--surface);
  color: var(--on-surface);
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: calc(5.5rem + env(safe-area-inset-bottom));
  padding-top: env(safe-area-inset-top);
}
.material {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.5rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  vertical-align: middle;
  display: inline-block;
  color: #64748b;
}
.material.m-fill {
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}
.topbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 50;
  background: rgba(247, 249, 251, 0.8);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}
.topbar-inner {
  display: grid;
  grid-template-columns: 2.5rem 1fr 2.5rem;
  align-items: center;
  gap: 0.75rem;
  height: 4rem;
  padding: 0 1rem;
  max-width: 42rem;
  margin: 0 auto;
  box-sizing: border-box;
}
.topbar-side {
  justify-self: center;
}
.topbar-center {
  display: flex;
  justify-content: center;
  align-items: center;
  min-width: 0;
}
.lottery-trigger {
  display: inline-flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 0.125rem;
  max-width: 100%;
  margin: 0;
  padding: 0.25rem 0.5rem;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  cursor: pointer;
  font: inherit;
  color: inherit;
  -webkit-tap-highlight-color: transparent;
}
.lottery-trigger:focus-visible {
  outline: 2px solid var(--primary-container);
  outline-offset: 2px;
}
.lottery-trigger-name {
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--on-surface);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 14rem;
}
.lottery-trigger-chev {
  flex-shrink: 0;
  color: var(--secondary);
  opacity: 0.85;
}
.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 999px;
  border: none;
  background: transparent;
  cursor: pointer;
  text-decoration: none;
  transition:
    background 0.15s,
    transform 0.2s;
}
.icon-btn:hover {
  background: #f1f5f9;
}
.icon-btn:active {
  transform: scale(0.95);
}
.toolbar-ico {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
  display: block;
  pointer-events: none;
}
.main {
  padding: 5rem 1rem 2rem;
  max-width: 42rem;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.seg-ep {
  display: flex;
  width: 100%;
  padding: 0.375rem;
  background: var(--surface-low);
  border-radius: 0.75rem;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  box-sizing: border-box;
}

.seg-ep :deep(.el-radio-button) {
  flex: 1;
}

.seg-ep :deep(.el-radio-button__inner) {
  width: 100%;
  border-radius: 0.5rem;
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: none;
  background: transparent;
  color: var(--on-surface-variant);
  box-shadow: none;
}

.seg-ep :deep(.el-radio-button.is-active .el-radio-button__inner) {
  background: var(--surface-lowest);
  color: var(--primary);
  font-weight: 600;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.rank-card {
  background: var(--surface-lowest);
  border-radius: 1rem;
  padding: 1rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(194, 198, 216, 0.1);
}
.hall-empty-card {
  padding: 2rem 1rem 2.5rem;
}
.hall-empty {
  --el-empty-padding: 0;
}
.hall-empty-title {
  margin: 0 0 0.375rem;
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--on-surface);
}
.hall-empty-hint {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--on-surface-variant);
}
.rank-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 1.5rem 0.5rem;
}
.rank-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-decoration: none;
  color: inherit;
  cursor: pointer;
  border-radius: 0.5rem;
  padding: 0.25rem;
  transition: transform 0.15s, background 0.15s;
}
.rank-cell:hover {
  background: rgba(0, 80, 203, 0.06);
}
.rank-cell:active {
  transform: scale(0.97);
}
.medal-wrap {
  position: relative;
  width: 3rem;
  height: 3rem;
  margin-bottom: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}
.medal-img {
  width: 2.25rem;
  height: 2.25rem;
  object-fit: contain;
  display: block;
}
.rank-num {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 700;
  font-size: 0.875rem;
  color: #fff;
  margin-top: -4px;
  pointer-events: none;
}
.rank-num.small {
  font-size: 0.75rem;
}
.rank-name {
  font-size: 11px;
  font-weight: 500;
  text-align: center;
  line-height: 1.25;
  color: var(--on-surface);
  max-width: 100%;
}
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1.25rem;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(8px);
  border-radius: 1rem;
  border: 1px solid rgba(194, 198, 216, 0.2);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}
.filter-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.filter-ico {
  width: 2rem;
  height: 2rem;
  border-radius: 0.5rem;
  background: rgba(0, 80, 203, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
}
.filter-ico-img {
  width: 1.125rem;
  height: 1.125rem;
  object-fit: contain;
  display: block;
}
.filter-lbl {
  font-size: 0.875rem;
  font-weight: 600;
}
.filter-chip {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 1rem;
  border: none;
  border-radius: 999px;
  background: var(--surface-low);
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--on-surface-variant);
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s;
}
.filter-chip:hover {
  background: #e6e8ea;
}
.chip-chev-img {
  width: 0.875rem;
  height: 0.875rem;
  object-fit: contain;
  display: block;
  flex-shrink: 0;
}
.scheme-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}
.scheme-item {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem;
  border-radius: 0.75rem;
  background: var(--surface-lowest);
  border: 1px solid rgba(194, 198, 216, 0.1);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  cursor: pointer;
  text-align: left;
  text-decoration: none;
  color: inherit;
  transition:
    box-shadow 0.3s,
    transform 0.15s;
  font-family: inherit;
}
.scheme-item:hover {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}
.scheme-item:active {
  transform: scale(0.95);
}
.scheme-grad {
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 0.375rem;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}
.scheme-grad .scheme-ico-img {
  width: 1rem;
  height: 1rem;
  object-fit: contain;
  display: block;
  filter: brightness(0) invert(1);
}
.g1 {
  background: linear-gradient(135deg, #0050cb, #0066ff);
}
.g2 {
  background: linear-gradient(135deg, #425ca0, #9bb4fe);
}
.g3 {
  background: linear-gradient(135deg, #a33200, #cc4204);
}
.g4 {
  background: linear-gradient(135deg, #94a3b8, #475569);
}
.g5 {
  background: linear-gradient(135deg, #0066ff, #0050cb);
}
.g6 {
  background: linear-gradient(135deg, #fb923c, #ea580c);
}
.g7 {
  background: linear-gradient(135deg, #06b6d4, #0e7490);
}
.g8 {
  background: linear-gradient(135deg, #6366f1, #4338ca);
}
.g9 {
  background: linear-gradient(135deg, #10b981, #047857);
}
.g10 {
  background: linear-gradient(135deg, #f43f5e, #be123c);
}
.scheme-name {
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--on-surface);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.scheme-filter-empty {
  margin: 0;
  padding: 1.25rem 1rem;
  text-align: center;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--on-surface-variant);
  background: var(--surface-lowest);
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.1);
}
</style>
