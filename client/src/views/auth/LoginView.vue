<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { loginClient } from '@/api/auth'
import { ApiError } from '@/api/client'
import { demoAppBrand } from '@/demo/demoAccount'

const router = useRouter()
const route = useRoute()

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({
  account: '',
  password: '',
})

const rules: FormRules = {
  account: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await loginClient(form.account.trim(), form.password)
    ElMessage.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (err) {
    const msg = err instanceof ApiError ? err.message : '登录失败，请稍后重试'
    ElMessage.error(msg || '账号或密码错误')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-aurora" aria-hidden="true"></div>

    <main class="login-shell">
      <header class="login-brand">
        <span class="login-brand__mark">{{ demoAppBrand.slice(0, 1) }}</span>
        <div class="login-brand__text">
          <h1 class="login-brand__name">{{ demoAppBrand }}</h1>
          <p class="login-brand__tag">数字精算 · 安全登录</p>
        </div>
      </header>

      <section class="login-card">
        <h2 class="login-card__title">欢迎回来</h2>
        <p class="login-card__subtitle">输入账号与密码进入终端</p>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-position="top"
          size="large"
          @submit.prevent="onSubmit"
        >
          <el-form-item label="账号" prop="account">
            <el-input
              v-model="form.account"
              placeholder="请输入账号"
              autocomplete="username"
              clearable
            />
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              autocomplete="current-password"
              show-password
              @keyup.enter="onSubmit"
            />
          </el-form-item>

          <el-button
            type="primary"
            class="login-submit"
            :loading="loading"
            native-type="submit"
          >
            登录
          </el-button>
        </el-form>
      </section>

      <footer class="login-foot">登录即表示同意平台服务条款与隐私政策</footer>
    </main>
  </div>
</template>

<style scoped>
.login-page {
  position: relative;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem var(--page-gutter);
  background:
    radial-gradient(140% 90% at 50% -10%, #e6ebfa 0%, #f7f9fb 55%, #f7f9fb 100%);
  overflow: hidden;
}

/* 浮动光晕：半透明 + 模糊（数字精算主义浮层语言） */
.login-aurora {
  position: absolute;
  inset: -20% 30% auto -10%;
  height: 460px;
  background: radial-gradient(closest-side, rgba(0, 102, 255, 0.18), transparent 70%);
  filter: blur(40px);
  pointer-events: none;
}

.login-shell {
  position: relative;
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 0.875rem;
}

.login-brand__mark {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 16px;
  background: linear-gradient(145deg, #0066ff, #0050cb);
  color: #fff;
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 1.5rem;
  box-shadow: 0 12px 28px rgba(0, 80, 203, 0.28);
}

.login-brand__name {
  margin: 0;
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 1.375rem;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.login-brand__tag {
  margin: 0.125rem 0 0;
  font-size: 0.8125rem;
  color: #64748b;
}

.login-card {
  padding: var(--card-pad);
  border-radius: 20px;
  background: #ffffff;
  box-shadow: 0 24px 60px rgba(15, 35, 95, 0.08);
}

.login-card__title {
  margin: 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 1.25rem;
  color: #0f172a;
}

.login-card__subtitle {
  margin: 0.375rem 0 1.5rem;
  font-size: 0.875rem;
  color: #64748b;
}

.login-submit {
  width: 100%;
  margin-top: 0.25rem;
  height: 46px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.login-foot {
  text-align: center;
  font-size: 0.75rem;
  color: #94a3b8;
}

:deep(.el-form-item__label) {
  font-weight: 600;
  color: #334155;
  padding-bottom: 0.25rem;
}
</style>
