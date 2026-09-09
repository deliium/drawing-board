<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import AppShell from './components/AppShell.vue'
import MigrationHealthPanel from './components/MigrationHealthPanel.vue'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

onMounted(() => {
  if (isDev) {
    console.debug('[MigrationHealth] mount gated=DEV')
  }
})
</script>

<template>
  <AppShell>
    <RouterView />
    <MigrationHealthPanel v-if="isDev" />
  </AppShell>
</template>
