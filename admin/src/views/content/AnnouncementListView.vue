<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import type { Announcement } from '@/types/content'
import { ElMessage } from 'element-plus'
import { useContentStore } from '@/stores/content'
import AdminDialog from '@/components/AdminDialog.vue'
import RichHtmlField from '@/components/RichHtmlField.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'

const content = useContentStore()
const { announcements } = storeToRefs(content)

onMounted(() => {
  void content.hydrate()
})

const pageSize = ref(10)
const currentPage = ref(1)

const pagedRows = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return announcements.value.slice(start, start + pageSize.value)
})

const dialogVisible = ref(false)
const editing = ref<Announcement | null>(null)

function fmtTime(iso: string | null) {
  if (!iso) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso))
}

function openNew() {
  editing.value = content.newAnnouncement()
  dialogVisible.value = true
}

function openEdit(row: Announcement) {
  editing.value = { ...row }
  dialogVisible.value = true
}

async function onSave() {
  if (!editing.value) return
  const row = editing.value
  if (!row.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  if (row.status === '已发布' && !row.publishedAt) {
    row.publishedAt = new Date().toISOString()
  }
  if (row.status === '草稿') {
    row.publishedAt = null
  }
  try {
    await content.upsertAnnouncement({ ...row })
    ElMessage.success('已保存')
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function onDelete(row: Announcement) {
  const ok = await adminConfirmDialog({
    title: '确认',
    message: `删除公告「${row.title}」？`,
    tone: 'warning',
  })
  if (!ok) return
  try {
    await content.removeAnnouncement(row.id)
    ElMessage.success('已删除')
    if (announcements.value.length <= (currentPage.value - 1) * pageSize.value && currentPage.value > 1) {
      currentPage.value -= 1
    }
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function onTogglePin(row: Announcement) {
  if (!row.pinned && row.status !== '已发布') {
    ElMessage.warning('仅已发布公告可置顶')
    return
  }
  const next = !row.pinned
  if (next) {
    const ok = await adminConfirmDialog({
      title: '确认置顶',
      message: `置顶后该公告将展示在会员端 Banner 下方，并替换当前置顶公告。`,
      tone: 'primary',
    })
    if (!ok) return
  }
  try {
    await content.pinAnnouncement(row.id, next)
    ElMessage.success(next ? '已置顶' : '已取消置顶')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}
</script>

<template>
  <div>
    <h1 class="admin-page-title">公告管理</h1>
    <p class="admin-page-desc">维护平台公告标题、发布状态与正文；置顶公告展示在会员端 Banner 下方，同时仅允许一条置顶。</p>

    <div class="toolbar">
      <el-button type="primary" @click="openNew">新建公告</el-button>
    </div>

    <el-table :data="pagedRows" stripe style="width: 100%">
      <el-table-column prop="id" label="编号" min-width="120" />
      <el-table-column prop="title" label="标题" min-width="180" />
      <el-table-column label="状态" min-width="88">
        <template #default="{ row }">
          <el-tag :type="row.status === '已发布' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="置顶" min-width="72">
        <template #default="{ row }">
          <el-tag v-if="row.pinned" type="warning" size="small">置顶</el-tag>
          <span v-else class="text-muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="发布时间" min-width="160">
        <template #default="{ row }">{{ fmtTime(row.publishedAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button
            link
            :type="row.pinned ? 'warning' : 'primary'"
            :disabled="!row.pinned && row.status !== '已发布'"
            @click="onTogglePin(row)"
          >
            {{ row.pinned ? '取消置顶' : '置顶' }}
          </el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, prev, pager, next"
        :total="announcements.length"
      />
    </div>

    <AdminDialog v-model="dialogVisible" title="编辑公告" width="720px" destroy-on-close @closed="editing = null">
      <template v-if="editing">
        <el-form label-width="88px">
          <el-form-item label="标题">
            <el-input v-model="editing.title" maxlength="120" show-word-limit />
          </el-form-item>
          <el-form-item label="状态">
            <el-radio-group v-model="editing.status">
              <el-radio-button label="草稿" />
              <el-radio-button label="已发布" />
            </el-radio-group>
          </el-form-item>
          <el-form-item label="正文">
            <RichHtmlField v-model="editing.bodyHtml" />
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
  margin-bottom: 12px;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}
.text-muted {
  color: var(--el-text-color-secondary);
}
</style>
