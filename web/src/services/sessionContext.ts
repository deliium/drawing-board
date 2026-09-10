import { reactive } from 'vue'

export type SessionState = 'unknown' | 'authenticated' | 'anonymous'

export const sessionContext = reactive({
  state: 'unknown' as SessionState,
  user: null as null | { id: string; email: string },
})

export function setAuthenticatedUser(user: { id: string; email: string } | null) {
  sessionContext.user = user
  sessionContext.state = user ? 'authenticated' : 'anonymous'
}
