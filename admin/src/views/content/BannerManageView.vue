<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import AdminDialog from '@/components/AdminDialog.vue'
import {
  deleteBanner,
  fetchBannerList,
  saveBanner,
  setBannerEnabled,
  type LobbyBanner,
} from '@/api/banners'
import { uploadCmsImage } from '@/api/upload'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'

const loading = ref(false)
const list = ref<LobbyBanner[]>([])
const total = ref(0)
const pageSize = ref(10)
const currentPage = ref(1)

const statusFilter = ref<'' | 'true' | 'false'>('')
const createdRange = ref<[Date, Date] | null>(null)
const appliedQuery = ref({
  enabled: '' as '' | 'true' | 'false',
  createdFrom: '',
  createdTo: '',
})

const dialogVisible = ref(false)
const editing = ref<Partial<LobbyBanner> & { imageUrl: string } | null>(null)
const uploading = ref(false)

function fmtTime(iso: string) {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso))
}

function fmtDate(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

async function reload() {
  loading.value = true
  try {
    const res = await fetchBannerList({
      page: currentPage.value,
      pageSize: pageSize.value,
      enabled: appliedQuery.value.enabled,
      createdFrom: appliedQuery.value.createdFrom || undefined,
      createdTo: appliedQuery.value.createdTo || undefined,
    })
    list.value = res.items
    total.value = res.total
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  currentPage.value = 1
  appliedQuery.value.enabled = statusFilter.value
  if (createdRange.value) {
    appliedQuery.value.createdFrom = fmtDate(createdRange.value[0])
    appliedQuery.value.createdTo = fmtDate(createdRange.value[1])
  } else {
    appliedQuery.value.createdFrom = ''
    appliedQuery.value.createdTo = ''
  }
  void reload()
}

onMounted(() => {
  void reload()
})

watch(currentPage, () => {
  void reload()
})

function openNew() {
  editing.value = { remark: '', imageUrl: '', linkUrl: '', sort: 0, enabled: true }
  dialogVisible.value = true
}

function openEdit(row: LobbyBanner) {
  editing.value = { ...row }
  dialogVisible.value = true
}

async function onPickImage(file: File) {
  uploading.value = true
  try {
    const url = await uploadCmsImage(file)
    if (editing.value) editing.value.imageUrl = url
    ElMessage.success('图片已上传')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '上传失败')
  } finally {
    uploading.value = false
  }
  return false
}

async function onSave() {
  if (!editing.value) return
  const row = editing.value
  if (!row.imageUrl?.trim()) {
    ElMessage.warning('请上传 Banner 图片')
    return
  }
  try {
    await saveBanner({
      id: row.id,
      remark: row.remark ?? '',
      imageUrl: row.imageUrl,
      linkUrl: row.linkUrl ?? '',
      sort: row.sort ?? 0,
      enabled: row.enabled ?? true,
    })
    ElMessage.success('已保存')
    dialogVisible.value = false
    void reload()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function onToggleEnabled(row: LobbyBanner) {
  try {
    await setBannerEnabled(row.id, !row.enabled)
    ElMessage.success(row.enabled ? '已禁用' : '已启用')
    void reload()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

async function onDelete(row: LobbyBanner) {
  const ok = await adminConfirmDialog({
    title: '确认删除',
    message: `删除 Banner「${row.remark || row.id}」？`,
    tone: 'warning',
  })
  if (!ok) return
  try {
    await deleteBanner(row.id)
    ElMessage.success('已删除')
    if (list.value.length === 1 && currentPage.value > 1) currentPage.value -= 1
    else void reload()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
</script>

<template>
  <div v-loading="loading">
    <h1 class="admin-page-title">Banner 管理</h1>

    <div class="toolbar filters">
      <div class="admin-filter-date">
        <el-date-picker v-model="createdRange" type="daterange" range-separator="至" start-placeholder="开始"
          end-placeholder="结束" />
      </div>
      <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px">
        <el-option label="全部" value="" />
        <el-option label="启用" value="true" />
        <el-option label="禁用" value="false" />
      </el-select>
      <el-button type="primary" @click="onSearch">查询</el-button>
      <el-button type="primary" :icon="Plus" class="toolbar-spacer" @click="openNew">新建 Banner</el-button>
    </div>

    <el-table :data="list" stripe style="width: 100%">
      <el-table-column prop="id" label="ID" min-width="120" />
      <el-table-column prop="remark" label="Banner 备注" min-width="140" show-overflow-tooltip />
      <el-table-column label="图片" min-width="120">
        <template #default="{ row }">
          <el-image :src="row.imageUrl" fit="cover" class="banner-thumb" :preview-src-list="[row.imageUrl]"
            preview-teleported />
        </template>
      </el-table-column>
      <el-table-column prop="sort" label="排序" min-width="72" />
      <el-table-column label="外链" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">
          <a v-if="row.linkUrl" :href="row.linkUrl" target="_blank" rel="noopener noreferrer" class="link-url">
            {{ row.linkUrl }}
          </a>
          <span v-else class="text-muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" min-width="80">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="160">
        <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="更新时间" min-width="160">
        <template #default="{ row }">{{ fmtTime(row.updatedAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" @click="onToggleEnabled(row)">
            {{ row.enabled ? '禁用' : '启用' }}
          </el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination v-model:current-page="currentPage" :page-size="pageSize" layout="total, prev, pager, next"
        :total="total" />
    </div>

    <AdminDialog v-model="dialogVisible" :title="editing?.id ? '编辑 Banner' : '新建 Banner'" width="560px" destroy-on-close
      @closed="editing = null">
      <template v-if="editing">
        <el-form label-width="96px">
          <el-form-item label="备注">
            <el-input v-model="editing.remark" placeholder="仅后台可见" maxlength="255" show-word-limit />
          </el-form-item>
          <el-form-item label="图片" required>
            <div class="upload-row">
              <el-upload :show-file-list="false" accept="image/jpeg,image/png,image/gif,image/webp"
                :before-upload="onPickImage" :disabled="uploading">
                <el-button :loading="uploading">上传图片</el-button>
              </el-upload>
              <el-image v-if="editing.imageUrl" :src="editing.imageUrl" fit="cover" class="banner-preview" />
            </div>
          </el-form-item>
          <el-form-item label="外链">
            <el-input v-model="editing.linkUrl" placeholder="https:// 可选，会员端点击在新窗口打开" maxlength="2048" />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="editing.sort" :min="0" :max="9999" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="editing.enabled" />
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </AdminDialog>
  </div>
</template>

<style scoped>
.admin-page-desc {
  margin: 0 0 1rem;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.toolbar-spacer {
  margin-left: auto;
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.banner-thumb {
  width: 96px;
  height: 40px;
  border-radius: 6px;
}

.banner-preview {
  width: 200px;
  height: 86px;
  border-radius: 8px;
}

.upload-row {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-url {
  color: var(--el-color-primary);
  text-decoration: none;
}

.link-url:hover {
  text-decoration: underline;
}

.text-muted {
  color: var(--el-text-color-secondary);
}
</style>
