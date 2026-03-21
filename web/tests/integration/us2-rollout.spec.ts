import { describe, expect, it } from 'vitest'
import { getMetrics, trackMetric } from '../../src/services/migrationHealth'

describe('US2 rollout / health signals', () => {
  it('records migration health metrics', () => {
    const before = getMetrics().length
    trackMetric('test.signal', 1)
    expect(getMetrics().length).toBeGreaterThan(before)
  })
})
