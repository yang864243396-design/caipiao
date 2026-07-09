<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

/**
 * H5 日期范围选择：底部弹层 + 日历点选起止日。
 * 输出 value-format 与 el-date-picker 一致：['YYYY-MM-DD', 'YYYY-MM-DD']
 */

const props = withDefaults(
  defineProps<{
    modelValue: [string, string] | null
    startPlaceholder?: string
    endPlaceholder?: string
    size?: 'large' | 'default'
    /** 最多可选连续自然日数（含起止日）；未设则不限制 */
    maxDays?: number
    /** 是否允许清空已选日期 */
    clearable?: boolean
  }>(),
  {
    startPlaceholder: '开始日期',
    endPlaceholder: '结束日期',
    size: 'large',
    maxDays: undefined,
    clearable: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: [string, string] | null): void
}>()

const WEEK_LABELS = ['日', '一', '二', '三', '四', '五', '六']

const open = ref(false)
const viewYear = ref(2026)
const viewMonth = ref(0)
const draftStart = ref<Date | null>(null)
const draftEnd = ref<Date | null>(null)
const rangeLimitWarning = ref('')

function warnMaxDaysInSheet(): void {
  if (!props.maxDays) return
  rangeLimitWarning.value = `最多查询连续 ${props.maxDays} 天，已自动截断为允许范围`
}

function warnMaxDaysToast(): void {
  if (!props.maxDays) return
  ElMessage.warning({
    message: `最多查询连续 ${props.maxDays} 天`,
    zIndex: 3500,
  })
}

function clearRangeLimitWarning(): void {
  rangeLimitWarning.value = ''
}

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function ymd(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function parseYmd(raw: string): Date | null {
  const m = raw.trim().match(/^(\d{4})-(\d{1,2})-(\d{1,2})$/)
  if (!m) return null
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
}

function dayStart(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
}

/** 起止日 inclusive 自然日数 */
function spanDays(a: Date, b: Date): number {
  const lo = Math.min(dayStart(a), dayStart(b))
  const hi = Math.max(dayStart(a), dayStart(b))
  return Math.round((hi - lo) / 86_400_000) + 1
}

function capEndToMax(start: Date, end: Date): Date {
  const max = props.maxDays
  if (!max || max < 1) return end
  if (spanDays(start, end) <= max) return end
  const capped = new Date(start)
  capped.setDate(capped.getDate() + max - 1)
  return capped
}

interface Cell {
  day: number
  inMonth: boolean
  y: number
  m: number
}

function cellDate(c: Cell): Date {
  return new Date(c.y, c.m, c.day)
}

const displayText = computed(() => {
  const v = props.modelValue
  if (!v || !v[0] || !v[1]) return ''
  return `${v[0]} — ${v[1]}`
})

const hintText = computed(() => {
  if (!draftStart.value) return '请选择开始日期'
  if (!draftEnd.value) return '请选择结束日期'
  return `${ymd(draftStart.value)} 至 ${ymd(draftEnd.value)}`
})

const canConfirm = computed(() => draftStart.value !== null && draftEnd.value !== null)

const presetOptions = computed(() => {
  const max = props.maxDays
  const all = [
    { days: 1, label: '今天' },
    { days: 2, label: '近2天' },
    { days: 3, label: '近3天' },
  ]
  if (!max) return all
  return all.filter((p) => p.days <= max)
})

const maxDaysHint = computed(() =>
  props.maxDays && props.maxDays > 0 ? `最多连续 ${props.maxDays} 天` : '',
)

const monthTitle = computed(() => `${viewYear.value} 年 ${viewMonth.value + 1} 月`)

const cells = computed<Cell[]>(() => {
  const first = new Date(viewYear.value, viewMonth.value, 1)
  const startWeekday = first.getDay()
  const daysInMonth = new Date(viewYear.value, viewMonth.value + 1, 0).getDate()
  const prevDays = new Date(viewYear.value, viewMonth.value, 0).getDate()
  const out: Cell[] = []
  for (let i = startWeekday - 1; i >= 0; i--) {
    out.push({ day: prevDays - i, inMonth: false, y: viewYear.value, m: viewMonth.value - 1 })
  }
  for (let d = 1; d <= daysInMonth; d++) {
    out.push({ day: d, inMonth: true, y: viewYear.value, m: viewMonth.value })
  }
  let next = 1
  while (out.length % 7 !== 0 || out.length < 42) {
    out.push({ day: next++, inMonth: false, y: viewYear.value, m: viewMonth.value + 1 })
    if (out.length >= 42) break
  }
  return out
})

const today = new Date()

function syncDraftFromModel(): void {
  const v = props.modelValue
  draftStart.value = v?.[0] ? parseYmd(v[0]) : null
  draftEnd.value = v?.[1] ? parseYmd(v[1]) : null
  const anchor = draftEnd.value ?? draftStart.value ?? new Date()
  viewYear.value = anchor.getFullYear()
  viewMonth.value = anchor.getMonth()
}

watch(
  () => open.value,
  (v) => {
    if (v) {
      clearRangeLimitWarning()
      syncDraftFromModel()
    }
  },
)

function isToday(c: Cell): boolean {
  const d = cellDate(c)
  return (
    c.inMonth &&
    d.getFullYear() === today.getFullYear() &&
    d.getMonth() === today.getMonth() &&
    d.getDate() === today.getDate()
  )
}

function isRangeStart(c: Cell): boolean {
  if (!draftStart.value) return false
  const d = cellDate(c)
  return dayStart(d) === dayStart(draftStart.value)
}

function isRangeEnd(c: Cell): boolean {
  if (!draftEnd.value) return false
  const d = cellDate(c)
  return dayStart(d) === dayStart(draftEnd.value)
}

function isInRange(c: Cell): boolean {
  if (!draftStart.value || !draftEnd.value) return false
  const t = dayStart(cellDate(c))
  const a = dayStart(draftStart.value)
  const b = dayStart(draftEnd.value)
  const lo = Math.min(a, b)
  const hi = Math.max(a, b)
  return t >= lo && t <= hi
}

function prevMonth(): void {
  if (viewMonth.value === 0) {
    viewMonth.value = 11
    viewYear.value -= 1
  } else {
    viewMonth.value -= 1
  }
}

function nextMonth(): void {
  if (viewMonth.value === 11) {
    viewMonth.value = 0
    viewYear.value += 1
  } else {
    viewMonth.value += 1
  }
}

function pickDay(c: Cell): void {
  if (!c.inMonth) {
    viewYear.value = c.m < 0 ? viewYear.value - 1 : c.m > 11 ? viewYear.value + 1 : c.y
    viewMonth.value = (c.m + 12) % 12
  }
  const picked = new Date(viewYear.value, viewMonth.value, c.day)

  if (!draftStart.value || (draftStart.value && draftEnd.value)) {
    draftStart.value = picked
    draftEnd.value = null
    clearRangeLimitWarning()
    return
  }

  draftEnd.value = picked
  if (dayStart(draftEnd.value) < dayStart(draftStart.value)) {
    const tmp = draftStart.value
    draftStart.value = draftEnd.value
    draftEnd.value = tmp
  }
  if (props.maxDays && draftStart.value && draftEnd.value) {
    const capped = capEndToMax(draftStart.value, draftEnd.value)
    if (dayStart(capped) !== dayStart(draftEnd.value)) {
      warnMaxDaysInSheet()
      draftEnd.value = capped
    }
  }
}

function applyPreset(days: number): void {
  const max = props.maxDays
  const span = max && max > 0 ? Math.min(days, max) : days
  const end = new Date()
  const start = new Date()
  start.setDate(end.getDate() - (span - 1))
  draftStart.value = start
  draftEnd.value = end
  clearRangeLimitWarning()
  viewYear.value = end.getFullYear()
  viewMonth.value = end.getMonth()
}

function openPicker(): void {
  open.value = true
}

function close(): void {
  open.value = false
}

function resetDraft(): void {
  draftStart.value = null
  draftEnd.value = null
  clearRangeLimitWarning()
}

function confirm(): void {
  if (!draftStart.value || !draftEnd.value) return
  let end = draftEnd.value
  if (props.maxDays && spanDays(draftStart.value, end) > props.maxDays) {
    end = capEndToMax(draftStart.value, end)
    warnMaxDaysToast()
  }
  emit('update:modelValue', [ymd(draftStart.value), ymd(end)])
  close()
}

function clearValue(): void {
  if (!props.clearable) return
  emit('update:modelValue', null)
  resetDraft()
}
</script>

<template>
  <div class="drp-field" :class="`drp-field--${size}`">
    <button type="button" class="drp-trigger" :class="{ 'is-empty': !displayText }" @click="openPicker">
      <span class="material-sym drp-trigger-ico" aria-hidden="true">calendar_month</span>
      <span class="drp-trigger-text">{{ displayText || `${startPlaceholder} — ${endPlaceholder}` }}</span>
      <span
        v-if="clearable && displayText"
        class="material-sym drp-trigger-clear"
        role="button"
        tabindex="0"
        aria-label="清除日期"
        @click.stop="clearValue"
      >
        close
      </span>
    </button>

    <Teleport to="body">
      <transition name="drp-sheet">
        <div v-if="open" class="drp-overlay" @click.self="close">
          <div class="drp-sheet" role="dialog" aria-modal="true" aria-label="选择日期范围">
            <div class="drp-sheet-handle" aria-hidden="true" />
            <header class="drp-head">
              <span class="drp-title">选择时间</span>
              <button type="button" class="drp-close" aria-label="关闭" @click="close">
                <span class="material-sym">close</span>
              </button>
            </header>

            <p class="drp-hint">
              {{ hintText }}<span v-if="maxDaysHint" class="drp-hint-limit">（{{ maxDaysHint }}）</span>
            </p>
            <p v-if="rangeLimitWarning" class="drp-range-warn" role="alert">
              {{ rangeLimitWarning }}
            </p>

            <div v-if="presetOptions.length" class="drp-presets">
              <button
                v-for="p in presetOptions"
                :key="p.days"
                type="button"
                class="drp-preset"
                @click="applyPreset(p.days)"
              >
                {{ p.label }}
              </button>
            </div>

            <div class="drp-cal-nav">
              <button type="button" class="drp-nav-btn" aria-label="上个月" @click="prevMonth">‹</button>
              <span class="drp-month-title">{{ monthTitle }}</span>
              <button type="button" class="drp-nav-btn" aria-label="下个月" @click="nextMonth">›</button>
            </div>

            <div class="drp-week-row">
              <span v-for="w in WEEK_LABELS" :key="w" class="drp-week-cell">{{ w }}</span>
            </div>
            <div class="drp-day-grid">
              <button
                v-for="(c, i) in cells"
                :key="i"
                type="button"
                class="drp-day"
                :class="{
                  'is-out': !c.inMonth,
                  'is-today': isToday(c),
                  'is-in-range': isInRange(c),
                  'is-start': isRangeStart(c),
                  'is-end': isRangeEnd(c),
                }"
                @click="pickDay(c)"
              >
                {{ c.day }}
              </button>
            </div>

            <footer class="drp-foot">
              <el-button v-if="clearable" class="drp-foot-btn" @click="resetDraft">重置</el-button>
              <el-button type="primary" class="drp-foot-btn" :disabled="!canConfirm" @click="confirm">
                确定
              </el-button>
            </footer>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<style scoped>
.drp-field {
  width: 100%;
  min-width: 0;
}

.drp-trigger {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  min-height: 40px;
  padding: 0 0.85rem;
  border: none;
  border-radius: 0.75rem;
  background: #fff;
  box-shadow: 0 8px 22px -14px rgba(15, 23, 42, 0.12);
  color: #191c1e;
  font: inherit;
  font-size: 0.875rem;
  text-align: left;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.drp-field--large .drp-trigger {
  min-height: 44px;
  font-size: 0.9375rem;
}

.drp-trigger.is-empty .drp-trigger-text {
  color: #94a3b8;
}

.drp-trigger-ico {
  flex-shrink: 0;
  font-size: 1.125rem;
  color: #94a3b8;
}

.drp-trigger-text {
  flex: 1 1 0;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.drp-trigger-clear {
  flex-shrink: 0;
  font-size: 1rem;
  color: #94a3b8;
  padding: 0.15rem;
}

.drp-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgba(15, 23, 42, 0.42);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  padding-bottom: env(safe-area-inset-bottom);
}

.drp-sheet {
  width: 100%;
  max-width: 40rem;
  max-height: min(88dvh, 640px);
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  background: #fff;
  border-radius: 1.25rem 1.25rem 0 0;
  box-shadow: 0 -12px 48px rgba(15, 35, 95, 0.18);
}

.drp-sheet-handle {
  width: 2.5rem;
  height: 4px;
  margin: 0.65rem auto 0;
  border-radius: 999px;
  background: #e2e8f0;
}

.drp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1.15rem 0.35rem;
}

.drp-title {
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 700;
  font-size: 1.0625rem;
  color: #0f172a;
}

.drp-close {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 10px;
  color: #94a3b8;
  background: #f1f5f9;
  border: none;
}

.drp-hint {
  margin: 0;
  padding: 0 1.15rem 0.65rem;
  font-size: 0.8125rem;
  color: #64748b;
  text-align: center;
}

.drp-hint-limit {
  color: #94a3b8;
}

.drp-range-warn {
  margin: 0 1.15rem 0.65rem;
  padding: 0.5rem 0.75rem;
  border-radius: 0.65rem;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: #b45309;
  text-align: center;
  background: rgba(245, 158, 11, 0.12);
}

.drp-presets {
  display: flex;
  gap: 0.5rem;
  padding: 0 1.15rem 0.75rem;
}

.drp-preset {
  flex: 1;
  padding: 0.45rem 0.5rem;
  border: none;
  border-radius: 0.65rem;
  background: #eef2f7;
  color: #424656;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 650;
}

.drp-preset:active {
  background: var(--el-color-primary-light-9, #e6ebfa);
  color: #0050cb;
}

.drp-cal-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.25rem 1rem 0.5rem;
}

.drp-nav-btn {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  color: #475569;
  font-size: 1.25rem;
  background: transparent;
}

.drp-nav-btn:active {
  background: #f1f5f9;
  color: #0050cb;
}

.drp-month-title {
  flex: 1;
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 700;
  font-size: 0.9375rem;
  color: #0f172a;
}

.drp-week-row,
.drp-day-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 0.1rem;
  padding: 0 0.85rem;
}

.drp-week-row {
  padding-bottom: 0.35rem;
  margin-bottom: 0.25rem;
}

.drp-week-cell {
  text-align: center;
  font-size: 0.75rem;
  color: #94a3b8;
  padding: 0.2rem 0;
}

.drp-day-grid {
  padding-bottom: 0.75rem;
}

.drp-day {
  aspect-ratio: 1;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 10px;
  font-size: 0.875rem;
  color: #0f172a;
  background: transparent;
  min-height: 40px;
}

.drp-day.is-out {
  color: #cbd5e1;
}

.drp-day.is-today {
  color: #0050cb;
  font-weight: 700;
}

.drp-day.is-in-range {
  background: rgba(0, 80, 203, 0.08);
  border-radius: 0;
}

.drp-day.is-start,
.drp-day.is-end {
  background: linear-gradient(145deg, #0066ff, #0050cb);
  color: #fff;
  font-weight: 700;
  border-radius: 10px;
  box-shadow: 0 6px 16px -8px rgba(0, 80, 203, 0.45);
}

.drp-foot {
  display: flex;
  gap: 0.65rem;
  padding: 0.75rem 1.15rem calc(1rem + env(safe-area-inset-bottom));
  border-top: 1px solid #f1f5f9;
}

.drp-foot-btn {
  flex: 1;
  margin: 0;
}

.drp-sheet-enter-active,
.drp-sheet-leave-active {
  transition: opacity 0.22s ease;
}

.drp-sheet-enter-active .drp-sheet,
.drp-sheet-leave-active .drp-sheet {
  transition: transform 0.28s cubic-bezier(0.32, 0.72, 0, 1);
}

.drp-sheet-enter-from,
.drp-sheet-leave-to {
  opacity: 0;
}

.drp-sheet-enter-from .drp-sheet,
.drp-sheet-leave-to .drp-sheet {
  transform: translateY(100%);
}
</style>
