<script setup lang="ts">

import { onMounted, ref } from 'vue'

import { storeToRefs } from 'pinia'

import { ElMessage } from 'element-plus'

import AdminDialog from '@/components/AdminDialog.vue'

import { adminConfirmDialog } from '@/utils/adminConfirmDialog'

import { useSchemeTemplateLibraryStore } from '@/stores/schemeTemplateLibrary'

import type { SchemeTemplateRow } from '@shared/mock/schemeTemplateLibrary'

import type { SchemeRoundRule } from '@shared/schemeRoundRules'

import {

  SCHEME_ROUND_MULT_CAP,

  defaultSchemeRoundRules,

  schemeRoundRulesFromConfig,

  validateSchemeRoundRules,

} from '@shared/schemeRoundRules'



const tplStore = useSchemeTemplateLibraryStore()
const { templates, total, page, pageSize, loading } = storeToRefs(tplStore)

pageSize.value = 10

async function reloadList() {
  await tplStore.loadList({
    page: page.value,
    pageSize: pageSize.value,
    name: appliedName.value,
  })
}

const keyword = ref('')
const appliedName = ref('')

function onSearch() {
  page.value = 1
  appliedName.value = keyword.value.trim()
  void reloadList()
}

onMounted(() => {
  void reloadList()
})

const dialogVisible = ref(false)
const editingId = ref<string | null>(null)

const draftName = ref('')

const draftBrief = ref('')

const draftSortOrder = ref(10)

const draftEnabled = ref(true)

const draftRounds = ref<SchemeRoundRule[]>(defaultSchemeRoundRules())

const saving = ref(false)

function roundCount(row: SchemeTemplateRow): number {

  return schemeRoundRulesFromConfig(row.config).length

}



function openCreate() {

  editingId.value = null

  draftName.value = ''

  draftBrief.value = ''

  draftRounds.value = defaultSchemeRoundRules()
  draftSortOrder.value = (total.value + 1) * 10

  draftEnabled.value = true

  dialogVisible.value = true

}



async function openEdit(row: SchemeTemplateRow) {
  try {
    const latest = await tplStore.fetchTemplate(row.id)
    editingId.value = latest.id
    draftName.value = latest.name
    draftBrief.value = latest.brief ?? ''
    draftSortOrder.value = latest.sortOrder
    draftEnabled.value = latest.enabled
    draftRounds.value = schemeRoundRulesFromConfig(latest.config).map((r) => ({ ...r }))
    dialogVisible.value = true
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载模板失败')
  }
}



function addRound() {

  draftRounds.value.push({ mult: 0, afterHit: 1, afterMiss: 1 })

}



function removeRound(index: number) {

  draftRounds.value.splice(index, 1)

}



async function submitTemplate() {

  const name = draftName.value.trim()

  if (!name) {

    ElMessage.warning('请填写方案名称')

    return

  }

  const roundErr = validateSchemeRoundRules(draftRounds.value)

  if (roundErr) {

    ElMessage.warning(roundErr)

    return

  }

  saving.value = true

  try {

    const payload = {

      name,

      brief: draftBrief.value,

      sortOrder: draftSortOrder.value,

      enabled: draftEnabled.value,

      rounds: draftRounds.value,

    }

    if (editingId.value) {
      await tplStore.updateTemplate(editingId.value, payload)
      ElMessage.success('模板已更新，客户端高级倍投列表将同步')
    } else {

      await tplStore.createTemplate(payload)

      ElMessage.success('模板已创建，客户端高级倍投列表将同步')

    }

    dialogVisible.value = false

  } catch (e) {

    ElMessage.error(e instanceof Error ? e.message : '保存失败')

  } finally {

    saving.value = false

  }

}



async function onRemove(row: SchemeTemplateRow) {

  const ok = await adminConfirmDialog({

    title: '删除模板',

    message: `确认删除模板「${row.name}」？`,

    tone: 'warning',

  })

  if (!ok) return

  try {

    await tplStore.removeTemplate(row.id)

    ElMessage.success('已删除，客户端列表将同步移除')

  } catch (e) {

    ElMessage.error(e instanceof Error ? e.message : '删除失败')

  }

}



async function toggleEnabled(row: SchemeTemplateRow, enabled: boolean) {

  try {

    await tplStore.updateTemplate(row.id, { enabled })

  } catch (e) {

    ElMessage.error(e instanceof Error ? e.message : '更新失败')

  }

}

</script>



<template>

  <div>

    <h1 class="admin-page-title">方案模板库</h1>

    <div class="toolbar">
      <el-input v-model="keyword" placeholder="方案名称" clearable style="width: 220px" @keyup.enter="onSearch" />
      <el-button type="primary" @click="onSearch">查询</el-button>
      <div class="toolbar-spacer" />
      <el-button type="primary" @click="openCreate">创建模板</el-button>
    </div>



    <el-table v-loading="loading" :data="templates" stripe style="width: 100%">

      <el-table-column prop="sortOrder" label="排序" min-width="72" />

      <el-table-column prop="name" label="方案名称" min-width="160" />

      <el-table-column label="局数" min-width="64" align="center">

        <template #default="{ row }">{{ roundCount(row) }}</template>

      </el-table-column>

      <el-table-column prop="brief" label="说明" min-width="140" show-overflow-tooltip />

      <el-table-column label="启用" min-width="88">

        <template #default="{ row }">

          <el-switch :model-value="row.enabled" @change="(v: boolean) => toggleEnabled(row, v)" />

        </template>

      </el-table-column>

      <el-table-column label="操作" min-width="120" fixed="right">

        <template #default="{ row }">

          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>

          <el-button link type="danger" @click="onRemove(row)">删除</el-button>

        </template>

      </el-table-column>

    </el-table>

    <div class="pager">
      <el-pagination v-model:current-page="page" :page-size="pageSize" layout="total, prev, pager, next" :total="total"
        @current-change="reloadList" />
    </div>

    <AdminDialog v-model="dialogVisible" :title="editingId ? '编辑高级倍投方案' : '创建高级倍投方案'" width="min(100%, 640px)"
      destroy-on-close>

      <el-form label-position="top">

        <el-form-item label="方案名称" required>

          <el-input v-model="draftName" placeholder="如：三期推波方案" maxlength="48" />

        </el-form-item>

        <el-form-item label="排序">

          <el-input-number v-model="draftSortOrder" :min="0" :max="9999" />

        </el-form-item>

        <el-form-item label="说明">

          <el-input v-model="draftBrief" type="textarea" :rows="2" placeholder="可选，运营备注" />

        </el-form-item>

        <el-form-item label="启用">

          <el-switch v-model="draftEnabled" />

        </el-form-item>

        <el-form-item label="局次规则" required>

          <p style="margin: 0 0 0.5rem; font-size: 12px; color: var(--el-text-color-secondary)">

            倍数上限 {{ SCHEME_ROUND_MULT_CAP }} 倍；中后 / 挂后为目标局数（从 1 开始）。

          </p>

          <div style="display: flex; justify-content: flex-end; margin-bottom: 0.5rem">

            <el-button type="primary" plain size="small" @click="addRound">新增局数</el-button>

          </div>

          <el-table :data="draftRounds" size="small" stripe empty-text="暂无局数" style="width: 100%">

            <el-table-column label="局数" min-width="48" align="center">

              <template #default="{ $index }">{{ $index + 1 }}</template>

            </el-table-column>

            <el-table-column label="倍数" min-width="88" align="center">

              <template #default="{ row }">

                <el-input-number v-model="row.mult" :min="0" :max="SCHEME_ROUND_MULT_CAP" size="small"
                  :controls="false" />

              </template>

            </el-table-column>

            <el-table-column label="中后" min-width="88" align="center">

              <template #default="{ row }">

                <el-input-number v-model="row.afterHit" :min="1" size="small" :controls="false" />

              </template>

            </el-table-column>

            <el-table-column label="挂后" min-width="88" align="center">

              <template #default="{ row }">

                <el-input-number v-model="row.afterMiss" :min="1" size="small" :controls="false" />

              </template>

            </el-table-column>

            <el-table-column label="删除" min-width="56" align="center">

              <template #default="{ $index }">

                <el-button link type="danger" size="small" :disabled="draftRounds.length <= 1"
                  @click="removeRound($index)">

                  删除

                </el-button>

              </template>

            </el-table-column>

          </el-table>

        </el-form-item>

      </el-form>

      <template #footer>

        <el-button @click="dialogVisible = false">取消</el-button>

        <el-button type="primary" :loading="saving" @click="submitTemplate">保存</el-button>

      </template>

    </AdminDialog>

  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
  align-items: center;
}

.toolbar-spacer {
  flex: 1;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
</style>
