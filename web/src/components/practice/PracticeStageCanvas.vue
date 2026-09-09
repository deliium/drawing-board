<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { drawTraceTemplates, hiragana5Traces } from '../../curriculum/hiragana5'
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

const canvasRef = ref<HTMLCanvasElement | null>(null)
const guideRef = ref<HTMLCanvasElement | null>(null)
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

function newClientId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `local-${Date.now()}`
}

usePracticeCanvas({
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
  const cssW = 300
  const cssH = 300
  const dpr = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1
  guide.width = Math.round(cssW * dpr)
  guide.height = Math.round(cssH * dpr)
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, cssW, cssH)
  if (props.mode === 'trace') {
    drawTraceTemplates(ctx, hiragana5Traces, props.glyph, cssW, cssH)
  }
}

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

defineExpose({ clear, undo, canvasRef })
</script>

<template>
  <div class="stage">
    <div class="frame">
      <canvas
        ref="guideRef"
        class="guide"
        width="300"
        height="300"
        aria-hidden="true"
      />
      <canvas
        ref="canvasRef"
        class="ink"
        width="300"
        height="300"
        :aria-label="
          mode === 'trace'
            ? 'Trace practice canvas'
            : mode === 'free'
              ? 'Free-write canvas'
              : 'Result canvas'
        "
      />
    </div>
    <div v-if="inputEnabled" class="tools">
      <button type="button" @click="undo">Undo</button>
      <button type="button" @click="clear">Clear</button>
    </div>
  </div>
</template>

<style scoped>
.stage {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.frame {
  position: relative;
  width: 300px;
  height: 300px;
}
.guide,
.ink {
  position: absolute;
  inset: 0;
  width: 300px;
  height: 300px;
  border-radius: 6px;
}
.guide {
  border: 2px solid #334155;
  background: #fff;
  pointer-events: none;
}
.ink {
  background: transparent;
  touch-action: none;
  border: 2px solid transparent;
}
.tools {
  display: flex;
  gap: 8px;
}
.tools button {
  padding: 6px 12px;
  border: 1px solid #cbd5e1;
  background: #f8fafc;
  border-radius: 4px;
  cursor: pointer;
}
</style>
