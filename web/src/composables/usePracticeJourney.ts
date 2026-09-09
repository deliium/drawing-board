import { computed, ref, type Ref } from 'vue'
import { resolveSubmitLogicalSize } from '../canvas/layout'
import { t } from '../i18n'
import type { ApiError } from '../services/apiClient'
import {
  abandonAttempt,
  assessAttempt,
  createAttempt,
  getAttempt,
  getAttemptAssessment,
  submitAttempt,
  type Attempt,
  type AttemptAssessment,
  type AttemptStrokeInput,
} from '../services/attemptsApi'
import {
  getLesson,
  HIRAGANA5_LESSON_ID,
  type Lesson,
  type LessonCharacter,
} from '../services/curriculumApi'
import { listProgress, type ProgressItem } from '../services/progressApi'
import type { Point, Stroke } from '../services/strokeSync'

export type JourneyStage =
  | 'loading'
  | 'error'
  | 'empty'
  | 'intro'
  | 'animate'
  | 'trace'
  | 'freewrite'
  | 'submitting'
  | 'result'
  | 'complete'

export type PracticeSessionSnapshot = {
  attemptId: number
  clientAttemptId: string
  characterId: string
  stage: JourneyStage
  strokes?: Stroke[]
  notice?: string
}

export type JourneyBanner = {
  kind: 'error' | 'notice' | 'info'
  message: string
} | null

const LESSON_ID = HIRAGANA5_LESSON_ID

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function journeyDebug(...args: unknown[]) {
  if (isDev) console.debug('[practiceJourney]', ...args)
}

function journeyWarn(...args: unknown[]) {
  if (isDev) console.warn('[practiceJourney]', ...args)
}

export function sessionKey(characterId: string): string {
  return `practice:v1:${characterId}`
}

export function newClientAttemptId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `attempt-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

function errMessage(err: unknown, fallback: string): string {
  if (err && typeof err === 'object' && 'message' in err) {
    const m = (err as ApiError).message
    if (typeof m === 'string' && m.trim()) return m
  }
  if (err instanceof Error && err.message) return err.message
  return fallback
}

function strokesToSubmit(strokes: Stroke[]): AttemptStrokeInput[] {
  return strokes.map((s) => ({
    color: s.color,
    width: s.width,
    startedAtUnixMs: s.startedAtUnixMs,
    points: s.points.map((p: Point) => ({ x: p.x, y: p.y })),
  }))
}

function readSession(characterId: string): PracticeSessionSnapshot | null {
  if (typeof sessionStorage === 'undefined') return null
  try {
    const raw = sessionStorage.getItem(sessionKey(characterId))
    if (!raw) return null
    return JSON.parse(raw) as PracticeSessionSnapshot
  } catch {
    return null
  }
}

function writeSession(snap: PracticeSessionSnapshot): void {
  if (typeof sessionStorage === 'undefined') return
  try {
    sessionStorage.setItem(sessionKey(snap.characterId), JSON.stringify(snap))
  } catch (err) {
    journeyWarn('session write failed', err)
  }
}

function clearSession(characterId: string): void {
  if (typeof sessionStorage === 'undefined') return
  try {
    sessionStorage.removeItem(sessionKey(characterId))
  } catch {
    // ignore
  }
}

export type UsePracticeJourneyOptions = {
  characterId: Ref<string>
  /** Live CSS-logical canvas size at submit time (from PracticeStageCanvas). */
  getLogicalSize?: () => { width: number; height: number } | null
  getCanvasElement?: () => HTMLCanvasElement | null
}

export function usePracticeJourney(options: UsePracticeJourneyOptions) {
  const stage = ref<JourneyStage>('loading')
  const banner = ref<JourneyBanner>(null)
  const lesson = ref<Lesson | null>(null)
  const character = ref<LessonCharacter | null>(null)
  const progress = ref<ProgressItem | null>(null)
  const attempt = ref<Attempt | null>(null)
  const assessment = ref<AttemptAssessment | null>(null)
  const strokes = ref<Stroke[]>([])
  const softWarn = ref<string | null>(null)
  const clientAttemptId = ref<string>('')

  function setStage(next: JourneyStage, extra?: Record<string, unknown>) {
    stage.value = next
    const codes = (assessment.value?.feedback ?? [])
      .slice(0, 2)
      .map((f) => f.code)
    journeyDebug('stage', next, {
      attemptId: attempt.value?.id,
      pass: assessment.value?.pass,
      feedbackCodes: codes.length ? codes : undefined,
      ...extra,
    })
  }

  function persistSession(overrides?: Partial<PracticeSessionSnapshot>) {
    if (!attempt.value || !options.characterId.value) return
    writeSession({
      attemptId: attempt.value.id,
      clientAttemptId: clientAttemptId.value || attempt.value.clientAttemptId || '',
      characterId: options.characterId.value,
      stage: stage.value,
      strokes: strokes.value,
      ...overrides,
    })
  }

  async function load(): Promise<void> {
    const characterId = options.characterId.value
    setStage('loading')
    banner.value = null
    softWarn.value = null
    assessment.value = null
    attempt.value = null
    strokes.value = []
    journeyDebug('load', characterId)

    if (!characterId) {
      setStage('empty')
      return
    }

    try {
      const [lessonRes, progressRes] = await Promise.all([
        getLesson(LESSON_ID),
        listProgress({ lessonId: LESSON_ID }),
      ])
      lesson.value = lessonRes
      const ch = lessonRes.characters.find((c) => c.id === characterId) ?? null
      character.value = ch
      progress.value = progressRes.items.find((p) => p.characterId === characterId) ?? null

      if (!ch) {
        setStage('empty')
        return
      }

      const session = readSession(characterId)
      if (session?.attemptId) {
        await resumeFromSession(session)
        return
      }

      if (progress.value?.status === 'passed') {
        setStage('intro')
        banner.value = {
          kind: 'info',
          message: t('practice.alreadyDone'),
        }
        return
      }

      setStage('intro')
    } catch (err) {
      journeyWarn('load failed', err)
      banner.value = { kind: 'error', message: errMessage(err, t('practice.loadError')) }
      setStage('error')
    }
  }

  async function resumeFromSession(session: PracticeSessionSnapshot): Promise<void> {
    journeyDebug('resume', session.attemptId, session.stage)
    try {
      const at = await getAttempt(session.attemptId)
      attempt.value = at
      clientAttemptId.value = session.clientAttemptId || at.clientAttemptId || ''

      if (at.status === 'draft') {
        if (session.strokes && session.strokes.length > 0) {
          strokes.value = session.strokes
        } else {
          strokes.value = []
          banner.value = {
            kind: 'notice',
            message: t('practice.drawingLost'),
          }
        }
        const restored =
          session.stage === 'trace' || session.stage === 'freewrite' || session.stage === 'animate'
            ? session.stage
            : 'freewrite'
        setStage(restored)
        return
      }

      if (at.status === 'submitted') {
        setStage('submitting')
        const assessed = await assessAttempt(at.id)
        assessment.value = assessed
        clearSession(options.characterId.value)
        setStage('result')
        return
      }

      if (at.status === 'assessed') {
        const assessed = await getAttemptAssessment(at.id)
        assessment.value = assessed
        clearSession(options.characterId.value)
        setStage('result')
        return
      }

      // abandoned or unknown — start fresh intro
      clearSession(options.characterId.value)
      attempt.value = null
      setStage('intro')
    } catch (err) {
      journeyWarn('resume failed', err)
      clearSession(options.characterId.value)
      banner.value = { kind: 'error', message: errMessage(err, t('practice.resumeError')) }
      setStage('error')
    }
  }

  async function ensureDraft(): Promise<boolean> {
    if (attempt.value?.status === 'draft') return true
    const characterId = options.characterId.value
    if (!characterId) return false
    clientAttemptId.value = newClientAttemptId()
    try {
      const created = await createAttempt({
        characterId,
        lessonId: LESSON_ID,
        clientAttemptId: clientAttemptId.value,
      })
      attempt.value = created
      journeyDebug('draft created', created.id, clientAttemptId.value)
      persistSession({ stage: 'trace' })
      return true
    } catch (err) {
      journeyWarn('create draft failed', err)
      banner.value = { kind: 'error', message: errMessage(err, t('practice.startError')) }
      return false
    }
  }

  async function start(): Promise<void> {
    banner.value = null
    setStage('animate')
  }

  async function continueFromAnimate(): Promise<void> {
    const ok = await ensureDraft()
    if (!ok) return
    strokes.value = []
    softWarn.value = null
    setStage('trace')
    persistSession({ stage: 'trace', strokes: [] })
  }

  function nextFromTrace(): void {
    const need = character.value?.strokeCount ?? 0
    if (strokes.value.length < need) {
      softWarn.value = t('practice.strokeCountSoft', {
        need,
        got: strokes.value.length,
      })
    } else {
      softWarn.value = null
    }
    if (strokes.value.length < 1) {
      softWarn.value = t('practice.drawBeforeContinue')
      return
    }
    setStage('freewrite')
    strokes.value = []
    persistSession({ stage: 'freewrite', strokes: [] })
  }

  function showOrderAgain(): void {
    setStage('animate')
    persistSession({ stage: 'animate' })
  }

  async function submit(): Promise<void> {
    if (!attempt.value || attempt.value.status !== 'draft') {
      banner.value = { kind: 'error', message: t('practice.noDraft') }
      return
    }
    if (strokes.value.length < 1) {
      softWarn.value = t('practice.drawBeforeSubmit')
      return
    }
    softWarn.value = null
    banner.value = null
    setStage('submitting')
    persistSession({ stage: 'submitting' })

    const logical = resolveSubmitLogicalSize({
      fromComposable: options.getLogicalSize?.() ?? null,
      canvas: options.getCanvasElement?.() ?? null,
      reason: 'submit',
    })
    const width = logical.width
    const height = logical.height
    try {
      const submitted = await submitAttempt(attempt.value.id, {
        width,
        height,
        strokes: strokesToSubmit(strokes.value),
      })
      attempt.value = { ...attempt.value, status: submitted.status }
      persistSession({ stage: 'submitting' })
      const assessed = await assessAttempt(attempt.value.id)
      assessment.value = assessed
      attempt.value = { ...attempt.value, status: 'assessed' }
      clearSession(options.characterId.value)
      journeyDebug('assessed', {
        pass: assessed.pass,
        score: assessed.score,
        feedbackCodes: assessed.feedback?.map((f) => f.code),
        canvas: { width, height },
      })
      setStage('result')
    } catch (err) {
      journeyWarn('submit/assess failed', err)
      banner.value = {
        kind: 'error',
        message: errMessage(err, t('practice.submitError')),
      }
      const status = attempt.value?.status
      if (status === 'submitted') {
        setStage('submitting')
      } else {
        setStage('freewrite')
        persistSession({ stage: 'freewrite' })
      }
    }
  }

  async function retryAssess(): Promise<void> {
    if (!attempt.value) return
    banner.value = null
    setStage('submitting')
    try {
      const assessed = await assessAttempt(attempt.value.id)
      assessment.value = assessed
      attempt.value = { ...attempt.value, status: 'assessed' }
      clearSession(options.characterId.value)
      setStage('result')
    } catch (err) {
      journeyWarn('retry assess failed', err)
      banner.value = {
        kind: 'error',
        message: errMessage(err, t('practice.submitError')),
      }
    }
  }

  async function retry(): Promise<void> {
    banner.value = null
    assessment.value = null
    strokes.value = []
    softWarn.value = null
    attempt.value = null
    const ok = await ensureDraft()
    if (!ok) return
    setStage('trace')
    persistSession({ stage: 'trace', strokes: [] })
  }

  function complete(): void {
    clearSession(options.characterId.value)
    setStage('complete')
  }

  async function cancelPractice(): Promise<void> {
    const at = attempt.value
    clearSession(options.characterId.value)
    if (at && at.status === 'draft') {
      try {
        await abandonAttempt(at.id)
        journeyDebug('abandoned', at.id)
      } catch (err) {
        journeyWarn('abandon failed (best-effort)', err)
      }
    }
    attempt.value = null
    strokes.value = []
    assessment.value = null
    setStage('intro')
  }

  function onStrokesChanged(next: Stroke[]): void {
    strokes.value = next
    if (stage.value === 'trace' || stage.value === 'freewrite') {
      persistSession({ strokes: next })
    }
  }

  const canNextFromTrace = computed(() => strokes.value.length >= 1)

  const feedback = computed(() => {
    const items = assessment.value?.feedback ?? []
    return items.slice(0, 2).filter((f) => f.code || (f.message && f.message.trim()))
  })

  return {
    stage,
    banner,
    lesson,
    character,
    progress,
    attempt,
    assessment,
    strokes,
    softWarn,
    clientAttemptId,
    canNextFromTrace,
    feedback,
    load,
    start,
    continueFromAnimate,
    nextFromTrace,
    showOrderAgain,
    submit,
    retryAssess,
    retry,
    complete,
    cancelPractice,
    onStrokesChanged,
    persistSession,
    setStage,
  }
}
