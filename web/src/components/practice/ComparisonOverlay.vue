<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { announce } from '../../a11y/announce'
import { readLogicalCanvasSize } from '../../canvas/layout'
import { drawStrokes } from '../../canvas/drawStrokes'
import { drawTraceTemplates, hiragana5Traces } from '../../curriculum/hiragana5'
import { useLocale } from '../../composables/useLocale'
import { formatCorrectionDisplay } from '../../i18n/corrections'
import type { AssessmentFeedback } from '../../services/attemptsApi'
import type { Stroke } from '../../services/strokeSync'

const props = withDefaults(
  defineProps<{
    glyph: string
    learnerStrokes: Stroke[]
    pass: boolean
    score: number
    feedback: AssessmentFeedback[]
    focusOnMount?: boolean
  }>(),
  {
    feedback: () => [],
    focusOnMount: true,
  },
)

const { t, locale } = useLocale()

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function overlayDebug(...args: unknown[]) {
  if (isDev) console.debug('[compareOverlay]', ...args)
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const headingRef = ref<HTMLHeadingElement | null>(null)

function paint() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const logical = readLogicalCanvasSize(canvas) ?? { width: 300, height: 300 }
  const cssSize = Math.min(logical.width, logical.height)
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1
  canvas.width = Math.round(cssSize * dpr)
  canvas.height = Math.round(cssSize * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, cssSize, cssSize)

  drawTraceTemplates(ctx, hiragana5Traces, props.glyph, cssSize, cssSize)
  drawStrokes(ctx, props.learnerStrokes, { cssWidth: cssSize, cssHeight: cssSize, clear: false })

  const template = hiragana5Traces.characters.find((c) => c.glyph === props.glyph)
  overlayDebug('strokes', {
    learner: props.learnerStrokes.length,
    template: template?.strokes.length ?? 0,
  })
}

onMounted(async () => {
  paint()
  if (props.focusOnMount) {
    await nextTick()
    headingRef.value?.focus()
    const summary = [
      props.pass ? t('result.pass') : t('result.fail'),
      t('result.match', { score: props.score.toFixed(2) }),
      ...visibleFeedback.value.map((f) => f.text),
    ].join('. ')
    announce(summary, 'polite')
  }
})

watch(
  () => [props.glyph, props.learnerStrokes],
  () => paint(),
  { deep: true },
)

watch(locale, () => {
  // Re-announce is unnecessary; display updates via computed.
})

const visibleFeedback = computed(() => {
  void locale.value
  return props.feedback.slice(0, 2).map((f) => ({
    ...f,
    text: formatCorrectionDisplay(f.code, f.message, { glyph: props.glyph }),
  })).filter((f) => f.text.trim())
})

const passLabel = computed(() => (props.pass ? t('result.pass') : t('result.fail')))
const matchLabel = computed(() => t('result.match', { score: props.score.toFixed(2) }))
</script>

<template>
  <section class="overlay" :aria-label="t('result.compareAria')">
    <canvas ref="canvasRef" class="canvas" aria-hidden="true" />
    <div
      class="summary"
      role="status"
      aria-live="polite"
      aria-atomic="true"
      :aria-label="t('result.aria')"
    >
      <h2 ref="headingRef" class="heading" tabindex="-1">{{ t('result.heading') }}</h2>
      <p class="verdict">
        {{ passLabel }}
        · {{ matchLabel }}
      </p>
      <ul v-if="visibleFeedback.length" class="feedback">
        <li v-for="item in visibleFeedback" :key="`${item.rank}-${item.code}`">
          {{ item.text }}
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
}

.canvas {
  width: var(--canvas-size);
  height: var(--canvas-size);
  background: var(--paper-raised);
  border: 1px solid var(--ink-muted);
  border-radius: var(--radius-sm);
}

.summary {
  text-align: center;
  max-width: 28rem;
  width: 100%;
}

.heading {
  margin: 0 0 var(--space-2);
  font-size: 1.15rem;
  font-weight: 600;
}

.verdict {
  margin: 0;
}

.feedback {
  list-style: disc;
  text-align: left;
  margin: var(--space-2) auto 0;
  padding-left: 1.25rem;
}
</style>
