<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { apiFetch } from '../services/apiClient'
import { trackMetric } from '../services/migrationHealth'
import { createWsClient, type WsMessage } from '../services/wsClient'
import { applyIncomingMessage, type Point, type Stroke } from '../services/strokeSync'
import { sessionContext, setAuthenticatedUser } from '../services/sessionContext'

type Candidate = { text: string; score: number }

const wsDebug =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function boardDebug(...args: unknown[]) {
  if (wsDebug) console.debug('[BoardPage.ws]', ...args)
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const color = ref('#1d4ed8')
const width = ref(4)
const tool = ref<'pencil' | 'eraser'>('pencil')
const email = ref('')
const password = ref('')
const authErr = ref<string | null>(null)
const candidates = ref<Candidate[] | null>(null)
const strokes = ref<Stroke[]>([])
const wsReady = ref(false)
const user = computed(() => sessionContext.user)
const clientId = Math.random().toString(36).slice(2)

const ws = createWsClient(
  (msg: WsMessage) => {
    trackMetric('ws.message', 1)
    handleIncoming(msg)
  },
  (ready) => {
    wsReady.value = ready
  },
)

function handleIncoming(message: WsMessage) {
  const result = applyIncomingMessage(strokes.value, message as Parameters<typeof applyIncomingMessage>[1])
  if (result.action === 'ignored-foreign-stroke' || result.action === 'ignored-unknown-delete') {
    boardDebug('ignore inbound', result.action, result.reason)
    return
  }
  boardDebug('apply inbound', result.action)
  strokes.value = result.strokes
}

async function loadStrokes() {
  if (!user.value) return
  try {
    strokes.value = await apiFetch<Stroke[]>('/api/strokes')
  } catch {
    strokes.value = []
  }
}

async function doRegister() {
  authErr.value = null
  try {
    await apiFetch('/api/register', {
      method: 'POST',
      body: JSON.stringify({ email: email.value, password: password.value }),
    })
    password.value = ''
    const me = await apiFetch<{ id: number; email: string }>('/api/me')
    setAuthenticatedUser(me)
    await loadStrokes()
  } catch {
    authErr.value = 'Registration failed'
  }
}

async function doLogin() {
  authErr.value = null
  try {
    await apiFetch('/api/login', {
      method: 'POST',
      body: JSON.stringify({ email: email.value, password: password.value }),
    })
    password.value = ''
    const me = await apiFetch<{ id: number; email: string }>('/api/me')
    setAuthenticatedUser(me)
    await loadStrokes()
  } catch {
    authErr.value = 'Login failed'
  }
}

async function doLogout() {
  await apiFetch('/api/logout', { method: 'POST' })
  ws.close()
  setAuthenticatedUser(null)
  strokes.value = []
  candidates.value = null
}

async function doClear() {
  await apiFetch('/api/strokes/clear', { method: 'POST' })
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
  } catch {
    candidates.value = []
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

function doUndo() {
  if (!strokes.value.length) return
  const last = strokes.value[strokes.value.length - 1]
  strokes.value = strokes.value.slice(0, -1)
  if (last.id && last.id > 0) {
    apiFetch(`/api/strokes/delete?id=${last.id}`, { method: 'POST' }).catch(() => {})
    ws.send({ type: 'delete', delete: last.id })
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
        ws.send({ type: 'delete', delete: target.id })
        apiFetch(`/api/strokes/delete?id=${target.id}`, { method: 'POST' }).catch(() => {})
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
      const stroke: Stroke = {
        points: [...points],
        color: color.value,
        width: width.value,
        clientId,
        startedAtUnixMs: Date.now(),
      }
      strokes.value = [...strokes.value, stroke]
      ws.send({ type: 'stroke', stroke })
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
      ws.close()
      return
    }
    const wsUrl = location.port === '5173'
      ? `ws://${location.hostname}:5173/ws`
      : `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`
    boardDebug('connect session', wsUrl)
    ws.connect(wsUrl)
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
      <button v-if="user" :disabled="strokes.length === 0" @click="doUndo">Undo</button>
      <span style="margin-left: auto; opacity: 0.7">
        {{ user ? (wsReady ? 'Practice ready' : 'Connecting…') : 'Sign in to practice' }}
      </span>
      <button v-if="user" @click="doClear">Clear</button>
      <button v-if="user" @click="doLogout">Logout</button>
    </header>

    <div v-if="!user" style="padding: 12px; display: flex; gap: 8px; align-items: center">
      <input v-model="email" placeholder="email" />
      <input v-model="password" type="password" placeholder="password" />
      <button @click="doLogin">Login</button>
      <button @click="doRegister">Register</button>
      <span v-if="authErr" style="color: crimson">{{ authErr }}</span>
    </div>

    <div v-else style="padding: 12px; display: flex; gap: 8px; align-items: center">
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
