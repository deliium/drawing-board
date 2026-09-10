export type Point = { x: number; y: number }

export type StrokeSyncState = 'pending' | 'saving' | 'saved' | 'failed'

export type Stroke = {
  id?: string
  points: Point[]
  color: string
  width: number
  clientId: string
  startedAtUnixMs: number
  opId?: string
  sync?: StrokeSyncState
}

export type StrokeMessage = {
  type: 'stroke'
  opId?: string
  boardRev?: number
  stroke: Stroke
}
export type DeleteMessage = {
  type: 'delete'
  opId?: string
  delete?: string
  deleteOpId?: string
  boardRev?: number
}
export type ClearMessage = {
  type: 'clear'
  opId?: string
  boardRev?: number
  clear?: boolean
}
export type AckMessage = {
  type: 'ack'
  opId: string
  ok: boolean
  boardRev?: number
  strokeId?: string
  delete?: string
  clear?: boolean
  error?: string
  message?: string
}
export type IncomingMessage = StrokeMessage | DeleteMessage | ClearMessage | AckMessage

export type ApplyResult = {
  strokes: Stroke[]
  action:
    | 'merged-id'
    | 'acked-stroke'
    | 'acked-delete'
    | 'acked-clear'
    | 'clear-applied'
    | 'nack-stroke'
    | 'nack-cancelled'
    | 'ignored-foreign-stroke'
    | 'deleted'
    | 'deleted-by-opId'
    | 'ignored-unknown-delete'
    | 'ignored-duplicate-ack'
    | 'ignored-stale-ack'
    | 'noop'
  reason?: string
  boardRev?: number
}

/**
 * Apply an inbound WS message under personal-training isolation + boardRev authority:
 * - ack ok assigns server id / clears pending for matching local opId
 * - clear empties the canvas
 * - delete by id or deleteOpId
 * - unmatched stroke creates remain non-authoritative
 */
export function applyIncomingMessage(
  strokes: Stroke[],
  message: IncomingMessage,
  localBoardRev?: number,
): ApplyResult {
  if (message.type === 'ack') {
    return applyAck(strokes, message, localBoardRev)
  }

  if (message.type === 'clear') {
    return {
      strokes: [],
      action: 'clear-applied',
      reason: 'clear echo',
      boardRev: message.boardRev,
    }
  }

  if (message.type === 'stroke') {
    if (
      typeof message.boardRev === 'number' &&
      typeof localBoardRev === 'number' &&
      message.boardRev < localBoardRev
    ) {
      return { strokes, action: 'ignored-stale-ack', reason: 'stale_rev', boardRev: message.boardRev }
    }
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
      return { strokes: updated, action: 'merged-id', boardRev: message.boardRev }
    }
    return {
      strokes,
      action: 'ignored-foreign-stroke',
      reason: 'no matching pending local stroke',
    }
  }

  if (message.type === 'delete') {
    if (message.deleteOpId) {
      const exists = strokes.some((st) => st.opId === message.deleteOpId)
      if (!exists && (message.delete == null || !strokes.some((st) => st.id === message.delete))) {
        return {
          strokes,
          action: 'ignored-unknown-delete',
          reason: `unknown deleteOpId ${message.deleteOpId}`,
        }
      }
      return {
        strokes: strokes.filter(
          (st) => st.opId !== message.deleteOpId && (message.delete == null || st.id !== message.delete),
        ),
        action: 'deleted-by-opId',
        boardRev: message.boardRev,
      }
    }
    const id = message.delete
    if (id == null) {
      return { strokes, action: 'ignored-unknown-delete', reason: 'missing delete id' }
    }
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
      boardRev: message.boardRev,
    }
  }

  return { strokes, action: 'noop' }
}

function applyAck(strokes: Stroke[], ack: AckMessage, localBoardRev?: number): ApplyResult {
  if (
    typeof ack.boardRev === 'number' &&
    typeof localBoardRev === 'number' &&
    ack.ok &&
    ack.boardRev < localBoardRev
  ) {
    return { strokes, action: 'ignored-stale-ack', reason: 'stale_rev', boardRev: ack.boardRev }
  }

  if (ack.ok && ack.clear) {
    return {
      strokes: [],
      action: 'acked-clear',
      boardRev: ack.boardRev,
    }
  }

  if (ack.ok && ack.strokeId != null) {
    const idx = strokes.findIndex((st) => st.opId === ack.opId && !st.id)
    if (idx < 0) {
      const already = strokes.findIndex((st) => st.opId === ack.opId && st.id === ack.strokeId)
      if (already >= 0) {
        return { strokes, action: 'ignored-duplicate-ack', boardRev: ack.boardRev }
      }
      // Pending stroke was undone/cleared locally — do not resurrect.
      return {
        strokes,
        action: 'ignored-duplicate-ack',
        reason: 'no pending stroke for opId',
        boardRev: ack.boardRev,
      }
    }
    const updated = [...strokes]
    updated[idx] = { ...updated[idx], id: ack.strokeId, sync: 'saved' }
    return { strokes: updated, action: 'acked-stroke', boardRev: ack.boardRev }
  }

  if (ack.ok && ack.delete != null) {
    return {
      strokes: strokes.filter((st) => st.id !== ack.delete),
      action: 'acked-delete',
      boardRev: ack.boardRev,
    }
  }

  if (!ack.ok) {
    if (ack.error === 'op_cancelled') {
      return {
        strokes: strokes.filter((st) => st.opId !== ack.opId),
        action: 'nack-cancelled',
        reason: ack.error,
        boardRev: ack.boardRev,
      }
    }
    const idx = strokes.findIndex((st) => st.opId === ack.opId)
    if (idx < 0) {
      return { strokes, action: 'ignored-duplicate-ack', reason: 'nack for unknown op' }
    }
    const updated = [...strokes]
    updated[idx] = { ...updated[idx], sync: 'failed' }
    return { strokes: updated, action: 'nack-stroke', reason: ack.error, boardRev: ack.boardRev }
  }

  return { strokes, action: 'noop' }
}
