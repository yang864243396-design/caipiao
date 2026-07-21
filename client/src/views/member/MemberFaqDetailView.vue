<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import { fetchFaqDetail, type FaqDetail } from '@/api/content/faq'
const route = useRoute()
const router = useRouter()
const apiArticle = ref<FaqDetail | null>(null)

const title = computed(() => apiArticle.value?.title ?? '问题详情')
const bodyHtml = computed(() => apiArticle.value?.bodyHtml ?? '')

async function loadDetail() {
  const id = String(route.params.id ?? '')
  if (!id) return
  try {
    apiArticle.value = await fetchFaqDetail(id)
  } catch (e) {
    const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '加载详情失败'
    ElMessage.error(message)
  }
}

onMounted(() => {
  void loadDetail()
})

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member-faq' })
}
</script>

<template>
  <div class="mfd member-subpage" data-page="member-faq-detail">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回常见问题" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">问题详情</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="mfd-main">
      <article class="mfd-card">
        <h2 class="mfd-h2">{{ title }}</h2>
        <div v-if="bodyHtml" class="mfd-rich cms-rich-html" v-html="bodyHtml" />
        <p v-else class="mfd-p">暂无内容</p>
      </article>
    </main>
  </div>
</template>

<style scoped>
.mfd {
  --mfd-primary: #0050cb;
  --mfd-primary-strong: #0066ff;
  --mfd-surface: #f7f9fb;
  --mfd-card: #ffffff;
  --mfd-on: #191c1e;
  --mfd-on-var: #424656;
  min-height: 100dvh;
  background: var(--mfd-surface);
  color: var(--mfd-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mfd-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1.15rem 1.15rem 2rem;
}

.mfd-card {
  background: var(--mfd-card);
  border-radius: 1rem;
  padding: 1.25rem 1.15rem;
  box-shadow: 0 12px 40px rgba(0, 80, 203, 0.06);
}

.mfd-h2 {
  margin: 0 0 1rem;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', sans-serif;
  font-size: 1.05rem;
  font-weight: 700;
  line-height: 1.5;
}

.mfd-p,
.mfd-rich :deep(p) {
  margin: 0 0 0.85rem;
  font-size: 0.9375rem;
  line-height: 1.75;
  color: var(--mfd-on-var);
}

.mfd-rich :deep(p:last-child) {
  margin-bottom: 0;
}
</style>
