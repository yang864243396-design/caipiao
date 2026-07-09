<script setup lang="ts">
/**
 * 全局统一内容弹窗（屏幕居中），视觉与 ConfirmDialog 一致，支持 slot / HTML 正文。
 */

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    icon?: string
    confirmText?: string
    cancelText?: string
    showCancel?: boolean
    confirmLoading?: boolean
    /** 为 false 时仅 emit confirm，由调用方在异步完成后关闭 */
    autoCloseOnConfirm?: boolean
    zIndex?: number
    wide?: boolean
  }>(),
  {
    title: '提示',
    icon: 'info',
    confirmText: '知道了',
    cancelText: '取消',
    showCancel: false,
    confirmLoading: false,
    autoCloseOnConfirm: true,
    zIndex: 6000,
    wide: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

function close(): void {
  emit('update:modelValue', false)
}

function onConfirm(): void {
  emit('confirm')
  if (props.autoCloseOnConfirm) {
    close()
  }
}

function onCancel(): void {
  emit('cancel')
  close()
}

function onOverlayClick(): void {
  if (props.showCancel) {
    onCancel()
  } else {
    onConfirm()
  }
}
</script>

<template>
  <Teleport to="body">
    <transition name="cd-fade">
      <div
        v-if="modelValue"
        class="cd-overlay"
        :style="{ zIndex: props.zIndex }"
        @click.self="onOverlayClick"
      >
        <div
          class="cd-card"
          :class="{ 'cd-card--wide': wide }"
          role="dialog"
          aria-modal="true"
        >
          <div class="cd-body">
            <span class="cd-icon cd-icon--primary" aria-hidden="true">
              <span class="material-sym">{{ icon }}</span>
            </span>
            <h2 class="cd-title">{{ title }}</h2>
            <div v-if="$slots.default" class="cd-content cms-rich-html">
              <slot />
            </div>
          </div>
          <div class="cd-actions">
            <el-button
              v-if="showCancel"
              class="cd-btn"
              :disabled="confirmLoading"
              @click="onCancel"
            >
              {{ cancelText }}
            </el-button>
            <el-button
              type="primary"
              class="cd-btn"
              :loading="confirmLoading"
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
  min-width: 0;
  background: #fff;
  border-radius: 1.5rem;
  overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 80, 203, 0.15);
}

.cd-card--wide {
  max-width: min(92vw, 28rem);
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

.cd-title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: #0f172a;
}

.cd-content {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  max-height: min(50vh, 20rem);
  overflow-x: hidden;
  overflow-y: auto;
  text-align: left;
  font-size: 0.875rem;
  line-height: 1.65;
  color: #64748b;
  padding: 0.15rem 0.25rem 0;
  -webkit-overflow-scrolling: touch;
}

.cd-content :deep(a) {
  color: var(--el-color-primary, #0066ff);
}

.cd-content :deep(p) {
  margin: 0 0 0.65rem;
}

.cd-content :deep(p:last-child) {
  margin-bottom: 0;
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

.cd-fade-enter-active,
.cd-fade-leave-active {
  transition: opacity 0.2s ease;
}

.cd-fade-enter-from,
.cd-fade-leave-to {
  opacity: 0;
}
</style>
