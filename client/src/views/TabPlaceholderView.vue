<script setup lang="ts">
import { useRouter } from 'vue-router'
import LobbyTabBar from '@/components/lobby/LobbyTabBar.vue'

const router = useRouter()

withDefaults(
  defineProps<{
    title: string
    /** 与底栏/功能块统一的说明插图 */
    imageSrc?: string
  }>(),
  { imageSrc: '' }
)

function goLobby() {
  router.push('/')
}
</script>

<template>
  <div class="ph">
    <div v-if="imageSrc" class="hero">
      <img
        :src="imageSrc"
        :alt="title"
        class="hero-img"
        width="200"
        height="200"
        loading="lazy"
        decoding="async"
      />
    </div>
    <el-empty :description="`${title} · 此页为占位，后续按设计逐屏对接。`" class="ph-empty" />
    <el-button type="primary" round @click="goLobby">返回游戏大厅</el-button>
    <LobbyTabBar />
  </div>
</template>

<style scoped>
.ph {
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 1.5rem 1.5rem 6.5rem;
  background: #f7f9fb;
  color: #191c1e;
  text-align: center;
  gap: 1rem;
}

.hero {
  margin-bottom: 0;
  width: min(12rem, 60vw);
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.hero-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.ph-empty {
  padding: 0;
}

.ph-empty :deep(.el-empty__description) {
  margin-top: 0.5rem;
  color: #424656;
  font-size: 0.9rem;
  max-width: 18rem;
  line-height: 1.5;
}
</style>
