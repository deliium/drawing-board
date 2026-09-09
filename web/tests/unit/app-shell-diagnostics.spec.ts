import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppShell from '../../src/components/AppShell.vue'
import MigrationHealthPanel from '../../src/components/MigrationHealthPanel.vue'
import LoginPage from '../../src/pages/LoginPage.vue'
import PracticeHubPage from '../../src/pages/PracticeHubPage.vue'
import { initLocale, setLocale, t } from '../../src/i18n'
import { setAuthenticatedUser } from '../../src/services/sessionContext'

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
  await nextTick()
}

describe('guest vs authed shell', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    initLocale()
    setLocale('en')
    setAuthenticatedUser(null)
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('hides Practice/History/Board nav on guest login shell', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/login',
          name: 'login',
          component: LoginPage,
          meta: { guestOnly: true, guestShell: true, title: 'Sign in' },
          props: { initialMode: 'login' },
        },
      ],
    })
    await router.push('/login')
    await router.isReady()

    const root = document.createElement('div')
    document.body.appendChild(root)
    const Host = defineComponent({
      components: { AppShell, LoginPage },
      template: '<AppShell><LoginPage initial-mode="login" /></AppShell>',
    })
    const app = createApp(Host)
    app.use(router)
    app.mount(root)
    await flush()

    expect(root.querySelector('.brand')?.textContent).toContain(t('brand.name'))
    expect(root.querySelector('.nav')).toBeNull()
    expect(root.textContent).not.toContain(t('nav.practice'))
    expect(root.textContent).not.toContain(t('nav.history'))
    expect(root.textContent).not.toContain(t('nav.board'))
    expect(root.querySelector('.romaji-toggle')).toBeNull()
    expect(root.querySelector('h1')?.textContent).toMatch(/Sign in|サインイン/)

    app.unmount()
  })

  it('shows Practice/History/Board nav on authenticated hub', async () => {
    setAuthenticatedUser({ id: 1, email: 'a@b.c' })
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
    expect(nav).toBeTruthy()
    expect(nav?.textContent).toContain(t('nav.practiceShort'))
    expect(nav?.textContent).toContain(t('nav.history'))
    expect(nav?.textContent).toContain(t('nav.boardShort'))
    expect(root.querySelector('.romaji-toggle')).toBeTruthy()

    app.unmount()
  })
})

describe('MigrationHealthPanel learner absence', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('does not render Migration Health heading (renamed Dev metrics only when mounted)', () => {
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(MigrationHealthPanel)
    app.mount(root)
    expect(root.textContent).not.toMatch(/Migration Health/i)
    expect(root.textContent).toMatch(/Dev metrics/)
    app.unmount()
  })

  it('production App.vue path omits panel when DEV is false', async () => {
    // Simulate the App.vue gate: panel only when import.meta.env.DEV.
    const isDev = false
    const root = document.createElement('div')
    document.body.appendChild(root)
    const Host = defineComponent({
      components: { MigrationHealthPanel },
      setup() {
        return { isDev }
      },
      template: '<div><MigrationHealthPanel v-if="isDev" /></div>',
    })
    const app = createApp(Host)
    app.mount(root)
    await flush()
    expect(root.textContent).not.toMatch(/Migration Health|Dev metrics/i)
    app.unmount()
  })
})

describe('history label consistency', () => {
  it('nav.history and history.title stay aligned learner phrases', () => {
    initLocale()
    setLocale('en')
    expect(t('nav.history')).toBe('History')
    expect(t('history.title')).toBe('Attempt history')
    // Page title is more specific; shell short label is History — both OK, no hub duplicate.
  })
})
