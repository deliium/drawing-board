import { createRouter, createWebHashHistory } from 'vue-router'
import BoardPage from '../pages/BoardPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import PracticeCharacterPage from '../pages/PracticeCharacterPage.vue'
import PracticeHistoryPage from '../pages/PracticeHistoryPage.vue'
import PracticeHubPage from '../pages/PracticeHubPage.vue'
import { requireAuth } from './guards'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'board', component: BoardPage, meta: { requiresAuth: true } },
    {
      path: '/practice',
      name: 'practice-hub',
      component: PracticeHubPage,
      meta: { requiresAuth: true },
    },
    {
      path: '/practice/history',
      name: 'practice-history',
      component: PracticeHistoryPage,
      meta: { requiresAuth: true },
    },
    {
      path: '/practice/:characterId',
      name: 'practice-character',
      component: PracticeCharacterPage,
      meta: { requiresAuth: true },
    },
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { guestOnly: true },
      props: { initialMode: 'login' },
    },
    {
      path: '/register',
      name: 'register',
      component: LoginPage,
      meta: { guestOnly: true },
      props: { initialMode: 'register' },
    },
  ],
})

router.beforeEach(requireAuth)

export default router
