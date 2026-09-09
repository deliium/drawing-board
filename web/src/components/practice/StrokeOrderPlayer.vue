<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { hiragana5Traces, type TracePoint } from '../../curriculum/hiragana5'

const props = defineProps<{
  glyph: string
  playing?: boolean
}>()

const emit = defineEmits<{
  done: []
}>()

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function orderDebug(...args: unknown[]) {
  if (isDev) console.debug('[strokeOrder]', ...args)
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const strokeIndex = ref(0)
const reducedMotion = ref(false)
const cssSize = 280

const entry = computed(() => hiragana5Traces.characters.find((c) => c.glyph === props.glyph))

let raf = 0
let cancelled = false

function prefersReduced(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function paint(upTo: number, progressWithin = 1) {
  const canvas = canvasRef.value
  const strokes = entry.value?.strokes
  if (!canvas || !strokes) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1
  canvas.width = Math.round(cssSize * dpr)
  canvas.height = Math.round(cssSize * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, cssSize, cssSize)

  ctx.strokeStyle = 'rgba(51, 65, 85, 0.95)'
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
    ctx.moveTo(pts[0].x * cssSize, pts[0].y * cssSize)
    for (let j = 1; j < pts.length; j++) {
      ctx.lineTo(pts[j].x * cssSize, pts[j].y * cssSize)
    }
    ctx.stroke()
  }
}

function sliceStroke(stroke: TracePoint[], t: number): TracePoint[] {
  if (stroke.length <= 1) return stroke
  const target = Math.max(1, Math.ceil(stroke.length * Math.min(1, Math.max(0, t))))
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

onMounted(() => {
  reducedMotion.value = prefersReduced()
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
})

defineExpose({ skip, paintFinal })
</script>

<template>
  <div class="player">
    <canvas
      ref="canvasRef"
      class="canvas"
      :width="cssSize"
      :height="cssSize"
      :aria-label="`Stroke order for ${glyph}`"
    />
    <p v-if="reducedMotion" class="hint">Motion reduced — showing final strokes. You can continue.</p>
    <button type="button" class="linkish" @click="skip">Skip</button>
  </div>
</template>

<style scoped>
.player {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.canvas {
  width: 280px;
  height: 280px;
  background: #fff;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
}
.hint {
  margin: 0;
  font-size: 0.9rem;
  opacity: 0.75;
}
.linkish {
  background: none;
  border: none;
  color: #334155;
  text-decoration: underline;
  cursor: pointer;
  padding: 4px;
}
</style>
