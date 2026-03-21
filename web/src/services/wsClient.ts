export type StrokeMessage = { type: 'stroke'; stroke: unknown }
export type DeleteMessage = { type: 'delete'; delete: number }
export type WsMessage = StrokeMessage | DeleteMessage

export function createWsClient(
  onMessage: (msg: WsMessage) => void,
  onReadyChange?: (ready: boolean) => void,
) {
  let ws: WebSocket | null = null

  const connect = (url: string) => {
    ws = new WebSocket(url)
    ws.onopen = () => onReadyChange?.(true)
    ws.onclose = () => onReadyChange?.(false)
    ws.onerror = () => {
      try {
        ws?.close()
      } catch {
        // Ignore close failures.
      }
    }
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as WsMessage
        onMessage(msg)
      } catch {
        // Ignore malformed messages to preserve runtime safety.
      }
    }
  }

  const send = (msg: WsMessage) => {
    if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg))
  }

  const close = () => {
    try {
      ws?.close()
      onReadyChange?.(false)
    } catch {
      // No-op on close failures.
    }
  }

  return { connect, send, close }
}
