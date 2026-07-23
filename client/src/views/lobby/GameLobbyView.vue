<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import ContentDialog from '@/components/ui/ContentDialog.vue'
import { useLobbyPageContent } from '@/composables/useLobbyPageContent'
import { useMaintenanceClient } from '@/composables/useMaintenanceClient'

const router = useRouter()

function goCreateCustomScheme() {
  void router.push({
    name: 'advanced-scheme-edit',
    params: { schemeId: 'new' },
    query: { draft: '1', kind: 'custom', fresh: '1' },
  })
}
const {
  popupAnnouncement,
  shouldBlockLobby,
  shouldShowMaintenancePopup,
  startSync,
  stopSync,
} = useMaintenanceClient()

const NEWS_ICON = '/images/lobby/news-item.png'
const BENTO_COPY_ICON = '/images/lobby/bento-copy-hall.png'
const BENTO_CUSTOM_ICON = '/images/lobby/bento-custom-scheme.png'
const BENTO_DOWNLOAD_ICON = '/images/lobby/bento-scheme-download.png'

const {
  banners,
  latestAnnouncement,
  newsRows,
  load: loadLobbyContent,
} = useLobbyPageContent(NEWS_ICON)

const bannerIndex = ref(0)
let bannerTimer: ReturnType<typeof setInterval> | null = null

function stopBannerTimer() {
  if (bannerTimer) {
    clearInterval(bannerTimer)
    bannerTimer = null
  }
}

function startBannerTimer() {
  stopBannerTimer()
  if (banners.value.length <= 1) return
  bannerTimer = setInterval(() => {
    bannerIndex.value = (bannerIndex.value + 1) % banners.value.length
  }, 5000)
}

const maintDialogVisible = ref(false)

onMounted(() => {
  startSync()
  void loadLobbyContent().then(() => startBannerTimer())
  if (shouldShowMaintenancePopup.value) maintDialogVisible.value = true
})

onUnmounted(() => {
  stopSync()
  stopBannerTimer()
})

watch(banners, () => {
  bannerIndex.value = 0
  startBannerTimer()
})

watch(shouldShowMaintenancePopup, (v) => {
  if (v) maintDialogVisible.value = true
})

/** 公告栏左侧图标：占位 PNG，可自行改为其它路径或资源 */
const ANNOUNCE_IMG = '/images/lobby/announce-placeholder.png'
</script>

<template>
  <div class="lobby" data-page="stitch-lobby">
    <header class="mss-head lobby-mss-head" role="banner">
      <div class="mss-head-deco" aria-hidden="true" />
      <div class="mss-head-bar">
        <span class="mss-head-spacer" aria-hidden="true" />
        <h1 class="mss-title">游戏大厅</h1>
        <span class="mss-head-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="main">
      <div class="main-hero-group">
      <!-- 主屏 Banner 轮播：GET /public/banners -->
      <section v-if="banners.length" class="section hero-section">
        <div class="hero-carousel" aria-label="大厅主屏轮播">
          <div class="hero-track" :style="{ transform: `translateX(-${bannerIndex * 100}%)` }">
            <div v-for="b in banners" :key="b.id" class="hero-slide">
              <a
                v-if="b.linkUrl"
                :href="b.linkUrl"
                class="hero-link"
                target="_blank"
                rel="noopener noreferrer"
                :aria-label="`打开 Banner 外链`"
              >
                <img :src="b.imageUrl" alt="" class="hero-img" width="800" height="343" decoding="async" />
              </a>
              <img
                v-else
                :src="b.imageUrl"
                alt=""
                class="hero-img"
                width="800"
                height="343"
                decoding="async"
              />
            </div>
          </div>
          <div v-if="banners.length > 1" class="hero-dots">
            <button
              v-for="(b, i) in banners"
              :key="`${b.id}-dot`"
              type="button"
              class="hero-dot"
              :class="{ 'is-active': i === bannerIndex }"
              :aria-label="`第 ${i + 1} 张 Banner`"
              @click="bannerIndex = i"
            />
          </div>
        </div>
      </section>

      <!-- Announcement: 最新已发布公告 -->
      <section v-if="latestAnnouncement" class="section">
        <RouterLink
          class="ann-bar ann-bar-link"
          :to="{ name: 'announcement-detail', params: { id: latestAnnouncement.id } }"
          :aria-label="`查看公告：${latestAnnouncement.title}`"
        >
          <img
            :src="ANNOUNCE_IMG"
            alt=""
            width="24"
            height="24"
            class="ann-ico"
            decoding="async"
          />
          <p class="ann-txt">公告：{{ latestAnnouncement.title }}</p>
          <span class="material m-sm ann-arrow" aria-hidden="true">arrow_forward_ios</span>
        </RouterLink>
      </section>

      <!-- 入口：跟单大厅 / 自创方案 / 方案下载 同一行 -->
      <section class="section bento">
        <RouterLink class="bento-large bento-large-link" to="/copy-hall">
          <div class="bento-body">
            <div class="bento-icon big">
              <img
                :src="BENTO_COPY_ICON"
                alt=""
                width="36"
                height="36"
                class="bento-ico bento-ico-lg"
                decoding="async"
              />
            </div>
            <div class="bento-txt">
              <h3 class="bento-h">跟单大厅</h3>
            </div>
          </div>
        </RouterLink>
        <button type="button" class="bento-s b-left bento-s-link" @click="goCreateCustomScheme">
          <div class="bento-icon terr">
            <img
              :src="BENTO_CUSTOM_ICON"
              alt=""
              width="22"
              height="22"
              class="bento-ico bento-ico-sm"
              decoding="async"
            />
          </div>
          <h3 class="bento-h sm">自创方案</h3>
        </button>
        <RouterLink class="bento-s b-right bento-s-link" :to="{ name: 'scheme-download' }">
          <div class="bento-icon pri">
            <img
              :src="BENTO_DOWNLOAD_ICON"
              alt=""
              width="22"
              height="22"
              class="bento-ico bento-ico-sm"
              decoding="async"
            />
          </div>
          <h3 class="bento-h sm">方案下载</h3>
        </RouterLink>
      </section>
      </div>

      <!-- 最新动态：公告列表前 3 条 -->
      <section v-if="newsRows.length" class="section news-block">
        <div class="news-head">
          <h2 class="news-h2">最新动态</h2>
          <RouterLink :to="{ name: 'member-announcements' }" class="link-all">
            查看全部
          </RouterLink>
        </div>
        <div class="news-card">
          <RouterLink
            v-for="(n, i) in newsRows"
            :key="n.id"
            :to="{ name: 'announcement-detail', params: { id: n.id } }"
            class="news-row news-row-link"
            :class="{ 'news-border': i > 0 }"
            :aria-label="`阅读公告：${n.title}`"
          >
            <div class="news-icon" :class="n.tone">
              <img
                :src="n.iconImg"
                alt=""
                width="22"
                height="22"
                class="news-ico-img"
                decoding="async"
              />
            </div>
            <div class="news-mid">
              <h4 class="news-title">{{ n.title }}</h4>
            </div>
            <span class="news-time">{{ n.time }}</span>
          </RouterLink>
        </div>
      </section>
    </main>

    <!-- 全站维护拦截：与 admin「系统维护」Mock 同源（Cookie/localStorage） -->
    <div
      v-if="shouldBlockLobby"
      class="lobby-maint-overlay"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="lobby-maint-title"
    >
      <div class="lobby-maint-panel">
        <h2 id="lobby-maint-title" class="lobby-maint-title">系统维护中</h2>
        <p class="lobby-maint-desc">
          平台正在进行维护升级，大厅功能暂不可用。请稍后再试，或查看维护公告了解详情。
        </p>
        <el-button v-if="popupAnnouncement" type="primary" round @click="maintDialogVisible = true">
          查看维护公告
        </el-button>
      </div>
    </div>

    <ContentDialog
      v-model="maintDialogVisible"
      :title="popupAnnouncement?.title ?? '平台公告'"
      icon="campaign"
      confirm-text="知道了"
      wide
    >
      <div
        v-if="popupAnnouncement"
        v-html="popupAnnouncement.bodyHtml"
      />
    </ContentDialog>
  </div>
</template>

<style scoped>
.lobby {
  /* 复用会员二级页顶栏（mss-head）色板 */
  --mss-primary: #0050cb;
  --mss-primary-strong: #0066ff;
  --c-surface: #f7f9fb;
  --c-on-surface: #191c1e;
  --c-on-surface-variant: #424656;
  --c-primary: #0050cb;
  --c-primary-container: #0066ff;
  --c-surface-c-low: #f2f4f6;
  --c-surface-c-lowest: #ffffff;
  --c-tertiary: #a33200;
  --c-tertiary-10: rgba(163, 50, 0, 0.1);
  --c-primary-10: rgba(0, 80, 203, 0.1);
  --c-outline: #727687;
  --c-outline-variant: #c2c6d8;
  min-height: 100dvh;
  background: var(--c-surface);
  color: var(--c-on-surface);
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  /* 仅留底栏避让，避免多余空白造成空滚；顶栏自带 safe-area */
  padding-bottom: calc(3.75rem + env(safe-area-inset-bottom));
}
.material {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.5rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  vertical-align: middle;
  display: inline-block;
}
.material.m-fill {
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}
.material.m-sm {
  font-size: 0.875rem;
}
.main {
  max-width: 64rem;
  margin: 0 auto;
  padding: 0.75rem var(--page-gutter) 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.main-hero-group {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.section {
  margin: 0;
}
.hero-carousel {
  position: relative;
  width: 100%;
  aspect-ratio: 21 / 9;
  min-height: 7.5rem;
  border-radius: 1.5rem;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}
.hero-track {
  display: flex;
  height: 100%;
  transition: transform 0.6s ease;
}
.hero-slide {
  flex: 0 0 100%;
  height: 100%;
}
.hero-link {
  display: block;
  width: 100%;
  height: 100%;
}
.hero-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.hero-dots {
  position: absolute;
  bottom: 0.75rem;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 0.375rem;
  z-index: 2;
}
.hero-dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.45);
  cursor: pointer;
  padding: 0;
  transition: background 0.2s;
}
.hero-dot.is-active {
  background: #fff;
}
.ann-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: var(--card-pad);
  background: var(--c-surface-c-low);
  border-radius: 1rem;
}
.ann-bar-link {
  text-decoration: none;
  color: inherit;
  cursor: pointer;
  transition: background 0.15s, transform 0.2s;
}
.ann-bar-link:hover {
  background: #e9ecf2;
}
.ann-bar-link:active {
  transform: scale(0.995);
}
.ann-ico {
  flex-shrink: 0;
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
  display: block;
}
.ann-txt {
  margin: 0;
  flex: 1;
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--c-on-surface-variant);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ann-arrow {
  color: var(--c-outline-variant);
  flex-shrink: 0;
  opacity: 0.8;
}
.bento {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.65rem;
  align-items: stretch;
}
.bento-large {
  position: relative;
  min-height: 6.25rem;
  background: var(--c-surface-c-lowest);
  border-radius: 1rem;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  overflow: hidden;
  transition: box-shadow 0.3s;
}
.bento-large:hover {
  box-shadow: 0 10px 20px -8px rgba(0, 0, 0, 0.12);
}
.bento-large-link {
  text-decoration: none;
  color: inherit;
}
.bento .b-left,
.bento .b-right {
  grid-column: auto;
  grid-row: auto;
}
.bento-s {
  padding: 0.55rem 0.4rem 0.6rem;
  border-radius: 1rem;
  background: var(--c-surface-c-lowest);
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  min-width: 0;
  min-height: 6.25rem;
  box-sizing: border-box;
  transition: box-shadow 0.2s;
  cursor: pointer;
}
.bento-s-link {
  text-decoration: none;
  color: inherit;
  border: none;
  font: inherit;
  width: 100%;
  appearance: none;
  -webkit-appearance: none;
}
.bento-s:hover {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}
.bento-body {
  padding: 0.55rem 0.4rem 0.6rem;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  min-height: 6.25rem;
  box-sizing: border-box;
}
.bento-icon.big {
  width: 3.25rem;
  height: 3.25rem;
  background: var(--c-primary-10);
  color: var(--c-primary);
  border-radius: 0.85rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 0 0.45rem;
  flex-shrink: 0;
}
.bento-icon.big .bento-ico-lg {
  width: 2rem;
  height: 2rem;
  object-fit: contain;
  display: block;
}
.bento-txt {
  margin: 0;
  min-width: 0;
  width: 100%;
}
.bento-h {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--c-on-surface);
  line-height: 1.2;
  text-align: center;
  letter-spacing: -0.02em;
}
.bento-h.sm {
  font-size: 1.0625rem;
}
.bento-icon.terr {
  width: 3.25rem;
  height: 3.25rem;
  background: var(--c-tertiary-10);
  color: var(--c-tertiary);
  border-radius: 0.85rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 0 0.45rem;
  flex-shrink: 0;
}
.bento-icon.terr .bento-ico-sm,
.bento-icon.pri .bento-ico-sm {
  width: 1.85rem;
  height: 1.85rem;
  object-fit: contain;
  display: block;
}
.bento-icon.pri {
  width: 3.25rem;
  height: 3.25rem;
  background: var(--c-primary-10);
  color: var(--c-primary);
  border-radius: 0.85rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 0 0.45rem;
  flex-shrink: 0;
}
.news-block {
  padding-bottom: 0.5rem;
}
.news-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 1rem;
}
.news-h2 {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--c-on-surface);
}
.link-all {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--c-primary);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  font-family: inherit;
  text-decoration: none;
}

.link-all:hover {
  text-decoration: underline;
}
.news-card {
  background: var(--c-surface-c-lowest);
  border-radius: 1.5rem;
  overflow: hidden;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}
.news-row {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.65rem 1rem;
  transition: background 0.15s;
  cursor: pointer;
}
.news-row-link {
  text-decoration: none;
  color: inherit;
}
.news-row:hover {
  background: var(--c-surface-c-low);
}
.news-row.news-border {
  border-top: 1px solid #eceef0;
}
.news-icon {
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.65rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.news-icon .news-ico-img {
  width: 1.15rem;
  height: 1.15rem;
  object-fit: contain;
  display: block;
}
.news-icon.blue {
  background: #eff6ff;
  color: #2563eb;
}
.news-icon.amber {
  background: #fffbeb;
  color: #d97706;
}
.news-icon.green {
  background: #f0fdf4;
  color: #16a34a;
}
.news-mid {
  flex: 1;
  min-width: 0;
}
.news-title {
  margin: 0;
  font-weight: 700;
  font-size: 0.875rem;
  line-height: 1.3;
  color: var(--c-on-surface);
}
.news-body {
  margin: 0.15rem 0 0;
  font-size: 0.875rem;
  color: var(--c-on-surface-variant);
  line-height: 1.4;
}
.news-time {
  font-size: 0.6875rem;
  color: var(--c-outline);
  font-weight: 500;
  white-space: nowrap;
  flex-shrink: 0;
}

.lobby-maint-overlay {
  position: fixed;
  inset: 0;
  z-index: 5000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--card-pad);
  background: rgb(15 23 42 / 42%);
  backdrop-filter: blur(20px);
}

.lobby-maint-panel {
  width: min(100%, 22rem);
  padding: var(--card-pad);
  border-radius: 1rem;
  background: var(--c-surface-c-lowest);
  box-shadow: 0 24px 48px rgb(26 62 138 / 12%);
  text-align: center;
}

.lobby-maint-title {
  margin: 0 0 0.75rem;
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--c-on-surface);
}

.lobby-maint-desc {
  margin: 0 0 1.25rem;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--c-on-surface-variant);
}
</style>
