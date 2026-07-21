<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import { fetchShareCatalogRows } from '@/api/schemes/shareCatalog'
import { shareAddToCloud } from '@/api/schemes/shareAddToCloud'
import { ApiError } from '@/api/client'
import type { SchemeDownloadRow } from '@/api/schemes/shareCatalog'

const router = useRouter()

const schemeIdInput = ref('')
const loading = ref(false)
const downloadingId = ref<string | null>(null)
const rows = ref<SchemeDownloadRow[]>([])

const successVisible = ref(false)
const successMessage = ref('')

function goBack() {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'lobby' })
}

async function loadRows(keyword = ''): Promise<void> {
  loading.value = true
  try {
    rows.value = await fetchShareCatalogRows(keyword)
    if (keyword && rows.value.length === 0) {
      ElMessage.info('未找到匹配方案')
    }
  } catch {
    ElMessage.error('加载方案列表失败')
    rows.value = []
  } finally {
    loading.value = false
  }
}

function onSearch() {
  void loadRows(schemeIdInput.value.trim())
}

function onReset() {
  schemeIdInput.value = ''
  void loadRows()
}

function formatFund(yuan: number) {
  return `${yuan.toLocaleString('zh-CN', { maximumFractionDigits: 1 })} 元`
}

function onSuccessGoCloud(): void {
  successVisible.value = false
  void router.push({ name: 'cloud' })
}

function onSuccessStay(): void {
  successVisible.value = false
}

async function onDownload(row: SchemeDownloadRow): Promise<void> {
  if (downloadingId.value) return
  downloadingId.value = row.schemeId
  try {
    const result = await shareAddToCloud(row.schemeId)
    successMessage.value = `方案「${result.definition.schemeName}」已下载成功，已添加至云端（${result.instance.statusLabel}）。您可前往云端中心查看并开启。`
    successVisible.value = true
  } catch (e) {
    ElMessage.error(e instanceof ApiError ? e.message : '下载失败')
  } finally {
    downloadingId.value = null
  }
}

onMounted(() => {
  void loadRows()
})
</script>

<template>
  <div class="sdw" data-page="scheme-download">
    <header class="sdw-header" role="banner">
      <div class="sdw-header-top">
        <button type="button" class="sdw-back" aria-label="返回" @click="goBack">
          <span class="material-sym" aria-hidden="true">arrow_back_ios_new</span>
        </button>
        <h1 class="sdw-title">方案下载</h1>
        <span class="sdw-header-spacer" aria-hidden="true" />
      </div>
    </header>

    <main class="sdw-main">
      <section class="sdw-search-card" aria-label="方案搜索">
        <el-input
          v-model="schemeIdInput"
          clearable
          size="large"
          placeholder="请输入方案 ID"
          class="sdw-search-inp"
          @keyup.enter="onSearch"
          @clear="onReset"
        />
        <el-button type="primary" size="large" round class="sdw-search-btn" @click="onSearch">
          搜索
        </el-button>
      </section>

      <section class="sdw-table-card" aria-label="可下载方案列表">
        <el-table
          v-loading="loading"
          :data="rows"
          stripe
          class="sdw-table detail-bet-table"
          size="small"
          empty-text="暂无匹配方案"
          :style="{ width: '100%' }"
        >
          <el-table-column prop="schemeName" label="方案" min-width="72" />
          <el-table-column prop="lotteryLabel" label="彩种" min-width="96" class-name="sdw-td-wrap" />
          <el-table-column prop="playMethod" label="玩法" min-width="88" class-name="sdw-td-wrap" />
          <el-table-column label="方案资金" min-width="72" align="right">
            <template #default="{ row }">
              <span class="sdw-fund">{{ formatFund(row.fundYuan) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" min-width="56" align="center" fixed="right">
            <template #default="{ row }">
              <button
                type="button"
                class="sdw-dl-btn"
                :class="{ 'sdw-dl-btn--busy': downloadingId === row.schemeId }"
                :disabled="downloadingId === row.schemeId"
                :aria-busy="downloadingId === row.schemeId"
                :aria-label="`下载方案 ${row.schemeName}`"
                @click="onDownload(row)"
              >
                <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                  <path
                    fill="currentColor"
                    d="M19.35 10.04A7.49 7.49 0 0012 4C9.11 4 6.6 5.64 5.35 8.04A5.994 5.994 0 000 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM17 13l-5 5-5-5h3V9h4v4h3z"
                  />
                </svg>
              </button>
            </template>
          </el-table-column>
        </el-table>
        <p v-if="rows.length > 0" class="sdw-hint">
          共 {{ rows.length }} 条；下载将创建跟单方案并添加至云端（待开启）。
        </p>
      </section>
    </main>

    <ConfirmDialog
      v-model="successVisible"
      title="下载成功"
      :message="successMessage"
      icon="check_circle"
      tone="primary"
      confirm-text="前往云端中心"
      cancel-text="继续浏览"
      @confirm="onSuccessGoCloud"
      @cancel="onSuccessStay"
    />
  </div>
</template>

<style scoped>
.sdw {
  --sdw-primary: #0066ff;
  --sdw-surface: #f7f9fb;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--sdw-surface);
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: calc(5rem + env(safe-area-inset-bottom));
}

.sdw-header {
  flex-shrink: 0;
  padding-top: env(safe-area-inset-top);
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
}

.sdw-header-top {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 0.5rem;
  height: var(--page-titlebar-height);
  min-height: var(--page-titlebar-height);
  box-sizing: border-box;
  padding: 0 var(--page-titlebar-pad-x);
}

.sdw-back {
  justify-self: start;
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: #0f172a;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sdw-back:focus-visible {
  outline: 2px solid var(--sdw-primary);
  outline-offset: 2px;
}

.sdw-back .material-sym {
  font-size: var(--page-titlebar-back-icon-size);
  color: #191c1e;
}

.sdw-title {
  margin: 0;
  justify-self: center;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.sdw-header-spacer {
  justify-self: end;
  width: var(--page-titlebar-action-size);
}

.sdw-main {
  flex: 1;
  min-height: 0;
  padding: 1rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-width: 40rem;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.sdw-search-card {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem;
  align-items: center;
  padding: 1rem;
  background: #fff;
  border-radius: 0.75rem;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
}

.sdw-search-inp {
  flex: 1 1 10rem;
  min-width: 0;
}

.sdw-search-btn {
  flex-shrink: 0;
  min-width: 5rem;
  font-weight: 600;
}

.sdw-table-card {
  background: #fff;
  border-radius: 0.75rem;
  padding: 0.5rem 0.25rem 0.75rem;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
  overflow: hidden;
}

.sdw-table :deep(.el-table) {
  --el-table-border-color: transparent;
  --el-table-header-bg-color: #f0f5ff;
  --el-table-header-text-color: #0050cb;
}

.sdw-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.sdw-table :deep(.el-table__header th) {
  font-size: 11px;
  font-weight: 700;
}

.sdw-table :deep(.el-table__header th .cell) {
  text-align: center;
}

.sdw-table :deep(.el-table__body .el-table__cell) {
  font-size: 12px;
  vertical-align: middle;
}

.sdw-table :deep(.sdw-td-wrap .cell) {
  white-space: normal !important;
  word-break: break-word;
  line-height: 1.45;
}

.sdw-fund {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: #0f172a;
}

.sdw-dl-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: linear-gradient(180deg, #0066ff 0%, #0050cb 100%);
  color: #fff;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 102, 255, 0.25);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.sdw-dl-btn:active {
  transform: scale(0.96);
}

.sdw-dl-btn--busy {
  opacity: 0.65;
  cursor: wait;
}

.sdw-dl-btn:focus-visible {
  outline: 2px solid #0066ff;
  outline-offset: 2px;
}

.sdw-hint {
  margin: 0.5rem 1rem 0;
  font-size: 11px;
  line-height: 1.45;
  color: #727687;
}
</style>
