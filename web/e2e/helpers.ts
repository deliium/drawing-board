import { expect, type APIRequestContext, type Cookie, type Page } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '../..')

export function uniqueEmail(prefix = 'e2e'): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.com`
}

export async function registerViaUI(page: Page, email: string, password = 'password123'): Promise<void> {
  await page.goto('/#/register')
  await page.getByLabel(/email/i).fill(email)
  await page.locator('#auth-password').fill(password)
  await page.getByRole('button', { name: /create account|アカウント作成/i }).click()
  await expect(page).toHaveURL(/#\/($|practice)/, { timeout: 20_000 })
}

export function csrfFromCookies(cookies: Cookie[]): string {
  const token = cookies.find((c) => c.name === 'csrf')?.value
  if (!token) throw new Error('csrf cookie missing')
  return token
}

export async function apiFetchJSON(
  request: APIRequestContext,
  cookies: Cookie[],
  method: string,
  path: string,
  body?: unknown,
): Promise<{ status: number; json: any }> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    Cookie: cookies.map((c) => `${c.name}=${c.value}`).join('; '),
  }
  if (method !== 'GET' && method !== 'HEAD') {
    headers['X-CSRF-Token'] = csrfFromCookies(cookies)
  }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await request.fetch(path, {
    method,
    headers,
    data: body !== undefined ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let json: any = null
  try {
    json = text ? JSON.parse(text) : null
  } catch {
    json = { raw: text }
  }
  return { status: res.status(), json }
}

/** Load hiragana5 gold fixture strokes (unit space) for deterministic assess. */
export function loadGoldStrokes(glyph: string): Array<{ points: Array<{ x: number; y: number }> }> {
  const path = join(root, 'internal/recognize/testdata/hiragana5/gold', `${glyph}.json`)
  const raw = JSON.parse(readFileSync(path, 'utf8')) as {
    strokes: Array<{ points: Array<{ x: number; y: number }> }>
  }
  return raw.strokes
}

export async function assessCharacterViaAPI(
  request: APIRequestContext,
  cookies: Cookie[],
  characterId: string,
  glyph: string,
): Promise<{ attemptId: string; pass: boolean; score: number }> {
  const created = await apiFetchJSON(request, cookies, 'POST', '/api/attempts', {
    characterId,
    lessonId: 'lesson:hiragana5',
    clientAttemptID: `e2e-${Date.now()}`,
  })
  // API may use clientAttemptId camelCase
  if (created.status >= 400) {
    const retry = await apiFetchJSON(request, cookies, 'POST', '/api/attempts', {
      characterId,
      lessonId: 'lesson:hiragana5',
      clientAttemptId: `e2e-${Date.now()}`,
    })
    if (retry.status >= 400) {
      throw new Error(`create attempt failed: ${retry.status} ${JSON.stringify(retry.json)}`)
    }
    Object.assign(created, retry)
  }
  const attemptId = created.json.id as string
  const strokes = loadGoldStrokes(glyph).map((s) => ({
    color: '#111827',
    width: 3,
    startedAtUnixMs: Date.now(),
    points: s.points,
  }))
  const submitted = await apiFetchJSON(request, cookies, 'POST', `/api/attempts/${attemptId}/submit`, {
    width: 300,
    height: 300,
    strokes,
  })
  if (submitted.status >= 400) {
    throw new Error(`submit failed: ${submitted.status} ${JSON.stringify(submitted.json)}`)
  }
  const assessed = await apiFetchJSON(request, cookies, 'POST', `/api/attempts/${attemptId}/assess`, {})
  if (assessed.status >= 400) {
    throw new Error(`assess failed: ${assessed.status} ${JSON.stringify(assessed.json)}`)
  }
  return {
    attemptId,
    pass: Boolean(assessed.json.pass),
    score: Number(assessed.json.score),
  }
}
