<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import { useMaintenanceOpsStore } from '@/stores/maintenanceOps'
import { useContentStore } from '@/stores/content'

const ops = useMaintenanceOpsStore()
const content = useContentStore()
const { maintenanceOn, popupAnnouncementId, loading, saving } = storeToRefs(ops)
const { announcements } = storeToRefs(content)

onMounted(async () => {
  await Promise.all([content.hydrate(), ops.hydrate()])
})

const publishedAnn = computed(() =>
  announcements.value.filter((a) => a.status === '已发布' && a.publishedAt),
)

async function onToggle() {
  try {
    await ops.persist()
    ElMessage.success(maintenanceOn.value ? '已开启全站维护' : '已关闭全站维护')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}

async function onPopupChange() {
  try {
    await ops.persist()
    ElMessage.success('弹窗公告已更新')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}
</script>

<template>
  <div v-loading="loading">
    <h1 class="admin-page-title">系统维护</h1>
    <p class="admin-page-desc">
      §2.1 / §24-Q116：全站维护优先于彩种级；保存后会员端通过 <code>GET /public/maintenance</code> 读取状态与弹窗公告。
    </p>
    <el-card shadow="never">
      <div class="maint-form">
        <div class="maint-row">
          <span>全站维护模式</span>
          <el-switch v-model="maintenanceOn" :loading="saving" @change="onToggle" />
          <el-tag v-if="maintenanceOn" type="warning">会员端将拦截大厅</el-tag>
          <el-tag v-else type="success">关闭</el-tag>
        </div>

        <div v-if="maintenanceOn">
          <div class="maint-label">维护时弹窗公告</div>
          <p class="maint-hint">
            从已发布公告中选择一条，供会员端大厅弹层展示（公开接口会附带公告正文）。
          </p>
          <el-select
            v-model="popupAnnouncementId"
            clearable
            filterable
            :loading="saving"
            placeholder="选择公告（或清空为不指定）"
            style="width: min(100%, 420px)"
            @change="onPopupChange"
          >
            <el-option
              v-for="a in publishedAnn"
              :key="a.id"
              :label="`${a.id} · ${a.title}`"
              :value="a.id"
            />
          </el-select>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.admin-page-desc {
  margin: 0 0 1rem;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.maint-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.maint-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}
.maint-label {
  font-weight: 600;
  margin-bottom: 0.5rem;
}
.maint-hint {
  margin: 0 0 0.75rem;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
