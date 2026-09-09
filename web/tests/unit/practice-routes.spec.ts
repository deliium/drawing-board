import { describe, expect, it } from 'vitest'
import router from '../../src/router'
import { requireAuth } from '../../src/router/guards'
import { setAuthenticatedUser } from '../../src/services/sessionContext'
import { vi } from 'vitest'

describe('practice route surface', () => {
  it('registers auth-gated practice hub, history, and character routes', () => {
    const hub = router.getRoutes().find((r) => r.name === 'practice-hub')
    const history = router.getRoutes().find((r) => r.name === 'practice-history')
    const character = router.getRoutes().find((r) => r.name === 'practice-character')
    const login = router.getRoutes().find((r) => r.name === 'login')
    const register = router.getRoutes().find((r) => r.name === 'register')
    expect(hub?.path).toBe('/practice')
    expect(hub?.meta.requiresAuth).toBe(true)
    expect(history?.path).toBe('/practice/history')
    expect(history?.meta.requiresAuth).toBe(true)
    expect(character?.path).toBe('/practice/:characterId')
    expect(character?.meta.requiresAuth).toBe(true)
    expect(login?.meta.guestShell).toBe(true)
    expect(register?.meta.guestShell).toBe(true)
  })

  it('redirects anonymous users from practice hub to login', () => {
    setAuthenticatedUser(null)
    const next = vi.fn()
    requireAuth(
      { name: 'practice-hub', meta: { requiresAuth: true }, fullPath: '/practice' } as never,
      {} as never,
      next,
    )
    expect(next).toHaveBeenCalledWith({ name: 'login' })
  })
})
