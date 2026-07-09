<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import type { FaqArticle } from '@/types/content'
import { ElMessage } from 'element-plus'
import { useContentStore } from '@/stores/content'
import AdminDialog from '@/components/AdminDialog.vue'
import RichHtmlField from '@/components/RichHtmlField.vue'
import { adminConfirmDialog } from '@/utils/adminConfirmDialog'

const content = useContentStore()
const { faqArticles } = storeToRefs(content)

onMounted(() => {
  void content.hydrate()
})

const pageSize = ref(10)
const currentPage = ref(1)

const pagedRows = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return [...faqArticles.value].sort((a, b) => a.sort - b.sort).slice(start, start + pageSize.value)
})

const dialogVisible = ref(false)
const editing = ref<FaqArticle | null>(null)

function openNew() {
  editing.value = content.newFaqArticle()
  dialogVisible.value = true
}

function openEdit(row: FaqArticle) {
  editing.value = { ...row }
  dialogVisible.value = true
}

async function onSave() {
  if (!editing.value?.title.trim()) {
    ElMessage.warning('请输入问题标题')
    return
  }
  try {
    await content.upsertFaqArticle({ ...editing.value })
    ElMessage.success('已保存')
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function onDelete(row: FaqArticle) {
  const ok = await adminConfirmDialog({
    title: '确认',
    message: `删除「${row.title}」？`,
    tone: 'warning',
  })
  if (!ok) return
  try {
    await content.removeFaqArticle(row.id)
    ElMessage.success('已删除')
    if (faqArticles.value.length <= (currentPage.value - 1) * pageSize.value && currentPage.value > 1) {
      currentPage.value -= 1
    }
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}
</script>

<template>
  <div>
    <h1 class="admin-page-title">常见问题</h1>
    <p class="admin-page-desc">维护问题标题与正文内容。</p>

    <div class="toolbar">
      <el-button type="primary" @click="openNew">新建问题</el-button>
    </div>

    <el-table :data="pagedRows" stripe style="width: 100%">
      <el-table-column prop="title" label="问题标题" min-width="240" />
      <el-table-column label="操作" min-width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="pageSize"
        layout="total, prev, pager, next"
        :total="faqArticles.length"
      />
    </div>

    <AdminDialog v-model="dialogVisible" title="编辑问题" width="720px" destroy-on-close @closed="editing = null">
      <template v-if="editing">
        <el-form label-width="80px">
          <el-form-item label="问题标题">
            <el-input v-model="editing.title" maxlength="200" show-word-limit />
          </el-form-item>
          <el-form-item label="内容">
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
</style>
