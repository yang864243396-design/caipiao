<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import { fetchDailyLotteryReport, type DailyLotteryRow, type DailyLotterySummary } from '@/api/reports'
import { useLotteryCatalogStore } from '@/stores/lotteryCatalog'

function todayYmd(): string {
  const d = new Date()
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const catalog = useLotteryCatalogStore()
const { rows: lotteryRows } = storeToRefs(catalog)

const dateRange = ref<[string, string]>([todayYmd(), todayYmd()])
const lotteryCode = ref('')
const loading = ref(false)
const summary = ref<DailyLotterySummary>({
  betCount: 0,
  betAmountYuan: 0,
  platformPnlYuan: 0,
  dateFrom: '',
  dateTo: '',
})
const rows = ref<DailyLotteryRow[]>([])

const lotteryOptions = computed(() =>
  [...lotteryRows.value].sort((a, b) => a.sortOrder - b.sortOrder),
)

function fmtMoney(v: number) {
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function pnlClass(v: number) {
  if (v > 0) return 'pnl-pos'
  if (v < 0) return 'pnl-neg'
  return ''
}

async function load() {
  loading.value = true
  try {
    const res = await fetchDailyLotteryReport({
      dateFrom: dateRange.value[0],
      dateTo: dateRange.value[1],
      lotteryCode: lotteryCode.value || undefined,
    })
    summary.value = res.summary
    rows.value = res.items
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void catalog.hydrate()
  void load()
})
</script>

<template>
  <div>
    <h1 class="admin-page-title">经营报表</h1>

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
      <el-select v-model="lotteryCode" clearable filterable placeholder="全部彩种" style="width: 180px">
        <el-option
          v-for="lot in lotteryOptions"
          :key="lot.code"
          :label="lot.displayName"
          :value="lot.code"
        />
      </el-select>
      <el-button type="primary" :loading="loading" @click="load">查询</el-button>
    </div>

    <el-card v-loading="loading" shadow="never">
      <div class="report-summary">
        <div class="report-summary__item">
          <span class="report-summary__label">总投注笔数</span>
          <span class="report-summary__value">{{ summary.betCount.toLocaleString('zh-CN') }}</span>
        </div>
        <div class="report-summary__item">
          <span class="report-summary__label">总投注金额</span>
          <span class="report-summary__value">¥ {{ fmtMoney(summary.betAmountYuan) }}</span>
        </div>
        <div class="report-summary__item">
          <span class="report-summary__label">总盈亏</span>
          <span class="report-summary__value" :class="pnlClass(summary.platformPnlYuan)">
            ¥ {{ fmtMoney(summary.platformPnlYuan) }}
          </span>
        </div>
      </div>

      <el-table
        :data="rows"
        stripe
        style="width: 100%; margin-top: 1.25rem"
        :default-sort="{ prop: 'date', order: 'descending' }"
      >
        <template #empty>
          <span style="font-size: 13px; color: var(--el-text-color-secondary)">所选条件内暂无数据</span>
        </template>
        <el-table-column prop="date" label="日期" min-width="120" sortable />
        <el-table-column prop="lottery" label="彩种" min-width="140" />
        <el-table-column prop="betCount" label="投注笔数" min-width="100" align="center" sortable />
        <el-table-column label="投注金额" min-width="130" align="right" sortable prop="betAmountYuan">
          <template #default="{ row }">{{ fmtMoney(row.betAmountYuan) }}</template>
        </el-table-column>
        <el-table-column label="盈亏" min-width="130" align="right" sortable prop="platformPnlYuan">
          <template #default="{ row }">
            <span :class="pnlClass(row.platformPnlYuan)">{{ fmtMoney(row.platformPnlYuan) }}</span>
          </template>
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

.report-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 2.5rem;
}

.report-summary__item {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.report-summary__label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.report-summary__value {
  font-size: 22px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.pnl-pos {
  color: var(--el-color-danger);
}

.pnl-neg {
  color: var(--el-color-success);
}
</style>
