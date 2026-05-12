<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import LobbyTabBar from '@/components/lobby/LobbyTabBar.vue'

interface RunningScheme {
  id: string
  lotteryName: string
  schemeName: string
  statusLabel: string
  turnover: string
  countdown: string
  pnl: string
  runTime: string
  lookbackPnl: string
  multiplier: string
  simBet: boolean
}

const router = useRouter()

const totalStopLoss = ref('0')
const totalTakeProfit = ref('0')
const planMultiplier = ref('1')
const breakPeriodStop = ref(false)
const lookbackSummary = ref('无')

const runningSchemes = ref<RunningScheme[]>([
  {
    id: 's1',
    lotteryName: '美国数据分分彩',
    schemeName: '漠北万位',
    statusLabel: '等待开启',
    turnover: '0.0',
    countdown: '00:07',
    pnl: '0.0',
    runTime: '00:00:00',
    lookbackPnl: '0.0',
    multiplier: '1',
    simBet: false,
  },
  {
    id: 's2',
    lotteryName: '腾讯分分彩',
    schemeName: '刚好',
    statusLabel: '等待开启',
    turnover: '0.0',
    countdown: '00:07',
    pnl: '0.0',
    runTime: '00:00:00',
    lookbackPnl: '0.0',
    multiplier: '1',
    simBet: true,
  },
])

const lookbackDialogVisible = ref(false)

/** 回头设置（与运营中心最新原型对齐，接口对接时可改为服务端 DTO） */
const lookback = reactive({
  runMode: 'prod' as 'sim' | 'prod',
  /** 个别判断 | 整体判断（互斥） */
  judgment: 'individual' as 'individual' | 'overall',
  /** —— 个别判断：单方案盈亏回头 —— */
  singleProfitThreshold: '100.00',
  singleLossThreshold: '0.00',
  /** —— 整体判断 —— */
  overallProfitThreshold: '',
  overallLossThreshold: '',
  schemeWinsMin: '',
  schemeWinsMax: '',
  periodProfit: '',
  periodLoss: '',
})

function openLookbackDialog() {
  lookbackDialogVisible.value = true
}

function cancelLookback() {
  lookbackDialogVisible.value = false
}

function confirmLookback() {
  const parts: string[] = []
  parts.push(lookback.runMode === 'sim' ? '模拟运行' : '正式运行')
  if (lookback.judgment === 'individual') {
    parts.push('个别判断')
    parts.push(
      `单方案盈亏(${lookback.singleProfitThreshold || '—'}/${lookback.singleLossThreshold || '—'})`
    )
  } else {
    parts.push('整体判断')
    parts.push(
      `整体盈亏(${lookback.overallProfitThreshold || '—'}/${lookback.overallLossThreshold || '—'})`
    )
    parts.push(`方案几(${lookback.schemeWinsMin || '—'}~${lookback.schemeWinsMax || '—'}次)`)
    parts.push(`单期盈亏(${lookback.periodProfit || '—'}/${lookback.periodLoss || '—'})`)
  }
  lookbackSummary.value = parts.join(' · ')
  lookbackDialogVisible.value = false
  ElMessage.success('已保存回头设置')
}

function onHeaderSearch() {
  ElMessage.info('方案搜索：后续对接接口')
}

function onHeaderAdd() {
  ElMessage.info('新增方案：后续对接接口')
}

function enableAllSchemes() {
  ElMessage.success('已请求一键开启方案（演示）')
}

function openBetRecords() {
  void router.push({ name: 'bet-records' })
}

function startScheme(s: RunningScheme) {
  ElMessage.success(`已请求开启：${s.schemeName}`)
}

function removeScheme(id: string) {
  runningSchemes.value = runningSchemes.value.filter((x) => x.id !== id)
  ElMessage.success('已从列表移除（演示）')
}

const schemeCount = computed(() => runningSchemes.value.length)
</script>

<template>
  <div class="cc" data-page="cloud-center">
    <header class="cc-head" role="banner">
      <div class="cc-head-top">
        <h1 class="cc-title">云端中心</h1>
        <div class="cc-head-actions">
          <button type="button" class="cc-icon-btn" aria-label="搜索方案" @click="onHeaderSearch">
            <span class="cc-ms" aria-hidden="true">search</span>
          </button>
          <button type="button" class="cc-icon-btn" aria-label="新增方案" @click="onHeaderAdd">
            <span class="cc-ms" aria-hidden="true">add_circle</span>
          </button>
        </div>
      </div>

      <div class="cc-stats">
        <div class="cc-stat-col">
          <h3 class="cc-stat-h">正式运行</h3>
          <div class="cc-stat-rows">
            <div class="cc-stat-row">
              <span>总投注</span>
              <span>0</span>
            </div>
            <div class="cc-stat-row">
              <span>总盈亏</span>
              <span class="cc-stat-em">0</span>
            </div>
            <div class="cc-stat-row cc-stat-row--pill">
              <span>运行中盈亏</span>
              <span>0</span>
            </div>
          </div>
        </div>
        <div class="cc-stat-divider" aria-hidden="true" />
        <div class="cc-stat-col">
          <h3 class="cc-stat-h">模拟运行</h3>
          <div class="cc-stat-rows">
            <div class="cc-stat-row">
              <span>总投注</span>
              <span>0</span>
            </div>
            <div class="cc-stat-row">
              <span>总盈亏</span>
              <span class="cc-stat-em">0</span>
            </div>
            <div class="cc-stat-row cc-stat-row--pill">
              <span>运行中盈亏</span>
              <span>0</span>
            </div>
          </div>
        </div>
      </div>
    </header>

    <main class="cc-main">
      <section class="cc-panel">
        <div class="cc-panel-grid">
          <div class="cc-field">
            <label class="cc-lbl">
              总止损
              <span class="cc-ms cc-lbl-ico" aria-hidden="true">info</span>
            </label>
            <el-input v-model="totalStopLoss" type="number" size="large" class="cc-el-inp" />
          </div>
          <div class="cc-field">
            <label class="cc-lbl">
              总止盈
              <span class="cc-ms cc-lbl-ico" aria-hidden="true">trending_up</span>
            </label>
            <el-input v-model="totalTakeProfit" type="number" size="large" class="cc-el-inp" />
          </div>
        </div>

        <div class="cc-field">
          <label class="cc-lbl">方案倍数系数</label>
          <div class="cc-mult-wrap">
            <div class="cc-mult-prefix" aria-hidden="true">乘</div>
            <el-input v-model="planMultiplier" type="number" size="large" class="cc-el-inp cc-el-inp--grow" />
          </div>
        </div>

        <div class="cc-row-between">
          <div class="cc-hint">
            <span class="cc-ms cc-hint-ico" aria-hidden="true">history</span>
            <span class="cc-hint-txt">目前回头设置：{{ lookbackSummary }}</span>
          </div>
          <div class="cc-switch-row">
            <span class="cc-switch-lbl">断期停投</span>
            <el-switch v-model="breakPeriodStop" />
          </div>
        </div>

        <div class="cc-actions">
          <el-button type="primary" size="large" round class="cc-btn cc-btn--primary" @click="enableAllSchemes">
            一键开启方案
          </el-button>
          <el-button size="large" round class="cc-btn cc-btn--outline" @click="openLookbackDialog">
            回头设置
          </el-button>
          <el-button size="large" round class="cc-btn cc-btn--outline" @click="openBetRecords">
            投注记录
          </el-button>
        </div>
      </section>

      <section class="cc-list-sec">
        <div class="cc-list-head">
          <h2 class="cc-list-h2">运行中方案</h2>
          <span class="cc-list-meta">共 {{ schemeCount }} 个方案</span>
        </div>

        <div v-for="s in runningSchemes" :key="s.id" class="cc-card">
          <div class="cc-card-hd">
            <div class="cc-card-title-row">
              <h3 class="cc-card-h3">{{ s.lotteryName }}</h3>
              <button type="button" class="cc-fav" aria-label="收藏">
                <span class="cc-ms cc-fav-ico" aria-hidden="true">favorite</span>
              </button>
            </div>
            <span class="cc-badge">{{ s.statusLabel }}</span>
          </div>

          <div class="cc-kv-grid">
            <div class="cc-kv">
              <span class="cc-k">方案名称</span>
              <span class="cc-v">{{ s.schemeName }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">投注流水</span>
              <span class="cc-v">{{ s.turnover }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">倒计时</span>
              <span class="cc-v cc-v--primary">{{ s.countdown }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">盈亏</span>
              <span class="cc-v cc-v--error">{{ s.pnl }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">运行时间</span>
              <span class="cc-v">{{ s.runTime }}</span>
            </div>
            <div class="cc-kv">
              <span class="cc-k">回头盈亏</span>
              <span class="cc-v cc-v--error">{{ s.lookbackPnl }}</span>
            </div>
          </div>

          <div class="cc-mult-inline">
            <span class="cc-mult-lbl">倍数系数：</span>
            <el-input v-model="s.multiplier" type="number" size="small" class="cc-el-inp cc-el-inp--w" />
          </div>

          <div class="cc-card-foot">
            <div class="cc-foot-left">
              <el-button type="primary" round class="cc-start-btn" @click="startScheme(s)">开启方案</el-button>
              <el-button class="cc-del-btn" round @click="removeScheme(s.id)" aria-label="删除">
                <span class="cc-ms cc-ms--sm" aria-hidden="true">delete</span>
              </el-button>
            </div>
            <div class="cc-foot-right">
              <span class="cc-sim-lbl">模拟投注</span>
              <el-switch v-model="s.simBet" />
            </div>
          </div>
        </div>
      </section>
    </main>

    <LobbyTabBar />

    <el-dialog
      v-model="lookbackDialogVisible"
      class="cc-lookback-dialog"
      width="min(32rem, 92vw)"
      align-center
      destroy-on-close
      :show-close="false"
      append-to-body
    >
      <template #header>
        <div class="lb-dlg-head">
          <h2 class="lb-dlg-title">回头设置</h2>
          <button type="button" class="lb-dlg-close" aria-label="关闭" @click="cancelLookback">
            <span class="cc-ms lb-dlg-close-ico" aria-hidden="true">close</span>
          </button>
        </div>
      </template>

      <div class="lb-body">
        <!-- 运行模式选择 -->
        <section class="lb-section">
          <div class="lb-section-head">
            <span class="cc-ms lb-section-ico" aria-hidden="true">play_arrow</span>
            <span class="lb-section-title">运行模式选择</span>
          </div>
          <div class="lb-run-grid" role="radiogroup" aria-label="运行模式">
            <label class="lb-run-opt" :class="{ 'is-active': lookback.runMode === 'sim' }">
              <input v-model="lookback.runMode" type="radio" class="lb-sr-only" value="sim" />
              <span class="lb-run-card">模拟运行</span>
            </label>
            <label class="lb-run-opt" :class="{ 'is-active': lookback.runMode === 'prod' }">
              <input v-model="lookback.runMode" type="radio" class="lb-sr-only" value="prod" />
              <span class="lb-run-card">正式运行</span>
            </label>
          </div>
        </section>

        <!-- 回头条件逻辑配置 -->
        <section class="lb-section lb-section--logic">
          <div class="lb-section-head">
            <span class="cc-ms lb-section-ico" aria-hidden="true">settings_suggest</span>
            <span class="lb-section-title">回头条件逻辑配置</span>
          </div>

          <el-radio-group v-model="lookback.judgment" class="lb-judgment-rg">
            <!-- 个别判断 -->
            <div class="lb-judge-section">
              <el-radio value="individual" size="large" class="lb-judge-top-radio">
                <span class="lb-judge-label-txt">个别判断</span>
              </el-radio>
              <div
                class="lb-judge-indent"
                :class="{ 'lb-judge-panel--inactive': lookback.judgment !== 'individual' }"
              >
                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">单方案盈亏回头</div>
                  <div class="lb-row2">
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-sp">盈利阈值</label>
                      <el-input
                        id="lb-sp"
                        v-model="lookback.singleProfitThreshold"
                        type="number"
                        size="small"
                        class="lb-inp"
                        :disabled="lookback.judgment !== 'individual'"
                      />
                    </div>
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-sl">亏损阈值</label>
                      <el-input
                        id="lb-sl"
                        v-model="lookback.singleLossThreshold"
                        type="number"
                        size="small"
                        class="lb-inp"
                        placeholder="0.00"
                        :disabled="lookback.judgment !== 'individual'"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 整体判断 -->
            <div class="lb-judge-section">
              <el-radio value="overall" size="large" class="lb-judge-top-radio">
                <span class="lb-judge-label-txt">整体判断</span>
              </el-radio>
              <div
                class="lb-judge-indent"
                :class="{ 'lb-judge-panel--inactive': lookback.judgment !== 'overall' }"
              >
                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">整体盈亏回头</div>
                  <div class="lb-row2">
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-op">盈利阈值</label>
                      <el-input
                        id="lb-op"
                        v-model="lookback.overallProfitThreshold"
                        type="number"
                        size="small"
                        class="lb-inp"
                        placeholder="盈利阈值"
                        :disabled="lookback.judgment !== 'overall'"
                      />
                    </div>
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-ol">亏损阈值</label>
                      <el-input
                        id="lb-ol"
                        v-model="lookback.overallLossThreshold"
                        type="number"
                        size="small"
                        class="lb-inp"
                        placeholder="亏损阈值"
                        :disabled="lookback.judgment !== 'overall'"
                      />
                    </div>
                  </div>
                </div>

                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">方案中几回头</div>
                  <div class="lb-wins-inline">
                    <span class="lb-wins-txt lb-wins-op">&gt;=</span>
                    <el-input
                      v-model="lookback.schemeWinsMin"
                      type="number"
                      size="small"
                      class="lb-inp lb-inp--wins"
                      placeholder="最小"
                      :disabled="lookback.judgment !== 'overall'"
                    />
                    <span class="lb-wins-txt lb-wins-op">&lt;=</span>
                    <el-input
                      v-model="lookback.schemeWinsMax"
                      type="number"
                      size="small"
                      class="lb-inp lb-inp--wins"
                      placeholder="最大"
                      :disabled="lookback.judgment !== 'overall'"
                    />
                    <span class="lb-wins-txt">次即回头</span>
                  </div>
                </div>

                <div class="lb-logic-card">
                  <div class="lb-logic-card-h">单期盈亏回头</div>
                  <div class="lb-row2">
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-pp">盈利</label>
                      <el-input
                        id="lb-pp"
                        v-model="lookback.periodProfit"
                        type="number"
                        size="small"
                        class="lb-inp"
                        placeholder="0.00"
                        :disabled="lookback.judgment !== 'overall'"
                      />
                    </div>
                    <div class="lb-cell">
                      <label class="lb-field-lbl" for="lb-ploss">亏损</label>
                      <el-input
                        id="lb-ploss"
                        v-model="lookback.periodLoss"
                        type="number"
                        size="small"
                        class="lb-inp"
                        placeholder="0.00"
                        :disabled="lookback.judgment !== 'overall'"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </el-radio-group>
        </section>

        <div class="lb-alert" role="alert">
          <span class="cc-ms lb-alert-ico" aria-hidden="true">error_outline</span>
          <p class="lb-alert-txt">
            注意：配置变更将重置相关方案的所有递增步长。请确保阈值设置合理，避免频繁重置影响最终收益。
          </p>
        </div>
      </div>

      <template #footer>
        <div class="lb-footer">
          <button type="button" class="lb-footer-cancel" @click="cancelLookback">取消</button>
          <el-button type="primary" class="lb-footer-save" size="large" round @click="confirmLookback">
            确认并保存设置
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.cc {
  --cc-primary: #0050cb;
  --cc-primary-strong: #0066ff;
  --cc-surface: #f7f9fb;
  --cc-card: #ffffff;
  --cc-on: #191c1e;
  --cc-on-var: #424656;
  --cc-container: #f1f5f9;
  --cc-variant: #f8fafc;
  --cc-outline: rgba(226, 232, 240, 0.85);
  --cc-error: #ba1a1a;
  min-height: 100dvh;
  background: var(--cc-surface);
  color: var(--cc-on);
  font-family: Inter, 'Noto Sans SC', system-ui, sans-serif;
  padding-bottom: calc(5.5rem + env(safe-area-inset-bottom));
  -webkit-font-smoothing: antialiased;
}

.cc-ms {
  font-family: 'Material Symbols Outlined', sans-serif;
  font-size: 1.375rem;
  line-height: 1;
  font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24;
  display: inline-block;
  user-select: none;
}

.cc-ms--sm {
  font-size: 1.125rem;
}

/* ===== Header ===== */
.cc-head {
  background: linear-gradient(180deg, var(--cc-primary-strong) 0%, var(--cc-primary) 100%);
  color: #fff;
  padding: max(1.75rem, env(safe-area-inset-top)) 1.25rem 3.75rem;
  border-radius: 0 0 2rem 2rem;
  box-shadow: 0 20px 40px -24px rgba(0, 80, 203, 0.45);
}

@media (min-width: 640px) {
  .cc-head {
    border-radius: 0 0 2.5rem 2.5rem;
  }
}

.cc-head-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.cc-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.cc-head-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-icon-btn {
  width: 2.25rem;
  height: 2.25rem;
  border: none;
  border-radius: 0.75rem;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.cc-icon-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.cc-icon-btn .cc-ms {
  font-size: 1.375rem;
}

.cc-stats {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: stretch;
  gap: 0;
}

.cc-stat-col {
  padding: 0 0.35rem;
  text-align: center;
}

.cc-stat-h {
  margin: 0 0 0.75rem;
  font-size: 0.8125rem;
  font-weight: 700;
}

.cc-stat-rows {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.cc-stat-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.6875rem;
  opacity: 0.92;
  padding: 0 0.35rem;
}

.cc-stat-em {
  font-weight: 700;
  opacity: 1;
}

.cc-stat-row--pill {
  font-weight: 700;
  opacity: 1;
  background: rgba(255, 255, 255, 0.12);
  border-radius: 0.5rem;
  padding: 0.4rem 0.45rem;
  margin-top: 0.15rem;
}

.cc-stat-divider {
  width: 1px;
  align-self: stretch;
  min-height: 6rem;
  background: linear-gradient(
    180deg,
    transparent,
    rgba(255, 255, 255, 0.22) 15%,
    rgba(255, 255, 255, 0.22) 85%,
    transparent
  );
}

/* ===== Main ===== */
.cc-main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 0 1.15rem 2rem;
  margin-top: -1.75rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.cc-panel {
  background: var(--cc-card);
  border-radius: 1.25rem;
  padding: 1.25rem;
  box-shadow: 0 24px 48px -28px rgba(15, 23, 42, 0.12), 0 4px 16px -8px rgba(15, 23, 42, 0.06);
  display: flex;
  flex-direction: column;
  gap: 1.15rem;
}

.cc-panel-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.cc-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 0;
}

.cc-lbl {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--cc-on-var);
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
}

.cc-lbl-ico {
  font-size: 0.875rem;
  opacity: 0.85;
}

.cc-mult-wrap {
  display: flex;
  align-items: stretch;
  gap: 0.75rem;
}

.cc-mult-prefix {
  flex-shrink: 0;
  min-width: 2.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 0.875rem;
  font-weight: 700;
  border-radius: 0.75rem;
  font-family: 'Noto Sans SC', sans-serif;
}

.cc-el-inp--grow {
  flex: 1;
  min-width: 0;
}

.cc-el-inp--w {
  width: 5rem;
  max-width: 40vw;
}

.cc-row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.cc-hint {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  color: var(--cc-on-var);
  min-width: 0;
}

.cc-hint-ico {
  font-size: 1.125rem;
  color: var(--cc-primary-strong);
}

.cc-hint-txt {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cc-switch-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-switch-lbl {
  font-size: 0.75rem;
  color: var(--cc-on-var);
}

.cc-actions {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.65rem;
}

@media (max-width: 380px) {
  .cc-actions {
    grid-template-columns: 1fr;
  }
}

.cc-btn {
  font-size: 0.6875rem;
  font-weight: 700;
  padding: 0.65rem 0.5rem;
  margin: 0;
  width: 100%;
  border: none;
}

.cc-btn--primary {
  box-shadow: 0 8px 20px -8px rgba(0, 80, 203, 0.45);
}

.cc-btn--outline {
  background: #fff;
  color: var(--el-color-primary);
  border: 1px solid rgba(0, 80, 203, 0.35);
}

.cc-btn--outline:hover {
  background: rgba(0, 102, 255, 0.06);
}

/* ===== List ===== */
.cc-list-sec {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cc-list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 0 0.15rem;
}

.cc-list-h2 {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: -0.01em;
}

.cc-list-meta {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--cc-on-var);
}

.cc-card {
  background: var(--cc-card);
  border-radius: 1.25rem;
  padding: 1.25rem;
  box-shadow: 0 12px 32px -20px rgba(15, 23, 42, 0.14);
}

.cc-card-hd {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1.15rem;
}

.cc-card-title-row {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
}

.cc-card-h3 {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
}

.cc-fav {
  border: none;
  background: transparent;
  padding: 0;
  line-height: 0;
  cursor: pointer;
  color: #cbd5e1;
}

.cc-fav:hover {
  color: #94a3b8;
}

.cc-fav-ico {
  font-size: 1.25rem;
}

.cc-badge {
  flex-shrink: 0;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--cc-on-var);
  background: var(--cc-variant);
  padding: 0.2rem 0.45rem;
  border-radius: 0.35rem;
}

.cc-kv-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.65rem 1.25rem;
}

.cc-kv {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  padding-bottom: 0.35rem;
  border-bottom: 1px solid rgba(226, 232, 240, 0.65);
}

.cc-k {
  font-size: 0.75rem;
  color: var(--cc-on-var);
}

.cc-v {
  font-size: 0.75rem;
  font-weight: 600;
  text-align: right;
  min-width: 0;
}

.cc-v--primary {
  color: var(--cc-primary-strong);
  font-weight: 800;
}

.cc-v--error {
  color: var(--cc-error);
  font-weight: 800;
}

.cc-mult-inline {
  margin-top: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-mult-lbl {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--cc-on-var);
}

.cc-card-foot {
  margin-top: 1.25rem;
  padding-top: 1.1rem;
  border-top: 1px solid rgba(226, 232, 240, 0.65);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.cc-foot-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.cc-start-btn {
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.55rem 1.1rem;
}

.cc-del-btn {
  padding: 0.5rem 0.65rem;
  min-height: auto;
  color: var(--cc-on-var);
  background: var(--cc-variant);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

.cc-foot-right {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.cc-sim-lbl {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--cc-on-var);
}
</style>

<style>
/* 对话框：覆盖 Element Plus，对齐「数字精算主义」与运营中心回头弹窗设计 */
.cc-lookback-dialog.el-dialog {
  border-radius: 1.5rem;
  overflow: hidden;
  padding: 0;
  box-shadow: 0 20px 50px rgba(0, 80, 203, 0.15);
}

.cc-lookback-dialog .el-dialog__header {
  margin: 0;
  padding: 0;
  border-bottom: none;
}

.cc-lookback-dialog .el-dialog__body {
  padding: 0 1.5rem 1.25rem;
  max-height: min(75dvh, 34rem);
  overflow-y: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.cc-lookback-dialog .el-dialog__body::-webkit-scrollbar {
  display: none;
}

.cc-lookback-dialog .el-dialog__footer {
  margin: 0;
  padding: 0;
  border-top: none;
}

.lb-dlg-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem 1.5rem 0.75rem;
}

.lb-dlg-title {
  margin: 0;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #191c1e;
}

.lb-dlg-close {
  width: 2rem;
  height: 2rem;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: #727687;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.lb-dlg-close:hover {
  background: #e6e8ea;
}

.lb-dlg-close-ico {
  font-size: 1.25rem;
}

.lb-body {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.lb-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.lb-section-head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.lb-section-ico {
  font-size: 1.25rem;
  color: #0050cb;
}

.lb-section-title {
  font-size: 0.875rem;
  font-weight: 700;
  color: #191c1e;
}

.lb-section--logic {
  gap: 1rem;
}

.lb-judgment-rg.el-radio-group {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 1.25rem;
  width: 100%;
}

.lb-judge-section {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.lb-judge-top-radio.el-radio {
  height: auto;
  margin-right: 0;
  align-items: center;
}

.lb-judge-top-radio .el-radio__label {
  padding-left: 0.5rem;
}

.lb-judge-label-txt {
  font-size: 0.9375rem;
  font-weight: 700;
  color: #191c1e;
  line-height: 1.3;
}

.lb-judge-indent {
  padding-left: 1.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  transition:
    opacity 0.25s ease,
    filter 0.25s ease;
}

.lb-judge-panel--inactive {
  opacity: 0.4;
  pointer-events: none;
  filter: grayscale(0.45);
}

.lb-wins-inline {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  transition: opacity 0.15s;
}

.lb-wins-txt {
  font-size: 0.75rem;
  color: #424656;
  flex-shrink: 0;
}

.lb-wins-op {
  font-family: Inter, ui-monospace, monospace;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.lb-inp--wins.el-input {
  width: 4.25rem;
  min-width: 3.25rem;
  flex-shrink: 0;
}

.lb-inp--wins .el-input__wrapper {
  justify-content: center;
}

.lb-run-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.lb-run-opt {
  cursor: pointer;
  margin: 0;
}

.lb-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.lb-run-card {
  display: block;
  text-align: center;
  padding: 0.75rem 0.5rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.85);
  background: #f2f4f6;
  font-size: 0.875rem;
  font-weight: 600;
  color: #424656;
  transition:
    border-color 0.15s,
    background 0.15s,
    color 0.15s;
}

.lb-run-opt.is-active .lb-run-card,
.lb-run-opt:focus-within .lb-run-card {
  border-color: #0050cb;
  background: rgba(0, 102, 255, 0.08);
  color: #0050cb;
}

.lb-logic-card {
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(194, 198, 216, 0.55);
  background: rgba(242, 244, 246, 0.45);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.lb-logic-card-h {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: #191c1e;
  line-height: 1.35;
}

.lb-row2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  transition: opacity 0.15s;
}

.lb-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.lb-field-lbl {
  font-size: 0.625rem;
  font-weight: 700;
  color: #727687;
  letter-spacing: 0.02em;
  padding-left: 0.15rem;
}

.lb-inp.el-input {
  width: 100%;
}

.lb-alert {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  padding: 0.75rem;
  border-radius: 0.75rem;
  background: rgba(255, 218, 214, 0.22);
  border: 1px solid rgba(186, 26, 26, 0.14);
}

.lb-alert-ico {
  font-size: 1.125rem;
  color: #ba1a1a;
  flex-shrink: 0;
  margin-top: 0.05rem;
}

.lb-alert-txt {
  margin: 0;
  font-size: 0.6875rem;
  line-height: 1.65;
  font-weight: 500;
  color: #424656;
}

.lb-footer {
  display: flex;
  align-items: stretch;
  gap: 0.75rem;
  width: 100%;
  padding: 1rem 1.5rem calc(1rem + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid rgba(194, 198, 216, 0.25);
}

.lb-footer-cancel {
  flex: 1;
  border: none;
  border-radius: 0.75rem;
  background: transparent;
  font-size: 0.875rem;
  font-weight: 700;
  color: #424656;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s;
}

.lb-footer-cancel:hover {
  background: #f2f4f6;
}

.lb-footer-save {
  flex: 2;
  margin: 0;
  border: none;
  font-weight: 700;
  box-shadow: 0 8px 16px rgba(0, 80, 203, 0.25);
  background: linear-gradient(180deg, #0066ff 0%, #0050cb 100%);
}
</style>
