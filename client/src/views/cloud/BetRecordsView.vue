<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  BET_RECORD_GROUP_PAGE_SIZE,
  fetchBetRecordGroups,
  type BetRecordMode,
  type BetRecordSchemeGroup,
} from '@/api/cloud/betRecords'
import type { MoneySummary } from '@/api/types'

const router = useRouter()
const route = useRoute()

const activeTab = ref<BetRecordMode>('real')
const loading = ref(false)
const loadingMore = ref(false)
const loadError = ref('')

const summary = ref<MoneySummary>({ totalBet: 0, dayPnl: 0, winRate: 0 })
const schemeGroups = ref<BetRecordSchemeGroup[]>([])
const dateFrom = ref('')
const dateTo = ref('')
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const loadSentinel = ref<HTMLElement | null>(null)

let loadObserver: IntersectionObserver | null = null

const hasData = computed(() => schemeGroups.value.length > 0)

function formatMoney(n: number, signed = false): string {
  const abs = Math.abs(n).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  if (!signed) return `¥${abs}`
  if (n > 0) return `+¥${abs}`
  if (n < 0) return `-¥${abs}`
  return `¥${abs}`
}

function syncTabFromRoute(): void {
  activeTab.value = route.query.mode === 'sim' ? 'sim' : 'real'
}

function resetPagination(): void {
  nextCursor.value = null
  hasMore.value = false
}

async function fetchGroupsPage(cursor?: string, append = false): Promise<void> {
  const data = await fetchBetRecordGroups({
    mode: activeTab.value,
    days: 3,
    cursor,
    limit: BET_RECORD_GROUP_PAGE_SIZE,
  })
  summary.value = data.summary
  dateFrom.value = data.dateFrom ?? ''
  dateTo.value = data.dateTo ?? ''
  const items = data.groups?.items ?? []
  schemeGroups.value = append ? [...schemeGroups.value, ...items] : items
  hasMore.value = data.groups?.page?.hasMore ?? false
  nextCursor.value = data.groups?.page?.nextCursor ?? null
}

async function loadRecords(reset = true): Promise<void> {
  if (reset) {
    loading.value = true
    resetPagination()
  }
  loadError.value = ''
  try {
    await fetchGroupsPage(undefined, false)
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : '加载失败'
    summary.value = { totalBet: 0, dayPnl: 0, winRate: 0 }
    schemeGroups.value = []
    dateFrom.value = ''
    dateTo.value = ''
    if (reset) ElMessage.error(loadError.value)
  } finally {
    if (reset) loading.value = false
    await nextTick()
    setupLoadObserver()
  }
}

async function loadMore(): Promise<void> {
  if (!hasMore.value || !nextCursor.value || loading.value || loadingMore.value) return
  loadingMore.value = true
  try {
    await fetchGroupsPage(nextCursor.value, true)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loadingMore.value = false
    await nextTick()
    setupLoadObserver()
  }
}

function setupLoadObserver(): void {
  loadObserver?.disconnect()
  if (!loadSentinel.value) return
  loadObserver = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) void loadMore()
    },
    { root: null, rootMargin: '120px', threshold: 0 },
  )
  loadObserver.observe(loadSentinel.value)
}

onMounted(() => {
  syncTabFromRoute()
  void loadRecords()
})

onUnmounted(() => {
  loadObserver?.disconnect()
  loadObserver = null
})

watch(() => route.query.mode, () => {
  syncTabFromRoute()
  void loadRecords()
})

function setTab(tab: BetRecordMode): void {
  activeTab.value = tab
  void router.replace({ name: 'bet-records', query: tab === 'sim' ? { mode: 'sim' } : {} })
  void loadRecords()
}

function goBack() {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'cloud' })
}

function openScheme(schemeId: string) {
  void router.push({
    name: 'bet-records-scheme',
    params: { schemeId },
    query: activeTab.value === 'sim' ? { mode: 'sim' } : {},
  })
}

function onRefresh() {
  void loadRecords().then(() => {
    if (!loadError.value) ElMessage.success('已刷新')
  })
}
</script>



<template>

  <div class="br" data-page="bet-records">

    <header class="br-head" role="banner">

      <button type="button" class="br-back-btn br-back" aria-label="返回" @click="goBack">
        <span class="material-sym" aria-hidden="true">arrow_back_ios_new</span>
      </button>

      <h1 class="br-title">最近三日投注记录</h1>

      <button

        type="button"

        class="br-icon-btn br-refresh"

        aria-label="刷新"

        :disabled="loading"

        @click="onRefresh"

      >

        <span class="br-ms" aria-hidden="true">refresh</span>

      </button>

    </header>



    <main class="br-main">

      <section class="br-tabs-wrap" aria-label="记录类型">

        <div class="br-tabs">

          <button

            type="button"

            class="br-tab"

            :class="{ 'br-tab--active': activeTab === 'real' }"

            @click="setTab('real')"

          >

            真实记录

          </button>

          <button

            type="button"

            class="br-tab"

            :class="{ 'br-tab--active': activeTab === 'sim' }"

            @click="setTab('sim')"

          >

            模拟记录

          </button>

        </div>

      </section>



      <section class="br-summary-wrap">

        <div class="br-summary">

          <div class="br-sum-cell">

            <span class="br-sum-lbl">总投注额</span>

            <span class="br-sum-val">{{ formatMoney(summary.totalBet) }}</span>

          </div>

          <div class="br-sum-divider" aria-hidden="true" />

          <div class="br-sum-cell">

            <span class="br-sum-lbl">当日盈亏</span>

            <div class="br-sum-pnl">

              <span

                class="br-sum-val br-sum-val--pnl"

                :class="{

                  'is-pos': summary.dayPnl > 0,

                  'is-neg': summary.dayPnl < 0,

                  'is-zero': summary.dayPnl === 0,

                }"

              >

                {{ formatMoney(summary.dayPnl, true) }}

              </span>

              <span

                v-if="summary.dayPnl > 0"

                class="br-ms br-sum-trend"

                aria-hidden="true"

              >trending_up</span>

            </div>

          </div>

          <div class="br-sum-divider" aria-hidden="true" />

          <div class="br-sum-cell br-sum-cell--end">

            <span class="br-sum-lbl">胜率</span>

            <span class="br-sum-val">{{ hasData ? `${summary.winRate.toFixed(1)}%` : '—' }}</span>

          </div>

        </div>

        <p v-if="dateFrom && dateTo" class="br-range-hint">

          统计区间 {{ dateFrom }} 至 {{ dateTo }}（UTC+8 自然日）

        </p>

      </section>



      <section class="br-scheme-sec" aria-label="按方案汇总">

        <ul v-if="schemeGroups.length" class="br-scheme-list" role="list">

          <li v-for="group in schemeGroups" :key="group.schemeId">

            <button type="button" class="br-scheme-card" @click="openScheme(group.schemeId)">

              <div class="br-scheme-head">

                <span class="br-scheme-name">{{ group.schemeName }}</span>

                <span class="br-ms br-scheme-chev" aria-hidden="true">chevron_right</span>

              </div>

              <div class="br-scheme-stats">

                <div class="br-scheme-stat">

                  <span class="br-scheme-stat-lbl">投注金额</span>

                  <span class="br-scheme-stat-val">{{ formatMoney(group.totalBet) }}</span>

                </div>

                <div class="br-scheme-stat">

                  <span class="br-scheme-stat-lbl">当日盈亏</span>

                  <span

                    class="br-scheme-stat-val"

                    :class="{

                      'is-pos': group.dayPnl > 0,

                      'is-neg': group.dayPnl < 0,

                      'is-zero': group.dayPnl === 0,

                    }"

                  >

                    {{ formatMoney(group.dayPnl, true) }}

                  </span>

                </div>

                <div class="br-scheme-stat br-scheme-stat--end">

                  <span class="br-scheme-stat-lbl">胜率</span>

                  <span class="br-scheme-stat-val">{{ group.winRate.toFixed(1) }}%</span>

                </div>

              </div>

            </button>

          </li>

        </ul>

        <div v-if="hasMore || loadingMore" ref="loadSentinel" class="br-load-sentinel" aria-hidden="true" />
        <p v-if="loadingMore" class="br-load-hint">加载中…</p>
        <p v-else-if="schemeGroups.length && !hasMore" class="br-load-hint">已加载全部</p>

        <p v-else-if="!loading && !schemeGroups.length" class="br-scheme-empty">暂无数据</p>

      </section>

    </main>

  </div>

</template>



<style scoped src="./bet-records-shared.css"></style>



<style scoped>

.br-scheme-sec {

  flex: 1;

  padding: 0 var(--page-gutter) 1.5rem;

  background: var(--br-surface-low);

}



.br-range-hint {

  margin: 0.75rem 0 0;

  font-size: 0.75rem;

  line-height: 1.5;

  color: var(--br-text-muted, #64748b);

  text-align: center;

}



.br-scheme-list {

  list-style: none;

  margin: 0;

  padding: 0;

  display: flex;

  flex-direction: column;

  gap: 0.75rem;

}



.br-scheme-card {

  width: 100%;

  border: none;

  text-align: left;

  cursor: pointer;

  font: inherit;

  color: inherit;

  background: #fff;

  border-radius: 0.75rem;

  padding: var(--card-pad);

  box-shadow: 0 4px 20px rgba(15, 23, 42, 0.06);

  transition: transform 0.15s, box-shadow 0.15s;

}



.br-scheme-card:hover {

  box-shadow: 0 8px 28px rgba(15, 23, 42, 0.1);

}



.br-scheme-card:active {

  transform: scale(0.99);

}



.br-scheme-head {

  display: flex;

  align-items: center;

  justify-content: space-between;

  gap: 0.75rem;

  margin-bottom: 0.75rem;

}



.br-scheme-name {

  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;

  font-size: 0.9375rem;

  font-weight: 800;

  color: var(--br-on);

  line-height: 1.35;

}



.br-scheme-chev {

  font-size: 1.25rem;

  color: rgba(66, 70, 86, 0.45);

}



.br-scheme-stats {

  display: flex;

  align-items: flex-start;

  gap: 0.75rem;

}



.br-scheme-stat {

  flex: 1;

  min-width: 0;

  display: flex;

  flex-direction: column;

  gap: 0.15rem;

}



.br-scheme-stat--end {

  align-items: flex-end;

  text-align: right;

}



.br-scheme-stat-lbl {

  font-size: 0.625rem;

  font-weight: 700;

  letter-spacing: 0.06em;

  text-transform: uppercase;

  color: rgba(66, 70, 86, 0.55);

}



.br-scheme-stat-val {

  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;

  font-size: 0.9375rem;

  font-weight: 800;

  font-variant-numeric: tabular-nums;

  color: var(--br-on);

}



.br-scheme-stat-val.is-pos {

  color: var(--br-primary-strong);

}



.br-scheme-stat-val.is-neg {

  color: var(--br-error);

}



.br-scheme-stat-val.is-zero {

  color: var(--br-on-var);

}

.br-scheme-empty {

  margin: 2rem 0 0;

  text-align: center;

  font-size: 0.875rem;

  color: rgba(66, 70, 86, 0.45);

}

.br-load-sentinel {
  height: 1px;
  margin-top: 0.5rem;
}

.br-load-hint {
  margin: 0.75rem 0 0;
  text-align: center;
  font-size: 0.75rem;
  color: rgba(66, 70, 86, 0.45);
}

</style>


