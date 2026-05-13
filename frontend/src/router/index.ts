import { createMemoryHistory, createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuth } from '../stores/auth'
import { pathPrefix as prefix } from '../config'

const LoginView = () => import('../views/LoginView.vue')
const AppLayout = () => import('../components/AppLayout.vue')
const DashboardView = () => import('../views/DashboardView.vue')
const AccountsView = () => import('../views/AccountsView.vue')
const LogsView = () => import('../views/LogsView.vue')
const SettingsView = () => import('../views/SettingsView.vue')

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
