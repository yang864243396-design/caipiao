<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const schemeTitle = ref(decodeURIComponent(String(route.query.title ?? '') || '方案'))
const editingTitle = ref(false)
const tempTitle = ref(schemeTitle.value)

function toggleTitleEdit() {
  editingTitle.value = true
  tempTitle.value = schemeTitle.value
}

function confirmTitleEdit() {
  const t = tempTitle.value.trim()
  schemeTitle.value = t || '方案'
  editingTitle.value = false
}

function cancelTitleEdit() {
  editingTitle.value = false
  tempTitle.value = schemeTitle.value
}

interface SchemeRoundRule {
  mult: number
  afterHit: number
  afterMiss: number
}

function defaultRows(): SchemeRoundRule[] {
  return [
    { mult: 0, afterHit: 2, afterMiss: 1 },
    { mult: 1, afterHit: 2, afterMiss: 3 },
    { mult: 3, afterHit: 2, afterMiss: 1 },
  ]
}

const rows = ref<SchemeRoundRule[]>(defaultRows())

function goBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }
  router.push({
    name: 'advanced-scheme-edit',
    params: { schemeId: String(route.params.schemeId) },
    query: { ...route.query },
  })
}

function addRound() {
  rows.value.push({ mult: 0, afterHit: 1, afterMiss: 1 })
}

function removeRow(index: number) {
  rows.value.splice(index, 1)
}

const MULT_CAP = 200_000

function onSave() {
  const bad = rows.value.some((r) => !Number.isFinite(r.mult) || r.mult > MULT_CAP)
  if (bad) {
    ElMessage.warning(`倍数须在 0～${MULT_CAP} 之间`)
    return
  }
  ElMessage.success('已保存（演示）')
  router.back()
}
</script>

<template>
  <div class="ase">
    <header class="ase-header">
      <div class="ase-header-bar">
        <button type="button" class="ase-back" aria-label="返回" @click="goBack">
          <svg class="ase-back-ico" viewBox="0 0 24 24" width="22" height="22" aria-hidden="true">
            <path fill="currentColor" d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z" />
          </svg>
        </button>
        <div class="ase-title-wrap">
          <template v-if="!editingTitle">
            <h1 class="ase-title">{{ schemeTitle }}</h1>
            <button type="button" class="ase-title-edit" aria-label="编辑方案名称" @click="toggleTitleEdit">
              <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                <path fill="currentColor"
                  d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1.004 1.004 0 000-1.42l-2.34-2.34a1 1 0 00-1.42 0l-1.83 1.83 3.75 3.75 1.84-1.82z" />
              </svg>
            </button>
          </template>
          <div v-else class="ase-title-inline">
            <el-input v-model="tempTitle" size="small" maxlength="48" placeholder="方案名称"
              @keyup.enter="confirmTitleEdit" />
            <el-button link type="primary" size="small" @click="confirmTitleEdit">确定</el-button>
            <el-button link size="small" @click="cancelTitleEdit">取消</el-button>
          </div>
        </div>
        <div class="ase-header-actions">
          <el-button type="primary" plain size="small" class="ase-add-row" @click="addRound">新增局数</el-button>
        </div>
      </div>
    </header>

    <p class="ase-hint ase-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>

    <main class="ase-main">
      <div class="ase-table-card">
        <el-table :data="rows" class="detail-bet-table ase-el-table" size="small" stripe empty-text="暂无局数">
          <el-table-column label="局数" :min-width="48" align="center">
            <template #default="{ $index }">
              {{ $index + 1 }}
            </template>
          </el-table-column>
          <el-table-column label="倍数" :min-width="56" align="center" class-name="ase-cell-input">
            <template #default="{ row }">
              <el-input-number v-model="row.mult" :min="0" :max="MULT_CAP" size="small" :controls="false" />
            </template>
          </el-table-column>
          <el-table-column label="中后" :min-width="56" align="center" class-name="ase-cell-input">
            <template #default="{ row }">
              <el-input-number v-model="row.afterHit" :min="0" size="small" :controls="false" />
            </template>
          </el-table-column>
          <el-table-column label="挂后" :min-width="56" align="center" class-name="ase-cell-input">
            <template #default="{ row }">
              <el-input-number v-model="row.afterMiss" :min="0" size="small" :controls="false" />
            </template>
          </el-table-column>
          <el-table-column label="删除" :min-width="44" align="center" class-name="ase-cell-del">
            <template #default="{ $index }">
              <button type="button" class="ase-del-btn" :disabled="rows.length <= 1"
                :aria-label="`删除第 ${$index + 1} 局`" @click="removeRow($index)">
                <svg class="ase-del-svg" viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                  <path fill="currentColor"
                    d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z" />
                </svg>
              </button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </main>

    <footer class="ase-footer">
      <el-button type="primary" class="ase-save" @click="onSave">保存</el-button>
    </footer>
  </div>
</template>

<style scoped>
.ase {
  --ase-surface: #f7f9fb;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--ase-surface);
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: env(safe-area-inset-bottom);
}

.ase-header {
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  box-shadow: 0 12px 40px rgba(25, 28, 30, 0.06);
  border-bottom: 1px solid rgba(226, 232, 240, 0.8);
}

.ase-header-bar {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 0.5rem;
  padding: max(0.75rem, env(safe-area-inset-top)) 0.75rem 0.75rem;
}

.ase-back {
  justify-self: start;
  width: 2.25rem;
  height: 2.25rem;
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

.ase-title-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  min-width: 0;
  grid-column: 2;
}

.ase-title {
  margin: 0;
  max-width: 12rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 1rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.ase-title-edit {
  flex-shrink: 0;
  width: 1.75rem;
  height: 1.75rem;
  padding: 0;
  border: none;
  border-radius: 0.375rem;
  background: #f1f5f9;
  color: #0066ff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ase-title-inline {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.ase-title-inline :deep(.el-input) {
  width: min(52vw, 12rem);
}

.ase-header-actions {
  justify-self: end;
  display: flex;
}

.ase-hint {
  margin: 0;
  padding: 0.5rem 1rem;
  font-size: 0.6875rem;
  line-height: 1.45;
}

.ase-hint--danger {
  color: #ba1a1a;
}

.ase-main {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 1rem;
}

.ase-table-card {
  background: #fff;
  border-radius: 0.75rem;
  overflow: hidden;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

/* 可编辑列：整块单元格即为输入区域，无外框叠加 */
.detail-bet-table :deep(td.ase-cell-input) {
  padding: 0;
  vertical-align: middle;
}

.detail-bet-table :deep(td.ase-cell-input .cell) {
  padding: 0 !important;
}

.detail-bet-table :deep(td.ase-cell-input .el-input-number) {
  display: flex;
  width: 100%;
  vertical-align: unset;
}

.detail-bet-table :deep(td.ase-cell-input .el-input) {
  flex: 1;
  width: auto !important;
  min-width: 0;
}

.detail-bet-table :deep(td.ase-cell-input .el-input__wrapper) {
  box-shadow: none !important;
  border: none !important;
  border-radius: 0;
  background: transparent !important;
  min-height: 2.375rem;
  padding-inline: 0.5rem;
}

.detail-bet-table :deep(td.ase-cell-input .el-input__inner) {
  text-align: center;
}

.detail-bet-table :deep(.el-table) {
  --el-table-border-color: transparent;
  --el-table-bg-color: transparent;
  --el-table-header-bg-color: #f8fafc;
}

.detail-bet-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.detail-bet-table :deep(.el-table__header th) {
  font-size: 10px;
  font-weight: 700;
  color: #64748b !important;
  text-transform: uppercase;
}

.detail-bet-table :deep(.el-table__header th .cell) {
  text-align: center;
}

.detail-bet-table :deep(td.ase-cell-del .cell) {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 0.25rem !important;
}

.ase-del-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 999px;
  background: #e2e8f0;
  color: #64748b;
  cursor: pointer;
  flex-shrink: 0;
  -webkit-tap-highlight-color: transparent;
}

.ase-del-btn:hover:not(:disabled) {
  background: #cbd5e1;
  color: #dc2626;
}

.ase-del-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.ase-del-svg {
  display: block;
  flex-shrink: 0;
}

.ase-footer {
  flex-shrink: 0;
  padding: 0.75rem 1rem max(0.75rem, env(safe-area-inset-bottom));
  padding-top: 0.5rem;
  background: rgba(255, 255, 255, 0.96);
  border-top: 1px solid #e2e8f0;
  backdrop-filter: blur(14px);
  display: flex;
  justify-content: center;
}

.ase-save {
  width: min(100%, 20rem);
  height: 2.75rem;
  margin: 0;
  border-radius: 0.625rem;
  font-weight: 700;
  background: #0066ff;
  border: none;
}
</style>
