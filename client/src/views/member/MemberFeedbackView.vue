<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import { submitFeedback } from '@/api/content/feedback'

const router = useRouter()
const subject = ref('')
const content = ref('')
const submitting = ref(false)
const maxLen = 500

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member' })
}

async function onSubmit(): Promise<void> {
  if (!subject.value.trim()) {
    ElMessage.warning('请填写主题')
    return
  }
  if (!content.value.trim()) {
    ElMessage.warning('请填写详细内容')
    return
  }
  submitting.value = true
  try {
    await submitFeedback({
      subject: subject.value.trim(),
      content: content.value.trim(),
    })
    ElMessage.success('提交成功，感谢您的反馈')
    subject.value = ''
    content.value = ''
  } catch (e) {
    const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '提交失败'
    ElMessage.error(message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="mfb member-subpage" data-page="member-feedback">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">意见回馈</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="mfb-main">
      <section class="mfb-card">
        <div class="mfb-field">
          <label class="mfb-lbl" for="mfb-subject">主题</label>
          <el-input
            id="mfb-subject"
            v-model="subject"
            maxlength="80"
            show-word-limit
            clearable
            placeholder="请简要描述问题或建议"
            size="large"
            class="mfb-input"
          />
        </div>
        <p class="mfb-hint">
          我们会尽快处理您的反馈，感谢您的支持。
        </p>
        <div class="mfb-field mfb-field--stack">
          <label class="mfb-lbl" for="mfb-body">详细内容</label>
          <el-input
            id="mfb-body"
            v-model="content"
            type="textarea"
            :rows="10"
            :maxlength="maxLen"
            show-word-limit
            :placeholder="`请详细描述情况，最多 ${maxLen} 字`"
            class="mfb-textarea"
          />
        </div>
      </section>

      <div class="mfb-actions">
        <el-button type="primary" size="large" round class="mfb-submit" :loading="submitting" @click="onSubmit">
          提交反馈
        </el-button>
      </div>
    </main>
  </div>
</template>

<style scoped>
.mfb {
  --mfb-primary: #0050cb;
  --mfb-primary-strong: #0066ff;
  --mfb-surface: #f7f9fb;
  --mfb-card: #ffffff;
  --mfb-on: #191c1e;
  --mfb-on-var: #424656;
  --mfb-on-mute: #727687;
  min-height: 100dvh;
  background: var(--mfb-surface);
  color: var(--mfb-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mfb-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1.15rem 1.15rem 2rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mfb-card {
  background: var(--mfb-card);
  border-radius: 1.25rem;
  padding: 1.25rem;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mfb-field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.mfb-field--stack {
  gap: 0.5rem;
}

.mfb-lbl {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--mfb-on);
  letter-spacing: 0.02em;
}

.mfb-hint {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.65;
  color: var(--mfb-primary);
  font-weight: 600;
}

.mfb-input :deep(.el-input__wrapper) {
  border-radius: 0.75rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
}

.mfb-textarea :deep(.el-textarea__inner) {
  border-radius: 0.75rem;
  font-size: 0.9375rem;
  line-height: 1.65;
  padding: 0.75rem 1rem;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.1);
}

.mfb-actions {
  display: flex;
  justify-content: center;
}

.mfb-submit {
  width: 100%;
  max-width: 22rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  box-shadow: 0 14px 32px -16px rgba(0, 80, 203, 0.55);
}
</style>
