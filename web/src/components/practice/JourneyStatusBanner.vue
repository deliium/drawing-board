<script setup lang="ts">
import { useLocale } from '../../composables/useLocale'
import type { JourneyBanner, JourneyStage } from '../../composables/usePracticeJourney'

defineProps<{
  stage: JourneyStage
  banner?: JourneyBanner
  softWarn?: string | null
}>()

const { t } = useLocale()
</script>

<template>
  <div
    class="banner-wrap"
    role="status"
    aria-live="polite"
    aria-atomic="true"
    :aria-label="t('banner.status')"
  >
    <p v-if="stage === 'loading'" class="banner info">{{ t('practice.loading') }}</p>
    <p v-else-if="stage === 'empty'" class="banner error">{{ t('practice.emptyCharacter') }}</p>
    <p v-else-if="stage === 'submitting'" class="banner info">{{ t('practice.checking') }}</p>
    <p v-if="banner" class="banner" :class="banner.kind">{{ banner.message }}</p>
    <p v-if="softWarn" class="banner notice">{{ softWarn }}</p>
  </div>
</template>

<style scoped>
.banner-wrap {
  min-height: 1.5rem;
  text-align: center;
  padding: var(--space-1) var(--space-2);
}

.banner {
  margin: var(--space-1) 0;
  font-size: 0.95rem;
}

.info {
  color: var(--ink-muted);
}

.notice {
  color: #92400e;
}

.error {
  color: var(--danger);
}
</style>
