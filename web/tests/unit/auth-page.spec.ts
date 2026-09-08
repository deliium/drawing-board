import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import LoginPage from '../../src/pages/LoginPage.vue'
import { setAuthenticatedUser } from '../../src/services/sessionContext'

const apiFetch = vi.fn()

vi.mock('../../src/services/apiClient', () => ({
  apiFetch: (...args: unknown[]) => apiFetch(...args),
}))

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
    document.body.innerHTML = ''
  })

  it('shows client validation for empty submit', async () => {
    const { root } = await mountAuth('login')
    const form = root.querySelector('form') as HTMLFormElement
    form.requestSubmit?.()
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flush()
    const alert = root.querySelector('[role="alert"]')
    expect(alert?.textContent).toContain('Enter email and password')
    expect(apiFetch).not.toHaveBeenCalled()
  })

  it('disables submit while loading and maps success to board', async () => {
    let resolveFetch: (value: unknown) => void = () => undefined
    apiFetch.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveFetch = resolve
        }),
    )
    const { root, router } = await mountAuth('login')
    await setInput(root.querySelector('#auth-email') as HTMLInputElement, 'learner@example.com')
    await setInput(root.querySelector('#auth-password') as HTMLInputElement, 'password1')

    const form = root.querySelector('form') as HTMLFormElement
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flush()

    const submit = root.querySelector('.auth-submit') as HTMLButtonElement
    expect(submit.disabled).toBe(true)
    expect(form.getAttribute('aria-busy')).toBe('true')

    resolveFetch({ id: 7, email: 'learner@example.com' })
    await vi.waitFor(() => {
      expect(router.currentRoute.value.name).toBe('board')
    })
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
