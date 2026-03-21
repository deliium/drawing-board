import { describe, expect, it } from 'vitest'
import router from '../../src/router'

describe('US1 core migration workflow', () => {
  it('exposes authenticated board route for core workflow', () => {
    const board = router.getRoutes().find((route) => route.name === 'board')
    expect(board?.meta?.requiresAuth).toBe(true)
  })
})
