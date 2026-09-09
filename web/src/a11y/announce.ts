/** DEV-only live-region announce helper. Log: [a11y.announce] */

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

let politeEl: HTMLElement | null = null
let assertiveEl: HTMLElement | null = null

function ensureRegion(kind: 'polite' | 'assertive'): HTMLElement | null {
  if (typeof document === 'undefined') return null
  const id = kind === 'polite' ? 'a11y-announce-polite' : 'a11y-announce-assertive'
  let el = document.getElementById(id)
  if (!el) {
    el = document.createElement('div')
    el.id = id
    el.setAttribute('role', kind === 'assertive' ? 'alert' : 'status')
    el.setAttribute('aria-live', kind)
    el.setAttribute('aria-atomic', 'true')
    el.className = 'visually-hidden'
    el.style.cssText =
      'position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0'
    document.body.appendChild(el)
  }
  if (kind === 'polite') politeEl = el
  else assertiveEl = el
  return el
}

export function announce(text: string, politeness: 'polite' | 'assertive' = 'polite'): void {
  const trimmed = text.trim()
  if (!trimmed) return
  if (isDev) console.debug('[a11y.announce]', politeness, trimmed)
  const el = ensureRegion(politeness)
  if (!el) return
  el.textContent = ''
  // Force a DOM change so SR re-announces.
  requestAnimationFrame(() => {
    el.textContent = trimmed
  })
}

export function clearAnnouncements(): void {
  if (politeEl) politeEl.textContent = ''
  if (assertiveEl) assertiveEl.textContent = ''
}
