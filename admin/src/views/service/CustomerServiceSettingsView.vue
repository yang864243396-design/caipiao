<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'
import {
  deleteCustomerServiceAgent,
  fetchCustomerServiceAgents,
  saveCustomerServiceAgent,
  type CustomerServiceAgent,
} from '@/api/customerService'

const rows = ref<CustomerServiceAgent[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const editing = ref<CustomerServiceAgent | null>(null)
const isNew = ref(false)

async function loadRows() {
  loading.value = true
  try {
    rows.value = await fetchCustomerServiceAgents()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
    rows.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadRows()
})

function openNew() {
  const sort = (rows.value.reduce((m, r) => Math.max(m, r.sort), 0) || 0) + 1
  editing.value = {
    id: `cs_${Date.now().toString(36)}`,
    name: '',
    tgLink: '',
    workHours: '',
    sort,
    enabled: true,
  }
  isNew.value = true
  dialogVisible.value = true
}

function openEdit(row: CustomerServiceAgent) {
  editing.value = { ...row }
  isNew.value = false
  dialogVisible.value = true
}

async function onSave() {
  const row = editing.value
  if (!row) return
  if (!row.name.trim()) {
    ElMessage.warning('请输入客服姓名')
    return
  }
  if (!row.tgLink.trim()) {
    ElMessage.warning('请输入 Telegram 链接或 @用户名')
    return
  }
  try {
    await saveCustomerServiceAgent({
      ...row,
      id: row.id.trim(),
      name: row.name.trim(),
      tgLink: row.tgLink.trim(),
      workHours: row.workHours.trim(),
      sort: Number(row.sort) || 0,
    })
    ElMessage.success('已保存')
    dialogVisible.value = false
    await loadRows()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function onDelete(row: CustomerServiceAgent) {
  const ok = await adminConfirmDialog({
    title: '确认',
    message: `确定删除客服「${row.name}」？`,
    tone: 'warning',
  })
  if (!ok) return
  try {
    await deleteCustomerServiceAgent(row.id)
    ElMessage.success('已删除')
    await loadRows()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
</script>

<template>
  <div class="admin-page">
    <div class="admin-page-head">
      <h1 class="admin-page-title">客服设置</h1>
      <el-button type="primary" @click="openNew">新增客服</el-button>
    </div>
    <p class="admin-page-desc">配置会员端「联系客服」弹窗展示的客服人员；Telegram 可填完整链接或 @用户名。</p>

    <el-table v-loading="loading" :data="rows" stripe class="admin-table">
      <el-table-column prop="name" label="姓名" min-width="120" />
      <el-table-column prop="tgLink" label="Telegram" min-width="200" show-overflow-tooltip />
      <el-table-column prop="workHours" label="上班时间" min-width="160" show-overflow-tooltip />
      <el-table-column prop="sort" label="排序" width="80" align="center" />
      <el-table-column label="启用" width="88" align="center">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialogVisible" :title="isNew ? '新增客服' : '编辑客服'" width="520px" destroy-on-close>
      <el-form v-if="editing" label-width="96px" @submit.prevent="onSave">
        <el-form-item label="姓名" required>
          <el-input v-model="editing.name" maxlength="128" placeholder="客服姓名" />
        </el-form-item>
        <el-form-item label="Telegram" required>
          <el-input v-model="editing.tgLink" maxlength="512" placeholder="https://t.me/xxx 或 @username" />
        </el-form-item>
        <el-form-item label="上班时间">
          <el-input v-model="editing.workHours" maxlength="256" placeholder="如 09:00-22:00（UTC+8）" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="editing.sort" :min="0" :max="9999" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="editing.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </AdminDialog>
  </div>
</template>

<style scoped>
.admin-page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.admin-page-desc {
  margin: 0 0 1rem;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>
