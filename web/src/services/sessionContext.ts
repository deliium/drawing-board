import { reactive } from 'vue'

export type SessionState = 'unknown' | 'authenticated' | 'anonymous'

export const sessionContext = reactive({
  state: 'unknown' as SessionState,
  user: null as null | { id: number; email: string },
})

export function setAuthenticatedUser(user: { id: number; email: string } | null) {
  sessionContext.user = user
  sessionContext.state = user ? 'authenticated' : 'anonymous'
}
