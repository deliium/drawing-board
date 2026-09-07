export type Point = { x: number; y: number }

export type Stroke = {
  id?: number
  points: Point[]
  color: string
  width: number
  clientId: string
  startedAtUnixMs: number
}

export type StrokeMessage = { type: 'stroke'; stroke: Stroke }
export type DeleteMessage = { type: 'delete'; delete: number }
export type IncomingMessage = StrokeMessage | DeleteMessage

export type ApplyResult = {
  strokes: Stroke[]
  action: 'merged-id' | 'ignored-foreign-stroke' | 'deleted' | 'ignored-unknown-delete' | 'noop'
  reason?: string
}

/**
 * Apply an inbound WS message under personal-training isolation:
 * - stroke echoes that match a pending local stroke (clientId + startedAtUnixMs, no id) assign server id
 * - unmatched strokes are ignored (no shared multi-user canvas)
 * - delete only removes strokes already present locally
 */
export function applyIncomingMessage(strokes: Stroke[], message: IncomingMessage): ApplyResult {
  if (message.type === 'stroke') {
    const incoming = message.stroke
    const existingIndex = strokes.findIndex(
      (st) =>
        st.clientId === incoming.clientId &&
        st.startedAtUnixMs === incoming.startedAtUnixMs &&
        !st.id,
    )
    if (existingIndex >= 0) {
      const updated = [...strokes]
      updated[existingIndex] = { ...updated[existingIndex], id: incoming.id }
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
