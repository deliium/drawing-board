import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppShell from '../../src/components/AppShell.vue'
import PracticeHubPage from '../../src/pages/PracticeHubPage.vue'
import { initLocale, t } from '../../src/i18n'
import { setAuthenticatedUser } from '../../src/services/sessionContext'
import {
  resetFeatureFlagsForTest,
  setFeatureFlagsForTest,
} from '../../src/services/featuresApi'

vi.mock('../../src/services/curriculumApi', () => ({
  HIRAGANA5_LESSON_ID: 'lesson:hiragana5',
  getLesson: vi.fn(async () => ({
    id: 'lesson:hiragana5',
    code: 'hiragana5',
    title: 'Hiragana vowels',
    setId: 'hiragana5',
    contentVersion: 'v1',
    characters: [],
  })),
}))

vi.mock('../../src/services/progressApi', () => ({
  listProgress: vi.fn(async () => ({ items: [] })),
  getProgressNext: vi.fn(async () => ({
    lessonId: 'lesson:hiragana5',
    characterId: null,
    reasonCode: 'all_caught_up',
    masteryState: 'not_started',
  })),
  clearPracticeData: vi.fn(),
}))

vi.mock('../../src/services/apiClient', () => ({
  apiFetch: vi.fn(),
}))

async function flush() {
  await nextTick()
  await Promise.resolve()
}

describe('feature flag nav gating', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    initLocale()
    resetFeatureFlagsForTest()
    setAuthenticatedUser({ id: '11111111-1111-4111-8111-111111111111', email: 'a@b.c' })
  })

  afterEach(() => {
    resetFeatureFlagsForTest()
  })

  it('hides Practice and History when FEATURE_PRACTICE is off', async () => {
    setFeatureFlagsForTest({ practice: false, progress: false })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/',
          name: 'board',
          component: { template: '<div>board</div>' },
          meta: { requiresAuth: true, title: 'Free board' },
        },
      ],
    })
    await router.push('/')
    await router.isReady()

    const root = document.createElement('div')
    document.body.appendChild(root)
    const Host = defineComponent({
      components: { AppShell },
      template: '<AppShell><div>board</div></AppShell>',
    })
    const app = createApp(Host)
    app.use(router)
    app.mount(root)
    await flush()

    const nav = root.querySelector('.nav')
    expect(nav?.textContent).not.toContain(t('nav.practiceShort'))
    expect(nav?.textContent).not.toContain(t('nav.history'))
    expect(nav?.textContent).toContain(t('nav.boardShort'))
    app.unmount()
  })

  it('hides History only when progress is off but practice is on', async () => {
    setFeatureFlagsForTest({ practice: true, progress: false })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/practice',
          name: 'practice-hub',
          component: PracticeHubPage,
          meta: { requiresAuth: true, title: 'Practice' },
        },
      ],
    })
    await router.push('/practice')
    await router.isReady()

    const root = document.createElement('div')
    document.body.appendChild(root)
    const Host = defineComponent({
      components: { AppShell, PracticeHubPage },
      template: '<AppShell><PracticeHubPage /></AppShell>',
    })
    const app = createApp(Host)
    app.use(router)
    app.mount(root)
    await flush()
    await flush()

    const nav = root.querySelector('.nav')
    expect(nav?.textContent).toContain(t('nav.practiceShort'))
    expect(nav?.textContent).not.toContain(t('nav.history'))
    app.unmount()
  })
})
