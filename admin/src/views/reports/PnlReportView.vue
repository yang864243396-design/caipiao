<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchPnlReport, type PnlDailyRow } from '@/api/reports'

function todayYmd(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const dateRange = ref<[string, string]>([todayYmd(), todayYmd()])
const loading = ref(false)
const platformPnl = ref(0)
const validBet = ref(0)
const rows = ref<PnlDailyRow[]>([])

function fmtMoney(v: number) {
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

async function load() {
  loading.value = true
  try {
    const res = await fetchPnlReport({
      dateFrom: dateRange.value[0],
      dateTo: dateRange.value[1],
    })
    platformPnl.value = res.summary.platformPnlYuan
    validBet.value = res.summary.validBetYuan
    rows.value = res.items
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div>
    <h1 class="admin-page-title">盈亏报表</h1>

    <div class="report-toolbar">
      <div class="admin-filter-date">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          start-placeholder="开始"
          end-placeholder="结束"
        />
      </div>
      <el-button type="primary" :loading="loading" @click="load">查询</el-button>
    </div>

    <el-card v-loading="loading" shadow="never">
      <div style="font-variant-numeric: tabular-nums; line-height: 1.8">
        <div>平台盈亏：¥ {{ fmtMoney(platformPnl) }}</div>
        <div>有效投注：¥ {{ fmtMoney(validBet) }}</div>
      </div>
      <el-table v-if="rows.length" :data="rows" stripe style="width: 100%; margin-top: 1.25rem">
        <el-table-column prop="period" label="日期" min-width="120" />
        <el-table-column label="有效投注" min-width="120" align="right">
          <template #default="{ row }">{{ fmtMoney(row.validBetYuan) }}</template>
        </el-table-column>
        <el-table-column label="平台盈亏" min-width="120" align="right">
          <template #default="{ row }">{{ fmtMoney(row.platformPnlYuan) }}</template>
        </el-table-column>
      </el-table>

    </el-card>
  </div>
</template>

<style scoped>
.report-toolbar {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  align-items: center;
  margin-bottom: 1rem;
}
</style>
