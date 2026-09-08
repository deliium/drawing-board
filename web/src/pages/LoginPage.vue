<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch, type ApiError } from '../services/apiClient'
import { setAuthenticatedUser } from '../services/sessionContext'

const props = withDefaults(
  defineProps<{ initialMode?: 'login' | 'register' }>(),
  { initialMode: 'login' },
)

const route = useRoute()
const router = useRouter()

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

const heading = computed(() => (mode.value === 'login' ? 'Sign in' : 'Create account'))
const submitLabel = computed(() => (mode.value === 'login' ? 'Sign in' : 'Create account'))
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

const ERROR_COPY: Record<string, string> = {
  bad_json: 'Something went wrong. Try again.',
  missing_fields: 'Enter email and password.',
  invalid_email: 'Enter a valid email address.',
  password_too_short: 'Password must be at least 8 characters.',
  registration_failed: 'Unable to create account. If you already have one, sign in.',
  invalid_credentials: 'Email or password is incorrect.',
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
    formError.value = ERROR_COPY.missing_fields
    if (!trimmed) next.email = ERROR_COPY.missing_fields
    if (!password.value) next.password = ERROR_COPY.missing_fields
    fieldErrors.value = next
    return false
  }
  if (!validEmail(trimmed.toLowerCase())) {
    next.email = ERROR_COPY.invalid_email
    formError.value = ERROR_COPY.invalid_email
    fieldErrors.value = next
    return false
  }
  if (password.value.length < 8) {
    next.password = ERROR_COPY.password_too_short
    formError.value = ERROR_COPY.password_too_short
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

function mapApiError(err: unknown): string {
  const apiErr = err as ApiError
  if (apiErr?.code && ERROR_COPY[apiErr.code]) {
    return ERROR_COPY[apiErr.code]
  }
  if (apiErr?.status && apiErr.status >= 500) {
    return 'Unable to reach the server. Try again.'
  }
  if (typeof apiErr?.message === 'string' && apiErr.message && !apiErr.message.startsWith('Request failed:')) {
    return apiErr.message
  }
  return 'Unable to reach the server. Try again.'
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
    const user = await apiFetch<{ id: number; email: string }>(path, {
      method: 'POST',
      body: JSON.stringify({ email: email.value.trim(), password: password.value }),
    })
    password.value = ''
    setAuthenticatedUser(user)
    authDebug(mode.value, 'success')
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
    <form class="auth-form" @submit.prevent="submit" :aria-busy="submitting">
      <h1 class="auth-heading">{{ heading }}</h1>
      <p class="auth-lede">Private Japanese handwriting practice for your account.</p>

      <div class="auth-modes" role="tablist" aria-label="Authentication mode">
        <button
          type="button"
          role="tab"
          class="auth-mode"
          :aria-selected="mode === 'login'"
          :class="{ active: mode === 'login' }"
          @click="setMode('login')"
        >
          Sign in
        </button>
        <button
          type="button"
          role="tab"
          class="auth-mode"
          :aria-selected="mode === 'register'"
          :class="{ active: mode === 'register' }"
          @click="setMode('register')"
        >
          Create account
        </button>
      </div>

      <div class="auth-field">
        <label for="auth-email">Email</label>
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
        <label for="auth-password">Password</label>
        <input
          id="auth-password"
          v-model="password"
          type="password"
          name="password"
          :autocomplete="passwordAutocomplete"
          required
          minlength="8"
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
        {{ submitting ? 'Please wait…' : submitLabel }}
      </button>

      <p v-if="mode === 'register' && formError.includes('already have one')" class="auth-switch-hint">
        <button type="button" class="auth-link" @click="setMode('login')">Switch to Sign in</button>
      </p>
    </form>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: calc(100vh - 48px);
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 24px 16px 48px;
  background: linear-gradient(160deg, #f0f4f8 0%, #e2e8f0 48%, #dbe4ee 100%);
}

.auth-form {
  width: min(100%, 420px);
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 28px 22px;
  border: 1px solid #c5d0db;
  background: rgba(255, 255, 255, 0.88);
}

.auth-heading {
  margin: 0;
  font-family: "IBM Plex Serif", "Source Serif 4", "Noto Serif JP", Georgia, serif;
  font-size: 1.75rem;
  font-weight: 600;
  color: #1a2332;
}

.auth-lede {
  margin: 0;
  color: #4a5568;
  font-size: 0.95rem;
  line-height: 1.4;
}

.auth-modes {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.auth-mode {
  min-height: 44px;
  border: 1px solid #9aa8b8;
  background: transparent;
  color: #1a2332;
  font: inherit;
  cursor: pointer;
}

.auth-mode.active {
  background: #1a2332;
  color: #f5f8fb;
  border-color: #1a2332;
}

.auth-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.auth-field label {
  font-size: 0.9rem;
  color: #1a2332;
}

.auth-field input {
  min-height: 44px;
  padding: 10px 12px;
  border: 1px solid #9aa8b8;
  background: #fff;
  font: inherit;
  width: 100%;
  box-sizing: border-box;
}

.auth-field-error {
  margin: 0;
  color: #9b1c1c;
  font-size: 0.85rem;
}

.auth-alert {
  margin: 0;
  color: #9b1c1c;
  font-size: 0.95rem;
}

.auth-submit {
  min-height: 48px;
  border: none;
  background: #1f6f8b;
  color: #f5f8fb;
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
  color: #1f6f8b;
  text-decoration: underline;
  font: inherit;
  cursor: pointer;
  min-height: 44px;
}
</style>
