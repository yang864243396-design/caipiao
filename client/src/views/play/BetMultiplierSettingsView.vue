<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const subTabLabels = ['小白倍投', '一键倍投', '简单倍投', '高级倍投'] as const
/** Tab 值与 el-radio-button :label 同型（字符串），避免与原生 radio 的值类型不一致导致选中态 class 不匹配 */
type SubTabId = `${0 | 1 | 2 | 3}`
/** 从玩法详情进入时默认「简单倍投」 */
const activeSubTab = ref<SubTabId>('2')

const tabFromRoute = route.query.activeTab
const tabParsed = tabFromRoute == null || tabFromRoute === ''
  ? ''
  : String(Array.isArray(tabFromRoute) ? tabFromRoute[0] : tabFromRoute)
if (tabParsed === '0' || tabParsed === '1' || tabParsed === '2' || tabParsed === '3') {
  activeSubTab.value = tabParsed as SubTabId
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push({ name: 'play-detail' })
}

function validateBetMultiplier(): string | null {
  if (activeSubTab.value === '0') {
    if (!newbiePrincipal.value.trim()) return '请填写总本金'
    if (newbieProfitType.value === 'rate' && !newbieRateVal.value.trim()) return '请填写收益利率'
    if (newbieProfitType.value === 'fixed' && !newbieFixedVal.value.trim()) return '请填写固定利润'
    if (newbieProfitType.value === 'accum') {
      if (!newbieAccumStart.value.trim()) return '请填写累加起步'
      if (!newbieAccumStep.value.trim()) return '请填写累进步长'
    }
  }
  if (activeSubTab.value === '1') {
    if (!oneclickCycle.value.trim()) return '请填写计划周期'
    if (!oneclickProfit.value.trim()) return '请填写收益利润'
  }
  if (activeSubTab.value === '2' && !simpleMultiples.value.trim()) return '请填写倍数序列'
  if (activeSubTab.value === '3' && !selectedAdvancedId.value) return '请选择高级倍投方案'
  return null
}

/** 从方案配置进入倍投设定时，校验失败将报错文案带回方案页右侧展示 */
function navigateBackToSchemeWithError(msg: string) {
  const schemeId = route.query.schemeId
  if (schemeId == null || String(schemeId) === '') {
    ElMessage.error(msg)
    return
  }
  const q: Record<string, string> = {
    bmsError: encodeURIComponent(msg),
  }
  if (route.query.title != null && String(route.query.title) !== '') {
    q.title = String(route.query.title)
  }
  if (route.query.lottery != null && String(route.query.lottery) !== '') {
    q.lottery = String(route.query.lottery)
  }
  void router.push({
    name: 'advanced-scheme-edit',
    params: { schemeId: String(schemeId) },
    query: q,
  })
}

function navigateBackToSchemeWithKind() {
  const schemeId = route.query.schemeId
  if (schemeId == null || String(schemeId) === '') return
  const q: Record<string, string> = {
    bmsKind: String(activeSubTab.value),
  }
  if (route.query.title != null && String(route.query.title) !== '') {
    q.title = String(route.query.title)
  }
  if (route.query.lottery != null && String(route.query.lottery) !== '') {
    q.lottery = String(route.query.lottery)
  }
  void router.push({
    name: 'advanced-scheme-edit',
    params: { schemeId: String(schemeId) },
    query: q,
  })
}

/**
 * 倍投设定本期不在前端做持久化：每次从入口（玩法详情 / 各方案配置）进入都是独立、
 * 全新的编辑会话。点击「确认」时校验通过后提示并返回；后续接入接口后，该数据会随
 * 当前方案一并保存到云端。
 */
function onConfirm() {
  const err = validateBetMultiplier()
  if (err) {
    if (route.query.fromScheme === '1') navigateBackToSchemeWithError(err)
    else ElMessage.error(err)
    return
  }
  ElMessage.success('已保存倍投方式')
  if (route.query.fromScheme === '1' && route.query.schemeId != null && String(route.query.schemeId) !== '') {
    navigateBackToSchemeWithKind()
  } else {
    router.back()
  }
}

function onCancel() {
  router.back()
}

// —— 小白倍投 ——
const newbiePrincipal = ref('')
const newbieMode = ref('元')
const newbieProfitType = ref<'rate' | 'fixed' | 'accum'>('rate')
const newbieRateVal = ref('')
const newbieFixedVal = ref('')
const newbieAccumStart = ref('')
const newbieAccumStep = ref('')

/** 小白倍投：四个「一键生成」方案互斥选中 */
type NewbieGeneratePreset = 'line' | 'followStop' | 'suspend1' | 'suspend2'
const newbieGeneratePreset = ref<NewbieGeneratePreset>('line')

watch(newbieProfitType, (mode) => {
  if (mode === 'rate') {
    newbieFixedVal.value = ''
    newbieAccumStart.value = ''
    newbieAccumStep.value = ''
  } else if (mode === 'fixed') {
    newbieRateVal.value = ''
    newbieAccumStart.value = ''
    newbieAccumStep.value = ''
  } else {
    newbieRateVal.value = ''
    newbieFixedVal.value = ''
  }
})

// —— 一键倍投 ——
const oneclickCycle = ref('')
const oneclickProfit = ref('')
/** 一键倍投底部两个一键生成按钮（与小白倍投相同样式与交互形态） */
type OneclickGeneratePreset = 'line' | 'wave'
const oneclickGeneratePreset = ref<OneclickGeneratePreset>('line')

// —— 简单倍投 ——
const simpleMultiples = ref('1,2,4')

// —— 高级倍投 ——
interface AdvancedScheme {
  id: string
  title: string
}
const advancedSchemes = ref<AdvancedScheme[]>([
  { id: '1', title: '两期中跟挂停' },
  { id: '2', title: '三期推波方案' },
  { id: '3', title: '四期倍投计划' },
  { id: '4', title: '六期倍投方案' },
])
const selectedAdvancedId = ref<string | null>(null)

function addAdvancedScheme() {
  const n = advancedSchemes.value.length + 1
  advancedSchemes.value.push({
    id: String(Date.now()),
    title: `新方案 ${n}`,
  })
}

function openAdvancedSchemeEditor(row: AdvancedScheme) {
  router.push({
    name: 'advanced-scheme-edit',
    params: { schemeId: row.id },
    query: { title: encodeURIComponent(row.title) },
  })
}

function removeAdvancedScheme(row: AdvancedScheme) {
  advancedSchemes.value = advancedSchemes.value.filter((item) => item.id !== row.id)
  if (selectedAdvancedId.value === row.id) selectedAdvancedId.value = null
  ElMessage.success('已删除方案')
}

/** 倍投计划表列：仅用 `minWidth`，与全局 el-table 无横滚约定一致（见 DESIGN.md §8） */
interface PlanTableColumn {
  prop: string
  label: string
  minWidth: number
  /** 利润率列单行省略（其它列为多行换行） */
  overflowEllipsis?: boolean
}

const tableColumns: PlanTableColumn[] = [
  { prop: 'period', label: '期数', minWidth: 34 },
  { prop: 'mult', label: '倍数', minWidth: 40 },
  { prop: 'curBet', label: '本期投入', minWidth: 48 },
  { prop: 'totalBet', label: '总投入', minWidth: 42 },
  { prop: 'prize', label: '奖金', minWidth: 40 },
  { prop: 'profit', label: '利润', minWidth: 40 },
  { prop: 'margin', label: '利润率%', minWidth: 48, overflowEllipsis: true },
]

const planTableEmpty = computed(() => [] as Record<string, string>[])

const showPlanTable = computed(() => activeSubTab.value === '0' || activeSubTab.value === '1')
</script>

<template>
  <div class="bms">
    <header class="bms-header">
      <div class="bms-header-top">
        <button type="button" class="bms-back" aria-label="返回" @click="goBack">
          <svg class="bms-back-ico" viewBox="0 0 24 24" width="22" height="22" aria-hidden="true">
            <path fill="currentColor" d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z" />
          </svg>
        </button>
        <h1 class="bms-title">倍投设定</h1>
        <div class="bms-header-right">
          <el-button v-if="activeSubTab === '3'" type="primary" class="bms-add-scheme" plain size="small"
            @click="addAdvancedScheme">
            新增方案
          </el-button>
        </div>
      </div>
      <div class="bms-tabs-row">
        <el-radio-group v-model="activeSubTab" class="detail-tab-rg" size="small">
          <el-radio-button v-for="(lbl, i) in subTabLabels" :key="lbl" :value="String(i)">{{ lbl }}</el-radio-button>
        </el-radio-group>
      </div>
    </header>

    <main class="bms-main">
      <!-- 小白倍投 -->
      <template v-if="activeSubTab === '0'">
        <div class="bms-card">
          <div class="bms-field-row">
            <span class="bms-lbl">总本金</span>
            <el-input v-model="newbiePrincipal" size="small" class="bms-inp-short" />
            <span class="bms-unit">元</span>
            <span class="bms-lbl bms-lbl--push">投注模式</span>
            <el-select v-model="newbieMode" size="small" class="bms-select-mode">
              <el-option label="元" value="元" />
              <el-option label="角" value="角" />
            </el-select>
          </div>

          <div class="bms-radio-block">
            <label class="bms-radio-row">
              <input v-model="newbieProfitType" type="radio" value="rate" class="bms-native-radio" />
              <span class="bms-radio-lbl">收益利率</span>
              <el-input v-model="newbieRateVal" size="small" class="bms-inp-grow"
                :disabled="newbieProfitType !== 'rate'" />
            </label>
            <label class="bms-radio-row">
              <input v-model="newbieProfitType" type="radio" value="fixed" class="bms-native-radio" />
              <span class="bms-radio-lbl">固定利润</span>
              <el-input v-model="newbieFixedVal" size="small" class="bms-inp-grow"
                :disabled="newbieProfitType !== 'fixed'" />
            </label>
            <div class="bms-radio-row bms-radio-row--accum">
              <label class="bms-accum-label">
                <input v-model="newbieProfitType" type="radio" value="accum" class="bms-native-radio" />
                <span class="bms-radio-lbl">累加利润：起步</span>
              </label>
              <el-input v-model="newbieAccumStart" size="small" class="bms-inp-tiny"
                :disabled="newbieProfitType !== 'accum'" />
              <span class="bms-radio-lbl">累进</span>
              <el-input v-model="newbieAccumStep" size="small" class="bms-inp-tiny"
                :disabled="newbieProfitType !== 'accum'" />
            </div>
          </div>

          <div class="bms-action-grid">
            <el-button type="warning" :plain="newbieGeneratePreset !== 'line'"
              :class="['bms-btn-generate', { 'bms-btn-generate--solid': newbieGeneratePreset === 'line' }]"
              @click="newbieGeneratePreset = 'line'">
              一键生成<br>直线倍投
            </el-button>
            <el-button type="warning" :plain="newbieGeneratePreset !== 'followStop'"
              :class="['bms-btn-generate', { 'bms-btn-generate--solid': newbieGeneratePreset === 'followStop' }]"
              @click="newbieGeneratePreset = 'followStop'">
              一键生成<br>中跟挂停
            </el-button>
            <el-button type="warning" :plain="newbieGeneratePreset !== 'suspend1'"
              :class="['bms-btn-generate', { 'bms-btn-generate--solid': newbieGeneratePreset === 'suspend1' }]"
              @click="newbieGeneratePreset = 'suspend1'">
              一键生成<br>挂停1期
            </el-button>
            <el-button type="warning" :plain="newbieGeneratePreset !== 'suspend2'"
              :class="['bms-btn-generate', { 'bms-btn-generate--solid': newbieGeneratePreset === 'suspend2' }]"
              @click="newbieGeneratePreset = 'suspend2'">
              一键生成<br>挂停2期
            </el-button>
          </div>
        </div>
        <p class="bms-hint bms-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>
      </template>

      <!-- 一键倍投 -->
      <template v-else-if="activeSubTab === '1'">
        <div class="bms-card">
          <div class="bms-field-row bms-field-row--oneclick">
            <span class="bms-lbl">计划周期</span>
            <div class="bms-oneclick-grow">
              <el-input v-model="oneclickCycle" size="small" />
            </div>
          </div>
          <div class="bms-field-row bms-field-row--oneclick">
            <span class="bms-lbl">收益利润</span>
            <div class="bms-inp-suffix-wrap bms-oneclick-grow">
              <el-input v-model="oneclickProfit" size="small" />
              <span class="bms-suffix">%</span>
            </div>
          </div>
          <div class="bms-action-grid">
            <el-button
              type="warning"
              :plain="oneclickGeneratePreset !== 'line'"
              :class="['bms-btn-generate', { 'bms-btn-generate--solid': oneclickGeneratePreset === 'line' }]"
              @click="oneclickGeneratePreset = 'line'"
            >
              一键生成(直线)<br>倍投计划
            </el-button>
            <el-button
              type="warning"
              :plain="oneclickGeneratePreset !== 'wave'"
              :class="['bms-btn-generate', { 'bms-btn-generate--solid': oneclickGeneratePreset === 'wave' }]"
              @click="oneclickGeneratePreset = 'wave'"
            >
              一键生成(推波)<br>倍投计划
            </el-button>
          </div>
        </div>
        <p class="bms-hint bms-hint--primary">* 默认以「元」模式一键生成计划倍数</p>
        <p class="bms-hint bms-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>
      </template>

      <!-- 简单倍投（默认） -->
      <template v-else-if="activeSubTab === '2'">
        <div class="bms-card bms-card--simple">
          <el-input v-model="simpleMultiples" type="textarea" :rows="3" size="small" resize="none"
            class="bms-textarea" />
        </div>
        <p class="bms-hint bms-hint--danger">* 倍数计算上限为 200000 倍为止，超出不计</p>
      </template>

      <!-- 高级倍投 -->
      <template v-else>
        <div class="bms-advanced-list">
          <div class="bms-advanced-head">
            <span>方案</span>
            <span>操作</span>
          </div>
          <div v-for="(row, idx) in advancedSchemes" :key="row.id" class="bms-advanced-row"
            :class="{ 'bms-advanced-row--alt': idx % 2 === 1 }">
            <label class="bms-advanced-left">
              <input v-model="selectedAdvancedId" type="radio" class="bms-native-radio" :value="row.id" @click.stop />
              <span
                class="bms-advanced-title bms-advanced-title--link"
                role="button"
                tabindex="0"
                @click.prevent.stop="openAdvancedSchemeEditor(row)"
                @keyup.enter.prevent="openAdvancedSchemeEditor(row)"
              >{{ row.title }}</span>
            </label>
            <div class="bms-advanced-ops">
              <button
                type="button"
                class="bms-icon-btn bms-icon-btn--edit"
                aria-label="编辑"
                @click.stop="openAdvancedSchemeEditor(row)"
              />
              <button
                type="button"
                class="bms-icon-btn bms-icon-btn--del"
                aria-label="删除"
                @click.stop="removeAdvancedScheme(row)"
              />
            </div>
          </div>
        </div>
      </template>

      <div v-if="showPlanTable" class="table-card" aria-label="倍投计划表">
        <el-table :data="planTableEmpty" class="detail-bet-table" size="small" stripe empty-text="暂无数据"
          :style="{ width: '100%' }">
          <el-table-column v-for="col in tableColumns" :key="col.prop" :prop="col.prop" :label="col.label"
            :min-width="col.minWidth" header-align="center"
            :class-name="col.overflowEllipsis ? 'bms-td-margin' : 'bms-td-wrap'"
            :show-overflow-tooltip="!!col.overflowEllipsis" />
        </el-table>
      </div>
    </main>

    <footer class="bms-footer">
      <el-button type="primary" class="bms-footer-btn bms-footer-btn--ok" @click="onConfirm">确认</el-button>
      <el-button class="bms-footer-btn bms-footer-btn--cancel" @click="onCancel">取消</el-button>
    </footer>
  </div>
</template>

<style scoped>
.bms {
  --bms-warn: #f39800;
  --bms-surface: #f7f9fb;
  --pri: #0066ff;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--bms-surface);
  color: #191c1e;
  font-family: 'Inter', 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: env(safe-area-inset-bottom);
}

.bms-header {
  flex-shrink: 0;
  padding-top: max(0.875rem, env(safe-area-inset-top));
  padding-left: 0;
  padding-right: 0;
  padding-bottom: 0;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  color: #191c1e;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.04);
  border-bottom: 1px solid #f1f5f9;
}

/* 左右等宽列 + 中间 auto，标题相对视口水平居中，右侧是否显示「新增方案」不挤偏 */
.bms-header-top {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  column-gap: 0.5rem;
  padding: 0 0.75rem 0.75rem;
}

.bms-tabs-row {
  width: 100%;
}

.bms-back {
  justify-self: start;
  flex-shrink: 0;
  width: 2.25rem;
  height: 2.25rem;
  padding: 0;
  border: none;
  background: transparent;
  color: #0f172a;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  -webkit-tap-highlight-color: transparent;
}

.bms-back:focus-visible {
  outline: 2px solid #0066ff;
  outline-offset: 2px;
}

.bms-back-ico {
  display: block;
}

.bms-title {
  margin: 0;
  justify-self: center;
  text-align: center;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.bms-header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  justify-self: end;
  min-width: 0;
}

.bms-add-scheme {
  font-weight: 600;
}

.bms-main {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0.75rem 0.75rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  -webkit-overflow-scrolling: touch;
}

.bms-card {
  background: #fff;
  border-radius: 0.75rem;
  padding: 1rem 0.875rem;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
}

.bms-card--simple {
  padding: 0.75rem;
}

.bms-field-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.35rem 0.5rem;
  margin-bottom: 0.875rem;
}

.bms-lbl {
  font-size: 0.8125rem;
  color: #334155;
  font-weight: 500;
  flex-shrink: 0;
}

.bms-lbl--push {
  margin-left: auto;
}

@media (max-width: 360px) {
  .bms-lbl--push {
    margin-left: 0;
    width: 100%;
  }
}

.bms-inp-short {
  width: 5rem;
}

.bms-unit {
  font-size: 0.8125rem;
  color: #64748b;
}

.bms-select-mode {
  width: 5rem;
}

.bms-radio-block {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  margin-bottom: 1rem;
}

.bms-radio-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  font-size: 0.8125rem;
  cursor: pointer;
}

.bms-radio-row--accum {
  flex-wrap: wrap;
}

.bms-accum-label {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  cursor: pointer;
}

.bms-native-radio {
  width: 1rem;
  height: 1rem;
  flex-shrink: 0;
  accent-color: #0066ff;
}

.bms-radio-lbl {
  color: #334155;
  flex-shrink: 0;
}

.bms-inp-grow {
  flex: 1 1 8rem;
  min-width: 6rem;
}

.bms-inp-tiny {
  width: 4rem;
}

.bms-action-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
}

.bms-btn-generate {
  margin: 0;
  height: auto;
  padding: 0.5rem 0.35rem;
  font-size: 0.6875rem;
  line-height: 1.35;
  white-space: normal;
  border-radius: 0.5rem;
}

.bms-btn-generate--solid {
  --el-button-bg-color: var(--bms-warn);
  --el-button-border-color: var(--bms-warn);
  --el-button-hover-bg-color: #e08900;
  --el-button-hover-border-color: #e08900;
}

.bms-btn-generate.is-plain {
  --el-button-text-color: var(--bms-warn);
  --el-button-border-color: var(--bms-warn);
  --el-button-bg-color: #fff;
}

.bms-field-row--oneclick {
  flex-wrap: nowrap;
  margin-bottom: 0.75rem;
}

.bms-field-row--oneclick .bms-lbl {
  flex: 0 0 5.25rem;
  text-align: left;
}

.bms-field-row--oneclick+.bms-field-row--oneclick {
  margin-bottom: 1rem;
}

.bms-oneclick-grow {
  flex: 1 1 0;
  min-width: 0;
}

.bms-oneclick-grow :deep(.el-input) {
  width: 100%;
}

.bms-inp-suffix-wrap {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.bms-inp-suffix-wrap :deep(.el-input) {
  flex: 1;
}

.bms-suffix {
  font-size: 0.875rem;
  color: #64748b;
  font-weight: 600;
}

.bms-textarea :deep(.el-textarea__inner) {
  font-size: 0.9375rem;
  line-height: 1.5;
  border-radius: 0.5rem;
}

.bms-hint {
  margin: 0;
  font-size: 0.6875rem;
  line-height: 1.45;
}

.bms-hint--danger {
  color: #ba1a1a;
}

.bms-hint--primary {
  color: #0066ff;
}

.bms-advanced-list {
  background: #fff;
  border-radius: 0.75rem;
  overflow: hidden;
  box-shadow: 0 8px 30px rgba(25, 28, 30, 0.06);
}

.bms-advanced-head {
  display: flex;
  justify-content: space-between;
  padding: 0.65rem 0.875rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: #0066ff;
  border-bottom: 1px solid #f1f5f9;
}

.bms-advanced-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.75rem 0.875rem;
  font-size: 0.8125rem;
}

.bms-advanced-row--alt {
  background: #f8fafc;
}

.bms-advanced-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.bms-advanced-title {
  color: #0f172a;
}

.bms-advanced-title--link {
  cursor: pointer;
  color: #0066ff;
  font-weight: 600;
}

.bms-advanced-title--link:hover {
  text-decoration: underline;
}

.bms-advanced-ops {
  display: flex;
  gap: 0.35rem;
  flex-shrink: 0;
}

.bms-icon-btn {
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  flex-shrink: 0;
  background-size: 55%;
  background-repeat: no-repeat;
  background-position: center;
  -webkit-tap-highlight-color: transparent;
}

.bms-icon-btn--edit {
  background-color: #0066ff;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='white'%3E%3Cpath d='M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z'/%3E%3C/svg%3E");
}

.bms-icon-btn--del {
  background-color: #e2e8f0;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%2364748b'%3E%3Cpath d='M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z'/%3E%3C/svg%3E");
}

.bms-icon-btn:focus-visible {
  outline: 2px solid #0066ff;
  outline-offset: 2px;
}

/* 底部计划表：与玩法详情「投注」Tab 的 .table-card + .detail-bet-table 一致 */
.table-card {
  margin-top: 0.25rem;
  background: #fff;
  border-radius: 0.75rem;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  overflow: hidden;
  padding: 0;
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

.detail-bet-table :deep(.el-table__body .el-table__cell) {
  font-size: 11px;
  vertical-align: top;
}

/* 除利润率外：内容可换行，不撑出横向滚动 */
.detail-bet-table :deep(.bms-td-wrap .cell) {
  white-space: normal !important;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.45;
}

/* 利润率列：单行省略号，不超出单元格 */
.detail-bet-table :deep(td.bms-td-margin) {
  overflow: hidden;
}

.detail-bet-table :deep(.bms-td-margin .cell) {
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
}

.bms-footer {
  flex-shrink: 0;
  display: flex;
  gap: 0.65rem;
  padding: 0.75rem;
  padding-bottom: max(0.75rem, env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
  border-top: 1px solid #e2e8f0;
  backdrop-filter: blur(12px);
}

.bms-footer-btn {
  flex: 1;
  margin: 0;
  height: 2.75rem;
  font-weight: 700;
  border-radius: 0.625rem;
}

.bms-footer-btn--ok {
  background: #0066ff;
  border: none;
}

.bms-footer-btn--cancel {
  --el-button-bg-color: #fff;
  --el-button-text-color: #0066ff;
  --el-button-border-color: #0066ff;
}
</style>
