import type { NavigationGuardNext, RouteLocationNormalized } from 'vue-router'
import { sessionContext } from '../services/sessionContext'

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function routerDebug(...args: unknown[]) {
  if (isDev) console.debug('[router]', ...args)
}

export function requireAuth(
  to: RouteLocationNormalized,
  _from: RouteLocationNormalized,
  next: NavigationGuardNext,
) {
  if (to.meta.requiresAuth && sessionContext.state !== 'authenticated') {
    routerDebug('redirect reason=auth from=', to.fullPath, 'to=login')
    next({ name: 'login' })
    return
  }
  if (to.meta.guestOnly && sessionContext.state === 'authenticated') {
    routerDebug('redirect reason=guest from=', to.fullPath, 'to=board')
    next({ name: 'board' })
    return
  }
  next()
}
