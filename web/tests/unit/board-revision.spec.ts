import { describe, expect, it, vi } from 'vitest'
import { applyIncomingMessage, type Stroke } from '../../src/services/strokeSync'
import { createWsClient, type AckMessage } from '../../src/services/wsClient'
import { shouldAcceptRecognizeResponse } from '../../src/services/recognizeGate'

function pendingStroke(overrides: Partial<Stroke> = {}): Stroke {
  return {
    points: [
      { x: 0, y: 0 },
      { x: 10, y: 10 },
    ],
    color: '#1d4ed8',
    width: 4,
    clientId: 'local-client',
    startedAtUnixMs: 1_700_000_000_000,
    opId: 'op-local-1',
    sync: 'saving',
    ...overrides,
  }
}

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

describe('board revision client races', () => {
  it('pending undo then late ack does not resurrect stroke', () => {
    const result = applyIncomingMessage([], {
      type: 'ack',
      opId: 'op-local-1',
      ok: true,
      strokeId: '42424242-4242-4242-8242-424242424242',
      boardRev: 3,
    })
    expect(result.action).toBe('ignored-duplicate-ack')
    expect(result.strokes).toHaveLength(0)
  })

  it('clear echo empties strokes', () => {
    const local = pendingStroke({ id: '11111111-1111-4111-8111-111111111111', sync: 'saved' })
    const result = applyIncomingMessage([local], {
      type: 'clear',
      opId: 'op-clear',
      boardRev: 5,
      clear: true,
    })
    expect(result.action).toBe('clear-applied')
    expect(result.strokes).toHaveLength(0)
    expect(result.boardRev).toBe(5)
  })

  it('acked clear empties strokes', () => {
    const local = pendingStroke({ id: '11111111-1111-4111-8111-111111111111', sync: 'saved' })
    const result = applyIncomingMessage([local], {
      type: 'ack',
      opId: 'op-clear',
      ok: true,
      clear: true,
      boardRev: 4,
    })
    expect(result.action).toBe('acked-clear')
    expect(result.strokes).toHaveLength(0)
  })

  it('op_cancelled removes pending stroke', () => {
    const local = pendingStroke()
    const result = applyIncomingMessage([local], {
      type: 'ack',
      opId: 'op-local-1',
      ok: false,
      error: 'op_cancelled',
      boardRev: 2,
    })
    expect(result.action).toBe('nack-cancelled')
    expect(result.strokes).toHaveLength(0)
  })

  it('stale stroke echo ignored when local rev is ahead', () => {
    const local = pendingStroke()
    const result = applyIncomingMessage(
      [local],
      {
        type: 'stroke',
        opId: 'op-local-1',
        boardRev: 1,
        stroke: { ...local, id: '99999999-9999-4999-8999-999999999999' },
      },
      5,
    )
    expect(result.action).toBe('ignored-stale-ack')
    expect(result.strokes[0].id).toBeUndefined()
  })

  it('clear drops pending creates from queue; late stroke ack ignored', () => {
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    const inbound: AckMessage[] = []
    const client = createWsClient({
      onMessage: (m) => {
        if (m.type === 'ack') inbound.push(m)
      },
      WebSocketImpl: FakeWebSocket,
    })
    client.setBoardRev(2)
    client.connect('ws://test/ws')
    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)

    client.send({
      type: 'stroke',
      opId: 'op-pending',
      stroke: {
        points: [
          { x: 0, y: 0 },
          { x: 1, y: 1 },
        ],
        color: '#000',
        width: 2,
        clientId: 'c',
        startedAtUnixMs: 1,
      },
    })
    expect(client.getQueueLength()).toBe(1)
    client.dropPendingCreates()
    expect(client.getQueueLength()).toBe(0)

    // Late ack for dropped create must not re-queue
    sockets[0].onmessage?.({
      data: JSON.stringify({ type: 'ack', opId: 'op-pending', ok: true, strokeId: '99999999-9999-4999-8999-999999999990', boardRev: 3 }),
    } as MessageEvent)
    expect(client.getQueueLength()).toBe(0)
  })

  it('stamps baseRev from tracked boardRev on send', () => {
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    const client = createWsClient({
      onMessage: () => {},
      WebSocketImpl: FakeWebSocket,
    })
    client.setBoardRev(7)
    client.connect('ws://test/ws')
    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)
    client.send({ type: 'clear', opId: 'op-clear-base' })
    expect(sockets[0].send).toHaveBeenCalled()
    const payload = JSON.parse(String(sockets[0].send.mock.calls[0][0]))
    expect(payload.baseRev).toBe(7)
    expect(payload.type).toBe('clear')
  })

  it('stale_board ack invokes onStaleBoard', () => {
    const { FakeWebSocket, sockets } = installFakeWebSocket()
    let staleRev = -1
    const client = createWsClient({
      onMessage: () => {},
      onStaleBoard: (rev) => {
        staleRev = rev
      },
      WebSocketImpl: FakeWebSocket,
    })
    client.setBoardRev(1)
    client.connect('ws://test/ws')
    sockets[0].readyState = 1
    sockets[0].onopen?.({} as Event)
    client.send({ type: 'delete', opId: 'op-stale', delete: '11111111-1111-4111-8111-111111111111' })
    sockets[0].onmessage?.({
      data: JSON.stringify({
        type: 'ack',
        opId: 'op-stale',
        ok: false,
        error: 'stale_board',
        boardRev: 9,
      }),
    } as MessageEvent)
    expect(staleRev).toBe(9)
    expect(client.getBoardRev()).toBe(9)
  })
})

describe('recognize response discard', () => {
  it('accepts matching attempt and boardRev', () => {
    expect(
      shouldAcceptRecognizeResponse({
        attempt: 2,
        currentAttempt: 2,
        responseBoardRev: 5,
        requestedBoardRev: 5,
        localBoardRev: 5,
      }),
    ).toBe(true)
  })

  it('discards when boardRev moved locally', () => {
    expect(
      shouldAcceptRecognizeResponse({
        attempt: 1,
        currentAttempt: 1,
        responseBoardRev: 5,
        requestedBoardRev: 5,
        localBoardRev: 6,
      }),
    ).toBe(false)
  })

  it('discards superseded attempt', () => {
    expect(
      shouldAcceptRecognizeResponse({
        attempt: 1,
        currentAttempt: 2,
        responseBoardRev: 5,
        requestedBoardRev: 5,
        localBoardRev: 5,
      }),
    ).toBe(false)
  })
})
