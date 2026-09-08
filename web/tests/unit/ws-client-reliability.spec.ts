import { describe, expect, it, vi } from 'vitest'
import {
  ACK_TIMEOUT_MS,
  createWsClient,
  MAX_PENDING_OPS,
  type AckMessage,
} from '../../src/services/wsClient'

type FakeSocket = {
  readyState: number
  onopen: ((ev?: Event) => void) | null
  onclose: ((ev?: CloseEvent) => void) | null
  onerror: ((ev?: Event) => void) | null
  onmessage: ((ev: MessageEvent) => void) | null
  send: ReturnType<typeof vi.fn>
  close: ReturnType<typeof vi.fn>
}

function installFakeWebSocket() {
  const sockets: FakeSocket[] = []
  class FakeWebSocket {
    static CONNECTING = 0
    static OPEN = 1
    static CLOSING = 2
    static CLOSED = 3
    readyState = FakeWebSocket.CONNECTING
    onopen: FakeSocket['onopen'] = null
    onclose: FakeSocket['onclose'] = null
    onerror: FakeSocket['onerror'] = null
    onmessage: FakeSocket['onmessage'] = null
    send = vi.fn()
    close = vi.fn(() => {
      this.readyState = FakeWebSocket.CLOSED
      this.onclose?.({} as CloseEvent)
    })
    constructor(public url: string) {
      sockets.push(this as unknown as FakeSocket)
    }
  }
  return { FakeWebSocket: FakeWebSocket as unknown as typeof WebSocket, sockets }
}

describe('wsClient reliability queue', () => {
  it('queues sends while connecting and flushes on open', () => {
    vi.useFakeTimers()
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    const client = createWsClient({
      onMessage: () => {},
      WebSocketImpl: FakeWebSocket,
    })

    client.connect('ws://test/ws')
    expect(client.getStatus()).toBe('connecting')

    const ok = client.send({
      type: 'stroke',
      opId: 'op-1',
      stroke: {
        points: [
          { x: 0, y: 0 },
          { x: 1, y: 1 },
        ],
        color: '#1d4ed8',
        width: 4,
        clientId: 'c',
        startedAtUnixMs: 1,
      },
    })
    expect(ok).toBe(true)
    expect(client.getQueueLength()).toBe(1)
    expect(sockets[0].send).not.toHaveBeenCalled()

    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)
    expect(sockets[0].send).toHaveBeenCalledTimes(1)
    expect(client.getStatus()).toBe('saving')
    vi.useRealTimers()
  })

  it('never silently drops a disconnected send', () => {
    const { FakeWebSocket } = installFakeWebSocket()
    const client = createWsClient({
      onMessage: () => {},
      WebSocketImpl: FakeWebSocket,
    })
    // no connect
    const ok = client.send({
      type: 'delete',
      opId: 'op-del',
      delete: 9,
    })
    expect(ok).toBe(true)
    expect(client.getQueueLength()).toBe(1)
    expect(client.getStatus()).toBe('connecting')
  })

  it('completes op on ack and ignores duplicate ack', () => {
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    const acks: AckMessage[] = []
    const client = createWsClient({
      onMessage: () => {},
      onAck: (a) => acks.push(a),
      WebSocketImpl: FakeWebSocket,
    })
    client.connect('ws://test/ws')
    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)

    client.send({
      type: 'stroke',
      opId: 'op-dup',
      stroke: {
        points: [
          { x: 0, y: 0 },
          { x: 1, y: 1 },
        ],
        color: '#000000',
        width: 2,
        clientId: 'c',
        startedAtUnixMs: 2,
      },
    })

    const ackPayload = JSON.stringify({
      type: 'ack',
      opId: 'op-dup',
      ok: true,
      strokeId: 7,
    })
    sockets[0].onmessage?.({ data: ackPayload } as MessageEvent)
    expect(client.getQueueLength()).toBe(0)
    expect(acks).toHaveLength(1)

    sockets[0].onmessage?.({ data: ackPayload } as MessageEvent)
    expect(acks).toHaveLength(1)
    expect(client.getStatus()).toBe('saved')
  })

  it('rejects enqueue when queue is full', () => {
    const { FakeWebSocket } = installFakeWebSocket()
    let rejectReason = ''
    const client = createWsClient({
      onMessage: () => {},
      onQueueReject: (r) => {
        rejectReason = r
      },
      maxPendingOps: 1,
      WebSocketImpl: FakeWebSocket,
    })
    expect(
      client.send({
        type: 'delete',
        opId: 'a',
        delete: 1,
      }),
    ).toBe(true)
    expect(
      client.send({
        type: 'delete',
        opId: 'b',
        delete: 2,
      }),
    ).toBe(false)
    expect(rejectReason).toBe('queue_full')
    expect(client.getStatus()).toBe('error')
    expect(MAX_PENDING_OPS).toBe(32)
  })

  it('schedules reconnect after unexpected close', () => {
    vi.useFakeTimers()
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    const client = createWsClient({
      onMessage: () => {},
      WebSocketImpl: FakeWebSocket,
    })
    client.connect('ws://test/ws')
    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)
    sockets[0].readyState = 3
    sockets[0].onclose?.({} as CloseEvent)
    expect(client.getStatus()).toBe('offline')
    vi.advanceTimersByTime(20_000)
    expect(sockets.length).toBeGreaterThan(1)
    vi.useRealTimers()
  })

  it('clearPending drops session queue (reload authority)', () => {
    const { FakeWebSocket } = installFakeWebSocket()
    const client = createWsClient({
      onMessage: () => {},
      WebSocketImpl: FakeWebSocket,
    })
    client.send({ type: 'delete', opId: 'x', delete: 1 })
    expect(client.getQueueLength()).toBe(1)
    client.clearPending()
    expect(client.getQueueLength()).toBe(0)
  })

  it('ack timeout marks failure after max attempts', () => {
    vi.useFakeTimers()
    let now = 1_000
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    const acks: AckMessage[] = []
    const client = createWsClient({
      onMessage: () => {},
      onAck: (a) => acks.push(a),
      WebSocketImpl: FakeWebSocket,
      now: () => now,
    })
    client.connect('ws://test/ws')
    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)
    client.send({
      type: 'stroke',
      opId: 'slow',
      stroke: {
        points: [
          { x: 0, y: 0 },
          { x: 1, y: 1 },
        ],
        color: '#fff',
        width: 1,
        clientId: 'c',
        startedAtUnixMs: 1,
      },
    })

    for (let i = 0; i < 5; i += 1) {
      now += ACK_TIMEOUT_MS + 1
      vi.advanceTimersByTime(ACK_TIMEOUT_MS + 1)
    }
    expect(acks.some((a) => a.ok === false && a.error === 'ack_timeout')).toBe(true)
    expect(client.getStatus()).toBe('error')
    vi.useRealTimers()
  })
})
