export type ApiError = {
  status: number
  message: string
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const error: ApiError = { status: res.status, message: `Request failed: ${res.status}` }
    throw error
  }
  return res.json() as Promise<T>
}
