<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useMembersStore } from '@/stores/members'

import type { MemberSearchField } from '@/api/members'

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

function openDetail(id: string) {
  router.push({ name: 'member-detail', params: { id } })
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
      <el-input v-model="keyword" clearable :placeholder="keywordPlaceholder" style="width: min(100%, 280px)"
        @keyup.enter="onSearch" />
      <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
    </div>

    <el-table :data="pagedRows" stripe style="width: 100%">
      <el-table-column prop="id" label="会员ID" min-width="100" />
      <el-table-column prop="account" label="会员账号" min-width="120" />
      <el-table-column prop="status" label="状态" min-width="72" />
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
      <el-table-column label="操作" min-width="96" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row.id)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination v-model:current-page="currentPage" :page-size="pageSize" layout="total, prev, pager, next"
        :total="tableTotal" />
    </div>
  </div>
</template>

<style scoped>
.admin-page-desc {
  margin: 0 0 1rem;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

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
