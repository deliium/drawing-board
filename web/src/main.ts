import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/base.css'
import { probeCriticalFonts } from './fonts'
import { initLocale } from './i18n'
import { apiFetch } from './services/apiClient'
import { setAuthenticatedUser } from './services/sessionContext'
import { loadFeatureFlags } from './services/featuresApi'

async function bootstrap() {
  initLocale()
  probeCriticalFonts()
  try {
    const me = await apiFetch<{ id: string; email: string }>('/api/me')
    setAuthenticatedUser(me)
    await loadFeatureFlags()
  } catch {
    setAuthenticatedUser(null)
  }
  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#root')
}

void bootstrap()
