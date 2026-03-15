import { createMemoryHistory, createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuth } from '../stores/auth'
import { pathPrefix as prefix } from '../config'

import LoginView from '../views/LoginView.vue'
import AppLayout from '../components/AppLayout.vue'
import DashboardView from '../views/DashboardView.vue'
import AccountsView from '../views/AccountsView.vue'
import LogsView from '../views/LogsView.vue'
import SettingsView from '../views/SettingsView.vue'

export function buildRoutes(): RouteRecordRaw[] {
  return [
    { path: '/login', component: LoginView, meta: { guest: true } },
    { path: '/', redirect: '/dashboard' },
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: '/dashboard', component: DashboardView },
        { path: '/accounts', component: AccountsView },
        { path: '/logs', component: LogsView },
        { path: '/settings', component: SettingsView },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
  ]
}

const history = typeof window === 'undefined' ? createMemoryHistory() : createWebHistory(prefix)

const router = createRouter({
  history,
  routes: buildRoutes()
})

router.beforeEach((to) => {
  const { isAuthenticated } = useAuth()
  if (!to.meta.guest && !isAuthenticated.value) {
    return '/login'
  }
  if (to.meta.guest && isAuthenticated.value) {
    return '/dashboard'
  }
})

export default router
