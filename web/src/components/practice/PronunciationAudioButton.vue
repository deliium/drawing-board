<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useLocale } from '../../composables/useLocale'

const props = defineProps<{
  audioRef: string
  glyph: string
}>()

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const { t } = useLocale()

const audioEl = ref<HTMLAudioElement | null>(null)
const playing = ref(false)
const unavailable = ref(false)
const liveMessage = ref('')
const warnedRefs = new Set<string>()

const label = computed(() => t('intro.playAudio', { glyph: props.glyph }))

function resetLiveSoon() {
  window.setTimeout(() => {
    liveMessage.value = ''
  }, 2500)
}

function onPlay() {
  playing.value = true
  liveMessage.value = t('intro.audioPlaying', { glyph: props.glyph })
  if (isDev) console.debug('[PronunciationAudioButton] play', props.audioRef)
}

function onEnded() {
  playing.value = false
}

function onError() {
  playing.value = false
  unavailable.value = true
  liveMessage.value = t('intro.audioUnavailable')
  resetLiveSoon()
  if (!warnedRefs.has(props.audioRef)) {
    warnedRefs.add(props.audioRef)
    console.warn('[audio] load error ref=%s', props.audioRef)
  }
}

async function play() {
  const el = audioEl.value
  if (!el || unavailable.value) return
  try {
    el.currentTime = 0
    await el.play()
  } catch (err) {
    if (isDev) console.debug('[PronunciationAudioButton] play failed', err)
    onError()
  }
}

watch(
  () => props.audioRef,
  () => {
    unavailable.value = false
    playing.value = false
    liveMessage.value = ''
  },
)

onBeforeUnmount(() => {
  const el = audioEl.value
  if (!el) return
  try {
    el.pause()
  } catch {
    /* jsdom lacks media pause */
  }
})
</script>

<template>
  <div class="audio-wrap">
    <audio
      ref="audioEl"
      :src="audioRef"
      preload="metadata"
      @play="onPlay"
      @ended="onEnded"
      @pause="playing = false"
      @error="onError"
    />
    <button
      type="button"
      class="play"
      :aria-label="label"
      :aria-pressed="playing"
      :disabled="unavailable"
      @click="play"
    >
      {{ unavailable ? t('intro.audioUnavailable') : label }}
    </button>
    <p class="sr-live" aria-live="polite">{{ liveMessage }}</p>
  </div>
</template>

<style scoped>
.audio-wrap {
  margin: var(--space-2) 0;
}

.play {
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  cursor: pointer;
  font: inherit;
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
}

.play:disabled {
  opacity: 0.65;
  cursor: default;
}

.sr-live {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
