import { apiFetch } from './apiClient'

export type FeatureFlags = {
  practice: boolean
  progress: boolean
  review: boolean
  audio: boolean
}

const ALL_ON: FeatureFlags = {
  practice: true,
  progress: true,
  review: true,
  audio: true,
}

let cached: FeatureFlags = { ...ALL_ON }
const listeners = new Set<(f: FeatureFlags) => void>()

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function notify() {
  for (const fn of listeners) fn(cached)
}

/** Current feature flags (defaults all-on until /api/features loads). */
export function getFeatureFlags(): FeatureFlags {
  return cached
}

export function setFeatureFlagsForTest(flags: Partial<FeatureFlags>) {
  cached = { ...cached, ...flags }
  notify()
}

export function resetFeatureFlagsForTest() {
  cached = { ...ALL_ON }
  notify()
}

export function subscribeFeatureFlags(fn: (f: FeatureFlags) => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

/** Fetch authenticated feature kill switches; soft-fail keeps last/default. */
export async function loadFeatureFlags(): Promise<FeatureFlags> {
  try {
    const next = await apiFetch<FeatureFlags>('/api/features')
    cached = {
      practice: Boolean(next.practice),
      progress: Boolean(next.progress),
      review: Boolean(next.review),
      audio: Boolean(next.audio),
    }
    if (isDev) console.debug('[features] loaded', cached)
    notify()
  } catch (err) {
    if (isDev) console.debug('[features] load failed; keeping', cached, err)
  }
  return cached
}
