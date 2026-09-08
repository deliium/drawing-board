import { describe, expect, it } from 'vitest'
import { createWsClient, MAX_PENDING_OPS } from '../../src/services/wsClient'

describe('ws parity contract', () => {
  it('creates client with reliability methods', () => {
    const client = createWsClient({ onMessage: () => {} })
    expect(typeof client.connect).toBe('function')
    expect(typeof client.send).toBe('function')
    expect(typeof client.close).toBe('function')
    expect(typeof client.clearPending).toBe('function')
    expect(typeof client.dropOp).toBe('function')
    expect(typeof client.getStatus).toBe('function')
    expect(MAX_PENDING_OPS).toBe(32)
  })

  it('rejects mutating messages without opId', () => {
    const client = createWsClient({ onMessage: () => {} })
    const ok = client.send({
      type: 'delete',
      opId: '',
      delete: 1,
    })
    expect(ok).toBe(false)
  })
})
