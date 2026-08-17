<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchPublishedPlayRules, type PublishedPlayRule } from '@/api/playRules'

const loading = ref(false)
const error = ref('')
const rows = ref<PublishedPlayRule[]>([])

async function load() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await fetchPublishedPlayRules()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '加载规则列表失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <section>
    <div class="page-head">
      <div>
        <h1 class="admin-page-title">玩法规则</h1>
        <p>仅展示当前已发布的规则版本；本页不提供新增、编辑或停用操作。</p>
      </div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="state" />
    <el-empty v-else-if="!loading && rows.length === 0" description="暂无已发布规则" />
    <el-table v-else :data="rows" v-loading="loading" stripe style="width: 100%">
      <el-table-column label="目录定位" min-width="220">
        <template #default="{ row }">{{ row.templateCode }} / {{ row.typeId }} / {{ row.subId }}</template>
      </el-table-column>
      <el-table-column label="适用彩种" min-width="140">
        <template #default="{ row }">{{ row.lotteryCode || '目录默认' }}</template>
      </el-table-column>
      <el-table-column prop="evaluatorKey" label="评估器" min-width="150" />
      <el-table-column prop="ruleVersion" label="规则版本" width="100" align="center" />
      <el-table-column prop="evaluatorVersion" label="评估器版本" width="110" align="center" />
      <el-table-column label="策略" width="90" align="center">
        <template #default="{ row }"><el-tag :type="row.strategyEnabled ? 'success' : 'info'">{{ row.strategyEnabled ? '启用' : '关闭' }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="publishedAt" label="发布时间" min-width="170" />
    </el-table>
  </section>
</template>

<style scoped>
.page-head { display: flex; align-items: start; justify-content: space-between; gap: 1rem; margin-bottom: 1rem; }
.page-head p { margin: .35rem 0 0; color: var(--el-text-color-secondary); font-size: 13px; }
.state { margin-bottom: 1rem; }
@media (max-width: 768px) { .page-head { align-items: stretch; flex-direction: column; } }
</style>
