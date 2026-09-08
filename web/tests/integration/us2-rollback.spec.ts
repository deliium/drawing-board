import { describe, expect, it } from 'vitest'
import { createWsClient } from '../../src/services/wsClient'

describe('US2 realtime client lifecycle', () => {
  it('supports explicit websocket close for teardown', () => {
    const client = createWsClient({ onMessage: () => {} })
    expect(() => client.close()).not.toThrow()
  })
})
