<script setup lang="ts">
import LobbyTabBar from '@/components/lobby/LobbyTabBar.vue'

/**
 * 与 Stitch 导出 / Tailwind 内联 config 1:1（无 tailwind 运行时，纯 scoped CSS 等价）
 */
const HERO_IMG = '/images/lobby/hero-championship.png'
const AVATAR_IMG = '/images/lobby/avatar-user.png'
/** 通知图标：占位 PNG，可自行改为其它路径或资源 */
const NOTIFY_IMG = '/images/lobby/notify-placeholder.png'
/** 公告栏左侧图标：占位 PNG，可自行改为其它路径或资源 */
const ANNOUNCE_IMG = '/images/lobby/announce-placeholder.png'
/** 通用 UI 图标占位（底栏、bento、动态行等），可自行替换 */
const ICON_PLACEHOLDER = '/images/lobby/icon-placeholder.png'

const news = [
  {
    id: 'n1',
    iconImg: ICON_PLACEHOLDER,
    tone: 'blue',
    title: '充值须知：支付渠道升级提醒',
    body: '为保障资金安全，请务必查看最新支付指南',
    time: '10:45 AM',
  },
  {
    id: 'n2',
    iconImg: ICON_PLACEHOLDER,
    tone: 'amber',
    title: '服务器维护：每周例行更新',
    body: '本周五凌晨 02:00-04:00 进行系统扩容',
    time: '昨天',
  },
  {
    id: 'n3',
    iconImg: ICON_PLACEHOLDER,
    tone: 'green',
    title: '安全中心：账户保护功能加强',
    body: '支持物理硬件密钥二次验证，全面提升安全性',
    time: '3天前',
  },
] as const
</script>

<template>
  <div class="lobby" data-page="stitch-lobby">
    <header class="topbar" role="banner">
      <div class="topbar-inner">
        <div class="top-left">
          <div class="avatar-wrap">
            <img
              :src="AVATAR_IMG"
              alt="用户头像"
              width="40"
              height="40"
              class="avatar-img"
              decoding="async"
            />
          </div>
          <h1 class="brand">精密终端</h1>
        </div>
        <div class="top-right">
          <button type="button" class="icon-btn" aria-label="通知">
            <img
              :src="NOTIFY_IMG"
              alt=""
              width="24"
              height="24"
              class="notify-ico"
              decoding="async"
            />
          </button>
        </div>
      </div>
    </header>

    <main class="main">
      <!-- Hero: aspect-[21/9] + gradient + overlay text -->
      <section class="section hero-section">
        <div class="hero-card">
          <img
            :src="HERO_IMG"
            alt="Precision Championship 2024"
            class="hero-img"
            width="800"
            height="343"
            decoding="async"
          />
          <div class="hero-grad">
            <span class="ongoing">Ongoing</span>
            <h2 class="hero-title">精密冠军赛 2024</h2>
            <p class="hero-sub">决战巅峰，赢取丰厚赛季积分</p>
          </div>
        </div>
      </section>

      <!-- Announcement -->
      <section class="section">
        <RouterLink
          class="ann-bar ann-bar-link"
          :to="{ name: 'announcement-detail', params: { id: 'version-2-4' } }"
          aria-label="查看公告详情"
        >
          <img
            :src="ANNOUNCE_IMG"
            alt=""
            width="24"
            height="24"
            class="ann-ico"
            decoding="async"
          />
          <p class="ann-txt">
            公告：精密终端 2.4 版本已上线，新增多维方案分析工具，请各位玩家及时更新。
          </p>
          <span class="material m-sm ann-arrow" aria-hidden="true">arrow_forward_ios</span>
        </RouterLink>
      </section>

      <!-- Bento grid: md:grid-cols-4 -->
      <section class="section bento">
        <RouterLink class="bento-large bento-large-link" to="/copy-hall">
          <span class="live-pill" aria-label="直播">
            <span class="live-dot" />
            LIVE
          </span>
          <div class="bento-body">
            <div class="bento-icon big">
              <img
                :src="ICON_PLACEHOLDER"
                alt=""
                width="36"
                height="36"
                class="bento-ico bento-ico-lg"
                decoding="async"
              />
            </div>
            <div class="bento-txt">
              <h3 class="bento-h">跟单大厅</h3>
              <p class="bento-p">实时数据对接，极速成交体验</p>
            </div>
          </div>
        </RouterLink>
        <RouterLink class="bento-s b-left bento-s-link" to="/play/custom-scheme/new">
          <div class="bento-icon terr">
            <img
              :src="ICON_PLACEHOLDER"
              alt=""
              width="22"
              height="22"
              class="bento-ico bento-ico-sm"
              decoding="async"
            />
          </div>
          <h3 class="bento-h sm">自创方案</h3>
          <p class="bento-p sm">自由定制专属逻辑与参数</p>
        </RouterLink>
        <div class="bento-s b-right">
          <div class="bento-icon pri">
            <img
              :src="ICON_PLACEHOLDER"
              alt=""
              width="22"
              height="22"
              class="bento-ico bento-ico-sm"
              decoding="async"
            />
          </div>
          <h3 class="bento-h sm">方案下载</h3>
          <p class="bento-p sm">离线同步，随时随地查阅</p>
        </div>
      </section>

      <!-- 最新动态 -->
      <section class="section news-block">
        <div class="news-head">
          <h2 class="news-h2">最新动态</h2>
          <button type="button" class="link-all">查看全部</button>
        </div>
        <div class="news-card">
          <RouterLink
            v-for="(n, i) in news"
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
              <p class="news-body">{{ n.body }}</p>
            </div>
            <span class="news-time">{{ n.time }}</span>
          </RouterLink>
        </div>
      </section>
    </main>

    <LobbyTabBar />
  </div>
</template>

<style scoped>
.lobby {
  --c-surface: #f7f9fb;
  --c-on-surface: #191c1e;
  --c-on-surface-variant: #424656;
  --c-primary: #0050cb;
  --c-primary-container: #0066ff;
  --c-surface-c-low: #f2f4f6;
  --c-surface-c-lowest: #ffffff;
  --c-surface-c-high: #e6e8ea;
  --c-tertiary: #a33200;
  --c-tertiary-10: rgba(163, 50, 0, 0.1);
  --c-primary-10: rgba(0, 80, 203, 0.1);
  --c-error: #ba1a1a;
  --c-outline: #727687;
  --c-outline-variant: #c2c6d8;
  --c-blue-600: #2563eb;
  --c-slate-400: #94a3b8;
  --c-slate-50: #f8fafc;
  --c-slate-600: #475569;
  min-height: max(884px, 100dvh);
  background: var(--c-surface);
  color: var(--c-on-surface);
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  padding-bottom: 6rem;
  padding-top: env(safe-area-inset-top);
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
.notify-ico {
  display: block;
  width: 1.4rem;
  height: 1.4rem;
  object-fit: contain;
  flex-shrink: 0;
  pointer-events: none;
}
.icon-btn:hover .notify-ico {
  opacity: 0.85;
}
.topbar {
  position: sticky;
  top: 0;
  z-index: 50;
  width: 100%;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(40px);
  -webkit-backdrop-filter: blur(40px);
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.04);
}
.topbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  width: 100%;
  max-width: 64rem;
  margin: 0 auto;
}
.top-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.avatar-wrap {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 999px;
  background: var(--c-surface-c-high);
  overflow: hidden;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.brand {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--c-blue-600);
}
.top-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.icon-btn {
  padding: 0.5rem;
  border-radius: 999px;
  line-height: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  transition:
    background 0.15s,
    transform 0.2s;
}
.icon-btn:hover {
  background: var(--c-slate-50);
}
.icon-btn:active {
  transform: scale(0.95);
}
.main {
  max-width: 64rem;
  margin: 0 auto;
  padding: 0 1.5rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 2rem;
}
.section {
  margin: 0;
}
.hero-section {
  margin-top: 1.5rem;
}
.hero-card {
  position: relative;
  width: 100%;
  aspect-ratio: 21 / 9;
  min-height: 7.5rem;
  border-radius: 1.5rem;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  cursor: pointer;
}
.hero-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.7s ease;
}
.hero-card:hover .hero-img {
  transform: scale(1.05);
}
.hero-grad {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    to top,
    rgba(0, 0, 0, 0.8) 0%,
    rgba(0, 0, 0, 0.2) 45%,
    transparent 100%
  );
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  padding: 2rem;
}
.ongoing {
  display: inline-block;
  width: fit-content;
  padding: 0.2rem 0.75rem;
  background: var(--c-primary);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  border-radius: 999px;
  margin-bottom: 0.75rem;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
}
.hero-title {
  margin: 0;
  color: #fff;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: clamp(1.875rem, 5vw, 3rem);
  font-weight: 900;
  line-height: 1;
  letter-spacing: -0.02em;
}
.hero-sub {
  margin: 0.5rem 0 0;
  color: rgba(255, 255, 255, 0.8);
  font-size: 1rem;
  font-weight: 500;
}
.ann-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1.25rem;
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
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  align-items: stretch;
}
.bento-large {
  grid-column: 1 / -1;
  grid-row: 1 / 3;
  position: relative;
  min-height: 12rem;
  background: var(--c-surface-c-lowest);
  border-radius: 1.5rem;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  overflow: hidden;
  transition: box-shadow 0.3s;
}
.bento-large:hover {
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
}
.bento-large-link {
  text-decoration: none;
  color: inherit;
}
.bento .b-left {
  grid-column: 1;
  grid-row: 3;
}
.bento .b-right {
  grid-column: 2;
  grid-row: 3;
}
@media (min-width: 768px) {
  .bento {
    grid-template-columns: repeat(4, 1fr);
  }
  .bento-large {
    grid-column: 1 / 3;
    grid-row: 1 / 3;
    min-height: 100%;
  }
  .bento .b-left {
    grid-column: 3;
    grid-row: 1;
  }
  .bento .b-right {
    grid-column: 4;
    grid-row: 1;
  }
}
.bento-s {
  padding: 1.5rem;
  border-radius: 1.5rem;
  background: var(--c-surface-c-lowest);
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  transition: box-shadow 0.2s;
  cursor: pointer;
}
.bento-s-link {
  text-decoration: none;
  color: inherit;
}
.bento-s:hover {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}
.live-pill {
  position: absolute;
  top: 1rem;
  right: 1rem;
  z-index: 10;
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  background: var(--c-error);
  color: #fff;
  font-size: 10px;
  font-weight: 900;
  padding: 0.25rem 0.65rem 0.25rem 0.4rem;
  border-radius: 999px;
  animation: pulse 2s ease-in-out infinite;
}
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.85;
  }
}
.live-dot {
  width: 0.375rem;
  height: 0.375rem;
  background: #fff;
  border-radius: 50%;
}
.bento-body {
  padding: 2rem;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 16rem;
}
.bento-icon.big {
  width: 3.5rem;
  height: 3.5rem;
  background: var(--c-primary-10);
  color: var(--c-primary);
  border-radius: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1.5rem;
}
.bento-icon.big .bento-ico-lg {
  width: 2.25rem;
  height: 2.25rem;
  object-fit: contain;
  display: block;
}
.bento-txt {
  margin-top: auto;
}
.bento-h {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--c-on-surface);
}
.bento-h.sm {
  font-size: 1.125rem;
}
.bento-p {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
  color: var(--c-on-surface-variant);
  line-height: 1.4;
}
.bento-p.sm {
  font-size: 0.75rem;
  margin-top: 0.25rem;
}
.bento-icon.terr {
  width: 2.5rem;
  height: 2.5rem;
  background: var(--c-tertiary-10);
  color: var(--c-tertiary);
  border-radius: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1rem;
}
.bento-icon.terr .bento-ico-sm,
.bento-icon.pri .bento-ico-sm {
  width: 1.35rem;
  height: 1.35rem;
  object-fit: contain;
  display: block;
}
.bento-icon.pri {
  width: 2.5rem;
  height: 2.5rem;
  background: var(--c-primary-10);
  color: var(--c-primary);
  border-radius: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1rem;
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
  font-size: 1.5rem;
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
  gap: 1rem;
  padding: 1.25rem;
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
  width: 3rem;
  height: 3rem;
  border-radius: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.news-icon .news-ico-img {
  width: 1.4rem;
  height: 1.4rem;
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
  font-size: 1rem;
  color: var(--c-on-surface);
}
.news-body {
  margin: 0.15rem 0 0;
  font-size: 0.875rem;
  color: var(--c-on-surface-variant);
  line-height: 1.4;
}
.news-time {
  font-size: 0.75rem;
  color: var(--c-outline);
  font-weight: 500;
  white-space: nowrap;
  flex-shrink: 0;
}
</style>
