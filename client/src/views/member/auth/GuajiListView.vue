<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  activateGuajiAccount,
  deleteGuajiAccount,
  fetchGuajiAccounts,
  fetchGuajiAuthStatus,
  reauthGuajiAccount,
  type GuajiAccountRow,
  type GuajiAuthStatus,
} from '@/api/guaji/accounts'
import { invalidateGuajiAuthCache, resolveGuajiAuthStatus } from '@/composables/useGuajiAuthGuard'
import { confirmDialog } from '@/utils/confirmDialog'
import { formatClientApiError, formatGuajiAccountError } from '@/utils/guajiError'

const router = useRouter()
const loading = ref(false)
const rows = ref<GuajiAccountRow[]>([])
const authStatus = ref<GuajiAuthStatus | null>(null)

const showExpiredBanner = computed(
  () => authStatus.value?.activeAuthExpired || rows.value.some((r) => r.authExpired),
)

async function load() {
  loading.value = true
  try {
    const [items, status] = await Promise.all([fetchGuajiAccounts(), fetchGuajiAuthStatus()])
    rows.value = items
    authStatus.value = status
  } catch (e) {
    ElMessage.error(formatClientApiError(e, '加载失败'))
  } finally {
    loading.value = false
  }
}

async function onActivate(row: GuajiAccountRow) {
  const ok = await confirmDialog({ message: '将停止全部挂机方案，确定切换启用该授权账号？' })
  if (!ok) return
  loading.value = true
  try {
    await activateGuajiAccount(row.id)
    invalidateGuajiAuthCache()
    ElMessage.success('已切换启用；方案已暂停，请到云端中心开启')
    await load()
  } catch (e) {
    ElMessage.error(formatClientApiError(e, '切换失败'))
  } finally {
    loading.value = false
  }
}

/** 行级加载：正在重新授权的账号 id，防重复点击 */
const reauthingId = ref<number | null>(null)

async function onReauth(row: GuajiAccountRow) {
  if (reauthingId.value !== null) return
  reauthingId.value = row.id
  try {
    const acct = await reauthGuajiAccount(row.id)
    invalidateGuajiAuthCache()
    await load()
    if (acct.authExpired) {
      ElMessage.error(formatGuajiAccountError(acct.lastTokenError) || '重新授权失败，请解绑后重新绑定')
      return
    }
    const status = await resolveGuajiAuthStatus(true)
    if (row.isActive && !status.hasActiveGuajiAuth) {
      ElMessage.error('重新授权失败，启用账号仍未恢复')
      return
    }
    ElMessage.success('重新授权成功')
    if (row.isActive && status.hasActiveGuajiAuth) {
      await router.replace('/member')
    }
  } catch (e) {
    ElMessage.error(formatClientApiError(e, '重新授权失败'))
  } finally {
    reauthingId.value = null
  }
}

async function onDelete(row: GuajiAccountRow) {
  const ok = await confirmDialog({ message: '确定解绑该授权账号？解绑后将停止全部挂机方案。' })
  if (!ok) return
  loading.value = true
  try {
    await deleteGuajiAccount(row.id)
    invalidateGuajiAuthCache()
    ElMessage.success('已解绑')
    await load()
  } catch (e) {
    ElMessage.error(formatClientApiError(e, '解绑失败'))
  } finally {
    loading.value = false
  }
}

function accountStatusHint(row: GuajiAccountRow): string {
  const err = formatGuajiAccountError(row.lastTokenError)
  if (err) return err
  if (row.authExpired) return '授权已失效，请点击「重新授权」恢复'
  return ''
}

onMounted(load)
</script>

<template>
  <div class="guaji-list-page member-subpage" data-page="member-auth-list">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回个人中心" @click="router.push('/member')">
          <span class="mss-ms" aria-hidden="true">arrow_back</span>
        </button>
        <h1 class="mss-title">授权账号</h1>
        <button type="button" class="mss-head-icon-btn" aria-label="添加授权账号"
          @click="router.push({ path: '/member/auth/bind', query: { from: 'list' } })">
          <span class="mss-ms" aria-hidden="true">add</span>
        </button>
      </div>
    </header>

    <el-alert
      v-if="showExpiredBanner"
      type="warning"
      :closable="false"
      show-icon
      class="auth-expired-banner"
      title="授权已失效"
      description="当前启用账号的第三方授权已过期，挂机方案已暂停。请点击对应账号的「重新授权」，恢复后再前往云端中心开启方案。"
    />

    <el-skeleton v-if="loading && !rows.length" animated :rows="4" />
    <el-empty v-else-if="!rows.length" description="暂无绑定，请先添加授权账号">
      <el-button type="primary" @click="router.push({ path: '/member/auth/bind', query: { from: 'list' } })">添加授权账号</el-button>
    </el-empty>

    <div v-else class="card-list">
      <el-card v-for="row in rows" :key="row.id" shadow="never" class="acct-card" :class="{ 'acct-card--expired': row.authExpired }">
        <div class="acct-main">
          <div class="acct-name">{{ row.guajiUsername }}</div>
          <el-tag v-if="row.authExpired" type="danger" size="small">授权已失效</el-tag>
          <el-tag v-else-if="row.isActive" type="success" size="small">当前启用</el-tag>
          <el-tag v-else type="info" size="small">已绑未启用</el-tag>
        </div>
        <p v-if="accountStatusHint(row)" class="acct-error">{{ accountStatusHint(row) }}</p>
        <div class="acct-actions">
          <el-button v-if="!row.isActive" size="small" type="primary" :disabled="reauthingId !== null"
            @click="onActivate(row)">设为启用</el-button>
          <el-button size="small" :type="row.authExpired ? 'primary' : 'default'" :loading="reauthingId === row.id"
            :disabled="reauthingId !== null && reauthingId !== row.id" @click="onReauth(row)">重新授权</el-button>
          <el-button size="small" type="danger" plain :disabled="reauthingId !== null"
            @click="onDelete(row)">解绑</el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.guaji-list-page {
  min-height: 100dvh;
  background: var(--mss-surface, #f7f9fb);
  padding-bottom: 2rem;
}
.auth-expired-banner {
  max-width: 36rem;
  margin: 0.75rem auto 0;
  padding: 0 1rem;
}

.card-list,
.guaji-list-page > .el-skeleton,
.guaji-list-page > .el-empty {
  max-width: 36rem;
  margin: 0 auto;
  padding: 1rem;
}

.auth-expired-banner + .el-skeleton,
.auth-expired-banner + .el-empty,
.auth-expired-banner + .card-list {
  padding-top: 0.75rem;
}
.card-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.acct-card {
  border: none;
}

.acct-card--expired {
  background: rgba(254, 242, 242, 0.55);
}
.acct-main {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}
.acct-name {
  font-weight: 600;
  font-size: 1rem;
}
.acct-error {
  margin: 0 0 0.75rem;
  font-size: 0.8125rem;
  color: var(--el-color-danger);
}
.acct-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
