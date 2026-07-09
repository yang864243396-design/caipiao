<script setup lang="ts">
import { computed } from 'vue'

/**
 * 全局统一确认弹窗（屏幕居中），样式源自会员中心「登出确认」。
 * 既可声明式 v-model 使用，也可通过 @/utils/confirmDialog 的 confirmDialog() 命令式调用。
 */

type Tone = 'primary' | 'warning' | 'danger'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    message?: string
    icon?: string
    confirmText?: string
    cancelText?: string
    showCancel?: boolean
    tone?: Tone
  }>(),
  {
    title: '提示',
    message: '',
    icon: '',
    confirmText: '确定',
    cancelText: '取消',
    showCancel: true,
    tone: 'primary',
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const toneIcon = computed(() => {
  if (props.icon) return props.icon
  if (props.tone === 'danger') return 'warning'
  if (props.tone === 'warning') return 'error_outline'
  return 'help_outline'
})

function onConfirm(): void {
  emit('confirm')
  emit('update:modelValue', false)
}

function onCancel(): void {
  emit('cancel')
  emit('update:modelValue', false)
}
</script>

<template>
  <Teleport to="body">
    <transition name="cd-fade">
      <div v-if="modelValue" class="cd-overlay" @click.self="onCancel">
        <div class="cd-card" role="dialog" aria-modal="true">
          <div class="cd-body">
            <span class="cd-icon" :class="`cd-icon--${tone}`" aria-hidden="true">
              <span class="material-sym">{{ toneIcon }}</span>
            </span>
            <h2 class="cd-title">{{ title }}</h2>
            <p v-if="message" class="cd-desc">{{ message }}</p>
          </div>
          <div class="cd-actions">
            <el-button v-if="showCancel" class="cd-btn" @click="onCancel">{{ cancelText }}</el-button>
            <el-button
              type="primary"
              class="cd-btn"
              :class="{ 'cd-btn--danger': tone === 'danger' }"
              @click="onConfirm"
            >
              {{ confirmText }}
            </el-button>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<style scoped>
.cd-overlay {
  position: fixed;
  inset: 0;
  z-index: 3200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.42);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.cd-card {
  width: 100%;
  max-width: 22rem;
  background: #fff;
  border-radius: 1.5rem;
  overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 80, 203, 0.15);
}

.cd-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 0.65rem;
  padding: 2rem 1.5rem 0.5rem;
}

.cd-icon {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 16px;
  font-size: 1.6rem;
}

.cd-icon .material-sym {
  font-size: 1.6rem;
}

.cd-icon--primary {
  color: var(--el-color-primary, #0050cb);
  background: var(--el-color-primary-light-9, #e6ebfa);
}

.cd-icon--warning {
  color: #d97706;
  background: #fef3c7;
}

.cd-icon--danger {
  color: #dc2626;
  background: #fee2e2;
}

.cd-title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: #0f172a;
}

.cd-desc {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: #64748b;
}

.cd-actions {
  display: flex;
  gap: 0.75rem;
  padding: 1.25rem 1.5rem 1.5rem;
}

.cd-btn {
  flex: 1;
  height: 42px;
  margin: 0;
  font-weight: 600;
}

.cd-btn--danger {
  --el-color-primary: #dc2626;
  --el-color-primary-light-3: #ef4444;
  --el-color-primary-light-5: #f87171;
}

.cd-fade-enter-active,
.cd-fade-leave-active {
  transition: opacity 0.2s ease;
}

.cd-fade-enter-from,
.cd-fade-leave-to {
  opacity: 0;
}
</style>
