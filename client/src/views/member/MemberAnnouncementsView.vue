<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import { fetchAnnouncements, type AnnouncementListItem } from '@/api/content/announcements'

const router = useRouter()
const loading = ref(false)
const items = ref<AnnouncementListItem[]>([])

async function loadList() {
  loading.value = true
  try {
    items.value = await fetchAnnouncements()
  } catch (e) {
    {
      const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '加载公告失败'
      ElMessage.error(message)
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadList()
})

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'member' })
}

function openRow(row: AnnouncementListItem): void {
  void router.push({ name: 'announcement-detail', params: { id: row.id } })
}
</script>

<template>
  <div class="man member-subpage" data-page="member-announcements" v-loading="loading">
    <header class="mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <button type="button" class="mss-back" aria-label="返回会员中心" @click="goBack">
          <span class="mss-ms" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="mss-title">平台公告</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="man-main">
      <section class="man-list" aria-label="公告列表">
        <button
          v-for="row in items"
          :key="row.id"
          type="button"
          class="man-row"
          @click="openRow(row)"
        >
          <span class="man-bar" aria-hidden="true" />
          <span class="man-row-body">
            <span class="man-row-title">{{ row.title }}</span>
            <span class="man-row-date">{{ row.date }}</span>
          </span>
        </button>
      </section>
    </main>
  </div>
</template>

<style scoped>
.man {
  --man-primary: #0050cb;
  --man-primary-strong: #0066ff;
  --man-surface: #f7f9fb;
  --man-card: #ffffff;
  --man-on: #191c1e;
  --man-on-mute: #727687;
  min-height: 100dvh;
  background: var(--man-surface);
  color: var(--man-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.man-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1rem var(--page-gutter) 2rem;
}

.man-list {
  background: var(--man-card);
  border-radius: 1.25rem;
  padding: 0.2rem 0;
  box-shadow:
    0 24px 48px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
  overflow: hidden;
}

.man-row {
  width: 100%;
  border: none;
  cursor: pointer;
  background: transparent;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 0.55rem;
  padding: 0.55rem var(--page-gutter);
  text-align: left;
  color: inherit;
  font: inherit;
  transition: background 0.12s;
}

.man-row + .man-row {
  box-shadow: 0 -1px 0 rgba(15, 23, 42, 0.06);
}

.man-row:hover {
  background: rgba(0, 80, 203, 0.04);
}

.man-bar {
  width: 3px;
  border-radius: 999px;
  height: 1.85rem;
  background: linear-gradient(180deg, var(--man-primary-strong), var(--man-primary));
  align-self: center;
}

.man-row-body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.12rem;
}

.man-row-title {
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.35;
  letter-spacing: 0.01em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.man-row-date {
  font-size: 0.6875rem;
  font-weight: 500;
  line-height: 1.3;
  color: var(--man-on-mute);
}
</style>
