<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()

const BACK_ICO = '/images/lobby/icon-placeholder.png'

const LOTTERY_LABELS: Record<string, string> = {
  tencent_ffc: '腾讯分分彩',
  tencent_10: '腾讯十分彩',
  qiqu_tencent: '奇趣腾讯分分彩',
  us_ffc: '美国数据分分彩',
  cq_ssc: '重庆时时彩',
  xj_ssc: '新疆时时彩',
  tj_ssc: '天津时时彩',
  fc_3d: '福彩3D',
}

const runMode = ref<'prod' | 'sim'>('prod')
const schemeName = ref(decodeURIComponent(String(route.query.title ?? '') || '精算方案-A01'))
const shareStatus = ref('private')
const schemeFunds = ref('10000')
/** 24h HH:mm，开始与结束均需设置 */
const startTime = ref('00:00')
const endTime = ref('23:59')
const stopLoss = ref('')
const takeProfit = ref('')
const multCoeff = ref('1.0')
const betMode = ref('2')
/** 方案内容按组划分，默认一组 */
const schemeGroups = ref<string[]>([''])

const gameNameDisplay = computed(() => {
  const id = String(route.query.lottery ?? '')
  return LOTTERY_LABELS[id] ?? '—'
})

function tokenCount(raw: string): number {
  const t = raw.trim()
  if (!t) return 0
  return t
    .split(/[\s,，、\n\r]+/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0).length
}

const shareOptions = [
  { label: '私密 (仅自己可见)', value: 'private' },
  { label: '公开 (允许他人跟单)', value: 'public' },
]

const betModeOptions = [
  { label: '2元模式', value: '2' },
  { label: '10元模式', value: '10' },
  { label: '自由配置', value: 'custom' },
]

/** 倍投设定 Tab 与中文名称（与 BetMultiplierSettingsView 一致） */
const BET_MULTIPLIER_KIND_LABELS: Record<string, string> = {
  '0': '小白倍投',
  '1': '一键倍投',
  '2': '简单倍投',
  '3': '高级倍投',
}

/** 从本页进入倍投设定再返回时恢复滚动（避免回到页面顶部） */
function scrollRestoreStorageKey(): string {
  return `advanced-scheme-edit:scrollY:${String(route.params.schemeId ?? '')}`
}

function readDocumentScrollY(): number {
  return window.scrollY || document.documentElement.scrollTop || 0
}

onMounted(() => {
  const raw = sessionStorage.getItem(scrollRestoreStorageKey())
  if (raw == null) return
  sessionStorage.removeItem(scrollRestoreStorageKey())
  const y = Number(raw)
  if (!Number.isFinite(y) || y < 0) return
  nextTick(() => {
    requestAnimationFrame(() => {
      window.scrollTo(0, y)
      requestAnimationFrame(() => {
        window.scrollTo(0, y)
      })
    })
  })
})

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push({ name: 'custom-scheme-new' })
}

/** 倍投设定页校验失败：query.bmsError；确认成功：query.bmsKind（0–3） */
const betMultiplierError = ref('')
/** 最近一次在倍投设定页确认通过的方式名称，显示在齿轮下（与报错二选一优先报错） */
const betMultiplierSelectedLabel = ref('')

watch(
  () => route.query.bmsKind,
  (k) => {
    if (k == null || k === '') return
    const id = String(Array.isArray(k) ? k[0] : k)
    const lbl = BET_MULTIPLIER_KIND_LABELS[id]
    if (lbl) {
      betMultiplierSelectedLabel.value = lbl
      betMultiplierError.value = ''
    }
    const nextQuery = { ...route.query } as Record<string, string | string[] | undefined>
    delete nextQuery.bmsKind
    void router.replace({ query: nextQuery })
  },
  { immediate: true }
)

watch(
  () => route.query.bmsError,
  (q) => {
    if (q == null || q === '') return
    const raw = String(Array.isArray(q) ? q[0] : q)
    try {
      betMultiplierError.value = decodeURIComponent(raw)
    } catch {
      betMultiplierError.value = raw
    }
    betMultiplierSelectedLabel.value = ''
    const nextQuery = { ...route.query } as Record<string, string | string[] | undefined>
    delete nextQuery.bmsError
    delete nextQuery.activeTab
    void router.replace({ query: nextQuery })
  },
  { immediate: true }
)

function goBetMultiplierSettings() {
  betMultiplierError.value = ''
  sessionStorage.setItem(scrollRestoreStorageKey(), String(readDocumentScrollY()))
  const tabEntry = Object.entries(BET_MULTIPLIER_KIND_LABELS).find(
    ([, l]) => l === betMultiplierSelectedLabel.value
  )
  router.push({
    name: 'bet-multiplier-settings',
    query: {
      fromScheme: '1',
      schemeId: String(route.params.schemeId ?? ''),
      ...(tabEntry ? { activeTab: tabEntry[0] } : {}),
      ...(route.query.title != null && route.query.title !== ''
        ? { title: String(route.query.title) }
        : {}),
      ...(route.query.lottery != null && String(route.query.lottery) !== ''
        ? { lottery: String(route.query.lottery) }
        : {}),
    },
  })
}

async function onPaste(groupIdx: number) {
  try {
    const text = await navigator.clipboard.readText()
    if (!text?.trim()) {
      ElMessage.info('剪贴板为空')
      return
    }
    const cur = schemeGroups.value[groupIdx] ?? ''
    schemeGroups.value[groupIdx] = cur ? `${cur.trimEnd()}\n${text.trim()}` : text.trim()
    ElMessage.success('已粘贴')
  } catch {
    ElMessage.warning('无法读取剪贴板，请长按使用系统粘贴')
  }
}

function onClearContent(groupIdx: number) {
  schemeGroups.value[groupIdx] = ''
  ElMessage.info('已清空')
}

function onDeleteGroup(groupIdx: number) {
  if (schemeGroups.value.length <= 1) {
    ElMessageBox.confirm('仅剩一组，将清空该组内容？', '清空组', {
      confirmButtonText: '清空',
      cancelButtonText: '取消',
      type: 'warning',
    })
      .then(() => {
        schemeGroups.value[0] = ''
        ElMessage.success('已清空')
      })
      .catch(() => { })
    return
  }
  ElMessageBox.confirm('确定删除该分组？', '删除组', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      schemeGroups.value.splice(groupIdx, 1)
      ElMessage.success('已删除')
    })
    .catch(() => { })
}

function onAddGroup() {
  schemeGroups.value.push('')
}

function onSaveCloud() {
  ElMessage.success('已保存并同步至云端（演示）')
  router.push({ name: 'lobby' })
}

// ----- 运行时段弹窗（滚轮 + 开始/结束切换） -----
const TW_ITEM_H = 44
const twHours24 = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'))
const twMinutes = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, '0'))

const timeDialogVisible = ref(false)
const timeActive = ref<'start' | 'end'>('start')
const pendingStart = ref('00:00')
const pendingEnd = ref('23:59')

const selHourIdx = ref(0)
const selMinIdx = ref(0)

const hourScrollRef = ref<HTMLElement | null>(null)
const minScrollRef = ref<HTMLElement | null>(null)

let twScrollTimer: ReturnType<typeof setTimeout> | null = null

function parseHm(s: string): { h: number; m: number } | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec((s ?? '').trim())
  if (!m) return null
  const h = Number(m[1])
  const mi = Number(m[2])
  if (Number.isNaN(h) || Number.isNaN(mi) || h < 0 || h > 23 || mi < 0 || mi > 59) return null
  return { h, m: mi }
}

function normalizeHm(s: string, fallback = '00:00'): string {
  const p = parseHm(s)
  if (!p) return fallback
  return `${String(p.h).padStart(2, '0')}:${String(p.m).padStart(2, '0')}`
}

/** 24h：小时 0–23 → selHourIdx 0–23 */
function hmToPickerParts(hm: string): { hi: number; mi: number } {
  const p = parseHm(hm) ?? { h: 0, m: 0 }
  return { hi: p.h, mi: p.m }
}

function pickerPartsToHm(hi: number, mi: number): string {
  const h = Math.max(0, Math.min(23, hi))
  const m = Math.max(0, Math.min(59, mi))
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

function hmFromPicker(): string {
  return pickerPartsToHm(selHourIdx.value, selMinIdx.value)
}

function loadPickerFromHm(hm: string) {
  const { hi, mi } = hmToPickerParts(normalizeHm(hm))
  selHourIdx.value = hi
  selMinIdx.value = mi
}

function snapScroll(el: HTMLElement | null, idx: number, maxIdx: number) {
  if (!el) return
  const i = Math.max(0, Math.min(maxIdx, idx))
  el.scrollTo({ top: i * TW_ITEM_H, behavior: 'auto' })
}

function snapAllScrolls() {
  snapScroll(hourScrollRef.value, selHourIdx.value, 23)
  snapScroll(minScrollRef.value, selMinIdx.value, 59)
}

function scheduleTwScrollSync(kind: 'h' | 'm') {
  if (twScrollTimer) clearTimeout(twScrollTimer)
  twScrollTimer = setTimeout(() => finalizeTwScroll(kind), 72)
}

function finalizeTwScroll(kind: 'h' | 'm') {
  if (kind === 'h' && hourScrollRef.value) {
    const idx = Math.round(hourScrollRef.value.scrollTop / TW_ITEM_H)
    selHourIdx.value = Math.max(0, Math.min(23, idx))
    snapScroll(hourScrollRef.value, selHourIdx.value, 23)
  }
  if (kind === 'm' && minScrollRef.value) {
    const idx = Math.round(minScrollRef.value.scrollTop / TW_ITEM_H)
    selMinIdx.value = Math.max(0, Math.min(59, idx))
    snapScroll(minScrollRef.value, selMinIdx.value, 59)
  }
}

function twSelectHour(idx: number) {
  selHourIdx.value = idx
  snapScroll(hourScrollRef.value, idx, 23)
}

function twSelectMin(idx: number) {
  selMinIdx.value = idx
  snapScroll(minScrollRef.value, idx, 59)
}

function setTimeActive(tab: 'start' | 'end') {
  if (tab === timeActive.value) return
  if (timeActive.value === 'start') pendingStart.value = hmFromPicker()
  else pendingEnd.value = hmFromPicker()
  timeActive.value = tab
  const hm = tab === 'start' ? pendingStart.value : pendingEnd.value
  loadPickerFromHm(hm)
  nextTick(() => snapAllScrolls())
}

function formatHm24Label(hm: string): string {
  return normalizeHm(hm)
}

const displayStartSummary = computed(() => formatHm24Label(pendingStart.value))
const displayEndSummary = computed(() => formatHm24Label(pendingEnd.value))

function openTimeDialog(focus: 'start' | 'end' = 'start') {
  pendingStart.value = normalizeHm(startTime.value)
  pendingEnd.value = normalizeHm(endTime.value || '23:59', '23:59')
  timeActive.value = focus
  const hm = focus === 'start' ? pendingStart.value : pendingEnd.value
  loadPickerFromHm(hm)
  timeDialogVisible.value = true
  nextTick(() => snapAllScrolls())
}

function confirmTimeDialog() {
  if (timeActive.value === 'start') pendingStart.value = hmFromPicker()
  else pendingEnd.value = hmFromPicker()

  startTime.value = normalizeHm(pendingStart.value)
  endTime.value = normalizeHm(pendingEnd.value)
  timeDialogVisible.value = false
}

const displayMainStart = computed(() => formatHm24Label(startTime.value))
const displayMainEnd = computed(() => formatHm24Label(endTime.value))

function onTimeDialogOpened() {
  nextTick(() => snapAllScrolls())
}
</script>

<template>
  <div class="scf">
    <header class="scf-header">
      <button type="button" class="scf-back" aria-label="返回" @click="goBack">
        <img :src="BACK_ICO" alt="" width="24" height="24" class="scf-back-img" decoding="async" />
      </button>
      <h1 class="scf-title">方案配置</h1>
      <div class="scf-header-right" aria-hidden="true" />
    </header>

    <main class="scf-main">
      <section class="scf-section">
        <div class="scf-section-head">
          <h2 class="scf-section-title">基础设置</h2>
        </div>
        <div class="scf-card scf-stack">
          <div class="scf-field">
            <span class="scf-lbl">运行模式</span>
            <div class="scf-seg" role="group" aria-label="运行模式">
              <button type="button" class="scf-seg-btn" :class="{ 'is-active': runMode === 'prod' }"
                @click="runMode = 'prod'">
                正式运行
              </button>
              <button type="button" class="scf-seg-btn" :class="{ 'is-active': runMode === 'sim' }"
                @click="runMode = 'sim'">
                模拟运行
              </button>
            </div>
          </div>
          <div class="scf-field">
            <label class="scf-lbl" for="scf-name">方案名称</label>
            <el-input id="scf-name" v-model="schemeName" size="large" class="scf-el-inp" placeholder="方案名称" />
          </div>
          <div class="scf-field">
            <span class="scf-lbl">分享状态</span>
            <el-select v-model="shareStatus" class="scf-el-select" size="large" placeholder="选择">
              <el-option v-for="o in shareOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </div>
          <div class="scf-grid2">
            <div class="scf-field">
              <label class="scf-lbl" for="scf-funds">方案资金</label>
              <div class="scf-suffix-wrap">
                <el-input id="scf-funds" v-model="schemeFunds" size="large" class="scf-el-inp scf-el-inp--suffix"
                  type="number" />
                <span class="scf-suffix">CNY</span>
              </div>
            </div>
            <div class="scf-field">
              <span class="scf-lbl">游戏名称</span>
              <div class="scf-readonly">{{ gameNameDisplay }}</div>
            </div>
          </div>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head scf-section-head--plain">
          <h2 class="scf-section-title">运行逻辑</h2>
        </div>
        <div class="scf-card scf-stack">
          <div class="scf-tip">
            <p>
              提示：方案保存后将自动同步至精算云中心，系统将根据设定的时间范围自动执行逻辑任务。
            </p>
          </div>
          <div class="scf-grid2">
            <div class="scf-field">
              <span class="scf-lbl">开始时间</span>
              <button type="button" class="scf-time-hit" aria-haspopup="dialog" @click="openTimeDialog('start')">
                <span class="scf-time-hit-val">{{ displayMainStart }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">schedule</span>
              </button>
            </div>
            <div class="scf-field">
              <span class="scf-lbl">结束时间</span>
              <button type="button" class="scf-time-hit" aria-haspopup="dialog" @click="openTimeDialog('end')">
                <span class="scf-time-hit-val">{{ displayMainEnd }}</span>
                <span class="scf-ms scf-ms--sm scf-time-hit-ico" aria-hidden="true">schedule</span>
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head scf-section-head--plain">
          <h2 class="scf-section-title">风险控制</h2>
        </div>
        <div class="scf-card scf-grid2">
          <div class="scf-field">
            <label class="scf-lbl" for="scf-sl">止损金额</label>
            <el-input id="scf-sl" v-model="stopLoss" size="large" class="scf-el-inp scf-el-inp--danger"
              placeholder="0.00" type="number" />
          </div>
          <div class="scf-field">
            <label class="scf-lbl" for="scf-tp">止盈金额</label>
            <el-input id="scf-tp" v-model="takeProfit" size="large" class="scf-el-inp scf-el-inp--profit"
              placeholder="0.00" type="number" />
          </div>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head scf-section-head--plain">
          <h2 class="scf-section-title">投注参数</h2>
        </div>
        <div class="scf-card scf-stack">
          <div class="scf-grid2">
            <div class="scf-field">
              <label class="scf-lbl" for="scf-mult">倍数系数</label>
              <el-input id="scf-mult" v-model="multCoeff" size="large" class="scf-el-inp" type="number" step="0.1" />
            </div>
            <div class="scf-field">
              <span class="scf-lbl">投注模式</span>
              <el-select v-model="betMode" class="scf-el-select" size="large">
                <el-option v-for="o in betModeOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </div>
          </div>
          <button type="button" class="scf-mode-card" @click="goBetMultiplierSettings">
            <div class="scf-mode-left">
              <span class="scf-mode-ico-bg" aria-hidden="true">
                <span class="scf-ms scf-ms--white">analytics</span>
              </span>
              <div class="scf-mode-texts">
                <p class="scf-mode-title">方案模式设置</p>
                <p class="scf-mode-sub">配置数学期望与倍增逻辑</p>
              </div>
            </div>
            <div class="scf-mode-right">
              <span class="scf-ms scf-ms--primary scf-mode-gear" aria-hidden="true">settings</span>
              <p v-if="betMultiplierError" class="scf-mode-err" role="alert">
                {{ betMultiplierError }}
              </p>
              <p v-else-if="betMultiplierSelectedLabel" class="scf-mode-err">
                {{ betMultiplierSelectedLabel }}
              </p>
            </div>
          </button>
        </div>
      </section>

      <section class="scf-section">
        <div class="scf-section-head">
          <h2 class="scf-section-title">方案内容</h2>
          <button type="button" class="scf-add-btn" @click="onAddGroup">
            <span class="scf-ms scf-ms--sm" aria-hidden="true">add</span>
            <span>新增</span>
          </button>
        </div>
        <div class="scf-groups-stack">
          <div
            v-for="(_, idx) in schemeGroups"
            :key="idx"
            class="scf-content-card"
          >
            <div class="scf-group-bar">
              <h3 class="scf-group-title">第 {{ idx + 1 }} 组</h3>
              <div class="scf-content-toolbar scf-content-toolbar--group" role="toolbar" :aria-label="`第 ${idx + 1} 组操作`">
                <button type="button" class="scf-tb-btn" @click="onPaste(idx)">
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">content_paste</span>
                  <span>粘贴</span>
                </button>
                <button type="button" class="scf-tb-btn scf-tb-btn--muted" @click="onClearContent(idx)">
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">backspace</span>
                  <span>清空</span>
                </button>
                <button type="button" class="scf-tb-btn scf-tb-btn--danger" @click="onDeleteGroup(idx)">
                  <span class="scf-ms scf-ms--sm" aria-hidden="true">delete</span>
                  <span>删除组</span>
                </button>
              </div>
            </div>
            <div class="scf-textarea-wrap">
              <el-input
                v-model="schemeGroups[idx]"
                type="textarea"
                :rows="5"
                resize="none"
                class="scf-area"
                placeholder="请输入数字，使用逗号分隔 (例如: 01,02,03...)"
              />
              <div class="scf-area-meta">
                <span>支持格式: 数字,空格,换行</span>
                <span>当前计数: {{ tokenCount(schemeGroups[idx]) }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div class="scf-main-pad" aria-hidden="true" />
    </main>

    <el-dialog
      v-model="timeDialogVisible"
      title="运行时段"
      width="min(22rem, calc(100vw - 2rem))"
      class="scf-tw-dialog"
      modal-class="scf-tw-overlay"
      append-to-body
      align-center
      destroy-on-close
      @opened="onTimeDialogOpened"
    >
      <div class="scf-tw">
        <div class="scf-tw-wheel-wrap">
          <div class="scf-tw-highlight" aria-hidden="true" />
          <div class="scf-tw-row">
            <div class="scf-tw-mask scf-tw-mask--hour">
              <div
                ref="hourScrollRef"
                class="scf-tw-scroll"
                role="listbox"
                aria-label="小时（24 小时制）"
                @scroll.passive="scheduleTwScrollSync('h')"
              >
                <div class="scf-tw-spacer" aria-hidden="true" />
                <div
                  v-for="(h, idx) in twHours24"
                  :key="'h' + h"
                  class="scf-tw-cell"
                  :class="{ 'is-sel': selHourIdx === idx }"
                  role="option"
                  :aria-selected="selHourIdx === idx"
                  @click="twSelectHour(idx)"
                >
                  {{ h }}
                </div>
                <div class="scf-tw-spacer" aria-hidden="true" />
              </div>
            </div>
            <span class="scf-tw-colon" aria-hidden="true">:</span>
            <div class="scf-tw-mask scf-tw-mask--min">
              <div
                ref="minScrollRef"
                class="scf-tw-scroll"
                role="listbox"
                aria-label="分钟"
                @scroll.passive="scheduleTwScrollSync('m')"
              >
                <div class="scf-tw-spacer" aria-hidden="true" />
                <div
                  v-for="(n, idx) in twMinutes"
                  :key="'m' + n"
                  class="scf-tw-cell"
                  :class="{ 'is-sel': selMinIdx === idx }"
                  role="option"
                  :aria-selected="selMinIdx === idx"
                  @click="twSelectMin(idx)"
                >
                  {{ n }}
                </div>
                <div class="scf-tw-spacer" aria-hidden="true" />
              </div>
            </div>
          </div>
        </div>

        <div class="scf-tw-summary">
          <button
            type="button"
            class="scf-tw-sum-half"
            :class="{ 'is-active': timeActive === 'start' }"
            @click="setTimeActive('start')"
          >
            <span class="scf-tw-sum-lbl">开始时间</span>
            <span class="scf-tw-sum-val">{{ displayStartSummary }}</span>
          </button>
          <button
            type="button"
            class="scf-tw-sum-half"
            :class="{ 'is-active': timeActive === 'end' }"
            @click="setTimeActive('end')"
          >
            <span class="scf-tw-sum-lbl">结束时间</span>
            <span class="scf-tw-sum-val">{{ displayEndSummary }}</span>
          </button>
        </div>

        <el-button type="primary" class="scf-tw-confirm" size="large" @click="confirmTimeDialog">
          <span>确认选择</span>
          <span class="scf-tw-check" aria-hidden="true">
            <span class="scf-ms scf-ms--fill scf-ms--white scf-tw-check-ico">check</span>
          </span>
        </el-button>
      </div>
    </el-dialog>

    <footer class="scf-footer">
      <el-button type="primary" class="scf-cloud-btn" size="large" @click="onSaveCloud">
        <span class="scf-ms scf-ms--fill scf-cloud-ico" aria-hidden="true">cloud_upload</span>
        保存并同步至云端
      </el-button>
    </footer>
  </div>
</template>

<style scoped>
.scf {
  --scf-surface: #f7f9fb;
  --scf-primary: #0050cb;
  --scf-primary-strong: #0066ff;
  --scf-on-variant: #424656;
  --scf-outline: #c2c6d8;
  --scf-error: #ba1a1a;
  --scf-tertiary: #a33200;
  --scf-secondary-container: #9bb4fe;
  --scf-on-secondary-container: #f8f7ff;
  --scf-error-container: #ffdad6;
  min-height: 100dvh;
  background: var(--scf-surface);
  color: #191c1e;
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: env(safe-area-inset-bottom);
}

.scf-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.375rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  vertical-align: middle;
  user-select: none;
}

.scf-ms--sm {
  font-size: 1.25rem;
}

.scf-ms--primary {
  color: var(--scf-primary-strong);
}

.scf-ms--white {
  color: #fff;
}

.scf-ms--fill {
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}

.scf-header {
  position: sticky;
  top: 0;
  z-index: 50;
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

.scf-back {
  justify-self: start;
  width: 2.25rem;
  height: 2.25rem;
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.scf-back-img {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
  display: block;
  pointer-events: none;
}

.scf-back:focus-visible {
  outline: 2px solid var(--scf-primary-strong);
  outline-offset: 2px;
}

.scf-title {
  margin: 0;
  justify-self: center;
  text-align: center;
  font-size: 1.0625rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.scf-header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  justify-self: end;
  min-width: 0;
}

.scf-main {
  padding: 1.25rem 1rem 0;
  max-width: 32rem;
  margin: 0 auto;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.scf-main-pad {
  height: 6rem;
}

.scf-section {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.scf-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0.25rem;
}

.scf-section-head--plain {
  justify-content: flex-start;
}

.scf-section-title {
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--scf-on-variant);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.scf-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
  background: var(--scf-secondary-container);
  color: var(--scf-on-secondary-container);
}

.scf-card {
  background: #fff;
  border-radius: 0.875rem;
  padding: 1.15rem 1rem;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scf-stack {
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}

.scf-grid2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

@media (max-width: 380px) {
  .scf-grid2 {
    grid-template-columns: 1fr;
  }
}

.scf-field {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  min-width: 0;
}

.scf-lbl {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--scf-on-variant);
  padding-left: 0.15rem;
}

.scf-seg {
  display: flex;
  gap: 0.25rem;
  padding: 0.25rem;
  background: #f2f4f6;
  border-radius: 0.5rem;
}

.scf-seg-btn {
  flex: 1;
  border: none;
  border-radius: 0.375rem;
  padding: 0.5rem 0.35rem;
  font-size: 0.875rem;
  font-weight: 600;
  font-family: inherit;
  color: var(--scf-on-variant);
  background: transparent;
  cursor: pointer;
  transition:
    background 0.15s,
    box-shadow 0.15s,
    color 0.15s;
}

.scf-seg-btn:hover {
  background: rgba(255, 255, 255, 0.55);
}

.scf-seg-btn.is-active {
  background: #fff;
  color: var(--scf-primary-strong);
  box-shadow: 0 1px 4px rgba(25, 28, 30, 0.08);
}

.scf-el-inp :deep(.el-input__wrapper) {
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  padding-left: 0.9rem;
}

.scf-el-inp :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px rgba(0, 102, 255, 0.35) inset;
}

.scf-time-hit {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 2.5rem;
  padding: 0.55rem 0.9rem;
  border: none;
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition:
    box-shadow 0.15s,
    background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-time-hit:hover {
  background: rgba(242, 244, 246, 0.85);
}

.scf-time-hit:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.28);
}

.scf-time-hit-val {
  font-size: 0.9375rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--scf-primary-strong);
}

.scf-time-hit-ico {
  flex-shrink: 0;
  opacity: 0.65;
  color: var(--scf-primary-strong);
}

.scf-el-inp--danger :deep(.el-input__inner) {
  color: var(--scf-error);
  font-weight: 700;
}

.scf-el-inp--profit :deep(.el-input__inner) {
  color: var(--scf-tertiary);
  font-weight: 700;
}

.scf-el-select {
  width: 100%;
}

.scf-el-select :deep(.el-select__wrapper) {
  border-radius: 0.5rem;
  background: #f2f4f6;
  box-shadow: none;
  min-height: 2.5rem;
}

.scf-suffix-wrap {
  position: relative;
}

.scf-el-inp--suffix :deep(.el-input__wrapper) {
  padding-right: 3rem;
}

.scf-suffix {
  position: absolute;
  right: 0.85rem;
  top: 50%;
  transform: translateY(-50%);
  font-size: 0.8125rem;
  font-weight: 700;
  color: #727687;
  pointer-events: none;
}

.scf-readonly {
  min-height: 2.5rem;
  padding: 0.55rem 0.9rem;
  border-radius: 0.5rem;
  background: rgba(230, 232, 234, 0.35);
  border: 1px solid rgba(194, 198, 216, 0.35);
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--scf-on-variant);
  display: flex;
  align-items: center;
}

.scf-tip {
  border-left: 4px solid var(--scf-error);
  border-radius: 0 0.5rem 0.5rem 0;
  padding: 0.65rem 0.75rem;
  background: rgba(255, 218, 214, 0.45);
}

.scf-tip p {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 500;
  line-height: 1.55;
  color: var(--scf-error);
}

.scf-mode-card {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.95rem 1rem;
  border: none;
  border-radius: 0.875rem;
  background: rgba(0, 102, 255, 0.06);
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-mode-card:hover {
  background: rgba(0, 102, 255, 0.1);
}

.scf-mode-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
  flex: 1 1 0;
}

.scf-mode-right {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.35rem;
  max-width: min(11rem, 42%);
  padding-top: 0.1rem;
}

.scf-mode-err {
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--scf-error);
  text-align: right;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.scf-mode-ico-bg {
  flex-shrink: 0;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 999px;
  background: var(--scf-primary-strong);
  display: flex;
  align-items: center;
  justify-content: center;
}

.scf-mode-texts {
  text-align: left;
  min-width: 0;
}

.scf-mode-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--scf-primary-strong);
}

.scf-mode-sub {
  margin: 0.15rem 0 0;
  font-size: 11px;
  color: var(--scf-on-variant);
  opacity: 0.78;
}

.scf-mode-gear {
  flex-shrink: 0;
  transition: transform 0.15s;
}

.scf-mode-card:hover .scf-mode-gear {
  transform: translateX(3px);
}

.scf-add-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
  padding: 0.35rem 0.65rem;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: var(--scf-primary-strong);
  font-size: 0.8125rem;
  font-weight: 700;
  font-family: inherit;
  cursor: pointer;
  transition: background 0.15s;
}

.scf-add-btn:hover {
  background: rgba(0, 80, 203, 0.06);
}

.scf-groups-stack {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.scf-group-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  flex-wrap: wrap;
  padding: 0.65rem 1rem;
  border-bottom: 1px solid rgba(194, 198, 216, 0.2);
  background: #fff;
  min-width: 0;
}

.scf-group-title {
  margin: 0;
  flex-shrink: 0;
  font-size: 0.875rem;
  font-weight: 700;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  letter-spacing: -0.01em;
  color: var(--scf-primary-strong);
}

.scf-content-toolbar--group {
  flex: 1;
  display: flex;
  justify-content: flex-end;
  align-items: stretch;
  align-self: stretch;
  min-width: min(12rem, 100%);
  border-bottom: none;
}

.scf-content-toolbar--group .scf-tb-btn {
  flex: 0 1 auto;
  padding: 0.5rem 0.55rem;
}

.scf-content-card {
  background: #fff;
  border-radius: 0.875rem;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.scf-content-toolbar {
  display: flex;
  border-bottom: 1px solid rgba(194, 198, 216, 0.2);
}

.scf-tb-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.75rem 0.25rem;
  border: none;
  border-right: 1px solid rgba(194, 198, 216, 0.2);
  background: #fff;
  font-size: 0.75rem;
  font-weight: 700;
  font-family: inherit;
  color: var(--scf-primary-strong);
  cursor: pointer;
  transition: background 0.15s;
}

.scf-tb-btn:last-child {
  border-right: none;
}

.scf-tb-btn:hover {
  background: #f2f4f6;
}

.scf-tb-btn--muted {
  color: var(--scf-on-variant);
}

.scf-tb-btn--danger {
  color: var(--scf-error);
}

.scf-textarea-wrap {
  padding: 1rem;
}

.scf-area :deep(.el-textarea__inner) {
  border: none;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.65);
  padding: 1rem;
  font-size: 0.9375rem;
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  line-height: 1.55;
  box-shadow: none;
}

.scf-area :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 2px rgba(0, 102, 255, 0.18);
}

.scf-area-meta {
  margin-top: 0.65rem;
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 11px;
  color: #727687;
  padding: 0 0.2rem;
}

.scf-footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 50;
  padding: 0.85rem 1rem max(1rem, env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  box-shadow: 0 -10px 40px rgba(25, 28, 30, 0.06);
}

.scf-cloud-btn {
  width: 100%;
  height: 3.25rem;
  margin: 0;
  border-radius: 0.75rem;
  font-weight: 700;
  font-size: 1rem;
  border: none;
  box-shadow: 0 8px 24px rgba(0, 102, 255, 0.22);
}

.scf-cloud-ico {
  margin-right: 0.35rem;
  font-size: 1.35rem;
  vertical-align: -0.15em;
}

/* ----- 运行时段弹窗（滚轮） ----- */
.scf-tw {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding-bottom: 0.15rem;
}

.scf-tw-wheel-wrap {
  position: relative;
  padding: 0.35rem 0 0.15rem;
}

.scf-tw-highlight {
  position: absolute;
  left: 0.3rem;
  right: 0.3rem;
  top: 50%;
  transform: translateY(-50%);
  height: 44px;
  border-radius: 0.5rem;
  background: rgba(0, 102, 255, 0.09);
  pointer-events: none;
  z-index: 0;
}

.scf-tw-row {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: stretch;
  justify-content: center;
  gap: 0.2rem;
}

.scf-tw-mask {
  position: relative;
  flex: 1 1 0;
  min-width: 0;
  border-radius: 0.65rem;
  background: rgba(247, 249, 251, 0.92);
  -webkit-mask-image: linear-gradient(to bottom, transparent 0%, #000 14%, #000 86%, transparent 100%);
  mask-image: linear-gradient(to bottom, transparent 0%, #000 14%, #000 86%, transparent 100%);
}

.scf-tw-mask--hour {
  max-width: 5rem;
}

.scf-tw-mask--min {
  max-width: 4.35rem;
}

.scf-tw-scroll {
  height: 220px;
  overflow-y: auto;
  scroll-snap-type: y mandatory;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.scf-tw-scroll::-webkit-scrollbar {
  width: 0;
  height: 0;
}

.scf-tw-spacer {
  height: 88px;
  flex-shrink: 0;
}

.scf-tw-cell {
  height: 44px;
  scroll-snap-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.0625rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: #9499ae;
  cursor: pointer;
  user-select: none;
  transition:
    color 0.12s,
    font-weight 0.12s,
    transform 0.12s;
}

.scf-tw-cell.is-sel {
  color: var(--scf-primary-strong);
  font-weight: 800;
  font-size: 1.125rem;
}

.scf-tw-colon {
  align-self: center;
  font-weight: 800;
  font-size: 1.25rem;
  color: var(--scf-primary-strong);
  padding: 0 0.05rem;
  line-height: 1;
}

.scf-tw-summary {
  display: flex;
  gap: 0.65rem;
  padding: 0.65rem;
  border-radius: 0.75rem;
  background: rgba(242, 244, 246, 0.98);
}

.scf-tw-sum-half {
  flex: 1;
  min-width: 0;
  padding: 0.55rem 0.6rem;
  border: none;
  border-radius: 0.55rem;
  background: transparent;
  cursor: pointer;
  text-align: center;
  font-family: inherit;
  transition:
    background 0.15s,
    box-shadow 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.scf-tw-sum-half:hover {
  background: rgba(255, 255, 255, 0.55);
}

.scf-tw-sum-half.is-active {
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 4px 22px rgba(25, 28, 30, 0.06);
}

.scf-tw-sum-half:focus-visible {
  outline: 2px solid rgba(0, 102, 255, 0.35);
  outline-offset: 2px;
}

.scf-tw-sum-lbl {
  display: block;
  font-size: 11px;
  font-weight: 600;
  color: #727687;
  margin-bottom: 0.25rem;
}

.scf-tw-sum-val {
  display: block;
  font-size: 1rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: #191c1e;
}

.scf-tw-confirm {
  width: 100%;
  margin: 0;
  height: 3rem;
  border-radius: 0.75rem;
  font-weight: 700;
  font-size: 1rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: none;
  box-shadow: 0 8px 24px rgba(0, 102, 255, 0.22);
}

.scf-tw-check {
  width: 1.4rem;
  height: 1.4rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.28);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.scf-tw-check-ico {
  font-size: 1rem !important;
}
</style>

<style>
/* Teleport 遮罩需全局类名（modal-class） */
.scf-tw-overlay.el-overlay {
  background-color: rgba(15, 23, 42, 0.42);
  backdrop-filter: blur(26px);
  -webkit-backdrop-filter: blur(26px);
}

.scf-tw-dialog.el-dialog {
  padding: 0;
  border-radius: 1rem;
  overflow: hidden;
  box-shadow: 0 28px 56px rgba(25, 28, 30, 0.08);
}

.scf-tw-dialog .el-dialog__header {
  padding: 1rem 1rem 0.25rem;
  margin-right: 0;
}

.scf-tw-dialog .el-dialog__title {
  font-family:
    'Plus Jakarta Sans',
    'Noto Sans SC',
    system-ui,
    sans-serif;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.scf-tw-dialog .el-dialog__body {
  padding: 0 1rem 1.15rem;
}

@media (max-width: 420px) {
  .scf-tw-dialog.el-dialog {
    width: calc(100vw - 1.5rem) !important;
    max-width: 22rem;
    margin-left: auto;
    margin-right: auto;
  }
}
</style>
