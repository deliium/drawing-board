import { describe, expect, it } from 'vitest'
import router from '../../src/router'

describe('US3 route parity', () => {
  it('exposes the migrated board and guest auth routes', () => {
    const board = router.getRoutes().find((route) => route.name === 'board')
    const login = router.getRoutes().find((route) => route.name === 'login')
    const register = router.getRoutes().find((route) => route.name === 'register')
    expect(board?.path).toBe('/')
    expect(board?.meta.requiresAuth).toBe(true)
    expect(login?.path).toBe('/login')
    expect(login?.meta.guestOnly).toBe(true)
    expect(register?.path).toBe('/register')
    expect(register?.meta.guestOnly).toBe(true)
  })
})
