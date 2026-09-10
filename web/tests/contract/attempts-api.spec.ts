import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  assessAttempt,
  createAttempt,
  getAttempt,
  getAttemptAssessment,
  submitAttempt,
} from '../../src/services/attemptsApi'

describe('attempts API contract', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('createAttempt posts characterId and optional clientAttemptId', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        id: '11111111-1111-4111-8111-111111111111',
        characterId: 'hira:あ',
        glyph: 'あ',
        status: 'draft',
        startedAt: '2026-09-09T00:00:00Z',
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const out = await createAttempt({
      characterId: 'hira:あ',
      lessonId: 'lesson:hiragana5',
      clientAttemptId: 'client-1',
    })
    expect(out.id).toBe('11111111-1111-4111-8111-111111111111')
    expect(out.status).toBe('draft')

    expect(fetchMock).toHaveBeenCalledOnce()
    const [path, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(path).toBe('/api/attempts')
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({
      characterId: 'hira:あ',
      lessonId: 'lesson:hiragana5',
      clientAttemptId: 'client-1',
    })
  })

  it('submitAttempt and assessAttempt use stable paths', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: '77777777-7777-4777-8777-777777777777',
          status: 'submitted',
          submittedAt: '2026-09-09T00:00:01Z',
          strokeCount: 2,
          width: 300,
          height: 300,
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          attemptId: '77777777-7777-4777-8777-777777777777',
          characterId: 'hira:い',
          status: 'assessed',
          pass: true,
          score: 0.9,
          scoreKind: 'match',
          assessor: 'target_compare',
          setId: 'hiragana5',
          reasons: ['top_match'],
          feedback: [],
        }),
      })
    vi.stubGlobal('fetch', fetchMock)

    const attemptId = '77777777-7777-4777-8777-777777777777'
    await submitAttempt(attemptId, {
      width: 300,
      height: 300,
      strokes: [{ points: [{ x: 1, y: 2 }] }],
    })
    await assessAttempt(attemptId)

    expect((fetchMock.mock.calls[0] as [string])[0]).toBe(`/api/attempts/${attemptId}/submit`)
    expect((fetchMock.mock.calls[1] as [string])[0]).toBe(`/api/attempts/${attemptId}/assess`)
  })

  it('surfaces API error codes from failed responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({ error: 'invalid_status', message: 'attempt is not draft' }),
      }),
    )

    await expect(getAttempt('33333333-3333-4333-8333-333333333333')).rejects.toMatchObject({
      status: 409,
      code: 'invalid_status',
    })
  })

  it('assessAttempt surfaces feedback messages from the contract', async () => {
    const attemptId = '77777777-7777-4777-8777-777777777777'
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        attemptId,
        characterId: 'hira:あ',
        glyph: 'あ',
        status: 'assessed',
        pass: false,
        score: 0.4,
        scoreKind: 'match',
        assessor: 'target_compare',
        setId: 'hiragana5',
        reasons: ['stroke_count_mismatch'],
        feedback: [
          {
            rank: 1,
            code: 'stroke_count_mismatch',
            message: 'Use 3 strokes for 「あ」 (you used 2). Assessed attempts are immutable — start a new attempt and focus on stroke count.',
          },
        ],
        candidates: [{ text: 'あ', score: 0.4, scoreKind: 'match' }],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const out = await assessAttempt(attemptId)
    expect(out.scoreKind).toBe('match')
    expect(out.feedback.length).toBeGreaterThan(0)
    expect(out.feedback.length).toBeLessThanOrEqual(2)
    for (const item of out.feedback) {
      expect(item.code).toBeTruthy()
      expect(item.message.trim().length).toBeGreaterThan(0)
    }
  })

  it('getAttemptAssessment uses GET path', async () => {
    const attemptId = '99999999-9999-4999-8999-999999999999'
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        attemptId,
        characterId: 'hira:う',
        status: 'assessed',
        pass: false,
        score: 0.2,
        scoreKind: 'match',
        assessor: 'target_compare',
        setId: 'hiragana5',
        reasons: ['empty_strokes'],
        feedback: [
          {
            rank: 1,
            code: 'empty_strokes',
            message: 'No strokes were submitted. Draw the character, then try again.',
          },
        ],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const out = await getAttemptAssessment(attemptId)
    expect(out.scoreKind).toBe('match')
    expect(out.feedback[0]?.message.trim().length).toBeGreaterThan(0)
    expect((fetchMock.mock.calls[0] as [string])[0]).toBe(`/api/attempts/${attemptId}/assessment`)
  })
})
