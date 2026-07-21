<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import { fetchFaqList, type FaqListItem } from '@/api/content/faq'

const router = useRouter()
const list = ref<FaqListItem[]>([])

async function loadList() {
  try {
    list.value = await fetchFaqList()
  } catch (e) {
    {
      const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '加载 FAQ 失败'
      ElMessage.error(message)
    }
  }
}

onMounted(() => {
  void loadList()
})

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member' })
}

function openItem(id: string): void {
  void router.push({ name: 'member-faq-detail', params: { id } })
}
</script>

<template>
  <div class="mfq member-subpage" data-page="member-faq">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">常见问题</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="mfq-main">
      <section class="mfq-card" aria-label="问题列表">
        <button v-for="row in list" :key="row.id" type="button" class="mfq-row" @click="openItem(row.id)">
          <span class="mfq-row-text">{{ row.title }}</span>
          <span class="mfq-ms mfq-row-chev" aria-hidden="true">chevron_right</span>
        </button>
      </section>
    </main>
  </div>
</template>

<style scoped>
.mfq {
  --mfq-primary: #0050cb;
  --mfq-primary-strong: #0066ff;
  --mfq-surface: #f7f9fb;
  --mfq-card: #ffffff;
  --mfq-on: #191c1e;
  min-height: 100dvh;
  background: var(--mfq-surface);
  color: var(--mfq-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.mfq-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.35rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.mfq-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1.15rem 1.15rem 2rem;
}

.mfq-card {
  background: var(--mfq-card);
  border-radius: 1.25rem;
  padding: 0.35rem 0;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
  overflow: hidden;
}

.mfq-row {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 1rem 1.15rem;
  border: none;
  background: transparent;
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: var(--mfq-primary);
  font-weight: 650;
  font-size: 0.9375rem;
  line-height: 1.45;
  transition: background 0.12s;
}

.mfq-row + .mfq-row {
  box-shadow: 0 -1px 0 rgba(15, 23, 42, 0.06);
}

.mfq-row:hover {
  background: rgba(0, 102, 255, 0.06);
}

.mfq-row-text {
  flex: 1;
  min-width: 0;
}

.mfq-row-chev {
  flex-shrink: 0;
  font-size: 1.25rem;
  color: rgba(0, 80, 203, 0.65);
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}
</style>
