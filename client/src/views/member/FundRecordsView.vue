<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import DateRangePickerField from '@/components/ui/DateRangePickerField.vue'
import {
  fetchFundRecords,
  toFundRecordDisplayRow,
  type FundCurrency,
  type FundFlowType,
  type FundRecordDisplayRow,
} from '@/api/funds/records'
import { PRIMARY_CURRENCIES } from '@/api/guaji/accounts'

/** 会员中心 · 资金记录（bet_debit / payout 等流水） */

const PAGE_SIZE = 20

function ymd(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function todayRange(): [string, string] {
  const s = ymd(new Date())
  return [s, s]
}

interface SelectOption<T extends string = string> {
  value: T
  label: string
}

const router = useRouter()

const dateRange = ref<[string, string] | null>(todayRange())
const flowType = ref<FundFlowType>('all')
const currency = ref<FundCurrency>('all')

const flowTypeOptions: SelectOption<FundFlowType>[] = [
  { value: 'all', label: '全部' },
  { value: 'income', label: '收入' },
  { value: 'expense', label: '支出' },
]

const currencyOptions = computed<SelectOption<FundCurrency>[]>(() => [
  { value: 'all', label: '全部币种' },
  ...PRIMARY_CURRENCIES.map((c) => ({ value: c as FundCurrency, label: c })),
])

const ready = ref(false)
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const rows = ref<FundRecordDisplayRow[]>([])
const loadSentinel = ref<HTMLElement | null>(null)

let loadObserver: IntersectionObserver | null = null

watch(dateRange, (v) => {
  if (!v || !v[0] || !v[1]) {
    dateRange.value = todayRange()
  }
})

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member' })
}

function resetPagination(): void {
  nextCursor.value = null
  hasMore.value = false
}

async function fetchFundPage(cursor?: string, append = false): Promise<void> {
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) return
  const result = await fetchFundRecords({
    dateFrom: dateRange.value[0],
    dateTo: dateRange.value[1],
    flowType: flowType.value,
    currency: currency.value,
    cursor,
    limit: PAGE_SIZE,
  })
  const mapped = result.items.map(toFundRecordDisplayRow)
  rows.value = append ? [...rows.value, ...mapped] : mapped
  hasMore.value = result.page.hasMore
  nextCursor.value = result.page.nextCursor ?? null
}

async function runSearch(auto = false): Promise<void> {
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) {
    if (!auto) ElMessage.warning('请选择日期区间')
    return
  }
  loading.value = true
  resetPagination()
  try {
    await fetchFundPage(undefined, false)
  } catch {
    if (!auto) ElMessage.error('加载资金记录失败')
    rows.value = []
  } finally {
    loading.value = false
    ready.value = true
  }
}

async function onSearch(): Promise<void> {
  await runSearch(false)
}

async function loadMore(): Promise<void> {
  if (!ready.value || !hasMore.value || !nextCursor.value || loading.value || loadingMore.value) return
  loadingMore.value = true
  try {
    await fetchFundPage(nextCursor.value, true)
  } catch {
    ElMessage.error('加载更多失败')
  } finally {
    loadingMore.value = false
  }
}

function setupLoadObserver(): void {
  loadObserver?.disconnect()
  loadObserver = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) void loadMore()
    },
    { root: null, rootMargin: '120px 0px' },
  )
  if (loadSentinel.value) loadObserver.observe(loadSentinel.value)
}

watch(loadSentinel, (el) => {
  if (!loadObserver) return
  loadObserver.disconnect()
  if (el) loadObserver.observe(el)
})

watch(ready, (v) => {
  if (v) setupLoadObserver()
})

onMounted(() => {
  void runSearch(true)
  setupLoadObserver()
})

onUnmounted(() => {
  loadObserver?.disconnect()
  loadObserver = null
})
</script>

<template>
  <div class="fr member-subpage" data-page="member-fund-records">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back</span>
        </button>
        <h1 class="mss-title">资金记录</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="fr-main">
      <section class="fr-card fr-filters">
        <div class="fr-field fr-field--inline">
          <div class="fr-lbl">
            <span class="fr-lbl-bar" aria-hidden="true" />
            <span>日期</span>
          </div>
          <DateRangePickerField v-model="dateRange" size="large" class="fr-drp" />
        </div>
        <div class="fr-field fr-field--inline">
          <div class="fr-lbl">
            <span class="fr-lbl-bar" aria-hidden="true" />
            <span>类型</span>
          </div>
          <el-select v-model="flowType" size="large" class="fr-select">
            <el-option v-for="o in flowTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="fr-field fr-field--inline">
          <div class="fr-lbl">
            <span class="fr-lbl-bar" aria-hidden="true" />
            <span>币种</span>
          </div>
          <el-select v-model="currency" size="large" class="fr-select">
            <el-option v-for="o in currencyOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>

        <div class="fr-actions">
          <el-button type="primary" size="large" round class="fr-query" :loading="loading" @click="onSearch">
            查询
          </el-button>
        </div>
      </section>

      <section class="fr-results" aria-live="polite">
        <el-skeleton v-if="loading && !ready" animated :rows="4" />
        <div v-else-if="!rows.length" class="fr-empty">
          <span class="fr-ms fr-empty-ico" aria-hidden="true">receipt_long</span>
          <p class="fr-empty-title">暂无资金记录</p>
        </div>
        <template v-else>
          <ul class="fr-list" role="list">
            <li v-for="it in rows" :key="it.id" class="fr-item">
              <div class="fr-item-head">
                <span class="fr-item-scheme">{{ it.schemeName }}</span>
                <span class="fr-item-amount" :class="it.tone">{{ it.amount }}</span>
              </div>
              <div class="fr-item-meta">
                <span class="fr-item-tag">{{ it.flowType }}</span>
                <span class="fr-item-currency">{{ it.currency }}</span>
                <span class="fr-item-time">{{ it.time }}</span>
              </div>
              <div class="fr-item-foot">
                <span class="fr-item-foot-lbl">变化后余额</span>
                <span class="fr-item-foot-val">{{ it.balanceAfter }}</span>
              </div>
            </li>
          </ul>
          <div v-if="hasMore || loadingMore" ref="loadSentinel" class="fr-load-sentinel" aria-hidden="true" />
          <p v-if="loadingMore" class="fr-load-hint">加载中…</p>
          <p v-else-if="rows.length && !hasMore" class="fr-load-hint">已加载全部</p>
        </template>
      </section>
    </main>
  </div>
</template>

<style scoped>
.fr {
  min-height: 100dvh;
  background: #f7f9fb;
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
}

.fr-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.35rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.fr-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1rem 1.15rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.fr-card {
  background: #fff;
  border-radius: 1.25rem;
  padding: 1.15rem;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

.fr-filters {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.fr-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.fr-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 0.65rem;
}

.fr-field--inline .fr-lbl {
  flex: 0 0 4.75rem;
  white-space: nowrap;
}

.fr-field--inline .fr-select,
.fr-field--inline .fr-drp {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
}

.fr-lbl {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.8125rem;
  font-weight: 800;
  color: #191c1e;
  letter-spacing: 0.02em;
}

.fr-lbl-bar {
  width: 3px;
  height: 1rem;
  border-radius: 999px;
  background: rgba(0, 80, 203, 0.35);
}

.fr-select {
  width: 100%;
}

.fr-select :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.fr-drp {
  width: 100%;
  min-width: 0;
}

.fr-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 0.125rem;
}

.fr-query {
  font-weight: 800;
  letter-spacing: 0.03em;
  padding-left: 1.5rem;
  padding-right: 1.5rem;
  box-shadow: 0 14px 32px -16px rgba(0, 80, 203, 0.55);
}

.fr-results {
  min-height: 12rem;
}

.fr-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 2.5rem 1rem;
  gap: 0.35rem;
}

.fr-empty-ico {
  font-size: 2.25rem;
  color: rgba(0, 80, 203, 0.35);
}

.fr-empty-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: #424656;
}

.fr-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.fr-item {
  background: #fff;
  border-radius: 1rem;
  padding: 1rem 1.125rem;
  box-shadow: 0 12px 32px -22px rgba(15, 35, 95, 0.18);
}

.fr-item-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.fr-item-scheme {
  font-weight: 700;
  font-size: 0.9375rem;
  color: #0f172a;
  line-height: 1.45;
}

.fr-item-amount {
  flex-shrink: 0;
  font-weight: 800;
  font-size: 0.9375rem;
  font-variant-numeric: tabular-nums;
  color: #0050cb;
}

.fr-item-amount.income {
  color: #1f9d63;
}

.fr-item-amount.expense {
  color: #ba1a1a;
}

.fr-item-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem 0.65rem;
  margin-top: 0.45rem;
}

.fr-item-tag {
  font-size: 0.6875rem;
  font-weight: 750;
  letter-spacing: 0.02em;
  color: #424656;
  background: #eef2f7;
  border-radius: 999px;
  padding: 0.15rem 0.55rem;
}

.fr-item-currency {
  font-size: 0.75rem;
  font-weight: 700;
  color: #64748b;
}

.fr-item-time {
  font-size: 0.75rem;
  color: #94a3b8;
}

.fr-item-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.65rem;
  padding-top: 0.65rem;
  border-top: 1px solid #f1f5f9;
}

.fr-item-foot-lbl {
  font-size: 0.75rem;
  color: #64748b;
}

.fr-item-foot-val {
  font-size: 0.8125rem;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
}

.fr-load-sentinel {
  height: 1px;
}

.fr-load-hint {
  margin: 0.65rem 0 0;
  text-align: center;
  font-size: 0.75rem;
  color: #727687;
  line-height: 1.5;
}
</style>
