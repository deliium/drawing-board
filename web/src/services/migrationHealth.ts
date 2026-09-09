type Metric = {
  name: string
  value: number
  at: number
}

const metrics: Metric[] = []

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

/** DEV-only in-memory counters for local diagnostics. No-op in production builds. */
export function trackMetric(name: string, value = 1) {
  if (!isDev) return
  metrics.push({ name, value, at: Date.now() })
  console.debug('[MigrationHealth] metric', name, value)
}

export function getMetrics() {
  return [...metrics]
}
