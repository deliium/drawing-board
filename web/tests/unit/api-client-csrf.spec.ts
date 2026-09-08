import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch, readCookie } from '../../src/services/apiClient'

describe('apiClient CSRF', () => {
  beforeEach(() => {
    document.cookie = 'csrf=; Max-Age=0; path=/'
    vi.restoreAllMocks()
  })

  it('readCookie returns csrf value', () => {
    document.cookie = 'csrf=test-token-value; path=/'
    expect(readCookie('csrf')).toBe('test-token-value')
  })

  it('attaches X-CSRF-Token on mutating requests', async () => {
    document.cookie = 'csrf=abc123token; path=/'
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ok: true }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await apiFetch('/api/logout', { method: 'POST', body: '{}' })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const init = fetchMock.mock.calls[0][1] as RequestInit
    const headers = new Headers(init.headers)
    expect(headers.get('X-CSRF-Token')).toBe('abc123token')
    expect(init.credentials).toBe('include')
  })

  it('does not attach CSRF header on GET', async () => {
    document.cookie = 'csrf=abc123token; path=/'
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ id: 1 }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await apiFetch('/api/me')

    const init = fetchMock.mock.calls[0][1] as RequestInit
    const headers = new Headers(init.headers)
    expect(headers.get('X-CSRF-Token')).toBeNull()
  })
})
