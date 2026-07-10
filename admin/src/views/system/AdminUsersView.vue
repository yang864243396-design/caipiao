<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'
import { useAdminUsersStore, type AdminUser } from '@/stores/adminUsers'
import { useAdminRolesStore } from '@/stores/adminRoles'

const usersStore = useAdminUsersStore()
const rolesStore = useAdminRolesStore()
const { users, loading } = storeToRefs(usersStore)
const { roles } = storeToRefs(rolesStore)

onMounted(() => {
  void usersStore.hydrate()
  void rolesStore.hydrate()
})

const dialog = ref(false)
const editing = ref<AdminUser | null>(null)
const form = reactive({
  account: '',
  displayName: '',
  roleId: 'r_fin_approve',
  status: 'active' as 'active' | 'disabled',
  password: '',
})

const roleOptions = computed(() => roles.value.map((r) => ({ value: r.id, label: r.name })))

function resetForm() {
  form.account = ''
  form.displayName = ''
  form.roleId = roleOptions.value[0]?.value ?? 'r_fin_approve'
  form.status = 'active'
  form.password = ''
}

function openNew() {
  editing.value = null
  resetForm()
  dialog.value = true
}

function openEdit(row: AdminUser) {
  editing.value = row
  form.account = row.account
  form.displayName = row.displayName
  form.roleId = row.roleId
  form.status = row.status
  form.password = ''
  dialog.value = true
}

function formatTime(iso?: string) {
  if (!iso) return '—'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString('zh-CN')
}

async function saveForm() {
  const displayName = form.displayName.trim()
  const roleId = form.roleId.trim()
  if (!displayName || !roleId) {
    ElMessage.warning('请填写显示名并选择角色')
    return
  }
  try {
    if (editing.value) {
      const body = {
        displayName,
        roleId,
        status: form.status,
        password: form.password.trim() || undefined,
      }
      await usersStore.saveUser(editing.value.id, body)
      ElMessage.success('已保存')
    } else {
      const account = form.account.trim()
      const password = form.password.trim()
      if (!account) {
        ElMessage.warning('请填写登录账号')
        return
      }
      if (password.length < 6) {
        ElMessage.warning('初始密码至少 6 位')
        return
      }
      await usersStore.createUser({
        account,
        displayName,
        roleId,
        status: form.status,
        password,
      })
      ElMessage.success('已创建')
    }
    dialog.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function remove(row: AdminUser) {
  if (row.account === 'admin') {
    ElMessage.warning('不可删除内置超级管理员账号')
    return
  }
  const ok = await adminConfirmDialog({
    title: '确认',
    message: `删除账号「${row.account}」？`,
    tone: 'warning',
  })
  if (!ok) return
  try {
    const deleted = await usersStore.removeUser(row.id)
    ElMessage[deleted ? 'success' : 'warning'](deleted ? '已删除' : '不可删除该账号')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
</script>

<template>
  <div>
    <h1 class="admin-page-title">Admin 账号管理</h1>

    <div style="margin-bottom: 1rem">
      <el-button type="primary" @click="openNew">新增账号</el-button>
    </div>

    <el-table v-loading="loading" :data="users" stripe style="width: 100%">
      <el-table-column prop="account" label="登录账号" min-width="120" />
      <el-table-column prop="displayName" label="显示名" min-width="120" />
      <el-table-column label="绑定角色" min-width="140">
        <template #default="{ row }">
          {{ row.roleName || row.roleId }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status === 'active' ? '正常' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="最近登录" min-width="160">
        <template #default="{ row }">
          <span style="font-size: 12px; color: var(--el-text-color-secondary)">
            {{ formatTime(row.lastLoginAt) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" :disabled="row.account === 'admin'" @click="remove(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog v-model="dialog" :title="editing ? '编辑账号' : '新增账号'" width="min(92vw, 480px)">
      <el-form label-position="top">
        <el-form-item label="登录账号">
          <el-input v-model="form.account" :disabled="!!editing" placeholder="英文字母/数字" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="form.displayName" />
        </el-form-item>
        <el-form-item label="绑定角色">
          <el-select v-model="form.roleId" style="width: 100%">
            <el-option v-for="opt in roleOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio value="active">正常</el-radio>
            <el-radio value="disabled">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="editing ? '重置密码（留空则不修改）' : '初始密码'">
          <el-input v-model="form.password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="saveForm">保存</el-button>
      </template>
    </AdminDialog>
  </div>
</template>
