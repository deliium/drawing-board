import { apiFetch } from './apiClient'

export type AttemptStatus = 'draft' | 'submitted' | 'assessed' | 'abandoned'

export type CreateAttemptRequest = {
  characterId: string
  lessonId?: string
  clientAttemptId?: string
}

export type Attempt = {
  id: string
  characterId: string
  glyph?: string
  lessonId?: string
  status: AttemptStatus
  clientAttemptId?: string
  startedAt: string
  submittedAt?: string
  canvasWidth?: number
  canvasHeight?: number
  strokeCount?: number
}

export type AttemptStrokePoint = { x: number; y: number }

export type AttemptStrokeInput = {
  color?: string
  width?: number
  startedAtUnixMs?: number
  points: AttemptStrokePoint[]
}

export type SubmitAttemptRequest = {
  width: number
  height: number
  strokes: AttemptStrokeInput[]
}

export type SubmitAttemptResponse = {
  id: string
  status: 'submitted'
  submittedAt: string
  strokeCount: number
  width: number
  height: number
}

export type AssessmentFeedback = {
  rank: number
  code: string
  message: string
}

export type AssessmentCandidate = {
  text: string
  score: number
  scoreKind?: string
}

export type AttemptAssessment = {
  attemptId: string
  characterId: string
  glyph?: string
  status: 'assessed'
  pass: boolean
  score: number
  scoreKind: string
  assessor: string
  setId: string
  reasons: string[]
  feedback: AssessmentFeedback[]
  candidates?: AssessmentCandidate[]
}

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function debug(...args: unknown[]) {
  if (isDev) console.debug('[attemptsApi]', ...args)
}

export function createAttempt(body: CreateAttemptRequest): Promise<Attempt> {
  debug('create', body.characterId, Boolean(body.clientAttemptId))
  return apiFetch<Attempt>('/api/attempts', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getAttempt(id: string): Promise<Attempt> {
  debug('get', id)
  return apiFetch<Attempt>(`/api/attempts/${id}`)
}

export function submitAttempt(id: string, body: SubmitAttemptRequest): Promise<SubmitAttemptResponse> {
  debug('submit', id, 'strokes', body.strokes.length)
  return apiFetch<SubmitAttemptResponse>(`/api/attempts/${id}/submit`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function assessAttempt(id: string): Promise<AttemptAssessment> {
  debug('assess', id)
  return apiFetch<AttemptAssessment>(`/api/attempts/${id}/assess`, {
    method: 'POST',
    body: '{}',
  })
}

export function getAttemptAssessment(id: string): Promise<AttemptAssessment> {
  debug('getAssessment', id)
  return apiFetch<AttemptAssessment>(`/api/attempts/${id}/assessment`)
}

export function abandonAttempt(id: string): Promise<{ id: string; status: 'abandoned' }> {
  debug('abandon', id)
  return apiFetch<{ id: string; status: 'abandoned' }>(`/api/attempts/${id}/abandon`, {
    method: 'POST',
    body: '{}',
  })
}

export type AttemptHistoryItem = {
  id: string
  characterId: string
  glyph: string
  lessonId?: string
  status: AttemptStatus
  startedAt: string
  assessedAt?: string
  pass?: boolean
  score?: number
  scoreKind?: string
  feedback?: AssessmentFeedback[]
}

export type AttemptHistoryList = {
  items: AttemptHistoryItem[]
  nextCursor?: string
  limit: number
}

export type ListAttemptsQuery = {
  lessonId?: string
  characterId?: string
  status?: string
  limit?: number
  cursor?: string
}

export async function listAttempts(query: ListAttemptsQuery = {}): Promise<AttemptHistoryList> {
  const params = new URLSearchParams()
  if (query.lessonId) params.set('lessonId', query.lessonId)
  if (query.characterId) params.set('characterId', query.characterId)
  if (query.status) params.set('status', query.status)
  if (query.limit != null) params.set('limit', String(query.limit))
  if (query.cursor) params.set('cursor', query.cursor)
  const qs = params.toString()
  const path = qs ? `/api/attempts?${qs}` : '/api/attempts'
  debug('list', path)
  try {
    const out = await apiFetch<AttemptHistoryList>(path)
    debug('list ok', 'count', out.items?.length ?? 0, 'hasNext', Boolean(out.nextCursor))
    return out
  } catch (err) {
    debug('list error', err)
    throw err
  }
}
