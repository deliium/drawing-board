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
        id: 1,
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
    expect(out.id).toBe(1)
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
          id: 7,
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
          attemptId: 7,
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

    await submitAttempt(7, {
      width: 300,
      height: 300,
      strokes: [{ points: [{ x: 1, y: 2 }] }],
    })
    await assessAttempt(7)

    expect((fetchMock.mock.calls[0] as [string])[0]).toBe('/api/attempts/7/submit')
    expect((fetchMock.mock.calls[1] as [string])[0]).toBe('/api/attempts/7/assess')
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

    await expect(getAttempt(3)).rejects.toMatchObject({
      status: 409,
      code: 'invalid_status',
    })
  })

  it('getAttemptAssessment uses GET path', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        attemptId: 9,
        characterId: 'hira:う',
        status: 'assessed',
        pass: false,
        score: 0.2,
        scoreKind: 'match',
        assessor: 'target_compare',
        setId: 'hiragana5',
        reasons: ['empty_strokes'],
        feedback: [{ rank: 1, code: 'empty_strokes', message: '' }],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const out = await getAttemptAssessment(9)
    expect(out.scoreKind).toBe('match')
    expect((fetchMock.mock.calls[0] as [string])[0]).toBe('/api/attempts/9/assessment')
  })
})
