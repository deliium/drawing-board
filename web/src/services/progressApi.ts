import { apiFetch } from './apiClient'

export type ProgressStatus = 'unseen' | 'seen' | 'practicing' | 'passed'

export type MasteryState = 'not_started' | 'learning' | 'passed_once' | 'steady'

export type MasterySummary = {
  state: MasteryState | string
  reasonCode: string
  assessedCount: number
  passCount: number
  failCount: number
  consecutivePassesEnding: number
  lastPass?: boolean
  lastAssessedAt?: string
}

export type ReviewSummary = {
  box: number
  dueAt?: string
  isDue: boolean
  intervalDays: number
}

export type ProgressItem = {
  characterId: string
  status: ProgressStatus | string
  attemptCount: number
  passCount: number
  lastAttemptId?: string
  lastPassedAt?: string
  updatedAt: string
  mastery?: MasterySummary
  review?: ReviewSummary
}

export type ProgressList = {
  items: ProgressItem[]
}

export type ListProgressQuery = {
  lessonId?: string
  setId?: string
}

export type ProgressNext = {
  lessonId: string
  characterId: string | null
  glyph?: string | null
  reasonCode: string
  masteryState?: string
  dueAt?: string | null
  reviewBox?: number | null
  nextDueAt?: string | null
  nextDueCharacterId?: string | null
}

export type ClearPracticeDataResult = {
  attemptsDeleted: number
  progressRowsCleared: number
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
    const due = (out.items ?? []).filter((i) => i.review?.isDue).length
    debug('list ok', 'count', out.items?.length ?? 0, 'due', due)
    return out
  } catch (err) {
    debug('list error', err)
    throw err
  }
}

export async function getProgressNext(lessonId?: string): Promise<ProgressNext> {
  const params = new URLSearchParams()
  if (lessonId) params.set('lessonId', lessonId)
  const qs = params.toString()
  const path = qs ? `/api/progress/next?${qs}` : '/api/progress/next'
  debug('next', path)
  try {
    const out = await apiFetch<ProgressNext>(path)
    debug('next ok', out.characterId, out.reasonCode, out.dueAt, out.nextDueAt)
    return out
  } catch (err) {
    debug('next error', err)
    throw err
  }
}

export async function clearPracticeData(): Promise<ClearPracticeDataResult> {
  debug('clearPracticeData')
  try {
    const out = await apiFetch<ClearPracticeDataResult>('/api/practice-data', {
      method: 'DELETE',
      body: '{}',
    })
    debug('clear ok', out.attemptsDeleted, out.progressRowsCleared)
    return out
  } catch (err) {
    debug('clear error', err)
    throw err
  }
}
