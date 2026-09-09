<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { readLogicalCanvasSize } from '../../canvas/layout'
import { drawTraceTemplates, hiragana5Traces } from '../../curriculum/hiragana5'
import { useLocale } from '../../composables/useLocale'
import { usePracticeCanvas } from '../../composables/usePracticeCanvas'
import type { Point, Stroke } from '../../services/strokeSync'

const props = defineProps<{
  mode: 'trace' | 'free' | 'readonly'
  glyph: string
  strokes: Stroke[]
  enabled?: boolean
}>()

const emit = defineEmits<{
  'update:strokes': [Stroke[]]
}>()

const { t } = useLocale()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const guideRef = ref<HTMLCanvasElement | null>(null)
const frameRef = ref<HTMLElement | null>(null)
const tool = ref<'pencil' | 'eraser'>('pencil')
const color = ref('#111827')
const width = ref(3)
const syncStatus = ref<'connecting' | 'saving' | 'saved' | 'offline' | 'error'>('saved')

const strokesRef = ref<Stroke[]>([...props.strokes])
watch(
  () => props.strokes,
  (next) => {
    strokesRef.value = next
  },
  { deep: true },
)

const inputEnabled = computed(
  () => props.enabled !== false && (props.mode === 'trace' || props.mode === 'free'),
)

const canvasLabel = computed(() => {
  if (props.mode === 'trace') return t('canvas.trace')
  if (props.mode === 'free') return t('canvas.free')
  return t('canvas.result')
})

function newClientId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `local-${Date.now()}`
}

const practice = usePracticeCanvas({
  canvasRef,
  enabled: inputEnabled,
  strokes: strokesRef,
  tool,
  color,
  width,
  syncStatus,
  onStrokeComplete(points: Point[]) {
    const next = [
      ...strokesRef.value,
      {
        points,
        color: color.value,
        width: width.value,
        clientId: newClientId(),
        startedAtUnixMs: Date.now(),
        sync: 'saved' as const,
      },
    ]
    strokesRef.value = next
    emit('update:strokes', next)
  },
  onEraseAt() {},
  onCancelInProgress() {},
})

function paintGuides() {
  const guide = guideRef.value
  if (!guide) return
  const ctx = guide.getContext('2d')
  if (!ctx) return
  const logical =
    readLogicalCanvasSize(guide) ??
    readLogicalCanvasSize(canvasRef.value) ?? { width: 300, height: 300 }
  const cssW = logical.width
  const cssH = logical.height
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1
  guide.width = Math.round(cssW * dpr)
  guide.height = Math.round(cssH * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, cssW, cssH)
  if (props.mode === 'trace') {
    drawTraceTemplates(ctx, hiragana5Traces, props.glyph, cssW, cssH)
  }
}

let ro: ResizeObserver | null = null

onMounted(() => {
  paintGuides()
  if (typeof ResizeObserver !== 'undefined' && frameRef.value) {
    ro = new ResizeObserver(() => {
      paintGuides()
      practice.resync()
    })
    ro.observe(frameRef.value)
  }
})

onUnmounted(() => {
  ro?.disconnect()
})

watch(
  [() => props.mode, () => props.glyph, guideRef],
  () => paintGuides(),
  { immediate: true },
)

function clear() {
  if (!inputEnabled.value) return
  strokesRef.value = []
  emit('update:strokes', [])
}

function undo() {
  if (!inputEnabled.value || strokesRef.value.length === 0) return
  const next = strokesRef.value.slice(0, -1)
  strokesRef.value = next
  emit('update:strokes', next)
}

defineExpose({
  clear,
  undo,
  canvasRef,
  getLogicalSize: () => practice.getLogicalSize(),
})
</script>

<template>
  <div class="stage">
    <div ref="frameRef" class="frame">
      <canvas ref="guideRef" class="guide" aria-hidden="true" />
      <canvas ref="canvasRef" class="ink" :aria-label="canvasLabel" />
    </div>
    <div v-if="inputEnabled" class="tools">
      <button type="button" @click="undo">{{ t('action.undo') }}</button>
      <button type="button" @click="clear">{{ t('action.clear') }}</button>
    </div>
  </div>
</template>

<style scoped>
.stage {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}

.frame {
  position: relative;
  width: var(--canvas-size);
  height: var(--canvas-size);
}

.guide,
.ink {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  border-radius: var(--radius-sm);
}

.guide {
  border: 2px solid var(--ink);
  background: var(--paper-raised);
  pointer-events: none;
}

.ink {
  background: transparent;
  touch-action: none;
  border: 2px solid transparent;
}

.tools {
  display: flex;
  gap: var(--space-2);
}

.tools button {
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  border: 1px solid var(--rule);
  background: var(--paper-raised);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font: inherit;
  color: var(--ink);
}
</style>
