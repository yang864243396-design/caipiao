<script setup lang="ts">
import { RouterView } from 'vue-router'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import { confirmState, resolveConfirm } from '@/utils/confirmDialog'
</script>

<template>
  <el-config-provider :locale="zhCn">
    <div class="app-root">
      <RouterView />
    </div>
    <ConfirmDialog
      :model-value="confirmState.visible"
      :title="confirmState.title"
      :message="confirmState.message"
      :icon="confirmState.icon"
      :confirm-text="confirmState.confirmText"
      :cancel-text="confirmState.cancelText"
      :show-cancel="confirmState.showCancel"
      :tone="confirmState.tone"
      @confirm="resolveConfirm(true)"
      @cancel="resolveConfirm(false)"
      @update:model-value="(v) => { if (!v) resolveConfirm(false) }"
    />
  </el-config-provider>
</template>

<style>
.app-root {
  min-height: 100dvh;
}
</style>
