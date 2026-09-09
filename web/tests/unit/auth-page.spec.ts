import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import LoginPage from '../../src/pages/LoginPage.vue'
import { initLocale, setLocale } from '../../src/i18n'
import { setAuthenticatedUser } from '../../src/services/sessionContext'
import {
  getFeatureFlags,
  resetFeatureFlagsForTest,
  setFeatureFlagsForTest,
} from '../../src/services/featuresApi'

const apiFetch = vi.fn()

vi.mock('../../src/services/apiClient', () => ({
  apiFetch: (...args: unknown[]) => apiFetch(...args),
}))

const ALL_FEATURES = {
  practice: true,
  progress: true,
  review: true,
  audio: true,
}

async function flush() {
  await nextTick()
  await Promise.resolve()
  await nextTick()
}

async function mountAuth(initialMode: 'login' | 'register' = 'login') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'board', component: { template: '<div>board</div>' } },
      {
        path: '/login',
        name: 'login',
        component: LoginPage,
        props: { initialMode: 'login' },
      },
      {
        path: '/register',
        name: 'register',
        component: LoginPage,
        props: { initialMode: 'register' },
      },
    ],
  })
  await router.push(initialMode === 'register' ? '/register' : '/login')
  await router.isReady()

  const root = document.createElement('div')
  document.body.appendChild(root)
  const app = createApp(LoginPage, { initialMode })
  app.use(router)
  app.mount(root)
  await flush()
  return { root, app, router }
}

async function setInput(el: HTMLInputElement, value: string) {
  el.value = value
  el.dispatchEvent(new Event('input', { bubbles: true }))
  el.dispatchEvent(new Event('change', { bubbles: true }))
  await flush()
}

describe('AuthPage behavior', () => {
  beforeEach(() => {
    apiFetch.mockReset()
    setAuthenticatedUser(null)
    resetFeatureFlagsForTest()
    document.body.innerHTML = ''
    initLocale()
    setLocale('en')
  })

  it('shows client validation for empty submit', async () => {
    const { root } = await mountAuth('login')
    const form = root.querySelector('form') as HTMLFormElement
    form.requestSubmit?.()
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flush()
    const alert = root.querySelector('[role="alert"]')
    expect(alert?.textContent).toMatch(/email and password|メールとパスワード/i)
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('disables submit while loading and maps success to board', async () => {
    let resolveLogin: (value: unknown) => void = () => undefined
    apiFetch.mockImplementation((path: string) => {
      if (path === '/api/features') {
        return Promise.resolve(ALL_FEATURES)
      }
      return new Promise((resolve) => {
        resolveLogin = resolve
      })
    })
    const { root, router } = await mountAuth('login')
    await setInput(root.querySelector('#auth-email') as HTMLInputElement, 'learner@example.com')
    await setInput(root.querySelector('#auth-password') as HTMLInputElement, 'password1')

    const form = root.querySelector('form') as HTMLFormElement
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flush()

    const submit = root.querySelector('.auth-submit') as HTMLButtonElement
    expect(submit.disabled).toBe(true)
    expect(form.getAttribute('aria-busy')).toBe('true')

    resolveLogin({ id: 7, email: 'learner@example.com' })
    await vi.waitFor(() => {
      expect(router.currentRoute.value.name).toBe('board')
    })
    expect(apiFetch).toHaveBeenCalledWith('/api/features')
  })

  it('refreshes feature flags after login before navigating to board', async () => {
    setFeatureFlagsForTest({
      practice: true,
      progress: true,
      review: true,
      audio: true,
    })
    apiFetch.mockImplementation(async (path: string) => {
      if (path === '/api/login') {
        return { id: 3, email: 'flag@example.com' }
      }
      if (path === '/api/features') {
        return { practice: false, progress: false, review: false, audio: false }
      }
      throw new Error(`unexpected path ${path}`)
    })

    const { root, router } = await mountAuth('login')
    await setInput(root.querySelector('#auth-email') as HTMLInputElement, 'flag@example.com')
    await setInput(root.querySelector('#auth-password') as HTMLInputElement, 'password1')

    const form = root.querySelector('form') as HTMLFormElement
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))

    await vi.waitFor(() => {
      expect(router.currentRoute.value.name).toBe('board')
    })
    expect(getFeatureFlags()).toEqual({
      practice: false,
      progress: false,
      review: false,
      audio: false,
    })
    const paths = apiFetch.mock.calls.map((c) => c[0])
    expect(paths).toEqual(['/api/login', '/api/features'])
  })

  it('switches to create account mode via tab control', async () => {
    const { root, router } = await mountAuth('login')
    const tabs = Array.from(root.querySelectorAll('[role="tab"]')) as HTMLButtonElement[]
    const createTab = tabs.find((el) => el.textContent?.includes('Create account'))
    expect(createTab).toBeTruthy()
    createTab!.click()
    await vi.waitFor(() => {
      expect(router.currentRoute.value.name).toBe('register')
    })
    expect(root.querySelector('h1')?.textContent).toContain('Create account')
  })

  it('rejects passwords longer than 72 UTF-8 bytes client-side', async () => {
    const { root } = await mountAuth('register')
    const email = root.querySelector('#auth-email') as HTMLInputElement
    const password = root.querySelector('#auth-password') as HTMLInputElement
    await setInput(email, 'long@example.com')
    await setInput(password, 'a'.repeat(73))
    const form = root.querySelector('form') as HTMLFormElement
    form.requestSubmit?.()
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flush()
    const alert = root.querySelector('[role="alert"]')
    expect(alert?.textContent).toContain('72 bytes')
    expect(apiFetch).not.toHaveBeenCalled()
  })
})
