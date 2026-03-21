import { describe, expect, it } from 'vitest'
import router from '../../src/router'

describe('US1 deep-link parity', () => {
  it('defines root board route and login route', () => {
    const names = router.getRoutes().map((route) => route.name)
    expect(names).toContain('board')
    expect(names).toContain('login')
  })
})
