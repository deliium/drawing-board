import { describe, expect, it } from 'vitest'
import router from '../../src/router'

describe('US1 deep-link parity', () => {
  it('defines root board route and public auth routes', () => {
    const names = router.getRoutes().map((route) => route.name)
    expect(names).toContain('board')
    expect(names).toContain('login')
    expect(names).toContain('register')
  })
})
