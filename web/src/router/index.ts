import { createRouter, createWebHashHistory } from 'vue-router'
import BoardPage from '../pages/BoardPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import { requireAuth } from './guards'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'board', component: BoardPage, meta: { requiresAuth: true } },
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
