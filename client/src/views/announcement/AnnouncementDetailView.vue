<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { demoAppBrand } from '@/demo/demoAccount'
import { fetchAnnouncementDetail, type AnnouncementDetail } from '@/api/content/announcements'

/**
 * 公告详情页（数字精算主义 / Digital Actuarialism）
 * - 与 client/DESIGN.md 对齐：无 1px 实线分割、色阶分层、磨砂玻璃顶栏、editorial 字体层级
 * - 数据驱动：通过路由参数 :id 在 ANNOUNCEMENTS 中检索；缺省回退到 USDT-TON 渠道公告
 * - 文案中的高亮与链接通过 v-html 渲染（仅来自本地可信 mock）
 */

interface AnnouncementInfoCard {
  label: string
  url: string
}

interface AnnouncementData {
  id: string
  /** 顶部 chip 文案（英文大写效果由 CSS 处理） */
  category: string
  date: string
  title: string
  hero: string
  greeting?: string
  /** 段落中可包含 <em> 与 <strong> 等内联高亮 */
  paragraphs: string[]
  info?: AnnouncementInfoCard
  noteTitle: string
  /** 列表项可包含 <strong>、<a class="ann-link"> 等内联标签 */
  notes: string[]
  signature: string[]
}

const HERO_IMG = '/images/lobby/feature-announcement.png'

const ANNOUNCEMENTS: Record<string, AnnouncementData> = {
  'usdt-ton': {
    id: 'usdt-ton',
    category: 'Update',
    date: '2025/09/15',
    title: '新增【USDT-TON】渠道公告',
    hero: HERO_IMG,
    greeting: '亲爱的会员您好：',
    paragraphs: [
      '为了让您"充值""提款"更加快速、安全、便利，平台于 <em>09月13日</em> 正式新增 <em>【USDT-TON】</em> 渠道。',
      'TON (The Open Network) 是一项旨在提供高性能区块链服务的技术。通过此次升级，您可以享受到极速到账体验以及更低的网络手续费。',
    ],
    info: {
      label: 'TON区块链验证网址',
      url: 'https://tonviewer.com/',
    },
    noteTitle: '【注意事项】',
    notes: [
      '充值前请务必确认选择正确的 <strong>TON网络</strong>，若选择错误网络将导致资金无法找回。',
      '如有任何疑问，请点击联系 <a class="ann-link" href="javascript:void(0)">7x24小时在线客服</a> 咨询。',
      '为了您的账号安全，建议定期更换并保护好您的交易支付密码。',
    ],
    signature: ['感谢您一直以来对我们平台的支持与信任。', '祝您生活愉快，游戏顺心！'],
  },

  'version-2-4': {
    id: 'version-2-4',
    category: 'Release',
    date: '2025/09/01',
    title: `${demoAppBrand} 2.4 版本上线公告`,
    hero: HERO_IMG,
    greeting: '亲爱的会员您好：',
    paragraphs: [
      `<em>${demoAppBrand} 2.4</em> 已正式上线，本次升级聚焦于「方案分析」「跟单大厅」与「资金侧栏」三大主线，整体响应速度提升约 <em>32%</em>。`,
      '我们重新设计了多维方案分析工具，可在同一视图内对比方案命中率、资金曲线与风险阈值，帮助您更高效地完成精算决策。',
    ],
    info: {
      label: '版本说明与更新日志',
      url: 'https://docs.example.com/release/2.4',
    },
    noteTitle: '【更新提示】',
    notes: [
      '建议在 <strong>Wi-Fi 环境</strong> 下完成本次更新，下载体积约 28MB。',
      '如更新后遇到异常，请点击联系 <a class="ann-link" href="javascript:void(0)">7x24小时在线客服</a> 反馈问题。',
      '后续小版本将持续推送，建议在「会员中心 → 设置」中开启自动更新。',
    ],
    signature: ['感谢您与我们一同精进每一次决策。', '祝您游戏顺心，收益稳健！'],
  },

  n1: {
    id: 'n1',
    category: 'Update',
    date: '2025/09/15',
    title: '充值须知：支付渠道升级提醒',
    hero: HERO_IMG,
    greeting: '亲爱的会员您好：',
    paragraphs: [
      '为保障您的资金安全，平台已对全部支付渠道进行 <em>合规化升级</em>，新版本默认启用动态地址与 <em>实时风控</em>。',
      '充值前请务必查看最新支付指南，按页面指引选择对应网络与币种，避免错链转账造成资金不可追回。',
    ],
    info: {
      label: '最新支付指南',
      url: 'https://help.example.com/pay/guide',
    },
    noteTitle: '【充值须知】',
    notes: [
      '请确认 <strong>充值币种与网络</strong> 一致，错链转账无法找回。',
      '充值地址可能 <strong>每次刷新更换</strong>，请勿反复使用旧地址。',
      '如未在 30 分钟内到账，请联系 <a class="ann-link" href="javascript:void(0)">7x24小时在线客服</a>。',
    ],
    signature: ['感谢您对平台的信任与支持。', '祝您生活愉快，游戏顺心！'],
  },

  n2: {
    id: 'n2',
    category: 'Maintenance',
    date: '2025/09/12',
    title: '服务器维护：每周例行更新',
    hero: HERO_IMG,
    greeting: '亲爱的会员您好：',
    paragraphs: [
      '为持续优化服务质量，平台将于 <em>本周五凌晨 02:00–04:00</em> 进行系统扩容与例行维护。',
      '维护期间部分功能可能无法使用，登录、投注与出入金可能短暂不可用，敬请提前安排相关操作。',
    ],
    noteTitle: '【维护说明】',
    notes: [
      '维护时间：<strong>周五 02:00 - 04:00</strong>，预计 2 小时内完成。',
      '维护期间已下注方案 <strong>不受影响</strong>，照常结算。',
      '如有紧急问题请联系 <a class="ann-link" href="javascript:void(0)">7x24小时在线客服</a>。',
    ],
    signature: ['感谢您的理解与支持。', '我们将以更稳健的服务回馈每一位会员。'],
  },

  n3: {
    id: 'n3',
    category: 'Security',
    date: '2025/09/05',
    title: '安全中心：账户保护功能加强',
    hero: HERO_IMG,
    greeting: '亲爱的会员您好：',
    paragraphs: [
      '为持续提升账户安全水位，安全中心新增 <em>物理硬件密钥</em> 二次验证、可疑登录提醒与异地交易复核。',
      '建议您前往「会员中心 → 安全设置」开启全部安全选项，并妥善保管支付密码。',
    ],
    info: {
      label: '安全中心入口',
      url: 'https://app.example.com/member/security',
    },
    noteTitle: '【安全建议】',
    notes: [
      '务必启用 <strong>硬件密钥二次验证</strong> 或动态口令。',
      '不要在公共网络环境下输入支付密码。',
      '如发现异常登录请立即联系 <a class="ann-link" href="javascript:void(0)">7x24小时在线客服</a>。',
    ],
    signature: ['感谢您一直以来的信任。', '愿您账户安全，游戏顺心。'],
  },
}

const route = useRoute()
const router = useRouter()
const apiDetail = ref<AnnouncementDetail | null>(null)

const announcement = computed<AnnouncementData>(() => {
  const raw = route.params.id ?? route.query.id ?? ''
  const id = String(Array.isArray(raw) ? raw[0] : raw)
  return ANNOUNCEMENTS[id] ?? ANNOUNCEMENTS['usdt-ton']
})

watch(
  () => [announcement.value, apiDetail.value] as const,
  ([a, api]) => {
    document.title = `公告详情 · ${api?.title ?? a.title} · ${demoAppBrand}`
  },
  { immediate: true },
)

onMounted(async () => {
  const raw = route.params.id ?? route.query.id ?? ''
  const id = String(Array.isArray(raw) ? raw[0] : raw)
  if (!id) return
  try {
    apiDetail.value = await fetchAnnouncementDetail(id)
  } catch {
    /** 缺省仍展示 Mock 富文本 */
  }
})

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/')
}
</script>

<template>
  <div class="ann">
    <header class="ann-header" role="banner">
      <button type="button" class="ann-icon-btn ann-back" aria-label="返回" @click="goBack">
        <span class="ann-ms" aria-hidden="true">arrow_back_ios_new</span>
      </button>
      <h1 class="ann-header-title">公告详情</h1>
      <span class="ann-head-spacer" aria-hidden="true" />
    </header>

    <main class="ann-main">
      <article v-if="apiDetail" class="ann-article ann-article--api">
        <header class="ann-article-head">
          <div class="ann-meta">
            <span class="ann-chip">Notice</span>
            <span class="ann-date">{{ apiDetail.date }}</span>
          </div>
          <h2 class="ann-title">{{ apiDetail.title }}</h2>
          <div class="ann-accent" aria-hidden="true" />
        </header>
        <div class="ann-body ann-body--api cms-rich-html" v-html="apiDetail.bodyHtml" />
      </article>

      <article v-else class="ann-article" :key="announcement.id">
        <header class="ann-article-head">
          <div class="ann-meta">
            <span class="ann-chip">{{ announcement.category }}</span>
            <span class="ann-date">{{ announcement.date }}</span>
          </div>
          <h2 class="ann-title">{{ announcement.title }}</h2>
          <div class="ann-accent" aria-hidden="true" />
        </header>

        <figure class="ann-hero" v-if="announcement.hero">
          <img
            :src="announcement.hero"
            :alt="announcement.title"
            class="ann-hero-img"
            loading="lazy"
            decoding="async"
          />
        </figure>

        <div class="ann-body">
          <p v-if="announcement.greeting" class="ann-greeting">{{ announcement.greeting }}</p>

          <p v-for="(p, i) in announcement.paragraphs" :key="i" class="ann-p" v-html="p" />

          <aside v-if="announcement.info" class="ann-info">
            <span class="ann-ms ann-info-ico" aria-hidden="true">verified</span>
            <div class="ann-info-body">
              <p class="ann-info-label">{{ announcement.info.label }}</p>
              <a
                class="ann-info-link"
                :href="announcement.info.url"
                target="_blank"
                rel="noopener noreferrer"
              >
                {{ announcement.info.url }}
              </a>
            </div>
          </aside>

          <section class="ann-notes">
            <div class="ann-notes-head">
              <span class="ann-ms ann-notes-ico" aria-hidden="true">info</span>
              <h3 class="ann-notes-title">{{ announcement.noteTitle }}</h3>
            </div>
            <ol class="ann-notes-list">
              <li v-for="(n, i) in announcement.notes" :key="i" class="ann-notes-item">
                <span class="ann-notes-num">{{ String(i + 1).padStart(2, '0') }}.</span>
                <p class="ann-notes-text" v-html="n" />
              </li>
            </ol>
          </section>

          <footer class="ann-sign">
            <p
              v-for="(s, i) in announcement.signature"
              :key="i"
              class="ann-sign-line"
            >
              {{ s }}
            </p>
          </footer>
        </div>
      </article>

      <div class="ann-footer-logo" aria-hidden="true">ACTUARIALISM</div>
    </main>
  </div>
</template>

<style scoped>
.ann {
  overflow-x: hidden;
  --ann-surface: #f7f9fb;
  --ann-surface-low: #f2f4f6;
  --ann-surface-lowest: #ffffff;
  --ann-on-surface: #191c1e;
  --ann-on-variant: #424656;
  --ann-outline: #c2c6d8;
  --ann-primary: #0050cb;
  --ann-primary-strong: #0066ff;
  --ann-primary-soft: rgba(0, 80, 203, 0.08);
  min-height: 100dvh;
  background: var(--ann-surface);
  color: var(--ann-on-surface);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  padding-bottom: env(safe-area-inset-bottom);
}

.ann-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: var(--page-titlebar-icon-size);
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

/* ======== Top App Bar ======== */
.ann-header {
  position: sticky;
  top: 0;
  z-index: 50;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 0.5rem;
  height: calc(var(--page-titlebar-height) + env(safe-area-inset-top));
  min-height: calc(var(--page-titlebar-height) + env(safe-area-inset-top));
  box-sizing: border-box;
  padding: env(safe-area-inset-top) var(--page-titlebar-pad-x) 0;
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(28px);
  -webkit-backdrop-filter: blur(28px);
  box-shadow: 0 8px 32px rgba(25, 28, 30, 0.06);
}

.ann-icon-btn {
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  border: none;
  border-radius: 0.75rem;
  background: transparent;
  color: var(--ann-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s, transform 0.2s;
}

.ann-icon-btn:hover {
  background: rgba(0, 80, 203, 0.06);
}

.ann-icon-btn:active {
  transform: scale(0.94);
}

.ann-icon-btn:focus-visible {
  outline: 2px solid var(--ann-primary-strong);
  outline-offset: 2px;
}

.ann-back {
  justify-self: start;
}

.ann-back .ann-ms {
  font-size: var(--page-titlebar-back-icon-size);
}

.ann-head-spacer {
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  justify-self: end;
}

.ann-header-title {
  margin: 0;
  justify-self: center;
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.0625rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--ann-on-surface);
}

/* ======== Main ======== */
.ann-main {
  width: 100%;
  max-width: 36rem;
  margin: 0 auto;
  padding: 1.5rem var(--page-gutter) 4rem;
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

/* ======== Article Card ======== */
.ann-article {
  min-width: 0;
  overflow-x: hidden;
  background: var(--ann-surface-lowest);
  border-radius: 1.5rem;
  padding: var(--card-pad);
  box-shadow: 0 24px 60px -28px rgba(15, 23, 42, 0.18),
    0 4px 16px -8px rgba(15, 23, 42, 0.06);
  animation: ann-rise 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes ann-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.ann-article-head {
  margin-bottom: 1.75rem;
}

.ann-meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
}

.ann-chip {
  background: var(--ann-primary-strong);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  padding: 0.3rem 0.7rem;
  border-radius: 999px;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
}

.ann-date {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--ann-on-variant);
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
}

.ann-title {
  margin: 0 0 1rem;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: clamp(1.5rem, 4.6vw, 2.25rem);
  font-weight: 800;
  line-height: 1.18;
  letter-spacing: -0.02em;
  color: var(--ann-on-surface);
}

.ann-accent {
  width: 4rem;
  height: 0.3125rem;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--ann-primary), var(--ann-primary-strong));
}

/* ======== Hero ======== */
.ann-hero {
  margin: 0 0 1.75rem;
  border-radius: 1.25rem;
  overflow: hidden;
  aspect-ratio: 21 / 9;
  background: var(--ann-surface-low);
  position: relative;
}

.ann-hero-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.7s ease;
}

.ann-hero:hover .ann-hero-img {
  transform: scale(1.03);
}

/* ======== Body ======== */
.ann-body {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  color: var(--ann-on-variant);
  line-height: 1.75;
  font-size: 0.9375rem;
}

.ann-body--api {
  max-width: 100%;
  min-width: 0;
}

.ann-body--api :deep(p) {
  margin: 0 0 0.85rem;
}

.ann-body--api :deep(p:last-child) {
  margin-bottom: 0;
}

.ann-body--api :deep(a) {
  color: var(--ann-primary);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.ann-body--api :deep(strong) {
  color: var(--ann-on-surface);
  font-weight: 700;
}

.ann-body--api :deep(em) {
  font-style: normal;
  color: var(--ann-primary);
  font-weight: 700;
}

.ann-greeting {
  margin: 0;
  font-weight: 700;
  color: var(--ann-on-surface);
}

.ann-p {
  margin: 0;
}

.ann-p :deep(em) {
  font-style: normal;
  color: var(--ann-primary);
  font-weight: 700;
}

.ann-p :deep(strong) {
  color: var(--ann-on-surface);
}

/* ======== Info Card ======== */
.ann-info {
  display: flex;
  align-items: flex-start;
  gap: 0.875rem;
  padding: var(--card-pad);
  background: var(--ann-primary-soft);
  border-radius: 1rem;
  border-left: 4px solid var(--ann-primary);
}

.ann-info-ico {
  color: var(--ann-primary);
  margin-top: 0.125rem;
  flex-shrink: 0;
}

.ann-info-body {
  min-width: 0;
  flex: 1;
}

.ann-info-label {
  margin: 0 0 0.25rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--ann-on-surface);
  letter-spacing: 0.01em;
}

.ann-info-link {
  display: inline-block;
  color: var(--ann-primary);
  font-weight: 600;
  word-break: break-all;
  text-decoration: none;
  transition: color 0.15s, text-decoration-color 0.15s;
  text-decoration-line: underline;
  text-decoration-color: transparent;
  text-decoration-thickness: 2px;
  text-underline-offset: 4px;
}

.ann-info-link:hover {
  color: var(--ann-primary-strong);
  text-decoration-color: currentColor;
}

/* ======== Notes Box ======== */
.ann-notes {
  margin-top: 0.5rem;
  background: var(--ann-surface-low);
  border-radius: 1.25rem;
  padding: var(--card-pad);
}

.ann-notes-head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
  color: var(--ann-on-surface);
}

.ann-notes-ico {
  color: var(--ann-primary);
  font-size: 1.25rem;
}

.ann-notes-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--ann-on-surface);
}

.ann-notes-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.ann-notes-item {
  display: flex;
  gap: 0.625rem;
  align-items: flex-start;
}

.ann-notes-num {
  color: var(--ann-primary);
  font-weight: 800;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 0.875rem;
  letter-spacing: 0.02em;
  flex-shrink: 0;
  line-height: 1.55;
  font-variant-numeric: tabular-nums;
}

.ann-notes-text {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.65;
  color: var(--ann-on-variant);
}

.ann-notes-text :deep(strong) {
  color: var(--ann-on-surface);
  font-weight: 700;
}

.ann-notes-text :deep(.ann-link) {
  color: var(--ann-primary);
  font-weight: 700;
  cursor: pointer;
  text-decoration: none;
  transition: color 0.15s;
}

.ann-notes-text :deep(.ann-link:hover) {
  color: var(--ann-primary-strong);
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* ======== Signature ======== */
.ann-sign {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  position: relative;
}

.ann-sign::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 0.0625rem;
  background: linear-gradient(90deg, transparent, rgba(194, 198, 216, 0.5), transparent);
}

.ann-sign-line {
  margin: 0;
  font-size: 0.8125rem;
  font-style: italic;
  color: var(--ann-on-variant);
  line-height: 1.7;
}

/* ======== Footer Logo ======== */
.ann-footer-logo {
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 0.9375rem;
  font-weight: 800;
  letter-spacing: 0.32em;
  color: var(--ann-primary);
  opacity: 0.28;
  user-select: none;
}
</style>
