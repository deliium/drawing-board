<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { resolveSubmitLogicalSize } from '../canvas/layout'
import { hitTest } from '../canvas/hitTest'
import { useLocale } from '../composables/useLocale'
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
const { t } = useLocale()

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

const statusLabel = computed(() => {
  const key = `board.sync.${syncStatus.value}` as const
  return t(key)
})

const assessmentLabel = computed(() => {
  const a = assessment.value
  if (!a) return ''
  const score = a.score.toFixed(2)
  return a.pass
    ? t('board.targetPass', { target: a.target, score })
    : t('board.targetFail', { target: a.target, score })
})

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

function enqueueDeleteById(strokeId: string) {
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
  const logical = resolveSubmitLogicalSize({
    fromComposable: practiceCanvas.getLogicalSize(),
    canvas,
    reason: 'recognize',
  })
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
  <div class="board-page">
    <div class="toolbar">
      <label class="field">
        <span>{{ t('board.color') }}</span>
        <input v-model="color" type="color" :disabled="tool !== 'pencil'" :aria-label="t('board.color')" />
      </label>
      <label class="field">
        <span>{{ t('board.width') }}</span>
        <input
          v-model.number="width"
          type="range"
          min="1"
          max="20"
          :aria-label="t('board.width')"
        />
      </label>
      <div class="tools" role="radiogroup" :aria-label="t('board.tools')">
        <button
          type="button"
          role="radio"
          class="tool"
          :aria-checked="tool === 'pencil'"
          :class="{ active: tool === 'pencil' }"
          @click="tool = 'pencil'"
        >
          {{ t('board.pencil') }}
        </button>
        <button
          type="button"
          role="radio"
          class="tool"
          :aria-checked="tool === 'eraser'"
          :class="{ active: tool === 'eraser' }"
          @click="tool = 'eraser'"
        >
          {{ t('board.eraser') }}
        </button>
      </div>
      <button type="button" class="action" :disabled="strokes.length === 0" @click="doUndo">
        {{ t('board.undo') }}
      </button>
      <span
        class="sync"
        role="status"
        aria-live="polite"
        aria-atomic="true"
        :data-sync-status="syncStatus"
      >
        {{ statusLabel }}
      </span>
      <button
        type="button"
        class="action"
        :disabled="clearInFlight || syncStatus === 'error'"
        @click="doClear"
      >
        {{ t('board.clear') }}
      </button>
      <button type="button" class="action" @click="doLogout">{{ t('board.logout') }}</button>
    </div>

    <div class="recognize-row">
      <button type="button" class="primary" :disabled="!recognizeEnabled" @click="recognize">
        {{ recognizeInFlight ? t('board.recognizing') : t('board.recognize') }}
      </button>
      <span class="hint">{{ t('board.matchHint') }}</span>
      <div v-if="assessment" class="assessment" role="status" aria-live="polite">
        {{ assessmentLabel }}
      </div>
      <div v-if="candidates && candidates.length > 0" class="candidates">
        <span
          v-for="(c, i) in candidates"
          :key="i"
          class="chip"
          :title="`match score ${c.score}`"
          lang="ja"
        >
          {{ c.text }}
        </span>
      </div>
      <span v-else-if="candidates && candidates.length === 0">{{ t('board.noCandidates') }}</span>
    </div>

    <div class="stage">
      <canvas
        ref="canvasRef"
        class="board-canvas"
        :aria-label="t('board.canvasLabel')"
      />
    </div>
  </div>
</template>

<style scoped>
.board-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: calc(100dvh - 5rem);
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  align-items: center;
}

.field {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: var(--touch-min);
  font-size: 0.95rem;
}

.field input[type='color'] {
  width: var(--touch-min);
  height: var(--touch-min);
  padding: 0;
  border: 1px solid var(--rule);
  background: var(--paper-raised);
  cursor: pointer;
}

.field input[type='range'] {
  width: 7rem;
  min-height: var(--touch-min);
}

.tools {
  display: inline-flex;
  gap: var(--space-2);
}

.tool,
.action,
.primary {
  min-height: var(--touch-min);
  padding: 0 var(--space-3);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  color: var(--ink);
  font: inherit;
  cursor: pointer;
}

.tool.active {
  background: var(--ink);
  color: var(--paper-raised);
  border-color: var(--ink);
}

.primary {
  background: var(--accent);
  color: #f8fafc;
  border-color: var(--accent);
  font-weight: 600;
}

.primary:disabled,
.action:disabled,
.tool:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.sync {
  margin-left: auto;
  color: var(--ink-muted);
  font-size: 0.9rem;
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
}

.recognize-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  align-items: center;
}

.hint {
  color: var(--ink-muted);
  font-size: 0.9rem;
}

.assessment {
  font-size: 0.9rem;
}

.candidates {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.chip {
  min-height: var(--touch-min);
  display: inline-flex;
  align-items: center;
  padding: 0 var(--space-3);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  font-family: var(--font-ja);
}

.stage {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: var(--space-4) 0;
}

.board-canvas {
  width: var(--canvas-size);
  height: var(--canvas-size);
  touch-action: none;
  background: var(--paper-raised);
  border: 2px solid var(--ink);
  border-radius: var(--radius-sm);
}
</style>
