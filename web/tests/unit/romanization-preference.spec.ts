import { beforeEach, describe, expect, it } from 'vitest'
import { useRomanizationPreference } from '../../src/composables/useRomanizationPreference'

describe('useRomanizationPreference', () => {
  beforeEach(() => {
    localStorage.removeItem('romanizationVisible:v1')
    useRomanizationPreference().setRomanizationVisible(true)
  })

  it('defaults to visible and persists toggle', () => {
    const a = useRomanizationPreference()
    expect(a.romanizationVisible.value).toBe(true)
    a.setRomanizationVisible(false)
    expect(localStorage.getItem('romanizationVisible:v1')).toBe('false')
    const b = useRomanizationPreference()
    expect(b.romanizationVisible.value).toBe(false)
    b.toggleRomanizationVisible()
    expect(b.romanizationVisible.value).toBe(true)
    expect(localStorage.getItem('romanizationVisible:v1')).toBe('true')
  })
})
