/**
 * Accessibility parity (Prompt 14)
 *
 * Automated: axe-core serious/critical must be zero on Login, Practice hub,
 * Practice result fixture, and BoardPage tool chrome.
 *
 * Manual smoke (document + run on ≥1 real phone):
 * - Keyboard: Tab through shell → practice → submit; visible :focus-visible; Escape cancels stroke
 * - Zoom 200%: auth + practice primary actions usable, no clipped buttons
 * - Screen reader (VoiceOver/TalkBack): stage banner + result summary announced; tool radio state spoken
 * - Real device (iOS Safari or Android Chrome): draw/trace; login VK does not permanently hide errors; canvas stays square
 * - Locale: toggle EN↔JA; chrome + corrections + hub status switch; glyphs keep lang=ja
 */
import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { configureAxe } from 'vitest-axe'
import 'vitest-axe/extend-expect'
import LoginPage from '../../src/pages/LoginPage.vue'
import PracticeHubPage from '../../src/pages/PracticeHubPage.vue'
import BoardPage from '../../src/pages/BoardPage.vue'
import ComparisonOverlay from '../../src/components/practice/ComparisonOverlay.vue'
import AppShell from '../../src/components/AppShell.vue'
import { initLocale, setLocale } from '../../src/i18n'
import { setAuthenticatedUser } from '../../src/services/sessionContext'
import { stubCanvasContext } from '../helpers/stubCanvasContext'

vi.mock('../../src/services/wsClient', () => ({
  createWsClient: () => ({
    connect: vi.fn(),
    close: vi.fn(),
    send: vi.fn(() => true),
    clearPending: vi.fn(),
    dropPendingCreates: vi.fn(),
    dropOp: vi.fn(),
    getBoardRev: vi.fn(() => 0),
    setBoardRev: vi.fn(),
    getQueueLength: vi.fn(() => 0),
  }),
}))

const axe = configureAxe({
  rules: {
    // Canvas elements often lack text alternatives beyond aria-label in jsdom.
    'aria-allowed-attr': { enabled: true },
  },
})

async function flush() {
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

vi.mock('../../src/services/apiClient', () => ({
  apiFetch: vi.fn(async (path: string) => {
    if (path.includes('/api/lessons/')) {
      return {
        id: 'lesson:hiragana5',
        code: 'hiragana5',
        title: 'Hiragana vowels',
        titleJa: 'ひらがな母音',
        setId: 'hiragana5',
        contentVersion: 'hiragana5-content-v2',
        characters: [
          {
            id: 'hira:あ',
            glyph: 'あ',
            romanization: 'a',
            strokeCount: 3,
            pronunciation: { ipa: '/a/', jaHint: '「あ」の音' },
            descriptionEn: 'Open vowel',
            descriptionJa: '母音あ',
            example: { word: 'あさ', romanization: 'asa', meaningEn: 'morning', meaningJa: '朝' },
            sortKey: 1,
            position: 1,
          },
        ],
      }
    }
    if (path.includes('/api/progress/next')) {
      return {
        lessonId: 'lesson:hiragana5',
        characterId: 'hira:あ',
        glyph: 'あ',
        reasonCode: 'first_not_started',
        masteryState: 'not_started',
      }
    }
    if (path.includes('/api/progress')) {
      return { items: [] }
    }
    if (path.startsWith('/api/attempts')) {
      return { items: [], limit: 20 }
    }
    if (path.includes('/api/strokes')) {
      return { boardRev: 0, strokes: [] }
    }
    throw new Error(`unexpected ${path}`)
  }),
}))

describe('accessibility parity (axe)', () => {
  let canvasSpy: ReturnType<typeof stubCanvasContext>

  beforeEach(() => {
    document.body.innerHTML = ''
    initLocale()
    setLocale('en')
    canvasSpy = stubCanvasContext()
    setAuthenticatedUser({ id: '11111111-1111-4111-8111-111111111111', email: 'learner@example.com' })
  })

  afterEach(() => {
    canvasSpy.mockRestore()
    setAuthenticatedUser(null)
  })

  it('Login page has no serious/critical axe violations', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'board', component: { template: '<div />' } },
        { path: '/login', name: 'login', component: LoginPage },
        { path: '/register', name: 'register', component: LoginPage },
      ],
    })
    await router.push('/login')
    await router.isReady()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(LoginPage, { initialMode: 'login' })
    app.use(router)
    app.mount(root)
    await flush()
    const results = await axe(root)
    const bad = results.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
    expect(bad).toHaveLength(0)
    app.unmount()
  })

  it('Register mode has no serious/critical axe violations', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'board', component: { template: '<div />' } },
        { path: '/login', name: 'login', component: LoginPage },
        { path: '/register', name: 'register', component: LoginPage },
      ],
    })
    await router.push('/register')
    await router.isReady()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(LoginPage, { initialMode: 'register' })
    app.use(router)
    app.mount(root)
    await flush()
    expect(root.querySelector('[aria-controls="auth-panel"]')).toBeTruthy()
    const results = await axe(root)
    const bad = results.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
    expect(bad).toHaveLength(0)
    app.unmount()
  })

  it('AppShell + Practice history has no serious/critical axe violations', async () => {
    const { default: PracticeHistoryPage } = await import('../../src/pages/PracticeHistoryPage.vue')
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/practice', component: PracticeHubPage },
        { path: '/practice/history', component: PracticeHistoryPage },
      ],
    })
    await router.push('/practice/history')
    await router.isReady()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp({
      components: { AppShell, PracticeHistoryPage },
      template: '<AppShell><PracticeHistoryPage /></AppShell>',
    })
    app.use(router)
    app.mount(root)
    await flush()
    await flush()
    const results = await axe(root)
    const bad = results.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
    expect(bad).toHaveLength(0)
    app.unmount()
  })

  it('Comparison result summary has no serious/critical axe violations', async () => {
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(ComparisonOverlay, {
      glyph: 'あ',
      learnerStrokes: [],
      pass: false,
      score: 0.42,
      feedback: [
        { rank: 1, code: 'shape', message: 'The overall shape differs.' },
        { rank: 2, code: 'proportions', message: 'Adjust proportions.' },
      ],
      focusOnMount: false,
    })
    app.mount(root)
    await flush()
    expect(root.textContent).toMatch(/Match|マッチ/)
    expect(root.querySelector('[aria-live="polite"]')).toBeTruthy()
    const results = await axe(root)
    const bad = results.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
    expect(bad).toHaveLength(0)
    app.unmount()
  })

  it('BoardPage tools + status have no serious/critical axe violations', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'board', component: BoardPage },
        { path: '/login', name: 'login', component: { template: '<div />' } },
        { path: '/practice', name: 'practice', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const app = createApp(BoardPage)
    app.use(router)
    app.mount(root)
    await flush()
    await flush()
    expect(root.querySelector('[role="radiogroup"]')).toBeTruthy()
    expect(root.querySelector('[data-sync-status]')).toBeTruthy()
    const results = await axe(root)
    const bad = results.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
    expect(bad).toHaveLength(0)
    app.unmount()
  })
})
