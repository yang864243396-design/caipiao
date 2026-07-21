<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { fetchChaseOrders, toChaseDisplayRow } from '@/api/orders/chases'

/** 会员中心 · 追号记录（route 已下线，保留页面） */

type QuickDay = 'today' | 'yesterday'

function ymd(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

interface ChaseRow {
  time: string
  game: string
  chaseNo: string
  totalIssues: string
  doneIssues: string
  amount: string
  status: string
}

const router = useRouter()
const loading = ref(false)

const quickDay = ref<QuickDay>('today')
const gameId = ref('all')
const dateRange = ref<[string, string] | null>(null)

const gameOptions = [
  { value: 'all', label: '全部彩种' },
  { value: 'ssc', label: '时时彩' },
  { value: 'pk10', label: 'PK10' },
  { value: 'k3', label: '快3' },
  { value: 'x5', label: '11选5' },
]

const queried = ref(false)
const rows = ref<ChaseRow[]>([])

function syncQuickRange(which: QuickDay): void {
  const d = new Date()
  if (which === 'yesterday') d.setDate(d.getDate() - 1)
  const s = ymd(d)
  dateRange.value = [s, s]
}

syncQuickRange(quickDay.value)

function onQuickDay(which: QuickDay): void {
  quickDay.value = which
  syncQuickRange(which)
}

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member' })
}

function onSearch(): void {
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) {
    ElMessage.warning('请选择日期区间')
    return
  }
  void loadRows()
}

async function loadRows(): Promise<void> {
  if (!dateRange.value?.[0] || !dateRange.value[1]) return
  loading.value = true
  try {
    const result = await fetchChaseOrders({
      dateFrom: dateRange.value[0],
      dateTo: dateRange.value[1],
      gameCode: gameId.value,
    })
    queried.value = true
    rows.value = result.items.map((item) => toChaseDisplayRow(item))
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  onSearch()
})
</script>

<template>
  <div class="mbr member-subpage" data-page="member-chase-records">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">追号记录</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <nav class="mss-quick mss-quick--underline" aria-label="日期快捷">
      <div class="mss-quick-underline-track">
        <button
          type="button"
          class="mss-quick-tab"
          :class="{ 'is-active': quickDay === 'today' }"
          @click="onQuickDay('today')"
        >
          今日
        </button>
        <span class="mss-quick-tab-divider" aria-hidden="true" />
        <button
          type="button"
          class="mss-quick-tab"
          :class="{ 'is-active': quickDay === 'yesterday' }"
          @click="onQuickDay('yesterday')"
        >
          昨日
        </button>
      </div>
    </nav>

    <main class="mbr-main">
      <section class="mbr-card mbr-filters">
        <div class="mbr-field">
          <div class="mbr-lbl">
            <span class="mbr-lbl-bar" aria-hidden="true" />
            <span>彩种</span>
          </div>
          <el-select v-model="gameId" size="large" class="mbr-select">
            <el-option v-for="o in gameOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="mbr-field">
          <div class="mbr-lbl">
            <span class="mbr-lbl-bar" aria-hidden="true" />
            <span>日期</span>
          </div>
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            size="large"
            class="mbr-dp"
            style="width: 100%; max-width: 100%; box-sizing: border-box"
          />
        </div>

        <div class="mbr-actions">
          <el-button type="primary" size="large" round class="mbr-query" @click="onSearch">
            查询
          </el-button>
        </div>
      </section>

      <section class="mbr-results" aria-live="polite">
        <template v-if="!queried">
          <div class="mbr-empty">
            <span class="mbr-ms mbr-empty-ico" aria-hidden="true">timeline</span>
            <p class="mbr-empty-title">暂无追号记录</p>
            <p class="mbr-empty-desc">调整筛选条件后点击查询查看追号任务</p>
          </div>
        </template>
        <template v-else>
          <el-table
            :data="rows"
            stripe
            size="small"
            class="mbr-table member-list-table"
            empty-text="暂无数据"
            style="width: 100%"
          >
            <el-table-column prop="time" label="时间" :min-width="44" />
            <el-table-column prop="game" label="彩种" :min-width="40" />
            <el-table-column prop="chaseNo" label="追号单号" :min-width="44" />
            <el-table-column prop="totalIssues" label="总期数" :min-width="40" />
            <el-table-column prop="doneIssues" label="已完成" :min-width="40" />
            <el-table-column prop="amount" label="金额" :min-width="36" />
            <el-table-column prop="status" label="状态" :min-width="36" />
          </el-table>
          <p v-if="false" class="mbr-footnote">数据仅供参考，以第三方平台为准</p>
        </template>
      </section>
    </main>
  </div>
</template>

<style scoped>
.mbr {
  --mbr-primary: #0050cb;
  --mbr-primary-strong: #0066ff;
  --mbr-surface: #f7f9fb;
  --mbr-tonal: #eef2f7;
  --mbr-card: #ffffff;
  --mbr-on: #191c1e;
  --mbr-on-var: #424656;
  --mbr-on-mute: #727687;
  min-height: 100dvh;
  background: var(--mbr-surface);
  color: var(--mbr-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mbr-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.35rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.mbr-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1rem 1.15rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mbr-card {
  background: var(--mbr-card);
  border-radius: 1.25rem;
  padding: 1.15rem;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

.mbr-filters {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mbr-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.mbr-lbl {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--mbr-on);
  letter-spacing: 0.02em;
}

.mbr-lbl-bar {
  width: 3px;
  height: 1rem;
  border-radius: 999px;
  background: rgba(0, 80, 203, 0.35);
}

.mbr-select {
  width: 100%;
}

.mbr-select :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mbr-dp {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  display: block;
}

.mbr-dp :deep(.el-input__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mbr-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding-top: 0.25rem;
}

.mbr-query {
  font-weight: 800;
  letter-spacing: 0.03em;
  padding-left: 1.5rem;
  padding-right: 1.5rem;
  box-shadow: 0 14px 32px -16px rgba(0, 80, 203, 0.55);
}

.mbr-results {
  min-height: 12rem;
  min-width: 0;
  max-width: 100%;
  overflow-x: hidden;
}

.mbr-table {
  width: 100%;
  background: var(--mbr-card);
  border-radius: 1.25rem;
  overflow: hidden;
  box-shadow: 0 18px 40px -28px rgba(15, 23, 42, 0.12);
  --el-table-border-color: transparent;
  --el-table-header-bg-color: #f8fafc;
  --el-table-bg-color: var(--mbr-card);
}

.mbr-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.mbr-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 2.5rem 1rem;
  gap: 0.35rem;
}

.mbr-empty-ico {
  font-size: 2.25rem;
  color: rgba(0, 80, 203, 0.35);
}

.mbr-empty-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--mbr-on-var);
}

.mbr-empty-desc {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--mbr-on-mute);
  line-height: 1.55;
}

.mbr-footnote {
  margin: 0.75rem 0 0;
  font-size: 0.6875rem;
  line-height: 1.5;
  color: var(--mbr-on-mute);
  text-align: center;
}
</style>
