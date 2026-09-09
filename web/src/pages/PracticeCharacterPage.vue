<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute } from 'vue-router'
import CharacterIntroPanel from '../components/practice/CharacterIntroPanel.vue'
import ComparisonOverlay from '../components/practice/ComparisonOverlay.vue'
import JourneyActions from '../components/practice/JourneyActions.vue'
import JourneyStatusBanner from '../components/practice/JourneyStatusBanner.vue'
import PracticeStageCanvas from '../components/practice/PracticeStageCanvas.vue'
import StrokeOrderPlayer from '../components/practice/StrokeOrderPlayer.vue'
import { usePracticeJourney } from '../composables/usePracticeJourney'
import type { Stroke } from '../services/strokeSync'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const route = useRoute()

const characterId = computed(() => {
  const raw = route.params.characterId
  const value = Array.isArray(raw) ? raw[0] : raw
  if (typeof value !== 'string') return ''
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
})

const characterIdRef = ref(characterId.value)
watch(characterId, (id) => {
  characterIdRef.value = id
})

const {
  stage,
  banner,
  lesson,
  character,
  progress,
  assessment,
  strokes,
  softWarn,
  canNextFromTrace,
  feedback,
  load,
  start,
  continueFromAnimate,
  nextFromTrace,
  showOrderAgain,
  submit,
  retryAssess,
  retry,
  complete,
  cancelPractice,
  onStrokesChanged,
} = usePracticeJourney({ characterId: characterIdRef })

onMounted(() => {
  if (isDev) console.debug('[PracticeCharacterPage] mount', characterId.value)
  void load()
})

watch(characterId, () => {
  void load()
})

onBeforeRouteLeave(() => {
  if (stage.value === 'trace' || stage.value === 'freewrite' || stage.value === 'animate') {
    void cancelPractice()
  }
})

function onStrokesUpdate(next: Stroke[]) {
  onStrokesChanged(next)
}

const showCanvas = computed(() => stage.value === 'trace' || stage.value === 'freewrite')
const canvasMode = computed(() => (stage.value === 'trace' ? ('trace' as const) : ('free' as const)))

const nextCharacterId = computed(() => {
  const chars = lesson.value?.characters ?? []
  const idx = chars.findIndex((c) => c.id === characterId.value)
  if (idx < 0 || idx >= chars.length - 1) return null
  return chars[idx + 1]?.id ?? null
})

const alreadyPassed = computed(() => progress.value?.status === 'passed')
</script>

<template>
  <div class="page">
    <header class="top">
      <router-link to="/practice" class="quiet">← Lesson</router-link>
      <router-link to="/" class="quiet">Free board</router-link>
    </header>

    <JourneyStatusBanner :stage="stage" :banner="banner" :soft-warn="softWarn" />

    <template v-if="stage === 'error'">
      <JourneyActions stage="error">
        <template #error-actions>
          <button type="button" class="primary" @click="load()">Retry load</button>
        </template>
      </JourneyActions>
    </template>

    <template v-else-if="character">
      <CharacterIntroPanel
        v-if="stage === 'intro' || stage === 'animate'"
        :character="character"
      />

      <StrokeOrderPlayer v-if="stage === 'animate'" :glyph="character.glyph" />

      <PracticeStageCanvas
        v-if="showCanvas"
        :mode="canvasMode"
        :glyph="character.glyph"
        :strokes="strokes"
        @update:strokes="onStrokesUpdate"
      />

      <ComparisonOverlay
        v-if="stage === 'result' && assessment"
        :glyph="character.glyph"
        :learner-strokes="strokes"
        :pass="assessment.pass"
        :score="assessment.score"
        :feedback="feedback"
      />

      <p v-if="stage === 'complete'" class="done">Completed.</p>

      <JourneyActions
        :stage="stage"
        :can-next-trace="canNextFromTrace"
        :pass="assessment?.pass ?? null"
        :already-passed="alreadyPassed"
        @start="start()"
        @continue-animate="continueFromAnimate()"
        @next-trace="nextFromTrace()"
        @show-order="showOrderAgain()"
        @submit="submit()"
        @retry="retry()"
        @retry-assess="retryAssess()"
        @done="complete()"
        @cancel="cancelPractice()"
      >
        <template #complete-links>
          <router-link
            v-if="nextCharacterId"
            class="primary-link"
            :to="`/practice/${encodeURIComponent(nextCharacterId)}`"
          >
            Next character
          </router-link>
          <router-link to="/practice" class="quiet">Back to lesson</router-link>
        </template>
      </JourneyActions>
    </template>
  </div>
</template>

<style scoped>
.page {
  max-width: 40rem;
  margin: 0 auto;
  padding: 12px 16px 32px;
}
.top {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}
.quiet {
  color: #475569;
  text-decoration: none;
  font-size: 0.95rem;
}
.quiet:hover {
  text-decoration: underline;
}
.done {
  text-align: center;
  margin: 24px 0 8px;
}
.primary {
  padding: 8px 14px;
  border: 1px solid #1e293b;
  background: #1e293b;
  color: #f8fafc;
  border-radius: 4px;
  cursor: pointer;
}
.primary-link {
  display: inline-block;
  padding: 8px 14px;
  background: #1e293b;
  color: #f8fafc;
  border-radius: 4px;
  text-decoration: none;
}
</style>
