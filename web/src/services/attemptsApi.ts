import { apiFetch } from './apiClient'

export type AttemptStatus = 'draft' | 'submitted' | 'assessed' | 'abandoned'

export type CreateAttemptRequest = {
  characterId: string
  lessonId?: string
  clientAttemptId?: string
}

export type Attempt = {
  id: number
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
  id: number
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
  attemptId: number
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

export function getAttempt(id: number): Promise<Attempt> {
  debug('get', id)
  return apiFetch<Attempt>(`/api/attempts/${id}`)
}

export function submitAttempt(id: number, body: SubmitAttemptRequest): Promise<SubmitAttemptResponse> {
  debug('submit', id, 'strokes', body.strokes.length)
  return apiFetch<SubmitAttemptResponse>(`/api/attempts/${id}/submit`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function assessAttempt(id: number): Promise<AttemptAssessment> {
  debug('assess', id)
  return apiFetch<AttemptAssessment>(`/api/attempts/${id}/assess`, {
    method: 'POST',
    body: '{}',
  })
}

export function getAttemptAssessment(id: number): Promise<AttemptAssessment> {
  debug('getAssessment', id)
  return apiFetch<AttemptAssessment>(`/api/attempts/${id}/assessment`)
}

export function abandonAttempt(id: number): Promise<{ id: number; status: 'abandoned' }> {
  debug('abandon', id)
  return apiFetch<{ id: number; status: 'abandoned' }>(`/api/attempts/${id}/abandon`, {
    method: 'POST',
    body: '{}',
  })
}
