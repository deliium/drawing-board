import { afterEach, describe, expect, it, vi } from 'vitest'
import { listAttempts } from '../../src/services/attemptsApi'
import {
  clearPracticeData,
  getProgressNext,
  listProgress,
} from '../../src/services/progressApi'

describe('attempt history + mastery API contract', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('listProgress returns mastery projection fields', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        items: [
          {
            characterId: 'hira:あ',
            status: 'passed',
            attemptCount: 2,
            passCount: 2,
            updatedAt: '2026-09-09T00:00:00Z',
            mastery: {
              state: 'steady',
              reasonCode: 'two_consecutive_passes',
              assessedCount: 2,
              passCount: 2,
              failCount: 0,
              consecutivePassesEnding: 2,
            },
            review: {
              box: 2,
              dueAt: '2026-09-12T00:00:00Z',
              isDue: false,
              intervalDays: 3,
            },
          },
        ],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const out = await listProgress({ lessonId: 'lesson:hiragana5' })
    expect(out.items[0]?.mastery?.state).toBe('steady')
    expect(out.items[0]?.review?.box).toBe(2)
    expect(out.items[0]?.review?.intervalDays).toBe(3)
  })

  it('getProgressNext and clearPracticeData hit expected paths', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          lessonId: 'lesson:hiragana5',
          characterId: 'hira:う',
          glyph: 'う',
          reasonCode: 'due_review',
          masteryState: 'steady',
          dueAt: '2026-09-08T12:00:00Z',
          reviewBox: 2,
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ attemptsDeleted: 3, progressRowsCleared: 1 }),
      })
    vi.stubGlobal('fetch', fetchMock)

    const next = await getProgressNext('lesson:hiragana5')
    expect(next.characterId).toBe('hira:う')
    expect(next.reasonCode).toBe('due_review')
    expect(next.dueAt).toBe('2026-09-08T12:00:00Z')
    expect((fetchMock.mock.calls[0] as [string])[0]).toBe(
      '/api/progress/next?lessonId=lesson%3Ahiragana5',
    )

    const cleared = await clearPracticeData()
    expect(cleared.attemptsDeleted).toBe(3)
    const clearCall = fetchMock.mock.calls[1] as [string, RequestInit]
    expect(clearCall[0]).toBe('/api/practice-data')
    expect(clearCall[1]?.method).toBe('DELETE')
  })

  it('listAttempts paginates with cursor and omits points in typed items', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          items: [
            {
              id: '99999999-9999-4999-8999-999999999999',
              characterId: 'hira:あ',
              glyph: 'あ',
              status: 'assessed',
              startedAt: '2026-09-09T12:00:00Z',
              pass: true,
              score: 0.8,
              scoreKind: 'match',
              feedback: [{ rank: 1, code: 'shape', message: 'shape tip' }],
            },
          ],
          nextCursor: 'abc',
          limit: 1,
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ items: [], limit: 1 }),
      })
    vi.stubGlobal('fetch', fetchMock)

    const page1 = await listAttempts({ limit: 1 })
    expect(page1.items[0]?.glyph).toBe('あ')
    expect(page1.nextCursor).toBe('abc')
    expect(Object.prototype.hasOwnProperty.call(page1.items[0], 'points')).toBe(false)

    await listAttempts({ limit: 1, cursor: 'abc', characterId: 'hira:あ' })
    expect((fetchMock.mock.calls[1] as [string])[0]).toContain('cursor=abc')
    expect((fetchMock.mock.calls[1] as [string])[0]).toContain('characterId=hira')
  })
})
