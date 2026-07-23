<script setup lang="ts">
import { onActivated, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ApiError } from '@/api/client'
import {
  createClientSchemeTemplate,
  getClientSchemeTemplate,
  isNewSchemeTemplateRoute,
  isSchemeTemplateId,
  saveClientSchemeTemplate,
} from '@/api/schemeTemplates'
import { refreshSchemeTemplatesState } from '@/composables/useSchemeTemplateLibrary'
import {
  getSchemeDefinition,
  isMemberDefinitionId,
  saveSchemeRounds,
  type SchemeRoundRule,
} from '@/api/schemes/betMultiplier'
import {
  getDraftAdvancedTemplate,
  isDraftAdvancedTemplateId,
  newDraftAdvancedTemplateId,
  upsertDraftAdvancedTemplate,
} from '@/utils/draftAdvancedTemplates'
import { SCHEME_DRAFT_ID, isDraftSchemeId } from '@/utils/schemeDraftStorage'

const route = useRoute()
const router = useRouter()

const schemeTitle = ref(decodeURIComponent(String(route.query.title ?? '') || '方案'))
const editingTitle = ref(false)
const tempTitle = ref(schemeTitle.value)
const templateMemberOwned = ref(false)

function routeParamId(): string {
  return String(route.params.schemeId ?? '').trim()
}

function ownerSchemeContext(): string {
  const fromQuery = route.query.schemeId != null ? String(route.query.schemeId).trim() : ''
  if (fromQuery) return fromQuery
  const fromParam = routeParamId()
  if (fromParam && !isSchemeTemplateId(fromParam) && !isDraftAdvancedTemplateId(fromParam)) {
    return fromParam
  }
  return ''
}

function isDraftOwner(): boolean {
  return isDraftSchemeId(ownerSchemeContext())
}

function ownerDefinitionId(): string {
  const ctx = ownerSchemeContext()
  return isMemberDefinitionId(ctx) ? ctx : ''
}

function templatesFetchDefinitionId(): string {
  const ctx = ownerSchemeContext()
  if (isMemberDefinitionId(ctx) || isDraftSchemeId(ctx)) return ctx
  return ''
}

function memberDefinitionId(): string {
  const fromParam = routeParamId()
  return isMemberDefinitionId(fromParam) ? fromParam : ''
}

function templateId(): string {
  const id = routeParamId()
  if (isNewSchemeTemplateRoute(id)) return ''
  if (isSchemeTemplateId(id) || isDraftAdvancedTemplateId(id)) return id
  return ''
}

function isNewTemplate(): boolean {
  return isNewSchemeTemplateRoute(routeParamId()) || route.query.newTemplate === '1'
}

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


function defaultRows(): SchemeRoundRule[] {
  return [
    { mult: 0, afterHit: 2, afterMiss: 1 },
    { mult: 1, afterHit: 2, afterMiss: 3 },
    { mult: 3, afterHit: 2, afterMiss: 1 },
  ]
}

const rows = ref<SchemeRoundRule[]>(defaultRows())

function normalizeRoundRows(raw: unknown): SchemeRoundRule[] | null {
  if (!Array.isArray(raw) || raw.length === 0) return null
  const parsed = raw
    .map((item) => {
      if (item == null || typeof item !== 'object') return null
      const row = item as Record<string, unknown>
      const mult = Number(row.mult)
      const afterHit = Number(row.afterHit)
      const afterMiss = Number(row.afterMiss)
      if (!Number.isFinite(mult) || !Number.isFinite(afterHit) || !Number.isFinite(afterMiss)) {
        return null
      }
      return { mult, afterHit, afterMiss }
    })
    .filter((r): r is SchemeRoundRule => r != null)
  return parsed.length > 0 ? parsed : null
}

async function loadDefinitionRounds(definitionId: string) {
  try {
    const def = await getSchemeDefinition(definitionId)
    const loaded = normalizeRoundRows(def.config?.rounds)
    if (loaded) rows.value = loaded
  } catch {
    /* 加载失败保留默认表单 */
  }
}

async function loadTemplateRounds(id: string) {
  if (isDraftAdvancedTemplateId(id)) {
    const tpl = getDraftAdvancedTemplate(id)
    if (tpl) {
      templateMemberOwned.value = true
      rows.value = tpl.rounds.map((r) => ({ ...r }))
      schemeTitle.value = tpl.name
    }
    return
  }
  const definitionId = templatesFetchDefinitionId()
  if (!definitionId) return
  try {
    const tpl = await getClientSchemeTemplate(definitionId, id)
    templateMemberOwned.value = Boolean(tpl.memberOwned)
    const loaded = normalizeRoundRows(tpl.config?.rounds)
    if (loaded) rows.value = loaded
    if (tpl.name) schemeTitle.value = tpl.name
  } catch {
    /* 加载失败保留默认表单 */
  }
}

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

async function onSave() {
  const bad = rows.value.some((r) => !Number.isFinite(r.mult) || r.mult > MULT_CAP)
  if (bad) {
    ElMessage.warning(`倍数须在 0～${MULT_CAP} 之间`)
    return
  }

  const name = schemeTitle.value.trim() || '新方案'
  const definitionId = memberDefinitionId()
  const tplId = templateId()

  if (definitionId) {
    try {
      await saveSchemeRounds(definitionId, rows.value)
    } catch (e) {
      const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '保存失败'
      ElMessage.error(message)
      return
    }
    ElMessage.success('已保存期次规则')
    router.back()
    return
  }

  if (isNewTemplate()) {
    if (isDraftOwner()) {
      const id = newDraftAdvancedTemplateId()
      const ok = upsertDraftAdvancedTemplate({ id, name, rounds: rows.value.map((r) => ({ ...r })) })
      if (!ok) {
        ElMessage.warning('方案草稿丢失，请返回方案编辑页后重试')
        return
      }
      refreshSchemeTemplatesState(SCHEME_DRAFT_ID)
      ElMessage.success('已创建高级倍投方案')
      router.back()
      return
    }
    const definitionId = ownerDefinitionId()
    if (!definitionId) {
      ElMessage.warning('缺少方案 ID，无法保存')
      return
    }
    try {
      await createClientSchemeTemplate({
        name,
        definitionId,
        rounds: rows.value,
      })
      refreshSchemeTemplatesState(definitionId)
    } catch (e) {
      const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '保存失败'
      ElMessage.error(message)
      return
    }
    ElMessage.success('已创建高级倍投方案')
    router.back()
    return
  }

  if (tplId) {
    if (isDraftAdvancedTemplateId(tplId)) {
      const ok = upsertDraftAdvancedTemplate({
        id: tplId,
        name,
        rounds: rows.value.map((r) => ({ ...r })),
      })
      if (!ok) {
        ElMessage.warning('方案草稿丢失，请返回方案编辑页后重试')
        return
      }
      refreshSchemeTemplatesState(SCHEME_DRAFT_ID)
      ElMessage.success('已保存期次规则')
      router.back()
      return
    }
    if (!templateMemberOwned.value) {
      ElMessage.warning('平台预置方案不可修改，请使用「新增方案」创建自己的方案')
      return
    }
    const definitionId = ownerDefinitionId()
    if (!definitionId) {
      ElMessage.warning('缺少方案 ID，无法保存')
      return
    }
    try {
      await saveClientSchemeTemplate(definitionId, tplId, {
        name,
        brief: undefined,
        rounds: rows.value,
      })
      refreshSchemeTemplatesState(definitionId)
    } catch (e) {
      const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : '保存失败'
      ElMessage.error(message)
      return
    }
    ElMessage.success('已保存期次规则')
    router.back()
    return
  }

  ElMessage.error('无法保存：方案标识无效')
}

onMounted(() => {
  void reloadRounds()
})

onActivated(() => {
  void reloadRounds()
})

function reloadRounds() {
  rows.value = defaultRows()
  templateMemberOwned.value = false
  const definitionId = memberDefinitionId()
  if (definitionId) {
    void loadDefinitionRounds(definitionId)
    return
  }
  const tplId = templateId()
  if (tplId) void loadTemplateRounds(tplId)
}
</script>

<template>
  <div class="ase">
    <header class="ase-header">
      <div class="ase-header-bar">
        <button type="button" class="ase-back" aria-label="返回" @click="goBack">
          <span class="material-sym" aria-hidden="true">arrow_back_ios_new</span>
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
    <p class="ase-hint">中后 / 挂后填写目标局数（从 1 开始，对应当前表中的「局数」列）</p>

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
              <el-input-number v-model="row.afterHit" :min="1" size="small" :controls="false" />
            </template>
          </el-table-column>
          <el-table-column label="挂后" :min-width="56" align="center" class-name="ase-cell-input">
            <template #default="{ row }">
              <el-input-number v-model="row.afterMiss" :min="1" size="small" :controls="false" />
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
  height: calc(var(--page-titlebar-height) + env(safe-area-inset-top));
  min-height: calc(var(--page-titlebar-height) + env(safe-area-inset-top));
  box-sizing: border-box;
  padding: env(safe-area-inset-top) var(--page-titlebar-pad-x) 0;
}

.ase-back {
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

.ase-back .material-sym {
  font-size: var(--page-titlebar-back-icon-size);
  color: #191c1e;
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
  padding: 0.5rem var(--page-gutter);
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
  padding: 1rem var(--page-gutter);
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
  padding: 0.75rem var(--page-gutter) max(0.75rem, env(safe-area-inset-bottom));
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
