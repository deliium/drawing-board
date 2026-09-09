import { createRouter, createWebHashHistory } from 'vue-router'
import BoardPage from '../pages/BoardPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import PracticeCharacterPage from '../pages/PracticeCharacterPage.vue'
import PracticeHistoryPage from '../pages/PracticeHistoryPage.vue'
import PracticeHubPage from '../pages/PracticeHubPage.vue'
import { requireAuth } from './guards'

const BRAND = 'Japanese Handwriting Practice'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'board',
      component: BoardPage,
      meta: { requiresAuth: true, title: 'Free board' },
    },
    {
      path: '/practice',
      name: 'practice-hub',
      component: PracticeHubPage,
      meta: { requiresAuth: true, title: 'Practice' },
    },
    {
      path: '/practice/history',
      name: 'practice-history',
      component: PracticeHistoryPage,
      meta: { requiresAuth: true, title: 'History' },
    },
    {
      path: '/practice/:characterId',
      name: 'practice-character',
      component: PracticeCharacterPage,
      meta: { requiresAuth: true, title: 'Practice character' },
    },
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { guestOnly: true, guestShell: true, title: 'Sign in' },
      props: { initialMode: 'login' },
    },
    {
      path: '/register',
      name: 'register',
      component: LoginPage,
      meta: { guestOnly: true, guestShell: true, title: 'Create account' },
      props: { initialMode: 'register' },
    },
  ],
})

router.beforeEach(requireAuth)

router.afterEach((to) => {
  const page = typeof to.meta.title === 'string' ? to.meta.title : ''
  document.title = page ? `${page} · ${BRAND}` : BRAND
})

export default router
