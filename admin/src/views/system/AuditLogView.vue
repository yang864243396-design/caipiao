<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useAuditLogStore } from '@/stores/auditLog'

const audit = useAuditLogStore()
const { list } = storeToRefs(audit)

onMounted(() => {
  void audit.hydrate()
})
const pageSize = ref(10)
const currentPage = ref(1)

const pagedRows = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return list.value.slice(start, start + pageSize.value)
})
</script>

<template>
  <div>
    <h1 class="admin-page-title">操作审计</h1>
    <p style="margin: 0 0 1rem; font-size: 13px; color: var(--el-text-color-secondary)">
      §5 P4 / §30.5：与审批、方案强停/改参/设置变更等写操作并列留痕；方案运营动作实时追加至列表顶部。
    </p>
    <el-table :data="pagedRows" stripe style="width: 100%">
      <el-table-column prop="id" label="审计ID" min-width="100" />
      <el-table-column prop="time" label="时间" min-width="160" />
      <el-table-column prop="actor" label="操作者" min-width="96" />
      <el-table-column prop="action" label="动作" min-width="280" show-overflow-tooltip />
      <el-table-column prop="ip" label="IP" min-width="120" />
    </el-table>

    <div style="display: flex; justify-content: flex-end; margin-top: 1rem">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, prev, pager, next"
        :total="list.length"
      />
    </div>
  </div>
</template>
