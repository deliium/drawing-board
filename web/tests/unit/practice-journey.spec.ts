import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import {
  newClientAttemptId,
  sessionKey,
  usePracticeJourney,
  type PracticeSessionSnapshot,
} from '../../src/composables/usePracticeJourney'

vi.mock('../../src/services/curriculumApi', () => ({
  HIRAGANA5_LESSON_ID: 'lesson:hiragana5',
  getLesson: vi.fn(),
}))

vi.mock('../../src/services/progressApi', () => ({
  listProgress: vi.fn(),
}))

vi.mock('../../src/services/attemptsApi', () => ({
  createAttempt: vi.fn(),
  getAttempt: vi.fn(),
  submitAttempt: vi.fn(),
  assessAttempt: vi.fn(),
  getAttemptAssessment: vi.fn(),
  abandonAttempt: vi.fn(),
}))

import { getLesson } from '../../src/services/curriculumApi'
import { listProgress } from '../../src/services/progressApi'
import {
  abandonAttempt,
  assessAttempt,
  createAttempt,
  getAttempt,
  getAttemptAssessment,
  submitAttempt,
} from '../../src/services/attemptsApi'

const lessonFixture = {
  id: 'lesson:hiragana5',
  code: 'hiragana5',
  title: 'Hiragana vowels',
  setId: 'hiragana5',
  contentVersion: 'hiragana5-content-v2',
  characters: [
    {
      id: 'hira:あ',
      glyph: 'あ',
      romanization: 'a',
      strokeCount: 3,
      pronunciation: { ipa: '/a/' },
      descriptionEn: 'Open vowel',
      example: { word: 'あさ', romanization: 'asa', meaningEn: 'morning' },
      sortKey: 1,
      position: 1,
    },
  ],
}

function mountJourney(characterId = 'hira:あ') {
  const scope = effectScope()
  const id = ref(characterId)
  const api = scope.run(() => usePracticeJourney({ characterId: id }))!
  return { scope, id, api, dispose: () => scope.stop() }
}

describe('usePracticeJourney', () => {
  let harness: ReturnType<typeof mountJourney> | null = null

  beforeEach(() => {
    sessionStorage.clear()
    vi.mocked(getLesson).mockResolvedValue(lessonFixture as never)
    vi.mocked(listProgress).mockResolvedValue({ items: [] })
  })

  afterEach(() => {
    harness?.dispose()
    harness = null
    vi.clearAllMocks()
    sessionStorage.clear()
  })

  it('loads intro for a known character', async () => {
    harness = mountJourney()
    await harness.api.load()
    expect(harness.api.stage.value).toBe('intro')
    expect(harness.api.character.value?.glyph).toBe('あ')
    expect(harness.api.character.value?.example.word).toBe('あさ')
  })

  it('empty stage for unknown character', async () => {
    harness = mountJourney('hira:か')
    await harness.api.load()
    expect(harness.api.stage.value).toBe('empty')
  })

  it('error stage when lesson fetch fails', async () => {
    vi.mocked(getLesson).mockRejectedValue({ status: 500, message: 'boom' })
    harness = mountJourney()
    await harness.api.load()
    expect(harness.api.stage.value).toBe('error')
    expect(harness.api.banner.value?.kind).toBe('error')
  })

  it('happy path submit → result with ≤2 feedback', async () => {
    vi.mocked(createAttempt).mockResolvedValue({
      id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: '2026-09-09T00:00:00Z',
      clientAttemptId: 'c1',
    })
    vi.mocked(submitAttempt).mockResolvedValue({
      id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      status: 'submitted',
      submittedAt: '2026-09-09T00:01:00Z',
      strokeCount: 3,
      width: 300,
      height: 300,
    })
    vi.mocked(assessAttempt).mockResolvedValue({
      attemptId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      characterId: 'hira:あ',
      status: 'assessed',
      pass: true,
      score: 0.92,
      scoreKind: 'match',
      assessor: 'target_compare',
      setId: 'hiragana5',
      reasons: [],
      feedback: [],
    })

    harness = mountJourney()
    await harness.api.load()
    await harness.api.start()
    expect(harness.api.stage.value).toBe('animate')
    await harness.api.continueFromAnimate()
    expect(harness.api.stage.value).toBe('trace')
    expect(createAttempt).toHaveBeenCalledOnce()

    harness.api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 's1', startedAtUnixMs: 1 },
      { points: [{ x: 2, y: 2 }], color: '#000', width: 2, clientId: 's2', startedAtUnixMs: 2 },
      { points: [{ x: 3, y: 3 }], color: '#000', width: 2, clientId: 's3', startedAtUnixMs: 3 },
    ])
    harness.api.nextFromTrace()
    expect(harness.api.stage.value).toBe('freewrite')
    harness.api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 'f1', startedAtUnixMs: 4 },
    ])
    await harness.api.submit()
    expect(harness.api.stage.value).toBe('result')
    expect(harness.api.assessment.value?.pass).toBe(true)
    expect(harness.api.feedback.value.length).toBeLessThanOrEqual(2)
  })

  it('retry creates a new clientAttemptId', async () => {
    vi.mocked(createAttempt)
      .mockResolvedValueOnce({
        id: '11111111-1111-4111-8111-111111111111',
        characterId: 'hira:あ',
        status: 'draft',
        startedAt: 't',
        clientAttemptId: 'old',
      })
      .mockResolvedValueOnce({
        id: '22222222-2222-4222-8222-222222222222',
        characterId: 'hira:あ',
        status: 'draft',
        startedAt: 't',
        clientAttemptId: 'new',
      })

    harness = mountJourney()
    await harness.api.load()
    await harness.api.start()
    await harness.api.continueFromAnimate()
    const first = harness.api.clientAttemptId.value
    harness.api.assessment.value = {
      attemptId: '11111111-1111-4111-8111-111111111111',
      characterId: 'hira:あ',
      status: 'assessed',
      pass: false,
      score: 0.2,
      scoreKind: 'match',
      assessor: 'target_compare',
      setId: 'hiragana5',
      reasons: [],
      feedback: [{ rank: 1, code: 'stroke_count_mismatch', message: 'Use 3 strokes.' }],
    }
    harness.api.setStage('result')
    await harness.api.retry()
    expect(harness.api.stage.value).toBe('trace')
    expect(createAttempt).toHaveBeenCalledTimes(2)
    expect(harness.api.clientAttemptId.value).not.toBe(first)
    expect(harness.api.clientAttemptId.value.length).toBeGreaterThan(0)
  })

  it('resumes draft from sessionStorage', async () => {
    const snap: PracticeSessionSnapshot = {
      attemptId: '42424242-4242-4242-8242-424242424242',
      clientAttemptId: 'resume-1',
      characterId: 'hira:あ',
      stage: 'freewrite',
      strokes: [
        { points: [{ x: 9, y: 9 }], color: '#000', width: 2, clientId: 'r', startedAtUnixMs: 1 },
      ],
    }
    sessionStorage.setItem(sessionKey('hira:あ'), JSON.stringify(snap))
    vi.mocked(getAttempt).mockResolvedValue({
      id: '42424242-4242-4242-8242-424242424242',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'resume-1',
    })

    harness = mountJourney()
    await harness.api.load()
    expect(harness.api.stage.value).toBe('freewrite')
    expect(harness.api.strokes.value).toHaveLength(1)
  })

  it('resumes draft with empty strokes and shows not-saved notice', async () => {
    const snap: PracticeSessionSnapshot = {
      attemptId: '43434343-4343-4343-8343-434343434343',
      clientAttemptId: 'resume-empty',
      characterId: 'hira:あ',
      stage: 'freewrite',
      // strokes omitted — session lost ink
    }
    sessionStorage.setItem(sessionKey('hira:あ'), JSON.stringify(snap))
    vi.mocked(getAttempt).mockResolvedValue({
      id: '43434343-4343-4343-8343-434343434343',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'resume-empty',
    })

    harness = mountJourney()
    await harness.api.load()
    expect(harness.api.stage.value).toBe('freewrite')
    expect(harness.api.strokes.value).toHaveLength(0)
    expect(harness.api.banner.value?.kind).toBe('notice')
    expect(harness.api.banner.value?.message).toMatch(/Drawing was not saved/i)
  })

  it('submit failure stays on freewrite without claiming pass', async () => {
    vi.mocked(createAttempt).mockResolvedValue({
      id: '50505050-5050-4505-8505-505050505050',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'c-fail-submit',
    })
    vi.mocked(submitAttempt).mockRejectedValue({ status: 500, message: 'submit failed' })

    harness = mountJourney()
    await harness.api.load()
    await harness.api.start()
    await harness.api.continueFromAnimate()
    harness.api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 't1', startedAtUnixMs: 1 },
    ])
    harness.api.nextFromTrace()
    harness.api.onStrokesChanged([
      { points: [{ x: 2, y: 2 }], color: '#000', width: 2, clientId: 'f1', startedAtUnixMs: 2 },
    ])
    await harness.api.submit()

    expect(harness.api.stage.value).toBe('freewrite')
    expect(harness.api.assessment.value).toBeNull()
    expect(harness.api.banner.value?.kind).toBe('error')
    expect(harness.api.banner.value?.message).toMatch(/submit failed|Could not check/i)
    expect(assessAttempt).not.toHaveBeenCalled()
  })

  it('assess failure after submit stays submitting; retryAssess recovers', async () => {
    vi.mocked(createAttempt).mockResolvedValue({
      id: '51515151-5151-4515-8515-515151515151',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: 't',
      clientAttemptId: 'c-fail-assess',
    })
    vi.mocked(submitAttempt).mockResolvedValue({
      id: '51515151-5151-4515-8515-515151515151',
      status: 'submitted',
      submittedAt: 't',
      strokeCount: 1,
      width: 300,
      height: 300,
    })
    vi.mocked(assessAttempt)
      .mockRejectedValueOnce({ status: 429, message: 'rate limited' })
      .mockResolvedValueOnce({
        attemptId: '51515151-5151-4515-8515-515151515151',
        characterId: 'hira:あ',
        status: 'assessed',
        pass: true,
        score: 0.9,
        scoreKind: 'match',
        assessor: 'target_compare',
        setId: 'hiragana5',
        reasons: [],
        feedback: [],
      })

    harness = mountJourney()
    await harness.api.load()
    await harness.api.start()
    await harness.api.continueFromAnimate()
    harness.api.onStrokesChanged([
      { points: [{ x: 1, y: 1 }], color: '#000', width: 2, clientId: 't1', startedAtUnixMs: 1 },
    ])
    harness.api.nextFromTrace()
    harness.api.onStrokesChanged([
      { points: [{ x: 2, y: 2 }], color: '#000', width: 2, clientId: 'f1', startedAtUnixMs: 2 },
    ])
    await harness.api.submit()

    expect(harness.api.stage.value).toBe('submitting')
    expect(harness.api.assessment.value).toBeNull()
    expect(harness.api.banner.value?.kind).toBe('error')
    expect(harness.api.attempt.value?.status).toBe('submitted')

    await harness.api.retryAssess()
    expect(harness.api.stage.value).toBe('result')
    expect(harness.api.assessment.value?.pass).toBe(true)
  })

  it('resumes submitted attempt by assessing idempotently', async () => {
    sessionStorage.setItem(
      sessionKey('hira:あ'),
      JSON.stringify({
        attemptId: '77777777-7777-4777-8777-777777777777',
        clientAttemptId: 'mid',
        characterId: 'hira:あ',
        stage: 'submitting',
      }),
    )
    vi.mocked(getAttempt).mockResolvedValue({
      id: '77777777-7777-4777-8777-777777777777',
      characterId: 'hira:あ',
      status: 'submitted',
      startedAt: 't',
    })
    vi.mocked(assessAttempt).mockResolvedValue({
      attemptId: '77777777-7777-4777-8777-777777777777',
      characterId: 'hira:あ',
      status: 'assessed',
      pass: false,
      score: 0.3,
      scoreKind: 'match',
      assessor: 'target_compare',
      setId: 'hiragana5',
      reasons: [],
      feedback: [{ rank: 1, code: 'shape_mismatch', message: 'Check the shape.' }],
    })

    harness = mountJourney()
    await harness.api.load()
    expect(assessAttempt).toHaveBeenCalledWith('77777777-7777-4777-8777-777777777777')
    expect(harness.api.stage.value).toBe('result')
  })

  it('resumes assessed via GET assessment', async () => {
    sessionStorage.setItem(
      sessionKey('hira:あ'),
      JSON.stringify({
        attemptId: '99999999-9999-4999-8999-999999999999',
        clientAttemptId: 'done',
        characterId: 'hira:あ',
        stage: 'result',
      }),
    )
    vi.mocked(getAttempt).mockResolvedValue({
      id: '99999999-9999-4999-8999-999999999999',
      characterId: 'hira:あ',
      status: 'assessed',
      startedAt: 't',
    })
    vi.mocked(getAttemptAssessment).mockResolvedValue({
      attemptId: '99999999-9999-4999-8999-999999999999',
      characterId: 'hira:あ',
      status: 'assessed',
      pass: true,
      score: 0.88,
      scoreKind: 'match',
      assessor: 'target_compare',
      setId: 'hiragana5',
      reasons: [],
      feedback: [],
    })

    harness = mountJourney()
    await harness.api.load()
    expect(getAttemptAssessment).toHaveBeenCalledWith('99999999-9999-4999-8999-999999999999')
    expect(harness.api.stage.value).toBe('result')
    expect(harness.api.assessment.value?.pass).toBe(true)
  })

  it('cancel abandons draft and clears session', async () => {
    vi.mocked(createAttempt).mockResolvedValue({
      id: '33333333-3333-4333-8333-333333333333',
      characterId: 'hira:あ',
      status: 'draft',
      startedAt: 't',
    })
    vi.mocked(abandonAttempt).mockResolvedValue({ id: '33333333-3333-4333-8333-333333333333', status: 'abandoned' })

    harness = mountJourney()
    await harness.api.load()
    await harness.api.start()
    await harness.api.continueFromAnimate()
    expect(sessionStorage.getItem(sessionKey('hira:あ'))).toBeTruthy()
    await harness.api.cancelPractice()
    expect(abandonAttempt).toHaveBeenCalledWith('33333333-3333-4333-8333-333333333333')
    expect(sessionStorage.getItem(sessionKey('hira:あ'))).toBeNull()
    expect(harness.api.stage.value).toBe('intro')
  })

  it('newClientAttemptId returns a non-empty id', () => {
    expect(newClientAttemptId().length).toBeGreaterThan(4)
  })
})
