import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import PracticeHistoryPage from '../../src/pages/PracticeHistoryPage.vue'
import PracticeHubPage from '../../src/pages/PracticeHubPage.vue'
import { initLocale, setLocale, t } from '../../src/i18n'

vi.mock('../../src/services/curriculumApi', () => ({
  HIRAGANA5_LESSON_ID: 'lesson:hiragana5',
  getLesson: vi.fn(),
}))

vi.mock('../../src/services/progressApi', () => ({
  listProgress: vi.fn(),
  getProgressNext: vi.fn(),
  clearPracticeData: vi.fn(),
}))

vi.mock('../../src/services/attemptsApi', () => ({
  listAttempts: vi.fn(),
}))

import { getLesson } from '../../src/services/curriculumApi'
import { clearPracticeData, getProgressNext, listProgress } from '../../src/services/progressApi'
import { listAttempts } from '../../src/services/attemptsApi'

async function flush() {
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

describe('practice hub mastery + history', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    initLocale()
    setLocale('en')
    vi.mocked(getLesson).mockResolvedValue({
      id: 'lesson:hiragana5',
      code: 'hiragana5',
      title: 'Hiragana vowels',
      setId: 'hiragana5',
      contentVersion: 'v1',
      characters: [
        {
          id: 'hira:あ',
          glyph: 'あ',
          romanization: 'a',
          strokeCount: 3,
          pronunciation: {},
          descriptionEn: 'a',
          example: { word: 'あさ', romanization: 'asa', meaningEn: 'morning' },
          sortKey: 1,
          position: 1,
        },
      ],
    })
    vi.mocked(listProgress).mockResolvedValue({
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
            dueAt: '2026-09-01T00:00:00Z',
            isDue: true,
            intervalDays: 3,
          },
        },
      ],
    })
    vi.mocked(getProgressNext).mockResolvedValue({
      lessonId: 'lesson:hiragana5',
      characterId: 'hira:あ',
      glyph: 'あ',
      reasonCode: 'due_review',
      masteryState: 'steady',
      dueAt: '2026-09-01T00:00:00Z',
      reviewBox: 2,
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders mastery label, suggested next, and clear confirm flow', async () => {
    vi.mocked(clearPracticeData).mockResolvedValue({ attemptsDeleted: 2, progressRowsCleared: 1 })
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/practice', component: PracticeHubPage },
        { path: '/practice/:characterId', component: { template: '<div />' } },
        { path: '/practice/history', component: { template: '<div />' } },
        { path: '/', component: { template: '<div />' } },
      ],
    })
    await router.push('/practice')
    await router.isReady()

    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(PracticeHubPage)
    app.use(router)
    app.mount(root)
    await flush()
    await flush()

    expect(root.textContent).toContain(t('mastery.steady'))
    expect(root.textContent).toContain(t('hub.reviewDue'))
    expect(root.textContent).toContain(t('next.reason.due_review'))
    expect(root.textContent).toContain('あ')
    // Shell owns Practice/History/Board links; hub no longer duplicates them.
    expect(root.textContent).not.toContain(t('nav.board'))

    const clearBtn = Array.from(root.querySelectorAll('button')).find((b) =>
      b.textContent?.includes(t('hub.clearData')),
    )
    expect(clearBtn).toBeTruthy()
    clearBtn!.click()
    await flush()
    expect(confirmSpy).toHaveBeenCalled()
    expect(clearPracticeData).toHaveBeenCalled()
    expect(root.textContent).toContain(t('hub.clearSuccess'))

    app.unmount()
    confirmSpy.mockRestore()
  })

  it('shows caught-up banner with next due sentence', async () => {
    vi.mocked(listProgress).mockResolvedValue({ items: [] })
    vi.mocked(getProgressNext).mockResolvedValue({
      lessonId: 'lesson:hiragana5',
      characterId: null,
      reasonCode: 'all_caught_up',
      masteryState: 'steady',
      nextDueAt: '2026-09-14T12:00:00Z',
      nextDueCharacterId: 'hira:あ',
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/practice', component: PracticeHubPage },
        { path: '/', component: { template: '<div />' } },
      ],
    })
    await router.push('/practice')
    await router.isReady()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(PracticeHubPage)
    app.use(router)
    app.mount(root)
    await flush()
    await flush()
    expect(root.textContent).toContain(t('next.reason.all_caught_up'))
    expect(root.textContent).toMatch(/Next light review around/i)
    app.unmount()
  })

  it('history empty state and load-more pagination', async () => {
    vi.mocked(listAttempts)
      .mockResolvedValueOnce({
        items: [
          {
            id: '11111111-1111-4111-8111-111111111111',
            characterId: 'hira:あ',
            glyph: 'あ',
            status: 'assessed',
            startedAt: '2026-09-09T12:00:00Z',
            pass: false,
            score: 0.4,
            scoreKind: 'match',
            feedback: [{ rank: 1, code: 'shape', message: 'shape tip' }],
          },
        ],
        nextCursor: 'cur1',
        limit: 20,
      })
      .mockResolvedValueOnce({
        items: [
          {
            id: '22222222-2222-4222-8222-222222222222',
            characterId: 'hira:い',
            glyph: 'い',
            status: 'assessed',
            startedAt: '2026-09-08T12:00:00Z',
            pass: true,
            score: 0.9,
            scoreKind: 'match',
          },
        ],
        limit: 20,
      })

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/practice/history', component: PracticeHistoryPage },
        { path: '/practice', component: { template: '<div />' } },
        { path: '/practice/:characterId', component: { template: '<div />' } },
      ],
    })
    await router.push('/practice/history')
    await router.isReady()

    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(PracticeHistoryPage)
    app.use(router)
    app.mount(root)
    await flush()
    await flush()

    expect(root.textContent).toContain('あ')
    expect(root.textContent).toContain(t('history.fail'))

    const more = Array.from(root.querySelectorAll('button')).find((b) =>
      b.textContent?.includes(t('history.loadMore')),
    )
    expect(more).toBeTruthy()
    more!.click()
    await flush()
    expect(listAttempts).toHaveBeenCalledTimes(2)
    expect(root.textContent).toContain('い')

    app.unmount()
  })

  it('history empty state shows CTA', async () => {
    vi.mocked(listAttempts).mockResolvedValue({ items: [], limit: 20 })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/practice/history', component: PracticeHistoryPage },
        { path: '/practice', component: { template: '<div />' } },
      ],
    })
    await router.push('/practice/history')
    await router.isReady()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(PracticeHistoryPage)
    app.use(router)
    app.mount(root)
    await flush()
    expect(root.textContent).toContain(t('history.empty'))
    expect(root.textContent).toContain(t('history.practiceCta'))
    app.unmount()
  })

  it('exposes mastery and next reason i18n keys in EN/JA', () => {
    setLocale('en')
    expect(t('mastery.reason.no_assessed_attempts')).toMatch(/completed attempts/i)
    expect(t('next.reason.all_steady')).toMatch(/steady/i)
    expect(t('next.reason.due_review')).toMatch(/light review/i)
    expect(t('next.reason.all_caught_up')).toMatch(/due/i)
    setLocale('ja')
    expect(t('mastery.steady')).toBe('安定')
    expect(t('hub.reviewDue')).toMatch(/復習/)
    expect(t('history.empty')).toMatch(/まだ/)
  })
})
