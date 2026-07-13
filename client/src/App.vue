<script setup lang="ts">
import { RouterView } from 'vue-router'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import LobbyTabBar from '@/components/lobby/LobbyTabBar.vue'
import { useLayoutMode } from '@/composables/useLayoutMode'
import { confirmState, resolveConfirm } from '@/utils/confirmDialog'

/** 同步 html.layout-web（屏幕 ≥1920×1080） */
useLayoutMode()
</script>

<template>
  <el-config-provider :locale="zhCn">
    <div class="app-root">
      <RouterView />
      <!-- 全局导航：H5 底栏 / Web 顶栏（对齐第三方桌面壳） -->
      <LobbyTabBar />
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
