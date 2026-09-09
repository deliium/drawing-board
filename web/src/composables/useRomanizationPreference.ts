import { getCurrentScope, onScopeDispose, ref, type Ref } from 'vue'

const STORAGE_KEY = 'romanizationVisible:v1'
const EVENT = 'drawing-board:romanization'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function readStored(): boolean {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === null) return true
    if (raw === 'false') return false
    if (raw === 'true') return true
  } catch {
    /* ignore */
  }
  return true
}

function writeStored(visible: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, visible ? 'true' : 'false')
  } catch {
    /* ignore */
  }
}

const visibleRef = ref(readStored())

function apply(next: boolean): void {
  visibleRef.value = next
  writeStored(next)
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(EVENT, { detail: next }))
  }
  if (isDev) console.debug('[useRomanizationPreference] set', next)
}

/**
 * Client preference: show Hepburn romanization on practice surfaces.
 * Default true; persisted in localStorage key romanizationVisible:v1.
 */
export function useRomanizationPreference(): {
  romanizationVisible: Ref<boolean>
  setRomanizationVisible: (next: boolean) => void
  toggleRomanizationVisible: () => void
} {
  const onExternal = (ev: Event) => {
    const detail = (ev as CustomEvent<boolean>).detail
    if (typeof detail === 'boolean') {
      visibleRef.value = detail
    } else {
      visibleRef.value = readStored()
    }
  }
  if (typeof window !== 'undefined') {
    window.addEventListener(EVENT, onExternal)
    if (getCurrentScope()) {
      onScopeDispose(() => window.removeEventListener(EVENT, onExternal))
    }
  }

  return {
    romanizationVisible: visibleRef,
    setRomanizationVisible: apply,
    toggleRomanizationVisible: () => apply(!visibleRef.value),
  }
}
