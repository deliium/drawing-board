<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppShell from './components/AppShell.vue'
import MigrationHealthPanel from './components/MigrationHealthPanel.vue'
</script>

<template>
  <AppShell>
    <RouterView />
    <MigrationHealthPanel />
  </AppShell>
</template>
