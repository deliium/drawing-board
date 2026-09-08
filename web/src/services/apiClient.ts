export type ApiError = {
  status: number
  message: string
  code?: string
}

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function apiDebug(...args: unknown[]) {
  if (isDev) console.debug('[apiFetch]', ...args)
}

/** Read a cookie value from document.cookie (browser only). Never logs the value. */
export function readCookie(name: string): string | undefined {
  if (typeof document === 'undefined') return undefined
  const parts = document.cookie.split(';')
  for (const part of parts) {
    const trimmed = part.trim()
    if (!trimmed) continue
    const eq = trimmed.indexOf('=')
    if (eq < 0) continue
    const key = trimmed.slice(0, eq)
    if (key === name) {
      return decodeURIComponent(trimmed.slice(eq + 1))
    }
  }
  return undefined
}

function isMutatingMethod(method: string): boolean {
  const m = method.toUpperCase()
  return m !== 'GET' && m !== 'HEAD' && m !== 'OPTIONS'
}

/** Ensure a CSRF cookie exists via GET /api/me (401 is fine). */
export async function bootstrapCsrf(): Promise<void> {
  apiDebug('bootstrapCsrf')
  try {
    await fetch('/api/me', { credentials: 'include' })
  } catch {
    // Network errors: caller will surface csrf_rejected on next mutation if needed.
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? 'GET').toUpperCase()
  const headers = new Headers(init?.headers)
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (isMutatingMethod(method)) {
    const csrf = readCookie('csrf')
    if (csrf) {
      headers.set('X-CSRF-Token', csrf)
      apiDebug('attached X-CSRF-Token for', method, path)
    } else {
      apiDebug('missing csrf cookie for', method, path)
    }
  }

  const res = await fetch(path, {
    ...init,
    credentials: 'include',
    headers,
  })
  if (!res.ok) {
    let code: string | undefined
    let message = `Request failed: ${res.status}`
    try {
      const body = (await res.json()) as { error?: string; message?: string }
      if (typeof body.error === 'string' && body.error.length > 0) {
        code = body.error
      }
      if (typeof body.message === 'string' && body.message.length > 0) {
        message = body.message
      } else if (code) {
        message = code
      }
    } catch {
      // non-JSON error body — keep generic message
    }

    if (res.status === 403 && code === 'csrf_rejected' && isMutatingMethod(method)) {
      apiDebug('csrf_rejected; re-bootstrapping once')
      await bootstrapCsrf()
      const retryHeaders = new Headers(headers)
      const refreshed = readCookie('csrf')
      if (refreshed) {
        retryHeaders.set('X-CSRF-Token', refreshed)
        const retry = await fetch(path, {
          ...init,
          credentials: 'include',
          headers: retryHeaders,
        })
        if (retry.ok) {
          return retry.json() as Promise<T>
        }
        try {
          const body = (await retry.json()) as { error?: string; message?: string }
          if (typeof body.error === 'string' && body.error.length > 0) code = body.error
          if (typeof body.message === 'string' && body.message.length > 0) message = body.message
        } catch {
          // keep prior message
        }
        const error: ApiError = { status: retry.status, message, code }
        throw error
      }
    }

    apiDebug(res.status, code)
    const error: ApiError = { status: res.status, message, code }
    throw error
  }
  return res.json() as Promise<T>
}
