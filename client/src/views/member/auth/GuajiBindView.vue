<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { bindGuajiAccount } from '@/api/guaji/accounts'
import { logoutClient } from '@/api/auth'
import { formatClientApiError } from '@/utils/guajiError'
import { invalidateGuajiAuthCache } from '@/composables/useGuajiAuthGuard'
import { confirmDialog } from '@/utils/confirmDialog'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const mfaStep = ref(false)
const loginKey = ref('')

/** 来源：授权列表（个人中心切换授权）进入时返回列表；首绑被守卫拦截时返回登录 */
const fromList = computed(() => String(route.query.from ?? '') === 'list')

const form = reactive({
  username: '',
  password: '',
  googleCode: '',
})

async function submit() {
  if (!form.username.trim() || !form.password) {
    ElMessage.warning('请填写第三方用户名与密码')
    return
  }
  loading.value = true
  try {
    const res = await bindGuajiAccount({
      username: form.username.trim(),
      password: form.password,
      loginKey: mfaStep.value ? loginKey.value : undefined,
      googleCode: mfaStep.value ? form.googleCode : undefined,
    })
    if (res.mfaRequired) {
      mfaStep.value = true
      loginKey.value = res.loginKey ?? ''
      ElMessage.info('请完成二次验证（Google 验证码）')
      return
    }
    invalidateGuajiAuthCache()
    ElMessage.success('绑定成功')
    await router.replace('/member')
  } catch (e) {
    ElMessage.error(formatClientApiError(e, '绑定失败'))
  } finally {
    loading.value = false
  }
}

async function onCancel() {
  const ok = await confirmDialog({ message: '确定放弃绑定并返回？' })
  if (ok) await router.replace('/member/auth/list')
}

async function onBack() {
  // 从授权列表进入：直接返回列表，不退出登录
  if (fromList.value) {
    await router.replace('/member/auth/list')
    return
  }
  const ok = await confirmDialog({
    title: '返回登录',
    message: '将退出当前账号并返回登录页，确定继续？',
    confirmText: '退出并返回',
    cancelText: '取消',
  })
  if (!ok) return
  logoutClient()
  invalidateGuajiAuthCache()
  await router.replace({ name: 'login' })
}
</script>

<template>
  <div class="guaji-auth-page member-subpage" data-page="member-auth-bind">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" :aria-label="fromList ? '返回授权列表' : '返回登录'" @click="onBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">绑定授权账号</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <div class="page-body">
      <p class="page-hint">填写第三方 Hash 挂机平台的用户名与密码。首绑成功后将自动设为启用。</p>

    <el-card shadow="never" class="form-card">
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="第三方用户名">
          <el-input v-model="form.username" autocomplete="username" :disabled="mfaStep" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password autocomplete="current-password" :disabled="mfaStep" />
        </el-form-item>
        <el-form-item v-if="mfaStep" label="Google 验证码">
          <el-input v-model="form.googleCode" maxlength="8" inputmode="numeric" />
        </el-form-item>
        <div class="actions">
          <el-button v-if="mfaStep" @click="onCancel">取消</el-button>
          <el-button type="primary" :loading="loading" native-type="submit">
            {{ mfaStep ? '完成验证' : '绑定' }}
          </el-button>
        </div>
      </el-form>
    </el-card>
    </div>
  </div>
</template>

<style scoped>
.guaji-auth-page {
  min-height: 100dvh;
  background: var(--mss-surface, #f7f9fb);
  padding-bottom: 2rem;
}
.page-body {
  max-width: 28rem;
  margin: 0 auto;
  padding: 1.25rem 1rem 0;
}
.page-hint {
  margin: 0 0 1.25rem;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.form-card {
  border: none;
  background: var(--el-bg-color);
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}
</style>
