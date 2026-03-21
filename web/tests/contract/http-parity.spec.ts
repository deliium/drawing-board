import { describe, expect, it } from 'vitest'
import { apiFetch } from '../../src/services/apiClient'

describe('http parity contract', () => {
  it('api client exports stable fetch helper', () => {
    expect(typeof apiFetch).toBe('function')
  })
})
