<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import {
  fetchAdminBetOrders,
  fetchAdminChaseOrders,
  type AdminBetOrderRow,
  type AdminChaseOrderRow,
} from '@/api/orders'
import { useLotteryCatalogStore } from '@/stores/lotteryCatalog'

const tab = ref<'bet' | 'chase'>('bet')
const filterIssueNo = ref('')
const filterMemberAccount = ref('')
const filterSchemeName = ref('')
const filterLotteryCode = ref('')
const filterChaseNo = ref('')
const filterChaseMemberAccount = ref('')
const filterChaseStatus = ref('')
const filterChaseLotteryCode = ref('')
const pageSize = ref(10)
const currentBetPage = ref(1)
const currentChasePage = ref(1)
const loading = ref(false)

const catalog = useLotteryCatalogStore()

const betRows = ref<AdminBetOrderRow[]>([])
const betTotal = ref(0)
const chaseRows = ref<AdminChaseOrderRow[]>([])
const chaseTotal = ref(0)

const lotteryOptions = computed(() =>
  [...catalog.rows]
    .sort((a, b) => a.sortOrder - b.sortOrder)
    .map((row) => ({ value: row.code, label: row.displayName })),
)

const pagedBets = computed(() => betRows.value)
const pagedChase = computed(() => chaseRows.value)
const betTableTotal = computed(() => betTotal.value)
const chaseTableTotal = computed(() => chaseTotal.value)

async function reloadBets() {
  loading.value = true
  try {
    const res = await fetchAdminBetOrders({
      issueNo: filterIssueNo.value.trim(),
      memberAccount: filterMemberAccount.value.trim(),
      schemeName: filterSchemeName.value.trim(),
      lotteryCode: filterLotteryCode.value,
      page: currentBetPage.value,
      pageSize: pageSize.value,
    })
    betRows.value = res.items
    betTotal.value = res.total
  } finally {
    loading.value = false
  }
}

async function reloadChase() {
  loading.value = true
  try {
    const res = await fetchAdminChaseOrders({
      chaseNo: filterChaseNo.value.trim(),
      memberAccount: filterChaseMemberAccount.value.trim(),
      status: filterChaseStatus.value,
      lotteryCode: filterChaseLotteryCode.value,
      page: currentChasePage.value,
      pageSize: pageSize.value,
    })
    chaseRows.value = res.items
    chaseTotal.value = res.total
  } finally {
    loading.value = false
  }
}

async function reloadActive() {
  if (tab.value === 'bet') await reloadBets()
  else await reloadChase()
}

function onBetSearch() {
  currentBetPage.value = 1
  void reloadBets()
}

function onBetReset() {
  filterIssueNo.value = ''
  filterMemberAccount.value = ''
  filterSchemeName.value = ''
  filterLotteryCode.value = ''
  currentBetPage.value = 1
  void reloadBets()
}

function onChaseSearch() {
  currentChasePage.value = 1
  void reloadChase()
}

function onChaseReset() {
  filterChaseNo.value = ''
  filterChaseMemberAccount.value = ''
  filterChaseStatus.value = ''
  filterChaseLotteryCode.value = ''
  currentChasePage.value = 1
  void reloadChase()
}

onMounted(() => {
  void catalog.hydrate()
  void reloadActive()
})

watch(tab, () => {
  void reloadActive()
})

watch(currentBetPage, () => {
  if (tab.value === 'bet') void reloadBets()
})

watch(currentChasePage, () => {
  if (tab.value === 'chase') void reloadChase()
})

function fmtTime(iso: string) {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(
    new Date(iso),
  )
}

function betResultLabel(status: string): string {
  switch (status) {
    case 'hit':
      return '中'
    case 'miss':
      return '挂'
    case 'pending':
      return '待开奖'
    case 'cancel':
      return '撤单'
    default:
      return status || '—'
  }
}

function betResultTagType(status: string): 'success' | 'info' | 'warning' | '' {
  switch (status) {
    case 'hit':
      return 'success'
    case 'miss':
      return 'info'
    case 'pending':
      return 'warning'
    default:
      return ''
  }
}

function chaseStatusLabel(status: string): string {
  switch (status) {
    case 'running':
      return '追号中'
    case 'completed':
      return '已完成'
    case 'cancelled':
      return '已取消'
    default:
      return status || '—'
  }
}

function chaseStatusTagType(status: string): 'success' | 'info' | 'warning' | '' {
  switch (status) {
    case 'running':
      return 'warning'
    case 'completed':
      return 'success'
    case 'cancelled':
      return 'info'
    default:
      return ''
  }
}
</script>

<template>
  <div v-loading="loading">
    <div class="page-head">
      <div>
        <h1 class="admin-page-title">投注与追号</h1>
      </div>
    </div>

    <el-tabs v-model="tab">
      <el-tab-pane label="投注订单" name="bet">
        <div class="toolbar toolbar--bet">
          <el-input v-model="filterIssueNo" clearable placeholder="期号" class="toolbar-field"
            @keyup.enter="onBetSearch" />
          <el-input v-model="filterMemberAccount" clearable placeholder="会员账号" class="toolbar-field"
            @keyup.enter="onBetSearch" />
          <el-input v-model="filterSchemeName" clearable placeholder="方案" class="toolbar-field"
            @keyup.enter="onBetSearch" />
          <el-select v-model="filterLotteryCode" clearable filterable placeholder="彩种"
            class="toolbar-field toolbar-field--lottery">
            <el-option v-for="opt in lotteryOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
          <div class="toolbar-actions">
            <el-button type="primary" @click="onBetSearch">查询</el-button>
            <el-button @click="onBetReset">重置</el-button>
          </div>
        </div>

        <el-table :data="pagedBets" stripe style="width: 100%">
          <el-table-column prop="orderNo" label="注单编号" min-width="160" show-overflow-tooltip />
          <el-table-column prop="issueNo" label="期号" min-width="140" show-overflow-tooltip />
          <el-table-column prop="member" label="会员账号" min-width="108" />
          <el-table-column prop="lottery" label="彩种" min-width="120" />
          <el-table-column prop="schemeName" label="方案" min-width="120" show-overflow-tooltip />
          <el-table-column label="投注金额" min-width="100" align="right">
            <template #default="{ row }">{{ row.amount.toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="返奖金额" min-width="100" align="right">
            <template #default="{ row }">
              {{ row.resultStatus === 'hit' ? row.payoutAmount.toFixed(2) : '—' }}
            </template>
          </el-table-column>
          <el-table-column label="状态" min-width="88" align="center">
            <template #default="{ row }">
              <el-tag :type="betResultTagType(row.resultStatus)" size="small" effect="light">
                {{ betResultLabel(row.resultStatus) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间" min-width="160">
            <template #default="{ row }">{{ fmtTime(row.created) }}</template>
          </el-table-column>
        </el-table>
        <div class="pager">
          <el-pagination v-model:current-page="currentBetPage" :page-size="pageSize" layout="total, prev, pager, next"
            :total="betTableTotal" />
        </div>
      </el-tab-pane>

      <el-tab-pane label="追号任务" name="chase">
        <div class="toolbar toolbar--bet">
          <el-input v-model="filterChaseNo" clearable placeholder="追号单号" class="toolbar-field"
            @keyup.enter="onChaseSearch" />
          <el-input v-model="filterChaseMemberAccount" clearable placeholder="会员账号" class="toolbar-field"
            @keyup.enter="onChaseSearch" />
          <el-select v-model="filterChaseStatus" clearable placeholder="状态" class="toolbar-field">
            <el-option label="追号中" value="running" />
            <el-option label="已完成" value="completed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
          <el-select v-model="filterChaseLotteryCode" clearable filterable placeholder="彩种"
            class="toolbar-field toolbar-field--lottery">
            <el-option v-for="opt in lotteryOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
          <div class="toolbar-actions">
            <el-button type="primary" @click="onChaseSearch">查询</el-button>
            <el-button @click="onChaseReset">重置</el-button>
          </div>
        </div>
        <el-table :data="pagedChase" stripe style="width: 100%">
          <el-table-column prop="chaseNo" label="追号单号" min-width="140" show-overflow-tooltip />
          <el-table-column prop="member" label="会员账号" min-width="108" />
          <el-table-column prop="lottery" label="彩种" min-width="120" />
          <el-table-column label="追号进度" min-width="100" align="center">
            <template #default="{ row }">{{ row.doneIssues }}/{{ row.totalIssues }}</template>
          </el-table-column>
          <el-table-column prop="periodsLeft" label="剩余期数" min-width="96" align="center" />
          <el-table-column label="追号金额" min-width="100" align="right">
            <template #default="{ row }">{{ row.amount.toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="状态" min-width="88" align="center">
            <template #default="{ row }">
              <el-tag :type="chaseStatusTagType(row.status)" size="small" effect="light">
                {{ chaseStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="开始时间" min-width="160">
            <template #default="{ row }">{{ fmtTime(row.created) }}</template>
          </el-table-column>
        </el-table>
        <div class="pager">
          <el-pagination v-model:current-page="currentChasePage" :page-size="pageSize" layout="total, prev, pager, next"
            :total="chaseTableTotal" />
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
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
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 1rem;
}

.toolbar--bet {
  align-items: flex-end;
}

.toolbar-field {
  width: min(100%, 180px);
}

.toolbar-field--lottery {
  width: min(100%, 200px);
}

.toolbar-actions {
  display: flex;
  gap: 0.5rem;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
