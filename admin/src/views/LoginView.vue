<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('admin')
const password = ref('admin123')
const loading = ref(false)

async function onSubmit() {
  loading.value = true
  try {
    const ok = await auth.login(username.value.trim(), password.value)
    if (!ok) {
      ElMessage.error('账号或密码错误')
      return
    }
    ElMessage.success('登录成功')
    const redir = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redir)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <el-card class="login-card" shadow="never">
      <template #header>
        <div class="login-card-header">
          <span class="login-title">管理后台</span>
        </div>
      </template>

      <el-form label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="账号">
          <el-input v-model="username" autocomplete="username" :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" type="password" show-password autocomplete="current-password"
            @keyup.enter="onSubmit" />
        </el-form-item>
        <el-button type="primary" class="login-btn" :loading="loading" native-type="submit">
          登录
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: radial-gradient(120% 80% at 50% 0%, #e8f1ff 0%, var(--admin-surface-bg) 55%);
}

.login-card {
  width: 100%;
  max-width: 400px;
  border-radius: var(--el-border-radius-large);
  box-shadow: 0 16px 48px rgb(26 62 138 / 8%);
}

.login-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.login-title {
  font-family: var(--admin-font-display);
  font-weight: 800;
  font-size: 1.125rem;
  letter-spacing: -0.02em;
}

.login-btn {
  width: 100%;
  margin-top: 0.5rem;
}

.login-hint {
  margin: 1rem 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
</style>
