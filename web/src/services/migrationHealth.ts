type Metric = {
  name: string
  value: number
  at: number
}

const metrics: Metric[] = []

export function trackMetric(name: string, value = 1) {
  metrics.push({ name, value, at: Date.now() })
}

export function getMetrics() {
  return [...metrics]
}
