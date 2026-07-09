<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { InfoFilled, QuestionFilled, WarningFilled } from '@element-plus/icons-vue'
import {
  adminConfirmState,
  resolveAdminConfirm,
  resolveAdminPrompt,
  type AdminConfirmTone,
} from '@/utils/adminConfirmDialog'

const promptInput = ref('')

watch(
  () => adminConfirmState.visible,
  (visible) => {
    if (visible && adminConfirmState.mode === 'prompt') {
      promptInput.value = adminConfirmState.promptValue
    }
  },
)

const toneIcon = computed(() => {
  const tone: AdminConfirmTone = adminConfirmState.tone
  if (tone === 'danger' || tone === 'warning') return WarningFilled
  return adminConfirmState.mode === 'prompt' ? QuestionFilled : InfoFilled
})

const iconClass = computed(() => `admin-confirm-icon--${adminConfirmState.tone}`)

function onCancel(): void {
  if (adminConfirmState.mode === 'prompt') {
    resolveAdminPrompt(null)
    return
  }
  resolveAdminConfirm(false)
}

function onConfirm(): void {
  if (adminConfirmState.mode === 'prompt') {
    resolveAdminPrompt(promptInput.value.trim())
    return
  }
  resolveAdminConfirm(true)
}
</script>

<template>
  <Teleport to="body">
    <transition name="admin-confirm-fade">
      <div
        v-if="adminConfirmState.visible"
        class="admin-confirm-overlay"
        @click.self="onCancel"
      >
        <div class="admin-confirm-card" role="dialog" aria-modal="true">
          <div class="admin-confirm-body">
            <span class="admin-confirm-icon" :class="iconClass" aria-hidden="true">
              <el-icon><component :is="toneIcon" /></el-icon>
            </span>
            <h2 class="admin-confirm-title">{{ adminConfirmState.title }}</h2>
            <p v-if="adminConfirmState.message" class="admin-confirm-desc">
              {{ adminConfirmState.message }}
            </p>
            <el-input
              v-if="adminConfirmState.mode === 'prompt'"
              v-model="promptInput"
              class="admin-confirm-input"
              :placeholder="adminConfirmState.promptPlaceholder || '请输入'"
              clearable
              @keyup.enter="onConfirm"
            />
          </div>
          <div class="admin-confirm-actions">
            <el-button v-if="adminConfirmState.showCancel" @click="onCancel">
              {{ adminConfirmState.cancelText }}
            </el-button>
            <el-button
              type="primary"
              :class="{ 'admin-confirm-btn-danger': adminConfirmState.tone === 'danger' }"
              @click="onConfirm"
            >
              {{ adminConfirmState.confirmText }}
            </el-button>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>
