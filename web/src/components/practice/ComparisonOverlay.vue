<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { drawTraceTemplates, hiragana5Traces } from '../../curriculum/hiragana5'
import { drawStrokes } from '../../canvas/drawStrokes'
import type { AssessmentFeedback } from '../../services/attemptsApi'
import type { Stroke } from '../../services/strokeSync'

const props = withDefaults(
  defineProps<{
    glyph: string
    learnerStrokes: Stroke[]
    pass: boolean
    score: number
    feedback: AssessmentFeedback[]
  }>(),
  {
    feedback: () => [],
  },
)

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function overlayDebug(...args: unknown[]) {
  if (isDev) console.debug('[compareOverlay]', ...args)
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const cssSize = 300

function paint() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
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

onMounted(paint)
watch(
  () => [props.glyph, props.learnerStrokes],
  () => paint(),
  { deep: true },
)

const visibleFeedback = computed(() =>
  props.feedback.slice(0, 2).filter((f) => f.message?.trim()),
)
</script>

<template>
  <section class="overlay" aria-label="Comparison result">
    <canvas ref="canvasRef" class="canvas" :width="cssSize" :height="cssSize" />
    <div class="summary">
      <p>
        {{ pass ? 'Pass' : 'Not yet' }}
        · Match {{ score.toFixed(2) }}
      </p>
      <ul v-if="visibleFeedback.length" class="feedback">
        <li v-for="item in visibleFeedback" :key="`${item.rank}-${item.code}`">
          {{ item.message }}
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
  gap: 10px;
}
.canvas {
  width: 300px;
  height: 300px;
  background: #fff;
  border: 1px solid #94a3b8;
  border-radius: 6px;
}
.summary {
  text-align: center;
  max-width: 28rem;
}
.feedback {
  list-style: disc;
  text-align: left;
  margin: 8px auto 0;
  padding-left: 1.25rem;
}
</style>
