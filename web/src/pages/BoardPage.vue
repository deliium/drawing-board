<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { logicalSizeForRecognize } from '../canvas/coords'
import { hitTest } from '../canvas/hitTest'
import { usePracticeCanvas } from '../composables/usePracticeCanvas'
import { apiFetch } from '../services/apiClient'
import { trackMetric } from '../services/migrationHealth'
import {
  createWsClient,
  type InboundAppMessage,
  type SyncStatus,
} from '../services/wsClient'
import { applyIncomingMessage, type Point, type Stroke } from '../services/strokeSync'
import { shouldAcceptRecognizeResponse } from '../services/recognizeGate'
import { sessionContext, setAuthenticatedUser } from '../services/sessionContext'

type Candidate = { text: string; score: number; scoreKind?: string }
type Assessment = {
  target: string
  pass: boolean
  score: number
  scoreKind: string
  reasons: string[]
}
type StrokesListResponse = { boardRev: number; strokes: Stroke[] }

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
const assessment = ref<Assessment | null>(null)
const strokes = ref<Stroke[]>([])
const syncStatus = ref<SyncStatus>('connecting')
const clearInFlight = ref(false)
const recognizeInFlight = ref(false)
const recognizeAttempt = ref(0)
const user = computed(() => sessionContext.user)
const canvasEnabled = computed(() => Boolean(user.value))
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
  onStaleBoard: (rev) => {
    boardDebug('stale_board reconcile', rev)
    void loadStrokes()
  },
  onQueueReject: (reason) => {
    boardDebug('queue reject', reason)
    syncStatus.value = 'error'
  },
})

const recognizeEnabled = computed(
  () =>
    syncStatus.value === 'saved' &&
    ws.getQueueLength() === 0 &&
    strokes.value.length > 0 &&
    !recognizeInFlight.value &&
    !clearInFlight.value,
)

function handleIncoming(message: InboundAppMessage) {
  const result = applyIncomingMessage(strokes.value, message, ws.getBoardRev())
  if (typeof result.boardRev === 'number') {
    ws.setBoardRev(Math.max(ws.getBoardRev(), result.boardRev))
  }
  if (
    result.action === 'ignored-foreign-stroke' ||
    result.action === 'ignored-unknown-delete' ||
    result.action === 'ignored-duplicate-ack' ||
    result.action === 'ignored-stale-ack'
  ) {
    boardDebug('ignore inbound', result.action, result.reason || 'stale_rev')
    return
  }
  if (result.action === 'clear-applied' || result.action === 'acked-clear') {
    boardDebug('clear_applied', result.action)
    clearInFlight.value = false
    candidates.value = null
    assessment.value = null
  }
  boardDebug('apply inbound', result.action)
  strokes.value = result.strokes
}

async function loadStrokes() {
  if (!user.value) return
  ws.clearPending()
  clearInFlight.value = false
  try {
    const payload = await apiFetch<StrokesListResponse>('/api/strokes')
    const rev = typeof payload.boardRev === 'number' ? payload.boardRev : 0
    const rows = Array.isArray(payload.strokes) ? payload.strokes : []
    ws.setBoardRev(rev)
    strokes.value = rows.map((s) => ({ ...s, sync: 'saved' as const }))
    boardDebug('load strokes', 'boardRev', rev, 'count', rows.length)
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
  assessment.value = null
  if (wsDebug) console.debug('[BoardPage] logout')
  await router.replace({ name: 'login' })
}

function doClear() {
  if (clearInFlight.value || syncStatus.value === 'error') return
  clearInFlight.value = true
  candidates.value = null
  assessment.value = null
  ws.dropPendingCreates()
  strokes.value = []
  const opId = newOpId()
  const queued = ws.send({ type: 'clear', opId, baseRev: ws.getBoardRev() })
  if (!queued) {
    boardDebug('clear not queued')
    clearInFlight.value = false
    void loadStrokes()
  }
}

function onStrokeComplete(points: Point[]) {
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
    baseRev: ws.getBoardRev(),
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
    boardDebug('stroke not queued; queue full or sync error')
    return
  }
  strokes.value = [...strokes.value, { ...stroke, sync: 'saving' }]
}

function enqueueDeleteById(strokeId: number) {
  const opId = newOpId()
  ws.send({ type: 'delete', opId, baseRev: ws.getBoardRev(), delete: strokeId })
}

function enqueueDeleteByOpId(createOpId: string) {
  const opId = newOpId()
  ws.send({ type: 'delete', opId, baseRev: ws.getBoardRev(), deleteOpId: createOpId })
}

function removeStrokeLocally(target: Stroke) {
  strokes.value = strokes.value.filter((st) => st !== target && st.opId !== target.opId)
  if (target.opId && (!target.id || target.sync === 'pending' || target.sync === 'saving')) {
    ws.dropOp(target.opId)
    enqueueDeleteByOpId(target.opId)
    return
  }
  if (target.id && target.id > 0) {
    enqueueDeleteById(target.id)
  }
}

function onEraseAt(point: Point) {
  const target = [...strokes.value].reverse().find((s) => hitTest(point, s))
  if (target) {
    removeStrokeLocally(target)
  }
}

function doUndo() {
  if (!strokes.value.length || syncStatus.value === 'error') return
  const last = strokes.value[strokes.value.length - 1]
  removeStrokeLocally(last)
}

const practiceCanvas = usePracticeCanvas({
  canvasRef,
  enabled: canvasEnabled,
  strokes,
  tool,
  color,
  width,
  syncStatus,
  onStrokeComplete,
  onEraseAt,
})

async function recognize() {
  const canvas = canvasRef.value
  if (!canvas || !recognizeEnabled.value) return
  const logical =
    practiceCanvas.getLogicalSize() ?? logicalSizeForRecognize(canvas)
  const requestedRev = ws.getBoardRev()
  const attempt = ++recognizeAttempt.value
  recognizeInFlight.value = true
  try {
    const result = await apiFetch<{
      candidates: Candidate[]
      boardRev: number
      scoreKind?: string
      assessment?: Assessment
    }>('/api/recognize', {
      method: 'POST',
      body: JSON.stringify({
        topN: 10,
        width: logical.width,
        height: logical.height,
        boardRev: requestedRev,
      }),
    })
    if (
      !shouldAcceptRecognizeResponse({
        attempt,
        currentAttempt: recognizeAttempt.value,
        responseBoardRev: result.boardRev,
        requestedBoardRev: requestedRev,
        localBoardRev: ws.getBoardRev(),
      })
    ) {
      boardDebug('recognize_discarded', 'attempt', attempt, 'respRev', result.boardRev, 'local', ws.getBoardRev())
      return
    }
    candidates.value = result.candidates || []
    assessment.value = result.assessment ?? null
    trackMetric('recognize.success', 1)
  } catch (err) {
    if (attempt === recognizeAttempt.value) {
      candidates.value = []
      assessment.value = null
    }
    trackMetric('recognize.reject', 1)
    if (wsDebug) {
      const msg = err instanceof Error ? err.message : 'recognize failed'
      console.debug('[BoardPage] recognize rejected:', msg)
    }
  } finally {
    if (attempt === recognizeAttempt.value) {
      recognizeInFlight.value = false
    }
  }
}

onMounted(() => {
  loadStrokes()
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
    <header style="padding: 12px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap">
      <b>Japanese Handwriting Practice</b>
      <router-link to="/practice" style="color: #475569; text-decoration: none; font-size: 0.95rem"
        >Practice hiragana</router-link
      >
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
      <button :disabled="clearInFlight || syncStatus === 'error'" @click="doClear">Clear</button>
      <button @click="doLogout">Logout</button>
    </header>

    <div style="padding: 12px; display: flex; gap: 8px; align-items: center; flex-wrap: wrap">
      <button :disabled="!recognizeEnabled" @click="recognize">
        {{ recognizeInFlight ? 'Recognizing…' : 'Recognize' }}
      </button>
      <span style="opacity: 0.7; font-size: 0.9em">Match scores (heuristic ranking, not confidence)</span>
      <div v-if="assessment" style="opacity: 0.85; font-size: 0.9em">
        Target {{ assessment.target }}:
        {{ assessment.pass ? 'pass' : 'no pass' }}
        (match score {{ assessment.score.toFixed(2) }})
      </div>
      <div v-if="candidates && candidates.length > 0" style="display: flex; gap: 8px; flex-wrap: wrap">
        <span
          v-for="(c, i) in candidates"
          :key="i"
          :title="`match score ${c.score}`"
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
