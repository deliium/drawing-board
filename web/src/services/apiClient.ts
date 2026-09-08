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

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...init,
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
    apiDebug(res.status, code)
    const error: ApiError = { status: res.status, message, code }
    throw error
  }
  return res.json() as Promise<T>
}
