<script setup lang="ts">

import { computed, ref, watch } from 'vue'

import { ElMessage } from 'element-plus'

import { fetchLotteryDraws, type LotteryDrawRow } from '@/api/games'

import { fetchSchemeBetHistory } from '@/api/schemes'

import type { SchemeInstanceRow } from '@/stores/schemeInstances'

import type { SchemeBetHistoryItem } from '@/types/schemes'



const props = defineProps<{

  modelValue: boolean

  scheme: SchemeInstanceRow | null

}>()



const emit = defineEmits<{

  'update:modelValue': [value: boolean]

}>()



const tab = ref<'bets' | 'draws'>('bets')

const historySubTab = ref(0)

const pageSize = ref(10)

const betPage = ref(1)



const historySubTabLabels = ['号码', '大小', '单双', '龙虎', '总和'] as const



const betItems = ref<SchemeBetHistoryItem[]>([])

const betsLoading = ref(false)

const drawRecords = ref<LotteryDrawRow[]>([])

const drawsLoading = ref(false)



const pagedBetItems = computed(() => slicePage(betItems.value, betPage.value))



const settledRows = computed(() => betItems.value.filter((r) => r.status === '已结算'))



const settledPnl = computed(() =>

  settledRows.value.reduce((sum, r) => sum + r.profitLoss, 0),

)



const executionWinRate = computed(() => {

  const total = settledRows.value.length

  if (total === 0) return '—'

  const wins = settledRows.value.filter((r) => r.win === true).length

  return `${Math.round((wins / total) * 1000) / 10}%`

})



async function loadBetHistory(instanceId: string) {
  betsLoading.value = true
  try {
    const result = await fetchSchemeBetHistory(instanceId, 30)
    betItems.value = result.items ?? []
  } catch (e) {
    betItems.value = []
    ElMessage.error(e instanceof Error ? e.message : '加载投注记录失败')
  } finally {
    betsLoading.value = false
  }
}

async function loadDrawHistory(lotteryCode: string) {
  drawsLoading.value = true
  try {
    const result = await fetchLotteryDraws(lotteryCode)
    drawRecords.value = result.items
  } catch (e) {
    drawRecords.value = []
    ElMessage.error(e instanceof Error ? e.message : '加载历史开奖失败')
  } finally {
    drawsLoading.value = false
  }
}

watch(
  () => [props.modelValue, props.scheme?.id, props.scheme?.lotteryCode] as const,
  ([open, instanceId, lotteryCode]) => {
    if (open && instanceId) {
      void loadBetHistory(instanceId)
      if (lotteryCode) void loadDrawHistory(lotteryCode)
    } else {
      betItems.value = []
      drawRecords.value = []
    }
  },
)

watch(

  () => props.scheme?.id,

  () => {

    tab.value = 'bets'

    historySubTab.value = 0

    betPage.value = 1

  },

)



watch(tab, () => {

  betPage.value = 1

})



function slicePage<T>(rows: T[], page: number) {

  const start = (page - 1) * pageSize.value

  return rows.slice(start, start + pageSize.value)

}



function formatBetRecordAmount(amount: string) {

  const n = Number(amount)

  if (!Number.isFinite(n)) return amount

  return String(Math.trunc(n))

}



function formatBetRecordPl(n: number) {

  return String(Math.abs(Math.trunc(n)))

}



function historyBigSmallDigit(b: string) {

  return parseInt(b, 10) >= 5 ? '大' : '小'

}



function historyParityDigit(b: string) {

  return parseInt(b, 10) % 2 === 1 ? '单' : '双'

}



function historyDragonTigerCells(digits: number[]) {

  const pairs: { label: string; a: number; b: number }[] = [

    { label: '龙', a: digits[0], b: digits[4] },

    { label: '虎', a: digits[1], b: digits[3] },

  ]

  return pairs.map(({ label, a, b }) => {

    if (a > b) return { label, char: '龙', kind: 'dragon' as const }

    if (a < b) return { label, char: '虎', kind: 'tiger' as const }

    return { label, char: '和', kind: 'tie' as const }

  })

}



function historyDigitsFromBalls(balls: string[]) {

  return balls.map((b) => parseInt(b, 10))

}



function close() {

  emit('update:modelValue', false)

}

</script>



<template>

  <el-drawer

    :model-value="modelValue"

    :title="scheme ? `方案 ${scheme.id} · 玩法详情数据` : '玩法详情数据'"

    size="min(920px, 96vw)"

    destroy-on-close

    @update:model-value="emit('update:modelValue', $event)"

  >

    <template v-if="scheme">

      <el-descriptions :column="3" border size="small" style="margin-bottom: 1rem">

        <el-descriptions-item label="会员账号">{{ scheme.memberName }}</el-descriptions-item>

        <el-descriptions-item label="彩种">{{ scheme.lotteryLabel }}</el-descriptions-item>

        <el-descriptions-item label="类型">{{ scheme.kind }}</el-descriptions-item>

        <el-descriptions-item label="投注通道">{{ scheme.simBet ? '模拟' : '正式' }}</el-descriptions-item>

        <el-descriptions-item label="业务主键">{{ scheme.refId }}</el-descriptions-item>

        <el-descriptions-item label="累计盈亏（投注记录）">

          <span

            :style="{

              color: settledPnl >= 0 ? 'var(--el-color-success)' : 'var(--el-color-danger)',

            }"

          >

            {{ settledPnl >= 0 ? '+' : '' }}{{ settledPnl.toFixed(2) }} 元

          </span>

        </el-descriptions-item>

        <el-descriptions-item label="投注胜率">

          {{ executionWinRate }}（{{ settledRows.length }} 注）

        </el-descriptions-item>

      </el-descriptions>



      <el-tabs v-model="tab" class="scheme-history-tabs">

        <el-tab-pane label="投注与记录" name="bets">

          <el-table v-loading="betsLoading" :data="pagedBetItems" stripe size="small" style="width: 100%">

            <el-table-column prop="time" label="时间" min-width="88" />

            <el-table-column prop="period" label="期数" min-width="110" show-overflow-tooltip />

            <el-table-column prop="playMethod" label="玩法" min-width="88" show-overflow-tooltip />

            <el-table-column prop="numbers" label="下注号码" min-width="100" show-overflow-tooltip>

              <template #default="{ row }">

                <span class="scheme-history-mono">{{ row.numbers || '—' }}</span>

              </template>

            </el-table-column>

            <el-table-column prop="draw" label="开奖号码" min-width="100" show-overflow-tooltip>

              <template #default="{ row }">

                <span class="scheme-history-mono">{{ row.draw || '—' }}</span>

              </template>

            </el-table-column>

            <el-table-column prop="multiplier" label="倍数" min-width="56" align="center" />

            <el-table-column prop="round" label="轮次" min-width="56" align="center" />

            <el-table-column label="金额" min-width="64" align="right">

              <template #default="{ row }">

                <span class="scheme-history-mono">{{ formatBetRecordAmount(row.amount) }}</span>

              </template>

            </el-table-column>

            <el-table-column label="盈亏" min-width="64" align="right">

              <template #default="{ row }">

                <span

                  class="scheme-history-mono"

                  :class="

                    row.status === '已撤单' || row.status === '待开奖'

                      ? 'scheme-history-pl--neutral'

                      : row.profitLoss > 0

                        ? 'scheme-history-pl--gain'

                        : row.profitLoss < 0

                          ? 'scheme-history-pl--loss'

                          : 'scheme-history-pl--neutral'

                  "

                >

                  {{
                    row.status === '已撤单' || row.status === '待开奖'
                      ? '—'
                      : formatBetRecordPl(row.profitLoss)
                  }}

                </span>

              </template>

            </el-table-column>

            <el-table-column label="结果" min-width="80" align="center">

              <template #default="{ row }">

                <el-tag

                  v-if="row.win === true"

                  type="success"

                  size="small"

                  effect="light"

                >中</el-tag>

                <el-tag

                  v-else-if="row.win === false"

                  type="danger"

                  size="small"

                  effect="light"

                >挂</el-tag>

                <el-tag v-else type="info" size="small" effect="light">{{ row.status }}</el-tag>

              </template>

            </el-table-column>

          </el-table>

          <div class="scheme-history-pager">

            <el-pagination

              v-model:current-page="betPage"

              :page-size="pageSize"

              layout="total, prev, pager, next"

              :total="betItems.length"

            />

          </div>

        </el-tab-pane>



        <el-tab-pane label="历史开奖" name="draws">

          <p class="scheme-history-hint">与 client「历史开奖」：号码 / 大小 / 单双 / 龙虎 / 总和 子 Tab。</p>

          <el-radio-group v-model="historySubTab" size="small" style="margin-bottom: 0.75rem">

            <el-radio-button

              v-for="(label, hi) in historySubTabLabels"

              :key="label"

              :value="hi"

            >

              {{ label }}

            </el-radio-button>

          </el-radio-group>

          <div v-loading="drawsLoading" class="scheme-draw-list">

            <el-empty v-if="!drawsLoading && drawRecords.length === 0" description="暂无历史开奖" />

            <article v-for="(rec, idx) in drawRecords" :key="`${rec.periodShort}-${rec.time}-${idx}`" class="scheme-draw-card">

              <div class="scheme-draw-head">

                <span>{{ scheme.lotteryLabel }}</span>

                <span>第 <strong>{{ rec.periodShort }}</strong> 期</span>

              </div>

              <div class="scheme-draw-body">

                <template v-if="historySubTab === 0">

                  <span v-for="(b, bi) in rec.balls" :key="bi" class="scheme-draw-ball">{{ b }}</span>

                </template>

                <template v-else-if="historySubTab === 1">

                  <span

                    v-for="(b, bi) in rec.balls"

                    :key="bi"

                    class="scheme-draw-tag"

                    :class="historyBigSmallDigit(b) === '大' ? 'scheme-draw-tag--warm' : 'scheme-draw-tag--cool'"

                  >

                    {{ historyBigSmallDigit(b) }}

                  </span>

                </template>

                <template v-else-if="historySubTab === 2">

                  <span

                    v-for="(b, bi) in rec.balls"

                    :key="bi"

                    class="scheme-draw-tag"

                  >

                    {{ historyParityDigit(b) }}

                  </span>

                </template>

                <template v-else-if="historySubTab === 3">

                  <span

                    v-for="(cell, ci) in historyDragonTigerCells(historyDigitsFromBalls(rec.balls))"

                    :key="ci"

                    class="scheme-draw-dt"

                  >

                    {{ cell.char }} · {{ cell.label }}

                  </span>

                </template>

                <template v-else>

                  <span class="scheme-draw-sum">总和 {{ rec.sum }}</span>

                  <span class="scheme-draw-tag">{{ rec.sum >= 23 ? '大' : '小' }}</span>

                  <span class="scheme-draw-tag">{{ rec.sum % 2 === 1 ? '单' : '双' }}</span>

                </template>

              </div>

              <div class="scheme-draw-time">{{ rec.time }}</div>

            </article>

          </div>

        </el-tab-pane>

      </el-tabs>



      <div style="margin-top: 1rem; text-align: right">

        <el-button @click="close">关闭</el-button>

      </div>

    </template>

  </el-drawer>

</template>



<style scoped>

.scheme-history-hint {

  margin: 0 0 0.75rem;

  font-size: 12px;

  color: var(--el-text-color-secondary);

}



.scheme-history-subtitle {

  margin: 1rem 0 0.5rem;

  font-size: 14px;

  font-weight: 600;

}



.scheme-history-mono {

  font-variant-numeric: tabular-nums;

}



.scheme-history-pl--gain {

  color: var(--el-color-success);

}



.scheme-history-pl--loss {

  color: var(--el-color-danger);

}



.scheme-history-pl--neutral {

  color: var(--el-text-color-secondary);

}



.scheme-history-pager {

  display: flex;

  justify-content: flex-end;

  margin-top: 1rem;

}



.scheme-draw-list {

  display: flex;

  flex-direction: column;

  gap: 0.75rem;

  max-height: 420px;

  overflow-y: auto;

}



.scheme-draw-card {

  padding: 0.75rem 1rem;

  border-radius: 8px;

  background: var(--el-fill-color-lighter);

}



.scheme-draw-head {

  display: flex;

  justify-content: space-between;

  font-size: 13px;

  margin-bottom: 0.5rem;

}



.scheme-draw-body {

  display: flex;

  flex-wrap: wrap;

  gap: 0.35rem;

  align-items: center;

  min-height: 2rem;

}



.scheme-draw-ball {

  display: inline-flex;

  align-items: center;

  justify-content: center;

  width: 1.75rem;

  height: 1.75rem;

  border-radius: 50%;

  background: var(--el-color-primary);

  color: #fff;

  font-size: 12px;

  font-weight: 600;

}



.scheme-draw-tag {

  padding: 0.15rem 0.45rem;

  border-radius: 4px;

  font-size: 12px;

  background: var(--el-fill-color);

}



.scheme-draw-tag--warm {

  color: var(--el-color-danger);

}



.scheme-draw-tag--cool {

  color: var(--el-color-primary);

}



.scheme-draw-dt {

  font-size: 12px;

  padding: 0.15rem 0.5rem;

  background: var(--el-fill-color);

  border-radius: 4px;

}



.scheme-draw-sum {

  font-weight: 600;

  margin-right: 0.5rem;

}



.scheme-draw-time {

  margin-top: 0.5rem;

  font-size: 12px;

  color: var(--el-text-color-secondary);

}

</style>


