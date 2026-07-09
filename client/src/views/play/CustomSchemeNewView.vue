<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog } from '@/utils/confirmDialog'
import OptionPickerModal from '@/components/ui/OptionPickerModal.vue'
import type { OptionPickerItem } from '@/components/ui/OptionPickerModal.vue'
import { checkSchemeNameAvailable, deleteSchemeDefinition } from '@/api/schemes/definitions'
import { clearSchemeDraft } from '@/utils/schemeDraftStorage'
import { fetchLotterySchemeOptions } from '@/api/schemes/schemeOptions'
import { fetchPlayTree } from '@/api/games/lotteries'
import { usePublicLotteries } from '@/composables/usePublicLotteries'
import { defaultPlaySelection, formatSubPlayLabel } from '@/utils/playConfig'
import {
  filterPlayTypesForRunType,
  filterSubPlaysForRunType,
  lotteryHasAdvTriggerPlay,
  syncRunTypePlaySelection,
  validateRunTypePlaySelection,
} from '@/utils/runTypeMatrix'
import type { PlayTypeNode } from '@/types/playCatalog'
import { ApiError } from '@/api/client'

const router = useRouter()
const submitting = ref(false)
const { lotteries, load: loadLotteries } = usePublicLotteries()

type PickerKind = 'lottery' | 'runType' | 'playType' | 'subPlay'

const schemeName = ref('')
const lotteryId = ref('')
const runTypeId = ref('fixed_rotate')
const playTypeId = ref('')
const subPlayId = ref('')

const lotteryOptions = computed<OptionPickerItem[]>(() =>
  lotteries.value.map((l) => ({ label: l.displayName, value: l.code })),
)

/** 运行类型定版 7 种（与后端 internal/schemes/run_types.go、lottery_scheme_option_sets 种子同源） */
const runTypeOptions = ref<OptionPickerItem[]>([
  { label: '定码轮换', value: 'fixed_rotate' },
  { label: '高级定码轮换', value: 'adv_fixed_rotate' },
  { label: '高级开某投某', value: 'adv_trigger_bet' },
  { label: '冷热温出号', value: 'hot_cold_warm' },
  { label: '随机出号', value: 'random_draw' },
  { label: '内置计画', value: 'builtin_plan' },
  { label: '固定号码', value: 'fixed_number' },
])

/** 后端已废弃的运行类型枚举，接口若仍返回则前端过滤不展示 */
const DEPRECATED_RUN_TYPES = new Set(['batch_fixed', 'dynamic_chase', 'plan_follow'])

const playTypeOptions = ref<OptionPickerItem[]>([])
const subPlayOptions = ref<OptionPickerItem[]>([])
const playTreeTypes = ref<PlayTypeNode[]>([])

const isBuiltinPlan = computed(() => runTypeId.value === 'builtin_plan')

/** 可选运行类型：过滤废弃值；当前彩种不支持高级开某投某时隐藏 */
const availableRunTypeOptions = computed<OptionPickerItem[]>(() => {
  let opts = runTypeOptions.value.filter((o) => !DEPRECATED_RUN_TYPES.has(String(o.value)))
  if (playTreeTypes.value.length > 0 && !lotteryHasAdvTriggerPlay(playTreeTypes.value)) {
    opts = opts.filter((o) => String(o.value) !== 'adv_trigger_bet')
  }
  return opts
})

const filteredPlayTypeOptions = computed<OptionPickerItem[]>(() =>
  filterPlayTypesForRunType(runTypeId.value, playTypeOptions.value, playTreeTypes.value),
)

const filteredSubPlayOptions = computed<OptionPickerItem[]>(() =>
  filterSubPlaysForRunType(
    runTypeId.value,
    subPlayOptions.value,
    playTypeId.value,
    playTreeTypes.value,
  ),
)

function applyRunTypePlaySync() {
  if (isBuiltinPlan.value || !playTreeTypes.value.length) return
  const synced = syncRunTypePlaySelection({
    runTypeId: runTypeId.value,
    playTypeId: playTypeId.value,
    subPlayId: subPlayId.value,
    playTreeTypes: playTreeTypes.value,
    playTypeOptions: playTypeOptions.value,
    subPlayOptions: subPlayOptions.value,
  })
  runTypeId.value = synced.runTypeId
  playTypeId.value = synced.playTypeId
  subPlayId.value = synced.subPlayId
}

function ensureSelectedInOptions(
  options: OptionPickerItem[],
  selected: { value: string },
  fallback: string,
) {
  if (options.length === 0) return
  if (!options.some((o) => String(o.value) === selected.value)) {
    selected.value = String(options[0]?.value ?? fallback)
  }
}

async function loadRunTypeOptions(code: string) {
  try {
    const data = await fetchLotterySchemeOptions(code)
    const fresh = data.runTypes.filter((o) => !DEPRECATED_RUN_TYPES.has(String(o.value)))
    if (fresh.length) {
      runTypeOptions.value = fresh
    }
  } catch {
    /* 保留默认 7 种运行类型 */
  }
}

async function loadPlayTreeOptions(code: string) {
  if (!code) return
  try {
    const tree = await fetchPlayTree(code)
    playTreeTypes.value = tree.playTypes
    playTypeOptions.value = tree.playTypes.map((t) => ({
      label: t.label,
      value: t.typeId,
    }))
    const routeType = playTypeId.value
    const hasType = tree.playTypes.some((t) => t.typeId === routeType)
    if (!hasType) {
      const def = defaultPlaySelection(tree)
      playTypeId.value = def.typeId
      subPlayId.value = def.subId
    }
    const typeNode = tree.playTypes.find((t) => t.typeId === playTypeId.value)
    subPlayOptions.value = (typeNode?.subPlays ?? []).map((s) => ({
      label: formatSubPlayLabel(s.label),
      value: s.subId,
    }))
    ensureSelectedInOptions(subPlayOptions.value, subPlayId, subPlayId.value)
    applyRunTypePlaySync()
  } catch (e) {
    playTreeTypes.value = []
    playTypeOptions.value = []
    subPlayOptions.value = []
    ElMessage.error(e instanceof ApiError ? e.message : '加载玩法树失败')
  }
}

watch(playTypeId, (typeId) => {
  if (!lotteryId.value || !typeId) return
  void fetchPlayTree(lotteryId.value).then((tree) => {
    const typeNode = tree.playTypes.find((t) => t.typeId === typeId)
    subPlayOptions.value = (typeNode?.subPlays ?? []).map((s) => ({
      label: formatSubPlayLabel(s.label),
      value: s.subId,
    }))
    applyRunTypePlaySync()
  })
})

watch(runTypeId, () => {
  applyRunTypePlaySync()
})

watch(lotteryId, (code) => {
  if (!code) return
  void loadRunTypeOptions(code)
  void loadPlayTreeOptions(code)
})

watch(availableRunTypeOptions, (opts) => {
  if (!opts.length) return
  if (!opts.some((o) => String(o.value) === runTypeId.value)) {
    runTypeId.value = 'fixed_rotate'
  }
})

onMounted(async () => {
  await loadLotteries()
  if (!lotteryId.value && lotteries.value.length) {
    lotteryId.value = lotteries.value[0].code
  }
})

const pickerOpen = ref(false)
const pickerKind = ref<PickerKind | null>(null)

const pickerTitle = computed(() => {
  switch (pickerKind.value) {
    case 'lottery':
      return '选择彩种'
    case 'runType':
      return '运行类型'
    case 'playType':
      return '玩法类型'
    case 'subPlay':
      return '子玩法'
    default:
      return ''
  }
})

const pickerOptions = computed<OptionPickerItem[]>(() => {
  switch (pickerKind.value) {
    case 'lottery':
      return lotteryOptions.value
    case 'runType':
      return availableRunTypeOptions.value
    case 'playType':
      return filteredPlayTypeOptions.value
    case 'subPlay':
      return filteredSubPlayOptions.value
    default:
      return []
  }
})

const pickerSelectedValue = computed(() => {
  switch (pickerKind.value) {
    case 'lottery':
      return lotteryId.value
    case 'runType':
      return runTypeId.value
    case 'playType':
      return playTypeId.value
    case 'subPlay':
      return subPlayId.value
    default:
      return ''
  }
})

function openPicker(k: PickerKind) {
  pickerKind.value = k
  pickerOpen.value = true
}

function onPickerConfirm(val: string | number) {
  const v = String(val)
  const k = pickerKind.value
  if (k === 'lottery') lotteryId.value = v
  else if (k === 'runType') runTypeId.value = v
  else if (k === 'playType') playTypeId.value = v
  else if (k === 'subPlay') subPlayId.value = v
  pickerKind.value = null
  if (k === 'lottery' || k === 'runType' || k === 'playType') {
    applyRunTypePlaySync()
  }
}

function onPickerCancel() {
  pickerKind.value = null
}

function labelOf(list: OptionPickerItem[] | readonly OptionPickerItem[], id: string) {
  return list.find((o) => String(o.value) === id)?.label ?? ''
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push({ name: 'lobby' })
}

function onSearchName() {
  ElMessage.info('方案检索将在对接接口后开放')
}

async function onNext(): Promise<void> {
  const title = schemeName.value.trim()
  if (!title) {
    await confirmDialog({
      title: '提示',
      message: '方案名称不能为空',
      tone: 'warning',
      confirmText: '我知道了',
      showCancel: false,
    })
    return
  }
  // 内置计画不选彩种与玩法，仅需方案名称
  if (!isBuiltinPlan.value && (!lotteryId.value || !playTypeId.value || !subPlayId.value)) {
    ElMessage.warning('请选择彩种与玩法')
    return
  }
  if (!isBuiltinPlan.value) {
    const matrixErr = validateRunTypePlaySelection(
      runTypeId.value,
      playTypeId.value,
      subPlayId.value,
      playTreeTypes.value,
    )
    if (matrixErr) {
      ElMessage.warning(matrixErr)
      return
    }
  }
  submitting.value = true
  try {
    const check = await checkSchemeNameAvailable(title)
    if (!check.available) {
      if (check.existingDefinitionId && !check.existingHasInstance) {
        const resume = await confirmDialog({
          title: '名称已占用',
          message: `方案「${title}」已存在但未添加至云端。继续编辑该方案，或删除旧草稿后重新新建？`,
          tone: 'warning',
          confirmText: '继续编辑',
          cancelText: '删除重建',
        })
        if (resume) {
          await router.push({
            name: 'advanced-scheme-edit',
            params: { schemeId: check.existingDefinitionId },
            query: { kind: 'custom' },
          })
          return
        }
        try {
          await deleteSchemeDefinition(check.existingDefinitionId)
        } catch (e) {
          ElMessage.error(e instanceof ApiError ? e.message : '删除旧方案失败')
          return
        }
      } else {
        ElMessage.error('方案名称已存在，请更换名称')
        return
      }
    }
    clearSchemeDraft()
    await router.push({
      name: 'advanced-scheme-edit',
      params: { schemeId: 'new' },
      query: {
        draft: '1',
        kind: 'custom',
        title: encodeURIComponent(title),
        lottery: isBuiltinPlan.value ? '' : lotteryId.value,
        runType: runTypeId.value,
        playType: isBuiltinPlan.value ? '' : playTypeId.value,
        subPlay: isBuiltinPlan.value ? '' : subPlayId.value,
      },
    })
  } catch (e) {
    ElMessage.error(e instanceof ApiError ? e.message : '校验方案名称失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="csn">
    <header class="csn-header">
      <button type="button" class="csn-back" aria-label="返回" @click="goBack">
        <svg class="csn-back-ico" viewBox="0 0 24 24" width="22" height="22" aria-hidden="true">
          <path fill="currentColor" d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z" />
        </svg>
      </button>
      <h1 class="csn-title">新增方案</h1>
    </header>

    <main class="csn-main">
      <div class="csn-card">
        <div class="csn-field">
          <label class="csn-lbl" for="csn-scheme-name">方案名称</label>
          <div class="csn-name-row">
            <el-input
              id="csn-scheme-name"
              v-model="schemeName"
              size="large"
              class="csn-inp"
              placeholder="输入方案名称..."
              clearable
              @keyup.enter="onSearchName"
            />
            <button type="button" class="csn-search-btn" aria-label="检索方案" @click="onSearchName">
              <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
                <path
                  fill="currentColor"
                  d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"
                />
              </svg>
            </button>
          </div>
        </div>

        <div v-if="!isBuiltinPlan" class="csn-field">
          <span class="csn-lbl" id="csn-lbl-lottery">彩种</span>
          <button
            type="button"
            class="csn-picker"
            aria-haspopup="dialog"
            :aria-expanded="pickerOpen && pickerKind === 'lottery'"
            aria-labelledby="csn-lbl-lottery csn-val-lottery"
            @click="openPicker('lottery')"
          >
            <span id="csn-val-lottery" class="csn-picker-val">{{
              labelOf(lotteryOptions, lotteryId)
            }}</span>
            <span class="csn-picker-ico" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="18" height="18" class="csn-gear">
                <path
                  fill="currentColor"
                  d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58a.5.5 0 00.12-.64l-1.92-3.32a.488.488 0 00-.6-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 00-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.488.488 0 00-.6.22L2.74 8.87c-.12.21-.08.47.12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58a.5.5 0 00-.12.64l1.92 3.32c.12.22.37.29.6.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.23.09.48.02.6-.22l1.92-3.32c.12-.22.07-.47-.12-.64l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
                />
              </svg>
            </span>
          </button>
        </div>

        <div class="csn-field">
          <span class="csn-lbl" id="csn-lbl-run">运行类型</span>
          <button
            type="button"
            class="csn-picker"
            aria-haspopup="dialog"
            :aria-expanded="pickerOpen && pickerKind === 'runType'"
            aria-labelledby="csn-lbl-run csn-val-run"
            @click="openPicker('runType')"
          >
            <span id="csn-val-run" class="csn-picker-val">{{ labelOf(availableRunTypeOptions, runTypeId) }}</span>
            <span class="csn-picker-ico" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="18" height="18" class="csn-gear">
                <path
                  fill="currentColor"
                  d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58a.5.5 0 00.12-.64l-1.92-3.32a.488.488 0 00-.6-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 00-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.488.488 0 00-.6.22L2.74 8.87c-.12.21-.08.47.12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58a.5.5 0 00-.12.64l1.92 3.32c.12.22.37.29.6.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.23.09.48.02.6-.22l1.92-3.32c.12-.22.07-.47-.12-.64l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
                />
              </svg>
            </span>
          </button>
        </div>

        <p v-if="isBuiltinPlan" class="csn-hint">
          内置计画无需选择彩种与玩法，创建后在方案内容中选择已收藏的跟单大厅方案
        </p>

        <div v-if="!isBuiltinPlan" class="csn-field">
          <span class="csn-lbl" id="csn-lbl-play">玩法类型</span>
          <button
            type="button"
            class="csn-picker"
            aria-haspopup="dialog"
            :aria-expanded="pickerOpen && pickerKind === 'playType'"
            aria-labelledby="csn-lbl-play csn-val-play"
            @click="openPicker('playType')"
          >
            <span id="csn-val-play" class="csn-picker-val">{{ labelOf(filteredPlayTypeOptions, playTypeId) }}</span>
            <span class="csn-picker-ico" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="18" height="18" class="csn-gear">
                <path
                  fill="currentColor"
                  d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58a.5.5 0 00.12-.64l-1.92-3.32a.488.488 0 00-.6-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 00-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.488.488 0 00-.6.22L2.74 8.87c-.12.21-.08.47.12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58a.5.5 0 00-.12.64l1.92 3.32c.12.22.37.29.6.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.23.09.48.02.6-.22l1.92-3.32c.12-.22.07-.47-.12-.64l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
                />
              </svg>
            </span>
          </button>
        </div>

        <div v-if="!isBuiltinPlan" class="csn-field">
          <span class="csn-lbl" id="csn-lbl-sub">子玩法</span>
          <button
            type="button"
            class="csn-picker"
            aria-haspopup="dialog"
            :aria-expanded="pickerOpen && pickerKind === 'subPlay'"
            aria-labelledby="csn-lbl-sub csn-val-sub"
            @click="openPicker('subPlay')"
          >
            <span id="csn-val-sub" class="csn-picker-val">{{ labelOf(filteredSubPlayOptions, subPlayId) }}</span>
            <span class="csn-picker-ico" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="18" height="18" class="csn-gear">
                <path
                  fill="currentColor"
                  d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58a.5.5 0 00.12-.64l-1.92-3.32a.488.488 0 00-.6-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 00-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.488.488 0 00-.6.22L2.74 8.87c-.12.21-.08.47.12.64l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58a.5.5 0 00-.12.64l1.92 3.32c.12.22.37.29.6.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.23.09.48.02.6-.22l1.92-3.32c.12-.22.07-.47-.12-.64l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
                />
              </svg>
            </span>
          </button>
        </div>
      </div>

      <el-button type="primary" class="csn-next-main" size="large" :loading="submitting" @click="onNext">
        <span class="csn-next-main-txt">下一步</span>
        <span class="csn-next-main-ico" aria-hidden="true">&gt;</span>
      </el-button>
    </main>

    <OptionPickerModal
      v-model="pickerOpen"
      :selected-value="pickerSelectedValue"
      :title="pickerTitle"
      :options="pickerOptions"
      selection-accent="primary"
      @confirm="onPickerConfirm"
      @cancel="onPickerCancel"
    />
  </div>
</template>

<style scoped>
.csn {
  --csn-surface: #f7f9fb;
  --csn-primary: #0066ff;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--csn-surface);
  color: #191c1e;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: env(safe-area-inset-bottom);
}

.csn-header {
  flex-shrink: 0;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 0.5rem;
  padding: max(0.75rem, env(safe-area-inset-top)) 0.75rem 0.875rem;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  box-shadow: 0 8px 32px rgba(25, 28, 30, 0.06);
}

.csn-back {
  justify-self: start;
  width: 2.25rem;
  height: 2.25rem;
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.csn-back:focus-visible {
  outline: 2px solid var(--csn-primary);
  outline-offset: 2px;
}

.csn-back-ico {
  display: block;
}

.csn-title {
  margin: 0;
  justify-self: center;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #0f172a;
  text-align: center;
}

.csn-main {
  flex: 1;
  padding: 1rem 1rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 28rem;
  margin: 0 auto;
  width: 100%;
}

.csn-card {
  background: #fff;
  border-radius: 1rem;
  padding: 1.25rem 1rem;
  box-shadow: 0 12px 40px rgba(25, 28, 30, 0.06);
}

.csn-field {
  margin-bottom: 1.125rem;
}

.csn-field:last-child {
  margin-bottom: 0;
}

.csn-lbl {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: #475569;
  margin-bottom: 0.5rem;
}

.csn-hint {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.csn-name-row {
  display: flex;
  align-items: stretch;
  gap: 0.5rem;
}

.csn-inp {
  flex: 1;
  min-width: 0;
}

.csn-inp :deep(.el-input__wrapper) {
  border-radius: 0.625rem;
  background: #f1f5f9;
  box-shadow: none;
  padding-left: 0.875rem;
}

.csn-inp :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--csn-primary) inset;
}

.csn-search-btn {
  flex-shrink: 0;
  width: 2.75rem;
  border: none;
  border-radius: 0.625rem;
  background: rgba(0, 102, 255, 0.12);
  color: #0050cb;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}

.csn-search-btn:hover {
  background: rgba(0, 102, 255, 0.18);
}

.csn-picker {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 2.75rem;
  padding: 0.5rem 0.65rem 0.5rem 0.875rem;
  border: none;
  border-radius: 0.625rem;
  background: #f1f5f9;
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition: box-shadow 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.csn-picker:hover {
  background: #eceef0;
}

.csn-picker:focus-visible {
  outline: 2px solid var(--csn-primary);
  outline-offset: 2px;
}

.csn-picker-val {
  flex: 1;
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 600;
  color: #0f172a;
}

.csn-picker-ico {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  color: #94a3b8;
}

.csn-gear {
  display: block;
  opacity: 0.85;
}

.csn-next-main {
  width: 100%;
  margin: 0;
  height: 3rem;
  border-radius: 0.75rem;
  font-weight: 700;
  font-size: 1rem;
  border: none;
  box-shadow: 0 8px 24px rgba(0, 102, 255, 0.22);
}

.csn-next-main-txt {
  margin-right: 0.35rem;
}

.csn-next-main-ico {
  font-size: 1.1rem;
  line-height: 1;
}
</style>
