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
  row.read = true
  void router.push({ name: 'announcement-detail', params: { id: row.id } })
}
</script>

<template>
  <div class="man member-subpage" data-page="member-announcements">
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
          <span class="man-row-title">{{ row.title }}</span>
          <span class="man-row-meta">
            <span class="man-row-date">{{ row.date }}</span>
            <span class="man-row-status" :class="{ 'is-read': row.read }">{{
              row.read ? '已读' : '未读'
            }}</span>
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
  --man-danger: #ba1a1a;
  min-height: 100dvh;
  background: var(--man-surface);
  color: var(--man-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.man-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 1.15rem 1.15rem 2rem;
}

.man-list {
  background: var(--man-card);
  border-radius: 1.25rem;
  padding: 0.35rem 0;
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
  grid-template-columns: auto 1fr auto;
  align-items: start;
  gap: 0.65rem 0.75rem;
  padding: 1rem 1.15rem;
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
  width: 4px;
  border-radius: 999px;
  min-height: 2.25rem;
  margin-top: 0.15rem;
  background: linear-gradient(180deg, var(--man-primary-strong), var(--man-primary));
  align-self: stretch;
}

.man-row-title {
  font-size: 0.9375rem;
  font-weight: 650;
  line-height: 1.5;
  letter-spacing: 0.01em;
}

.man-row-meta {
  grid-column: 2 / -1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.65rem;
  font-size: 0.75rem;
}

@media (min-width: 400px) {
  .man-row {
    grid-template-columns: auto 1fr auto;
    align-items: center;
  }

  .man-row-meta {
    grid-column: unset;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.2rem;
  }
}

.man-row-date {
  color: var(--man-on-mute);
  font-weight: 500;
}

.man-row-status {
  font-weight: 700;
  color: var(--man-danger);
  letter-spacing: 0.02em;
}

.man-row-status.is-read {
  color: var(--man-on-mute);
  font-weight: 600;
}
</style>
