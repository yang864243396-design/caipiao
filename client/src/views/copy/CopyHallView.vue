<script setup lang="ts">
import { computed, ref } from 'vue'
import LobbyTabBar from '@/components/lobby/LobbyTabBar.vue'
import OptionPickerModal from '@/components/ui/OptionPickerModal.vue'

/** 工具栏 / 筛选用占位 PNG */
const ICON_PLACEHOLDER = '/images/lobby/icon-placeholder.png'

const tab = ref<'master' | 'contrary'>('master')

/** 顶部彩种（演示列表，接口接入后可替换） */
const lotteryOptions = [
  '腾讯分分彩',
  '重庆时时彩',
  '新疆时时彩',
  '天津时时彩',
  '福彩3D',
  '排列三',
] as const

const selectedLottery = ref<(typeof lotteryOptions)[number]>('腾讯分分彩')
const lotteryDialogVisible = ref(false)

const lotteryPickerOptions = computed(() =>
  lotteryOptions.map((name) => ({ label: name, value: name }))
)

function openLotteryDialog() {
  lotteryDialogVisible.value = true
}

function onLotteryPickerConfirm(val: string | number) {
  selectedLottery.value = val as (typeof lotteryOptions)[number]
}

const topRanks = [
  { rank: 1, medal: 'yellow', name: '太乙后二', iconImg: ICON_PLACEHOLDER },
  { rank: 2, medal: 'slate', name: '紫燕万位', iconImg: ICON_PLACEHOLDER },
  { rank: 3, medal: 'orange', name: '莺凤十位', iconImg: ICON_PLACEHOLDER },
  { rank: 4, medal: 'blue', name: '宛天个位', iconImg: ICON_PLACEHOLDER },
  { rank: 5, medal: 'emerald', name: '路线6000+', iconImg: ICON_PLACEHOLDER },
  { rank: 6, medal: 'blue', name: '打狗前二', iconImg: ICON_PLACEHOLDER },
  { rank: 7, medal: 'emerald', name: '邯肖任四', iconImg: ICON_PLACEHOLDER },
  { rank: 8, medal: 'teal', name: '关冲70+', iconImg: ICON_PLACEHOLDER },
  { rank: 9, medal: 'cyan', name: '猎豹后二', iconImg: ICON_PLACEHOLDER },
  { rank: 10, medal: 'emerald', name: '青衫万位', iconImg: ICON_PLACEHOLDER },
] as const

const schemeCards = [
  { iconImg: ICON_PLACEHOLDER, grad: 'g1', name: '禄螭万位' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g2', name: '月华万位' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g3', name: '青鸾后二' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g4', name: '重明千位' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g5', name: '麒麟个位' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g6', name: '白泽前三' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g7', name: '玄武后一' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g8', name: '朱雀任二' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g9', name: '青衫万位' },
  { iconImg: ICON_PLACEHOLDER, grad: 'g10', name: '猎豹后二' },
] as const
</script>

<template>
  <div class="copy-hall">
    <header class="topbar">
      <div class="topbar-inner">
        <RouterLink to="/" class="icon-btn topbar-side" aria-label="返回">
          <img
            :src="ICON_PLACEHOLDER"
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
            :src="ICON_PLACEHOLDER"
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

    <main class="main">
      <!-- Segmented -->
      <el-radio-group v-model="tab" size="default" class="seg-ep">
        <el-radio-button value="master">大神榜</el-radio-button>
        <el-radio-button value="contrary">反买榜</el-radio-button>
      </el-radio-group>

      <!-- Top 10 grid -->
      <section class="rank-card">
        <div class="rank-grid">
          <div v-for="r in topRanks" :key="r.rank" class="rank-cell">
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
          </div>
        </div>
      </section>

      <!-- Filter -->
      <div class="filter-bar">
        <div class="filter-left">
          <div class="filter-ico">
            <img
              :src="ICON_PLACEHOLDER"
              alt=""
              width="18"
              height="18"
              class="filter-ico-img"
              decoding="async"
            />
          </div>
          <span class="filter-lbl">玩法筛选</span>
        </div>
        <button type="button" class="filter-chip">
          <span>定位胆</span>
          <img
            :src="ICON_PLACEHOLDER"
            alt=""
            width="14"
            height="14"
            class="chip-chev-img"
            decoding="async"
          />
        </button>
      </div>

      <!-- Scheme grid -->
      <div class="scheme-grid">
        <RouterLink
          v-for="c in schemeCards"
          :key="c.name"
          class="scheme-item"
          :to="{
            path: '/play/detail',
            query: { scheme: `${c.name} - 定位胆万位` },
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
.rank-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 1.5rem 0.5rem;
}
.rank-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
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
</style>
