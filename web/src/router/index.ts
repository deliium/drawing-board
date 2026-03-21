import { createRouter, createWebHashHistory } from 'vue-router'
import BoardPage from '../pages/BoardPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import { requireAuth } from './guards'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'board', component: BoardPage, meta: { requiresAuth: true } },
    { path: '/login', name: 'login', component: LoginPage },
  ],
})

router.beforeEach(requireAuth)

export default router
