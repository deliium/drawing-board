<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { getMetrics } from '../services/migrationHealth'

const metrics = computed(() => getMetrics().slice(-10))

onMounted(() => {
  console.debug('[MigrationHealth] panel ready count=', metrics.value.length)
})
</script>

<template>
  <section class="dev-metrics" aria-label="Dev metrics">
    <h3>Dev metrics</h3>
    <ul>
      <li v-for="(metric, idx) in metrics" :key="idx">
        {{ metric.name }} = {{ metric.value }}
      </li>
    </ul>
  </section>
</template>

<style scoped>
.dev-metrics {
  margin-top: var(--space-4);
  padding: var(--space-3);
  border-top: 1px dashed var(--rule);
  color: var(--ink-muted);
  font-size: 0.8rem;
}

.dev-metrics h3 {
  margin: 0 0 var(--space-2);
  font-size: 0.85rem;
  font-weight: 600;
}

.dev-metrics ul {
  margin: 0;
  padding-left: 1.2rem;
}
</style>
