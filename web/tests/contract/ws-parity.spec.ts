import { describe, expect, it } from 'vitest'
import { createWsClient } from '../../src/services/wsClient'

describe('ws parity contract', () => {
  it('creates client with send/connect/close methods', () => {
    const client = createWsClient(() => {})
    expect(typeof client.connect).toBe('function')
    expect(typeof client.send).toBe('function')
    expect(typeof client.close).toBe('function')
  })
})
