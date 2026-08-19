<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getAccessToken } from '@/api/client'
import { WS_CLIENT_BASE, WS_CLIENT_ENABLED } from '@/api/config'
import {
  BET_RECORD_DETAIL_PAGE_SIZE,
  fetchBetRecordDetail,
  toDisplayRow,
  type BetRecordDisplayRow,
  type BetRecordMode,
} from '@/api/cloud/betRecords'
import {
  getBetListMemoryCache,
  peekBetListRestore,
  saveBetListRestore,
  setBetListMemoryCache,
  takeBetListRestore,
} from '@/composables/useBetListRestore'
import { cloudRunningPollMs } from '@/composables/useCloudRunningPoll'
import { connectClientWs, isCloudRefreshEvent } from '@/composables/ws/useClientWs'

type SchemeBetListCache = {
  schemeName: string
  rows: BetRecordDisplayRow[]
  nextCursor: string | null
  hasMore: boolean
}

const route = useRoute()
const router = useRouter()

const mode = computed<BetRecordMode>(() => (route.query.mode === 'sim' ? 'sim' : 'real'))
const schemeId = computed(() => String(route.params.schemeId ?? ''))
const pageKey = computed(() => `cloud-scheme:${schemeId.value}:${mode.value}`)

const schemeName = ref('方案明细')
const displayRows = ref<BetRecordDisplayRow[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextCursor = ref<string | null>(null)
const loadSentinel = ref<HTMLElement | null>(null)

let loadObserver: IntersectionObserver | null = null
let stopLiveWs: (() => void) | null = null
let livePollTimer: ReturnType<typeof window.setInterval> | null = null
let silentRefreshInFlight = false
let silentRefreshQueued = false
let liveRefreshStopped = true

function resetPagination(): void {
  nextCursor.value = null
  hasMore.value = false
}

function persistCache(): void {
  setBetListMemoryCache<SchemeBetListCache>(pageKey.value, {
    schemeName: schemeName.value,
    rows: displayRows.value.slice(),
    nextCursor: nextCursor.value,
    hasMore: hasMore.value,
  })
}

async function fetchDetailPage(cursor?: string, append = false): Promise<void> {
  const data = await fetchBetRecordDetail(schemeId.value, {
    mode: mode.value,
    days: 3,
    cursor,
    limit: BET_RECORD_DETAIL_PAGE_SIZE,
  })
  schemeName.value = data.schemeName
  const mapped = data.records.items.map(toDisplayRow)
  displayRows.value = append ? [...displayRows.value, ...mapped] : mapped
  hasMore.value = data.records.page?.hasMore ?? false
  nextCursor.value = data.records.page?.nextCursor ?? null
  persistCache()
}

async function loadDetail(reset = true): Promise<void> {
  if (reset) {
    loading.value = true
    resetPagination()
  }
  try {
    await fetchDetailPage(undefined, false)
  } catch (e) {
    if (reset) {
      schemeName.value = '方案明细'
      displayRows.value = []
    }
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
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
    await fetchDetailPage(nextCursor.value, true)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loadingMore.value = false
    await nextTick()
    setupLoadObserver()
  }
}

async function ensureAnchorVisible(anchorRecordNo: string): Promise<void> {
  const target = anchorRecordNo.trim()
  if (!target) return
  for (let i = 0; i < 20; i++) {
    if (displayRows.value.some((r) => r.recordNo === target)) return
    if (!hasMore.value || !nextCursor.value) return
    await loadMore()
  }
}

function scrollToAnchor(anchorRecordNo: string, scrollY: number): void {
  // 优先还原离开时的滚动位置；锚点仅在内容高度变化导致 scrollY 无效时兜底
  if (Number.isFinite(scrollY) && scrollY > 0) {
    window.scrollTo(0, scrollY)
    return
  }
  const target = anchorRecordNo.trim()
  if (!target) return
  const el = document.querySelector(`[data-record-no="${CSS.escape(target)}"]`)
  if (el instanceof HTMLElement) {
    el.scrollIntoView({ block: 'center' })
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

async function bootstrap(): Promise<void> {
  const cached = getBetListMemoryCache<SchemeBetListCache>(pageKey.value)
  const pending = peekBetListRestore(pageKey.value)
  if (pending && cached?.rows.length) {
    const restore = takeBetListRestore(pageKey.value)!
    schemeName.value = cached.schemeName
    displayRows.value = cached.rows
    nextCursor.value = cached.nextCursor
    hasMore.value = cached.hasMore
    await nextTick()
    setupLoadObserver()
    await nextTick()
    scrollToAnchor(restore.anchorRecordNo, restore.scrollY)
    // 瞬时还原后静默刷新：避免进详情期间已结算的盈亏仍显示旧值
    void silentRefresh(restore.scrollY)
    return
  }
  const restore = takeBetListRestore(pageKey.value)
  await loadDetail()
  if (restore?.anchorRecordNo) {
    await ensureAnchorVisible(restore.anchorRecordNo)
    await nextTick()
    scrollToAnchor(restore.anchorRecordNo, restore.scrollY)
  }
}

/** 不打断滚动地重拉至不少于当前条数（避免进详情期间结算结果仍显示旧盈亏） */
async function silentRefresh(keepScrollY: number): Promise<void> {
  const want = Math.max(displayRows.value.length, BET_RECORD_DETAIL_PAGE_SIZE)
  try {
    const merged: BetRecordDisplayRow[] = []
    let cursor: string | undefined
    let more = true
    let name = schemeName.value
    while (more && merged.length < want) {
      const data = await fetchBetRecordDetail(schemeId.value, {
        mode: mode.value,
        days: 3,
        cursor,
        limit: BET_RECORD_DETAIL_PAGE_SIZE,
      })
      name = data.schemeName
      merged.push(...data.records.items.map(toDisplayRow))
      more = data.records.page?.hasMore ?? false
      cursor = data.records.page?.nextCursor ?? undefined
      if (!cursor) more = false
    }
    schemeName.value = name
    displayRows.value = merged
    hasMore.value = more
    nextCursor.value = cursor ?? null
    persistCache()
    await nextTick()
    setupLoadObserver()
    if (Number.isFinite(keepScrollY) && keepScrollY > 0) {
      window.scrollTo(0, keepScrollY)
    }
  } catch {
    /* 静默失败保留缓存展示 */
  }
}

function requestSilentRefresh(): void {
  if (liveRefreshStopped) return
  if (silentRefreshInFlight) {
    silentRefreshQueued = true
    return
  }
  silentRefreshInFlight = true
  void silentRefresh(window.scrollY).finally(() => {
    silentRefreshInFlight = false
    if (silentRefreshQueued) {
      silentRefreshQueued = false
      requestSilentRefresh()
    }
  })
}

function restartLivePoll(wsConnected: boolean): void {
  if (liveRefreshStopped) return
  if (livePollTimer) window.clearInterval(livePollTimer)
  livePollTimer = window.setInterval(requestSilentRefresh, cloudRunningPollMs(wsConnected))
}

function startLiveRefresh(): void {
  if (liveRefreshStopped) return
  restartLivePoll(false)
  const token = getAccessToken()
  if (!WS_CLIENT_ENABLED || !WS_CLIENT_BASE || !token) return
  stopLiveWs = connectClientWs(
    WS_CLIENT_BASE,
    token,
    (env) => {
      if (isCloudRefreshEvent(env)) requestSilentRefresh()
    },
    {
      onConnected: () => restartLivePoll(true),
      onDisconnected: () => restartLivePoll(false),
    },
  )
}

function stopLiveRefresh(): void {
  liveRefreshStopped = true
  stopLiveWs?.()
  stopLiveWs = null
  if (livePollTimer) {
    window.clearInterval(livePollTimer)
    livePollTimer = null
  }
}

onMounted(() => {
  liveRefreshStopped = false
  void bootstrap().finally(startLiveRefresh)
})

onUnmounted(() => {
  stopLiveRefresh()
  loadObserver?.disconnect()
  loadObserver = null
})

watch([schemeId, mode], () => {
  void loadDetail()
})

function goBack() {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'bet-records', query: { mode: mode.value } })
}

function openDetail(row: BetRecordDisplayRow): void {
  const recordNo = (row.recordNo || '').trim()
  if (!recordNo) {
    ElMessage.warning('该记录暂无法查看详情')
    return
  }
  persistCache()
  saveBetListRestore(pageKey.value, {
    scrollY: window.scrollY,
    anchorRecordNo: recordNo,
  })
  void router.push({ name: 'bet-detail', params: { recordNo } })
}
</script>

<template>
  <div class="br br-detail" data-page="bet-records-scheme">
    <header class="br-head" role="banner">
      <button type="button" class="br-back-btn br-back" aria-label="返回" @click="goBack">
        <span class="material-sym" aria-hidden="true">arrow_back_ios_new</span>
      </button>
      <h1 class="br-title">{{ schemeName }}</h1>
      <span class="br-head-spacer" aria-hidden="true" />
    </header>

    <main class="br-main">
      <section class="br-table-sec br-table-sec--detail">
        <div v-loading="loading" class="br-table-card">
          <el-table
            :data="displayRows"
            class="br-el-table br-detail-table br-table--clickable"
            size="small"
            stripe
            fit
            empty-text="暂无数据"
            :style="{ width: '100%' }"
            @row-click="openDetail"
          >
            <el-table-column prop="period" label="期号" width="32%" align="center" class-name="br-cell-order">
              <template #default="{ row }">
                <span class="br-td-order" :data-record-no="row.recordNo">{{ row.period }}</span>
              </template>
            </el-table-column>

            <el-table-column prop="multiplier" label="倍数" width="8%" align="center" />

            <el-table-column prop="round" label="轮次" width="8%" align="center" />

            <el-table-column prop="amount" label="金额" width="20%" align="center">
              <template #default="{ row }">
                <span class="br-td-num">{{ row.amount }}</span>
              </template>
            </el-table-column>

            <el-table-column label="盈亏" width="20%" align="center">
              <template #default="{ row }">
                <span
                  class="br-td-pl"
                  :class="{
                    'br-td-pl--gain': row.pnlPositive,
                    'br-td-pl--loss': !row.pnlPositive,
                  }"
                >
                  {{ row.pnl }}
                </span>
              </template>
            </el-table-column>

            <el-table-column label="状态" width="12%" align="center">
              <template #default="{ row }">
                <el-tag
                  :type="row.statusHit === true ? 'primary' : row.statusHit === false ? 'danger' : 'info'"
                  effect="light"
                  size="small"
                  class="br-status-tag"
                >
                  {{ row.statusLabel }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>

          <div v-if="hasMore || loadingMore" ref="loadSentinel" class="br-load-sentinel" aria-hidden="true" />
          <p v-if="loadingMore" class="br-load-hint">加载中…</p>
          <p v-else-if="displayRows.length && !hasMore" class="br-load-hint">已加载全部</p>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped src="./bet-records-shared.css"></style>

<style scoped>
.br-head-spacer {
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  justify-self: end;
}

.br-table-sec--detail {
  padding-top: 1rem;
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

.br-detail-table.el-table {
  width: 100% !important;
}

.br-detail-table :deep(.el-table__header-wrapper),
.br-detail-table :deep(.el-table__body-wrapper) {
  width: 100% !important;
}

.br-detail-table :deep(.el-table__header-wrapper table),
.br-detail-table :deep(.el-table__body-wrapper table) {
  table-layout: fixed !important;
  width: 100% !important;
}

.br-detail-table :deep(.el-table__header th.el-table__cell) {
  font-size: 10px;
  font-weight: 700;
  color: #64748b !important;
  padding: 5px 0 !important;
  vertical-align: middle;
}

.br-detail-table :deep(.el-table__body td.el-table__cell) {
  font-size: 11px;
  padding: 5px 0 !important;
  vertical-align: middle;
}

.br-detail-table :deep(th.el-table__cell .cell),
.br-detail-table :deep(td.el-table__cell .cell) {
  padding: 0 1px !important;
  line-height: 1.35;
  text-align: center;
}

.br-detail-table :deep(th.el-table__cell .cell) {
  white-space: normal;
  word-break: keep-all;
  overflow: visible;
  text-overflow: clip;
}

.br-detail-table :deep(td.br-cell-order .cell) {
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  overflow: visible;
  text-overflow: clip;
}

.br-detail-table :deep(.el-table__header th.el-table__cell > .cell) {
  display: block;
  width: 100%;
}

.br-td-num,
.br-td-pl {
  font-variant-numeric: tabular-nums;
}

.br-td-order {
  display: inline-block;
  max-width: 100%;
  font-size: 11px;
  line-height: 1.35;
  white-space: normal;
  word-break: break-all;
  overflow-wrap: anywhere;
  color: var(--br-on-var);
  font-family: Inter, ui-monospace, 'SFMono-Regular', Consolas, monospace;
  font-variant-numeric: tabular-nums;
}

.br-detail-table :deep(.br-status-tag) {
  padding-inline: 0.25rem;
  transform: scale(0.9);
  transform-origin: center;
}

.br-table--clickable :deep(.el-table__body tr) {
  cursor: pointer;
}

.br-table--clickable :deep(.el-table__body tr:hover > td.el-table__cell) {
  background: rgba(0, 80, 203, 0.04) !important;
}
</style>
