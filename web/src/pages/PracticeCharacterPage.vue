<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute } from 'vue-router'
import CharacterIntroPanel from '../components/practice/CharacterIntroPanel.vue'
import ComparisonOverlay from '../components/practice/ComparisonOverlay.vue'
import JourneyActions from '../components/practice/JourneyActions.vue'
import JourneyStatusBanner from '../components/practice/JourneyStatusBanner.vue'
import PracticeStageCanvas from '../components/practice/PracticeStageCanvas.vue'
import StrokeOrderPlayer from '../components/practice/StrokeOrderPlayer.vue'
import { useLocale } from '../composables/useLocale'
import { usePracticeJourney } from '../composables/usePracticeJourney'
import type { Stroke } from '../services/strokeSync'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

const route = useRoute()
const { t } = useLocale()

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

const stageCanvasRef = ref<{
  canvasRef?: HTMLCanvasElement | { value: HTMLCanvasElement | null } | null
  getLogicalSize?: () => { width: number; height: number } | null
} | null>(null)

function readStageCanvasEl(): HTMLCanvasElement | null {
  const exposed = stageCanvasRef.value
  if (!exposed?.canvasRef) return null
  const c = exposed.canvasRef
  if (c instanceof HTMLCanvasElement) return c
  if (typeof c === 'object' && c && 'value' in c) return c.value
  return null
}

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
} = usePracticeJourney({
  characterId: characterIdRef,
  getLogicalSize: () => stageCanvasRef.value?.getLogicalSize?.() ?? null,
  getCanvasElement: () => readStageCanvasEl(),
})

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

const notDueYet = computed(() => {
  const dueAt = progress.value?.review?.dueAt
  if (!dueAt || progress.value?.review?.isDue) return false
  const tDue = Date.parse(dueAt)
  return !Number.isNaN(tDue) && tDue > Date.now()
})
</script>

<template>
  <div class="page">
    <header class="top">
      <router-link to="/practice" class="quiet">{{ t('practice.backLesson') }}</router-link>
      <router-link to="/" class="quiet">{{ t('nav.board') }}</router-link>
    </header>

    <JourneyStatusBanner :stage="stage" :banner="banner" :soft-warn="softWarn" />

    <template v-if="stage === 'error'">
      <JourneyActions stage="error">
        <template #error-actions>
          <button type="button" class="primary" @click="load()">{{ t('practice.retryLoad') }}</button>
        </template>
      </JourneyActions>
    </template>

    <template v-else-if="character">
      <CharacterIntroPanel
        v-if="stage === 'intro' || stage === 'animate'"
        :character="character"
      />
      <p v-if="(stage === 'intro' || stage === 'animate') && notDueYet" class="not-due">
        {{ t('practice.notDueYet') }}
      </p>

      <StrokeOrderPlayer v-if="stage === 'animate'" :glyph="character.glyph" />

      <PracticeStageCanvas
        v-if="showCanvas"
        ref="stageCanvasRef"
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

      <p v-if="stage === 'complete'" class="done">{{ t('practice.completed') }}</p>

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
            {{ t('practice.nextCharacter') }}
          </router-link>
          <router-link to="/practice" class="quiet">{{ t('practice.backToLesson') }}</router-link>
        </template>
      </JourneyActions>
    </template>
  </div>
</template>

<style scoped>
.page {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: var(--space-3) 0 var(--space-5);
}

.top {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--space-2);
  gap: var(--space-3);
  flex-wrap: wrap;
}

.quiet {
  color: var(--ink-muted);
  text-decoration: none;
  font-size: 0.95rem;
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
}

.quiet:hover {
  text-decoration: underline;
  color: var(--ink);
}

.done {
  text-align: center;
  margin: var(--space-5) 0 var(--space-2);
}

.not-due {
  color: var(--ink-muted);
  font-size: 0.9rem;
  margin: 0 0 var(--space-3);
}

.primary {
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  border: 1px solid var(--accent);
  background: var(--accent);
  color: #f8fafc;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font: inherit;
}

.primary-link {
  display: inline-flex;
  align-items: center;
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  background: var(--accent);
  color: #f8fafc;
  border-radius: var(--radius-sm);
  text-decoration: none;
  font-weight: 600;
}
</style>
