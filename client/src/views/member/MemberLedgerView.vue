<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { fetchWalletLedger, toLedgerDisplayRow } from '@/api/member/ledger'

/** 会员中心 · 帐变记录（wallet ledger） */

type QuickDay = 'today' | 'yesterday'

function ymd(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

interface LedgerRow {
  time: string
  type: string
  orderId: string
  delta: string
  balance: string
}

const router = useRouter()

const quickDay = ref<QuickDay>('today')
const filterType = ref('all')
const orderNo = ref('')
const dateRange = ref<[string, string] | null>(null)

const typeOptions = [
  { value: 'all', label: '全部' },
  { value: 'deposit', label: '充值' },
  { value: 'withdraw', label: '提现' },
  { value: 'bet', label: '投注' },
  { value: 'payout', label: '派彩' },
  { value: 'adjust', label: '调整' },
]

const queried = ref(false)
const loading = ref(false)
const rows = ref<LedgerRow[]>([])

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

async function onSearch(): Promise<void> {
  if (!dateRange.value || !dateRange.value[0] || !dateRange.value[1]) {
    ElMessage.warning('请选择日期区间')
    return
  }
  loading.value = true
  try {
    const result = await fetchWalletLedger({
      dateFrom: dateRange.value[0],
      dateTo: dateRange.value[1],
      type: filterType.value,
      orderNo: orderNo.value.trim() || undefined,
    })
    rows.value = result.items.map(toLedgerDisplayRow)
    queried.value = true
  } catch {
    ElMessage.error('加载帐变记录失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  onSearch()
})
</script>

<template>
  <div class="mlg member-subpage" data-page="member-ledger">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">帐变记录</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <nav class="mss-quick" aria-label="日期快捷">
      <div class="mss-quick-track">
        <button
          type="button"
          class="mss-quick-btn"
          :class="{ 'is-active': quickDay === 'today' }"
          @click="onQuickDay('today')"
        >
          今日
        </button>
        <button
          type="button"
          class="mss-quick-btn"
          :class="{ 'is-active': quickDay === 'yesterday' }"
          @click="onQuickDay('yesterday')"
        >
          昨日
        </button>
      </div>
    </nav>

    <main class="mlg-main">
      <section class="mlg-card mlg-filters">
        <div class="mlg-field">
          <div class="mlg-lbl">
            <span class="mlg-lbl-bar" aria-hidden="true" />
            <span>类型</span>
          </div>
          <el-select v-model="filterType" placeholder="全部" size="large" class="mlg-select">
            <el-option v-for="o in typeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="mlg-field">
          <div class="mlg-lbl">
            <span class="mlg-lbl-bar" aria-hidden="true" />
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
            class="mlg-dp"
            style="width: 100%; max-width: 100%; box-sizing: border-box"
          />
        </div>
        <div class="mlg-field">
          <div class="mlg-lbl">
            <span class="mlg-lbl-bar" aria-hidden="true" />
            <span>订单号</span>
          </div>
          <el-input
            v-model="orderNo"
            clearable
            size="large"
            placeholder="可选，支持模糊搜索"
            class="mlg-input"
          />
        </div>

        <div class="mlg-actions">
          <el-button type="primary" size="large" round class="mlg-query" @click="onSearch">
            查询
          </el-button>
        </div>
      </section>

      <section class="mlg-results" aria-live="polite">
        <template v-if="!queried">
          <div class="mlg-empty">
            <span class="mlg-ms mlg-empty-ico" aria-hidden="true">receipt_long</span>
            <p class="mlg-empty-title">暂无帐变记录</p>
            <p class="mlg-empty-desc">请选择筛选条件后点击查询</p>
          </div>
        </template>
        <template v-else>
          <el-table
            :data="rows"
            stripe
            size="small"
            class="mlg-table member-list-table"
            empty-text="暂无数据"
            style="width: 100%"
          >
            <el-table-column prop="time" label="时间" :min-width="44" />
            <el-table-column prop="type" label="类型" :min-width="36" />
            <el-table-column prop="orderId" label="单号" :min-width="36" />
            <el-table-column prop="delta" label="变动" :min-width="36" />
            <el-table-column prop="balance" label="余额" :min-width="36" />
          </el-table>
          <p v-if="false" class="mlg-footnote">数据仅供参考，以第三方平台为准</p>
        </template>
      </section>
    </main>
  </div>
</template>

<style scoped>
.mlg {
  --mlg-primary: #0050cb;
  --mlg-primary-strong: #0066ff;
  --mlg-surface: #f7f9fb;
  --mlg-tonal: #eef2f7;
  --mlg-card: #ffffff;
  --mlg-on: #191c1e;
  --mlg-on-var: #424656;
  --mlg-on-mute: #727687;
  min-height: 100dvh;
  background: var(--mlg-surface);
  color: var(--mlg-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mlg-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.35rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.mlg-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1rem var(--page-gutter) 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mlg-card {
  background: var(--mlg-card);
  border-radius: 1.25rem;
  padding: var(--card-pad);
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
}

.mlg-filters {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mlg-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.mlg-lbl {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.8125rem;
  font-weight: 800;
  color: var(--mlg-on);
  letter-spacing: 0.02em;
}

.mlg-lbl-bar {
  width: 3px;
  height: 1rem;
  border-radius: 999px;
  background: rgba(0, 80, 203, 0.35);
}

.mlg-select {
  width: 100%;
}

.mlg-select :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mlg-dp {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  display: block;
  --el-date-editor-daterange-width: 100% !important;
  --el-date-editor-width: 100% !important;
}

.mlg-dp :deep(.el-date-editor.el-input__wrapper) {
  width: 100% !important;
  max-width: 100% !important;
  min-width: 0 !important;
  box-sizing: border-box;
}

.mlg-dp :deep(.el-input__wrapper) {
  max-width: 100% !important;
  min-width: 0 !important;
  box-sizing: border-box;
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
  overflow: hidden;
}

.mlg-input :deep(.el-input__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mlg-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding-top: 0.25rem;
}

.mlg-query {
  font-weight: 800;
  letter-spacing: 0.03em;
  padding-left: 1.5rem;
  padding-right: 1.5rem;
  box-shadow: 0 14px 32px -16px rgba(0, 80, 203, 0.55);
}

.mlg-results {
  min-height: 12rem;
  min-width: 0;
  max-width: 100%;
  overflow-x: hidden;
}

.mlg-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 2.5rem 1rem;
  gap: 0.35rem;
}

.mlg-empty-ico {
  font-size: 2.25rem;
  color: rgba(0, 80, 203, 0.35);
}

.mlg-empty-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--mlg-on-var);
}

.mlg-empty-desc {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--mlg-on-mute);
  line-height: 1.55;
  max-width: 18rem;
}

.mlg-table {
  width: 100%;
  background: var(--mlg-card);
  border-radius: 1.25rem;
  overflow: hidden;
  box-shadow: 0 18px 40px -28px rgba(15, 23, 42, 0.12);
  --el-table-border-color: transparent;
  --el-table-header-bg-color: #f8fafc;
  --el-table-bg-color: var(--mlg-card);
}

.mlg-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.mlg-footnote {
  margin: 0.75rem 0 0;
  font-size: 0.6875rem;
  line-height: 1.5;
  color: var(--mlg-on-mute);
  text-align: center;
}
</style>
