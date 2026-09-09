import { apiFetch } from './apiClient'

export type ProgressStatus = 'unseen' | 'seen' | 'practicing' | 'passed'

export type ProgressItem = {
  characterId: string
  status: ProgressStatus | string
  attemptCount: number
  passCount: number
  lastAttemptId?: number
  lastPassedAt?: string
  updatedAt: string
}

export type ProgressList = {
  items: ProgressItem[]
}

export type ListProgressQuery = {
  lessonId?: string
  setId?: string
}

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function debug(...args: unknown[]) {
  if (isDev) console.debug('[progressApi]', ...args)
}

export async function listProgress(query: ListProgressQuery = {}): Promise<ProgressList> {
  const params = new URLSearchParams()
  if (query.lessonId) params.set('lessonId', query.lessonId)
  if (query.setId) params.set('setId', query.setId)
  const qs = params.toString()
  const path = qs ? `/api/progress?${qs}` : '/api/progress'
  debug('list', path)
  try {
    const out = await apiFetch<ProgressList>(path)
    debug('list ok', 'count', out.items?.length ?? 0)
    return out
  } catch (err) {
    debug('list error', err)
    throw err
  }
}
