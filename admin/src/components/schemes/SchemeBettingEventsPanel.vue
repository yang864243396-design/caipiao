<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchSchemeBettingEvents, type SchemeBettingEventRow } from '@/api/schemes'
import {
  cancelSchemeBettingOutbox,
  enableEventSchemeBetting,
  fetchCorePartitionStatus,
  fetchSchemeBettingSummary,
  rearmSchemeBetting,
  resolveSchemeBettingUnknown,
  type CorePartitionStatus,
  type SchemeBettingSummary,
} from '@/api/schemeBettingDiagnostics'

const text = {
  schemeId: '\u65b9\u6848\u5b9e\u4f8b ID',
  query: '\u67e5\u8be2',
  diagnostics: '\u4e8b\u4ef6\u6295\u6ce8\u8bca\u65ad',
  pending: '\u5f85\u5904\u7406',
  deadlineRisk: '\u4e34\u8fd1\u622a\u6b62',
  externalUnknown: '\u5916\u90e8\u63a5\u5355\u672a\u77e5',
  sentUnknown: '\u5f85\u5bf9\u8d26',
  wrongPeriod: '\u63a5\u9519\u671f',
  blocked: '\u5f85\u91cd\u65b0\u5f00\u542f',
  running: '\u4e8b\u4ef6\u9a71\u52a8\u8fd0\u884c\u4e2d',
  apiDue: 'API \u5f85\u6d3e\u53d1',
  unfinalized: '\u8d22\u52a1\u5f85\u843d\u8d26',
  workerLeases: '\u6709\u6548 Worker \u79df\u7ea6',
  drawLeases: '\u5f00\u5956\u4e3b\u79df\u7ea6',
  drawStrategyP99: '\u5f00\u5956\u2192\u7b56\u7565 p99',
  strategyAcceptedP99: '\u7b56\u7565\u2192\u63a5\u5355 p99',
  deadlineCompletion: '\u622a\u6b62\u524d\u5b8c\u6210\u7387',
  periodConsistency: '\u63a5\u5355\u671f\u4e00\u81f4\u7387',
  decisionId: '\u51b3\u7b56 ID',
  scheme: '\u65b9\u6848',
  lottery: '\u5f69\u79cd',
  sourcePeriod: '\u5f00\u5956\u671f\u53f7',
  targetPeriod: '\u76ee\u6807\u671f\u53f7',
  decision: '\u51b3\u7b56',
  stateVersion: '\u72b6\u6001\u7248\u672c',
  result: '\u5224\u5b9a',
  reason: '\u7ed3\u679c\u539f\u56e0',
  safeDeadline: '\u5b89\u5168\u622a\u6b62',
  decidedAt: '\u51b3\u7b56\u65f6\u95f4',
  actions: '\u64cd\u4f5c',
  attempts: '\u6d3e\u53d1\u6b21\u6570',
  queuePosition: '\u961f\u5217\u4f4d\u7f6e',
  actualAcceptedPeriod: '\u5b9e\u9645\u63a5\u5355\u671f',
  drawSource: '\u5f00\u5956\u6765\u6e90',
  lastError: '\u6700\u8fd1\u9519\u8bef',
  timeline: '\u94fe\u8def\u65f6\u95f4',
  cancel: '\u53d6\u6d88',
  enable: '\u542f\u7528\u4e8b\u4ef6\u9a71\u52a8',
  rearm: '\u91cd\u65b0\u5f00\u542f',
  resolve: '\u4eba\u5de5\u5bf9\u8d26',
  cancelTitle: '\u53d6\u6d88\u5f85\u53d1\u9001\u6295\u6ce8',
  rearmTitle: '\u91cd\u65b0\u5f00\u542f\u8fde\u7eed\u6295\u6ce8\u94fe',
  enableTitle: '\u5c06\u65b9\u6848\u5207\u6362\u4e3a\u4e8b\u4ef6\u9a71\u52a8',
  reasonPrompt: '\u8bf7\u8f93\u5165\u64cd\u4f5c\u7406\u7531\uff08\u81f3\u5c11 4 \u4e2a\u5b57\u7b26\uff09',
  reasonRequired: '\u64cd\u4f5c\u7406\u7531\u81f3\u5c11 4 \u4e2a\u5b57\u7b26',
  cancelled: '\u6295\u6ce8\u4efb\u52a1\u5df2\u53d6\u6d88',
  rearmed: '\u65b9\u6848\u5df2\u521b\u5efa\u65b0\u7684\u521d\u59cb\u6295\u6ce8\u94fe',
  enabled: '\u65b9\u6848\u5df2\u5207\u6362\u4e3a\u4e8b\u4ef6\u9a71\u52a8\u5e76\u521b\u5efa\u521d\u59cb\u6295\u6ce8',
  resolveTitle: '\u6838\u5bf9\u7b2c\u4e09\u65b9\u63a5\u5355\u7ed3\u679c',
  resolved: '\u672a\u77e5\u63a5\u5355\u7ed3\u679c\u5df2\u5904\u7406',
  accepted: '\u5df2\u63a5\u5355',
  rejected: '\u672a\u63a5\u5355',
  evidence: '\u7b2c\u4e09\u65b9\u6838\u5bf9\u8bc1\u636e',
  providerOrder: '\u7b2c\u4e09\u65b9\u8ba2\u5355\u53f7',
  acceptedPeriod: '\u7b2c\u4e09\u65b9\u63a5\u5355\u671f\u53f7',
  providerAmount: '\u7b2c\u4e09\u65b9\u63a5\u5355\u91d1\u989d',
  providerCurrency: '\u5e01\u79cd',
  providerAccount: '\u7b2c\u4e09\u65b9\u8d26\u53f7 ID',
  operationReason: '\u64cd\u4f5c\u7406\u7531',
  evidenceRequired: '\u6838\u5bf9\u8bc1\u636e\u81f3\u5c11 8 \u4e2a\u5b57\u7b26',
  acceptedFieldsRequired: '\u5df2\u63a5\u5355\u65f6\u5fc5\u987b\u5b8c\u6574\u586b\u5199\u7b2c\u4e09\u65b9\u8ba2\u5355\u4fe1\u606f',
  confirm: '\u786e\u8ba4',
  close: '\u53d6\u6d88',
  loadFailed: '\u6295\u6ce8\u4e8b\u4ef6\u52a0\u8f7d\u5931\u8d25',
  partitionStatus: '\u6838\u5fc3\u8868\u5206\u533a',
  partitionMirroring: '\u955c\u50cf\u540c\u6b65\u4e2d',
  partitionValidated: '\u6570\u636e\u5df2\u6821\u9a8c',
  partitionCutover: '\u5df2\u5207\u6362\uff0c\u5f85\u91cd\u542f\u786e\u8ba4',
  partitionReady: '\u5206\u533a\u8868\u5df2\u5728\u7ebf\uff0c\u53ef\u56de\u6eda',
  partitionValidation: '\u6821\u9a8c',
  partitionTables: '\u5728\u7ebf\u5206\u533a\u8868',
  hit: '\u4e2d',
  miss: '\u672a\u4e2d',
}

const rows = ref<SchemeBettingEventRow[]>([])
const summary = ref<SchemeBettingSummary>()
const corePartition = ref<CorePartitionStatus>()
const loading = ref(false)
const actionLoading = ref('')
const schemeId = ref('')
const reconcileVisible = ref(false)
const reconcileRow = ref<SchemeBettingEventRow>()
const reconcileForm = ref({
  outcome: 'accepted' as 'accepted' | 'rejected',
  evidence: '',
  reason: '',
  providerOrderId: '',
  acceptedPeriod: '',
  providerAmount: 0,
  providerAccountId: 0,
  providerCurrency: '',
})
const modeLabel = computed(() => {
  const modes = summary.value?.modes ?? []
  return modes.length > 0 ? modes.join(' / ') : text.diagnostics
})
const partitionStatusTitle = computed(() => {
  const phase = corePartition.value?.phase
  const label = ({
    mirroring: text.partitionMirroring,
    validated: text.partitionValidated,
    cutover: text.partitionCutover,
    rollback_ready: text.partitionReady,
  } as const)[phase ?? 'mirroring']
  return `${text.partitionStatus}: ${label}`
})
const partitionStatusType = computed(() => {
  if (corePartition.value?.phase === 'rollback_ready') return 'success'
  if (corePartition.value?.phase === 'cutover' || corePartition.value?.restartRequired) return 'warning'
  return 'info'
})
const partitionStatusDescription = computed(() => {
  const status = corePartition.value
  if (!status) return ''
  const active = Object.values(status.activeTablesPartitioned).filter(Boolean).length
  const valid = status.lastValidation.valid ? '\u901a\u8fc7' : '\u5f85\u6821\u9a8c'
  return `${text.partitionValidation}: ${valid} | ${text.partitionTables}: ${active}/3`
})

async function load() {
  loading.value = true
  try {
    const [events, currentSummary, partition] = await Promise.all([
      fetchSchemeBettingEvents({ schemeId: schemeId.value, limit: 200 }),
      fetchSchemeBettingSummary(),
      fetchCorePartitionStatus(),
    ])
    rows.value = events
    summary.value = currentSummary
    corePartition.value = partition
  } catch {
    ElMessage.error(text.loadFailed)
  } finally {
    loading.value = false
  }
}

function fmt(value?: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value))
}

function decisionLabel(value: SchemeBettingEventRow['decisionStatus']) {
  return ({
    completed: '\u5df2\u751f\u6210',
    blocked: '\u5df2\u963b\u65ad',
    duplicate: '\u91cd\u590d',
    chain_broken: '\u94fe\u8def\u4e2d\u65ad',
  } as const)[value] ?? value
}

function decisionType(value: SchemeBettingEventRow['decisionStatus']) {
  if (value === 'completed') return 'success'
  if (value === 'blocked' || value === 'chain_broken') return 'danger'
  return 'info'
}

function outboxLabel(value?: SchemeBettingEventRow['outboxState']) {
  if (!value) return '-'
  return ({
    pending: '\u5f85\u5904\u7406',
    leased: '\u5df2\u79df\u7ea6',
    sent_unknown: '\u5f85\u5bf9\u8d26',
    accepted: '\u5df2\u63a5\u5355',
    rejected: '\u5df2\u62d2\u7edd',
    expired: '\u5df2\u8fc7\u671f',
    cancelled: '\u5df2\u53d6\u6d88',
    accepted_wrong_period: '\u63a5\u9519\u671f',
    external_acceptance_unknown: '\u5916\u90e8\u63a5\u5355\u672a\u77e5',
  } as const)[value] ?? value
}

function canCancel(row: SchemeBettingEventRow) {
  return row.outboxId != null && (row.outboxState === 'pending' || row.outboxState === 'leased')
}

function canEnable(row: SchemeBettingEventRow, index: number) {
  return row.bettingOwner === 'legacy'
    && rows.value.findIndex(item => item.schemeId === row.schemeId) === index
}

function canRearm(row: SchemeBettingEventRow, index: number) {
  return row.chainState === 'blocked_requires_rearm'
    && rows.value.findIndex(item => item.schemeId === row.schemeId) === index
}

function fmtMs(value?: number) {
  if (value == null || !Number.isFinite(value)) return '-'
  return value >= 1000 ? `${(value / 1000).toFixed(2)}s` : `${Math.round(value)}ms`
}

function fmtRate(value?: number) {
  if (value == null || !Number.isFinite(value)) return '-'
  return `${value.toFixed(2)}%`
}

function timelineText(row: SchemeBettingEventRow) {
  return [
    `\u5f00\u5956\u53d1\u751f: ${fmt(row.drawProviderAt)}`,
    `\u672c\u5730\u63a5\u6536: ${fmt(row.drawReceivedAt)}`,
    `\u5f00\u5956\u786e\u8ba4: ${fmt(row.drawConfirmedAt)}`,
    `\u7b56\u7565\u5f00\u59cb: ${fmt(row.strategyStartedAt)}`,
    `\u7b56\u7565\u5b8c\u6210: ${fmt(row.strategyCompletedAt)}`,
    `\u4efb\u52a1\u521b\u5efa: ${fmt(row.readyCreatedAt)}`,
    `\u6d3e\u53d1\u5f00\u59cb: ${fmt(row.attemptStartedAt)}`,
    `\u6d3e\u53d1\u5b8c\u6210: ${fmt(row.attemptFinishedAt)}`,
  ].join('<br>')
}

function canResolve(row: SchemeBettingEventRow) {
  return row.outboxId != null
    && (row.outboxState === 'sent_unknown' || row.outboxState === 'external_acceptance_unknown')
}

function openResolve(row: SchemeBettingEventRow) {
  reconcileRow.value = row
  reconcileForm.value = {
    outcome: 'accepted',
    evidence: '',
    reason: '',
    providerOrderId: '',
    acceptedPeriod: row.targetPeriod ?? '',
    providerAmount: row.requestedAmount ?? 0,
    providerAccountId: 0,
    providerCurrency: row.requestedCurrency ?? '',
  }
  reconcileVisible.value = true
}

async function submitResolve() {
  const row = reconcileRow.value
  const form = reconcileForm.value
  if (row?.outboxId == null) return
  if (form.reason.trim().length < 4) {
    ElMessage.warning(text.reasonRequired)
    return
  }
  if (form.evidence.trim().length < 8) {
    ElMessage.warning(text.evidenceRequired)
    return
  }
  if (form.outcome === 'accepted' && (
    !form.providerOrderId.trim() || !form.acceptedPeriod.trim() || form.providerAmount <= 0
    || form.providerAccountId <= 0 || !form.providerCurrency.trim()
  )) {
    ElMessage.warning(text.acceptedFieldsRequired)
    return
  }
  actionLoading.value = `resolve-${row.outboxId}`
  try {
    await resolveSchemeBettingUnknown(row.outboxId, {
      reason: form.reason.trim(),
      outcome: form.outcome,
      evidence: form.evidence.trim(),
      ...(form.outcome === 'accepted' ? {
        providerOrderId: form.providerOrderId.trim(),
        acceptedPeriod: form.acceptedPeriod.trim(),
        providerAmount: form.providerAmount,
        providerAccountId: form.providerAccountId,
        providerCurrency: form.providerCurrency.trim(),
      } : {}),
    })
    reconcileVisible.value = false
    ElMessage.success(text.resolved)
    await load()
  } finally {
    actionLoading.value = ''
  }
}

async function promptReason(title: string) {
  const result = await ElMessageBox.prompt(text.reasonPrompt, title, {
    confirmButtonText: '\u786e\u8ba4',
    cancelButtonText: '\u53d6\u6d88',
    inputType: 'textarea',
    inputValidator: value => value.trim().length >= 4 || text.reasonRequired,
  })
  return result.value.trim()
}

async function cancelOutbox(row: SchemeBettingEventRow) {
  if (row.outboxId == null) return
  try {
    const reason = await promptReason(text.cancelTitle)
    actionLoading.value = `cancel-${row.outboxId}`
    await cancelSchemeBettingOutbox(row.outboxId, reason)
    ElMessage.success(text.cancelled)
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') throw error
  } finally {
    actionLoading.value = ''
  }
}

async function enableEvent(row: SchemeBettingEventRow) {
  try {
    const reason = await promptReason(text.enableTitle)
    actionLoading.value = `enable-${row.schemeId}`
    await enableEventSchemeBetting(row.schemeId, reason)
    ElMessage.success(text.enabled)
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') throw error
  } finally {
    actionLoading.value = ''
  }
}

async function rearm(row: SchemeBettingEventRow) {
  try {
    const reason = await promptReason(text.rearmTitle)
    actionLoading.value = `rearm-${row.schemeId}`
    await rearmSchemeBetting(row.schemeId, reason)
    ElMessage.success(text.rearmed)
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') throw error
  } finally {
    actionLoading.value = ''
  }
}

onMounted(() => void load())
</script>

<template>
  <div class="event-panel">
    <div class="event-toolbar">
      <el-input v-model="schemeId" clearable :placeholder="text.schemeId" @keyup.enter="load" />
      <el-button type="primary" @click="load">{{ text.query }}</el-button>
      <el-tag type="info" effect="plain">{{ modeLabel }}</el-tag>
    </div>

    <el-alert
      v-if="corePartition"
      class="partition-status"
      :title="partitionStatusTitle"
      :description="partitionStatusDescription"
      :type="partitionStatusType"
      :closable="false"
      show-icon
    />

    <div v-if="summary" class="event-summary">
      <div class="summary-item"><span>{{ text.pending }}</span><strong>{{ summary.pending }}</strong></div>
      <div class="summary-item"><span>{{ text.deadlineRisk }}</span><strong :class="{ danger: summary.deadlineRisk > 0 }">{{ summary.deadlineRisk }}</strong></div>
      <div class="summary-item"><span>{{ text.sentUnknown }}</span><strong :class="{ danger: summary.sentUnknown > 0 }">{{ summary.sentUnknown }}</strong></div>
      <div class="summary-item"><span>{{ text.externalUnknown }}</span><strong :class="{ danger: summary.externalUnknown > 0 }">{{ summary.externalUnknown }}</strong></div>
      <div class="summary-item"><span>{{ text.wrongPeriod }}</span><strong :class="{ danger: summary.acceptedWrongPeriod > 0 }">{{ summary.acceptedWrongPeriod }}</strong></div>
      <div class="summary-item"><span>{{ text.blocked }}</span><strong :class="{ danger: summary.blockedRequiresRearm > 0 }">{{ summary.blockedRequiresRearm }}</strong></div>
      <div class="summary-item"><span>{{ text.running }}</span><strong>{{ summary.runningEventOwned }}</strong></div>
      <div class="summary-item"><span>{{ text.apiDue }}</span><strong :class="{ danger: summary.apiDue > 0 }">{{ summary.apiDue }}</strong></div>
      <div class="summary-item"><span>{{ text.unfinalized }}</span><strong :class="{ danger: summary.acceptedUnfinalized > 0 }">{{ summary.acceptedUnfinalized }}</strong></div>
      <div class="summary-item"><span>{{ text.workerLeases }}</span><strong>{{ summary.activeStrategyLeases + summary.activeDispatcherLeases }}</strong></div>
      <div class="summary-item"><span>{{ text.drawLeases }}</span><strong>{{ summary.activeDrawLeases }}</strong></div>
      <div class="summary-item"><span>{{ text.drawStrategyP99 }}</span><strong>{{ fmtMs(summary.drawToStrategyP99Ms) }}</strong></div>
      <div class="summary-item"><span>{{ text.strategyAcceptedP99 }}</span><strong>{{ fmtMs(summary.strategyToAcceptedP99Ms) }}</strong></div>
      <div class="summary-item"><span>{{ text.deadlineCompletion }}</span><strong>{{ fmtRate(summary.safeDeadlineCompletionRate) }}</strong></div>
      <div class="summary-item"><span>{{ text.periodConsistency }}</span><strong>{{ fmtRate(summary.providerPeriodConsistencyRate) }}</strong></div>
    </div>

    <el-table v-loading="loading" :data="rows" stripe style="width: 100%">
      <el-table-column :label="text.decisionId" min-width="92">
        <template #default="{ row }">{{ row.decisionId || '-' }}</template>
      </el-table-column>
      <el-table-column :label="text.scheme" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.schemeName || row.schemeId }}</template>
      </el-table-column>
      <el-table-column prop="lotteryCode" :label="text.lottery" min-width="130" show-overflow-tooltip />
      <el-table-column prop="sourcePeriod" :label="text.sourcePeriod" min-width="150" show-overflow-tooltip />
      <el-table-column :label="text.targetPeriod" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.targetPeriod || '-' }}</template>
      </el-table-column>
      <el-table-column :label="text.decision" min-width="100">
        <template #default="{ row }"><el-tag :type="decisionType(row.decisionStatus)" effect="plain">{{ decisionLabel(row.decisionStatus) }}</el-tag></template>
      </el-table-column>
      <el-table-column label="Outbox" min-width="120">
        <template #default="{ row }">{{ outboxLabel(row.outboxState) }}</template>
      </el-table-column>
      <el-table-column :label="text.stateVersion" min-width="110" align="right">
        <template #default="{ row }">{{ row.stateVersionBefore }} -&gt; {{ row.stateVersionAfter }}</template>
      </el-table-column>
      <el-table-column :label="text.result" min-width="72">
        <template #default="{ row }">{{ row.localHit == null ? '-' : row.localHit ? text.hit : text.miss }}</template>
      </el-table-column>
      <el-table-column prop="outcomeReason" :label="text.reason" min-width="180" show-overflow-tooltip />
      <el-table-column :label="text.attempts" min-width="88" align="right">
        <template #default="{ row }">{{ row.attemptCount ?? '-' }}</template>
      </el-table-column>
      <el-table-column :label="text.queuePosition" min-width="88" align="right">
        <template #default="{ row }">{{ row.queuePosition ?? '-' }}</template>
      </el-table-column>
      <el-table-column prop="providerOrderNo" :label="text.providerOrder" min-width="170" show-overflow-tooltip />
      <el-table-column prop="acceptedPeriodNo" :label="text.actualAcceptedPeriod" min-width="150" show-overflow-tooltip />
      <el-table-column prop="drawSource" :label="text.drawSource" min-width="110" show-overflow-tooltip />
      <el-table-column prop="lastError" :label="text.lastError" min-width="180" show-overflow-tooltip />
      <el-table-column :label="text.timeline" min-width="120">
        <template #default="{ row }">
          <el-tooltip :content="timelineText(row)" placement="top" raw-content>
            <span class="timeline-link">{{ fmt(row.strategyCompletedAt || row.readyCreatedAt) }}</span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column :label="text.safeDeadline" min-width="170">
        <template #default="{ row }">{{ fmt(row.safeDeadlineAt) }}</template>
      </el-table-column>
      <el-table-column :label="text.decidedAt" min-width="170">
        <template #default="{ row }">{{ fmt(row.decidedAt) }}</template>
      </el-table-column>
      <el-table-column :label="text.actions" fixed="right" min-width="150">
        <template #default="{ row, $index }">
          <el-button v-if="canCancel(row)" link type="danger" :loading="actionLoading === `cancel-${row.outboxId}`" @click="cancelOutbox(row)">{{ text.cancel }}</el-button>
          <el-button v-if="canResolve(row)" link type="warning" :loading="actionLoading === `resolve-${row.outboxId}`" @click="openResolve(row)">{{ text.resolve }}</el-button>
          <el-button v-if="canEnable(row, $index)" link type="primary" :loading="actionLoading === `enable-${row.schemeId}`" @click="enableEvent(row)">{{ text.enable }}</el-button>
          <el-button v-if="canRearm(row, $index)" link type="primary" :loading="actionLoading === `rearm-${row.schemeId}`" @click="rearm(row)">{{ text.rearm }}</el-button>
          <span v-if="!canCancel(row) && !canResolve(row) && !canEnable(row, $index) && !canRearm(row, $index)">-</span>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="reconcileVisible" :title="text.resolveTitle" width="min(560px, 94vw)" destroy-on-close>
      <el-form label-position="top">
        <el-form-item>
          <el-radio-group v-model="reconcileForm.outcome">
            <el-radio-button value="accepted">{{ text.accepted }}</el-radio-button>
            <el-radio-button value="rejected">{{ text.rejected }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <template v-if="reconcileForm.outcome === 'accepted'">
          <el-form-item :label="text.providerOrder">
            <el-input v-model="reconcileForm.providerOrderId" />
          </el-form-item>
          <div class="reconcile-grid">
            <el-form-item :label="text.acceptedPeriod">
              <el-input v-model="reconcileForm.acceptedPeriod" />
            </el-form-item>
            <el-form-item :label="text.providerAccount">
              <el-input-number v-model="reconcileForm.providerAccountId" :min="1" :precision="0" controls-position="right" />
            </el-form-item>
            <el-form-item :label="text.providerAmount">
              <el-input-number v-model="reconcileForm.providerAmount" :min="0" :precision="3" controls-position="right" />
            </el-form-item>
            <el-form-item :label="text.providerCurrency">
              <el-input v-model="reconcileForm.providerCurrency" />
            </el-form-item>
          </div>
        </template>
        <el-form-item :label="text.evidence">
          <el-input v-model="reconcileForm.evidence" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item :label="text.operationReason">
          <el-input v-model="reconcileForm.reason" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reconcileVisible = false">{{ text.close }}</el-button>
        <el-button type="primary" :loading="actionLoading.startsWith('resolve-')" @click="submitResolve">{{ text.confirm }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.event-panel { width: 100%; }
.event-toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.event-toolbar .el-input { width: min(100%, 280px); }
.partition-status { margin-bottom: 16px; }
.event-summary { display: grid; grid-template-columns: repeat(6, minmax(112px, 1fr)); gap: 1px; margin-bottom: 16px; border: 1px solid var(--el-border-color-lighter); border-radius: 6px; overflow: hidden; background: var(--el-border-color-lighter); }
.summary-item { display: flex; flex-direction: column; gap: 4px; padding: 10px 12px; background: var(--el-bg-color); }
.summary-item span { font-size: 12px; color: var(--el-text-color-secondary); }
.summary-item strong { font-size: 18px; line-height: 1; }
.summary-item strong.danger { color: var(--el-color-danger); }
.reconcile-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 16px; }
.reconcile-grid :deep(.el-input-number) { width: 100%; }
@media (max-width: 900px) {
  .event-toolbar { align-items: stretch; flex-wrap: wrap; }
  .event-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .reconcile-grid { grid-template-columns: 1fr; }
}
</style>
