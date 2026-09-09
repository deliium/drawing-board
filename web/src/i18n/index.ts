/** Lightweight EN/JA i18n (no vue-i18n). Log prefix: [i18n] */

import { en } from './locales/en'
import { ja } from './locales/ja'

export type Locale = 'en' | 'ja'

export const LOCALE_STORAGE_KEY = 'locale:v1'

type Catalog = Record<string, string>

const catalogs: Record<Locale, Catalog> = { en, ja }

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

let current: Locale = 'en'
const listeners = new Set<(locale: Locale) => void>()

function i18nLog(level: 'INFO' | 'WARN' | 'DEBUG', ...args: unknown[]) {
  if (level === 'DEBUG' && !isDev) return
  const fn = level === 'WARN' ? console.warn : console.info
  if (level === 'DEBUG') {
    console.debug('[i18n]', ...args)
    return
  }
  fn(`[i18n] ${level}`, ...args)
}

export function detectDefaultLocale(): Locale {
  if (typeof navigator !== 'undefined' && navigator.language?.toLowerCase().startsWith('ja')) {
    return 'ja'
  }
  return 'en'
}

export function loadStoredLocale(): Locale {
  if (typeof localStorage === 'undefined') return detectDefaultLocale()
  try {
    const raw = localStorage.getItem(LOCALE_STORAGE_KEY)
    if (raw === 'en' || raw === 'ja') {
      i18nLog('INFO', 'load', raw)
      return raw
    }
  } catch {
    // ignore
  }
  return detectDefaultLocale()
}

export function getLocale(): Locale {
  return current
}

export function setLocale(next: Locale): void {
  if (next !== 'en' && next !== 'ja') return
  current = next
  if (typeof document !== 'undefined') {
    document.documentElement.lang = next
  }
  if (typeof localStorage !== 'undefined') {
    try {
      localStorage.setItem(LOCALE_STORAGE_KEY, next)
    } catch {
      // ignore
    }
  }
  i18nLog('INFO', 'setLocale', next)
  for (const l of listeners) l(next)
}

export function onLocaleChange(fn: (locale: Locale) => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

/** Interpolate `{name}` placeholders. */
export function interpolate(template: string, params?: Record<string, string | number>): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (_, key: string) => {
    const v = params[key]
    return v === undefined || v === null ? `{${key}}` : String(v)
  })
}

export function t(key: string, params?: Record<string, string | number>): string {
  const catalog = catalogs[current] ?? catalogs.en
  let template = catalog[key]
  if (template === undefined) {
    template = catalogs.en[key]
    if (template === undefined) {
      i18nLog('WARN', 'missing key', key)
      return key
    }
    if (current !== 'en') {
      i18nLog('WARN', 'missing key (fallback en)', key)
    }
  }
  if (isDev) i18nLog('DEBUG', 'resolve', key)
  return interpolate(template, params)
}

/** Bootstrap locale from storage / navigator and set document.lang. */
export function initLocale(): Locale {
  const locale = loadStoredLocale()
  current = locale
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale
  }
  i18nLog('INFO', 'init', locale)
  return locale
}
