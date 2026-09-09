<script setup lang="ts">
import { useLocale } from '../../composables/useLocale'
import type { JourneyStage } from '../../composables/usePracticeJourney'

defineProps<{
  stage: JourneyStage
  canNextTrace?: boolean
  pass?: boolean | null
  alreadyPassed?: boolean
}>()

const emit = defineEmits<{
  start: []
  continueAnimate: []
  nextTrace: []
  showOrder: []
  submit: []
  retry: []
  retryAssess: []
  done: []
  cancel: []
}>()

const { t } = useLocale()
</script>

<template>
  <div class="actions">
    <template v-if="stage === 'intro'">
      <button type="button" class="primary" @click="emit('start')">{{ t('action.start') }}</button>
      <button v-if="alreadyPassed" type="button" @click="emit('done')">{{ t('action.done') }}</button>
    </template>

    <template v-else-if="stage === 'animate'">
      <button type="button" class="primary" @click="emit('continueAnimate')">
        {{ t('action.continue') }}
      </button>
      <button type="button" @click="emit('cancel')">{{ t('action.cancel') }}</button>
    </template>

    <template v-else-if="stage === 'trace'">
      <button type="button" class="primary" :disabled="!canNextTrace" @click="emit('nextTrace')">
        {{ t('action.next') }}
      </button>
      <button type="button" @click="emit('showOrder')">{{ t('action.showOrder') }}</button>
      <button type="button" @click="emit('cancel')">{{ t('action.cancel') }}</button>
    </template>

    <template v-else-if="stage === 'freewrite'">
      <button type="button" class="primary" @click="emit('submit')">{{ t('action.submit') }}</button>
      <button type="button" @click="emit('showOrder')">{{ t('action.showOrder') }}</button>
      <button type="button" @click="emit('cancel')">{{ t('action.cancel') }}</button>
    </template>

    <template v-else-if="stage === 'submitting'">
      <button type="button" disabled>{{ t('practice.checking') }}</button>
      <button type="button" @click="emit('retryAssess')">{{ t('action.retryCheck') }}</button>
    </template>

    <template v-else-if="stage === 'result'">
      <button v-if="pass" type="button" class="primary" @click="emit('done')">
        {{ t('action.done') }}
      </button>
      <button type="button" :class="pass ? '' : 'primary'" @click="emit('retry')">
        {{ t('action.tryAgain') }}
      </button>
      <button v-if="!pass" type="button" @click="emit('done')">{{ t('action.done') }}</button>
    </template>

    <template v-else-if="stage === 'complete'">
      <slot name="complete-links" />
    </template>

    <template v-else-if="stage === 'error'">
      <slot name="error-actions" />
    </template>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  justify-content: center;
  padding: var(--space-2);
}

button {
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  border: 1px solid var(--rule);
  background: var(--paper-raised);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font: inherit;
  color: var(--ink);
}

button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.primary {
  background: var(--accent);
  color: #f8fafc;
  border-color: var(--accent);
  font-weight: 600;
}
</style>
