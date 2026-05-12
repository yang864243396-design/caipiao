<script setup lang="ts">
import { RouterView } from 'vue-router'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
</script>

<template>
  <el-config-provider :locale="zhCn">
    <div class="app-root">
      <RouterView />
    </div>
  </el-config-provider>
</template>

<style>
.app-root {
  min-height: 100dvh;
}
</style>
