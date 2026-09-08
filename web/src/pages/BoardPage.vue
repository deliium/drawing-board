<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { apiFetch } from '../services/apiClient'
import { trackMetric } from '../services/migrationHealth'
import {
  createWsClient,
  type InboundAppMessage,
  type SyncStatus,
} from '../services/wsClient'
import { applyIncomingMessage, type Point, type Stroke } from '../services/strokeSync'
import { sessionContext, setAuthenticatedUser } from '../services/sessionContext'

type Candidate = { text: string; score: number }

const router = useRouter()

const wsDebug =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function boardDebug(...args: unknown[]) {
  if (wsDebug) console.debug('[BoardPage.ws]', ...args)
}

function newOpId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `op-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

const statusLabels: Record<SyncStatus, string> = {
  connecting: 'Connecting…',
  saving: 'Saving…',
  saved: 'Saved',
  offline: 'Offline — retrying…',
  error: 'Sync error',
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const color = ref('#1d4ed8')
const width = ref(4)
const tool = ref<'pencil' | 'eraser'>('pencil')
const candidates = ref<Candidate[] | null>(null)
const strokes = ref<Stroke[]>([])
const syncStatus = ref<SyncStatus>('connecting')
const user = computed(() => sessionContext.user)
const clientId = Math.random().toString(36).slice(2)
const statusLabel = computed(() => statusLabels[syncStatus.value])

const ws = createWsClient({
  onMessage: (msg: InboundAppMessage) => {
    trackMetric('ws.message', 1)
    handleIncoming(msg)
  },
  onStatusChange: (status) => {
    syncStatus.value = status
  },
  onQueueReject: (reason) => {
    boardDebug('queue reject', reason)
    syncStatus.value = 'error'
  },
})

function handleIncoming(message: InboundAppMessage) {
  const result = applyIncomingMessage(strokes.value, message)
  if (
    result.action === 'ignored-foreign-stroke' ||
    result.action === 'ignored-unknown-delete' ||
    result.action === 'ignored-duplicate-ack'
  ) {
    boardDebug('ignore inbound', result.action, result.reason)
    return
  }
  boardDebug('apply inbound', result.action)
  strokes.value = result.strokes
}

async function loadStrokes() {
  if (!user.value) return
  ws.clearPending()
  try {
    const rows = await apiFetch<Stroke[]>('/api/strokes')
    strokes.value = rows.map((s) => ({ ...s, sync: 'saved' as const }))
  } catch {
    strokes.value = []
  }
}

async function doLogout() {
  try {
    await apiFetch('/api/logout', { method: 'POST' })
  } catch {
    // still clear local session so the learner can reach the auth page
  }
  ws.clearPending()
  ws.close()
  setAuthenticatedUser(null)
  strokes.value = []
  candidates.value = null
  if (wsDebug) console.debug('[BoardPage] logout')
  await router.replace({ name: 'login' })
}

async function doClear() {
  await apiFetch('/api/strokes/clear', { method: 'POST' })
  ws.clearPending()
  strokes.value = []
}

async function recognize() {
  const canvas = canvasRef.value
  if (!canvas) return
  try {
    const result = await apiFetch<{ candidates: Candidate[] }>('/api/recognize', {
      method: 'POST',
      body: JSON.stringify({ topN: 10, width: canvas.width, height: canvas.height }),
    })
    candidates.value = result.candidates || []
    trackMetric('recognize.success', 1)
  } catch (err) {
    candidates.value = []
    trackMetric('recognize.reject', 1)
    if (wsDebug) {
      const msg = err instanceof Error ? err.message : 'recognize failed'
      console.debug('[BoardPage] recognize rejected:', msg)
    }
  }
}

function hitTest(p: Point, s: Stroke): boolean {
  const threshold = Math.max(6, s.width + 4)
  for (let i = 0; i < s.points.length - 1; i += 1) {
    const a = s.points[i]
    const b = s.points[i + 1]
    const dx = b.x - a.x
    const dy = b.y - a.y
    const len2 = dx * dx + dy * dy
    if (len2 === 0) continue
    const t = Math.max(0, Math.min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / len2))
    const cx = a.x + t * dx
    const cy = a.y + t * dy
    if (Math.hypot(p.x - cx, p.y - cy) <= threshold) return true
  }
  return false
}

function enqueueDelete(strokeId: number) {
  const opId = newOpId()
  ws.send({ type: 'delete', opId, delete: strokeId })
}

function doUndo() {
  if (!strokes.value.length) return
  const last = strokes.value[strokes.value.length - 1]
  strokes.value = strokes.value.slice(0, -1)
  if (last.opId && (!last.id || last.sync === 'pending' || last.sync === 'saving')) {
    ws.dropOp(last.opId)
  }
  if (last.id && last.id > 0) {
    enqueueDelete(last.id)
  }
}

function syncCanvasSize() {
  const canvas = canvasRef.value
  if (!canvas) return
  const rect = canvas.getBoundingClientRect()
  const targetWidth = Math.round(rect.width)
  const targetHeight = Math.round(rect.height)
  if (canvas.width !== targetWidth || canvas.height !== targetHeight) {
    canvas.width = targetWidth
    canvas.height = targetHeight
  }
}

watch(
  strokes,
  (next) => {
    const canvas = canvasRef.value
    if (!canvas) return
    const ctx = canvas.getContext('2d', { willReadFrequently: true })
    if (!ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    for (const s of next) {
      if (s.points.length < 2) continue
      ctx.strokeStyle = s.color
      ctx.lineWidth = s.width
      ctx.lineCap = 'round'
      ctx.lineJoin = 'round'
      ctx.beginPath()
      ctx.moveTo(s.points[0].x, s.points[0].y)
      for (let i = 1; i < s.points.length; i += 1) ctx.lineTo(s.points[i].x, s.points[i].y)
      ctx.stroke()
    }
  },
  { deep: true },
)

function setupDrawing() {
  const canvas = canvasRef.value
  if (!canvas || !user.value) return () => {}
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) return () => {}
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  const rect = () => canvas.getBoundingClientRect()
  let drawing = false
  let points: Point[] = []
  const toPoint = (e: PointerEvent): Point => ({
    x: e.clientX - rect().left,
    y: e.clientY - rect().top,
  })

  const onDown = (e: PointerEvent) => {
    const p = toPoint(e)
    if (tool.value === 'eraser') {
      const target = [...strokes.value].reverse().find((s) => hitTest(p, s))
      if (target?.id) {
        strokes.value = strokes.value.filter((st) => st.id !== target.id)
        enqueueDelete(target.id)
      }
      return
    }
    drawing = true
    points = [p]
    ctx.strokeStyle = color.value
    ctx.lineWidth = width.value
    ctx.beginPath()
    ctx.moveTo(p.x, p.y)
    canvas.setPointerCapture(e.pointerId)
  }

  const onMove = (e: PointerEvent) => {
    if (!drawing || tool.value !== 'pencil') return
    const p = toPoint(e)
    points.push(p)
    ctx.lineTo(p.x, p.y)
    ctx.stroke()
  }

  const onUp = () => {
    if (!drawing || tool.value !== 'pencil') {
      drawing = false
      points = []
      return
    }
    drawing = false
    if (points.length >= 2) {
      const opId = newOpId()
      const stroke: Stroke = {
        points: [...points],
        color: color.value,
        width: width.value,
        clientId,
        startedAtUnixMs: Date.now(),
        opId,
        sync: 'pending',
      }
      const queued = ws.send({
        type: 'stroke',
        opId,
        stroke: {
          points: stroke.points,
          color: stroke.color,
          width: stroke.width,
          clientId: stroke.clientId,
          startedAtUnixMs: stroke.startedAtUnixMs,
          opId,
        },
      })
      if (!queued) {
        boardDebug('stroke not queued; queue full')
        return
      }
      strokes.value = [...strokes.value, { ...stroke, sync: 'saving' }]
    }
    points = []
    ctx.closePath()
  }

  canvas.addEventListener('pointerdown', onDown)
  canvas.addEventListener('pointermove', onMove)
  canvas.addEventListener('pointerup', onUp)
  canvas.addEventListener('pointercancel', onUp)

  return () => {
    canvas.removeEventListener('pointerdown', onDown)
    canvas.removeEventListener('pointermove', onMove)
    canvas.removeEventListener('pointerup', onUp)
    canvas.removeEventListener('pointercancel', onUp)
  }
}

onMounted(() => {
  loadStrokes()
  syncCanvasSize()
  const onResize = () => syncCanvasSize()
  window.addEventListener('resize', onResize)
  onUnmounted(() => window.removeEventListener('resize', onResize))
})

watch(
  user,
  (next) => {
    if (!next) {
      ws.clearPending()
      ws.close()
      return
    }
    const wsUrl =
      location.port === '5173'
        ? `ws://${location.hostname}:5173/ws`
        : `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`
    boardDebug('connect session', wsUrl)
    ws.connect(wsUrl)
    loadStrokes()
  },
  { immediate: true },
)

watch(
  [canvasRef, user],
  () => {
    const cleanup = setupDrawing()
    return cleanup
  },
  { immediate: true },
)

onMounted(() => {
  const onKeyDown = (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z' && !e.shiftKey) {
      e.preventDefault()
      doUndo()
    }
  }
  window.addEventListener('keydown', onKeyDown)
  onUnmounted(() => window.removeEventListener('keydown', onKeyDown))
})
</script>

<template>
  <div style="height: 100vh; display: grid; grid-template-rows: auto auto 1fr">
    <header style="padding: 12px; display: flex; gap: 12px; align-items: center">
      <b>Japanese Handwriting Practice</b>
      <label>
        Color
        <input v-model="color" type="color" :disabled="tool !== 'pencil'" />
      </label>
      <label>
        Width
        <input v-model="width" type="range" min="1" max="20" />
      </label>
      <button :disabled="tool === 'pencil'" @click="tool = 'pencil'">Pencil</button>
      <button :disabled="tool === 'eraser'" @click="tool = 'eraser'">Eraser</button>
      <button :disabled="strokes.length === 0" @click="doUndo">Undo</button>
      <span style="margin-left: auto; opacity: 0.7" :data-sync-status="syncStatus">
        {{ statusLabel }}
      </span>
      <button @click="doClear">Clear</button>
      <button @click="doLogout">Logout</button>
    </header>

    <div style="padding: 12px; display: flex; gap: 8px; align-items: center">
      <button @click="recognize">Recognize</button>
      <div v-if="candidates && candidates.length > 0" style="display: flex; gap: 8px; flex-wrap: wrap">
        <span
          v-for="(c, i) in candidates"
          :key="i"
          :title="`${c.score}`"
          style="padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px"
        >
          {{ c.text }}
        </span>
      </div>
      <span v-else-if="candidates && candidates.length === 0">No candidates</span>
    </div>

    <div style="position: relative; display: flex; justify-content: center; align-items: center; padding: 20px">
      <canvas
        ref="canvasRef"
        style="
          width: 300px;
          height: 300px;
          touch-action: none;
          background: #fff;
          border: 2px solid #333;
          border-radius: 8px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
        "
      />
    </div>
  </div>
</template>
