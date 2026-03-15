<template>
  <div class="min-h-screen bg-gradient-animated font-apple transition-colors duration-300 overflow-x-hidden">
    <AppSidebar v-model:collapsed="sidebarCollapsed" />
    <main
      class="transition-[margin-left] duration-300 ease-out p-6 min-w-0"
      :style="{ marginLeft: `${sidebarWidth}px` }"
    >
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import AppSidebar from './AppSidebar.vue'

const sidebarCollapsed = useLocalStorage('e5-sidebar-collapsed', false)
const sidebarWidth = computed(() => (sidebarCollapsed.value ? 64 : 224))
</script>
