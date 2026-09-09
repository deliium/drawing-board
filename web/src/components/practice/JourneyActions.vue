<script setup lang="ts">
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
</script>

<template>
  <div class="actions">
    <template v-if="stage === 'intro'">
      <button type="button" class="primary" @click="emit('start')">Start</button>
      <button v-if="alreadyPassed" type="button" @click="emit('done')">Done</button>
    </template>

    <template v-else-if="stage === 'animate'">
      <button type="button" class="primary" @click="emit('continueAnimate')">Continue</button>
      <button type="button" @click="emit('cancel')">Cancel practice</button>
    </template>

    <template v-else-if="stage === 'trace'">
      <button type="button" class="primary" :disabled="!canNextTrace" @click="emit('nextTrace')">
        Next
      </button>
      <button type="button" @click="emit('showOrder')">Show order again</button>
      <button type="button" @click="emit('cancel')">Cancel practice</button>
    </template>

    <template v-else-if="stage === 'freewrite'">
      <button type="button" class="primary" @click="emit('submit')">Submit</button>
      <button type="button" @click="emit('showOrder')">Show order again</button>
      <button type="button" @click="emit('cancel')">Cancel practice</button>
    </template>

    <template v-else-if="stage === 'submitting'">
      <button type="button" disabled>Checking…</button>
      <button type="button" @click="emit('retryAssess')">Retry check</button>
    </template>

    <template v-else-if="stage === 'result'">
      <button v-if="pass" type="button" class="primary" @click="emit('done')">Done</button>
      <button type="button" :class="pass ? '' : 'primary'" @click="emit('retry')">Try again</button>
      <button v-if="!pass" type="button" @click="emit('done')">Done</button>
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
  gap: 8px;
  justify-content: center;
  padding: 8px;
}
button {
  padding: 8px 14px;
  border: 1px solid #cbd5e1;
  background: #f8fafc;
  border-radius: 4px;
  cursor: pointer;
}
button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.primary {
  background: #1e293b;
  color: #f8fafc;
  border-color: #1e293b;
}
</style>
