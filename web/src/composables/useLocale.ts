import { computed, onScopeDispose, ref, type Ref } from 'vue'
import {
  getLocale,
  initLocale,
  onLocaleChange,
  setLocale,
  t as translate,
  type Locale,
} from '../i18n'

let bootstrapped = false

function ensureBoot(): void {
  if (!bootstrapped) {
    initLocale()
    bootstrapped = true
  }
}

/**
 * Reactive locale preference. Call once at app bootstrap via initLocale in main.ts;
 * this composable tracks changes for templates.
 */
export function useLocale(): {
  locale: Ref<Locale>
  setLocale: (next: Locale) => void
  t: (key: string, params?: Record<string, string | number>) => string
  isJa: Ref<boolean>
} {
  ensureBoot()
  const locale = ref<Locale>(getLocale())

  const stop = onLocaleChange((next) => {
    locale.value = next
  })
  onScopeDispose(stop)

  function t(key: string, params?: Record<string, string | number>): string {
    // Depend on locale so computed templates re-evaluate.
    void locale.value
    return translate(key, params)
  }

  const isJa = computed(() => locale.value === 'ja')

  return {
    locale,
    setLocale: (next: Locale) => {
      setLocale(next)
      locale.value = next
    },
    t,
    isJa,
  }
}
