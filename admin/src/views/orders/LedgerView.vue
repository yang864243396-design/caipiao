<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchAdminLedgerEntries, type AdminLedgerRow } from '@/api/orders'
import type { FundCurrency, FundFlowType } from '@/types/members'

const PRIMARY_CURRENCIES = ['USDT', 'TRX', 'CNY'] as const

interface AppliedLedgerQuery {
  dateFrom: string
  dateTo: string
  flowType: FundFlowType
  currency: FundCurrency
  memberAccount: string
  ledgerNo: string
}

function todayYmd(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const dateRange = ref<[string, string]>([todayYmd(), todayYmd()])
const flowType = ref<FundFlowType>('all')
const currency = ref<FundCurrency>('all')
const filterMemberAccount = ref('')
const filterLedgerNo = ref('')

const pageSize = ref(10)
const currentPage = ref(1)
const loading = ref(false)
const ready = ref(false)
const rows = ref<AdminLedgerRow[]>([])
const total = ref(0)
const appliedQuery = ref<AppliedLedgerQuery | null>(null)

const flowTypeOptions: { value: FundFlowType; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'income', label: '收入' },
  { value: 'expense', label: '支出' },
]

const currencyOptions: { value: FundCurrency; label: string }[] = [
  { value: 'all', label: '全部币种' },
  ...PRIMARY_CURRENCIES.map((c) => ({ value: c as FundCurrency, label: c })),
]

async function reload() {
  if (!appliedQuery.value) return
  loading.value = true
  try {
    const q = appliedQuery.value
    const res = await fetchAdminLedgerEntries({
      dateFrom: q.dateFrom,
      dateTo: q.dateTo,
      flowType: q.flowType,
      currency: q.currency,
      memberAccount: q.memberAccount,
      ledgerNo: q.ledgerNo,
      page: currentPage.value,
      pageSize: pageSize.value,
    })
    rows.value = res.items
    total.value = res.total
    ready.value = true
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载钱包流水失败')
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function runSearch(auto = false) {
  if (!dateRange.value[0] || !dateRange.value[1]) {
    if (!auto) ElMessage.warning('请选择日期区间')
    return
  }
  currentPage.value = 1
  appliedQuery.value = {
    dateFrom: dateRange.value[0],
    dateTo: dateRange.value[1],
    flowType: flowType.value,
    currency: currency.value,
    memberAccount: filterMemberAccount.value.trim(),
    ledgerNo: filterLedgerNo.value.trim(),
  }
  await reload()
}

function onSearch() {
  void runSearch(false)
}

function onReset() {
  dateRange.value = [todayYmd(), todayYmd()]
  flowType.value = 'all'
  currency.value = 'all'
  filterMemberAccount.value = ''
  filterLedgerNo.value = ''
  currentPage.value = 1
  void runSearch(false)
}

function stubExport() {
  ElMessage.info('导出为占位：正式上线后接异步导出')
}

onMounted(() => {
  void runSearch(true)
})

watch(currentPage, () => {
  if (ready.value && appliedQuery.value) void reload()
})

function fmtMoney(v: number) {
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function fmtDelta(v: number) {
  const s = fmtMoney(Math.abs(v))
  return v >= 0 ? `+${s}` : `-${s}`
}

function fmtTime(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso || '—'
  return d.toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <div v-loading="loading">
    <div class="page-head">
      <div>
        <h1 class="admin-page-title">帐变流水</h1>

      </div>
      <el-button type="primary" plain @click="stubExport">导出</el-button>
    </div>

    <div class="toolbar toolbar--ledger">
      <div class="toolbar-field toolbar-field--date">
        <el-date-picker v-model="dateRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期"
          end-placeholder="结束日期" unlink-panels class="toolbar-date-picker" />
      </div>
      <div class="toolbar-field">
        <el-select v-model="flowType" placeholder="类型">
          <el-option v-for="o in flowTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
      </div>
      <div class="toolbar-field">
        <el-select v-model="currency" placeholder="币种">
          <el-option v-for="o in currencyOptions" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
      </div>
      <div class="toolbar-field">
        <el-input v-model="filterMemberAccount" clearable placeholder="会员账号" @keyup.enter="onSearch" />
      </div>
      <div class="toolbar-field">
        <el-input v-model="filterLedgerNo" clearable placeholder="流水号" @keyup.enter="onSearch" />
      </div>
      <div class="toolbar-actions">
        <el-button type="primary" @click="onSearch">查询</el-button>
        <el-button @click="onReset">重置</el-button>
      </div>
    </div>

    <el-table :data="rows" stripe style="width: 100%">
      <el-table-column prop="schemeName" label="方案名称" min-width="140" show-overflow-tooltip />
      <el-table-column prop="member" label="会员账号" min-width="108" />
      <el-table-column label="变动金额" min-width="120" align="right">
        <template #default="{ row }">
          <span :style="{
            color:
              row.flowTypeCode === 'income'
                ? 'var(--el-color-success)'
                : 'var(--el-color-danger)',
          }">
            {{ fmtDelta(row.amount) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="flowType" label="类型" min-width="80" />
      <el-table-column prop="currency" label="币种" min-width="72" />
      <el-table-column label="时间" min-width="160">
        <template #default="{ row }">{{ fmtTime(row.time) }}</template>
      </el-table-column>
      <el-table-column label="变化后余额" min-width="120" align="right">
        <template #default="{ row }">{{ fmtMoney(row.balanceAfter) }}</template>
      </el-table-column>
      <el-table-column prop="ledgerNo" label="流水号" min-width="120" show-overflow-tooltip />
    </el-table>

    <el-empty v-if="ready && !rows.length && !loading" description="暂无资金记录" style="margin-top: 1rem" />

    <div class="pager">
      <el-pagination v-model:current-page="currentPage" :page-size="pageSize" layout="total, prev, pager, next"
        :total="total" />
    </div>
  </div>
</template>

<style scoped>
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.admin-page-title {
  margin: 0 0 0.25rem;
}

.admin-page-desc {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  flex: 1;
  min-width: 12rem;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
  align-items: flex-end;
  margin-bottom: 1rem;
}

.toolbar-field {
  flex: 0 0 160px;
  width: 160px;
  max-width: 100%;
  min-width: 0;
}

.toolbar-field--date {
  flex: 0 0 280px;
  width: 280px;
}

.toolbar-field :deep(.el-select),
.toolbar-field :deep(.el-input) {
  width: 100%;
}

.toolbar-date-picker {
  width: 100%;
}

.toolbar-field--date :deep(.el-date-editor) {
  width: 100%;
  box-sizing: border-box;
}

.toolbar-actions {
  display: flex;
  gap: 0.5rem;
  flex: 0 0 auto;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
