import { afterEach, describe, expect, it, vi } from 'vitest'
import { getLesson, HIRAGANA5_LESSON_ID } from '../../src/services/curriculumApi'
import { listProgress } from '../../src/services/progressApi'

describe('curriculum + progress API contract', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('getLesson GETs encoded lesson id and returns pedagogy fields', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        id: 'lesson:hiragana5',
        code: 'hiragana5',
        title: 'Hiragana vowels (あいうえお)',
        setId: 'hiragana5',
        contentVersion: 'hiragana5-content-v2',
        characters: [
          {
            id: 'hira:あ',
            glyph: 'あ',
            romanization: 'a',
            strokeCount: 3,
            pronunciation: { ipa: '/a/', audioRef: null },
            descriptionEn: 'Open vowel',
            example: { word: 'あさ', romanization: 'asa', meaningEn: 'morning' },
            sortKey: 1,
            position: 1,
          },
        ],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const out = await getLesson(HIRAGANA5_LESSON_ID)
    expect(out.characters[0]?.strokeCount).toBe(3)
    expect(out.characters[0]?.example.word).toBe('あさ')
    expect((fetchMock.mock.calls[0] as [string])[0]).toBe(
      `/api/lessons/${encodeURIComponent('lesson:hiragana5')}`,
    )
  })

  it('listProgress supports lessonId and setId query filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        items: [
          {
            characterId: 'hira:あ',
            status: 'passed',
            attemptCount: 1,
            passCount: 1,
            updatedAt: '2026-09-09T00:00:00Z',
          },
        ],
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const byLesson = await listProgress({ lessonId: 'lesson:hiragana5' })
    expect(byLesson.items[0]?.status).toBe('passed')
    expect((fetchMock.mock.calls[0] as [string])[0]).toBe(
      '/api/progress?lessonId=lesson%3Ahiragana5',
    )

    await listProgress({ setId: 'hiragana5' })
    expect((fetchMock.mock.calls[1] as [string])[0]).toBe('/api/progress?setId=hiragana5')
  })
})
