<script setup lang="ts">
import { ref, computed, watch, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'
import { useMembersStore, type MemberRow } from '@/stores/members'
import type { MemberSearchField, MemberStatusCode } from '@/api/members'

const router = useRouter()
const members = useMembersStore()
const { list, total, loading } = storeToRefs(members)

const searchField = ref<MemberSearchField>('account')
const keyword = ref('')
/** 点击「查询」后生效的条件；分页沿用此快照 */
const appliedQuery = ref<{ keyword: string; searchField: MemberSearchField }>({
  keyword: '',
  searchField: 'account',
})
const pageSize = ref(10)
const currentPage = ref(1)

const pagedRows = computed(() => list.value)
const tableTotal = computed(() => total.value)
const keywordPlaceholder = computed(() => {
  if (searchField.value === 'guajiAccount') return '请输入授权账号'
  if (searchField.value === 'id') return '请输入会员 ID（数字）'
  return '请输入会员账号'
})

const dialogVisible = ref(false)
const dialogSaving = ref(false)
const editing = ref<MemberRow | null>(null)
const form = reactive({
  account: '',
  password: '',
  status: 'active' as MemberStatusCode,
})

const dialogTitle = computed(() => (editing.value ? '编辑会员' : '新增会员'))

watch(
  () => form.status,
  (next, prev) => {
    if (!dialogVisible.value || !editing.value) return
    if (next === 'frozen' && prev !== 'frozen') {
      ElMessage.warning('禁用该用户会停止该用户所有正在运行的方案')
    }
  },
)

async function reload() {
  await members.loadList({
    keyword: appliedQuery.value.keyword,
    searchField: appliedQuery.value.searchField,
    page: currentPage.value,
    pageSize: pageSize.value,
  })
}

function onSearch() {
  currentPage.value = 1
  appliedQuery.value = {
    keyword: keyword.value.trim(),
    searchField: searchField.value,
  }
  void reload()
}

onMounted(() => {
  void reload()
})

watch(currentPage, () => {
  void reload()
})

function fmt(iso: string) {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(
    new Date(iso),
  )
}

function fmtMoney(v: number) {
  return v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function statusToCode(status: MemberRow['status']): MemberStatusCode {
  return status === '禁用' ? 'frozen' : 'active'
}

function openDetail(id: string) {
  router.push({ name: 'member-detail', params: { id } })
}

function resetForm() {
  form.account = ''
  form.password = ''
  form.status = 'active'
}

function openCreate() {
  editing.value = null
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: MemberRow) {
  editing.value = row
  form.account = row.account
  form.password = ''
  form.status = statusToCode(row.status)
  dialogVisible.value = true
}

async function saveForm() {
  const password = form.password.trim()
  if (editing.value) {
    if (password && password.length < 6) {
      ElMessage.warning('密码至少 6 位')
      return
    }
    dialogSaving.value = true
    try {
      await members.update(editing.value.id, {
        password: password || undefined,
        status: form.status,
      })
      ElMessage.success('已保存')
      dialogVisible.value = false
      await reload()
    } catch (e) {
      ElMessage.error(e instanceof Error ? e.message : '保存失败')
    } finally {
      dialogSaving.value = false
    }
    return
  }

  const account = form.account.trim()
  if (!account) {
    ElMessage.warning('请填写会员账号')
    return
  }
  if (password.length < 6) {
    ElMessage.warning('密码至少 6 位')
    return
  }
  dialogSaving.value = true
  try {
    await members.create({
      account,
      password,
      status: form.status,
    })
    ElMessage.success('已创建')
    dialogVisible.value = false
    currentPage.value = 1
    await reload()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  } finally {
    dialogSaving.value = false
  }
}

async function onClearAuth(row: MemberRow) {
  const ok = await adminConfirmDialog({
    title: '清空授权',
    message: `确认清空会员「${row.account}」的全部授权？将先停止其所有正在运行的方案，再清除全部第三方授权，此操作不可恢复。`,
    tone: 'danger',
    confirmText: '确定清空',
  })
  if (!ok) return
  try {
    const res = await members.clearGuajiAuth(row.id)
    ElMessage.success(res.message ?? '已清空授权')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '清空授权失败')
  }
}
</script>

<template>
  <div v-loading="loading">
    <h1 class="admin-page-title">会员查询</h1>

    <div class="toolbar">
      <el-select v-model="searchField" style="width: 128px">
        <el-option label="会员账号" value="account" />
        <el-option label="会员 ID" value="id" />
        <el-option label="授权账号" value="guajiAccount" />
      </el-select>
      <el-input
        v-model="keyword"
        clearable
        :placeholder="keywordPlaceholder"
        style="width: min(100%, 280px)"
        @keyup.enter="onSearch"
      />
      <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
      <el-button type="primary" plain @click="openCreate">新增会员</el-button>
    </div>

    <el-table :data="pagedRows" stripe style="width: 100%">
      <el-table-column prop="id" label="会员ID" min-width="100" />
      <el-table-column prop="account" label="会员账号" min-width="120" />
      <el-table-column label="状态" min-width="88">
        <template #default="{ row }">
          <el-tag :type="row.status === '正常' ? 'success' : 'danger'" size="small">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="USDT" min-width="112" align="right">
        <template #default="{ row }">{{ fmtMoney(row.guajiBalances.usdt) }}</template>
      </el-table-column>
      <el-table-column label="TRX" min-width="112" align="right">
        <template #default="{ row }">{{ fmtMoney(row.guajiBalances.trx) }}</template>
      </el-table-column>
      <el-table-column label="CNY" min-width="112" align="right">
        <template #default="{ row }">{{ fmtMoney(row.guajiBalances.cny) }}</template>
      </el-table-column>
      <el-table-column label="注册" min-width="140">
        <template #default="{ row }">{{ fmt(row.registeredAt) }}</template>
      </el-table-column>
      <el-table-column label="最近登录" min-width="140">
        <template #default="{ row }">{{ fmt(row.lastLoginAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onClearAuth(row)">清空授权</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, prev, pager, next"
        :total="tableTotal"
      />
    </div>

    <AdminDialog v-model="dialogVisible" :title="dialogTitle" width="min(92vw, 440px)">
      <el-form label-position="top">
        <el-form-item label="会员账号" required>
          <el-input
            v-model="form.account"
            :disabled="!!editing"
            placeholder="登录账号"
            maxlength="32"
          />
        </el-form-item>
        <el-form-item :label="editing ? '新密码（留空则不修改）' : '密码'" :required="!editing">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="editing ? '留空则不修改密码' : '至少 6 位'"
            autocomplete="new-password"
          />
        </el-form-item>
        <el-form-item label="状态" required>
          <el-radio-group v-model="form.status">
            <el-radio value="active">正常</el-radio>
            <el-radio value="frozen">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="dialogSaving" @click="saveForm">确定</el-button>
      </template>
    </AdminDialog>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-bottom: 1rem;
  align-items: center;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
