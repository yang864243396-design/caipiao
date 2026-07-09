<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import SchemeHistoryDrawer from '@/components/schemes/SchemeHistoryDrawer.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'
import { useMembersStore, type MemberRow } from '@/stores/members'
import { useSchemeInstancesStore } from '@/stores/schemeInstances'
import { postMemberOp } from '@/api/memberOps'
import { fetchMemberGuajiAccounts, type AdminGuajiAccountRow } from '@/api/guajiAccounts'
import type { FundCurrency, FundFlowType, MemberFundRecordRow } from '@/types/members'
import type { SchemeInstanceRow } from '@/stores/schemeInstances'

const PRIMARY_CURRENCIES = ['USDT', 'TRX', 'CNY'] as const

interface AppliedFundQuery {
  dateFrom: string
  dateTo: string
  flowType: FundFlowType
  currency: FundCurrency
}

function todayYmd(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const route = useRoute()
const router = useRouter()
const id = computed(() => route.params.id as string)

const members = useMembersStore()
const schemes = useSchemeInstancesStore()

const member = ref<MemberRow | undefined>(undefined)
const detailLoading = ref(true)

const tab = ref<'overview' | 'ledger' | 'schemes' | 'guaji'>('overview')

const guajiRows = ref<AdminGuajiAccountRow[]>([])
const guajiLoading = ref(false)
let guajiLoaded = false

const dateRange = ref<[string, string]>([todayYmd(), todayYmd()])
const flowType = ref<FundFlowType>('all')
const currency = ref<FundCurrency>('all')
const fundRows = ref<MemberFundRecordRow[]>([])
const fundLoading = ref(false)
const fundReady = ref(false)
const fundPage = ref(1)
const fundPageSize = ref(10)
const fundTotal = ref(0)
const appliedFundQuery = ref<AppliedFundQuery | null>(null)

const flowTypeOptions: { value: FundFlowType; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'income', label: '收入' },
  { value: 'expense', label: '支出' },
]

const currencyOptions: { value: FundCurrency; label: string }[] = [
  { value: 'all', label: '全部币种' },
  ...PRIMARY_CURRENCIES.map((c) => ({ value: c as FundCurrency, label: c })),
]

async function loadGuajiAccounts(): Promise<void> {
  if (guajiLoading.value) return
  guajiLoading.value = true
  try {
    guajiRows.value = await fetchMemberGuajiAccounts(id.value)
    guajiLoaded = true
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '授权信息加载失败')
  } finally {
    guajiLoading.value = false
  }
}

watch(tab, (next) => {
  if (next === 'guaji' && !guajiLoaded) void loadGuajiAccounts()
})

const schemeStatusFilter = ref<string>('')
const schemePageSize = ref(10)
const schemePage = ref(1)

const memberSchemes = computed(() =>
  member.value ? schemes.forMember(member.value.id) : [],
)

const filteredMemberSchemes = computed(() => {
  if (!schemeStatusFilter.value) return memberSchemes.value
  return memberSchemes.value.filter((s) => s.status === schemeStatusFilter.value)
})

const pagedMemberSchemes = computed(() => {
  const start = (schemePage.value - 1) * schemePageSize.value
  return filteredMemberSchemes.value.slice(start, start + schemePageSize.value)
})

const drawerVisible = ref(false)
const selectedScheme = ref<SchemeInstanceRow | null>(null)

async function reloadFundRecords(): Promise<void> {
  if (!member.value || !appliedFundQuery.value) return
  fundLoading.value = true
  try {
    const q = appliedFundQuery.value
    const result = await members.loadFundRecords(member.value.id, {
      dateFrom: q.dateFrom,
      dateTo: q.dateTo,
      flowType: q.flowType,
      currency: q.currency,
      page: fundPage.value,
      pageSize: fundPageSize.value,
    })
    fundRows.value = result.items
    fundTotal.value = result.total
    fundReady.value = true
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载资金记录失败')
    fundRows.value = []
    fundTotal.value = 0
  } finally {
    fundLoading.value = false
  }
}

async function runFundSearch(auto = false): Promise<void> {
  if (!dateRange.value[0] || !dateRange.value[1]) {
    if (!auto) ElMessage.warning('请选择日期区间')
    return
  }
  fundPage.value = 1
  appliedFundQuery.value = {
    dateFrom: dateRange.value[0],
    dateTo: dateRange.value[1],
    flowType: flowType.value,
    currency: currency.value,
  }
  await reloadFundRecords()
}

async function onFundSearch(): Promise<void> {
  await runFundSearch(false)
}

async function loadMemberDetail() {
  detailLoading.value = true
  member.value = undefined
  try {
    member.value = await members.loadDetail(id.value)
    await schemes.hydrate()
    if (tab.value === 'ledger') await runFundSearch(true)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载会员详情失败')
    void router.replace({ name: 'member-list' })
  } finally {
    detailLoading.value = false
  }
}
onMounted(() => {
  void loadMemberDetail()
})

watch(id, () => {
  tab.value = 'overview'
  fundReady.value = false
  fundRows.value = []
  appliedFundQuery.value = null
  void loadMemberDetail()
})

watch(tab, (t) => {
  if (t === 'ledger' && !fundReady.value) void runFundSearch(true)
})

watch(fundPage, () => {
  if (fundReady.value && appliedFundQuery.value) void reloadFundRecords()
})

watch(schemeStatusFilter, () => {
  schemePage.value = 1
})

watch(
  () => member.value?.id,
  () => {
    dateRange.value = [todayYmd(), todayYmd()]
    flowType.value = 'all'
    currency.value = 'all'
    fundReady.value = false
    fundRows.value = []
    fundPage.value = 1
    fundTotal.value = 0
    appliedFundQuery.value = null
    schemeStatusFilter.value = ''
    schemePage.value = 1
  },
)
function fmt(iso: string) {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(iso),
  )
}

function fmtMoney(v: number) {
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function fmtDelta(v: number) {
  const s = fmtMoney(Math.abs(v))
  return v >= 0 ? `+${s}` : `-${s}`
}

function fmtFundTime(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('zh-CN', { hour12: false })
}
function applyMemberRow(row: MemberRow) {
  member.value = row
}

async function onToggleFreeze() {
  if (!member.value) return
  const next = member.value.status === '冻结' ? '解冻' : '冻结'
  const ok = await adminConfirmDialog({
    title: `${next}账号`,
    message:
      next === '冻结'
        ? '确认冻结该会员？将同时暂停其所有运行中/待开启的方案。'
        : '确认解冻该会员？',
    tone: 'warning',
  })
  if (!ok) return
  try {
    const res = await postMemberOp(member.value.id, { action: 'toggle_freeze' })
    applyMemberRow(res.member)
    ElMessage.success(res.message ?? `已${next}`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function openHistory(row: SchemeInstanceRow) {
  selectedScheme.value = row
  drawerVisible.value = true
}

function goSchemeMonitor() {
  router.push({ name: 'scheme-monitor' })
}

async function onSoftStop(row: { id: string; status: string }) {
  if (row.status !== '运行中') {
    ElMessage.warning('仅「运行中」实例可强停')
    return
  }
  const ok = await adminConfirmDialog({
    title: '强停方案',
    message: `确认对实例 ${row.id} 执行强制软停？`,
    tone: 'warning',
    confirmText: '强停',
  })
  if (!ok) return
  try {
    const ok = await schemes.softStop(row.id)
    ElMessage[ok ? 'success' : 'error'](ok ? '已封停，已写入审计日志' : '操作失败')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
</script>

<template>
  <div v-loading="detailLoading">
    <template v-if="member">
      <div style="display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; flex-wrap: wrap">

        <el-button text type="primary" @click="router.push({ name: 'member-list' })">← 返回列表</el-button>

        <h1 class="admin-page-title" style="margin: 0">会员详情 · {{ member.account }}</h1>

      </div>



      <el-tabs v-model="tab">

        <el-tab-pane label="概况" name="overview">

          <el-descriptions :column="2" border>

            <el-descriptions-item label="会员ID">{{ member.id }}</el-descriptions-item>

            <el-descriptions-item label="状态">{{ member.status }}</el-descriptions-item>

            <el-descriptions-item label="USDT 余额">{{ fmtMoney(member.guajiBalances?.usdt ?? 0) }}</el-descriptions-item>

            <el-descriptions-item label="TRX 余额">{{ fmtMoney(member.guajiBalances?.trx ?? 0) }}</el-descriptions-item>

            <el-descriptions-item label="CNY 余额">{{ fmtMoney(member.guajiBalances?.cny ?? 0) }}</el-descriptions-item>

            <el-descriptions-item label="方案条数">{{ memberSchemes.length }}</el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ fmt(member.registeredAt) }}</el-descriptions-item>

            <el-descriptions-item label="最近登录">{{ fmt(member.lastLoginAt) }}</el-descriptions-item>

          </el-descriptions>



          <div style="margin-top: 1.25rem">

            <div style="font-weight: 600; margin-bottom: 0.5rem">运营动作</div>

            <div style="display: flex; flex-wrap: wrap; gap: 0.5rem">

              <el-button size="small" type="warning" @click="onToggleFreeze">冻结 / 解冻</el-button>

            </div>

          </div>

        </el-tab-pane>



        <el-tab-pane label="余额与帐变" name="ledger" v-loading="fundLoading">


          <div class="fund-balances">
            <div class="fund-balance-item">
              <span class="fund-balance-label">USDT</span>
              <span class="fund-balance-value">{{ fmtMoney(member.guajiBalances?.usdt ?? 0) }}</span>
            </div>
            <div class="fund-balance-item">
              <span class="fund-balance-label">TRX</span>
              <span class="fund-balance-value">{{ fmtMoney(member.guajiBalances?.trx ?? 0) }}</span>
            </div>
            <div class="fund-balance-item">
              <span class="fund-balance-label">CNY</span>
              <span class="fund-balance-value">{{ fmtMoney(member.guajiBalances?.cny ?? 0) }}</span>
            </div>
          </div>

          <div class="fund-toolbar">
            <div class="fund-filter">
              <span class="fund-filter-label">日期</span>
              <el-date-picker v-model="dateRange" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期"
                end-placeholder="结束日期" style="width: min(100%, 280px)" />
            </div>
            <div class="fund-filter">
              <span class="fund-filter-label">类型</span>
              <el-select v-model="flowType" class="fund-filter-select">
                <el-option v-for="o in flowTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </div>
            <div class="fund-filter">
              <span class="fund-filter-label">币种</span>
              <el-select v-model="currency" class="fund-filter-select">
                <el-option v-for="o in currencyOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </div>
            <el-button type="primary" :loading="fundLoading" @click="onFundSearch">查询</el-button>
          </div>

          <el-table :data="fundRows" stripe style="width: 100%">

            <el-table-column prop="schemeName" label="方案名称" min-width="140" show-overflow-tooltip />

            <el-table-column label="变动金额" min-width="120" align="right">

              <template #default="{ row }">

                <span
                  :style="{ color: row.flowTypeCode === 'income' ? 'var(--el-color-success)' : 'var(--el-color-danger)' }">

                  {{ fmtDelta(row.amount) }}

                </span>

              </template>

            </el-table-column>

            <el-table-column prop="flowType" label="类型" min-width="80" />

            <el-table-column prop="currency" label="币种" min-width="72" />

            <el-table-column label="时间" min-width="160">

              <template #default="{ row }">{{ fmtFundTime(row.time) }}</template>

            </el-table-column>

            <el-table-column label="变化后余额" min-width="120" align="right">

              <template #default="{ row }">{{ fmtMoney(row.balanceAfter) }}</template>

            </el-table-column>

            <el-table-column prop="ledgerNo" label="流水号" min-width="120" show-overflow-tooltip />

          </el-table>

          <el-empty v-if="fundReady && !fundRows.length && !fundLoading" description="暂无资金记录"
            style="margin-top: 1rem" />

          <div class="pager">
            <el-pagination v-model:current-page="fundPage" :page-size="fundPageSize" layout="total, prev, pager, next"
              :total="fundTotal" />
          </div>

        </el-tab-pane>

        <el-tab-pane label="方案" name="schemes">

          <div style="

            display: flex;

            flex-wrap: wrap;

            gap: 0.75rem;

            margin-bottom: 0.75rem;

            align-items: center;

            justify-content: space-between;

          ">



            <el-button link type="primary" @click="goSchemeMonitor">打开全站方案监控</el-button>

          </div>

          <div style="display: flex; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 0.75rem">

            <el-select v-model="schemeStatusFilter" clearable placeholder="状态" style="width: 120px">

              <el-option label="待开启" value="待开启" />

              <el-option label="运行中" value="运行中" />

              <el-option label="已暂停" value="已暂停" />

              <el-option label="已封停" value="已封停" />

            </el-select>

          </div>

          <el-table :data="pagedMemberSchemes" stripe style="width: 100%">

            <el-table-column prop="id" label="实例ID" min-width="108" />

            <el-table-column label="方案名称" min-width="140" show-overflow-tooltip>

              <template #default="{ row }">{{ row.settings.schemeName }}</template>

            </el-table-column>

            <el-table-column prop="kind" label="类型" min-width="88" />

            <el-table-column label="投注通道" min-width="88">

              <template #default="{ row }">{{ row.simBet ? '模拟' : '正式' }}</template>

            </el-table-column>

            <el-table-column prop="lotteryLabel" label="彩种" min-width="120" />

            <el-table-column prop="refId" label="业务主键" min-width="120" />

            <el-table-column prop="status" label="状态" min-width="88" />

            <el-table-column label="创建" min-width="140">

              <template #default="{ row }">{{ fmt(row.createdAt) }}</template>

            </el-table-column>

            <el-table-column label="操作" min-width="160" fixed="right">

              <template #default="{ row }">

                <el-button link type="primary" @click="openHistory(row)">投注与盈亏</el-button>

                <el-button link type="primary" :disabled="row.status !== '运行中'" @click="onSoftStop(row)">

                  强停

                </el-button>

              </template>

            </el-table-column>

          </el-table>

          <div style="display: flex; justify-content: flex-end; margin-top: 1rem">

            <el-pagination v-model:current-page="schemePage" :page-size="schemePageSize"
              layout="total, prev, pager, next" :total="filteredMemberSchemes.length" />

          </div>

        </el-tab-pane>

        <el-tab-pane label="授权账号" name="guaji" v-loading="guajiLoading">

          <p style="margin: 0 0 1rem; font-size: 13px; color: var(--el-text-color-secondary)">
            第三方挂机授权 <strong>只读</strong>（无代绑 / 解绑）；展示当前绑定与最近同步状态。
          </p>

          <el-empty v-if="!guajiLoading && !guajiRows.length" description="该会员暂无第三方授权绑定" />

          <el-table v-else :data="guajiRows" stripe style="width: 100%">

            <el-table-column prop="guajiUsername" label="第三方用户名" min-width="140" />

            <el-table-column label="是否启用" min-width="96">

              <template #default="{ row }">

                <el-tag :type="row.isActive ? 'success' : 'info'" size="small">

                  {{ row.isActive ? '启用中' : '未启用' }}

                </el-tag>

              </template>

            </el-table-column>

            <el-table-column label="绑定时间" min-width="160">

              <template #default="{ row }">{{ fmt(row.boundAt) }}</template>

            </el-table-column>

            <el-table-column label="最近同步" min-width="160">

              <template #default="{ row }">{{ row.lastSyncAt ? fmt(row.lastSyncAt) : '—' }}</template>

            </el-table-column>

            <el-table-column label="最后投注" min-width="160">

              <template #default="{ row }">{{ row.lastBetAt ? fmt(row.lastBetAt) : '—' }}</template>

            </el-table-column>

            <el-table-column prop="lastTokenError" label="最近 Token 失效原因" min-width="200" show-overflow-tooltip>

              <template #default="{ row }">{{ row.lastTokenError || '—' }}</template>

            </el-table-column>

          </el-table>

        </el-tab-pane>

      </el-tabs>



      <SchemeHistoryDrawer v-model="drawerVisible" :scheme="selectedScheme" />
    </template>
  </div>
</template>

<style scoped>
.fund-desc {
  margin: 0 0 0.75rem;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.fund-balances {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem 2rem;
  margin-bottom: 1rem;
  padding: 0.875rem 1rem;
  background: var(--el-fill-color-blank);
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(15, 23, 42, 0.06);
}

.fund-balance-item {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  min-width: 8rem;
}

.fund-balance-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.fund-balance-value {
  font-size: 15px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.fund-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.25rem;
  margin-bottom: 1rem;
  align-items: center;
}

.fund-filter {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.fund-filter-label {
  flex-shrink: 0;
  font-size: 13px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}

.fund-filter-select {
  width: 120px;
  min-width: 120px;
  flex-shrink: 0;
}

.fund-filter-select :deep(.el-select__wrapper) {
  width: 100%;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
