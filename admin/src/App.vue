<script setup lang="ts">
import { RouterView } from 'vue-router'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import AdminConfirmDialog from '@/components/AdminConfirmDialog.vue'
</script>

<template>
  <el-config-provider :locale="zhCn">
    <RouterView />
    <AdminConfirmDialog />
  </el-config-provider>
</template>

<style>
html,
body,
#app {
  margin: 0;
  min-height: 100dvh;
  width: 100%;
  max-width: 100%;
}
</style>
