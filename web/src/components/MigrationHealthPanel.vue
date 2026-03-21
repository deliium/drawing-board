<script setup lang="ts">
import { computed } from 'vue'
import { getMetrics } from '../services/migrationHealth'

const metrics = computed(() => getMetrics().slice(-10))
</script>

<template>
  <section>
    <h3>Migration Health</h3>
    <ul>
      <li v-for="(metric, idx) in metrics" :key="idx">
        {{ metric.name }} = {{ metric.value }}
      </li>
    </ul>
  </section>
</template>
