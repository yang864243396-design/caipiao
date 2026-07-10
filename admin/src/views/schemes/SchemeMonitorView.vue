<script setup lang="ts">

import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

import { useRouter } from 'vue-router'

import { storeToRefs } from 'pinia'

import { ElMessage } from 'element-plus'

import { adminConfirmDialog } from '@/utils/adminConfirmDialog'

import SchemeHistoryDrawer from '@/components/schemes/SchemeHistoryDrawer.vue'

import ShareSnapshotCreateDialog from '@/components/schemes/ShareSnapshotCreateDialog.vue'

import { useSchemeInstancesStore } from '@/stores/schemeInstances'

import { useLotteryCatalogStore } from '@/stores/lotteryCatalog'

import type { SchemeMonitorQuery, SchemeMonitorSearchField, SchemeShareQuery, SchemeShareSearchField } from '@/api/schemes'

import type { SchemeInstanceRow, SchemeShareSnapshotRow } from '@/stores/schemeInstances'

import { startSchemeMonitorSync } from '@/composables/useAdminQueueSync'

import { useAdminPlayTypeLabelCache } from '@/composables/useAdminPlayTypeLabelCache'



const router = useRouter()

const store = useSchemeInstancesStore()

const catalog = useLotteryCatalogStore()

const { list, shareSnapshots, loading, shareLoading } = storeToRefs(store)

const { rows: lotteryRows } = storeToRefs(catalog)

const playTypeLabelCache = useAdminPlayTypeLabelCache()

let stopSync: (() => void) | null = null

onMounted(() => {
  void catalog.hydrate()
  void store.hydrate()
  void onSearch()
  stopSync = startSchemeMonitorSync(() => {
    void store.reload()
  })
})

onUnmounted(() => {
  stopSync?.()
})

function lotteryPlayTemplate(lotteryCode: string) {
  return lotteryRows.value.find((r) => r.code === lotteryCode)?.playTemplate?.trim() ?? ''
}

watch(
  [list, shareSnapshots, lotteryRows],
  ([userRows, shareRows]) => {
    const templates = [
      ...userRows.map((r) => lotteryPlayTemplate(r.lotteryCode)),
      ...shareRows.map((r) => lotteryPlayTemplate(r.lotteryCode)),
    ].filter(Boolean)
    void playTypeLabelCache.preloadTemplates(templates)
  },
  { immediate: true },
)



const monitorTab = ref<'user' | 'share'>('user')



const searchField = ref<SchemeMonitorSearchField>('account')

const keyword = ref('')

const kind = ref<string>('')

const status = ref<string>('')

const simBetFilter = ref<'' | 'formal' | 'sim'>('')

const lotteryCode = ref('')

/** 点击「查询」后生效的条件 */
const appliedQuery = ref<SchemeMonitorQuery>({
  searchField: 'account',
  keyword: '',
  kind: '',
  status: '',
  lotteryCode: '',
})

const shareSearchField = ref<SchemeShareSearchField>('schemeName')

const shareKeyword = ref('')

const shareLotteryCode = ref('')

/** 分享池 Tab：点击「查询」后生效的条件 */
const appliedShareQuery = ref<SchemeShareQuery>({
  searchField: 'schemeName',
  keyword: '',
  lotteryCode: '',
})

const lotteryOptions = computed(() =>
  [...lotteryRows.value].sort((a, b) => a.sortOrder - b.sortOrder),
)

const keywordPlaceholder = computed(() =>
  searchField.value === 'schemeName' ? '请输入方案名' : '请输入会员账号',
)

const shareKeywordPlaceholder = computed(() =>
  shareSearchField.value === 'snapshotId' ? '请输入快照 ID' : '请输入方案名称',
)

async function reloadUserList() {
  await store.loadUserList(appliedQuery.value)
}

async function reloadShareList() {
  await store.loadShareList(appliedShareQuery.value)
}

function onSearch() {
  currentPage.value = 1
  let simBet: boolean | undefined
  if (simBetFilter.value === 'formal') simBet = false
  else if (simBetFilter.value === 'sim') simBet = true
  appliedQuery.value = {
    searchField: searchField.value,
    keyword: keyword.value.trim(),
    kind: kind.value,
    status: status.value,
    lotteryCode: lotteryCode.value,
    simBet,
  }
  void reloadUserList()
}

function onShareSearch() {
  sharePage.value = 1
  appliedShareQuery.value = {
    searchField: shareSearchField.value,
    keyword: shareKeyword.value.trim(),
    lotteryCode: shareLotteryCode.value,
  }
  void reloadShareList()
}



const pageSize = ref(10)

const currentPage = ref(1)

const sharePage = ref(1)



const pagedRows = computed(() => {

  const start = (currentPage.value - 1) * pageSize.value

  return list.value.slice(start, start + pageSize.value)

})



const pagedShareRows = computed(() => {

  const start = (sharePage.value - 1) * pageSize.value

  return shareSnapshots.value.slice(start, start + pageSize.value)

})



const drawerVisible = ref(false)

const shareCreateVisible = ref(false)
const shareEditSnapshot = ref<SchemeShareSnapshotRow | null>(null)

function openShareCreateDialog() {
  shareEditSnapshot.value = null
  shareCreateVisible.value = true
}

function onEditShareSnapshot(row: SchemeShareSnapshotRow) {
  shareEditSnapshot.value = row
  shareCreateVisible.value = true
}

const selectedScheme = ref<SchemeInstanceRow | null>(null)



function openHistory(row: SchemeInstanceRow) {

  selectedScheme.value = row

  drawerVisible.value = true

}



function fmt(iso: string) {

  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(

    new Date(iso),

  )

}

function rowPlayTypeLabel(row: { lotteryCode: string; playTypeLabel?: string; settings: { playTypeId: string } }) {
  const fromApi = row.playTypeLabel?.trim()
  if (fromApi) return fromApi
  return playTypeLabelCache.resolvePlayTypeLabelForRow(
    lotteryPlayTemplate(row.lotteryCode),
    row.settings.playTypeId,
  )
}



function goMember(id: string) {

  router.push({ name: 'member-detail', params: { id } })

}



function stubExport() {

  ElMessage.info('导出为占位：正式上线后接异步导出')

}



async function onSoftStop(row: { id: string; status: string }) {

  if (row.status !== '运行中') {

    ElMessage.warning('仅「运行中」实例可强停')

    return

  }

  const ok = await adminConfirmDialog({

    title: '强停方案',

    message: `确认对实例 ${row.id} 执行强制软停？`,

    tone: 'warning',

    confirmText: '强停',

  })

  if (!ok) return

  try {
    const ok = await store.softStop(row.id)
    ElMessage[ok ? 'success' : 'error'](ok ? '已封停，仪表盘运行中方案数已更新' : '操作失败')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }

}



async function onReleaseStop(row: { id: string; status: string }) {

  if (row.status !== '已封停') {

    ElMessage.warning('仅「已封停」实例可解封')

    return

  }

  const ok = await adminConfirmDialog({

    title: '解封',

    message: `解封后实例变为「已暂停」，会员须手动恢复。确认解封 ${row.id}？`,

    tone: 'warning',

    confirmText: '解封',

  })

  if (!ok) return

  try {
    const ok = await store.releaseStop(row.id)
    ElMessage[ok ? 'success' : 'error'](ok ? '已解封 → 已暂停' : '操作失败')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }

}



async function onDeleteShareSnapshot(row: SchemeShareSnapshotRow) {

  const ok = await adminConfirmDialog({

    title: '删除分享池快照',

    message: `删除快照 ${row.id}？方案下载将立即不再展示。`,

    tone: 'warning',

    confirmText: '删除',

  })

  if (!ok) return

  const deleted = await store.deleteShareSnapshot(row.id)

  ElMessage[deleted ? 'success' : 'error'](deleted ? '快照已删除' : '操作失败')

}

</script>



<template>

  <div>

    <h1 class="admin-page-title">全站方案监控</h1>


    <el-tabs v-model="monitorTab" style="margin-bottom: 1rem">

      <el-tab-pane label="用户方案" name="user" />

      <el-tab-pane label="分享池" name="share" />

    </el-tabs>



    <template v-if="monitorTab === 'user'">

      <div style="

          display: flex;

          flex-wrap: wrap;

          gap: 0.75rem;

          margin-bottom: 1rem;

          align-items: center;

          justify-content: space-between;

        ">

        <div style="display: flex; flex-wrap: wrap; gap: 0.75rem; align-items: center; flex: 1; min-width: 0">

          <el-select v-model="searchField" style="width: 128px">

            <el-option label="会员账号" value="account" />

            <el-option label="方案名" value="schemeName" />

          </el-select>

          <el-input v-model="keyword" clearable :placeholder="keywordPlaceholder" style="width: min(100%, 240px)"
            @keyup.enter="onSearch" />

          <el-select v-model="lotteryCode" clearable filterable placeholder="彩种" style="width: 160px">

            <el-option v-for="lot in lotteryOptions" :key="lot.code" :label="lot.displayName" :value="lot.code">

              <span :class="{ 'monitor-lottery-maint': lot.saleStatus !== 'on_sale' }">

                {{ lot.displayName }}

              </span>

            </el-option>

          </el-select>

          <el-select v-model="kind" clearable placeholder="类型" style="width: 120px">

            <el-option label="自创" value="自创" />

            <el-option label="反买" value="反买" />

            <el-option label="跟单" value="跟单" />

          </el-select>

          <el-select v-model="status" clearable placeholder="状态" style="width: 120px">

            <el-option label="待开启" value="待开启" />

            <el-option label="运行中" value="运行中" />

            <el-option label="已暂停" value="已暂停" />

            <el-option label="已封停" value="已封停" />

          </el-select>

          <el-select v-model="simBetFilter" clearable placeholder="投注通道" style="width: 120px">

            <el-option label="正式" value="formal" />

            <el-option label="模拟" value="sim" />

          </el-select>

          <el-button type="primary" @click="onSearch">查询</el-button>

        </div>

        <el-button type="primary" plain @click="stubExport">导出</el-button>

      </div>



      <el-table v-loading="loading" :data="pagedRows" stripe style="width: 100%">

        <el-table-column prop="id" label="实例ID" min-width="100" />

        <el-table-column prop="memberName" label="会员账号" min-width="120">

          <template #default="{ row }">

            <el-button link type="primary" @click="goMember(row.memberId)">{{ row.memberName }}</el-button>

          </template>

        </el-table-column>

        <el-table-column label="方案名称" min-width="140" show-overflow-tooltip>

          <template #default="{ row }">{{ row.settings.schemeName }}</template>

        </el-table-column>

        <el-table-column prop="kind" label="类型" min-width="72" />

        <el-table-column label="运行类型" min-width="110">

          <template #default="{ row }">{{ row.runTypeLabel || '—' }}</template>

        </el-table-column>

        <el-table-column label="投注通道" min-width="88">

          <template #default="{ row }">{{ row.simBet ? '模拟' : '正式' }}</template>

        </el-table-column>

        <el-table-column prop="lotteryLabel" label="彩种" min-width="120" />

        <el-table-column label="玩法类型" min-width="100" show-overflow-tooltip>
          <template #default="{ row }">{{ rowPlayTypeLabel(row) }}</template>
        </el-table-column>

        <el-table-column prop="status" label="状态" min-width="120" show-overflow-tooltip />

        <el-table-column label="创建" min-width="140">

          <template #default="{ row }">{{ fmt(row.createdAt) }}</template>

        </el-table-column>

        <el-table-column label="更新" min-width="140">

          <template #default="{ row }">{{ fmt(row.updatedAt) }}</template>

        </el-table-column>

        <el-table-column label="操作" min-width="220" fixed="right">

          <template #default="{ row }">

            <el-button link type="primary" @click="openHistory(row)">投注与盈亏</el-button>

            <el-button link type="primary" :disabled="row.status !== '运行中'" @click="onSoftStop(row)">

              强停

            </el-button>

            <el-button link type="primary" :disabled="row.status !== '已封停'" @click="onReleaseStop(row)">

              解封

            </el-button>

          </template>

        </el-table-column>

      </el-table>



      <div style="display: flex; justify-content: flex-end; margin-top: 1rem">

        <el-pagination v-model:current-page="currentPage" :page-size="pageSize" layout="total, prev, pager, next"
          :total="list.length" />

      </div>

    </template>



    <template v-else>

      <div style="
          display: flex;
          flex-wrap: nowrap;
          gap: 0.75rem;
          margin-bottom: 1rem;
          align-items: center;
        ">
        <el-select v-model="shareSearchField" style="width: 128px; flex-shrink: 0">
          <el-option label="方案名称" value="schemeName" />
          <el-option label="快照 ID" value="snapshotId" />
        </el-select>

        <el-input v-model="shareKeyword" clearable :placeholder="shareKeywordPlaceholder"
          style="width: 240px; flex-shrink: 0" @keyup.enter="onShareSearch" />

        <el-select v-model="shareLotteryCode" clearable filterable placeholder="彩种"
          style="width: 160px; flex-shrink: 0">
          <el-option v-for="lot in lotteryOptions" :key="lot.code" :label="lot.displayName" :value="lot.code">
            <span :class="{ 'monitor-lottery-maint': lot.saleStatus !== 'on_sale' }">
              {{ lot.displayName }}
            </span>
          </el-option>
        </el-select>

        <el-button type="primary" style="flex-shrink: 0" @click="onShareSearch">查询</el-button>

        <el-button type="primary" style="flex-shrink: 0" @click="openShareCreateDialog">新建方案</el-button>
      </div>



      <el-table v-loading="shareLoading" :data="pagedShareRows" stripe style="width: 100%">

        <el-table-column prop="id" label="快照ID" min-width="100" />

        <el-table-column prop="kind" label="类型" min-width="72" />

        <el-table-column label="方案名称" min-width="140" show-overflow-tooltip>

          <template #default="{ row }">{{ row.settings.schemeName }}</template>

        </el-table-column>

        <el-table-column prop="lotteryLabel" label="彩种" min-width="120" />

        <el-table-column label="玩法类型" min-width="100" show-overflow-tooltip>
          <template #default="{ row }">{{ rowPlayTypeLabel(row) }}</template>
        </el-table-column>

        <el-table-column label="发布" min-width="140">

          <template #default="{ row }">{{ fmt(row.publishedAt) }}</template>

        </el-table-column>

        <el-table-column label="更新" min-width="140">

          <template #default="{ row }">{{ fmt(row.updatedAt) }}</template>

        </el-table-column>

        <el-table-column label="操作" min-width="160" fixed="right">

          <template #default="{ row }">

            <el-button link type="primary" @click="onEditShareSnapshot(row)">编辑</el-button>

            <el-button link type="danger" @click="onDeleteShareSnapshot(row)">删除</el-button>

          </template>

        </el-table-column>

      </el-table>



      <div style="display: flex; justify-content: flex-end; margin-top: 1rem">

        <el-pagination v-model:current-page="sharePage" :page-size="pageSize" layout="total, prev, pager, next"
          :total="shareSnapshots.length" />

      </div>

    </template>



    <SchemeHistoryDrawer v-model="drawerVisible" :scheme="selectedScheme" />

    <ShareSnapshotCreateDialog v-model="shareCreateVisible" :edit-snapshot="shareEditSnapshot" />

  </div>

</template>

<style scoped>
.monitor-lottery-maint {
  color: var(--el-color-danger);
}
</style>
