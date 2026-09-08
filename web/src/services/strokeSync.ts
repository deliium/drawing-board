export type Point = { x: number; y: number }

export type StrokeSyncState = 'pending' | 'saving' | 'saved' | 'failed'

export type Stroke = {
  id?: number
  points: Point[]
  color: string
  width: number
  clientId: string
  startedAtUnixMs: number
  opId?: string
  sync?: StrokeSyncState
}

export type StrokeMessage = { type: 'stroke'; opId?: string; stroke: Stroke }
export type DeleteMessage = { type: 'delete'; opId?: string; delete: number }
export type AckMessage = {
  type: 'ack'
  opId: string
  ok: boolean
  strokeId?: number
  delete?: number
  error?: string
  message?: string
}
export type IncomingMessage = StrokeMessage | DeleteMessage | AckMessage

export type ApplyResult = {
  strokes: Stroke[]
  action:
    | 'merged-id'
    | 'acked-stroke'
    | 'acked-delete'
    | 'nack-stroke'
    | 'ignored-foreign-stroke'
    | 'deleted'
    | 'ignored-unknown-delete'
    | 'ignored-duplicate-ack'
    | 'noop'
  reason?: string
}

/**
 * Apply an inbound WS message under personal-training isolation:
 * - ack ok assigns server id / clears pending for matching local opId
 * - stroke echoes that match a pending local stroke (opId, else clientId+startedAtUnixMs) assign server id
 * - unmatched strokes are ignored (no shared multi-user canvas)
 * - delete only removes strokes already present locally
 */
export function applyIncomingMessage(strokes: Stroke[], message: IncomingMessage): ApplyResult {
  if (message.type === 'ack') {
    return applyAck(strokes, message)
  }

  if (message.type === 'stroke') {
    const incoming = message.stroke
    const opId = incoming.opId || message.opId
    const existingIndex = strokes.findIndex((st) => {
      if (st.id) return false
      if (opId && st.opId === opId) return true
      return (
        st.clientId === incoming.clientId &&
        st.startedAtUnixMs === incoming.startedAtUnixMs &&
        !st.id
      )
    })
    if (existingIndex >= 0) {
      const updated = [...strokes]
      updated[existingIndex] = {
        ...updated[existingIndex],
        id: incoming.id,
        opId: opId || updated[existingIndex].opId,
        sync: 'saved',
      }
      return { strokes: updated, action: 'merged-id' }
    }
    return {
      strokes,
      action: 'ignored-foreign-stroke',
      reason: 'no matching pending local stroke',
    }
  }

  if (message.type === 'delete') {
    const id = message.delete
    const exists = strokes.some((st) => st.id === id)
    if (!exists) {
      return {
        strokes,
        action: 'ignored-unknown-delete',
        reason: `unknown id ${id}`,
      }
    }
    return {
      strokes: strokes.filter((st) => st.id !== id),
      action: 'deleted',
    }
  }

  return { strokes, action: 'noop' }
}

function applyAck(strokes: Stroke[], ack: AckMessage): ApplyResult {
  if (ack.ok && ack.strokeId != null) {
    const idx = strokes.findIndex((st) => st.opId === ack.opId && !st.id)
    if (idx < 0) {
      const already = strokes.findIndex((st) => st.opId === ack.opId && st.id === ack.strokeId)
      if (already >= 0) {
        return { strokes, action: 'ignored-duplicate-ack' }
      }
      return { strokes, action: 'ignored-duplicate-ack', reason: 'no pending stroke for opId' }
    }
    const updated = [...strokes]
    updated[idx] = { ...updated[idx], id: ack.strokeId, sync: 'saved' }
    return { strokes: updated, action: 'acked-stroke' }
  }

  if (ack.ok && ack.delete != null) {
    return {
      strokes: strokes.filter((st) => st.id !== ack.delete),
      action: 'acked-delete',
    }
  }

  if (!ack.ok) {
    const idx = strokes.findIndex((st) => st.opId === ack.opId)
    if (idx < 0) {
      return { strokes, action: 'ignored-duplicate-ack', reason: 'nack for unknown op' }
    }
    const updated = [...strokes]
    updated[idx] = { ...updated[idx], sync: 'failed' }
    return { strokes: updated, action: 'nack-stroke', reason: ack.error }
  }

  return { strokes, action: 'noop' }
}
