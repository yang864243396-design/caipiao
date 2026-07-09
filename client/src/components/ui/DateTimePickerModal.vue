<script setup lang="ts">
import { computed, ref, watch } from 'vue'

/**
 * 日期 + 时间两步选择弹窗（屏幕居中）。
 * 流程：先选日期（日历）→ 再选时间（时/分/秒）→ 确认输出 "YYYY-MM-DD HH:mm:ss"。
 * 样式遵循 client/DESIGN.md「数字精算主义」。
 */

const props = defineProps<{
  modelValue: boolean
  value?: string
  title?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'confirm', v: string): void
}>()

const WEEK_LABELS = ['日', '一', '二', '三', '四', '五', '六']

const step = ref<'date' | 'time'>('date')

// 当前面板年月
const viewYear = ref(2026)
const viewMonth = ref(0) // 0-based
// 选择结果
const selYear = ref(2026)
const selMonth = ref(0)
const selDay = ref(1)
const selHour = ref(0)
const selMin = ref(0)
const selSec = ref(0)

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function parseValue(raw: string | undefined): Date {
  if (raw) {
    const m = raw.trim().match(/^(\d{4})-(\d{1,2})-(\d{1,2})(?:[ T](\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?/)
    if (m) {
      return new Date(
        Number(m[1]),
        Number(m[2]) - 1,
        Number(m[3]),
        Number(m[4] ?? 0),
        Number(m[5] ?? 0),
        Number(m[6] ?? 0),
      )
    }
  }
  return new Date()
}

function syncFromValue(): void {
  const d = parseValue(props.value)
  selYear.value = d.getFullYear()
  selMonth.value = d.getMonth()
  selDay.value = d.getDate()
  selHour.value = d.getHours()
  selMin.value = d.getMinutes()
  selSec.value = d.getSeconds()
  viewYear.value = selYear.value
  viewMonth.value = selMonth.value
  step.value = 'date'
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) syncFromValue()
  },
  { immediate: true },
)

const monthTitle = computed(() => `${viewYear.value} 年 ${viewMonth.value + 1} 月`)

interface Cell {
  day: number
  inMonth: boolean
  y: number
  m: number
}

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
function isToday(c: Cell): boolean {
  return (
    c.inMonth &&
    c.y === today.getFullYear() &&
    c.m === today.getMonth() &&
    c.day === today.getDate()
  )
}
function isSelected(c: Cell): boolean {
  return c.inMonth && c.y === selYear.value && c.m === selMonth.value && c.day === selDay.value
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
function prevYear(): void {
  viewYear.value -= 1
}
function nextYear(): void {
  viewYear.value += 1
}

function pickDay(c: Cell): void {
  if (!c.inMonth) {
    // 跨月点击则跳转该月
    viewYear.value = c.m < 0 ? viewYear.value - 1 : c.m > 11 ? viewYear.value + 1 : c.y
    viewMonth.value = (c.m + 12) % 12
  }
  selYear.value = viewYear.value
  selMonth.value = viewMonth.value
  selDay.value = c.day
  step.value = 'time'
}

const hours = Array.from({ length: 24 }, (_, i) => i)
const minutes = Array.from({ length: 60 }, (_, i) => i)
const seconds = Array.from({ length: 60 }, (_, i) => i)

const dateSummary = computed(
  () => `${selYear.value}-${pad(selMonth.value + 1)}-${pad(selDay.value)}`,
)

function close(): void {
  emit('update:modelValue', false)
}

function backToDate(): void {
  step.value = 'date'
}

function confirm(): void {
  const out = `${dateSummary.value} ${pad(selHour.value)}:${pad(selMin.value)}:${pad(selSec.value)}`
  emit('confirm', out)
  emit('update:modelValue', false)
}
</script>

<template>
  <Teleport to="body">
    <transition name="dtp-fade">
      <div v-if="modelValue" class="dtp-overlay" @click.self="close">
        <div class="dtp-card" role="dialog" aria-modal="true">
          <header class="dtp-head">
            <span class="dtp-title">{{ title || (step === 'date' ? '选择日期' : '选择时间') }}</span>
            <button type="button" class="dtp-close" aria-label="关闭" @click="close">
              <span class="material-sym">close</span>
            </button>
          </header>

          <!-- 日期步骤 -->
          <div v-if="step === 'date'" class="dtp-body">
            <div class="dtp-cal-nav">
              <button type="button" class="dtp-nav-btn" aria-label="上一年" @click="prevYear">«</button>
              <button type="button" class="dtp-nav-btn" aria-label="上个月" @click="prevMonth">‹</button>
              <span class="dtp-month-title">{{ monthTitle }}</span>
              <button type="button" class="dtp-nav-btn" aria-label="下个月" @click="nextMonth">›</button>
              <button type="button" class="dtp-nav-btn" aria-label="下一年" @click="nextYear">»</button>
            </div>
            <div class="dtp-week-row">
              <span v-for="w in WEEK_LABELS" :key="w" class="dtp-week-cell">{{ w }}</span>
            </div>
            <div class="dtp-day-grid">
              <button
                v-for="(c, i) in cells"
                :key="i"
                type="button"
                class="dtp-day"
                :class="{
                  'is-out': !c.inMonth,
                  'is-today': isToday(c),
                  'is-selected': isSelected(c),
                }"
                @click="pickDay(c)"
              >
                {{ c.day }}
              </button>
            </div>
          </div>

          <!-- 时间步骤 -->
          <div v-else class="dtp-body">
            <p class="dtp-time-date">{{ dateSummary }}</p>
            <div class="dtp-time-grid">
              <div class="dtp-time-col">
                <span class="dtp-time-lbl">时</span>
                <el-select v-model="selHour" size="large" class="dtp-time-sel" popper-class="dtp-time-popper">
                  <el-option v-for="h in hours" :key="h" :label="String(h).padStart(2, '0')" :value="h" />
                </el-select>
              </div>
              <div class="dtp-time-col">
                <span class="dtp-time-lbl">分</span>
                <el-select v-model="selMin" size="large" class="dtp-time-sel" popper-class="dtp-time-popper">
                  <el-option v-for="mm in minutes" :key="mm" :label="String(mm).padStart(2, '0')" :value="mm" />
                </el-select>
              </div>
              <div class="dtp-time-col">
                <span class="dtp-time-lbl">秒</span>
                <el-select v-model="selSec" size="large" class="dtp-time-sel" popper-class="dtp-time-popper">
                  <el-option v-for="s in seconds" :key="s" :label="String(s).padStart(2, '0')" :value="s" />
                </el-select>
              </div>
            </div>
          </div>

          <footer class="dtp-foot">
            <el-button v-if="step === 'time'" class="dtp-foot-btn" @click="backToDate">上一步</el-button>
            <el-button class="dtp-foot-btn" @click="close">取消</el-button>
            <el-button
              v-if="step === 'time'"
              type="primary"
              class="dtp-foot-btn"
              @click="confirm"
            >
              确定
            </el-button>
          </footer>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<style scoped>
.dtp-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.42);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.dtp-card {
  width: 100%;
  max-width: 22rem;
  background: #fff;
  border-radius: 1.25rem;
  overflow: hidden;
  box-shadow: 0 24px 60px rgba(15, 35, 95, 0.22);
}

.dtp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem 0.5rem;
}

.dtp-title {
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 700;
  font-size: 1.0625rem;
  color: #0f172a;
}

.dtp-close {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 10px;
  color: #94a3b8;
  background: #f1f5f9;
}

.dtp-close .material-sym {
  font-size: 1rem;
}

.dtp-body {
  padding: 0.5rem 1.1rem 0.75rem;
}

.dtp-cal-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.25rem;
  padding: 0.35rem 0.25rem 0.75rem;
}

.dtp-nav-btn {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  color: #475569;
  font-size: 1rem;
  background: transparent;
}

.dtp-nav-btn:hover {
  background: #f1f5f9;
  color: #0050cb;
}

.dtp-month-title {
  flex: 1;
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-weight: 700;
  font-size: 0.9375rem;
  color: #0f172a;
}

.dtp-week-row,
.dtp-day-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 0.15rem;
}

.dtp-week-row {
  padding-bottom: 0.4rem;
  border-bottom: 1px solid #f1f5f9;
  margin-bottom: 0.4rem;
}

.dtp-week-cell {
  text-align: center;
  font-size: 0.75rem;
  color: #94a3b8;
  padding: 0.25rem 0;
}

.dtp-day {
  aspect-ratio: 1;
  display: grid;
  place-items: center;
  border-radius: 10px;
  font-size: 0.875rem;
  color: #0f172a;
  background: transparent;
  transition: background 0.15s, color 0.15s;
}

.dtp-day:hover {
  background: var(--el-color-primary-light-9, #e6ebfa);
}

.dtp-day.is-out {
  color: #cbd5e1;
}

.dtp-day.is-today {
  color: #0050cb;
  font-weight: 700;
}

.dtp-day.is-selected {
  background: linear-gradient(145deg, #0066ff, #0050cb);
  color: #fff;
  font-weight: 700;
  box-shadow: 0 8px 18px -8px rgba(0, 80, 203, 0.5);
}

.dtp-time-date {
  margin: 0.25rem 0 1rem;
  text-align: center;
  font-weight: 700;
  font-size: 1rem;
  color: #0050cb;
}

.dtp-time-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.75rem;
  padding-bottom: 0.5rem;
}

.dtp-time-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.4rem;
}

.dtp-time-lbl {
  font-size: 0.8125rem;
  color: #64748b;
}

.dtp-time-sel {
  width: 100%;
}

.dtp-foot {
  display: flex;
  gap: 0.6rem;
  padding: 0.75rem 1.1rem 1.1rem;
}

.dtp-foot-btn {
  flex: 1;
  margin: 0;
}

.dtp-fade-enter-active,
.dtp-fade-leave-active {
  transition: opacity 0.2s ease;
}

.dtp-fade-enter-from,
.dtp-fade-leave-to {
  opacity: 0;
}
</style>

<!-- popper 挂载到 body，需高于 .dtp-overlay (z-index: 3000) -->
<style>
.dtp-time-popper.el-popper {
  z-index: 3100 !important;
}
</style>
