<script setup lang="ts">
import type { JourneyBanner, JourneyStage } from '../../composables/usePracticeJourney'

defineProps<{
  stage: JourneyStage
  banner?: JourneyBanner
  softWarn?: string | null
}>()
</script>

<template>
  <div class="banner-wrap" role="status">
    <p v-if="stage === 'loading'" class="banner info">Loading…</p>
    <p v-else-if="stage === 'empty'" class="banner error">This character is not in the lesson.</p>
    <p v-else-if="stage === 'submitting'" class="banner info">Checking…</p>
    <p v-if="banner" class="banner" :class="banner.kind">{{ banner.message }}</p>
    <p v-if="softWarn" class="banner notice">{{ softWarn }}</p>
  </div>
</template>

<style scoped>
.banner-wrap {
  min-height: 1.5rem;
  text-align: center;
  padding: 4px 8px;
}
.banner {
  margin: 4px 0;
  font-size: 0.95rem;
}
.info {
  opacity: 0.8;
}
.notice {
  color: #92400e;
}
.error {
  color: #9f1239;
}
</style>
