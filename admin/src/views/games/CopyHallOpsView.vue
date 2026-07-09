<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { ElMessage } from 'element-plus'
import AdminDialog from '@/components/AdminDialog.vue'
import { useCopyHallOpsStore } from '@/stores/copyHallOps'
import type { CopyHallBoardKind } from '@shared/mock/copyHallRankings'
import type { CopyHallSchemeCandidate } from '@shared/mock/copyHallSchemeCatalog'
import { rankSlotPlayLabel } from '@/utils/copyHallPlayLabel'

const store = useCopyHallOpsStore()
onMounted(() => {
  void store.hydrate().catch((e: unknown) => {
    ElMessage.error(e instanceof Error ? e.message : '加载榜单失败')
  })
})

const {
  activeBoard,
  currentSlots,
  usedSchemeIds,
  pickerSchemes,
  pickerSearchField,
  pickerKeyword,
  poolLoading,
} = storeToRefs(store)

const boardLabels: Record<CopyHallBoardKind, string> = {
  master: '大神榜',
  contrary: '反买榜',
}

const pickerVisible = ref(false)
const pickingRank = ref<number | null>(null)
const pickerSelected = ref<CopyHallSchemeCandidate | null>(null)
const pickerSaving = ref(false)

function openPicker(rank: number) {
  pickingRank.value = rank
  pickerSelected.value = null
  store.resetPickerSearch()
  pickerVisible.value = true
  void store.searchPool()
}

const pickerKeywordPlaceholder = computed(() =>
  pickerSearchField.value === 'snapshotId' ? '请输入快照 ID' : '请输入方案名称',
)

function onPickerSearch() {
  store.searchPicker()
}

function pickerRowClassName({ row }: { row: CopyHallSchemeCandidate }) {
  return canPickScheme(row) ? '' : 'copy-hall-picker-row--disabled'
}

function canPickScheme(row: CopyHallSchemeCandidate): boolean {
  if (pickingRank.value == null || !row.schemeId) return false
  const occupied = currentSlots.value.find((s) => s.schemeId === row.schemeId)
  return !occupied || occupied.rank === pickingRank.value
}

function onPickerRowClick(row: CopyHallSchemeCandidate) {
  if (!canPickScheme(row)) {
    const occupied = currentSlots.value.find((s) => s.schemeId === row.schemeId)
    ElMessage.warning(`该方案已在第 ${occupied?.rank ?? '—'} 名，不可重复上榜`)
    return
  }
  pickerSelected.value = row
}

async function onConfirmPickScheme() {
  if (pickingRank.value == null) return
  const row = pickerSelected.value
  if (!row) {
    ElMessage.warning('请先选择要更换的方案')
    return
  }
  if (!canPickScheme(row)) {
    const occupied = currentSlots.value.find((s) => s.schemeId === row.schemeId)
    ElMessage.warning(`该方案已在第 ${occupied?.rank ?? '—'} 名，不可重复上榜`)
    return
  }
  const rank = pickingRank.value
  pickerSaving.value = true
  try {
    await store.assignScheme(rank, row)
    pickerVisible.value = false
    pickingRank.value = null
    pickerSelected.value = null
    ElMessage.success(`已保存：「${row.schemeName}」设为第 ${rank} 名`)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    pickerSaving.value = false
  }
}

function findInCatalog(schemeId: string) {
  return store.resolveScheme(schemeId)
}

function playLabelForSlot(row: { schemeId?: string; playMethod?: string; playTypeId?: string; subPlayId?: string }) {
  return rankSlotPlayLabel(
    {
      playMethod: row.playMethod ?? '',
      playTypeId: row.playTypeId,
      subPlayId: row.subPlayId,
    },
    findInCatalog(row.schemeId ?? ''),
  )
}

async function onMoveSlot(rank: number, direction: 'up' | 'down') {
  try {
    await store.moveSlot(rank, direction)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  }
}
</script>

<template>
  <div>
    <h1 class="admin-page-title">跟单大厅运营</h1>
    <p style="margin: 0 0 1rem; font-size: 13px; color: var(--el-text-color-secondary)">
      分享池方案与<strong>全站方案监控 · 分享池</strong>同源；全站共用大神榜 / 反买榜 Top 10，保存后同步至会员端。
    </p>

    <div
      style="
        display: flex;
        flex-wrap: nowrap;
        gap: 0.75rem;
        margin-bottom: 1rem;
        align-items: center;
      "
    >
      <el-radio-group v-model="activeBoard" style="flex-shrink: 0">
        <el-radio-button value="master">大神榜 Top 10</el-radio-button>
        <el-radio-button value="contrary">反买榜 Top 10</el-radio-button>
      </el-radio-group>
    </div>

    <h2 style="margin: 0 0 0.75rem; font-size: 15px; font-weight: 600">当前榜单</h2>

    <el-table
      :key="activeBoard"
      :data="currentSlots"
      stripe
      style="width: 100%; margin-bottom: 1.5rem"
    >
      <template #empty>
        <span style="font-size: 13px; color: var(--el-text-color-secondary)">
          暂无榜单数据
        </span>
      </template>

      <el-table-column prop="rank" label="名次" min-width="64" align="center" />

      <el-table-column prop="schemeName" label="方案名称" min-width="120">
        <template #default="{ row }">
          <span v-if="row.schemeId">{{ row.schemeName || '—' }}</span>
          <el-tag v-else type="info" size="small">待配置</el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="schemeId" label="快照ID" min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ row.schemeId || '—' }}</template>
      </el-table-column>

      <el-table-column label="彩种" min-width="108">
        <template #default="{ row }">
          {{
            row.lotteryLabel ||
              findInCatalog(row.schemeId)?.lotteryLabel ||
              row.lotteryCode ||
              '—'
          }}
        </template>
      </el-table-column>

      <el-table-column label="玩法" min-width="120">
        <template #default="{ row }">{{ playLabelForSlot(row) }}</template>
      </el-table-column>

      <el-table-column label="发布者" min-width="96">
        <template #default="{ row }">
          {{ findInCatalog(row.schemeId)?.publisherName ?? '—' }}
        </template>
      </el-table-column>

      <el-table-column label="类型" min-width="88">
        <template #default="{ row }">
          {{ findInCatalog(row.schemeId)?.kind ?? '—' }}
        </template>
      </el-table-column>

      <el-table-column label="投注次数" min-width="88" align="center">
        <template #default="{ row }">
          {{ findInCatalog(row.schemeId)?.betCount ?? '—' }}
        </template>
      </el-table-column>

      <el-table-column label="胜率" min-width="80" align="center">
        <template #default="{ row }">
          <template v-if="findInCatalog(row.schemeId)?.winRate != null">
            {{ findInCatalog(row.schemeId)?.winRate }}%
          </template>
          <span v-else>—</span>
        </template>
      </el-table-column>

      <el-table-column label="操作" min-width="180" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openPicker(row.rank)">更换方案</el-button>
          <el-button
            link
            type="primary"
            :disabled="row.rank <= 1"
            @click="onMoveSlot(row.rank, 'up')"
          >
            上移
          </el-button>
          <el-button
            link
            type="primary"
            :disabled="row.rank >= 10"
            @click="onMoveSlot(row.rank, 'down')"
          >
            下移
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <AdminDialog
      v-model="pickerVisible"
      :title="`选择方案 · 第 ${pickingRank ?? '—'} 名`"
      width="min(100%, 720px)"
      destroy-on-close
    >
      <p style="margin: 0 0 0.75rem; font-size: 12px; color: var(--el-text-color-secondary)">
        数据来源：全站方案监控 · 分享池（全彩种）
      </p>

      <div
        style="
          display: flex;
          flex-wrap: wrap;
          gap: 0.75rem;
          margin-bottom: 0.75rem;
          align-items: center;
        "
      >
        <el-select v-model="pickerSearchField" style="width: 128px">
          <el-option label="方案名称" value="schemeName" />
          <el-option label="快照 ID" value="snapshotId" />
        </el-select>

        <el-input
          v-model="pickerKeyword"
          clearable
          :placeholder="pickerKeywordPlaceholder"
          style="flex: 1; min-width: 160px"
          @keyup.enter="onPickerSearch"
        />

        <el-button type="primary" @click="onPickerSearch">查询</el-button>
      </div>

      <el-table
        v-loading="poolLoading"
        :data="pickerSchemes"
        stripe
        style="width: 100%"
        max-height="360"
        highlight-current-row
        :row-class-name="pickerRowClassName"
        @row-click="onPickerRowClick"
      >
        <el-table-column prop="schemeId" label="快照ID" min-width="108" show-overflow-tooltip />
        <el-table-column prop="schemeName" label="方案名称" min-width="140" />
        <el-table-column prop="lotteryLabel" label="彩种" min-width="108" />
        <el-table-column label="玩法" min-width="120">
          <template #default="{ row }">{{ playLabelForSlot(row) }}</template>
        </el-table-column>
        <el-table-column prop="kind" label="类型" min-width="72" />
        <el-table-column label="状态" min-width="88">
          <template #default="{ row }">
            <el-tag v-if="usedSchemeIds.has(row.schemeId)" type="warning" size="small">
              已在榜
            </el-tag>
            <span v-else style="color: var(--el-text-color-secondary); font-size: 12px">可选</span>
          </template>
        </el-table-column>
      </el-table>

      <p style="margin: 0.75rem 0 0; font-size: 12px; color: var(--el-text-color-secondary)">
        点击行选中方案，再点「确认更换」保存至第 {{ pickingRank }} 名。
      </p>

      <template #footer>
        <el-button @click="pickerVisible = false">取消</el-button>
        <el-button type="primary" :loading="pickerSaving" :disabled="!pickerSelected" @click="onConfirmPickScheme">
          确认更换
        </el-button>
      </template>
    </AdminDialog>

    <p style="margin: 1rem 0 0; font-size: 12px; color: var(--el-text-color-secondary)">
      当前：{{ boardLabels[activeBoard] }} · 全站共用榜单 · {{ currentSlots.length }} 条。通过「更换方案」从分享池挑选上榜。
    </p>
  </div>
</template>

<style scoped>
:deep(.copy-hall-picker-row--disabled) {
  cursor: not-allowed;
  color: var(--el-text-color-secondary);
}
</style>
