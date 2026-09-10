<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { readLogicalCanvasSize } from '../../canvas/layout'
import { pathSmoothPolyline, slicePolylineByFraction } from '../../canvas/smoothPath'
import { useLocale } from '../../composables/useLocale'
import { hiragana5Traces } from '../../curriculum/hiragana5'

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

function fixLog(...args: unknown[]) {
  console.info('[FIX]', '[strokeOrder]', ...args)
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const frameRef = ref<HTMLElement | null>(null)
const strokeIndex = ref(0)
const reducedMotion = ref(false)
const missingTrace = ref(false)

const entry = computed(() => hiragana5Traces.characters.find((c) => c.glyph === props.glyph))
const ariaLabel = computed(() => t('canvas.strokeOrder', { glyph: props.glyph }))

let raf = 0
let cancelled = false
let media: MediaQueryList | null = null
let ro: ResizeObserver | null = null

/** Last painted frame — ResizeObserver redraws this without resetting animation. */
let paintUpTo = -1
let paintProgress = 1

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
  paintUpTo = upTo
  paintProgress = progressWithin

  const canvas = canvasRef.value
  const strokes = entry.value?.strokes
  if (!canvas || !strokes || !strokes.length) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const size = cssSize()
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1
  canvas.width = Math.round(size * dpr)
  canvas.height = Math.round(size * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, size, size)

  orderDebug('paint', {
    glyph: props.glyph,
    cssW: size,
    cssH: size,
    dpr,
    strokeCount: strokes.length,
    upTo,
    progressWithin,
  })

  ctx.strokeStyle = 'rgba(30, 41, 59, 0.95)'
  ctx.lineWidth = 3
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  for (let i = 0; i < strokes.length; i++) {
    if (i > upTo) break
    const stroke = strokes[i]
    if (!stroke.length) continue
    const complete = i < upTo || progressWithin >= 1
    const pts = complete ? stroke : slicePolylineByFraction(stroke, progressWithin)
    if (!pts.length) continue
    const scaled = pts.map((p) => ({ x: p.x * size, y: p.y * size }))
    pathSmoothPolyline(ctx, scaled)
    ctx.stroke()
  }

  // 1-based order indices so order remains readable on final / reduced-motion frames.
  const labelR = Math.max(8, size * 0.028)
  ctx.font = `600 ${Math.max(12, Math.round(size * 0.045))}px var(--font-ui, sans-serif)`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  for (let i = 0; i < strokes.length; i++) {
    if (i > upTo) break
    const stroke = strokes[i]
    if (!stroke.length) continue
    const started = i < upTo || progressWithin > 0.05
    if (!started) continue
    const sx = stroke[0].x * size
    const sy = stroke[0].y * size
    const lx = Math.min(size - labelR - 2, Math.max(labelR + 2, sx))
    const ly = Math.min(size - labelR - 2, Math.max(labelR + 2, sy))
    ctx.beginPath()
    ctx.fillStyle = 'rgba(255, 255, 255, 0.92)'
    ctx.arc(lx, ly, labelR, 0, Math.PI * 2)
    ctx.fill()
    ctx.strokeStyle = 'rgba(30, 41, 59, 0.85)'
    ctx.lineWidth = 1.5
    ctx.stroke()
    ctx.fillStyle = 'rgba(30, 41, 59, 0.95)'
    ctx.fillText(String(i + 1), lx, ly)
  }
}

function repaintCurrentFrame() {
  if (paintUpTo < 0) return
  paint(paintUpTo, paintProgress)
}

function paintFinal() {
  const n = entry.value?.strokes.length ?? 0
  strokeIndex.value = Math.max(0, n - 1)
  paint(n - 1, 1)
  orderDebug(props.glyph, 'final frame', n)
}

async function runAnimation() {
  cancelled = false
  missingTrace.value = false
  paintUpTo = -1
  paintProgress = 1

  const strokes = entry.value?.strokes ?? []
  if (!strokes.length) {
    missingTrace.value = true
    fixLog('missing trace entry', {
      glyph: props.glyph,
      contentVersion: hiragana5Traces.contentVersion,
    })
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
  if (entry.value?.strokes?.length) {
    paintFinal()
  }
  emit('done')
}

function onMotionChange() {
  reducedMotion.value = prefersReduced()
  if (reducedMotion.value) {
    cancelled = true
    if (entry.value?.strokes?.length) {
      paintFinal()
    }
    emit('done')
  }
}

onMounted(async () => {
  reducedMotion.value = prefersReduced()
  if (typeof window !== 'undefined' && window.matchMedia) {
    media = window.matchMedia('(prefers-reduced-motion: reduce)')
    media.addEventListener?.('change', onMotionChange)
    media.addListener?.(onMotionChange)
  }
  await nextTick()
  if (typeof ResizeObserver !== 'undefined' && frameRef.value) {
    ro = new ResizeObserver(() => {
      repaintCurrentFrame()
    })
    ro.observe(frameRef.value)
  }
  fixLog('mount', { glyph: props.glyph, reducedMotion: reducedMotion.value })
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
  ro?.disconnect()
  ro = null
  if (media) {
    media.removeEventListener?.('change', onMotionChange)
    media.removeListener?.(onMotionChange)
  }
})

defineExpose({ skip, paintFinal, reducedMotion, missingTrace, repaintCurrentFrame })
</script>

<template>
  <div ref="frameRef" class="player">
    <canvas ref="canvasRef" class="canvas" :aria-label="ariaLabel" />
    <p v-if="missingTrace" class="hint" role="status">{{ t('strokeOrder.missing', { glyph }) }}</p>
    <p v-else-if="reducedMotion" class="hint">{{ t('strokeOrder.reduced') }}</p>
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
