<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { fetchDashboardKpi, type DashboardKpi } from '@/api/dashboard'
import { startDashboardKpiSync } from '@/composables/useAdminQueueSync'

const loading = ref(false)
const kpi = ref<DashboardKpi | null>(null)

let stopSync: (() => void) | null = null

function formatMoney(n: number) {
  return `¥ ${Math.abs(n).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

function formatSignedMoney(n: number) {
  const sign = n >= 0 ? '+' : '-'
  return `${sign}${formatMoney(n)}`
}

const display = computed(() => kpi.value)

const kpis = computed(() => {
  const d = display.value
  if (!d) return []
  return [
    { key: 'recharge', label: '今日成功充值', value: formatMoney(d.todayRecharge), hint: '自然日成功入账（UTC+8）' },
    { key: 'bet', label: '今日投注额', value: formatMoney(d.todayBetVolume), hint: '非撤单有效投注' },
    {
      key: 'member_total_pnl',
      label: '会员总盈亏',
      value: formatSignedMoney(d.memberTotalPnl),
      hint: '已结算订单会员盈亏合计',
    },
    {
      key: 'running_formal',
      label: '正式运行中方案',
      value: String(d.runningSchemesReal),
      hint: '正式投注且运行中的方案',
    },
    {
      key: 'running_sim',
      label: '模拟运行中方案',
      value: String(d.runningSchemesSim),
      hint: '模拟投注且运行中的方案',
    },
    {
      key: 'reg7',
      label: '近 7 日注册会员',
      value: String(d.registrationsLast7Days),
      hint: '滚动 7 天',
    },
  ]
})

async function loadKpi(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    kpi.value = await fetchDashboardKpi()
  } finally {
    if (showLoading) loading.value = false
  }
}

onMounted(() => {
  stopSync = startDashboardKpiSync(() => {
    void loadKpi(false)
  })
})

onUnmounted(() => {
  stopSync?.()
})
</script>

<template>
  <div v-loading="loading">
    <h1 class="admin-page-title">仪表盘</h1>
    <p class="admin-page-desc">
      KPI 来自 <code>GET /admin/dashboard/kpi</code>；提现/方案变更时 WS 自动刷新（降级 15s 轮询）。
    </p>

    <div class="admin-kpi-grid">
      <div v-for="k in kpis" :key="k.key" class="admin-kpi-card">
        <div class="admin-kpi-label">{{ k.label }}</div>
        <div class="admin-kpi-value">{{ k.value }}</div>
        <div class="admin-kpi-hint">{{ k.hint }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-page-desc {
  margin: 0 0 1.25rem;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.admin-kpi-hint {
  margin-top: 0.5rem;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
