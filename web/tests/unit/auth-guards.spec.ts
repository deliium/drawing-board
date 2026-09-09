import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { requireAuth } from '../../src/router/guards'
import { setAuthenticatedUser, sessionContext } from '../../src/services/sessionContext'
import router from '../../src/router'
import BoardPage from '../../src/pages/BoardPage.vue'

const root = join(dirname(fileURLToPath(import.meta.url)), '../..')
const loginPageSource = readFileSync(join(root, 'src/pages/LoginPage.vue'), 'utf8')
const boardPageSource = readFileSync(join(root, 'src/pages/BoardPage.vue'), 'utf8')

describe('auth guards', () => {
  beforeEach(() => {
    setAuthenticatedUser(null)
  })

  it('redirects anonymous users from board to login', () => {
    const next = vi.fn()
    requireAuth(
      { name: 'board', meta: { requiresAuth: true }, fullPath: '/' } as never,
      {} as never,
      next,
    )
    expect(next).toHaveBeenCalledWith({ name: 'login' })
  })

  it('redirects authenticated users away from guest auth routes', () => {
    setAuthenticatedUser({ id: 1, email: 'a@example.com' })
    expect(sessionContext.state).toBe('authenticated')
    const next = vi.fn()
    requireAuth(
      { name: 'login', meta: { guestOnly: true }, fullPath: '/login' } as never,
      {} as never,
      next,
    )
    expect(next).toHaveBeenCalledWith({ name: 'board' })
  })

  it('allows anonymous access to login', () => {
    const next = vi.fn()
    requireAuth(
      { name: 'login', meta: { guestOnly: true }, fullPath: '/login' } as never,
      {} as never,
      next,
    )
    expect(next).toHaveBeenCalledWith()
  })
})

describe('auth routes', () => {
  it('includes login and register guest routes and protected board', () => {
    const names = router.getRoutes().map((route) => route.name)
    expect(names).toContain('board')
    expect(names).toContain('login')
    expect(names).toContain('register')
    const board = router.getRoutes().find((route) => route.name === 'board')
    const login = router.getRoutes().find((route) => route.name === 'login')
    const register = router.getRoutes().find((route) => route.name === 'register')
    expect(board?.meta.requiresAuth).toBe(true)
    expect(login?.meta.guestOnly).toBe(true)
    expect(register?.meta.guestOnly).toBe(true)
  })
})

describe('auth page source', () => {
  it('exposes sign-in and create-account modes with accessible labels', () => {
    expect(loginPageSource).toContain("t('auth.tab.register')")
    expect(loginPageSource).toContain("t('auth.tab.login')")
    expect(loginPageSource).toContain('for="auth-email"')
    expect(loginPageSource).toContain('autocomplete="email"')
    expect(loginPageSource).toContain('current-password')
    expect(loginPageSource).toContain('new-password')
    expect(loginPageSource).toContain('role="alert"')
    expect(loginPageSource).toContain('aria-controls="auth-panel"')
    expect(loginPageSource).toContain('role="tablist"')
  })
})

describe('board page auth cleanup', () => {
  it('does not expose login/register form controls', () => {
    expect(boardPageSource).not.toMatch(/@click="doLogin"/)
    expect(boardPageSource).not.toMatch(/@click="doRegister"/)
    expect(boardPageSource).not.toMatch(/placeholder="email"/)
    expect(boardPageSource).toContain('doLogout')
    expect(BoardPage).toBeTruthy()
  })
})
