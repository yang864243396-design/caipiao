<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { fetchBetRecordItem, type BetRecordItemDetail } from '@/api/cloud/betRecords'
import { currencySymbol } from '@/utils/currencyDisplay'
import { betContentLines as resolveBetContentLines } from '@/utils/betContentDisplay'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const loadError = ref('')
const detail = ref<BetRecordItemDetail | null>(null)

const recordNo = computed(() => String(route.params.recordNo ?? '').trim())

const thirdPartyDisplay = computed(() => {
  const id = (detail.value?.thirdPartyId || '').trim()
  return id || '—'
})

const winSituation = computed(() => {
  const d = detail.value
  if (!d) return '—'
  const balls = (d.drawNumbers || '').trim() || '—'
  const label = (d.statusLabel || '').trim() || '—'
  return `${balls} · ${label}`
})

const betUnitsDisplay = computed(() => {
  const n = detail.value?.betUnits
  if (n == null || !Number.isFinite(n) || n <= 0) return '—'
  return String(n)
})

const amountDisplay = computed(() => {
  const d = detail.value
  if (!d) return '—'
  const sym = currencySymbol(d.currency || 'USDT')
  return `${d.amount.toFixed(2)} ${sym}`
})

const payoutDisplay = computed(() => {
  const d = detail.value
  if (!d) return '—'
  if (d.payoutAmount == null) return '—'
  const sym = currencySymbol(d.currency || 'USDT')
  return `${d.payoutAmount.toFixed(2)} ${sym}`
})

const betContentLines = computed(() =>
  resolveBetContentLines(detail.value?.betContentLines, detail.value?.betContent ?? ''),
)

async function load(): Promise<void> {
  const id = recordNo.value
  if (!id) {
    loadError.value = '记录不存在或已超出可查询范围'
    detail.value = null
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    detail.value = await fetchBetRecordItem(id)
  } catch (e) {
    detail.value = null
    loadError.value = '记录不存在或已超出可查询范围'
    const msg = e instanceof Error ? e.message : ''
    if (msg && !/不存在|超出|404|Not Found/i.test(msg)) {
      ElMessage.error(msg)
    }
  } finally {
    loading.value = false
  }
}

function goBack(): void {
  if (window.history.length > 1) router.back()
  else void router.push({ name: 'bet-records' })
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="bd" data-page="bet-detail" v-loading="loading">
    <header class="bd-head">
      <button type="button" class="bd-back" aria-label="返回" @click="goBack">
        <span class="material-sym">arrow_back_ios_new</span>
      </button>
      <h1 class="bd-title">投注详情</h1>
      <span class="bd-head-spacer" aria-hidden="true" />
    </header>

    <main class="bd-main">
      <section v-if="loadError" class="bd-card bd-empty">
        <p class="bd-empty-text">{{ loadError }}</p>
        <el-button type="primary" @click="goBack">返回</el-button>
      </section>

      <section v-else-if="detail" class="bd-card">
        <div class="bd-field">
          <span class="bd-label">注单编号</span>
          <span class="bd-value bd-value--mono">{{ thirdPartyDisplay }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注期数</span>
          <span class="bd-value">{{ detail.period || '—' }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注彩种</span>
          <span class="bd-value">{{ detail.lotteryLabel || '—' }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注玩法</span>
          <span class="bd-value">{{ detail.playType || '—' }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">中奖情况</span>
          <span class="bd-value">{{ winSituation }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注注数</span>
          <span class="bd-value">{{ betUnitsDisplay }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注倍率</span>
          <span class="bd-value">{{ detail.multiplier || '—' }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">倍投局数</span>
          <span class="bd-value">{{ detail.round || '—' }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注金额</span>
          <span class="bd-value">{{ amountDisplay }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">返奖金额</span>
          <span class="bd-value">{{ payoutDisplay }}</span>
        </div>
        <div class="bd-field">
          <span class="bd-label">投注时间</span>
          <span class="bd-value">{{ detail.placedAt || '—' }}</span>
        </div>
        <div class="bd-field bd-field--bet">
          <span class="bd-label">我的投注</span>
          <span class="bd-value bd-value--bet" aria-label="投注号码">
            <span v-for="(line, idx) in betContentLines" :key="idx" class="bd-bet-line">{{ line }}</span>
          </span>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.bd {
  min-height: 100dvh;
  background: #f7f9fb;
  color: #191c1e;
  padding-bottom: calc(1.5rem + env(safe-area-inset-bottom));
}

.bd-head {
  display: grid;
  grid-template-columns: var(--page-titlebar-action-size) 1fr var(--page-titlebar-action-size);
  align-items: center;
  height: var(--page-titlebar-height);
  min-height: var(--page-titlebar-height);
  box-sizing: border-box;
  padding: 0 var(--page-gutter);
  background: #fff;
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
  position: sticky;
  top: 0;
  z-index: 20;
}

.bd-back {
  display: grid;
  place-items: center;
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 0.5rem;
  background: transparent;
  color: #191c1e;
  cursor: pointer;
  justify-self: start;
}

.bd-back .material-sym {
  font-size: var(--page-titlebar-back-icon-size);
}

.bd-title {
  margin: 0;
  text-align: center;
  font-family: 'Plus Jakarta Sans', 'Noto Sans SC', system-ui, sans-serif;
  font-size: 1.05rem;
  font-weight: 700;
  color: #191c1e;
}

.bd-head-spacer {
  width: var(--page-titlebar-action-size);
  height: var(--page-titlebar-action-size);
  justify-self: end;
}

.bd-main {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 1rem var(--page-gutter);
}

.bd-card {
  background: #fff;
  border-radius: 0.875rem;
  padding: var(--card-pad);
  box-shadow: 0 4px 20px rgba(25, 28, 30, 0.04);
}

.bd-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 2rem 1rem;
}

.bd-empty-text {
  margin: 0;
  font-size: 0.875rem;
  color: rgba(66, 70, 86, 0.72);
  text-align: center;
}

.bd-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 36px;
  box-sizing: border-box;
  padding: 0.35rem 0;
  border-bottom: 1px solid rgba(242, 244, 246, 0.95);
}

.bd-field:last-child {
  border-bottom: none;
}

.bd-field--bet {
  align-items: flex-start;
}

.bd-label {
  flex: none;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #727687;
  line-height: 1.6;
  padding-top: 0.1rem;
}

.bd-value {
  min-width: 0;
  text-align: right;
  font-size: 0.875rem;
  font-weight: 600;
  color: #191c1e;
  word-break: break-all;
  line-height: 1.6;
}

.bd-value--mono {
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  font-size: 0.8125rem;
}

.bd-value--bet {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.15rem;
}

.bd-bet-line {
  display: block;
  font-variant-numeric: tabular-nums;
}
</style>
