export type StrokePayload = {
  id?: number
  points: { x: number; y: number }[]
  color: string
  width: number
  clientId: string
  startedAtUnixMs: number
}

export type StrokeMessage = { type: 'stroke'; stroke: StrokePayload }
export type DeleteMessage = { type: 'delete'; delete: number }
export type ErrorMessage = { type: 'error'; error: string; message?: string }
export type WsMessage = StrokeMessage | DeleteMessage | ErrorMessage

const debugEnabled =
  typeof import.meta !== 'undefined' &&
  // vite sets import.meta.env.DEV in development
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function debug(...args: unknown[]) {
  if (debugEnabled) {
    console.debug('[wsClient]', ...args)
  }
}

export type InboundAppMessage = StrokeMessage | DeleteMessage

export function createWsClient(
  onMessage: (msg: InboundAppMessage) => void,
  onReadyChange?: (ready: boolean) => void,
) {
  let ws: WebSocket | null = null

  const connect = (url: string) => {
    debug('connect', url)
    ws = new WebSocket(url)
    ws.onopen = () => {
      debug('open')
      onReadyChange?.(true)
    }
    ws.onclose = () => {
      debug('close')
      onReadyChange?.(false)
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
        const msg = JSON.parse(event.data) as WsMessage
        if (msg.type === 'error') {
          debug('inbound error frame', msg.error, msg.message)
          return
        }
        debug('inbound', msg.type)
        onMessage(msg)
      } catch {
        debug('ignore malformed message')
      }
    }
  }

  const send = (msg: Exclude<WsMessage, ErrorMessage>) => {
    if (ws?.readyState === WebSocket.OPEN) {
      debug('send', msg.type)
      ws.send(JSON.stringify(msg))
    } else {
      debug('send skipped; socket not open', msg.type)
    }
  }

  const close = () => {
    try {
      debug('close requested')
      ws?.close()
      onReadyChange?.(false)
    } catch {
      // No-op on close failures.
    }
  }

  return { connect, send, close }
}
