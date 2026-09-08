export type StrokePayload = {
  id?: number
  points: { x: number; y: number }[]
  color: string
  width: number
  clientId: string
  startedAtUnixMs: number
  opId?: string
  sync?: 'pending' | 'saving' | 'saved' | 'failed'
}

export type StrokeMessage = { type: 'stroke'; opId: string; stroke: StrokePayload }
export type DeleteMessage = { type: 'delete'; opId: string; delete: number }
export type AckMessage = {
  type: 'ack'
  opId: string
  ok: boolean
  strokeId?: number
  delete?: number
  error?: string
  message?: string
}
export type ErrorMessage = { type: 'error'; error: string; message?: string }
export type WsMessage = StrokeMessage | DeleteMessage | AckMessage | ErrorMessage

export type InboundAppMessage = StrokeMessage | DeleteMessage | AckMessage

export type SyncStatus = 'connecting' | 'saving' | 'saved' | 'offline' | 'error'

export const MAX_PENDING_OPS = 32
export const MAX_OP_ATTEMPTS = 5
export const ACK_TIMEOUT_MS = 10_000
export const BACKOFF_BASE_MS = 500
export const BACKOFF_MAX_MS = 8_000

const debugEnabled =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function debug(...args: unknown[]) {
  if (debugEnabled) {
    console.debug('[wsClient]', ...args)
  }
}

type QueuedOp = {
  opId: string
  msg: StrokeMessage | DeleteMessage
  attempts: number
  enqueuedAt: number
  awaitingAckSince?: number
}

export type WsClientOptions = {
  onMessage: (msg: InboundAppMessage) => void
  onStatusChange?: (status: SyncStatus) => void
  onAck?: (ack: AckMessage) => void
  onQueueReject?: (reason: string) => void
  maxPendingOps?: number
  /** Injectable for tests */
  now?: () => number
  /** Injectable for tests */
  schedule?: (fn: () => void, ms: number) => ReturnType<typeof setTimeout>
  clearSchedule?: (id: ReturnType<typeof setTimeout>) => void
  WebSocketImpl?: typeof WebSocket
}

export function createWsClient(options: WsClientOptions) {
  const {
    onMessage,
    onStatusChange,
    onAck,
    onQueueReject,
    maxPendingOps = MAX_PENDING_OPS,
    now = () => Date.now(),
    schedule = (fn, ms) => setTimeout(fn, ms),
    clearSchedule = (id) => clearTimeout(id),
    WebSocketImpl = WebSocket,
  } = options

  let ws: WebSocket | null = null
  let url = ''
  let intentionalClose = false
  let replacingSocket = false
  let reconnectAttempt = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let ackTimer: ReturnType<typeof setTimeout> | null = null
  let hardError = false
  const queue: QueuedOp[] = []
  const completedOpIds = new Set<string>()

  const emitStatus = () => {
    let status: SyncStatus
    if (hardError) {
      status = 'error'
    } else if (!ws || ws.readyState === WebSocket.CONNECTING) {
      status = 'connecting'
    } else if (ws.readyState !== WebSocket.OPEN) {
      status = intentionalClose ? 'connecting' : 'offline'
    } else if (queue.length > 0) {
      status = 'saving'
    } else {
      status = 'saved'
    }
    onStatusChange?.(status)
  }

  const clearReconnectTimer = () => {
    if (reconnectTimer != null) {
      clearSchedule(reconnectTimer)
      reconnectTimer = null
    }
  }

  const clearAckTimer = () => {
    if (ackTimer != null) {
      clearSchedule(ackTimer)
      ackTimer = null
    }
  }

  const backoffMs = (attempt: number) => {
    const exp = Math.min(BACKOFF_MAX_MS, BACKOFF_BASE_MS * 2 ** attempt)
    const jitter = exp * (0.8 + Math.random() * 0.4)
    return Math.min(BACKOFF_MAX_MS, Math.round(jitter))
  }

  const scheduleAckWatch = () => {
    clearAckTimer()
    const waiting = queue.find((op) => op.awaitingAckSince != null)
    if (!waiting || !waiting.awaitingAckSince) return
    const remaining = ACK_TIMEOUT_MS - (now() - waiting.awaitingAckSince)
    ackTimer = schedule(() => {
      ackTimer = null
      handleAckTimeout()
    }, Math.max(0, remaining))
  }

  const handleAckTimeout = () => {
    const op = queue.find((q) => q.awaitingAckSince != null)
    if (!op) return
    debug('ack timeout', op.opId, 'attempts', op.attempts)
    op.awaitingAckSince = undefined
    if (op.attempts >= MAX_OP_ATTEMPTS) {
      hardError = true
      const idx = queue.indexOf(op)
      if (idx >= 0) queue.splice(idx, 1)
      onAck?.({
        type: 'ack',
        opId: op.opId,
        ok: false,
        error: 'ack_timeout',
        message: 'operation timed out',
      })
      emitStatus()
      flush()
      return
    }
    flush()
  }

  const flush = () => {
    if (ws?.readyState !== WebSocket.OPEN) {
      emitStatus()
      return
    }
    const inFlight = queue.find((q) => q.awaitingAckSince != null)
    if (inFlight) {
      scheduleAckWatch()
      emitStatus()
      return
    }
    const next = queue[0]
    if (!next) {
      emitStatus()
      return
    }
    next.attempts += 1
    next.awaitingAckSince = now()
    debug('send', next.msg.type, next.opId, 'attempt', next.attempts)
    try {
      ws.send(JSON.stringify(next.msg))
    } catch {
      debug('send failed; will retry on reconnect', next.opId)
      next.awaitingAckSince = undefined
    }
    scheduleAckWatch()
    emitStatus()
  }

  const scheduleReconnect = () => {
    if (intentionalClose || !url) return
    clearReconnectTimer()
    const delay = backoffMs(reconnectAttempt)
    reconnectAttempt += 1
    debug('reconnect scheduled', delay, 'ms attempt', reconnectAttempt)
    emitStatus()
    reconnectTimer = schedule(() => {
      reconnectTimer = null
      connect(url)
    }, delay)
  }

  const connect = (nextUrl: string) => {
    url = nextUrl
    intentionalClose = false
    hardError = false
    clearReconnectTimer()
    debug('connect', nextUrl)
    if (ws) {
      replacingSocket = true
      try {
        ws.close()
      } catch {
        // ignore
      }
      replacingSocket = false
    }
    ws = new WebSocketImpl(nextUrl)
    emitStatus()

    ws.onopen = () => {
      debug('open')
      reconnectAttempt = 0
      for (const op of queue) {
        op.awaitingAckSince = undefined
      }
      emitStatus()
      flush()
    }

    ws.onclose = () => {
      debug('close')
      clearAckTimer()
      for (const op of queue) {
        op.awaitingAckSince = undefined
      }
      emitStatus()
      if (!intentionalClose && !replacingSocket) {
        scheduleReconnect()
      }
    }

    ws.onerror = () => {
      debug('error')
      try {
        ws?.close()
      } catch {
        // Ignore close failures.
      }
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(String(event.data)) as WsMessage
        if (msg.type === 'error') {
          debug('inbound error frame', msg.error, msg.message)
          return
        }
        if (msg.type === 'ack') {
          handleAck(msg)
          return
        }
        debug('inbound', msg.type)
        onMessage(msg)
      } catch {
        debug('ignore malformed message')
      }
    }
  }

  const handleAck = (ack: AckMessage) => {
    if (completedOpIds.has(ack.opId)) {
      debug('ignore duplicate ack', ack.opId)
      return
    }
    const idx = queue.findIndex((q) => q.opId === ack.opId)
    if (idx < 0) {
      debug('ignore ack for unknown op', ack.opId)
      return
    }
    const op = queue[idx]
    if (ack.ok) {
      completedOpIds.add(ack.opId)
      queue.splice(idx, 1)
      hardError = false
      debug('ack ok', ack.opId)
      onAck?.(ack)
      onMessage(ack)
      clearAckTimer()
      flush()
      emitStatus()
      return
    }

    const retryable = ack.error === 'rate_limited'
    if (retryable && op.attempts < MAX_OP_ATTEMPTS) {
      debug('nack retryable', ack.opId, ack.error)
      op.awaitingAckSince = undefined
      clearAckTimer()
      schedule(() => flush(), backoffMs(op.attempts))
      emitStatus()
      return
    }

    queue.splice(idx, 1)
    hardError = true
    debug('ack failed', ack.opId, ack.error)
    onAck?.(ack)
    onMessage(ack)
    clearAckTimer()
    flush()
    emitStatus()
  }

  const enqueue = (msg: StrokeMessage | DeleteMessage): boolean => {
    if (queue.length >= maxPendingOps) {
      debug('queue full; reject', msg.opId)
      hardError = true
      onQueueReject?.('queue_full')
      emitStatus()
      return false
    }
    if (queue.some((q) => q.opId === msg.opId) || completedOpIds.has(msg.opId)) {
      debug('skip duplicate enqueue', msg.opId)
      return true
    }
    hardError = false
    queue.push({
      opId: msg.opId,
      msg,
      attempts: 0,
      enqueuedAt: now(),
    })
    debug('enqueue', msg.type, msg.opId, 'queue=', queue.length)
    flush()
    emitStatus()
    return true
  }

  const send = (msg: StrokeMessage | DeleteMessage) => {
    if (!msg.opId) {
      debug('send rejected; missing opId', msg.type)
      onQueueReject?.('missing_op_id')
      return false
    }
    return enqueue(msg)
  }

  const dropOp = (opId: string) => {
    const idx = queue.findIndex((q) => q.opId === opId)
    if (idx >= 0) {
      const wasInFlight = queue[idx].awaitingAckSince != null
      queue.splice(idx, 1)
      if (wasInFlight) {
        clearAckTimer()
        flush()
      }
      emitStatus()
    }
  }

  const clearPending = () => {
    queue.length = 0
    completedOpIds.clear()
    hardError = false
    clearAckTimer()
    emitStatus()
  }

  const close = () => {
    intentionalClose = true
    clearReconnectTimer()
    clearAckTimer()
    try {
      debug('close requested')
      ws?.close()
    } catch {
      // No-op on close failures.
    }
    ws = null
    emitStatus()
  }

  const getQueueLength = () => queue.length
  const getStatus = (): SyncStatus => {
    if (hardError) return 'error'
    if (!ws || ws.readyState === WebSocket.CONNECTING) return 'connecting'
    if (ws.readyState !== WebSocket.OPEN) return intentionalClose ? 'connecting' : 'offline'
    if (queue.length > 0) return 'saving'
    return 'saved'
  }

  return {
    connect,
    send,
    close,
    dropOp,
    clearPending,
    getQueueLength,
    getStatus,
  }
}
