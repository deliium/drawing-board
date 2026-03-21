import type { NavigationGuardNext, RouteLocationNormalized } from 'vue-router'
import { sessionContext } from '../services/sessionContext'

export function requireAuth(
  to: RouteLocationNormalized,
  _from: RouteLocationNormalized,
  next: NavigationGuardNext,
) {
  if (to.meta.requiresAuth && sessionContext.state !== 'authenticated') {
    next({ name: 'login' })
    return
  }
  next()
}
