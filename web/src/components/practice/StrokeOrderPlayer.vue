<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { readLogicalCanvasSize } from '../../canvas/layout'
import { useLocale } from '../../composables/useLocale'
import { hiragana5Traces, type TracePoint } from '../../curriculum/hiragana5'

const props = defineProps<{
  glyph: string
  playing?: boolean
}>()

const emit = defineEmits<{
  done: []
}>()

const { t } = useLocale()

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function orderDebug(...args: unknown[]) {
  if (isDev) console.debug('[strokeOrder]', ...args)
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const strokeIndex = ref(0)
const reducedMotion = ref(false)

const entry = computed(() => hiragana5Traces.characters.find((c) => c.glyph === props.glyph))
const ariaLabel = computed(() => t('canvas.strokeOrder', { glyph: props.glyph }))

let raf = 0
let cancelled = false
let media: MediaQueryList | null = null

function prefersReduced(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function cssSize(): number {
  const logical = readLogicalCanvasSize(canvasRef.value)
  if (logical) return Math.min(logical.width, logical.height)
  return 300
}

function paint(upTo: number, progressWithin = 1) {
  const canvas = canvasRef.value
  const strokes = entry.value?.strokes
  if (!canvas || !strokes) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const size = cssSize()
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1
  canvas.width = Math.round(size * dpr)
  canvas.height = Math.round(size * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, size, size)

  ctx.strokeStyle = 'rgba(30, 41, 59, 0.95)'
  ctx.lineWidth = 3
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  for (let i = 0; i < strokes.length; i++) {
    if (i > upTo) break
    const stroke = strokes[i]
    if (!stroke.length) continue
    const complete = i < upTo || progressWithin >= 1
    const pts = complete ? stroke : sliceStroke(stroke, progressWithin)
    if (!pts.length) continue
    ctx.beginPath()
    ctx.moveTo(pts[0].x * size, pts[0].y * size)
    for (let j = 1; j < pts.length; j++) {
      ctx.lineTo(pts[j].x * size, pts[j].y * size)
    }
    ctx.stroke()
  }
}

function sliceStroke(stroke: TracePoint[], tFrac: number): TracePoint[] {
  if (stroke.length <= 1) return stroke
  const target = Math.max(1, Math.ceil(stroke.length * Math.min(1, Math.max(0, tFrac))))
  return stroke.slice(0, target)
}

function paintFinal() {
  const n = entry.value?.strokes.length ?? 0
  strokeIndex.value = Math.max(0, n - 1)
  paint(n - 1, 1)
  orderDebug(props.glyph, 'final frame', n)
}

async function runAnimation() {
  cancelled = false
  const strokes = entry.value?.strokes ?? []
  if (!strokes.length) {
    emit('done')
    return
  }
  if (reducedMotion.value || props.playing === false) {
    paintFinal()
    emit('done')
    return
  }

  for (let i = 0; i < strokes.length; i++) {
    if (cancelled) return
    strokeIndex.value = i
    orderDebug(props.glyph, 'play stroke', i)
    const steps = 18
    for (let s = 1; s <= steps; s++) {
      if (cancelled) return
      paint(i, s / steps)
      await new Promise<void>((resolve) => {
        raf = requestAnimationFrame(() => resolve())
      })
      await new Promise((r) => setTimeout(r, 16))
    }
    await new Promise((r) => setTimeout(r, 220))
  }
  emit('done')
}

function skip() {
  cancelled = true
  if (raf) cancelAnimationFrame(raf)
  paintFinal()
  emit('done')
}

function onMotionChange() {
  reducedMotion.value = prefersReduced()
  if (reducedMotion.value) {
    cancelled = true
    paintFinal()
    emit('done')
  }
}

onMounted(() => {
  reducedMotion.value = prefersReduced()
  if (typeof window !== 'undefined' && window.matchMedia) {
    media = window.matchMedia('(prefers-reduced-motion: reduce)')
    media.addEventListener?.('change', onMotionChange)
    media.addListener?.(onMotionChange)
  }
  void runAnimation()
})

watch(
  () => props.glyph,
  () => {
    cancelled = true
    void runAnimation()
  },
)

onUnmounted(() => {
  cancelled = true
  if (raf) cancelAnimationFrame(raf)
  if (media) {
    media.removeEventListener?.('change', onMotionChange)
    media.removeListener?.(onMotionChange)
  }
})

defineExpose({ skip, paintFinal, reducedMotion })
</script>

<template>
  <div class="player">
    <canvas ref="canvasRef" class="canvas" :aria-label="ariaLabel" />
    <p v-if="reducedMotion" class="hint">{{ t('strokeOrder.reduced') }}</p>
    <button type="button" class="linkish" @click="skip">{{ t('strokeOrder.skip') }}</button>
  </div>
</template>

<style scoped>
.player {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}

.canvas {
  width: var(--canvas-size);
  height: var(--canvas-size);
  background: var(--paper-raised);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
}

.hint {
  margin: 0;
  font-size: 0.9rem;
  color: var(--ink-muted);
}

.linkish {
  background: none;
  border: none;
  color: var(--ink-muted);
  text-decoration: underline;
  cursor: pointer;
  min-height: var(--touch-min);
  padding: var(--space-2) var(--space-3);
  font: inherit;
}
</style>
