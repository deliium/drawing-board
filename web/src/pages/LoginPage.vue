<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLocale } from '../composables/useLocale'
import { apiFetch, type ApiError } from '../services/apiClient'
import { loadFeatureFlags } from '../services/featuresApi'
import { setAuthenticatedUser } from '../services/sessionContext'

const props = withDefaults(
  defineProps<{ initialMode?: 'login' | 'register' }>(),
  { initialMode: 'login' },
)

const route = useRoute()
const router = useRouter()
const { t } = useLocale()

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function authDebug(...args: unknown[]) {
  if (isDev) console.debug('[AuthPage]', ...args)
}

type AuthMode = 'login' | 'register'

const mode = ref<AuthMode>(props.initialMode)
const email = ref('')
const password = ref('')
const fieldErrors = ref<{ email?: string; password?: string }>({})
const formError = ref('')
const submitting = ref(false)

const heading = computed(() =>
  mode.value === 'login' ? t('auth.heading.login') : t('auth.heading.register'),
)
const submitLabel = computed(() =>
  mode.value === 'login' ? t('auth.submit.login') : t('auth.submit.register'),
)
const passwordAutocomplete = computed(() =>
  mode.value === 'login' ? 'current-password' : 'new-password',
)

watch(
  () => route.name,
  (name) => {
    if (name === 'register') mode.value = 'register'
    else if (name === 'login') mode.value = 'login'
  },
  { immediate: true },
)

watch(
  () => props.initialMode,
  (m) => {
    if (route.name !== 'login' && route.name !== 'register') {
      mode.value = m
    }
  },
)

function errorCopy(code: string): string {
  const key = `auth.error.${code}`
  const translated = t(key)
  return translated === key ? t('auth.error.server') : translated
}

const maxPasswordBytes = 72

function passwordByteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

function validEmail(value: string): boolean {
  const at = value.lastIndexOf('@')
  if (at <= 0 || at === value.length - 1) return false
  const local = value.slice(0, at)
  const domain = value.slice(at + 1)
  if (!local || !domain) return false
  if (/\s/.test(local) || /\s/.test(domain)) return false
  if (domain.includes('.')) {
    return domain.split('.').every((p) => p.length > 0)
  }
  return /^[a-z0-9-]+$/i.test(domain)
}

function validateClient(): boolean {
  const next: { email?: string; password?: string } = {}
  const trimmed = email.value.trim()
  if (!trimmed || !password.value) {
    formError.value = errorCopy('missing_fields')
    if (!trimmed) next.email = errorCopy('missing_fields')
    if (!password.value) next.password = errorCopy('missing_fields')
    fieldErrors.value = next
    return false
  }
  if (!validEmail(trimmed.toLowerCase())) {
    next.email = errorCopy('invalid_email')
    formError.value = errorCopy('invalid_email')
    fieldErrors.value = next
    return false
  }
  if (mode.value === 'register' && password.value.length < 8) {
    next.password = errorCopy('password_too_short')
    formError.value = errorCopy('password_too_short')
    fieldErrors.value = next
    return false
  }
  if (passwordByteLength(password.value) > maxPasswordBytes) {
    next.password = errorCopy('password_too_long')
    formError.value = errorCopy('password_too_long')
    fieldErrors.value = next
    return false
  }
  fieldErrors.value = {}
  formError.value = ''
  return true
}

function setMode(next: AuthMode) {
  mode.value = next
  formError.value = ''
  fieldErrors.value = {}
  const target = next === 'register' ? 'register' : 'login'
  if (route.name !== target) {
    void router.replace({ name: target })
  }
}

function onTabKeydown(e: KeyboardEvent, current: AuthMode) {
  if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight' && e.key !== 'Home' && e.key !== 'End') {
    return
  }
  e.preventDefault()
  if (e.key === 'Home') {
    setMode('login')
    return
  }
  if (e.key === 'End') {
    setMode('register')
    return
  }
  setMode(current === 'login' ? 'register' : 'login')
}

function mapApiError(err: unknown): string {
  const apiErr = err as ApiError
  if (apiErr?.code) {
    const mapped = errorCopy(apiErr.code)
    if (mapped !== t('auth.error.server') || apiErr.code === 'server') {
      const key = `auth.error.${apiErr.code}`
      if (t(key) !== key) return mapped
    }
  }
  if (apiErr?.status && apiErr.status >= 500) {
    return t('auth.error.server')
  }
  if (typeof apiErr?.message === 'string' && apiErr.message && !apiErr.message.startsWith('Request failed:')) {
    return apiErr.message
  }
  return t('auth.error.server')
}

async function submit() {
  if (submitting.value) return
  if (!validateClient()) {
    authDebug(mode.value, 'error', 'client_validation')
    return
  }
  submitting.value = true
  formError.value = ''
  authDebug(mode.value, 'submit')
  const path = mode.value === 'login' ? '/api/login' : '/api/register'
  try {
    const user = await apiFetch<{ id: string; email: string }>(path, {
      method: 'POST',
      body: JSON.stringify({ email: email.value.trim(), password: password.value }),
    })
    password.value = ''
    setAuthenticatedUser(user)
    // Guest bootstrap skips /api/features (auth-only); refresh kill switches before nav.
    const flags = await loadFeatureFlags()
    if (isDev) {
      console.debug('[FIX] feature flags refreshed after auth', { mode: mode.value, flags })
    }
    authDebug(mode.value, 'success', flags)
    await router.replace({ name: 'board' })
  } catch (err) {
    const code = (err as ApiError)?.code
    formError.value = mapApiError(err)
    authDebug(mode.value, 'error', code)
    if (mode.value === 'login') {
      password.value = ''
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <form
      id="auth-panel"
      class="auth-form"
      role="tabpanel"
      :aria-labelledby="mode === 'login' ? 'auth-tab-login' : 'auth-tab-register'"
      @submit.prevent="submit"
      :aria-busy="submitting"
    >
      <h1 class="auth-heading">{{ heading }}</h1>
      <p class="auth-lede">{{ t('auth.lede') }}</p>

      <div class="auth-modes" role="tablist" :aria-label="t('auth.tablist')">
        <button
          id="auth-tab-login"
          type="button"
          role="tab"
          class="auth-mode"
          :aria-selected="mode === 'login'"
          aria-controls="auth-panel"
          :tabindex="mode === 'login' ? 0 : -1"
          :class="{ active: mode === 'login' }"
          @click="setMode('login')"
          @keydown="onTabKeydown($event, 'login')"
        >
          {{ t('auth.tab.login') }}
        </button>
        <button
          id="auth-tab-register"
          type="button"
          role="tab"
          class="auth-mode"
          :aria-selected="mode === 'register'"
          aria-controls="auth-panel"
          :tabindex="mode === 'register' ? 0 : -1"
          :class="{ active: mode === 'register' }"
          @click="setMode('register')"
          @keydown="onTabKeydown($event, 'register')"
        >
          {{ t('auth.tab.register') }}
        </button>
      </div>

      <div class="auth-field">
        <label for="auth-email">{{ t('auth.email') }}</label>
        <input
          id="auth-email"
          v-model="email"
          type="email"
          name="email"
          autocomplete="email"
          inputmode="email"
          required
          :disabled="submitting"
          :aria-invalid="Boolean(fieldErrors.email)"
          :aria-describedby="fieldErrors.email ? 'auth-email-error' : undefined"
        />
        <p v-if="fieldErrors.email" id="auth-email-error" class="auth-field-error">
          {{ fieldErrors.email }}
        </p>
      </div>

      <div class="auth-field">
        <label for="auth-password">{{ t('auth.password') }}</label>
        <input
          id="auth-password"
          v-model="password"
          type="password"
          name="password"
          :autocomplete="passwordAutocomplete"
          required
          :minlength="mode === 'register' ? 8 : undefined"
          :disabled="submitting"
          :aria-invalid="Boolean(fieldErrors.password)"
          :aria-describedby="fieldErrors.password ? 'auth-password-error' : undefined"
        />
        <p v-if="fieldErrors.password" id="auth-password-error" class="auth-field-error">
          {{ fieldErrors.password }}
        </p>
      </div>

      <p v-if="formError" class="auth-alert" role="alert">{{ formError }}</p>

      <button class="auth-submit" type="submit" :disabled="submitting">
        {{ submitting ? t('auth.submitting') : submitLabel }}
      </button>

      <p
        v-if="mode === 'register' && (formError.includes('already have one') || formError.includes('既にある'))"
        class="auth-switch-hint"
      >
        <button type="button" class="auth-link" @click="setMode('login')">
          {{ t('auth.switchLogin') }}
        </button>
      </p>
    </form>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: calc(100dvh - 5rem);
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: var(--space-5) var(--space-4) var(--space-5);
}

.auth-form {
  width: min(100%, 420px);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-5) var(--space-4);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--paper-raised) 92%, transparent);
}

.auth-heading {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--ink);
}

.auth-lede {
  margin: 0;
  color: var(--ink-muted);
  font-size: 0.95rem;
  line-height: 1.4;
}

.auth-modes {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-2);
}

.auth-mode {
  min-height: var(--touch-min);
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--ink);
  font: inherit;
  cursor: pointer;
}

.auth-mode.active {
  background: var(--ink);
  color: var(--paper-raised);
  border-color: var(--ink);
}

.auth-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.auth-field label {
  font-size: 0.9rem;
  color: var(--ink);
}

.auth-field input {
  min-height: var(--touch-min);
  padding: 10px 12px;
  border: 1px solid var(--rule);
  border-radius: var(--radius-sm);
  background: var(--paper-raised);
  font: inherit;
  width: 100%;
  color: var(--ink);
}

.auth-field-error {
  margin: 0;
  color: var(--danger);
  font-size: 0.85rem;
}

.auth-alert {
  margin: 0;
  color: var(--danger);
  font-size: 0.95rem;
}

.auth-submit {
  min-height: 48px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--accent);
  color: #f8fafc;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.auth-submit:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.auth-switch-hint {
  margin: 0;
  text-align: center;
}

.auth-link {
  background: none;
  border: none;
  color: var(--accent);
  text-decoration: underline;
  font: inherit;
  cursor: pointer;
  min-height: var(--touch-min);
}
</style>
