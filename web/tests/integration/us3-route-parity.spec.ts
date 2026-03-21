import { describe, expect, it } from 'vitest'
import router from '../../src/router'

describe('US3 route parity', () => {
  it('exposes the migrated board route', () => {
    const board = router.getRoutes().find((route) => route.name === 'board')
    expect(board?.path).toBe('/')
  })
})
